package scout

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// RodBrowserPage wraps a *rod.Page to implement the ai.BrowserPage interface.
type RodBrowserPage struct {
	page           *rod.Page
	clickStrategy  ClickStrategy
	consoleLogs    []string
	mu             sync.Mutex
	viewportWidth  int    // stored for strategy re-detection
	viewportHeight int    // stored for CDP screenshot downscale
	deviceCategory string // stored for strategy re-detection
}

// NewRodBrowserPage creates a new RodBrowserPage wrapping the given rod page.
// Defaults to desktop viewport width (1920) for click strategy selection.
func NewRodBrowserPage(page *rod.Page) *RodBrowserPage {
	return &RodBrowserPage{page: page, viewportWidth: 1920, viewportHeight: 1080}
}

// CaptureScreenshot takes a JPEG screenshot and returns it as base64.
// It first attempts a fast path using canvas.toDataURL() which reads directly
// from the GPU framebuffer, bypassing the Chrome compositor. This is
// significantly faster on SwiftShader. Falls back to CDP Page.captureScreenshot
// if the fast path fails (e.g. no canvas, tainted canvas, HTML overlays).
//
// The fast path is always preferred over CDP when a canvas exists, because CDP
// screenshots frequently stall for 60+ seconds on SwiftShader during complex
// WebGL animations. VLMs normalize coordinates to the stated viewport range
// regardless of image resolution, so coordinate targeting works even when the
// canvas internal resolution differs from the viewport.
func (r *RodBrowserPage) CaptureScreenshot() (string, error) {
	// Fast path: extract directly from the game canvas (avoids compositor overhead)
	// Uses toDataURL at low quality directly — no offscreen canvas copy, which stalls on SwiftShader.
	// Always preferred when a canvas exists because CDP screenshots frequently timeout
	// on SwiftShader during complex WebGL animations (60s+ stalls).
	// VLMs normalize coordinates to the stated viewport range regardless of image resolution,
	// so even when canvas internal resolution differs from viewport, click targeting works.
	// For Phaser games: pauses the game loop before toDataURL to prevent SwiftShader
	// contention (renders stop, so framebuffer read completes in ~1s instead of 30-40s).
	result, err := r.page.Eval(`() => {
		const c = document.querySelector('canvas');
		if (c && c.width > 0 && c.height > 0) {
			try {
				const g = window.game;
				const paused = g && g.loop && typeof g.loop.sleep === 'function';
				if (paused) g.loop.sleep();
				const data = c.toDataURL('image/jpeg', 0.15);
				if (paused) g.loop.wake();
				return data;
			} catch(e) {}
		}
		return '';
	}`)
	if err == nil && result != nil {
		dataURL := result.Value.Str()
		if strings.HasPrefix(dataURL, "data:image/") {
			parts := strings.SplitN(dataURL, ",", 2)
			if len(parts) == 2 && len(parts[1]) > 100 {
				log.Printf("Screenshot: canvas fast path (%d bytes)", len(parts[1]))
				return parts[1], nil
			}
		}
	}

	// Fallback: CDP screenshot (handles HTML overlays, no-canvas pages, tainted canvases,
	// and canvas games where internal resolution differs from viewport — ensures viewport-aligned coordinates)
	quality := 15
	data, err := r.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:           proto.PageCaptureScreenshotFormatJpeg,
		Quality:          &quality,
		OptimizeForSpeed: true,
	})
	if err != nil {
		return "", fmt.Errorf("screenshot failed: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	log.Printf("Screenshot: CDP viewport fallback (%d bytes)", len(encoded))
	return encoded, nil
}

// captureWithTimeout wraps CaptureScreenshot with a timeout to prevent
// SwiftShader stalls from blocking indefinitely. Returns empty string on timeout.
func captureWithTimeout(bp *RodBrowserPage, timeout time.Duration) string {
	ch := make(chan string, 1)
	go func() {
		b64, err := bp.CaptureScreenshot()
		if err != nil {
			ch <- ""
			return
		}
		ch <- b64
	}()
	select {
	case s := <-ch:
		return s
	case <-time.After(timeout):
		return ""
	}
}

// Click clicks at the given pixel coordinates using the configured click strategy.
// Defaults to JSDispatchStrategy if no strategy has been set.
func (r *RodBrowserPage) Click(x, y int) error {
	s := r.clickStrategy
	if s == nil {
		s = &JSDispatchStrategy{}
	}
	return s.Click(r.page, x, y)
}

// SetClickStrategy overrides the click strategy for this browser page.
func (r *RodBrowserPage) SetClickStrategy(s ClickStrategy) {
	r.clickStrategy = s
}

// RedetectClickStrategy re-runs canvas/framework detection on the current page
// and updates the click strategy. Called after navigation so the strategy
// matches the actual page content rather than the initial about:blank.
// Polls briefly for canvas appearance since some frameworks create the canvas
// element asynchronously after DOMContentLoaded.
func (r *RodBrowserPage) RedetectClickStrategy() {
	meta := &PageMeta{}

	// Brief poll for canvas — some frameworks create it after load.
	// Only retries on eval failure (page still loading); a successful eval
	// returning false is trusted since WaitIdle already ran.
	pollInterval := 150 * time.Millisecond
	for i := 0; i < 5; i++ {
		hasCanvas, err := r.page.Eval(`() => document.querySelector('canvas') !== null`)
		if err == nil && hasCanvas != nil {
			if hasCanvas.Value.Bool() {
				meta.CanvasFound = true
			}
			break // eval succeeded — trust the result
		}
		// Eval failed (page navigating/detached), retry after a brief wait
		if i < 4 {
			time.Sleep(pollInterval)
		}
	}

	globals, err := r.page.Eval(`() => {
		const found = [];
		if (window.Phaser) found.push("Phaser " + (Phaser.VERSION || ""));
		if (window.PIXI) found.push("PIXI " + (PIXI.VERSION || ""));
		if (window.cc && window.cc.game) found.push("Cocos");
		if (window.THREE) found.push("Three.js");
		if (window.BABYLON) found.push("Babylon.js");
		if (window.PlayCanvas) found.push("PlayCanvas");
		return found;
	}`)
	if err == nil && globals != nil && globals.Value.Arr() != nil {
		for _, v := range globals.Value.Arr() {
			if v.Nil() {
				continue
			}
			meta.JSGlobals = append(meta.JSGlobals, v.Str())
		}
		detectFrameworkFromGlobals(meta)
	}
	r.clickStrategy = SelectClickStrategy(meta, r.viewportWidth, r.deviceCategory)
}

// TypeText types the given text by inserting it into the page.
func (r *RodBrowserPage) TypeText(text string) error {
	return r.page.InsertText(text)
}

// Scroll scrolls the page by the given delta in pixels.
func (r *RodBrowserPage) Scroll(dx, dy float64) error {
	return r.page.Mouse.Scroll(dx, dy, 3)
}

// EvalJS evaluates a JavaScript expression and returns the result as a string.
func (r *RodBrowserPage) EvalJS(expr string) (string, error) {
	// Use rod's Eval with a function that receives the expression as a parameter
	// to avoid injection issues with backticks or quotes in the expression.
	result, err := r.page.Eval(`(expr) => {
		try {
			const result = eval(expr);
			return String(result);
		} catch(e) {
			return "Error: " + e.message;
		}
	}`, expr)
	if err != nil {
		return "", fmt.Errorf("JS eval failed: %w", err)
	}
	if result == nil {
		return "", nil
	}
	return result.Value.Str(), nil
}

// WaitVisible waits for an element matching the selector to become visible.
func (r *RodBrowserPage) WaitVisible(selector string, timeout time.Duration) error {
	el, err := r.page.Timeout(timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("wait visible %q: element not found: %w", selector, err)
	}
	if err := el.WaitVisible(); err != nil {
		return fmt.Errorf("wait visible %q: %w", selector, err)
	}
	return nil
}

// GetPageInfo returns the page title, URL, and visible text content.
func (r *RodBrowserPage) GetPageInfo() (title, pageURL, visibleText string, err error) {
	info, infoErr := r.page.Info()
	if infoErr != nil {
		return "", "", "", fmt.Errorf("get page info: %w", infoErr)
	}
	title = info.Title
	pageURL = info.URL

	// Get visible text (body innerText, truncated)
	result, evalErr := r.page.Eval(`() => {
		const text = document.body ? document.body.innerText : '';
		return text.substring(0, 3000);
	}`)
	if evalErr == nil && result != nil {
		visibleText = result.Value.Str()
	}
	return title, pageURL, visibleText, nil
}

// GetConsoleLogs returns captured console log messages and clears the buffer.
func (r *RodBrowserPage) GetConsoleLogs() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	logs := make([]string, len(r.consoleLogs))
	copy(logs, r.consoleLogs)
	r.consoleLogs = r.consoleLogs[:0]
	return logs, nil
}

// Navigate navigates the page to the given URL and waits for load + idle.
// After loading, re-detects the click strategy for the new page content.
func (r *RodBrowserPage) Navigate(url string) error {
	if err := r.page.Navigate(url); err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}
	if err := r.page.WaitLoad(); err != nil {
		return fmt.Errorf("wait load after navigate: %w", err)
	}
	_ = r.page.WaitIdle(3 * time.Second) // best-effort; timeout is not an error
	r.RedetectClickStrategy()
	return nil
}

// newHeadlessLauncher creates a launcher.Launcher pre-configured for Phaser/WebGL game testing.
// - HeadlessNew: uses --headless=new (Chrome 112+) which shares the full browser
//   rendering pipeline, giving proper WebGL/canvas support unlike old --headless.
// - SwiftShader: CPU-based Vulkan/GLES backend for WebGL without a real GPU.
// - Autoplay: allow game audio without user gesture (Phaser Web Audio API).
// - Font hinting: disabled for consistent screenshot rendering across environments.
// Do NOT use --disable-gpu or --disable-software-rasterizer — they break WebGL.
func newHeadlessLauncher() *launcher.Launcher {
	l := launcher.New().
		HeadlessNew(true).
		NoSandbox(true).
		Set("disable-dev-shm-usage").
		Set("use-gl", "angle").
		Set("use-angle", "swiftshader").
		Set("enable-unsafe-swiftshader").
		Set("autoplay-policy", "no-user-gesture-required").
		Set("font-render-hinting", "none").
		// Performance flags for SwiftShader (CPU-based GPU rendering):
		Set("in-process-gpu").                       // reduce IPC overhead between browser and GPU process
		Set("disable-hang-monitor").                 // SwiftShader is slow; prevent Chrome killing "hung" renderers
		Set("disable-background-timer-throttling").  // keep game loop timers running at full speed
		Set("disable-renderer-backgrounding").       // don't deprioritize the renderer in headless mode
		Set("disable-backgrounding-occluded-windows").
		Set("disable-ipc-flooding-protection").      // remove CDP message rate limiting
		Set("disable-extensions").                   // no extension overhead
		Set("disable-component-update").             // no background component update checks
		Set("disable-background-networking").         // no safe-browsing or other background network
		Set("mute-audio").                           // skip audio processing (Web Audio API) entirely
		Set("disable-smooth-scrolling").             // no scroll animations
		Set("no-first-run").                         // skip first-run dialog
		Set("disable-sync").                         // no Chrome sync overhead
		Set("disable-default-apps")                  // no default app installs

	if bin := lookupChromeBin(); bin != "" {
		l = l.Bin(bin)
	}
	return l
}

// ScoutURLHeadlessKeepAlive is like ScoutURLHeadless but returns a live page and cleanup function
// instead of closing the browser. This is used for agent mode where the browser stays open.
func ScoutURLHeadlessKeepAlive(ctx context.Context, gameURL string, cfg HeadlessConfig) (*PageMeta, *RodBrowserPage, func(), error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	l := newHeadlessLauncher()

	u, err := l.Launch()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("launching headless browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, nil, nil, fmt.Errorf("connecting to browser: %w", err)
	}

	cleanup := func() {
		browser.Close()
	}

	// Set viewport
	width := cfg.Width
	height := cfg.Height
	if width <= 0 {
		width = 1920
	}
	if height <= 0 {
		height = 1080
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		cleanup()
		return nil, nil, nil, fmt.Errorf("creating page: %w", err)
	}

	dpr := cfg.DevicePixelRatio
	if dpr <= 0 {
		dpr = 1
	}
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             width,
		Height:            height,
		DeviceScaleFactor: dpr,
	}); err != nil {
		cleanup()
		return nil, nil, nil, fmt.Errorf("setting viewport: %w", err)
	}

	browserPage := &RodBrowserPage{
		page:           page,
		viewportWidth:  width,
		viewportHeight: height,
		deviceCategory: cfg.DeviceCategory,
	}

	// Block analytics/tracking network requests that waste CPU and bandwidth.
	_ = proto.NetworkSetBlockedURLs{
		Urls: []string{
			"*google-analytics*", "*googletagmanager*", "*facebook.net*",
			"*doubleclick*", "*hotjar*", "*segment*", "*mixpanel*",
			"*sentry.io*", "*newrelic*", "*datadoghq*",
		},
	}.Call(page)

	// Inject WebGL context overrides before any game script runs:
	// - antialias:false — avoids expensive MSAA in SwiftShader
	// - preserveDrawingBuffer:true — required for canvas.toDataURL fast path
	_, _ = page.EvalOnNewDocument(`
		const _origGetContext = HTMLCanvasElement.prototype.getContext;
		HTMLCanvasElement.prototype.getContext = function(type, attrs) {
			if (type === 'webgl' || type === 'webgl2' || type === 'experimental-webgl') {
				attrs = Object.assign({}, attrs, {
					antialias: false,
					preserveDrawingBuffer: true,
					powerPreference: 'low-power'
				});
			}
			return _origGetContext.call(this, type, attrs);
		};
	`)

	// Collect console logs for agent visibility (mirrors ScoutURLHeadless pattern).
	// Cap at 2000 lines to prevent unbounded memory growth in long-running agent sessions.
	const maxConsoleLines = 2000
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		var parts []string
		for _, arg := range e.Args {
			str := arg.Value.Str()
			if str != "" {
				parts = append(parts, str)
			}
		}
		if len(parts) > 0 {
			line := fmt.Sprintf("[%s] %s", e.Type, strings.Join(parts, " "))
			browserPage.mu.Lock()
			if len(browserPage.consoleLogs) < maxConsoleLines {
				browserPage.consoleLogs = append(browserPage.consoleLogs, line)
			}
			browserPage.mu.Unlock()
		}
	})()

	// Navigate to the URL
	if err := page.Context(ctx).Navigate(gameURL); err != nil {
		cleanup()
		return nil, nil, nil, fmt.Errorf("navigating to %s: %w", gameURL, err)
	}

	if err := page.Context(ctx).WaitLoad(); err != nil {
		cleanup()
		return nil, nil, nil, fmt.Errorf("waiting for page load: %w", err)
	}

	_ = page.Context(ctx).WaitIdle(1500 * time.Millisecond) // best-effort

	// Wait for canvas + game framework readiness with exponential backoff polling
	canvasFound := false
	pollInterval := 100 * time.Millisecond
	for i := 0; i < 20; i++ {
		ready, evalErr := page.Eval(`() => {
			const canvas = document.querySelector('canvas');
			if (!canvas) return 'no_canvas';
			if (canvas.width === 0 || canvas.height === 0) return 'zero_size';
			// Check for common error indicators
			const errorDialog = document.querySelector('[role="dialog"], .error, .error-dialog, .error-overlay');
			if (errorDialog && errorDialog.textContent.toLowerCase().includes('error')) return 'error_visible';
			// Check game framework readiness
			if (window.Phaser && window.game) return 'ready';
			if (window.PIXI && window.PIXI.Application) return 'ready';
			if (canvas.width > 0 && canvas.height > 0) return 'canvas_ok';
			return 'waiting';
		}`)
		if evalErr == nil && ready != nil {
			state := ready.Value.Str()
			if state == "ready" || state == "canvas_ok" || state == "error_visible" {
				canvasFound = true
				break
			}
			if state == "no_canvas" && i > 15 {
				// Give up if no canvas at all
				break
			}
		}
		time.Sleep(pollInterval)
		if pollInterval < 500*time.Millisecond {
			pollInterval = pollInterval * 3 / 2 // 100→150→225→337→500
		}
	}

	// Extract rendered HTML
	htmlContent, err := page.HTML()
	if err != nil {
		cleanup()
		return nil, nil, nil, fmt.Errorf("getting page HTML: %w", err)
	}

	meta := ParseHTML(htmlContent)
	if canvasFound {
		meta.CanvasFound = true
	}

	// JS global detection
	globals, evalErr := page.Eval(`() => {
		const found = [];
		if (window.Phaser) found.push("Phaser " + (Phaser.VERSION || ""));
		if (window.PIXI) found.push("PIXI " + (PIXI.VERSION || ""));
		if (window.cc && window.cc.game) found.push("Cocos");
		if (window.THREE) found.push("Three.js");
		if (window.BABYLON) found.push("Babylon.js");
		if (window.PlayCanvas) found.push("PlayCanvas");
		const canvases = document.querySelectorAll('canvas');
		if (canvases.length > 0) found.push("canvas:" + canvases.length);
		return found;
	}`)
	if evalErr == nil && globals != nil && globals.Value.Arr() != nil {
		for _, v := range globals.Value.Arr() {
			if v.Nil() {
				continue
			}
			meta.JSGlobals = append(meta.JSGlobals, v.Str())
		}
		detectFrameworkFromGlobals(meta)
	}

	// Select click strategy based on detected framework, viewport, and device category
	browserPage.clickStrategy = SelectClickStrategy(meta, width, cfg.DeviceCategory)
	meta.ClickStrategy = browserPage.clickStrategy.Name()

	// Take initial screenshot via CaptureScreenshot (uses fast path + CDP fallback)
	// Timeout prevents SwiftShader stalls from blocking the entire scout.
	if shot := captureWithTimeout(browserPage, 20*time.Second); shot != "" {
		meta.ScreenshotB64 = shot
		meta.Screenshots = append(meta.Screenshots, shot)
	}

	return meta, browserPage, cleanup, nil
}

// lookupChromeBin returns the Chromium binary path from CHROME_BIN env,
// or empty string to let go-rod auto-detect.
func lookupChromeBin() string {
	if bin := os.Getenv("CHROME_BIN"); bin != "" {
		return bin
	}
	return ""
}

// detectFrameworkFromGlobals sets meta.Framework based on detected JS globals.
func detectFrameworkFromGlobals(meta *PageMeta) {
	for _, g := range meta.JSGlobals {
		gl := strings.ToLower(g)
		switch {
		case strings.HasPrefix(gl, "phaser"):
			meta.Framework = "phaser"
		case strings.HasPrefix(gl, "pixi"):
			meta.Framework = "pixi"
		case strings.HasPrefix(gl, "cocos"):
			meta.Framework = "cocos"
		case strings.HasPrefix(gl, "three"):
			meta.Framework = "threejs"
		case strings.HasPrefix(gl, "babylon"):
			meta.Framework = "babylon"
		case strings.HasPrefix(gl, "playcanvas"):
			meta.Framework = "playcanvas"
		}
	}
}

// ScoutURLHeadless uses headless Chrome to render a page and extract metadata.
// This handles JS-rendered games that require execution to reveal framework details.
func ScoutURLHeadless(ctx context.Context, gameURL string, cfg HeadlessConfig) (*PageMeta, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	l := newHeadlessLauncher()

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launching headless browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connecting to browser: %w", err)
	}
	defer browser.Close()

	// Set viewport
	width := cfg.Width
	height := cfg.Height
	if width <= 0 {
		width = 1920
	}
	if height <= 0 {
		height = 1080
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("creating page: %w", err)
	}
	defer page.Close()

	dpr := cfg.DevicePixelRatio
	if dpr <= 0 {
		dpr = 1
	}
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             width,
		Height:            height,
		DeviceScaleFactor: dpr,
	}); err != nil {
		return nil, fmt.Errorf("setting viewport: %w", err)
	}

	// Collect console logs for framework clues (capped at 512KB)
	const maxConsoleLogBytes = 512 * 1024
	var consoleLogs strings.Builder
	var consoleLogsMu sync.Mutex
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		consoleLogsMu.Lock()
		defer consoleLogsMu.Unlock()
		for _, arg := range e.Args {
			str := arg.Value.Str()
			if str != "" {
				if consoleLogs.Len() >= maxConsoleLogBytes {
					return
				}
				consoleLogs.WriteString(str)
				consoleLogs.WriteString(" ")
			}
		}
		consoleLogs.WriteString("\n")
	})()

	// Navigate to the URL
	if err := page.Context(ctx).Navigate(gameURL); err != nil {
		return nil, fmt.Errorf("navigating to %s: %w", gameURL, err)
	}

	// Wait for page load
	if err := page.Context(ctx).WaitLoad(); err != nil {
		return nil, fmt.Errorf("waiting for page load: %w", err)
	}

	// Extended wait for JS frameworks to initialize
	_ = page.Context(ctx).WaitIdle(3 * time.Second) // best-effort

	// Wait for canvas to appear with exponential backoff polling
	canvasFound := false
	canvasPollInterval := 100 * time.Millisecond
	for i := 0; i < 15; i++ {
		hasCanvas, evalErr := page.Eval(`() => document.querySelector('canvas') !== null`)
		if evalErr == nil && hasCanvas != nil && hasCanvas.Value.Bool() {
			canvasFound = true
			break
		}
		time.Sleep(canvasPollInterval)
		if canvasPollInterval < 500*time.Millisecond {
			canvasPollInterval = canvasPollInterval * 3 / 2
		}
	}

	// Extract rendered HTML
	htmlContent, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("getting page HTML: %w", err)
	}

	// Parse the rendered HTML using existing parser
	meta := ParseHTML(htmlContent)

	// Override canvas detection from live DOM check
	if canvasFound {
		meta.CanvasFound = true
	}

	// JS global detection — these are authoritative since they come from the running game
	globals, evalErr := page.Eval(`() => {
		const found = [];
		if (window.Phaser) found.push("Phaser " + (Phaser.VERSION || ""));
		if (window.PIXI) found.push("PIXI " + (PIXI.VERSION || ""));
		if (window.cc && window.cc.game) found.push("Cocos");
		if (window.THREE) found.push("Three.js");
		if (window.BABYLON) found.push("Babylon.js");
		if (window.PlayCanvas) found.push("PlayCanvas");
		const canvases = document.querySelectorAll('canvas');
		if (canvases.length > 0) found.push("canvas:" + canvases.length);
		return found;
	}`)
	if evalErr == nil && globals != nil && globals.Value.Arr() != nil {
		for _, v := range globals.Value.Arr() {
			if v.Nil() {
				continue
			}
			meta.JSGlobals = append(meta.JSGlobals, v.Str())
		}
		// Update framework from JS globals (authoritative)
		detectFrameworkFromGlobals(meta)
	}

	// Check for WebGL context via console logs
	consoleLogsMu.Lock()
	consoleStr := consoleLogs.String()
	consoleLogsMu.Unlock()
	if meta.Framework == "unknown" && consoleStr != "" {
		detected := DetectFramework(nil, consoleStr)
		if detected != "unknown" {
			meta.Framework = detected
		}
	}

	// --- Multi-screenshot capture ---
	// Capture screenshots of multiple game states to give the AI visibility
	// beyond just the loading screen. JPEG for faster encoding on SwiftShader.
	imgQuality := 30
	captureScreenshot := func() string {
		ch := make(chan string, 1)
		go func() {
			data, err := page.Screenshot(false, &proto.PageCaptureScreenshot{
				Format:           proto.PageCaptureScreenshotFormatJpeg,
				Quality:          &imgQuality,
				OptimizeForSpeed: true,
			})
			if err != nil || len(data) == 0 {
				ch <- ""
				return
			}
			ch <- base64.StdEncoding.EncodeToString(data)
		}()
		select {
		case s := <-ch:
			return s
		case <-time.After(20 * time.Second):
			return ""
		}
	}

	// Screenshot 1: Initial load state
	if shot := captureScreenshot(); shot != "" {
		meta.ScreenshotB64 = shot
		meta.Screenshots = append(meta.Screenshots, shot)
		if cfg.ScreenshotPath != "" {
			if raw, err := base64.StdEncoding.DecodeString(shot); err == nil {
				os.WriteFile(cfg.ScreenshotPath, raw, 0644)
			}
		}
	}

	// Screenshot 2: Click center of canvas to start the game (CDP for trusted events)
	if canvasFound {
		cx := float64(width / 2)
		cy := float64(height / 2)
		// Move cursor first — matches CDPMouseStrategy for consistency
		_ = (proto.InputDispatchMouseEvent{
			Type: proto.InputDispatchMouseEventTypeMouseMoved,
			X:    cx,
			Y:    cy,
		}).Call(page)
		_ = (proto.InputDispatchMouseEvent{
			Type:       proto.InputDispatchMouseEventTypeMousePressed,
			X:         cx,
			Y:         cy,
			Button:     proto.InputMouseButtonLeft,
			ClickCount: 1,
		}).Call(page)
		clickErr := (proto.InputDispatchMouseEvent{
			Type:       proto.InputDispatchMouseEventTypeMouseReleased,
			X:         cx,
			Y:         cy,
			Button:     proto.InputMouseButtonLeft,
			ClickCount: 1,
		}).Call(page)
		if clickErr == nil {
			time.Sleep(2 * time.Second)
			if shot := captureScreenshot(); shot != "" {
				meta.Screenshots = append(meta.Screenshots, shot)
			}
		}
	}

	// Screenshot 3: Look for and click common game buttons (Play, Start, Spin, OK)
	clickedButton, _ := page.Eval(`() => {
		const labels = ['play', 'start', 'spin', 'ok', 'continue', 'begin', 'tap to start', 'click to start'];
		// Try HTML buttons/links first
		const elements = document.querySelectorAll('button, a, [role="button"], .btn');
		for (const el of elements) {
			const text = (el.textContent || el.innerText || '').trim().toLowerCase();
			for (const label of labels) {
				if (text === label || text.includes(label)) {
					el.click();
					return true;
				}
			}
		}
		return false;
	}`)
	if clickedButton != nil && clickedButton.Value.Bool() {
		time.Sleep(2 * time.Second)
		if shot := captureScreenshot(); shot != "" {
			meta.Screenshots = append(meta.Screenshots, shot)
		}
	}

	return meta, nil
}
