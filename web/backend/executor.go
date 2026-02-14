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

// runningTest tracks the live state of a running test execution for reconnection support.
type runningTest struct {
	TestID     string             `json:"testId"`
	PlanID     string             `json:"planId"`
	PlanName   string             `json:"planName"`
	Mode       string             `json:"mode"`
	StartedAt  time.Time          `json:"startedAt"`
	TotalFlows int                `json:"totalFlows"`
	Flows      []store.FlowResult `json:"flows"`
	Logs       []string           `json:"logs"`
	Status     string             `json:"status"`
}

const maxRunningTestLogs = 500

var safeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\s.]+$`)

// executeTestRun runs the wizards-qa CLI as a subprocess and streams progress via WebSocket.
// Must be called in a goroutine with panic recovery (see launchTestRun).
func (s *Server) executeTestRun(planID, testID string, flowDir string, planName string, createdBy string) {
	startTime := time.Now()
	totalFlows := countFlowFiles(flowDir)

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
		Mode:       ModeMaestro,
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
			"mode":       ModeMaestro,
		},
	})

	cliPath := envOrDefault("WIZARDS_QA_CLI_PATH", "wizards-qa")

	ctx, cancel := context.WithTimeout(s.serverCtx, TestExecutionTimeout)
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

		flowName, status, duration := parseFlowLine(line)
		if flowName != "" {
			fr := store.FlowResult{
				Name:     flowName,
				Status:   status,
				Duration: duration,
			}
			flowResults = append(flowResults, fr)

			// Update running test state
			s.runningTests.AppendFlow(testID, fr)
		}

		// Update running test logs
		s.runningTests.AppendLog(testID, line)

		s.wsHub.Broadcast(ws.Message{
			Type: "test_progress",
			Data: map[string]interface{}{
				"testId":   testID,
				"planId":   planID,
				"line":     line,
				"flowName": flowName,
				"status":   status,
				"duration": duration,
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
	// Remove from running tests
	s.runningTests.Remove(testID)

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
		PlanID:      planID,
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
			result := injectAppId(normalizeFlowYAML(varSubstitute(string(content), plan.Variables)))
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

		// Variable substitution + normalize openLink syntax + inject appId for web flows
		result := injectAppId(normalizeFlowYAML(varSubstitute(string(content), plan.Variables)))

		dstPath := filepath.Join(tmpDir, filepath.Base(tmpl.Path))
		if err := os.WriteFile(dstPath, []byte(result), 0644); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("writing flow %s: %w", tmpl.Name, err)
		}
	}

	return tmpDir, nil
}

// normalizeFlowYAML fixes openLink object syntax to simple string syntax.
// Converts:
//
//	- openLink:
//	    url: "https://..."
//
// To:
//
//	- openLink: "https://..."
var openLinkObjRegex = regexp.MustCompile(`(?m)^(\s*- openLink):\s*\n\s+url:\s*"?([^"\n]+)"?\s*$`)

// extendedWaitTimeoutOnlyRegex matches two-line extendedWaitUntil blocks that have
// only a timeout and no visible/notVisible condition (invalid Maestro YAML).
var extendedWaitTimeoutOnlyRegex = regexp.MustCompile(`(?m)^[ \t]*- extendedWaitUntil:\s*\n[ \t]+timeout:\s*\d+\s*$`)

// selectorVisibleRegex matches any command followed by visible: on the next indented line.
// The replacement function skips extendedWaitUntil (which legitimately uses visible:).
// Handles both "- cmd:\n  visible:" and "- cmd: value\n  visible:" patterns.
var selectorVisibleRegex = regexp.MustCompile(`(?m)^(\s*- \w+):.*\n\s+visible:\s*"?([^"\n]+)"?\s*$`)

// selectorNotVisibleRegex — same but for notVisible:
var selectorNotVisibleRegex = regexp.MustCompile(`(?m)^(\s*- \w+):.*\n\s+notVisible:\s*"?([^"\n]+)"?\s*$`)

// maestroCommandAliases maps invalid/old command names to correct Maestro names.
var maestroCommandAliases = map[string]string{
	"waitFor":     "extendedWaitUntil",
	"screenshot":  "takeScreenshot",
	"openBrowser": "openLink",
}

// stripInvalidVisibleLines removes blank lines from the commands section and
// strips visible:/notVisible: lines from non-extendedWaitUntil command blocks.
// This catches AI patterns where blank lines between mapping fields break YAML
// parsing and where visible: is sprinkled after arbitrary commands.
func stripInvalidVisibleLines(content string) string {
	// Split metadata from commands on the --- separator
	parts := strings.SplitN(content, "---\n", 2)
	var meta, cmds string
	if len(parts) == 2 {
		meta = parts[0] + "---\n"
		cmds = parts[1]
	} else {
		cmds = content
	}

	lines := strings.Split(cmds, "\n")
	var out []string
	inExtendedWait := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Remove blank lines in the commands section
		if trimmed == "" {
			continue
		}
		// Track whether we're inside an extendedWaitUntil block
		if strings.Contains(line, "- extendedWaitUntil") {
			inExtendedWait = true
			out = append(out, line)
			continue
		}
		// A new command starts (indented "- ") — exit extendedWaitUntil tracking
		if strings.Contains(line, "- ") && !strings.HasPrefix(trimmed, "visible:") && !strings.HasPrefix(trimmed, "notVisible:") && !strings.HasPrefix(trimmed, "timeout:") && !strings.HasPrefix(trimmed, "point:") {
			inExtendedWait = false
		}
		// Strip visible:/notVisible: lines that are NOT inside extendedWaitUntil
		if !inExtendedWait && (strings.HasPrefix(trimmed, "visible:") || strings.HasPrefix(trimmed, "notVisible:")) {
			continue
		}
		out = append(out, line)
	}

	return meta + strings.Join(out, "\n")
}

func normalizeFlowYAML(content string) string {
	// First pass: strip blank lines and invalid visible/notVisible lines
	result := stripInvalidVisibleLines(content)
	result = openLinkObjRegex.ReplaceAllString(result, `$1: "$2"`)
	// Flatten {visible: "..."} / {notVisible: "..."} from any command except extendedWaitUntil
	skipExtended := func(match string) string {
		if strings.Contains(match, "extendedWaitUntil") {
			return match
		}
		return selectorVisibleRegex.ReplaceAllString(match, `$1: "$2"`)
	}
	result = selectorVisibleRegex.ReplaceAllStringFunc(result, skipExtended)
	skipExtendedNV := func(match string) string {
		if strings.Contains(match, "extendedWaitUntil") {
			return match
		}
		return selectorNotVisibleRegex.ReplaceAllString(match, `$1: "$2"`)
	}
	result = selectorNotVisibleRegex.ReplaceAllStringFunc(result, skipExtendedNV)
	// Fix invalid command names that the AI may have generated
	for old, correct := range maestroCommandAliases {
		result = strings.ReplaceAll(result, "- "+old+":", "- "+correct+":")
	}
	// Strip timeout-only extendedWaitUntil blocks (invalid Maestro YAML)
	result = extendedWaitTimeoutOnlyRegex.ReplaceAllString(result, "")
	return result
}

// injectAppId ensures web flows have appId in their metadata section.
// Maestro requires appId in every flow's YAML; for web testing via openLink,
// com.android.chrome satisfies the parser.
func injectAppId(content string) string {
	// Already has appId — leave unchanged
	if strings.Contains(content, "appId:") {
		return content
	}
	// Not a web flow — leave unchanged
	if !strings.Contains(content, "openLink:") && !strings.Contains(content, "runFlow:") {
		return content
	}
	// If content has no metadata separator, inject appId with separator
	if !strings.Contains(content, "\n---\n") && !strings.HasPrefix(content, "---\n") {
		return "appId: com.android.chrome\n---\n" + content
	}
	return "appId: com.android.chrome\n" + content
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

		// Metadata section: appId, url, tags
		hasMetadata := false
		if appId, ok := flowMap["appId"].(string); ok && appId != "" {
			sb.WriteString(fmt.Sprintf("appId: %s\n", appId))
			hasMetadata = true
		}
		if urlVal, ok := flowMap["url"].(string); ok && urlVal != "" {
			sb.WriteString(fmt.Sprintf("url: %s\n", urlVal))
			hasMetadata = true
		}
		if tags, ok := flowMap["tags"].([]interface{}); ok && len(tags) > 0 {
			sb.WriteString("tags:\n")
			for _, tag := range tags {
				sb.WriteString(fmt.Sprintf("  - %v\n", tag))
			}
			hasMetadata = true
		}
		if hasMetadata {
			sb.WriteString("---\n")
		}

		// Commands — serialize using yaml.Marshal for correct escaping
		if commands, ok := flowMap["commands"].([]interface{}); ok {
			for _, cmd := range commands {
				if cmdMap, ok := cmd.(map[string]interface{}); ok {
					// Extract comment before fixing
					if comment, ok := cmdMap["comment"].(string); ok && comment != "" {
						sb.WriteString(fmt.Sprintf("# %s\n", strings.ReplaceAll(comment, "\n", " ")))
					}
					// Split out spurious visible/notVisible into separate extendedWaitUntil commands
					for _, splitCmd := range splitVisibleFromCommand(cmdMap) {
						if fixed := fixCommandData(splitCmd, maestroCommandAliases); fixed != nil {
							var toMarshal interface{} = fixed
							// Single key with empty string → plain command (e.g. "- takeScreenshot")
							if len(fixed) == 1 {
								for k, v := range fixed {
									if s, ok := v.(string); ok && s == "" {
										toMarshal = k
									}
								}
							}
							cmdYAML, err := yaml.Marshal([]interface{}{toMarshal})
							if err == nil {
								sb.Write(cmdYAML)
							}
						}
					}
				}
			}
		}

		content := injectAppId(normalizeFlowYAML(sb.String()))
		filename := fmt.Sprintf("%02d-%s.yaml", i, sanitizeFlowName(name))
		if err := os.WriteFile(filepath.Join(dstDir, filename), []byte(content), 0644); err != nil {
			return fmt.Errorf("writing regenerated flow %s: %w", filename, err)
		}
	}

	log.Printf("Regenerated %d flow files for analysis %s in %s", len(flowsRaw), analysisID, dstDir)
	return nil
}

// fixCommandData fixes AI mistakes at the data level before yaml.Marshal.
// It translates command aliases, flattens invalid nested structures, strips
// newlines from string values, and removes invalid extendedWaitUntil blocks.
func fixCommandData(cmd map[string]interface{}, aliases map[string]string) map[string]interface{} {
	fixed := make(map[string]interface{})
	for key, value := range cmd {
		if key == "comment" {
			continue // handled separately before marshaling
		}
		if corrected, ok := aliases[key]; ok {
			key = corrected
		}
		switch v := value.(type) {
		case string:
			fixed[key] = strings.ReplaceAll(v, "\n", " ")
		case map[string]interface{}:
			// Flatten openLink: {url: "..."} → openLink: "..."
			if key == "openLink" {
				if urlVal, ok := v["url"]; ok {
					fixed[key] = strings.ReplaceAll(fmt.Sprintf("%v", urlVal), "\n", " ")
					continue
				}
			}
			// Strip visible/notVisible from non-extendedWaitUntil commands
			if key != "extendedWaitUntil" {
				_, hasVis := v["visible"]
				_, hasNV := v["notVisible"]
				if hasVis || hasNV {
					if hasVis && len(v) == 1 {
						// visible is the only key — flatten to string
						fixed[key] = strings.ReplaceAll(fmt.Sprintf("%v", v["visible"]), "\n", " ")
						continue
					}
					if hasNV && len(v) == 1 {
						fixed[key] = strings.ReplaceAll(fmt.Sprintf("%v", v["notVisible"]), "\n", " ")
						continue
					}
					// Has other keys too (e.g. point) — strip visible/notVisible, keep the rest
					delete(v, "visible")
					delete(v, "notVisible")
				}
			}
			// Skip extendedWaitUntil with only timeout (no visible/notVisible)
			if key == "extendedWaitUntil" {
				_, hasVisible := v["visible"]
				_, hasNotVisible := v["notVisible"]
				if !hasVisible && !hasNotVisible {
					continue
				}
			}
			// Recursively clean sub-map values
			cleanedSub := make(map[string]interface{})
			for sk, sv := range v {
				switch subV := sv.(type) {
				case string:
					cleanedSub[sk] = strings.ReplaceAll(subV, "\n", " ")
				case []interface{}:
					cleanedSub[sk] = fixCommandList(subV, aliases)
				default:
					cleanedSub[sk] = sv
				}
			}
			fixed[key] = cleanedSub
		case []interface{}:
			fixed[key] = fixCommandList(v, aliases)
		default:
			fixed[key] = value
		}
	}
	if len(fixed) == 0 {
		return nil
	}
	return fixed
}

// splitVisibleFromCommand extracts top-level visible/notVisible from a non-extendedWaitUntil
// command and returns the original command (cleaned) plus a separate extendedWaitUntil command.
// e.g. {openLink: "url", visible: "text"} → [{openLink: "url"}, {extendedWaitUntil: {visible: "text"}}]
func splitVisibleFromCommand(cmd map[string]interface{}) []map[string]interface{} {
	if _, isExtWait := cmd["extendedWaitUntil"]; isExtWait {
		return []map[string]interface{}{cmd}
	}
	vis, hasVis := cmd["visible"]
	nv, hasNV := cmd["notVisible"]
	if !hasVis && !hasNV {
		return []map[string]interface{}{cmd}
	}

	// Clone original without visible/notVisible
	cleaned := make(map[string]interface{})
	for k, v := range cmd {
		if k != "visible" && k != "notVisible" {
			cleaned[k] = v
		}
	}

	// Build extendedWaitUntil with the extracted condition
	waitInner := make(map[string]interface{})
	if hasVis {
		waitInner["visible"] = vis
	}
	if hasNV {
		waitInner["notVisible"] = nv
	}
	waitCmd := map[string]interface{}{"extendedWaitUntil": waitInner}

	return []map[string]interface{}{cleaned, waitCmd}
}

// fixCommandList recursively fixes a list of command maps (e.g. repeat.commands).
func fixCommandList(items []interface{}, aliases map[string]string) []interface{} {
	var result []interface{}
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			if fixed := fixCommandData(m, aliases); fixed != nil {
				result = append(result, fixed)
			}
		} else {
			result = append(result, item)
		}
	}
	return result
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

// parseFlowLine extracts flow name, pass/fail status, and duration from CLI output lines.
// CLI output format: "   ✅ 1. LoginFlow (234ms)"
func parseFlowLine(line string) (name, status, duration string) {
	trimmed := strings.TrimSpace(line)

	if strings.Contains(trimmed, "✅") || strings.Contains(trimmed, "PASS") {
		n, d := extractFlowNameAndDuration(trimmed)
		return n, "passed", d
	}
	if strings.Contains(trimmed, "❌") || strings.Contains(trimmed, "FAIL") {
		n, d := extractFlowNameAndDuration(trimmed)
		return n, "failed", d
	}

	return "", "", ""
}

var durationRegex = regexp.MustCompile(`\((\d[\dhms.]*(?:ms|s|m|h))\)\s*$`)
var leadingNumberRegex = regexp.MustCompile(`^\d+\.\s*`)

func extractFlowNameAndDuration(line string) (name, duration string) {
	// Extract duration first
	if m := durationRegex.FindStringSubmatch(line); len(m) == 2 {
		duration = m[1]
		line = line[:durationRegex.FindStringIndex(line)[0]]
	}

	line = strings.TrimSpace(line)
	for _, prefix := range []string{"✅", "❌", "PASS", "FAIL", ":", "-", " "} {
		line = strings.TrimPrefix(line, prefix)
	}
	line = strings.TrimSpace(line)

	// Remove leading number prefix like "1. " or "12. "
	line = leadingNumberRegex.ReplaceAllString(line, "")
	// Remove file extension
	line = strings.TrimSuffix(line, ".yaml")
	line = strings.TrimSuffix(line, ".yml")

	return strings.TrimSpace(line), duration
}

// countFlowFiles counts .yaml/.yml files in a directory.
func countFlowFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
			count++
		}
	}
	return count
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
