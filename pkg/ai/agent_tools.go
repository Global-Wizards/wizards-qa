package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BrowserTools returns the tool definitions for browser interaction in agent mode.
// viewportWidth/viewportHeight are used in tool descriptions so the AI knows the coordinate space.
func BrowserTools(viewportWidth, viewportHeight int) []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "screenshot",
			Description: "Capture a screenshot of the current page state. The click, type_text, scroll, and navigate tools already return screenshots automatically — use this tool only when you need to observe the page without interacting.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "click",
			Description: fmt.Sprintf("Click at the given pixel coordinates on the page. The viewport is %dx%d. Returns a screenshot of the result.", viewportWidth, viewportHeight),
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"x": map[string]interface{}{
						"type":        "integer",
						"description": fmt.Sprintf("X coordinate in pixels (0-%d)", viewportWidth),
					},
					"y": map[string]interface{}{
						"type":        "integer",
						"description": fmt.Sprintf("Y coordinate in pixels (0-%d)", viewportHeight),
					},
				},
				"required": []string{"x", "y"},
			},
		},
		{
			Name:        "type_text",
			Description: "Type text using the keyboard. Optionally click at coordinates first to focus an element. Returns a screenshot of the result.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The text to type",
					},
					"x": map[string]interface{}{
						"type":        "integer",
						"description": "Optional: X coordinate to click before typing",
					},
					"y": map[string]interface{}{
						"type":        "integer",
						"description": "Optional: Y coordinate to click before typing",
					},
				},
				"required": []string{"text"},
			},
		},
		{
			Name:        "scroll",
			Description: "Scroll the page in a given direction. Returns a screenshot of the result.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"direction": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"up", "down", "left", "right"},
						"description": "Direction to scroll",
					},
					"amount": map[string]interface{}{
						"type":        "integer",
						"description": "Amount to scroll in pixels (default 300)",
					},
				},
				"required": []string{"direction"},
			},
		},
		{
			Name:        "evaluate_js",
			Description: "Run a JavaScript expression in the page context and return the result. Useful for checking game state, reading element properties, or inspecting DOM structure.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "JavaScript expression to evaluate",
					},
				},
				"required": []string{"expression"},
			},
		},
		{
			Name:        "wait",
			Description: "Wait for a specified duration or until a CSS selector becomes visible.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"milliseconds": map[string]interface{}{
						"type":        "integer",
						"description": "Duration to wait in milliseconds",
					},
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector to wait for (waits until visible, up to 5s)",
					},
				},
			},
		},
		{
			Name:        "get_page_info",
			Description: "Get the current page title, URL, and visible text content. Useful for understanding page structure when screenshots are ambiguous.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "console_logs",
			Description: "Get recent browser console messages (errors, warnings, logs). Use this to diagnose loading failures, JS errors, and game initialization problems.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "navigate",
			Description: "Navigate to a URL or reload the current page. Use this to retry loading a game that failed to initialize, or to navigate to a different URL.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to navigate to. Use the current game URL to reload.",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "press_key",
			Description: "Press a keyboard key. Useful for game controls (Space=spin, Enter=confirm, Escape=close, arrow keys=navigate). Returns a screenshot.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Key to press: Enter, Space, Escape, Tab, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Backspace, Delete, or a single character (a-z, 0-9)",
					},
				},
				"required": []string{"key"},
			},
		},
		{
			Name:        "inspect_game_objects",
			Description: "List interactive game objects from the Phaser/PixiJS scene graph with their screen coordinates. Use this to find clickable buttons, their exact positions, and their current state. Only works with Phaser 3 and PixiJS games.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
	}
}

// AgentTools returns browser tools, optionally including request_more_steps and request_more_time
// for adaptive exploration and dynamic timeout.
func AgentTools(cfg AgentConfig) []ToolDefinition {
	tools := BrowserTools(cfg.ViewportWidth, cfg.ViewportHeight)
	if cfg.AdaptiveExploration {
		tools = append(tools, ToolDefinition{
			Name:        "request_more_steps",
			Description: "Request additional exploration steps when you determine there are significant unexplored areas of the game. Call this before you run out of steps. Steps are granted in increments of 5, up to 2x the initial step budget. Provide a reason explaining what remains to explore and how many additional steps you need.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Why more steps are needed — what areas remain unexplored",
					},
					"additional_steps": map[string]interface{}{
						"type":        "integer",
						"description": "Number of additional steps requested (will be capped at the maximum)",
					},
				},
				"required": []string{"reason", "additional_steps"},
			},
		})
	}
	if cfg.AdaptiveTimeout {
		tools = append(tools, ToolDefinition{
			Name:        "request_more_time",
			Description: "Request additional exploration time when significant game areas remain unexplored and you're running low on time. Time is granted in increments of 25% of the initial timeout, up to 2x the initial timeout. Call this proactively before you're cut off.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Why more time is needed — what areas remain unexplored",
					},
					"additional_minutes": map[string]interface{}{
						"type":        "integer",
						"description": "Additional minutes requested (will be capped at the maximum)",
					},
				},
				"required": []string{"reason", "additional_minutes"},
			},
		})
	}
	return tools
}

// Screenshot timeout limits — prevents slow WebGL games (SwiftShader) from
// consuming the entire analysis timeout budget on screenshots.
// Values are generous because SwiftShader (CPU-based GPU) is slow on complex
// WebGL games (e.g. Phaser 4 with heavy canvas compositing).
const (
	screenshotTimeout     = 20 * time.Second // auto-screenshots after click/type/scroll/navigate
	screenshotToolTimeout = 30 * time.Second // explicit screenshot tool
)

type ssResult struct {
	b64 string
	err error
}

// captureScreenshotOnce wraps CaptureScreenshot with a single-attempt timeout.
func captureScreenshotOnce(page BrowserPage, timeout time.Duration) (string, error) {
	ch := make(chan ssResult, 1)
	go func() {
		b64, err := page.CaptureScreenshot()
		ch <- ssResult{b64, err}
	}()
	select {
	case res := <-ch:
		return res.b64, res.err
	case <-time.After(timeout):
		return "", nil
	}
}

// captureScreenshotWithTimeout wraps CaptureScreenshot with a timeout and
// one automatic retry. Complex WebGL games rendered via SwiftShader sometimes
// need two attempts — the first screenshot can stall while the compositor
// finishes a heavy frame, but a second attempt often succeeds quickly once
// the frame buffer is ready.
func captureScreenshotWithTimeout(page BrowserPage, timeout time.Duration) (string, error) {
	return CaptureScreenshotWithTimeout(page, timeout)
}

// CaptureScreenshotWithTimeout is the exported version of captureScreenshotWithTimeout.
// Use this from external packages (e.g. web backend) to prevent CaptureScreenshot from
// stalling indefinitely on complex WebGL/SwiftShader pages.
func CaptureScreenshotWithTimeout(page BrowserPage, timeout time.Duration) (string, error) {
	b64, err := captureScreenshotOnce(page, timeout)
	if err != nil || b64 != "" {
		return b64, err
	}
	// First attempt timed out — retry once with the same timeout.
	return captureScreenshotOnce(page, timeout)
}

// BrowserToolExecutor executes browser tool calls against a BrowserPage.
type BrowserToolExecutor struct {
	Page         BrowserPage
	recentClicks []clickRecord // tracks recent click coordinates for dedup detection
}

type clickRecord struct {
	x, y int
}

// checkClickRepetition tracks a click and returns a warning if the same
// coordinates have been clicked 3+ times in a row (within 30px tolerance).
func (e *BrowserToolExecutor) checkClickRepetition(x, y int) string {
	e.recentClicks = append(e.recentClicks, clickRecord{x, y})
	// Keep only last 5 clicks
	if len(e.recentClicks) > 5 {
		e.recentClicks = e.recentClicks[len(e.recentClicks)-5:]
	}
	if len(e.recentClicks) < 3 {
		return ""
	}
	// Check if last 3 clicks are within 30px of each other
	n := len(e.recentClicks)
	a, b, c := e.recentClicks[n-3], e.recentClicks[n-2], e.recentClicks[n-1]
	if intAbs(a.x-c.x) < 30 && intAbs(a.y-c.y) < 30 &&
		intAbs(b.x-c.x) < 30 && intAbs(b.y-c.y) < 30 {
		return " WARNING: You have clicked near these coordinates 3+ times with no visible change. The element may not be interactive, or the game may be in an animation/transition state. Try: (1) wait 3-5 seconds for animations to complete, (2) use evaluate_js to check game state (e.g., window.game.scene.scenes[0].sys.settings.status), (3) try clicking different coordinates, (4) move on to explore other areas."
	}
	return ""
}

func intAbs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Execute runs a tool by name with the given JSON input. Returns text result, optional screenshot, and error.
func (e *BrowserToolExecutor) Execute(toolName string, inputJSON json.RawMessage) (textResult string, screenshotB64 string, err error) {
	switch toolName {
	case "screenshot":
		b64, ssErr := captureScreenshotWithTimeout(e.Page, screenshotToolTimeout)
		if ssErr != nil {
			return "", "", fmt.Errorf("screenshot: %w", ssErr)
		}
		if b64 == "" {
			return "Screenshot timed out — page may have complex rendering. Try again or continue without it.", "", nil
		}
		return "Screenshot captured successfully.", b64, nil

	case "click":
		var params struct {
			X int `json:"x"`
			Y int `json:"y"`
		}
		if err := json.Unmarshal(inputJSON, &params); err != nil {
			return "", "", fmt.Errorf("click: invalid params: %w", err)
		}
		if err := e.Page.Click(params.X, params.Y); err != nil {
			return "", "", fmt.Errorf("click: %w", err)
		}
		// Brief pause for the page to react
		time.Sleep(150 * time.Millisecond)
		// Auto-capture screenshot after click (with timeout to prevent slow WebGL from blocking)
		b64, _ := captureScreenshotWithTimeout(e.Page, screenshotTimeout)
		msg := fmt.Sprintf("Clicked at (%d, %d).", params.X, params.Y)
		if warn := e.checkClickRepetition(params.X, params.Y); warn != "" {
			msg += warn
		}
		return msg, b64, nil

	case "type_text":
		var params struct {
			Text string `json:"text"`
			X    *int   `json:"x,omitempty"`
			Y    *int   `json:"y,omitempty"`
		}
		if err := json.Unmarshal(inputJSON, &params); err != nil {
			return "", "", fmt.Errorf("type_text: invalid params: %w", err)
		}
		// Click first if coordinates provided
		if params.X != nil && params.Y != nil {
			if err := e.Page.Click(*params.X, *params.Y); err != nil {
				return "", "", fmt.Errorf("type_text click: %w", err)
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err := e.Page.TypeText(params.Text); err != nil {
			return "", "", fmt.Errorf("type_text: %w", err)
		}
		// Auto-capture screenshot after typing (with timeout to prevent slow WebGL from blocking)
		b64, _ := captureScreenshotWithTimeout(e.Page, screenshotTimeout)
		return fmt.Sprintf("Typed %q.", params.Text), b64, nil

	case "scroll":
		var params struct {
			Direction string `json:"direction"`
			Amount    int    `json:"amount"`
		}
		if err := json.Unmarshal(inputJSON, &params); err != nil {
			return "", "", fmt.Errorf("scroll: invalid params: %w", err)
		}
		amount := float64(params.Amount)
		if amount == 0 {
			amount = 300
		}
		var dx, dy float64
		switch params.Direction {
		case "up":
			dy = -amount
		case "down":
			dy = amount
		case "left":
			dx = -amount
		case "right":
			dx = amount
		default:
			return "", "", fmt.Errorf("scroll: invalid direction %q", params.Direction)
		}
		if err := e.Page.Scroll(dx, dy); err != nil {
			return "", "", fmt.Errorf("scroll: %w", err)
		}
		time.Sleep(150 * time.Millisecond)
		// Auto-capture screenshot after scroll (with timeout to prevent slow WebGL from blocking)
		b64, _ := captureScreenshotWithTimeout(e.Page, screenshotTimeout)
		return fmt.Sprintf("Scrolled %s by %.0f pixels.", params.Direction, amount), b64, nil

	case "evaluate_js":
		var params struct {
			Expression string `json:"expression"`
		}
		if err := json.Unmarshal(inputJSON, &params); err != nil {
			return "", "", fmt.Errorf("evaluate_js: invalid params: %w", err)
		}
		result, evalErr := e.Page.EvalJS(params.Expression)
		if evalErr != nil {
			return "", "", fmt.Errorf("evaluate_js: %w", evalErr)
		}
		// Truncate very long results
		if len(result) > 2000 {
			result = result[:2000] + "... (truncated)"
		}
		return result, "", nil

	case "wait":
		var params struct {
			Milliseconds int    `json:"milliseconds"`
			Selector     string `json:"selector"`
		}
		if err := json.Unmarshal(inputJSON, &params); err != nil {
			return "", "", fmt.Errorf("wait: invalid params: %w", err)
		}
		if params.Selector != "" {
			waitErr := e.Page.WaitVisible(params.Selector, 5*time.Second)
			if waitErr != nil {
				return fmt.Sprintf("Selector %q not visible after 5s.", params.Selector), "", nil
			}
			return fmt.Sprintf("Selector %q is now visible.", params.Selector), "", nil
		}
		if params.Milliseconds > 0 {
			if params.Milliseconds > 10000 {
				params.Milliseconds = 10000 // cap at 10s
			}
			time.Sleep(time.Duration(params.Milliseconds) * time.Millisecond)
			return fmt.Sprintf("Waited %dms.", params.Milliseconds), "", nil
		}
		return "No wait parameters specified.", "", nil

	case "get_page_info":
		title, pageURL, visibleText, infoErr := e.Page.GetPageInfo()
		if infoErr != nil {
			return "", "", fmt.Errorf("get_page_info: %w", infoErr)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Title: %s\n", title))
		sb.WriteString(fmt.Sprintf("URL: %s\n", pageURL))
		if visibleText != "" {
			sb.WriteString(fmt.Sprintf("Visible Text:\n%s", visibleText))
		}
		return sb.String(), "", nil

	case "console_logs":
		logs, logErr := e.Page.GetConsoleLogs()
		if logErr != nil {
			return "", "", fmt.Errorf("console_logs: %w", logErr)
		}
		if len(logs) == 0 {
			return "No console messages captured.", "", nil
		}
		// Return last 50 lines max
		if len(logs) > 50 {
			logs = logs[len(logs)-50:]
		}
		return strings.Join(logs, "\n"), "", nil

	case "navigate":
		var params struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(inputJSON, &params); err != nil {
			return "", "", fmt.Errorf("navigate: invalid params: %w", err)
		}
		if params.URL == "" {
			return "", "", fmt.Errorf("navigate: url is required")
		}
		if err := e.Page.Navigate(params.URL); err != nil {
			return "", "", fmt.Errorf("navigate: %w", err)
		}
		// Take a screenshot after navigation (with timeout to prevent slow WebGL from blocking)
		b64, _ := captureScreenshotWithTimeout(e.Page, screenshotTimeout)
		return fmt.Sprintf("Navigated to %s.", params.URL), b64, nil

	case "press_key":
		var params struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(inputJSON, &params); err != nil {
			return "", "", fmt.Errorf("press_key: invalid params: %w", err)
		}
		if err := e.Page.PressKey(params.Key); err != nil {
			return "", "", fmt.Errorf("press_key: %w", err)
		}
		time.Sleep(150 * time.Millisecond)
		b64, _ := captureScreenshotWithTimeout(e.Page, screenshotTimeout)
		return fmt.Sprintf("Pressed key %q.", params.Key), b64, nil

	case "inspect_game_objects":
		result, err := e.Page.EvalJS(`(() => {
			// Phaser 3
			if (window.game && window.game.scene) {
				const scenes = window.game.scene.scenes.filter(s => s.sys.settings.status >= 5);
				const canvas = document.querySelector('canvas');
				const rect = canvas ? canvas.getBoundingClientRect() : {left:0, top:0, width:1920, height:1080};
				const scaleX = rect.width / (window.game.scale ? window.game.scale.width : canvas.width);
				const scaleY = rect.height / (window.game.scale ? window.game.scale.height : canvas.height);
				const objects = [];
				for (const scene of scenes) {
					scene.children.list.forEach(obj => {
						if (!obj.active || !obj.visible) return;
						const hasInput = obj.input && obj.input.enabled;
						const name = obj.name || obj.type || obj.constructor.name;
						const sx = Math.round(obj.x * scaleX + rect.left);
						const sy = Math.round(obj.y * scaleY + rect.top);
						const w = obj.displayWidth ? Math.round(obj.displayWidth * scaleX) : 0;
						const h = obj.displayHeight ? Math.round(obj.displayHeight * scaleY) : 0;
						if (hasInput || obj.type === 'Text' || obj.type === 'Sprite' || obj.type === 'Image') {
							objects.push({
								scene: scene.sys.settings.key,
								name: name,
								type: obj.type,
								interactive: hasInput,
								x: sx, y: sy, w: w, h: h,
								text: obj.text ? obj.text.substring(0, 50) : undefined
							});
						}
					});
				}
				return JSON.stringify({engine: 'phaser3', scenes: scenes.map(s => s.sys.settings.key), objects: objects}, null, 2);
			}
			// PixiJS
			if (window.__PIXI_APP__ || window.app) {
				const app = window.__PIXI_APP__ || window.app;
				const objects = [];
				function walk(node, depth) {
					if (depth > 5) return;
					if (node.interactive || node.buttonMode) {
						const b = node.getBounds();
						objects.push({name: node.name || node.constructor.name, interactive: true, x: Math.round(b.x), y: Math.round(b.y), w: Math.round(b.width), h: Math.round(b.height)});
					}
					if (node.children) node.children.forEach(c => walk(c, depth+1));
				}
				walk(app.stage, 0);
				return JSON.stringify({engine: 'pixi', objects: objects}, null, 2);
			}
			return JSON.stringify({error: 'No supported game engine detected (need Phaser 3 or PixiJS)'});
		})()`)
		if err != nil {
			return "", "", fmt.Errorf("inspect_game_objects: %w", err)
		}
		return result, "", nil

	default:
		return "", "", fmt.Errorf("unknown tool: %s", toolName)
	}
}
