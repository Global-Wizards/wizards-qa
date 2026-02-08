package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BrowserTools returns the tool definitions for browser interaction in agent mode.
func BrowserTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "screenshot",
			Description: "Capture a screenshot of the current page state. Use this after interactions to see the result.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "click",
			Description: "Click at the given pixel coordinates on the page. The viewport is 1920x1080.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"x": map[string]interface{}{
						"type":        "integer",
						"description": "X coordinate in pixels (0-1920)",
					},
					"y": map[string]interface{}{
						"type":        "integer",
						"description": "Y coordinate in pixels (0-1080)",
					},
				},
				"required": []string{"x", "y"},
			},
		},
		{
			Name:        "type_text",
			Description: "Type text using the keyboard. Optionally click at coordinates first to focus an element.",
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
			Description: "Scroll the page in a given direction.",
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
	}
}

// BrowserToolExecutor executes browser tool calls against a BrowserPage.
type BrowserToolExecutor struct {
	Page BrowserPage
}

// Execute runs a tool by name with the given JSON input. Returns text result, optional screenshot, and error.
func (e *BrowserToolExecutor) Execute(toolName string, inputJSON json.RawMessage) (textResult string, screenshotB64 string, err error) {
	switch toolName {
	case "screenshot":
		b64, ssErr := e.Page.CaptureScreenshot()
		if ssErr != nil {
			return "", "", fmt.Errorf("screenshot: %w", ssErr)
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
		time.Sleep(500 * time.Millisecond)
		return fmt.Sprintf("Clicked at (%d, %d).", params.X, params.Y), "", nil

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
			time.Sleep(200 * time.Millisecond)
		}
		if err := e.Page.TypeText(params.Text); err != nil {
			return "", "", fmt.Errorf("type_text: %w", err)
		}
		return fmt.Sprintf("Typed %q.", params.Text), "", nil

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
		time.Sleep(300 * time.Millisecond)
		return fmt.Sprintf("Scrolled %s by %.0f pixels.", params.Direction, amount), "", nil

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

	default:
		return "", "", fmt.Errorf("unknown tool: %s", toolName)
	}
}
