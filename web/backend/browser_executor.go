package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/ai"
	"github.com/Global-Wizards/wizards-qa/pkg/scout"
	"github.com/Global-Wizards/wizards-qa/web/backend/store"
	"github.com/Global-Wizards/wizards-qa/web/backend/ws"
	"gopkg.in/yaml.v3"
)

// browserFlowMeta holds parsed metadata from a flow YAML file.
type browserFlowMeta struct {
	AppID string
	URL   string
	Tags  []string
}

// browserFlowFile holds a parsed flow ready for browser execution.
type browserFlowFile struct {
	Name     string
	Path     string
	Meta     browserFlowMeta
	Commands []interface{}
}

// launchBrowserTestRun starts executeBrowserTestRun in a goroutine with panic recovery.
func (s *Server) launchBrowserTestRun(planID, testID, flowDir, planName, viewport string, cleanupDir bool, createdBy string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in browser test execution %s: %v", testID, r)
				s.finishTestRun(planID, testID, planName, time.Now(), nil, fmt.Errorf("panic: %v", r), createdBy)
			}
			if cleanupDir {
				if err := os.RemoveAll(flowDir); err != nil && !os.IsNotExist(err) {
					log.Printf("Warning: failed to clean up temp dir %s: %v", flowDir, err)
				}
			}
		}()
		s.executeBrowserTestRun(planID, testID, flowDir, planName, createdBy, viewport)
	}()
}

// executeBrowserTestRun runs test flows in headless Chrome using the browser automation infrastructure.
func (s *Server) executeBrowserTestRun(planID, testID, flowDir, planName, createdBy, viewport string) {
	startTime := time.Now()

	// Parse all flow files
	flows, err := parseFlowDir(flowDir)
	if err != nil {
		s.finishTestRun(planID, testID, planName, startTime, nil, fmt.Errorf("parsing flows: %w", err), createdBy)
		return
	}
	totalFlows := len(flows)

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
		StartedAt:  startTime,
		TotalFlows: totalFlows,
		Flows:      []store.FlowResult{},
		Logs:       []string{},
		Status:     "running",
	}
	s.runningTestsMu.Lock()
	s.runningTests[testID] = rt
	s.runningTestsMu.Unlock()

	s.wsHub.Broadcast(ws.Message{
		Type: "test_started",
		Data: map[string]interface{}{
			"testId":     testID,
			"planId":     planID,
			"name":       planName,
			"totalFlows": totalFlows,
			"mode":       "browser",
		},
	})

	// Resolve viewport
	vp := scout.GetViewportByName(viewport)
	if vp == nil {
		vp = scout.GetViewportByName(scout.DefaultViewportName)
	}

	// Create AI client for vision queries
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	var aiClient *ai.ClaudeClient
	if apiKey != "" {
		aiClient = ai.NewClaudeClient(apiKey, "claude-sonnet-4-5-20250929", 0.3, 1024)
	}

	// Launch headless browser
	ctx, cancel := context.WithTimeout(s.serverCtx, 10*time.Minute)
	defer cancel()

	s.broadcastTestLog(testID, planID, "Launching headless browser...")

	pageMeta, browserPage, cleanup, err := scout.ScoutURLHeadlessKeepAlive(ctx, "about:blank", scout.HeadlessConfig{
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
	_ = pageMeta

	toolExec := &ai.BrowserToolExecutor{Page: browserPage}
	s.broadcastTestLog(testID, planID, fmt.Sprintf("Browser ready (%dx%d @ %.1fx)", vp.Width, vp.Height, vp.DevicePixelRatio))

	var flowResults []store.FlowResult

	for fi, flow := range flows {
		flowStart := time.Now()

		s.wsHub.Broadcast(ws.Message{
			Type: "test_flow_started",
			Data: map[string]interface{}{
				"testId":       testID,
				"flowName":     flow.Name,
				"commandCount": len(flow.Commands),
				"flowIndex":    fi,
			},
		})

		s.broadcastTestLog(testID, planID, fmt.Sprintf("--- Flow %d/%d: %s (%d commands) ---", fi+1, totalFlows, flow.Name, len(flow.Commands)))

		// Navigate to flow URL if specified
		if flow.Meta.URL != "" {
			s.broadcastTestLog(testID, planID, fmt.Sprintf("  Navigating to %s", flow.Meta.URL))
			if err := browserPage.Navigate(flow.Meta.URL); err != nil {
				s.broadcastTestLog(testID, planID, fmt.Sprintf("  ❌ Navigation failed: %v", err))
				flowResults = append(flowResults, store.FlowResult{
					Name:     flow.Name,
					Status:   "failed",
					Duration: time.Since(flowStart).Round(time.Millisecond).String(),
					Output:   fmt.Sprintf("Navigation failed: %v", err),
				})
				s.wsHub.Broadcast(ws.Message{
					Type: "test_progress",
					Data: map[string]interface{}{
						"testId":   testID,
						"flowName": flow.Name,
						"status":   "failed",
						"duration": time.Since(flowStart).Round(time.Millisecond).String(),
					},
				})
				continue
			}
			time.Sleep(1 * time.Second) // Wait for page to settle
		}

		// Execute each command
		flowPassed := true
		var flowError string
		for ci, cmd := range flow.Commands {
			cmdDesc := describeCommand(cmd)

			s.wsHub.Broadcast(ws.Message{
				Type: "test_command_progress",
				Data: map[string]interface{}{
					"testId":    testID,
					"flowName":  flow.Name,
					"stepIndex": ci,
					"command":   cmdDesc,
					"status":    "running",
				},
			})

			result, screenshot, cmdErr := executeFlowCommand(browserPage, toolExec, cmd, aiClient, vp.Width, vp.Height)

			status := "passed"
			if cmdErr != nil {
				status = "failed"
				flowPassed = false
				flowError = cmdErr.Error()
				s.broadcastTestLog(testID, planID, fmt.Sprintf("  ❌ Step %d: %s → %v", ci+1, cmdDesc, cmdErr))
			} else {
				s.broadcastTestLog(testID, planID, fmt.Sprintf("  ✅ Step %d: %s → %s", ci+1, cmdDesc, result))
			}

			// Broadcast step screenshot
			if screenshot != "" {
				s.wsHub.Broadcast(ws.Message{
					Type: "test_step_screenshot",
					Data: map[string]interface{}{
						"testId":        testID,
						"flowName":      flow.Name,
						"stepIndex":     ci,
						"command":        cmdDesc,
						"screenshotB64": screenshot,
						"result":        result,
						"status":        status,
					},
				})
			}

			s.wsHub.Broadcast(ws.Message{
				Type: "test_command_progress",
				Data: map[string]interface{}{
					"testId":    testID,
					"flowName":  flow.Name,
					"stepIndex": ci,
					"command":   cmdDesc,
					"status":    status,
				},
			})

			if cmdErr != nil {
				break // Stop flow on first failure
			}
		}

		flowDuration := time.Since(flowStart)
		flowStatus := store.StatusPassed
		if !flowPassed {
			flowStatus = store.StatusFailed
		}

		fr := store.FlowResult{
			Name:     flow.Name,
			Status:   flowStatus,
			Duration: formatDuration(flowDuration),
		}
		flowResults = append(flowResults, fr)

		// Update running test state
		s.runningTestsMu.Lock()
		if rt, ok := s.runningTests[testID]; ok {
			rt.Flows = append(rt.Flows, fr)
		}
		s.runningTestsMu.Unlock()

		statusEmoji := "✅"
		if !flowPassed {
			statusEmoji = "❌"
		}
		logLine := fmt.Sprintf("  %s %d. %s (%s)", statusEmoji, fi+1, flow.Name, formatDuration(flowDuration))
		if !flowPassed {
			logLine += " - " + flowError
		}

		s.broadcastTestLog(testID, planID, logLine)
		s.wsHub.Broadcast(ws.Message{
			Type: "test_progress",
			Data: map[string]interface{}{
				"testId":   testID,
				"planId":   planID,
				"line":     logLine,
				"flowName": flow.Name,
				"status":   flowStatus,
				"duration": formatDuration(flowDuration),
			},
		})
	}

	s.finishTestRun(planID, testID, planName, startTime, flowResults, nil, createdBy)
}

// broadcastTestLog sends a log line via WebSocket and updates the running test log buffer.
func (s *Server) broadcastTestLog(testID, planID, line string) {
	s.runningTestsMu.Lock()
	if rt, ok := s.runningTests[testID]; ok {
		if len(rt.Logs) >= maxRunningTestLogs {
			rt.Logs = rt.Logs[1:]
		}
		rt.Logs = append(rt.Logs, line)
	}
	s.runningTestsMu.Unlock()

	s.wsHub.Broadcast(ws.Message{
		Type: "test_progress",
		Data: map[string]interface{}{
			"testId": testID,
			"planId": planID,
			"line":   line,
		},
	})
}

// parseFlowDir reads and parses all YAML flow files in a directory, sorted by filename.
func parseFlowDir(dir string) ([]browserFlowFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading flow dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	var flows []browserFlowFile
	for _, fname := range files {
		content, err := os.ReadFile(filepath.Join(dir, fname))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", fname, err)
		}

		flow, err := parseFlowYAMLForBrowser(fname, string(content))
		if err != nil {
			log.Printf("Warning: skipping flow %s: %v", fname, err)
			continue
		}
		flows = append(flows, *flow)
	}

	if len(flows) == 0 {
		return nil, fmt.Errorf("no valid flow files found in %s", dir)
	}
	return flows, nil
}

// parseFlowYAMLForBrowser parses a Maestro YAML flow into metadata and commands for browser execution.
func parseFlowYAMLForBrowser(filename, content string) (*browserFlowFile, error) {
	data := []byte(content)
	parts := bytes.Split(data, []byte("\n---\n"))

	// Handle "---\n" at start
	if bytes.HasPrefix(data, []byte("---\n")) {
		rest := data[4:]
		innerParts := bytes.SplitN(rest, []byte("\n---\n"), 2)
		if len(innerParts) == 2 {
			parts = [][]byte{innerParts[0], innerParts[1]}
		} else {
			parts = [][]byte{innerParts[0]}
		}
	}

	flow := &browserFlowFile{
		Name: strings.TrimSuffix(strings.TrimSuffix(filename, ".yaml"), ".yml"),
		Path: filename,
	}

	if len(parts) == 1 {
		// No separator — try as bare command list
		var rawList []interface{}
		if err := yaml.Unmarshal(data, &rawList); err == nil && len(rawList) > 0 {
			flow.Commands = rawList
			return flow, nil
		}
		return nil, fmt.Errorf("no --- separator and not a command list")
	}

	// Parse metadata
	if len(parts[0]) > 0 {
		var metadata map[string]interface{}
		if err := yaml.Unmarshal(parts[0], &metadata); err == nil {
			if v, ok := metadata["appId"].(string); ok {
				flow.Meta.AppID = v
			}
			if v, ok := metadata["url"].(string); ok {
				flow.Meta.URL = v
			}
			if tags, ok := metadata["tags"].([]interface{}); ok {
				for _, t := range tags {
					if s, ok := t.(string); ok {
						flow.Meta.Tags = append(flow.Meta.Tags, s)
					}
				}
			}
		}
	}

	// Parse commands
	if len(parts) > 1 && len(bytes.TrimSpace(parts[1])) > 0 {
		var rawCmds []interface{}
		if err := yaml.Unmarshal(parts[1], &rawCmds); err != nil {
			return nil, fmt.Errorf("parsing commands: %w", err)
		}
		flow.Commands = rawCmds
	}

	if len(flow.Commands) == 0 {
		return nil, fmt.Errorf("flow has no commands")
	}

	return flow, nil
}

// executeFlowCommand executes a single Maestro flow command using the browser.
func executeFlowCommand(page ai.BrowserPage, toolExec *ai.BrowserToolExecutor, cmd interface{}, aiClient *ai.ClaudeClient, vpWidth, vpHeight int) (result string, screenshot string, err error) {
	switch c := cmd.(type) {
	case string:
		return executeStringCommand(page, toolExec, c)
	case map[string]interface{}:
		return executeMapCommand(page, toolExec, c, aiClient, vpWidth, vpHeight)
	default:
		return "", "", fmt.Errorf("unknown command type: %T", cmd)
	}
}

// executeStringCommand handles simple string commands like "back", "takeScreenshot".
func executeStringCommand(page ai.BrowserPage, toolExec *ai.BrowserToolExecutor, cmd string) (string, string, error) {
	switch cmd {
	case "back":
		_, err := page.EvalJS("history.back()")
		if err != nil {
			return "", "", fmt.Errorf("back: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
		ss, _ := page.CaptureScreenshot()
		return "Navigated back.", ss, nil

	case "takeScreenshot":
		ss, err := page.CaptureScreenshot()
		if err != nil {
			return "", "", fmt.Errorf("screenshot: %w", err)
		}
		return "Screenshot captured.", ss, nil

	case "hideKeyboard":
		return "hideKeyboard (no-op in browser).", "", nil

	default:
		return fmt.Sprintf("Skipped unsupported command: %s", cmd), "", nil
	}
}

// executeMapCommand handles map-style commands like {openLink: "url"}, {tapOn: ...}, etc.
func executeMapCommand(page ai.BrowserPage, toolExec *ai.BrowserToolExecutor, cmd map[string]interface{}, aiClient *ai.ClaudeClient, vpWidth, vpHeight int) (string, string, error) {
	for cmdName, value := range cmd {
		switch cmdName {
		case "openLink":
			url, _ := value.(string)
			if url == "" {
				if m, ok := value.(map[string]interface{}); ok {
					url, _ = m["url"].(string)
				}
			}
			if url == "" {
				return "", "", fmt.Errorf("openLink: missing URL")
			}
			input, _ := json.Marshal(map[string]string{"url": url})
			return toolExec.Execute("navigate", input)

		case "tapOn":
			return executeTapOn(page, toolExec, value, aiClient, vpWidth, vpHeight)

		case "inputText":
			text, _ := value.(string)
			if text == "" {
				return "", "", fmt.Errorf("inputText: empty text")
			}
			input, _ := json.Marshal(map[string]string{"text": text})
			return toolExec.Execute("type_text", input)

		case "scroll":
			return executeScroll(toolExec, value)

		case "extendedWaitUntil":
			return executeWaitUntil(page, value, aiClient, vpWidth, vpHeight)

		case "assertVisible":
			return executeAssertVisible(page, value, aiClient, true)

		case "assertNotVisible":
			return executeAssertVisible(page, value, aiClient, false)

		case "takeScreenshot":
			ss, err := page.CaptureScreenshot()
			if err != nil {
				return "", "", fmt.Errorf("screenshot: %w", err)
			}
			return "Screenshot captured.", ss, nil

		case "back":
			_, err := page.EvalJS("history.back()")
			if err != nil {
				return "", "", fmt.Errorf("back: %w", err)
			}
			time.Sleep(500 * time.Millisecond)
			ss, _ := page.CaptureScreenshot()
			return "Navigated back.", ss, nil

		case "pressKey":
			key, _ := value.(string)
			return fmt.Sprintf("pressKey %q (no-op in browser)", key), "", nil

		case "eraseText":
			// Simulate backspace key presses
			count := 10 // default
			if n, ok := value.(int); ok {
				count = n
			}
			if s, ok := value.(string); ok {
				if n, err := strconv.Atoi(s); err == nil {
					count = n
				}
			}
			for i := 0; i < count; i++ {
				page.EvalJS(`document.execCommand('delete', false)`)
			}
			ss, _ := page.CaptureScreenshot()
			return fmt.Sprintf("Erased %d characters.", count), ss, nil

		case "evalScript":
			expr, _ := value.(string)
			if expr == "" {
				if m, ok := value.(map[string]interface{}); ok {
					expr, _ = m["script"].(string)
				}
			}
			res, err := page.EvalJS(expr)
			if err != nil {
				return "", "", fmt.Errorf("evalScript: %w", err)
			}
			return res, "", nil

		case "launchApp", "clearState", "stopApp":
			return fmt.Sprintf("%s (no-op in browser mode)", cmdName), "", nil

		case "repeat":
			return executeRepeat(page, toolExec, value, aiClient, vpWidth, vpHeight)

		case "runFlow":
			return fmt.Sprintf("runFlow (not supported in browser mode)"), "", nil

		default:
			return fmt.Sprintf("Skipped unsupported command: %s", cmdName), "", nil
		}
	}
	return "", "", fmt.Errorf("empty command map")
}

// executeTapOn handles the tapOn command with point, text, or AI-based targeting.
func executeTapOn(page ai.BrowserPage, toolExec *ai.BrowserToolExecutor, value interface{}, aiClient *ai.ClaudeClient, vpWidth, vpHeight int) (string, string, error) {
	switch v := value.(type) {
	case string:
		// tapOn: "text" — use AI vision to find and click the text
		return tapOnText(page, toolExec, v, aiClient, vpWidth, vpHeight)

	case map[string]interface{}:
		// Check for point-based tap
		if pointStr, ok := v["point"].(string); ok {
			return tapOnPoint(page, toolExec, pointStr, vpWidth, vpHeight)
		}
		// Check for text-based tap
		if text, ok := v["text"].(string); ok {
			return tapOnText(page, toolExec, text, aiClient, vpWidth, vpHeight)
		}
		// Check for id-based tap (use AI or JS)
		if id, ok := v["id"].(string); ok {
			// Try clicking by element ID using JavaScript
			jsResult, err := page.EvalJS(fmt.Sprintf(`(() => {
				const el = document.getElementById('%s');
				if (el) { el.click(); return 'clicked'; }
				return 'not_found';
			})()`, id))
			if err == nil && jsResult == "clicked" {
				time.Sleep(250 * time.Millisecond)
				ss, _ := page.CaptureScreenshot()
				return fmt.Sprintf("Tapped element #%s", id), ss, nil
			}
			// Fall back to AI vision for the ID text
			return tapOnText(page, toolExec, id, aiClient, vpWidth, vpHeight)
		}
		return "", "", fmt.Errorf("tapOn: no recognizable selector (text, point, or id)")

	default:
		return "", "", fmt.Errorf("tapOn: unexpected value type %T", value)
	}
}

// tapOnText uses AI vision to find text on screen and click its coordinates.
func tapOnText(page ai.BrowserPage, toolExec *ai.BrowserToolExecutor, text string, aiClient *ai.ClaudeClient, vpWidth, vpHeight int) (string, string, error) {
	if aiClient == nil {
		return "", "", fmt.Errorf("tapOn text %q: AI client not configured (set ANTHROPIC_API_KEY)", text)
	}

	ss, err := page.CaptureScreenshot()
	if err != nil {
		return "", "", fmt.Errorf("tapOn text: screenshot failed: %w", err)
	}

	prompt := fmt.Sprintf(
		`Look at this screenshot of a web page (%dx%d viewport). Find the element containing the text "%s" and return ONLY the center coordinates as "x,y" (integer pixel values). If the text is not visible, return "NOT_FOUND".`,
		vpWidth, vpHeight, text,
	)

	response, err := aiClient.AnalyzeWithImage(prompt, ss)
	if err != nil {
		return "", "", fmt.Errorf("tapOn text %q: AI vision failed: %w", text, err)
	}

	response = strings.TrimSpace(response)
	if strings.Contains(strings.ToUpper(response), "NOT_FOUND") {
		return "", ss, fmt.Errorf("tapOn text %q: text not found on screen", text)
	}

	// Parse "x,y" from response
	x, y, parseErr := parseCoordinates(response)
	if parseErr != nil {
		return "", ss, fmt.Errorf("tapOn text %q: could not parse coordinates from AI response %q: %w", text, response, parseErr)
	}

	input, _ := json.Marshal(map[string]int{"x": x, "y": y})
	result, clickSS, clickErr := toolExec.Execute("click", input)
	if clickErr != nil {
		return "", ss, fmt.Errorf("tapOn text %q: click failed: %w", text, clickErr)
	}

	return fmt.Sprintf("Tapped on %q at (%d,%d). %s", text, x, y, result), clickSS, nil
}

// tapOnPoint handles tapOn with point: "x,y" or "x%,y%".
func tapOnPoint(page ai.BrowserPage, toolExec *ai.BrowserToolExecutor, pointStr string, vpWidth, vpHeight int) (string, string, error) {
	parts := strings.Split(pointStr, ",")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("tapOn point: invalid format %q, expected 'x,y'", pointStr)
	}

	xStr := strings.TrimSpace(parts[0])
	yStr := strings.TrimSpace(parts[1])

	var x, y int

	// Handle percentage coordinates
	if strings.HasSuffix(xStr, "%") {
		pct, err := strconv.ParseFloat(strings.TrimSuffix(xStr, "%"), 64)
		if err != nil {
			return "", "", fmt.Errorf("tapOn point: invalid x percentage %q", xStr)
		}
		x = int(pct / 100.0 * float64(vpWidth))
	} else {
		val, err := strconv.Atoi(xStr)
		if err != nil {
			return "", "", fmt.Errorf("tapOn point: invalid x coordinate %q", xStr)
		}
		x = val
	}

	if strings.HasSuffix(yStr, "%") {
		pct, err := strconv.ParseFloat(strings.TrimSuffix(yStr, "%"), 64)
		if err != nil {
			return "", "", fmt.Errorf("tapOn point: invalid y percentage %q", yStr)
		}
		y = int(pct / 100.0 * float64(vpHeight))
	} else {
		val, err := strconv.Atoi(yStr)
		if err != nil {
			return "", "", fmt.Errorf("tapOn point: invalid y coordinate %q", yStr)
		}
		y = val
	}

	input, _ := json.Marshal(map[string]int{"x": x, "y": y})
	return toolExec.Execute("click", input)
}

// executeScroll handles the scroll command.
func executeScroll(toolExec *ai.BrowserToolExecutor, value interface{}) (string, string, error) {
	direction := "down"
	amount := 300

	switch v := value.(type) {
	case string:
		direction = strings.ToLower(v)
	case map[string]interface{}:
		if d, ok := v["direction"].(string); ok {
			direction = strings.ToLower(d)
		}
		if a, ok := v["amount"].(int); ok {
			amount = a
		}
		if a, ok := v["amount"].(float64); ok {
			amount = int(a)
		}
	}

	input, _ := json.Marshal(map[string]interface{}{"direction": direction, "amount": amount})
	return toolExec.Execute("scroll", input)
}

// executeWaitUntil handles extendedWaitUntil by polling screenshots with AI vision.
func executeWaitUntil(page ai.BrowserPage, value interface{}, aiClient *ai.ClaudeClient, vpWidth, vpHeight int) (string, string, error) {
	m, ok := value.(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("extendedWaitUntil: expected map, got %T", value)
	}

	visibleText, _ := m["visible"].(string)
	notVisibleText, _ := m["notVisible"].(string)
	timeoutMs := 10000 // default 10s

	if t, ok := m["timeout"].(int); ok {
		timeoutMs = t
	}
	if t, ok := m["timeout"].(float64); ok {
		timeoutMs = int(t)
	}

	if visibleText == "" && notVisibleText == "" {
		return "", "", fmt.Errorf("extendedWaitUntil: needs visible or notVisible condition")
	}

	checkText := visibleText
	wantVisible := true
	if checkText == "" {
		checkText = notVisibleText
		wantVisible = false
	}

	if aiClient == nil {
		// Without AI, just wait and take a screenshot
		time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
		ss, _ := page.CaptureScreenshot()
		return fmt.Sprintf("Waited %dms (no AI for vision check)", timeoutMs), ss, nil
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	pollInterval := 1 * time.Second
	var lastSS string

	for time.Now().Before(deadline) {
		ss, err := page.CaptureScreenshot()
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}
		lastSS = ss

		prompt := fmt.Sprintf(
			`Look at this screenshot of a web page (%dx%d viewport). Is the text "%s" visible anywhere on the screen? Answer only "YES" or "NO".`,
			vpWidth, vpHeight, checkText,
		)

		response, err := aiClient.AnalyzeWithImage(prompt, ss)
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		isVisible := strings.Contains(strings.ToUpper(strings.TrimSpace(response)), "YES")

		if wantVisible && isVisible {
			return fmt.Sprintf("Text %q is now visible.", checkText), ss, nil
		}
		if !wantVisible && !isVisible {
			return fmt.Sprintf("Text %q is no longer visible.", checkText), ss, nil
		}

		time.Sleep(pollInterval)
	}

	condition := "visible"
	if !wantVisible {
		condition = "not visible"
	}
	return "", lastSS, fmt.Errorf("extendedWaitUntil: timed out waiting for %q to be %s after %dms", checkText, condition, timeoutMs)
}

// executeAssertVisible checks if text is visible (or not visible) using AI vision.
func executeAssertVisible(page ai.BrowserPage, value interface{}, aiClient *ai.ClaudeClient, wantVisible bool) (string, string, error) {
	text := ""
	switch v := value.(type) {
	case string:
		text = v
	case map[string]interface{}:
		if t, ok := v["text"].(string); ok {
			text = t
		}
	}

	if text == "" {
		return "", "", fmt.Errorf("assert: empty text")
	}

	ss, err := page.CaptureScreenshot()
	if err != nil {
		return "", "", fmt.Errorf("assert: screenshot failed: %w", err)
	}

	if aiClient == nil {
		// Without AI, just report the screenshot was taken
		cmdName := "assertVisible"
		if !wantVisible {
			cmdName = "assertNotVisible"
		}
		return fmt.Sprintf("%s %q (no AI to verify)", cmdName, text), ss, nil
	}

	prompt := fmt.Sprintf(
		`Look at this screenshot of a web page. Is the text "%s" visible anywhere on the screen? Answer only "YES" or "NO".`,
		text,
	)

	response, err := aiClient.AnalyzeWithImage(prompt, ss)
	if err != nil {
		return "", ss, fmt.Errorf("assert: AI vision failed: %w", err)
	}

	isVisible := strings.Contains(strings.ToUpper(strings.TrimSpace(response)), "YES")

	if wantVisible && !isVisible {
		return "", ss, fmt.Errorf("assertVisible failed: %q not found on screen", text)
	}
	if !wantVisible && isVisible {
		return "", ss, fmt.Errorf("assertNotVisible failed: %q is visible on screen", text)
	}

	if wantVisible {
		return fmt.Sprintf("assertVisible passed: %q is visible.", text), ss, nil
	}
	return fmt.Sprintf("assertNotVisible passed: %q is not visible.", text), ss, nil
}

// executeRepeat handles the repeat command with a fixed count or while-condition.
func executeRepeat(page ai.BrowserPage, toolExec *ai.BrowserToolExecutor, value interface{}, aiClient *ai.ClaudeClient, vpWidth, vpHeight int) (string, string, error) {
	m, ok := value.(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("repeat: expected map, got %T", value)
	}

	times := 1
	if t, ok := m["times"].(int); ok {
		times = t
	}
	if t, ok := m["times"].(float64); ok {
		times = int(t)
	}

	cmds, ok := m["commands"].([]interface{})
	if !ok || len(cmds) == 0 {
		return "", "", fmt.Errorf("repeat: no commands")
	}

	var lastResult string
	var lastSS string
	for i := 0; i < times; i++ {
		for _, subCmd := range cmds {
			r, ss, err := executeFlowCommand(page, toolExec, subCmd, aiClient, vpWidth, vpHeight)
			if err != nil {
				return r, ss, fmt.Errorf("repeat iteration %d: %w", i+1, err)
			}
			lastResult = r
			if ss != "" {
				lastSS = ss
			}
		}
	}

	return fmt.Sprintf("Repeated %d times. Last: %s", times, lastResult), lastSS, nil
}

// parseCoordinates parses "x,y" from a string, handling common AI response formats.
func parseCoordinates(s string) (int, int, error) {
	s = strings.TrimSpace(s)
	// Handle formats like "123,456" or "(123, 456)" or "x=123, y=456"
	s = strings.Trim(s, "()")
	s = strings.ReplaceAll(s, " ", "")

	// Try direct "x,y" format
	parts := strings.Split(s, ",")
	if len(parts) == 2 {
		x, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		y, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 == nil && err2 == nil {
			return x, y, nil
		}
	}

	// Try to find two numbers in the string
	var nums []int
	for _, word := range strings.FieldsFunc(s, func(r rune) bool {
		return !((r >= '0' && r <= '9') || r == '-')
	}) {
		if n, err := strconv.Atoi(word); err == nil {
			nums = append(nums, n)
		}
	}
	if len(nums) >= 2 {
		return nums[0], nums[1], nil
	}

	return 0, 0, fmt.Errorf("no coordinates found in %q", s)
}

// describeCommand returns a human-readable description of a flow command.
func describeCommand(cmd interface{}) string {
	switch c := cmd.(type) {
	case string:
		return c
	case map[string]interface{}:
		for name, value := range c {
			switch v := value.(type) {
			case string:
				if len(v) > 50 {
					v = v[:50] + "..."
				}
				return fmt.Sprintf("%s: %q", name, v)
			case map[string]interface{}:
				if text, ok := v["text"].(string); ok {
					return fmt.Sprintf("%s: {text: %q}", name, text)
				}
				if point, ok := v["point"].(string); ok {
					return fmt.Sprintf("%s: {point: %s}", name, point)
				}
				if vis, ok := v["visible"].(string); ok {
					return fmt.Sprintf("%s: {visible: %q}", name, vis)
				}
				return fmt.Sprintf("%s: {map}", name)
			default:
				return fmt.Sprintf("%s: %v", name, value)
			}
		}
	}
	return fmt.Sprintf("%v", cmd)
}
