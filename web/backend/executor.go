package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/web/backend/store"
	"github.com/Global-Wizards/wizards-qa/web/backend/ws"
	"gopkg.in/yaml.v3"
)

var safeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\s.]+$`)

// executeTestRun runs the wizards-qa CLI as a subprocess and streams progress via WebSocket.
// Must be called in a goroutine with panic recovery (see launchTestRun).
func (s *Server) executeTestRun(planID, testID string, flowDir string, planName string, createdBy string) {
	startTime := time.Now()

	if planID != "" {
		if err := s.store.UpdateTestPlanStatus(planID, store.StatusRunning, testID); err != nil {
			log.Printf("Warning: failed to update plan %s status to running: %v", planID, err)
		}
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

	ctx, cancel := context.WithTimeout(s.serverCtx, 10*time.Minute)
	defer cancel()

	args := []string{"run", "--flows", flowDir}
	if planName != "" && safeNameRegex.MatchString(planName) {
		args = append(args, "--name", planName)
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.finishTestRun(planID, testID, planName, startTime, nil, fmt.Errorf("stdout pipe: %w", err), createdBy)
		return
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		s.finishTestRun(planID, testID, planName, startTime, nil, fmt.Errorf("start: %w", err), createdBy)
		return
	}

	var flowResults []store.FlowResult

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

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

	if scanErr := scanner.Err(); scanErr != nil {
		log.Printf("Warning: scanner error reading test output for %s: %v", testID, scanErr)
	}

	err = cmd.Wait()
	if err != nil && stderrBuf.Len() > 0 {
		err = fmt.Errorf("%w\nstderr: %s", err, stderrBuf.String())
	}
	s.finishTestRun(planID, testID, planName, startTime, flowResults, err, createdBy)
}

// launchTestRun starts executeTestRun in a goroutine with panic recovery.
func (s *Server) launchTestRun(planID, testID, flowDir, planName string, cleanupDir bool, createdBy ...string) {
	userID := ""
	if len(createdBy) > 0 {
		userID = createdBy[0]
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in test execution %s: %v", testID, r)
				s.finishTestRun(planID, testID, planName, time.Now(), nil, fmt.Errorf("panic: %v", r), userID)
			}
			if cleanupDir {
				if err := os.RemoveAll(flowDir); err != nil && !os.IsNotExist(err) {
					log.Printf("Warning: failed to clean up temp dir %s: %v", flowDir, err)
				}
			}
		}()
		s.executeTestRun(planID, testID, flowDir, planName, userID)
	}()
}

// finishTestRun saves the result and broadcasts completion.
func (s *Server) finishTestRun(planID, testID, planName string, startTime time.Time, flows []store.FlowResult, runErr error, createdBy string) {
	duration := time.Since(startTime)
	status := store.StatusPassed
	errorOutput := ""

	if runErr != nil {
		status = store.StatusFailed
		errorOutput = runErr.Error()
	}

	passed := 0
	for i, f := range flows {
		if f.Status == store.StatusPassed {
			passed++
		}
		if f.Duration == "" {
			flows[i].Duration = "0s"
		}
	}

	successRate := 0.0
	if len(flows) > 0 {
		successRate = float64(passed) / float64(len(flows)) * 100
	}

	if len(flows) == 0 && runErr == nil {
		successRate = 100
	}

	// Look up project_id from the test plan
	var projectID string
	if planID != "" {
		if plan, err := s.store.GetTestPlan(planID); err == nil {
			projectID = plan.ProjectID
		}
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
		CreatedBy:   createdBy,
		ProjectID:   projectID,
	}

	if err := s.store.SaveTestResult(result); err != nil {
		log.Printf("Error saving test result %s: %v", testID, err)
	}

	if planID != "" {
		planStatus := store.StatusCompleted
		if runErr != nil {
			planStatus = store.StatusFailed
		}
		if err := s.store.UpdateTestPlanStatus(planID, planStatus, testID); err != nil {
			log.Printf("Warning: failed to update plan %s status to %s: %v", planID, planStatus, err)
		}
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

	// Fast path: analysis-linked plans copy directly from generated/{analysisID}/
	if plan.AnalysisID != "" {
		srcDir := filepath.Join(s.store.FlowsDir(), "generated", plan.AnalysisID)
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			// Directory missing (e.g. after container redeploy) — regenerate from DB
			log.Printf("Generated flows dir missing for %s, regenerating from analysis result", plan.AnalysisID)
			if rErr := s.regenerateFlowsFromAnalysis(plan.AnalysisID); rErr != nil {
				os.RemoveAll(tmpDir)
				return "", fmt.Errorf("generated flows missing and regeneration failed: %w (original: %v)", rErr, err)
			}
			entries, err = os.ReadDir(srcDir)
			if err != nil {
				os.RemoveAll(tmpDir)
				return "", fmt.Errorf("reading regenerated flows for analysis %s: %w", plan.AnalysisID, err)
			}
		}
		copied := 0
		for _, e := range entries {
			if e.IsDir() || !(strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
				continue
			}
			content, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
			if err != nil {
				log.Printf("Warning: could not read generated flow %s: %v", e.Name(), err)
				continue
			}
			result := normalizeFlowYAML(varSubstitute(string(content), plan.Variables))
			if err := os.WriteFile(filepath.Join(tmpDir, e.Name()), []byte(result), 0644); err != nil {
				os.RemoveAll(tmpDir)
				return "", fmt.Errorf("writing flow %s: %w", e.Name(), err)
			}
			copied++
		}
		if copied == 0 {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("no flow files found in generated/%s", plan.AnalysisID)
		}
		return tmpDir, nil
	}

	templates, err := s.store.ListTemplates()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("listing templates: %w", err)
	}

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

		// Variable substitution + normalize openBrowser syntax
		result := normalizeFlowYAML(varSubstitute(string(content), plan.Variables))

		dstPath := filepath.Join(tmpDir, filepath.Base(tmpl.Path))
		if err := os.WriteFile(dstPath, []byte(result), 0644); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("writing flow %s: %w", tmpl.Name, err)
		}
	}

	return tmpDir, nil
}

// normalizeFlowYAML fixes openBrowser object syntax to simple string syntax.
// Converts:
//
//	- openBrowser:
//	    url: "https://..."
//
// To:
//
//	- openBrowser: "https://..."
var openBrowserObjRegex = regexp.MustCompile(`(?m)^(\s*- openBrowser):\s*\n\s+url:\s*"?([^"\n]+)"?\s*$`)

func normalizeFlowYAML(content string) string {
	return openBrowserObjRegex.ReplaceAllString(content, `$1: "$2"`)
}

// regenerateFlowsFromAnalysis reconstructs YAML flow files from the analysis
// result stored in the database. This handles the case where the generated/
// directory was lost (e.g. after a container redeploy with ephemeral storage).
func (s *Server) regenerateFlowsFromAnalysis(analysisID string) error {
	analysis, err := s.store.GetAnalysis(analysisID)
	if err != nil {
		return fmt.Errorf("getting analysis: %w", err)
	}

	resultMap, ok := analysis.Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("analysis has no structured result")
	}

	flowsRaw, ok := resultMap["flows"].([]interface{})
	if !ok || len(flowsRaw) == 0 {
		return fmt.Errorf("no flows in analysis result")
	}

	dstDir := filepath.Join(s.store.FlowsDir(), "generated", analysisID)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("creating generated dir: %w", err)
	}

	// Sort: setup flow first, matching the CLI's WriteFlowsToFiles behavior
	sort.SliceStable(flowsRaw, func(i, j int) bool {
		iMap, _ := flowsRaw[i].(map[string]interface{})
		jMap, _ := flowsRaw[j].(map[string]interface{})
		iName, _ := iMap["name"].(string)
		jName, _ := jMap["name"].(string)
		return strings.EqualFold(iName, "setup") && !strings.EqualFold(jName, "setup")
	})

	for i, flowRaw := range flowsRaw {
		flowMap, ok := flowRaw.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := flowMap["name"].(string)
		if name == "" {
			name = fmt.Sprintf("flow-%d", i)
		}

		var sb strings.Builder

		// Metadata section: tags
		if tags, ok := flowMap["tags"].([]interface{}); ok && len(tags) > 0 {
			sb.WriteString("tags:\n")
			for _, tag := range tags {
				sb.WriteString(fmt.Sprintf("  - %v\n", tag))
			}
			sb.WriteString("---\n")
		}

		// Commands — serialize with yaml.v3 then normalize
		if commands, ok := flowMap["commands"].([]interface{}); ok {
			yamlBytes, err := yaml.Marshal(commands)
			if err == nil {
				sb.Write(yamlBytes)
			}
		}

		content := normalizeFlowYAML(sb.String())
		filename := fmt.Sprintf("%02d-%s.yaml", i, sanitizeFlowName(name))
		if err := os.WriteFile(filepath.Join(dstDir, filename), []byte(content), 0644); err != nil {
			return fmt.Errorf("writing regenerated flow %s: %w", filename, err)
		}
	}

	log.Printf("Regenerated %d flow files for analysis %s in %s", len(flowsRaw), analysisID, dstDir)
	return nil
}

// sanitizeFlowName replaces unsafe characters in flow names for use as filenames.
func sanitizeFlowName(name string) string {
	safe := strings.ToLower(name)
	for _, ch := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "} {
		safe = strings.ReplaceAll(safe, ch, "-")
	}
	return safe
}

// varSubstitute replaces {{VAR}} patterns in a single pass.
var varRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)

func varSubstitute(content string, vars map[string]string) string {
	return varRegex.ReplaceAllStringFunc(content, func(match string) string {
		key := match[2 : len(match)-2]
		if val, ok := vars[key]; ok {
			return val
		}
		return match
	})
}

// parseFlowLine extracts flow name and pass/fail status from CLI output lines.
func parseFlowLine(line string) (string, string) {
	trimmed := strings.TrimSpace(line)

	if strings.Contains(trimmed, "✅") || strings.Contains(trimmed, "PASS") {
		return extractFlowName(trimmed), "passed"
	}
	if strings.Contains(trimmed, "❌") || strings.Contains(trimmed, "FAIL") {
		return extractFlowName(trimmed), "failed"
	}

	return "", ""
}

func extractFlowName(line string) string {
	line = strings.TrimSpace(line)
	for _, prefix := range []string{"✅", "❌", "PASS", "FAIL", ":", "-", " "} {
		line = strings.TrimPrefix(line, prefix)
	}
	line = strings.TrimSpace(line)

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
