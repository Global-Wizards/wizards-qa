package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/web/backend/store"
	"github.com/Global-Wizards/wizards-qa/web/backend/ws"
)

// executeTestRun runs the wizards-qa CLI as a subprocess and streams progress via WebSocket.
func (s *Server) executeTestRun(planID, testID string, flowDir string, planName string) {
	startTime := time.Now()

	// Update plan status to running
	if planID != "" {
		_ = s.store.UpdateTestPlanStatus(planID, "running", testID)
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "test_started",
		Data: map[string]string{
			"testId": testID,
			"planId": planID,
			"name":   planName,
		},
	})

	cliPath := envOrDefault("WIZARDS_QA_CLI_PATH", "wizards-qa")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	args := []string{"run", "--flows", flowDir}
	if planName != "" {
		args = append(args, "--name", planName)
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.finishTestRun(planID, testID, planName, startTime, nil, fmt.Errorf("stdout pipe: %w", err))
		return
	}

	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	if err := cmd.Start(); err != nil {
		s.finishTestRun(planID, testID, planName, startTime, nil, fmt.Errorf("start: %w", err))
		return
	}

	var flowResults []store.FlowResult
	var allLines []string

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		allLines = append(allLines, line)

		// Parse progress from output lines
		flowName, status := parseFlowLine(line)
		if flowName != "" {
			flowResults = append(flowResults, store.FlowResult{
				Name:   flowName,
				Status: status,
			})
		}

		s.wsHub.Broadcast(ws.Message{
			Type: "test_progress",
			Data: map[string]interface{}{
				"testId":   testID,
				"planId":   planID,
				"line":     line,
				"flowName": flowName,
				"status":   status,
			},
		})
	}

	err = cmd.Wait()
	s.finishTestRun(planID, testID, planName, startTime, flowResults, err)
}

// finishTestRun saves the result and broadcasts completion.
func (s *Server) finishTestRun(planID, testID, planName string, startTime time.Time, flows []store.FlowResult, runErr error) {
	duration := time.Since(startTime)
	status := "passed"
	errorOutput := ""

	if runErr != nil {
		status = "failed"
		errorOutput = runErr.Error()
	}

	passed := 0
	for _, f := range flows {
		if f.Status == "passed" {
			passed++
		}
		if f.Duration == "" {
			f.Duration = "0s"
		}
	}

	successRate := 0.0
	if len(flows) > 0 {
		successRate = float64(passed) / float64(len(flows)) * 100
	}

	// If no flows parsed but command succeeded, set 100%
	if len(flows) == 0 && runErr == nil {
		successRate = 100
	}

	result := store.TestResultDetail{
		ID:          testID,
		Name:        planName,
		Status:      status,
		Timestamp:   startTime.Format(time.RFC3339),
		Duration:    formatDuration(duration),
		SuccessRate: successRate,
		Flows:       flows,
		ErrorOutput: errorOutput,
	}

	if err := s.store.SaveTestResult(result); err != nil {
		log.Printf("Error saving test result: %v", err)
	}

	if planID != "" {
		planStatus := "completed"
		if runErr != nil {
			planStatus = "failed"
		}
		_ = s.store.UpdateTestPlanStatus(planID, planStatus, testID)
	}

	msgType := "test_completed"
	if runErr != nil {
		msgType = "test_failed"
	}

	s.wsHub.Broadcast(ws.Message{
		Type: msgType,
		Data: map[string]interface{}{
			"testId":      testID,
			"planId":      planID,
			"status":      status,
			"duration":    formatDuration(duration),
			"successRate": successRate,
			"flowCount":   len(flows),
		},
	})
}

// prepareFlowDir copies selected templates to a temp dir with variable substitution.
func (s *Server) prepareFlowDir(plan *store.TestPlan) (string, error) {
	tmpDir, err := os.MkdirTemp("", "wizards-qa-run-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	templates, err := s.store.ListTemplates()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("listing templates: %w", err)
	}

	// Build lookup of selected flow names
	selected := make(map[string]bool)
	for _, name := range plan.FlowNames {
		selected[name] = true
	}

	flowsBase := filepath.Dir(s.store.FlowsDir())

	for _, tmpl := range templates {
		if !selected[tmpl.Name] {
			continue
		}

		srcPath := filepath.Join(flowsBase, tmpl.Path)
		content, err := os.ReadFile(srcPath)
		if err != nil {
			log.Printf("Warning: could not read template %s: %v", tmpl.Name, err)
			continue
		}

		// Perform variable substitution
		result := string(content)
		for varName, varValue := range plan.Variables {
			result = strings.ReplaceAll(result, "{{"+varName+"}}", varValue)
		}

		dstPath := filepath.Join(tmpDir, filepath.Base(tmpl.Path))
		if err := os.WriteFile(dstPath, []byte(result), 0644); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("writing flow %s: %w", tmpl.Name, err)
		}
	}

	return tmpDir, nil
}

// parseFlowLine extracts flow name and pass/fail status from CLI output lines.
func parseFlowLine(line string) (string, string) {
	trimmed := strings.TrimSpace(line)

	if strings.Contains(trimmed, "✅") || strings.Contains(trimmed, "PASS") {
		name := extractFlowName(trimmed)
		return name, "passed"
	}
	if strings.Contains(trimmed, "❌") || strings.Contains(trimmed, "FAIL") {
		name := extractFlowName(trimmed)
		return name, "failed"
	}

	return "", ""
}

func extractFlowName(line string) string {
	// Remove common prefixes/emoji
	line = strings.TrimSpace(line)
	for _, prefix := range []string{"✅", "❌", "PASS", "FAIL", ":", "-", " "} {
		line = strings.TrimPrefix(line, prefix)
	}
	line = strings.TrimSpace(line)

	// Take first word or up to common delimiters
	if idx := strings.IndexAny(line, "(:"); idx > 0 {
		line = line[:idx]
	}

	return strings.TrimSpace(line)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
