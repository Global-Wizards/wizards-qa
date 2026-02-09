package scout

import (
	"context"
	"encoding/base64"
	"fmt"
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
	page        *rod.Page
	consoleLogs []string
	mu          sync.Mutex
}

// NewRodBrowserPage creates a new RodBrowserPage wrapping the given rod page.
func NewRodBrowserPage(page *rod.Page) *RodBrowserPage {
	return &RodBrowserPage{page: page}
}

// CaptureScreenshot takes a JPEG screenshot and returns it as base64.
func (r *RodBrowserPage) CaptureScreenshot() (string, error) {
	quality := 80
	data, err := r.page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: &quality,
	})
	if err != nil {
		return "", fmt.Errorf("screenshot failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// Click clicks at the given pixel coordinates.
func (r *RodBrowserPage) Click(x, y int) error {
	if err := r.page.Mouse.MoveTo(proto.NewPoint(float64(x), float64(y))); err != nil {
		return fmt.Errorf("move to (%d,%d): %w", x, y, err)
	}
	return r.page.Mouse.Click(proto.InputMouseButtonLeft, 1)
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
	if evalErr == nil {
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
func (r *RodBrowserPage) Navigate(url string) error {
	if err := r.page.Navigate(url); err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}
	if err := r.page.WaitLoad(); err != nil {
		return fmt.Errorf("wait load after navigate: %w", err)
	}
	r.page.WaitIdle(5 * time.Second)
	return nil
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

	// Launch headless Chrome optimized for Phaser/WebGL game testing.
	// - HeadlessNew: uses --headless=new (Chrome 112+) which shares the full browser
	//   rendering pipeline, giving proper WebGL/canvas support unlike old --headless.
	// - SwiftShader: CPU-based Vulkan/GLES backend for WebGL without a real GPU.
	// - Autoplay: allow game audio without user gesture (Phaser Web Audio API).
	// - Font hinting: disabled for consistent screenshot rendering across environments.
	// Do NOT use --disable-gpu or --disable-software-rasterizer — they break WebGL.
	l := launcher.New().
		HeadlessNew(true).
		NoSandbox(true).
		Set("disable-dev-shm-usage").
		Set("use-gl", "angle").
		Set("use-angle", "swiftshader").
		Set("enable-unsafe-swiftshader").
		Set("autoplay-policy", "no-user-gesture-required").
		Set("font-render-hinting", "none")

	if bin := lookupChromeBin(); bin != "" {
		l = l.Bin(bin)
	}

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

	page.MustSetViewport(width, height, 1, false)

	browserPage := &RodBrowserPage{page: page}

	// Collect console logs for agent visibility (mirrors ScoutURLHeadless pattern)
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
			browserPage.consoleLogs = append(browserPage.consoleLogs, line)
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

	page.Context(ctx).WaitIdle(5 * time.Second)

	// Wait up to 20s for canvas + game framework readiness
	canvasFound := false
	for i := 0; i < 40; i++ {
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
		if evalErr == nil {
			state := ready.Value.Str()
			if state == "ready" || state == "canvas_ok" || state == "error_visible" {
				canvasFound = true
				break
			}
			if state == "no_canvas" && i > 20 {
				// Give up after 10s if no canvas at all
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
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
	if evalErr == nil && globals.Value.Arr() != nil {
		for _, v := range globals.Value.Arr() {
			meta.JSGlobals = append(meta.JSGlobals, v.Str())
		}
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

	// Take initial screenshot
	quality := 80
	data, ssErr := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: &quality,
	})
	if ssErr == nil && len(data) > 0 {
		shot := base64.StdEncoding.EncodeToString(data)
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

// ScoutURLHeadless uses headless Chrome to render a page and extract metadata.
// This handles JS-rendered games that require execution to reveal framework details.
func ScoutURLHeadless(ctx context.Context, gameURL string, cfg HeadlessConfig) (*PageMeta, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Launch headless Chrome optimized for Phaser/WebGL game testing.
	// - HeadlessNew: uses --headless=new (Chrome 112+) which shares the full browser
	//   rendering pipeline, giving proper WebGL/canvas support unlike old --headless.
	// - SwiftShader: CPU-based Vulkan/GLES backend for WebGL without a real GPU.
	// - Autoplay: allow game audio without user gesture (Phaser Web Audio API).
	// - Font hinting: disabled for consistent screenshot rendering across environments.
	// Do NOT use --disable-gpu or --disable-software-rasterizer — they break WebGL.
	l := launcher.New().
		HeadlessNew(true).
		NoSandbox(true).
		Set("disable-dev-shm-usage").
		Set("use-gl", "angle").
		Set("use-angle", "swiftshader").
		Set("enable-unsafe-swiftshader").
		Set("autoplay-policy", "no-user-gesture-required").
		Set("font-render-hinting", "none")

	if bin := lookupChromeBin(); bin != "" {
		l = l.Bin(bin)
	}

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

	page.MustSetViewport(width, height, 1, false)

	// Collect console logs for framework clues
	var consoleLogs strings.Builder
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		for _, arg := range e.Args {
			str := arg.Value.Str()
			if str != "" {
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
	page.Context(ctx).WaitIdle(5 * time.Second)

	// Wait up to 5s for canvas to appear
	canvasFound := false
	for i := 0; i < 10; i++ {
		hasCanvas, evalErr := page.Eval(`() => document.querySelector('canvas') !== null`)
		if evalErr == nil && hasCanvas.Value.Bool() {
			canvasFound = true
			break
		}
		time.Sleep(500 * time.Millisecond)
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
	if evalErr == nil && globals.Value.Arr() != nil {
		for _, v := range globals.Value.Arr() {
			meta.JSGlobals = append(meta.JSGlobals, v.Str())
		}
		// Update framework from JS globals (authoritative)
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

	// Check for WebGL context via console logs
	consoleStr := consoleLogs.String()
	if meta.Framework == "unknown" && consoleStr != "" {
		detected := DetectFramework(nil, consoleStr)
		if detected != "unknown" {
			meta.Framework = detected
		}
	}

	// --- Multi-screenshot capture ---
	// Capture screenshots of multiple game states to give the AI visibility
	// beyond just the loading screen. Use JPEG at quality 80 to keep base64
	// size manageable for the multimodal AI API (~50-150 KB per shot).
	jpegQuality := 80
	captureScreenshot := func() string {
		data, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormatJpeg,
			Quality: &jpegQuality,
		})
		if err != nil || len(data) == 0 {
			return ""
		}
		return base64.StdEncoding.EncodeToString(data)
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

	// Screenshot 2: Click center of canvas to start the game
	if canvasFound {
		_, clickErr := page.Eval(fmt.Sprintf(`() => {
			const c = document.querySelector('canvas');
			if (c) {
				const rect = c.getBoundingClientRect();
				const evt = new MouseEvent('click', {
					clientX: rect.left + rect.width / 2,
					clientY: rect.top + rect.height / 2,
					bubbles: true
				});
				c.dispatchEvent(evt);
			}
		}`))
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
