package scout

import (
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// ClickStrategy defines how clicks are dispatched to the page.
// Different game types require different click mechanisms for reliable input.
type ClickStrategy interface {
	Click(page *rod.Page, x, y int) error
	Name() string
}

// CDPMouseStrategy dispatches trusted mouse events via Chrome DevTools Protocol.
// This produces isTrusted=true events with correct offsetX/offsetY, which is
// required for canvas/WebGL games (Phaser, PixiJS, Three.js, etc.) that ignore
// untrusted synthetic events.
type CDPMouseStrategy struct {
	vpWidth, vpHeight int // cached viewport; 0 = not yet fetched
}

func (s *CDPMouseStrategy) Name() string { return "cdp_mouse" }

func (s *CDPMouseStrategy) Click(page *rod.Page, x, y int) error {
	vpW, vpH := s.vpWidth, s.vpHeight
	if vpW == 0 {
		w, h, ok := evalViewportSize(page)
		vpW, vpH = w, h
		if ok {
			s.vpWidth, s.vpHeight = w, h // cache only on success
		}
	}
	cx, cy := clampCoords(x, y, vpW, vpH)
	fx, fy := float64(cx), float64(cy)

	// Diagnostic: check what element is at click coordinates and if an overlay blocks the canvas
	overlayCleared := false
	hitInfo, evalErr := page.Eval(`(x, y) => {
		const el = document.elementFromPoint(x, y);
		if (!el) return JSON.stringify({tag: 'none', hit: 'none'});
		const tag = el.tagName.toLowerCase();
		const id = el.id || '';
		const cls = (el.className && typeof el.className === 'string') ? el.className.substring(0, 60) : '';
		const canvas = document.querySelector('canvas');
		const isCanvas = el === canvas;
		// Check canvas bounding rect for diagnostics
		let canvasRect = null;
		if (canvas) {
			const r = canvas.getBoundingClientRect();
			canvasRect = {l: Math.round(r.left), t: Math.round(r.top), w: Math.round(r.width), h: Math.round(r.height),
				iw: canvas.width, ih: canvas.height};
		}
		return JSON.stringify({tag, id, cls, isCanvas, canvasRect});
	}`, cx, cy)
	if evalErr == nil && hitInfo != nil {
		log.Printf("CDP click (%d,%d): hit=%s", cx, cy, hitInfo.Value.Str())
	}

	// If the click target is not the canvas, try to clear overlays blocking it
	if evalErr == nil && hitInfo != nil {
		hitStr := hitInfo.Value.Str()
		if len(hitStr) > 0 && hitStr[0] == '{' {
			// Check if we hit the canvas
			if !containsIsCanvas(hitStr) {
				// Try to disable pointer-events on overlaying elements
				cleared, clearErr := page.Eval(`(x, y) => {
					let cleared = 0;
					for (let i = 0; i < 5; i++) {
						const el = document.elementFromPoint(x, y);
						if (!el) break;
						if (el.tagName.toLowerCase() === 'canvas') break;
						// Don't remove body or html
						if (el === document.body || el === document.documentElement) break;
						el.style.pointerEvents = 'none';
						cleared++;
					}
					return cleared;
				}`, cx, cy)
				if clearErr == nil && cleared != nil && cleared.Value.Int() > 0 {
					log.Printf("CDP click (%d,%d): cleared %d overlay(s) blocking canvas", cx, cy, cleared.Value.Int())
					overlayCleared = true
				}
			}
		}
	}

	// Move cursor first — some engines (Three.js, Babylon) track position via
	// mousemove and only register clicks at the last-known cursor location.
	_ = (proto.InputDispatchMouseEvent{
		Type: proto.InputDispatchMouseEventTypeMouseMoved,
		X:    fx,
		Y:    fy,
	}).Call(page)

	if err := (proto.InputDispatchMouseEvent{
		Type:       proto.InputDispatchMouseEventTypeMousePressed,
		X:         fx,
		Y:         fy,
		Button:     proto.InputMouseButtonLeft,
		ClickCount: 1,
	}).Call(page); err != nil {
		return fmt.Errorf("cdp mouse press at (%d,%d): %w", x, y, err)
	}
	if err := (proto.InputDispatchMouseEvent{
		Type:       proto.InputDispatchMouseEventTypeMouseReleased,
		X:         fx,
		Y:         fy,
		Button:     proto.InputMouseButtonLeft,
		ClickCount: 1,
	}).Call(page); err != nil {
		return fmt.Errorf("cdp mouse release at (%d,%d): %w", x, y, err)
	}

	// If we cleared overlays, re-verify and log whether the canvas now receives the click
	if overlayCleared {
		recheck, _ := page.Eval(`(x, y) => {
			const el = document.elementFromPoint(x, y);
			return el ? el.tagName.toLowerCase() : 'none';
		}`, cx, cy)
		if recheck != nil {
			log.Printf("CDP click (%d,%d): after overlay clear, element at point: %s", cx, cy, recheck.Value.Str())
		}
	}

	return nil
}

// containsIsCanvas is a quick check for "isCanvas":true in the hit info JSON.
func containsIsCanvas(s string) bool {
	// Look for "isCanvas":true (with or without spaces)
	for i := 0; i < len(s)-14; i++ {
		if s[i] == 'i' && i+14 <= len(s) && s[i:i+10] == "isCanvas\":" {
			rest := s[i+10:]
			for j := 0; j < len(rest) && j < 5; j++ {
				if rest[j] == 't' {
					return true
				}
				if rest[j] != ' ' {
					break
				}
			}
		}
	}
	return false
}

// CDPTouchStrategy dispatches trusted touch events via Chrome DevTools Protocol.
// Used for mobile-viewport games that only listen for touch events.
type CDPTouchStrategy struct {
	vpWidth, vpHeight int // cached viewport; 0 = not yet fetched
}

func (s *CDPTouchStrategy) Name() string { return "cdp_touch" }

func (s *CDPTouchStrategy) Click(page *rod.Page, x, y int) error {
	vpW, vpH := s.vpWidth, s.vpHeight
	if vpW == 0 {
		w, h, ok := evalViewportSize(page)
		vpW, vpH = w, h
		if ok {
			s.vpWidth, s.vpHeight = w, h // cache only on success
		}
	}
	cx, cy := clampCoords(x, y, vpW, vpH)
	fx, fy := float64(cx), float64(cy)

	touchPoint := &proto.InputTouchPoint{X: fx, Y: fy}
	if err := (proto.InputDispatchTouchEvent{
		Type:        proto.InputDispatchTouchEventTypeTouchStart,
		TouchPoints: []*proto.InputTouchPoint{touchPoint},
	}).Call(page); err != nil {
		return fmt.Errorf("cdp touch start at (%d,%d): %w", x, y, err)
	}
	if err := (proto.InputDispatchTouchEvent{
		Type:        proto.InputDispatchTouchEventTypeTouchEnd,
		TouchPoints: []*proto.InputTouchPoint{},
	}).Call(page); err != nil {
		return fmt.Errorf("cdp touch end at (%d,%d): %w", x, y, err)
	}
	return nil
}

// JSDispatchStrategy dispatches pointer and mouse events via JavaScript.
// This targets the exact DOM element at (x,y) using elementFromPoint, which is
// ideal for pure HTML games with standard DOM buttons and interactive elements.
type JSDispatchStrategy struct{}

func (s *JSDispatchStrategy) Name() string { return "js_dispatch" }

func (s *JSDispatchStrategy) Click(page *rod.Page, x, y int) error {
	result, err := page.Eval(`(x, y) => {
		// Clamp coordinates to viewport bounds — AI occasionally clicks slightly outside
		x = Math.max(0, Math.min(x, window.innerWidth - 1));
		y = Math.max(0, Math.min(y, window.innerHeight - 1));
		let el = document.elementFromPoint(x, y);
		// Fallback: if still no element, target the canvas or body
		if (!el) el = document.querySelector('canvas') || document.body;
		if (!el) return 'no_element';
		const shared = { clientX: x, clientY: y, bubbles: true, cancelable: true, view: window };
		const ptrOpts = { ...shared, pointerId: 1, pointerType: 'mouse', isPrimary: true };
		// Move events — frameworks tracking cursor via move events need correct position
		el.dispatchEvent(new PointerEvent('pointermove', { ...ptrOpts, button: 0, buttons: 0 }));
		el.dispatchEvent(new MouseEvent('mousemove', { ...shared, button: 0, buttons: 0 }));
		el.dispatchEvent(new PointerEvent('pointerdown', { ...ptrOpts, button: 0, buttons: 1 }));
		el.dispatchEvent(new MouseEvent('mousedown', { ...shared, button: 0, buttons: 1 }));
		el.dispatchEvent(new PointerEvent('pointerup', { ...ptrOpts, button: 0, buttons: 0 }));
		el.dispatchEvent(new MouseEvent('mouseup', { ...shared, button: 0, buttons: 0 }));
		el.dispatchEvent(new MouseEvent('click', { ...shared, button: 0 }));
		return 'ok';
	}`, x, y)
	if err != nil {
		return fmt.Errorf("click at (%d,%d): %w", x, y, err)
	}
	if result == nil || result.Value.Str() == "no_element" {
		return fmt.Errorf("click at (%d,%d): no element at coordinates", x, y)
	}
	return nil
}

// isTouchCategory returns true if the device category represents a touch-only device.
func isTouchCategory(category string) bool {
	switch category {
	case "iPhone", "Android", "iPad", "Android Tablet":
		return true
	}
	return false
}

// isCanvasFramework returns true if the framework typically renders to a canvas element.
func isCanvasFramework(framework string) bool {
	switch framework {
	case "phaser", "pixi", "cocos", "threejs", "babylon", "playcanvas", "unity", "godot", "construct", "createjs":
		return true
	}
	return false
}

// SelectClickStrategy chooses the best click strategy based on page metadata,
// viewport width, and device category. Touch devices (phones + tablets) get touch
// events, canvas/WebGL games get trusted CDP mouse events, and pure HTML games
// get JS dispatch.
func SelectClickStrategy(meta *PageMeta, viewportWidth int, deviceCategory string) ClickStrategy {
	// Touch devices: phones + tablets always use touch
	if isTouchCategory(deviceCategory) {
		return &CDPTouchStrategy{}
	}
	// Fallback width heuristic for unknown category (e.g. CLI without preset)
	if deviceCategory == "" && viewportWidth <= 480 {
		return &CDPTouchStrategy{}
	}
	if meta.CanvasFound {
		return &CDPMouseStrategy{}
	}
	if isCanvasFramework(meta.Framework) {
		return &CDPMouseStrategy{}
	}
	return &JSDispatchStrategy{}
}

// evalViewportSize evaluates the page's viewport dimensions via JS.
// Returns (w, h, true) on success, or (1920, 1080, false) as a fallback
// so callers can decide whether to cache the result.
func evalViewportSize(page *rod.Page) (int, int, bool) {
	result, err := page.Eval(`() => [window.innerWidth, window.innerHeight]`)
	if err != nil || result == nil {
		return 1920, 1080, false
	}
	arr := result.Value.Arr()
	if len(arr) < 2 {
		return 1920, 1080, false
	}
	w := arr[0].Int()
	h := arr[1].Int()
	if w <= 0 || h <= 0 {
		return 1920, 1080, false
	}
	return w, h, true
}

// clampCoords clamps coordinates to stay within viewport bounds.
func clampCoords(x, y, w, h int) (int, int) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= w {
		x = w - 1
	}
	if y >= h {
		y = h - 1
	}
	return x, y
}
