package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/ai"
	"github.com/Global-Wizards/wizards-qa/pkg/scout"
	"github.com/Global-Wizards/wizards-qa/web/backend/store"
	"github.com/Global-Wizards/wizards-qa/web/backend/ws"
)

// launchAgentTestRun starts executeAgentTestRun in a goroutine with panic recovery.
func (s *Server) launchAgentTestRun(planID, testID, analysisID, planName, viewport, createdBy string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in agent test execution %s: %v", testID, r)
				s.finishTestRun(planID, testID, planName, time.Now(), nil, fmt.Errorf("panic: %v", r), createdBy)
			}
		}()
		s.executeAgentTestRun(planID, testID, analysisID, planName, createdBy, viewport)
	}()
}

// executeAgentTestRun runs test scenarios using an AI agent with browser tools.
// The agent receives each scenario's steps and autonomously executes them,
// calling report_result when done.
func (s *Server) executeAgentTestRun(planID, testID, analysisID, planName, createdBy, viewport string) {
	// Acquire browser test concurrency slot (only one Chrome for tests at a time)
	select {
	case s.browserTestSem <- struct{}{}:
		defer func() { <-s.browserTestSem }()
	case <-s.serverCtx.Done():
		s.finishTestRun(planID, testID, planName, time.Now(), nil, fmt.Errorf("server shutting down"), createdBy)
		return
	}

	startTime := time.Now()

	// Extract scenarios from analysis result
	scenarios, gameURL, err := s.extractScenariosFromAnalysis(analysisID)
	if err != nil {
		s.finishTestRun(planID, testID, planName, startTime, nil, fmt.Errorf("extracting scenarios: %w", err), createdBy)
		return
	}
	totalFlows := len(scenarios)

	if planID != "" {
		if err := s.store.UpdateTestPlanStatus(planID, store.StatusRunning, testID); err != nil {
			log.Printf("Warning: failed to update plan %s status to running: %v", planID, err)
		}
	}

	// Track running test state for reconnection
	rt := &runningTest{
		TestID:     testID,
		PlanID:     planID,
		PlanName:   planName,
		Mode:       ModeAgent,
		StartedAt:  startTime,
		TotalFlows: totalFlows,
		Flows:      []store.FlowResult{},
		Logs:       []string{},
		Status:     "running",
	}
	s.runningTests.Register(testID, rt)

	s.wsHub.Broadcast(ws.Message{
		Type: "test_started",
		Data: map[string]interface{}{
			"testId":     testID,
			"planId":     planID,
			"name":       planName,
			"totalFlows": totalFlows,
			"mode":       ModeAgent,
		},
	})

	// Resolve viewport
	vp := scout.GetViewportByName(viewport)
	if vp == nil {
		vp = scout.GetViewportByName(scout.DefaultViewportName)
	}

	// Create AI client
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		s.finishTestRun(planID, testID, planName, startTime, nil, fmt.Errorf("ANTHROPIC_API_KEY not set"), createdBy)
		return
	}
	aiModel := envOrDefault("WIZARDS_QA_TEST_MODEL", "claude-sonnet-4-5-20250929")
	aiClient := ai.NewClaudeClient(apiKey, aiModel, 0.3, 4096)

	// Launch headless browser
	ctx, cancel := context.WithTimeout(s.serverCtx, AnalysisTimeout)
	defer cancel()

	s.broadcastTestLog(testID, planID, "Launching headless browser for agent test execution...")

	_, browserPage, cleanup, err := scout.ScoutURLHeadlessKeepAlive(ctx, "about:blank", scout.HeadlessConfig{
		Enabled:          true,
		Width:            vp.Width,
		Height:           vp.Height,
		DevicePixelRatio: vp.DevicePixelRatio,
		Timeout:          30 * time.Second,
	})
	if err != nil {
		s.finishTestRun(planID, testID, planName, startTime, nil, fmt.Errorf("launching browser: %w", err), createdBy)
		return
	}
	defer cleanup()

	toolExec := &ai.BrowserToolExecutor{Page: browserPage}
	s.broadcastTestLog(testID, planID, fmt.Sprintf("Browser ready (%dx%d @ %.1fx)", vp.Width, vp.Height, vp.DevicePixelRatio))

	// Build tools: browser tools + report_result
	tools := testExecutorTools(vp.Width, vp.Height)

	var flowResults []store.FlowResult

	for fi, scenario := range scenarios {
		flowStart := time.Now()

		s.wsHub.Broadcast(ws.Message{
			Type: "test_flow_started",
			Data: map[string]interface{}{
				"testId":       testID,
				"flowName":     scenario.Name,
				"commandCount": len(scenario.Steps),
				"flowIndex":    fi,
			},
		})

		s.broadcastTestLog(testID, planID, fmt.Sprintf("--- Scenario %d/%d: %s ---", fi+1, totalFlows, scenario.Name))

		// Navigate to game URL
		s.broadcastTestLog(testID, planID, fmt.Sprintf("  Navigating to %s", gameURL))
		if err := browserPage.Navigate(gameURL); err != nil {
			s.broadcastTestLog(testID, planID, fmt.Sprintf("  ❌ Navigation failed: %v", err))
			fr := store.FlowResult{
				Name:     scenario.Name,
				Status:   store.StatusFailed,
				Duration: formatDuration(time.Since(flowStart)),
			}
			flowResults = append(flowResults, fr)
			s.wsHub.Broadcast(ws.Message{
				Type: "test_progress",
				Data: map[string]interface{}{
					"testId":   testID,
					"planId":   planID,
					"flowName": scenario.Name,
					"status":   "failed",
					"duration": formatDuration(time.Since(flowStart)),
				},
			})
			continue
		}
		time.Sleep(2 * time.Second) // Wait for page to settle

		// Take initial screenshot
		initialSS, _ := browserPage.CaptureScreenshot()

		// Build scenario description for the agent
		scenarioDesc := buildScenarioPrompt(scenario)

		// Build initial messages with screenshot (same pattern as agent.go)
		initialContent := []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("Execute the following test scenario:\n\n%s\n\nThe browser is already on the game page. Start executing the test steps now.", scenarioDesc),
			},
		}
		if initialSS != "" {
			initialContent = append(initialContent, map[string]interface{}{
				"type": "image",
				"source": map[string]interface{}{
					"type":       "base64",
					"media_type": "image/webp",
					"data":       initialSS,
				},
			})
		}
		messages := []ai.AgentMessage{
			{Role: "user", Content: initialContent},
		}

		// Agent loop — max 30 steps per scenario
		const maxSteps = 30
		flowPassed := false
		flowFailed := false
		var failReason string
		stepIndex := 0

		systemPrompt := agentTestSystemPrompt(vp.Width, vp.Height)

		for step := 0; step < maxSteps; step++ {
			if ctx.Err() != nil {
				failReason = "context cancelled"
				flowFailed = true
				break
			}

			resp, err := aiClient.CallWithTools(systemPrompt, messages, tools)
			if err != nil {
				failReason = fmt.Sprintf("AI call failed: %v", err)
				flowFailed = true
				break
			}

			// Append assistant response directly (same pattern as agent.go)
			messages = append(messages, ai.AgentMessage{Role: "assistant", Content: resp.Content})

			// Check if there are tool calls
			hasToolCalls := false
			for _, block := range resp.Content {
				if block.Type == "tool_use" {
					hasToolCalls = true
					break
				}
			}

			if !hasToolCalls {
				// No tool calls — agent stopped without report_result
				failReason = "agent stopped without calling report_result"
				flowFailed = true
				break
			}

			// Execute tool calls and build tool results
			var toolResults []interface{}

			for _, block := range resp.Content {
				if block.Type != "tool_use" {
					continue
				}

				// Check for report_result
				if block.Name == "report_result" {
					var reportInput struct {
						Status     string `json:"status"`
						Reason     string `json:"reason"`
						FailedStep *int   `json:"failedStep,omitempty"`
					}
					if err := json.Unmarshal(block.Input, &reportInput); err == nil {
						if reportInput.Status == "passed" {
							flowPassed = true
						} else {
							flowFailed = true
							failReason = reportInput.Reason
						}
					}
					toolResults = append(toolResults, ai.ToolResultBlock{
						Type:      "tool_result",
						ToolUseID: block.ID,
						Content:   "Result recorded.",
					})
					break
				}

				// Execute browser tool
				stepStart := time.Now()
				textResult, screenshotB64, toolErr := toolExec.Execute(block.Name, block.Input)

				status := "passed"
				if toolErr != nil {
					textResult = fmt.Sprintf("Error: %v", toolErr)
					status = "failed"
				}

				stepDuration := time.Since(stepStart)
				cmdDesc := block.Name

				s.broadcastTestLog(testID, planID, fmt.Sprintf("  Step %d: %s (%s) → %s", stepIndex+1, cmdDesc, stepDuration.Round(time.Millisecond), agentTruncate(textResult, 100)))

				s.wsHub.Broadcast(ws.Message{
					Type: "test_command_progress",
					Data: map[string]interface{}{
						"testId":    testID,
						"flowName":  scenario.Name,
						"stepIndex": stepIndex,
						"command":   cmdDesc,
						"status":    status,
					},
				})

				// Save and broadcast screenshot
				if screenshotB64 != "" {
					screenshotURL := ""
					dataDir := s.store.DataDir()
					if dataDir != "" {
						dstDir := filepath.Join(dataDir, "test-screenshots", testID)
						if mkErr := os.MkdirAll(dstDir, 0755); mkErr == nil {
							safeName := strings.ReplaceAll(scenario.Name, " ", "_")
							fname := fmt.Sprintf("flow-%s-step-%d.webp", safeName, stepIndex)
							dstPath := filepath.Join(dstDir, fname)
							if imgData, decErr := base64.StdEncoding.DecodeString(screenshotB64); decErr == nil {
								if writeErr := os.WriteFile(dstPath, imgData, 0644); writeErr == nil {
									screenshotURL = fmt.Sprintf("/api/tests/%s/steps/%s/%d/screenshot", testID, url.PathEscape(scenario.Name), stepIndex)
								}
							}
						}
					}
					s.wsHub.Broadcast(ws.Message{
						Type: "test_step_screenshot",
						Data: map[string]interface{}{
							"testId":        testID,
							"flowName":      scenario.Name,
							"stepIndex":     stepIndex,
							"command":       cmdDesc,
							"screenshotUrl": screenshotURL,
							"result":        agentTruncate(textResult, 200),
							"status":        status,
						},
					})
				}

				// Build tool result for the AI (same pattern as agent.go)
				if toolErr != nil {
					toolResults = append(toolResults, ai.ToolResultBlock{
						Type:      "tool_result",
						ToolUseID: block.ID,
						Content:   "Error: " + toolErr.Error(),
						IsError:   true,
					})
				} else if screenshotB64 != "" {
					toolResults = append(toolResults, ai.ToolResultBlock{
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
									"media_type": "image/webp",
									"data":       screenshotB64,
								},
							},
						},
					})
				} else {
					toolResults = append(toolResults, ai.ToolResultBlock{
						Type:      "tool_result",
						ToolUseID: block.ID,
						Content:   textResult,
					})
				}

				stepIndex++
			}

			if len(toolResults) > 0 {
				messages = append(messages, ai.AgentMessage{Role: "user", Content: toolResults})
			}

			// Prune old screenshots to keep context manageable
			pruneAgentTestScreenshots(messages, 5)

			if flowPassed || flowFailed {
				break
			}
		}

		// If agent exhausted steps without reporting
		if !flowPassed && !flowFailed {
			flowFailed = true
			failReason = "agent exhausted maximum steps without reporting result"
		}

		flowDuration := time.Since(flowStart)
		flowStatus := store.StatusPassed
		if flowFailed {
			flowStatus = store.StatusFailed
			s.broadcastTestLog(testID, planID, fmt.Sprintf("  ❌ %s: %s", scenario.Name, failReason))
		} else {
			s.broadcastTestLog(testID, planID, fmt.Sprintf("  ✅ %s", scenario.Name))
		}

		fr := store.FlowResult{
			Name:     scenario.Name,
			Status:   flowStatus,
			Duration: formatDuration(flowDuration),
			Reason:   failReason,
		}
		flowResults = append(flowResults, fr)

		// Update running test state
		s.runningTests.AppendFlow(testID, fr)

		statusEmoji := "✅"
		if flowFailed {
			statusEmoji = "❌"
		}
		logLine := fmt.Sprintf("  %s %d. %s (%s)", statusEmoji, fi+1, scenario.Name, formatDuration(flowDuration))
		if flowFailed {
			logLine += " - " + failReason
		}

		s.runningTests.AppendLog(testID, logLine)

		s.wsHub.Broadcast(ws.Message{
			Type: "test_progress",
			Data: map[string]interface{}{
				"testId":   testID,
				"planId":   planID,
				"line":     logLine,
				"flowName": scenario.Name,
				"status":   flowStatus,
				"duration": formatDuration(flowDuration),
			},
		})
	}

	s.finishTestRun(planID, testID, planName, startTime, flowResults, nil, createdBy)
}

// extractScenariosFromAnalysis loads the analysis result from the DB and extracts TestScenario data.
func (s *Server) extractScenariosFromAnalysis(analysisID string) ([]ai.TestScenario, string, error) {
	analysis, err := s.store.GetAnalysis(analysisID)
	if err != nil {
		return nil, "", fmt.Errorf("getting analysis: %w", err)
	}

	resultMap, ok := analysis.Result.(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("analysis has no structured result")
	}

	// The analysis result contains an "analysis" key with the comprehensive result
	analysisData, ok := resultMap["analysis"].(map[string]interface{})
	if !ok {
		// Try top-level (some result formats have scenarios at top level)
		analysisData = resultMap
	}

	scenariosRaw, ok := analysisData["scenarios"].([]interface{})
	if !ok || len(scenariosRaw) == 0 {
		return nil, "", fmt.Errorf("no scenarios found in analysis result")
	}

	// Re-marshal and unmarshal to get typed scenarios
	scenariosJSON, err := json.Marshal(scenariosRaw)
	if err != nil {
		return nil, "", fmt.Errorf("marshaling scenarios: %w", err)
	}

	var scenarios []ai.TestScenario
	if err := json.Unmarshal(scenariosJSON, &scenarios); err != nil {
		return nil, "", fmt.Errorf("unmarshaling scenarios: %w", err)
	}

	return scenarios, analysis.GameURL, nil
}

// testExecutorTools returns browser tools plus the report_result tool.
func testExecutorTools(vpWidth, vpHeight int) []ai.ToolDefinition {
	tools := ai.BrowserTools(vpWidth, vpHeight)
	tools = append(tools, ai.ToolDefinition{
		Name:        "report_result",
		Description: "Report the final result of the current test scenario. Call this when you have finished executing all steps and verified the expected outcomes, OR when a step has failed and you want to report the failure.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"passed", "failed"},
					"description": "Whether the test scenario passed or failed",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Explanation of why the test passed or failed",
				},
				"failedStep": map[string]interface{}{
					"type":        "integer",
					"description": "The 1-based index of the step that failed (only for failed tests)",
				},
			},
			"required": []string{"status", "reason"},
		},
	})
	return tools
}

// agentTestSystemPrompt returns the system prompt for the test executor agent.
func agentTestSystemPrompt(vpWidth, vpHeight int) string {
	return fmt.Sprintf(`You are a QA test executor agent. Your job is to execute test scenarios on a web game/application using browser automation tools.

## Instructions

1. You will receive a test scenario with named steps describing actions to perform and expected outcomes to verify.
2. Execute each step using the available browser tools (click, type_text, scroll, navigate, screenshot, etc.).
3. After each action, observe the screenshot to verify the result matches expectations.
4. If a step fails (expected element not found, action doesn't produce expected result), try reasonable recovery strategies (wait, scroll, retry) before reporting failure.
5. When all steps are complete and verified, call report_result with status "passed".
6. If a step cannot be completed after reasonable attempts, call report_result with status "failed" and explain which step failed and why.

## Browser Info

- Viewport: %dx%d pixels
- Tools that modify the page (click, type_text, scroll, navigate) automatically return a screenshot.
- Use the screenshot tool only when you need to observe without interacting.

## Important

- Execute steps in order.
- Be precise with click coordinates — look at the screenshot carefully to find the correct UI elements.
- Wait briefly after navigation or major state changes for the page to settle.
- Do NOT invent steps that aren't in the scenario.
- Always call report_result when done — never leave a scenario without a verdict.`, vpWidth, vpHeight)
}

// buildScenarioPrompt formats a TestScenario into a readable prompt for the agent.
func buildScenarioPrompt(scenario ai.TestScenario) string {
	desc := fmt.Sprintf("## %s\n\n", scenario.Name)
	if scenario.Description != "" {
		desc += scenario.Description + "\n\n"
	}
	desc += "### Steps:\n"
	for i, step := range scenario.Steps {
		desc += fmt.Sprintf("%d. **%s** — %s", i+1, step.Action, step.Target)
		if step.Value != "" {
			desc += fmt.Sprintf(" (value: %q)", step.Value)
		}
		if step.Expected != "" {
			desc += fmt.Sprintf("\n   Expected: %s", step.Expected)
		}
		desc += "\n"
	}
	return desc
}

// agentTruncate shortens a string to maxLen characters with ellipsis.
func agentTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// pruneAgentTestScreenshots removes old screenshots from messages, keeping the most recent ones.
// Uses the same approach as pruneOldScreenshots in pkg/ai/agent.go.
func pruneAgentTestScreenshots(messages []ai.AgentMessage, keepRecent int) {
	imageCount := 0
	// Count backwards so we keep the most recent
	for i := len(messages) - 1; i >= 0; i-- {
		messages[i].Content = stripAgentImages(messages[i].Content, &imageCount, keepRecent)
	}
}

// stripAgentImages recursively walks a content value and replaces image blocks beyond the
// keepRecent threshold with a lightweight text placeholder.
func stripAgentImages(v interface{}, count *int, keep int) interface{} {
	switch c := v.(type) {
	case []interface{}:
		for i, item := range c {
			c[i] = stripAgentImages(item, count, keep)
		}
		return c
	case map[string]interface{}:
		if c["type"] == "image" {
			*count++
			if *count > keep {
				return map[string]interface{}{
					"type": "text",
					"text": "[Screenshot removed]",
				}
			}
		}
		if c["type"] == "tool_result" {
			if content, ok := c["content"]; ok {
				c["content"] = stripAgentImages(content, count, keep)
			}
		}
		return c
	case ai.ToolResultBlock:
		if content, ok := c.Content.([]interface{}); ok {
			c.Content = stripAgentImages(content, count, keep)
		}
		return c
	case []ai.ResponseContentBlock:
		// Assistant messages use this type — no images here
		return c
	}
	return v
}
