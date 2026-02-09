package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/retry"
	"github.com/Global-Wizards/wizards-qa/pkg/scout"
)

// AgentExplore runs an agentic exploration loop where Claude interacts with a live browser page
// using tools (click, screenshot, type, etc.) and then synthesizes a structured analysis.
func (a *Analyzer) AgentExplore(
	ctx context.Context,
	browserPage BrowserPage,
	pageMeta *scout.PageMeta,
	gameURL string,
	cfg AgentConfig,
	onProgress ProgressFunc,
) (*ComprehensiveAnalysisResult, []AgentStep, error) {
	progress := func(step, message string) {
		if onProgress != nil {
			onProgress(step, message)
		}
	}

	agent, ok := a.Client.(ToolUseAgent)
	if !ok {
		return nil, nil, fmt.Errorf("AI client does not support tool use (agent mode requires Claude)")
	}

	tools := BrowserTools()
	executor := &BrowserToolExecutor{Page: browserPage}

	progress("agent_start", fmt.Sprintf("Starting agent exploration of %s (max %d steps)", gameURL, cfg.MaxSteps))

	// Take initial screenshot
	initialScreenshot, ssErr := browserPage.CaptureScreenshot()
	if ssErr != nil {
		return nil, nil, fmt.Errorf("initial screenshot failed: %w", ssErr)
	}

	// Capture any early console errors from page load
	var consoleSection string
	if consoleLogs, logErr := browserPage.GetConsoleLogs(); logErr == nil && len(consoleLogs) > 0 {
		// Include up to 30 lines of initial console output
		if len(consoleLogs) > 30 {
			consoleLogs = consoleLogs[len(consoleLogs)-30:]
		}
		consoleSection = fmt.Sprintf("\n\nBrowser console output during page load:\n%s", strings.Join(consoleLogs, "\n"))
	}

	// Build initial user message with page metadata + screenshot
	pageMetaJSON := buildPageMetaJSON(pageMeta)

	// Check for expired JWT tokens in the URL
	var tokenSection string
	tokenStatuses := checkURLTokenExpiry(gameURL)
	if len(tokenStatuses) > 0 {
		var parts []string
		for param, ts := range tokenStatuses {
			if ts.Expired {
				ago := time.Since(ts.ExpiresAt).Truncate(time.Minute)
				parts = append(parts, fmt.Sprintf("%s expired %s ago", param, ago))
			} else {
				remaining := time.Until(ts.ExpiresAt).Truncate(time.Minute)
				parts = append(parts, fmt.Sprintf("%s valid (%s remaining)", param, remaining))
			}
		}
		tokenSection = fmt.Sprintf("\n\nToken status: %s", strings.Join(parts, ", "))
	}

	initialContent := []interface{}{
		map[string]interface{}{
			"type": "image",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": "image/jpeg",
				"data":       initialScreenshot,
			},
		},
		map[string]interface{}{
			"type": "text",
			"text": fmt.Sprintf(`You are exploring a web-based game for QA testing.

Game URL: %s

Page metadata (auto-detected):
%s%s%s

Above is a screenshot of the initial page state. Begin your exploration by interacting with the game.
Remember to take screenshots after interactions to observe results.
When done exploring, include EXPLORATION_COMPLETE in your response.`, gameURL, string(pageMetaJSON), consoleSection, tokenSection),
		},
	}

	messages := []AgentMessage{
		{Role: "user", Content: initialContent},
	}

	var steps []AgentStep
	var allScreenshots []string
	allScreenshots = append(allScreenshots, initialScreenshot)

	totalStart := time.Now()

	// Reserve time for synthesis + flow generation (with retries) so exploration can't starve them
	synthesisReserve := 3 * time.Minute
	effectiveExplorationTimeout := cfg.TotalTimeout - synthesisReserve
	if effectiveExplorationTimeout < 2*time.Minute {
		effectiveExplorationTimeout = 2 * time.Minute
	}

	for step := 1; step <= cfg.MaxSteps; step++ {
		// Check exploration timeout (reserves time for synthesis)
		if cfg.TotalTimeout > 0 && time.Since(totalStart) > effectiveExplorationTimeout {
			progress("agent_step", fmt.Sprintf("Step %d: exploration timeout reached (reserving time for synthesis)", step))
			break
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, steps, ctx.Err()
		default:
		}

		// Non-blocking read of user hints
		if cfg.UserMessages != nil {
			select {
			case hint := <-cfg.UserMessages:
				progress("user_hint", hint)
				messages = append(messages, AgentMessage{
					Role:    "user",
					Content: fmt.Sprintf("[USER HINT]: %s\nPlease incorporate this guidance into your next action.", hint),
				})
			default:
			}
		}

		progress("agent_step", fmt.Sprintf("Step %d/%d: calling AI...", step, cfg.MaxSteps))

		resp, err := agent.CallWithTools(AgentSystemPrompt, messages, tools)
		if err != nil {
			return nil, steps, fmt.Errorf("agent step %d API call failed: %w", step, err)
		}

		// Append assistant response to messages
		messages = append(messages, AgentMessage{Role: "assistant", Content: resp.Content})

		// Emit AI reasoning text for live streaming
		for _, block := range resp.Content {
			if block.Type == "text" && block.Text != "" {
				progress("agent_reasoning", block.Text)
			}
		}

		// Check for EXPLORATION_COMPLETE in text blocks
		explorationComplete := false
		for _, block := range resp.Content {
			if block.Type == "text" && strings.Contains(block.Text, "EXPLORATION_COMPLETE") {
				explorationComplete = true
				break
			}
		}

		if explorationComplete || resp.StopReason == "end_turn" {
			// Check if there are any tool_use blocks to process first
			hasToolUse := false
			for _, block := range resp.Content {
				if block.Type == "tool_use" {
					hasToolUse = true
					break
				}
			}
			if !hasToolUse {
				progress("agent_step", fmt.Sprintf("Step %d: exploration complete (stop_reason=%s)", step, resp.StopReason))
				break
			}
		}

		if resp.StopReason != "tool_use" {
			// No tools to execute, we're done
			progress("agent_step", fmt.Sprintf("Step %d: no tool calls (stop_reason=%s)", step, resp.StopReason))
			break
		}

		// Execute each tool_use block
		var toolResults []interface{}
		for _, block := range resp.Content {
			if block.Type != "tool_use" {
				continue
			}

			toolStart := time.Now()
			progress("agent_action", fmt.Sprintf("Step %d: %s", step, formatToolAction(block.Name, block.Input)))

			textResult, screenshotB64, execErr := executor.Execute(block.Name, block.Input)

			stepRecord := AgentStep{
				StepNumber: step,
				ToolName:   block.Name,
				Input:      string(block.Input),
				DurationMs: int(time.Since(toolStart).Milliseconds()),
			}

			if execErr != nil {
				stepRecord.Error = execErr.Error()
				stepRecord.Result = "Error: " + execErr.Error()
				toolResults = append(toolResults, ToolResultBlock{
					Type:      "tool_result",
					ToolUseID: block.ID,
					Content:   "Error: " + execErr.Error(),
					IsError:   true,
				})
			} else {
				stepRecord.Result = textResult
				if screenshotB64 != "" {
					stepRecord.ScreenshotB64 = screenshotB64
					allScreenshots = append(allScreenshots, screenshotB64)

					// Return screenshot as image content block for the AI to see
					toolResults = append(toolResults, ToolResultBlock{
						Type:      "tool_result",
						ToolUseID: block.ID,
						Content: []interface{}{
							map[string]interface{}{
								"type": "text",
								"text": textResult,
							},
							map[string]interface{}{
								"type": "image",
								"source": map[string]interface{}{
									"type":       "base64",
									"media_type": "image/jpeg",
									"data":       screenshotB64,
								},
							},
						},
					})
				} else {
					toolResults = append(toolResults, ToolResultBlock{
						Type:      "tool_result",
						ToolUseID: block.ID,
						Content:   textResult,
					})
				}
			}

			stepRecord.DurationMs = int(time.Since(toolStart).Milliseconds())
			steps = append(steps, stepRecord)

			// Write screenshot to tmpDir for live streaming
			if cfg.ScreenshotDir != "" && screenshotB64 != "" {
				filename := fmt.Sprintf("step-%d-%s.jpg", step, block.Name)
				if raw, decErr := base64.StdEncoding.DecodeString(screenshotB64); decErr == nil {
					if err := os.WriteFile(filepath.Join(cfg.ScreenshotDir, filename), raw, 0644); err != nil {
						progress("agent_step", fmt.Sprintf("Warning: failed to write screenshot %s: %v", filename, err))
					}
					progress("agent_screenshot", filename)
				}
			}

			// Emit structured step detail for live streaming
			errStr := ""
			if execErr != nil {
				errStr = execErr.Error()
			}
			detail := map[string]interface{}{
				"stepNumber": step,
				"toolName":   block.Name,
				"input":      string(block.Input),
				"result":     truncate(textResult, 300),
				"error":      errStr,
				"durationMs": stepRecord.DurationMs,
			}
			if detailJSON, jsonErr := json.Marshal(detail); jsonErr == nil {
				progress("agent_step_detail", string(detailJSON))
			}
		}

		// Append tool results as a user message
		messages = append(messages, AgentMessage{Role: "user", Content: toolResults})

		// Prune old screenshots from conversation to prevent unbounded context growth.
		// Each base64 screenshot is ~100-200KB; without pruning, API calls escalate from
		// ~10s to 70s+ as screenshots accumulate, consuming the entire timeout budget.
		pruneOldScreenshots(messages, 4)
	}

	progress("agent_done", fmt.Sprintf("Agent exploration complete: %d steps, %d screenshots", len(steps), len(allScreenshots)))

	// Strip ALL screenshots before synthesis — the AI already observed them during
	// exploration and doesn't need them for structured JSON output. This reduces
	// the API payload by ~1.6MB and avoids input-too-large errors.
	pruneOldScreenshots(messages, 0)

	// --- Synthesis call ---
	progress("agent_synthesize", "Synthesizing analysis from exploration...")

	synthesisPrompt := `Based on your exploration of this game, provide a comprehensive QA analysis as a single JSON object.

You interacted with the game and observed its behavior through screenshots. Now produce a structured analysis based on ONLY what you actually observed during exploration.

Respond with a single JSON object matching this exact format:
{
  "gameInfo": {
    "name": "...",
    "description": "...",
    "genre": "...",
    "technology": "...",
    "features": ["..."]
  },
  "mechanics": [
    {
      "name": "...",
      "description": "...",
      "actions": ["click", "drag", etc.],
      "expected": "...",
      "priority": "high|medium|low"
    }
  ],
  "uiElements": [
    {
      "name": "...",
      "type": "button|canvas|input",
      "selector": "text or percentage coordinate",
      "location": {"x": "50%", "y": "80%"}
    }
  ],
  "userFlows": [
    {
      "name": "...",
      "description": "...",
      "steps": ["step 1", "step 2"],
      "expected": "...",
      "priority": "high|medium|low"
    }
  ],
  "edgeCases": [
    {
      "name": "...",
      "description": "...",
      "scenario": "...",
      "expected": "..."
    }
  ],
  "scenarios": [
    {
      "name": "...",
      "description": "...",
      "type": "happy-path|edge-case|failure",
      "steps": [
        {
          "action": "launch|click|input|wait|assert",
          "target": "description of target",
          "value": "",
          "expected": "what should happen",
          "coordinates": {"x": "50%", "y": "80%"}
        }
      ],
      "priority": "high|medium|low",
      "tags": ["smoke", "regression"]
    }
  ]
}

IMPORTANT: Base your analysis on what you actually observed during exploration. Include specific coordinates you discovered. Respond with valid JSON only.`

	// Add synthesis request as a user message (no tools for this call)
	messages = append(messages, AgentMessage{Role: "user", Content: synthesisPrompt})

	// Ensure synthesis has enough token budget for full JSON output
	if cfg.SynthesisMaxTokens > 0 {
		if bc, ok := a.Client.(*ClaudeClient); ok {
			origMaxTokens := bc.MaxTokens
			bc.MaxTokens = cfg.SynthesisMaxTokens
			defer func() { bc.MaxTokens = origMaxTokens }()
		}
		if gc, ok := a.Client.(*GeminiClient); ok {
			origMaxTokens := gc.MaxTokens
			gc.MaxTokens = cfg.SynthesisMaxTokens
			defer func() { gc.MaxTokens = origMaxTokens }()
		}
	}

	// Call without tools to get structured JSON — retry up to 3 times with backoff
	var synthResp *ToolUseResponse
	retryCfg := &retry.Config{
		MaxAttempts:  3,
		InitialDelay: 5 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
	synthAttempt := 0
	synthErr := retry.Do(ctx, retryCfg, func() error {
		synthAttempt++
		if synthAttempt > 1 {
			progress("synthesis_retry", fmt.Sprintf("Retrying synthesis (attempt %d/%d)...", synthAttempt, retryCfg.MaxAttempts))
		}
		var err error
		synthResp, err = agent.CallWithTools(AgentSystemPrompt, messages, nil)
		return err
	})
	if synthErr != nil {
		return nil, steps, fmt.Errorf("synthesis call failed: %w", synthErr)
	}

	if synthResp.StopReason == "max_tokens" {
		log.Printf("WARNING: Synthesis truncated (stop_reason=max_tokens, %d output tokens)", synthResp.Usage.OutputTokens)
	}

	// Extract text from synthesis response
	var synthesisText string
	for _, block := range synthResp.Content {
		if block.Type == "text" {
			synthesisText += block.Text
		}
	}

	// Parse as ComprehensiveAnalysisResult
	parsed, parseErr := parseComprehensiveJSON(synthesisText)
	if parseErr != nil && synthResp.StopReason == "max_tokens" {
		// JSON was truncated — try to repair by closing open brackets
		repaired, repairErr := repairTruncatedJSON(synthesisText)
		if repairErr == nil {
			parsed, parseErr = parseComprehensiveJSON(repaired)
		}
	}
	if parseErr != nil {
		return nil, steps, fmt.Errorf("failed to parse synthesis response: %w (stop_reason=%s, raw: %s)", parseErr, synthResp.StopReason, truncate(synthesisText, 500))
	}

	return parsed, steps, nil
}

// AnalyzeFromURLWithAgent runs the full agent pipeline:
// 1. AgentExplore (agentic loop with browser tools)
// 2. generateFlowsStructured (reuse existing Call #2)
func (a *Analyzer) AnalyzeFromURLWithAgent(
	ctx context.Context,
	browserPage BrowserPage,
	pageMeta *scout.PageMeta,
	gameURL string,
	agentCfg AgentConfig,
	onProgress ProgressFunc,
) (*scout.PageMeta, *AnalysisResult, []*MaestroFlow, []AgentStep, error) {
	progress := func(step, message string) {
		if onProgress != nil {
			onProgress(step, message)
		}
	}

	// Step 1: Agent exploration
	comprehensiveResult, agentSteps, err := a.AgentExplore(ctx, browserPage, pageMeta, gameURL, agentCfg, onProgress)
	if err != nil {
		return pageMeta, nil, nil, agentSteps, fmt.Errorf("agent exploration failed: %w", err)
	}

	result := comprehensiveResult.ToAnalysisResult()
	scenarios := comprehensiveResult.Scenarios

	// Report analysis results
	analysisDetail := fmt.Sprintf("Found %d mechanics, %d UI elements, %d user flows, %d edge cases",
		len(result.Mechanics), len(result.UIElements), len(result.UserFlows), len(result.EdgeCases))
	if result.GameInfo.Name != "" {
		analysisDetail = fmt.Sprintf("%s — %s (%s)", result.GameInfo.Name, result.GameInfo.Genre, analysisDetail)
	}
	progress("analyzed", analysisDetail)

	if len(scenarios) > 0 {
		progress("scenarios_done", fmt.Sprintf("Generated %d scenarios from agent exploration", len(scenarios)))
	}

	// Step 2: Flow generation — collect last 5 agent screenshots for grounding
	var flowScreenshots []string
	for i := len(agentSteps) - 1; i >= 0 && len(flowScreenshots) < 5; i-- {
		if agentSteps[i].ScreenshotB64 != "" {
			flowScreenshots = append(flowScreenshots, agentSteps[i].ScreenshotB64)
		}
	}
	// Add initial screenshot from pageMeta if we have fewer than 5
	if len(flowScreenshots) < 5 && pageMeta.ScreenshotB64 != "" {
		flowScreenshots = append(flowScreenshots, pageMeta.ScreenshotB64)
	}

	progress("flows", fmt.Sprintf("Converting %d scenarios to Maestro flows...", len(scenarios)))

	// Retry flow generation up to 3 times with backoff
	var flows []*MaestroFlow
	flowRetryCfg := &retry.Config{
		MaxAttempts:  3,
		InitialDelay: 5 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
	flowAttempt := 0
	flowErr := retry.Do(ctx, flowRetryCfg, func() error {
		flowAttempt++
		if flowAttempt > 1 {
			progress("flows_retry", fmt.Sprintf("Retrying flow generation (attempt %d/%d)...", flowAttempt, flowRetryCfg.MaxAttempts))
		}
		var err error
		flows, err = a.generateFlowsStructured(gameURL, pageMeta.Framework, result, scenarios, flowScreenshots)
		return err
	})
	if flowErr != nil {
		return pageMeta, result, nil, agentSteps, fmt.Errorf("flow generation failed: %w", flowErr)
	}

	for _, flow := range flows {
		flow.URL = gameURL
	}

	totalCommands := 0
	for _, f := range flows {
		totalCommands += len(f.Commands)
	}
	progress("flows_done", fmt.Sprintf("Generated %d flows with %d total commands", len(flows), totalCommands))

	return pageMeta, result, flows, agentSteps, nil
}

// formatToolAction creates a human-readable description of a tool call.
func formatToolAction(toolName string, inputJSON json.RawMessage) string {
	switch toolName {
	case "screenshot":
		return "taking screenshot"
	case "click":
		var p struct{ X, Y int }
		json.Unmarshal(inputJSON, &p)
		return fmt.Sprintf("click at (%d, %d)", p.X, p.Y)
	case "type_text":
		var p struct{ Text string }
		json.Unmarshal(inputJSON, &p)
		return fmt.Sprintf("type %q", truncate(p.Text, 30))
	case "scroll":
		var p struct {
			Direction string
			Amount    int
		}
		json.Unmarshal(inputJSON, &p)
		return fmt.Sprintf("scroll %s %d", p.Direction, p.Amount)
	case "evaluate_js":
		var p struct{ Expression string }
		json.Unmarshal(inputJSON, &p)
		return fmt.Sprintf("eval JS: %s", truncate(p.Expression, 50))
	case "wait":
		var p struct {
			Milliseconds int
			Selector     string
		}
		json.Unmarshal(inputJSON, &p)
		if p.Selector != "" {
			return fmt.Sprintf("wait for %q", p.Selector)
		}
		return fmt.Sprintf("wait %dms", p.Milliseconds)
	case "get_page_info":
		return "get page info"
	case "console_logs":
		return "get console logs"
	case "navigate":
		var p struct{ URL string }
		json.Unmarshal(inputJSON, &p)
		return fmt.Sprintf("navigate to %s", truncate(p.URL, 60))
	default:
		return toolName
	}
}

// pruneOldScreenshots walks messages from newest to oldest and replaces base64 image
// data in screenshots beyond the keepRecent most recent ones. This prevents the API
// payload from growing unboundedly as screenshots accumulate in the conversation.
func pruneOldScreenshots(messages []AgentMessage, keepRecent int) {
	imageCount := 0
	for i := len(messages) - 1; i >= 0; i-- {
		messages[i].Content = stripImages(messages[i].Content, &imageCount, keepRecent)
	}
}

// stripImages recursively walks a content value and replaces image blocks beyond the
// keepRecent threshold with a lightweight text placeholder.
func stripImages(v interface{}, count *int, keep int) interface{} {
	switch c := v.(type) {
	case []interface{}:
		for i, item := range c {
			c[i] = stripImages(item, count, keep)
		}
		return c
	case map[string]interface{}:
		if c["type"] == "image" {
			*count++
			if *count > keep {
				return map[string]interface{}{
					"type": "text",
					"text": "[Screenshot removed — older than context window]",
				}
			}
		}
		if c["type"] == "tool_result" {
			if content, ok := c["content"]; ok {
				c["content"] = stripImages(content, count, keep)
			}
		}
		return c
	case ToolResultBlock:
		c.Content = stripImages(c.Content, count, keep)
		return c
	default:
		return v
	}
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// repairTruncatedJSON attempts to fix JSON truncated by max_tokens by
// closing any open brackets/braces. Returns repaired string or error.
func repairTruncatedJSON(s string) (string, error) {
	start := strings.Index(s, "{")
	if start < 0 {
		return "", fmt.Errorf("no JSON object found")
	}
	s = s[start:]

	var stack []rune
	inString := false
	escaped := false
	for _, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch r {
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}

	if len(stack) == 0 {
		return s, nil // Already balanced
	}

	// Trim trailing comma/whitespace, then close brackets
	trimmed := strings.TrimRight(s, " \t\n\r,")
	for i := len(stack) - 1; i >= 0; i-- {
		trimmed += string(stack[i])
	}
	return trimmed, nil
}
