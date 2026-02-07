package scout

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// ScoutURLHeadless uses headless Chrome to render a page and extract metadata.
// This handles JS-rendered games that require execution to reveal framework details.
func ScoutURLHeadless(ctx context.Context, gameURL string, cfg HeadlessConfig) (*PageMeta, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Launch headless Chrome
	l, err := launcher.New().Headless(true).Launch()
	if err != nil {
		return nil, fmt.Errorf("launching headless browser: %w", err)
	}

	browser := rod.New().ControlURL(l)
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

	// Brief wait for JS frameworks to initialize
	page.Context(ctx).WaitIdle(2 * time.Second)

	// Extract rendered HTML
	htmlContent, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("getting page HTML: %w", err)
	}

	// Parse the rendered HTML using existing parser
	meta := ParseHTML(htmlContent)

	// Check for WebGL context via console logs
	consoleStr := consoleLogs.String()
	if meta.Framework == "unknown" && consoleStr != "" {
		// Try to detect framework from console output
		detected := DetectFramework(nil, consoleStr)
		if detected != "unknown" {
			meta.Framework = detected
		}
	}

	return meta, nil
}
