package scout

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

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

	// Launch headless Chrome with container-safe flags
	l := launcher.New().
		Headless(true).
		NoSandbox(true).
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		Set("disable-software-rasterizer")

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

	// Screenshot capture — use JPEG at quality 80 to keep base64 size
	// manageable for the multimodal AI API (~50-150 KB vs 200-500 KB for PNG).
	jpegQuality := 80
	screenshotData, screenshotErr := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: &jpegQuality,
	})
	if screenshotErr == nil && len(screenshotData) > 0 {
		meta.ScreenshotB64 = base64.StdEncoding.EncodeToString(screenshotData)
		if cfg.ScreenshotPath != "" {
			os.WriteFile(cfg.ScreenshotPath, screenshotData, 0644)
		}
	}

	return meta, nil
}
