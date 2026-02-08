package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/scout"
	"github.com/Global-Wizards/wizards-qa/pkg/util"
	"gopkg.in/yaml.v3"
)

// Analyzer analyzes games and generates test scenarios
type Analyzer struct {
	Client Client
}

// NewAnalyzer creates a new game analyzer
func NewAnalyzer(client Client) *Analyzer {
	return &Analyzer{
		Client: client,
	}
}

// NewAnalyzerFromConfig creates an analyzer from configuration
func NewAnalyzerFromConfig(provider, apiKey, model string, temperature float64, maxTokens int) (*Analyzer, error) {
	var client Client

	switch provider {
	case "anthropic", "claude":
		client = NewClaudeClient(apiKey, model, temperature, maxTokens)
	case "google", "gemini":
		client = NewGeminiClient(apiKey, model, temperature, maxTokens)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s (use 'anthropic' or 'google')", provider)
	}

	return &Analyzer{Client: client}, nil
}

// parseURLHints extracts game-relevant hints from URL parameters.
func parseURLHints(gameURL string) map[string]string {
	u, err := url.Parse(gameURL)
	if err != nil {
		return nil
	}
	hints := map[string]string{}
	if gt := u.Query().Get("game_type"); gt != "" {
		hints["gameType"] = gt
	}
	if mode := u.Query().Get("mode"); mode != "" {
		hints["mode"] = mode
	}
	if gid := u.Query().Get("game_id"); gid != "" {
		hints["gameId"] = gid
	}
	hints["domain"] = u.Host
	return hints
}

// AnalyzeGame analyzes a game from specification and URL
func (a *Analyzer) AnalyzeGame(specPath, gameURL string) (*AnalysisResult, error) {
	// Read game specification
	specContent, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	// Build prompt
	prompt := fillTemplate(GameAnalysisPrompt.Template, map[string]string{
		"spec": string(specContent),
		"url":  gameURL,
	})

	// Call AI
	result, err := a.Client.Analyze(prompt, map[string]interface{}{
		"spec": string(specContent),
		"url":  gameURL,
	})
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	return result, nil
}

// AnalyzeFromURL performs the full pipeline: scout page, analyze with AI, generate flows.
func (a *Analyzer) AnalyzeFromURL(ctx context.Context, gameURL string, timeout time.Duration) (*scout.PageMeta, *AnalysisResult, []*MaestroFlow, error) {
	// Step 1: Scout the page
	pageMeta, err := scout.ScoutURL(ctx, gameURL, timeout)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("page scout failed: %w", err)
	}

	return a.AnalyzeFromURLWithMeta(ctx, gameURL, pageMeta)
}

// ProgressFunc is a callback for reporting analysis progress.
type ProgressFunc func(step, message string)

// AnalyzeFromURLWithMeta performs analysis and flow generation using pre-fetched page metadata.
// Use this to avoid double-fetching when the caller already has PageMeta.
func (a *Analyzer) AnalyzeFromURLWithMeta(ctx context.Context, gameURL string, pageMeta *scout.PageMeta) (*scout.PageMeta, *AnalysisResult, []*MaestroFlow, error) {
	return a.AnalyzeFromURLWithMetaProgress(ctx, gameURL, pageMeta, nil)
}

// AnalyzeFromURLWithMetaProgress is like AnalyzeFromURLWithMeta but reports granular progress via onProgress.
func (a *Analyzer) AnalyzeFromURLWithMetaProgress(
	ctx context.Context, gameURL string, pageMeta *scout.PageMeta,
	onProgress ProgressFunc,
) (*scout.PageMeta, *AnalysisResult, []*MaestroFlow, error) {
	progress := func(step, message string) {
		if onProgress != nil {
			onProgress(step, message)
		}
	}

	// Step 2: Build prompt with page metadata and URL hints
	pageMetaJSON, _ := json.MarshalIndent(pageMeta, "", "  ")

	urlHints := parseURLHints(gameURL)
	urlHintsJSON, _ := json.Marshal(urlHints)

	screenshotSection := ""
	if pageMeta.ScreenshotB64 != "" {
		screenshotSection = "A screenshot of the game is attached. Describe what you see and use it to identify UI elements, buttons, game state, and interactive regions."
	}

	prompt := fillTemplate(URLAnalysisPrompt.Template, map[string]string{
		"url":               gameURL,
		"pageMeta":          string(pageMetaJSON),
		"framework":         pageMeta.Framework,
		"urlHints":          string(urlHintsJSON),
		"screenshotSection": screenshotSection,
	})

	progress("analyzing", "Analyzing game mechanics with AI...")

	// Step 3: Call AI for analysis — use multimodal when screenshot is available
	var result *AnalysisResult
	if pageMeta.ScreenshotB64 != "" {
		if imgAnalyzer, ok := a.Client.(ImageAnalyzer); ok {
			response, imgErr := imgAnalyzer.AnalyzeWithImage(prompt, pageMeta.ScreenshotB64)
			if imgErr != nil {
				return pageMeta, nil, nil, fmt.Errorf("AI multimodal analysis failed: %w", imgErr)
			}
			var parsed AnalysisResult
			if jsonErr := json.Unmarshal([]byte(response), &parsed); jsonErr != nil {
				parsed = AnalysisResult{RawResponse: response}
			}
			result = &parsed
		}
	}

	// Fallback to text-only if multimodal was not used or not available
	if result == nil {
		var err error
		result, err = a.Client.Analyze(prompt, map[string]interface{}{
			"url":      gameURL,
			"pageMeta": string(pageMetaJSON),
		})
		if err != nil {
			return pageMeta, nil, nil, fmt.Errorf("AI analysis failed: %w", err)
		}
	}

	// Retry with more aggressive prompt if analysis found no mechanics and we have a screenshot
	if len(result.Mechanics) == 0 && pageMeta.ScreenshotB64 != "" {
		if imgAnalyzer, ok := a.Client.(ImageAnalyzer); ok {
			retryPrompt := fmt.Sprintf(`The previous analysis found no game mechanics. Look at the screenshot more carefully.

Game URL: %s
URL Hints: %s

This is likely a %s game. Even if the page source is minimal (SPA/JS-rendered),
you MUST infer mechanics from the game type, URL parameters, and what you see in the screenshot.

Generate at least 3 mechanics, 3 UI elements, and 2 user flows.

Respond with structured JSON matching the AnalysisResult format (gameInfo, mechanics, uiElements, userFlows, edgeCases).`,
				gameURL, string(urlHintsJSON), urlHints["gameType"])

			retryResp, retryErr := imgAnalyzer.AnalyzeWithImage(retryPrompt, pageMeta.ScreenshotB64)
			if retryErr == nil {
				var retryResult AnalysisResult
				if jsonErr := json.Unmarshal([]byte(retryResp), &retryResult); jsonErr == nil && len(retryResult.Mechanics) > 0 {
					result = &retryResult
				}
			}
		}
	}

	progress("analyzed", fmt.Sprintf("Found %d mechanics, %d UI elements", len(result.Mechanics), len(result.UIElements)))
	progress("scenarios", "Generating test scenarios...")

	// Step 4: Generate scenarios
	scenarios, err := a.GenerateScenarios(result)
	if err != nil {
		return pageMeta, result, nil, fmt.Errorf("scenario generation failed: %w", err)
	}

	progress("scenarios_done", fmt.Sprintf("Generated %d scenarios", len(scenarios)))
	progress("flows", "Generating Maestro test flows...")

	// Step 5: Generate flows
	flows, err := a.GenerateFlows(scenarios)
	if err != nil {
		return pageMeta, result, nil, fmt.Errorf("flow generation failed: %w", err)
	}

	// Step 6: Set URL on each flow
	for _, flow := range flows {
		flow.URL = gameURL
	}

	progress("flows_done", fmt.Sprintf("Generated %d flows", len(flows)))

	return pageMeta, result, flows, nil
}

// GenerateScenarios generates test scenarios from game analysis
func (a *Analyzer) GenerateScenarios(analysis *AnalysisResult) ([]TestScenario, error) {
	// Convert analysis to string for context
	analysisStr := a.analysisToString(analysis)

	// Build prompt
	prompt := fillTemplate(ScenarioGenerationPrompt.Template, map[string]string{
		"analysis": analysisStr,
	})

	// Call AI
	response, err := a.Client.Generate(prompt, map[string]interface{}{
		"analysis": analysisStr,
	})
	if err != nil {
		return nil, fmt.Errorf("scenario generation failed: %w", err)
	}

	scenarios, parseErr := parseScenarioJSON(response)
	if parseErr != nil {
		// Fallback: wrap raw response as single scenario (preserves legacy behavior)
		scenarios = []TestScenario{
			{
				Name:        "AI Generated Scenarios",
				Description: "Scenarios generated by AI",
				Type:        "mixed",
				Priority:    "high",
				Steps: []Step{
					{
						Action:   "raw",
						Target:   "response",
						Expected: response,
					},
				},
			},
		}
	}

	return scenarios, nil
}

// GenerateFlows generates Maestro flows from test scenarios
func (a *Analyzer) GenerateFlows(scenarios []TestScenario) ([]*MaestroFlow, error) {
	// Convert scenarios to string
	scenariosStr := a.scenariosToString(scenarios)

	// Build prompt
	prompt := fillTemplate(FlowGenerationPrompt.Template, map[string]string{
		"scenarios": scenariosStr,
	})

	// Call AI
	response, err := a.Client.Generate(prompt, map[string]interface{}{
		"scenarios": scenariosStr,
	})
	if err != nil {
		return nil, fmt.Errorf("flow generation failed: %w", err)
	}

	// Parse YAML flows from response
	flows := a.parseFlowsFromResponse(response, scenarios)

	return flows, nil
}

// analysisToString converts analysis to readable string
func (a *Analyzer) analysisToString(analysis *AnalysisResult) string {
	var sb strings.Builder

	if analysis.GameInfo.Name != "" {
		sb.WriteString(fmt.Sprintf("Game: %s\n", analysis.GameInfo.Name))
		sb.WriteString(fmt.Sprintf("Description: %s\n", analysis.GameInfo.Description))
		sb.WriteString(fmt.Sprintf("Genre: %s\n", analysis.GameInfo.Genre))
		sb.WriteString(fmt.Sprintf("Technology: %s\n\n", analysis.GameInfo.Technology))
	}

	if len(analysis.Mechanics) > 0 {
		sb.WriteString("Mechanics:\n")
		for _, m := range analysis.Mechanics {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", m.Name, m.Description))
		}
		sb.WriteString("\n")
	}

	if len(analysis.UIElements) > 0 {
		sb.WriteString("UI Elements:\n")
		for _, ui := range analysis.UIElements {
			sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", ui.Name, ui.Type, ui.Selector))
		}
		sb.WriteString("\n")
	}

	if len(analysis.UserFlows) > 0 {
		sb.WriteString("User Flows:\n")
		for _, flow := range analysis.UserFlows {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", flow.Name, flow.Description))
		}
		sb.WriteString("\n")
	}

	if analysis.RawResponse != "" {
		sb.WriteString("Raw AI Response:\n")
		sb.WriteString(analysis.RawResponse)
		sb.WriteString("\n")
	}

	return sb.String()
}

// scenariosToString converts scenarios to readable string
func (a *Analyzer) scenariosToString(scenarios []TestScenario) string {
	var sb strings.Builder

	for i, scenario := range scenarios {
		sb.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, scenario.Name, scenario.Type))
		sb.WriteString(fmt.Sprintf("   Description: %s\n", scenario.Description))
		sb.WriteString(fmt.Sprintf("   Priority: %s\n", scenario.Priority))
		
		if len(scenario.Steps) > 0 {
			sb.WriteString("   Steps:\n")
			for j, step := range scenario.Steps {
				sb.WriteString(fmt.Sprintf("   %d. %s %s", j+1, step.Action, step.Target))
				if step.Value != "" {
					sb.WriteString(fmt.Sprintf(": %s", step.Value))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// parseFlowsFromResponse extracts Maestro flows from AI response
func (a *Analyzer) parseFlowsFromResponse(response string, scenarios []TestScenario) []*MaestroFlow {
	cleaned := stripCodeFences(response)
	docs := splitYAMLDocuments(cleaned)

	var flows []*MaestroFlow
	for i, doc := range docs {
		flow := parseYAMLDocument(doc)
		if flow == nil {
			continue
		}

		// Improve flow naming
		if flow.Name == "" {
			if i < len(scenarios) && scenarios[i].Name != "" {
				flow.Name = scenarios[i].Name
			} else {
				flow.Name = fmt.Sprintf("Flow %d", i+1)
			}
		}

		flows = append(flows, flow)
	}

	if len(flows) == 0 {
		// Fallback: wrap raw response as comment (preserves legacy behavior)
		flows = []*MaestroFlow{
			{
				Name: "AI Generated Flow",
				Commands: []map[string]interface{}{
					{"comment": response},
				},
			},
		}
	}

	return flows
}

// --- Parsing helpers ---

var codeFenceRe = regexp.MustCompile("(?s)```(?:json|yaml|yml)?\\s*\n?(.*?)```")

// stripCodeFences removes markdown code fences (```json ... ``` or ```yaml ... ```) from AI responses.
func stripCodeFences(s string) string {
	matches := codeFenceRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return s
	}
	var parts []string
	for _, m := range matches {
		parts = append(parts, m[1])
	}
	return strings.Join(parts, "\n")
}

// parseScenarioJSON attempts to parse the AI response as a JSON array of TestScenario.
func parseScenarioJSON(response string) ([]TestScenario, error) {
	cleaned := stripCodeFences(response)

	// Try direct unmarshal
	var scenarios []TestScenario
	if err := json.Unmarshal([]byte(cleaned), &scenarios); err == nil && len(scenarios) > 0 {
		return scenarios, nil
	}

	// Fallback: extract JSON array between first [ and last ]
	start := strings.Index(cleaned, "[")
	end := strings.LastIndex(cleaned, "]")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(cleaned[start:end+1]), &scenarios); err == nil && len(scenarios) > 0 {
			return scenarios, nil
		}
	}

	return nil, fmt.Errorf("could not parse scenarios from response")
}

// splitYAMLDocuments splits a string on YAML document separators (---).
func splitYAMLDocuments(s string) []string {
	lines := strings.Split(s, "\n")
	var docs []string
	var current []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if len(current) > 0 {
				doc := strings.TrimSpace(strings.Join(current, "\n"))
				if doc != "" {
					docs = append(docs, doc)
				}
			}
			current = nil
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		doc := strings.TrimSpace(strings.Join(current, "\n"))
		if doc != "" {
			docs = append(docs, doc)
		}
	}

	return docs
}

// parseYAMLDocument parses a single YAML document into a MaestroFlow.
func parseYAMLDocument(doc string) *MaestroFlow {
	// Try as a structured object with name/appId/commands keys
	var structured map[string]interface{}
	if err := yaml.Unmarshal([]byte(doc), &structured); err == nil && structured != nil {
		flow := &MaestroFlow{}
		if name, ok := structured["name"].(string); ok {
			flow.Name = name
		}
		if appId, ok := structured["appId"].(string); ok {
			flow.AppId = appId
		}
		if url, ok := structured["url"].(string); ok {
			flow.URL = url
		}
		if tags, ok := structured["tags"].([]interface{}); ok {
			for _, t := range tags {
				if ts, ok := t.(string); ok {
					flow.Tags = append(flow.Tags, ts)
				}
			}
		}

		// If it has a commands key, use that
		if cmds, ok := structured["commands"].([]interface{}); ok {
			flow.Commands = convertCommandList(cmds)
			return flow
		}

		// Otherwise try entire doc as a bare command list
		var list []interface{}
		if err := yaml.Unmarshal([]byte(doc), &list); err == nil && len(list) > 0 {
			flow.Commands = convertCommandList(list)
			return flow
		}

		// Has at least some metadata — return what we got
		if flow.Name != "" || flow.AppId != "" || flow.URL != "" {
			return flow
		}
	}

	// Try as a bare command list ([]interface{})
	var list []interface{}
	if err := yaml.Unmarshal([]byte(doc), &list); err == nil && len(list) > 0 {
		return &MaestroFlow{
			Commands: convertCommandList(list),
		}
	}

	return nil
}

// convertCommandList converts a []interface{} YAML list into Maestro command maps.
func convertCommandList(items []interface{}) []map[string]interface{} {
	var commands []map[string]interface{}
	for _, item := range items {
		switch v := item.(type) {
		case map[string]interface{}:
			commands = append(commands, v)
		case string:
			// Simple string command like "launchApp" → {"launchApp": ""}
			commands = append(commands, map[string]interface{}{v: ""})
		}
	}
	return commands
}

// fillTemplate fills a template with variables
func fillTemplate(template string, vars map[string]string) string {
	result := template
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// WriteFlowsToFiles writes Maestro flows to YAML files
func WriteFlowsToFiles(flows []*MaestroFlow, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for i, flow := range flows {
		filename := fmt.Sprintf("%02d-%s.yaml", i+1, util.SanitizeFilename(flow.Name))
		filepath := fmt.Sprintf("%s/%s", outputDir, filename)

		// Convert flow to YAML
		yamlContent := flowToYAML(flow)

		if err := os.WriteFile(filepath, []byte(yamlContent), 0644); err != nil {
			return fmt.Errorf("failed to write flow file: %w", err)
		}
	}

	return nil
}

// flowToYAML converts a MaestroFlow to YAML string
func flowToYAML(flow *MaestroFlow) string {
	var sb strings.Builder

	// Write metadata
	hasMetadata := false
	if flow.URL != "" {
		sb.WriteString(fmt.Sprintf("url: %s\n", flow.URL))
		hasMetadata = true
	}
	if flow.AppId != "" {
		sb.WriteString(fmt.Sprintf("appId: %s\n", flow.AppId))
		hasMetadata = true
	}
	if len(flow.Tags) > 0 {
		sb.WriteString("tags:\n")
		for _, tag := range flow.Tags {
			sb.WriteString(fmt.Sprintf("  - %s\n", tag))
		}
		hasMetadata = true
	}

	// Separator — only include when there's metadata (parser splits on \n---\n)
	if hasMetadata {
		sb.WriteString("---\n")
	}

	// Write commands
	for _, cmd := range flow.Commands {
		sb.WriteString(commandToYAML(cmd, 0))
	}

	return sb.String()
}

// commandToYAML converts a command map to YAML string
func commandToYAML(cmd map[string]interface{}, indent int) string {
	var sb strings.Builder
	prefix := strings.Repeat("  ", indent)

	for key, value := range cmd {
		switch v := value.(type) {
		case string:
			if key == "comment" {
				sb.WriteString(fmt.Sprintf("%s# %s\n", prefix, v))
			} else if v == "" {
				// Simple command with no value (e.g. "launchApp")
				sb.WriteString(fmt.Sprintf("%s- %s\n", prefix, key))
			} else if strings.ContainsAny(v, ",:%{}[]") {
				// Quote values containing YAML-special characters
				sb.WriteString(fmt.Sprintf("%s- %s: \"%s\"\n", prefix, key, v))
			} else {
				sb.WriteString(fmt.Sprintf("%s- %s: %s\n", prefix, key, v))
			}
		case map[string]interface{}:
			sb.WriteString(fmt.Sprintf("%s- %s:\n", prefix, key))
			for subKey, subValue := range v {
				subStr := fmt.Sprintf("%v", subValue)
				if strings.ContainsAny(subStr, ",:%{}[]") {
					sb.WriteString(fmt.Sprintf("%s    %s: \"%s\"\n", prefix, subKey, subStr))
				} else {
					sb.WriteString(fmt.Sprintf("%s    %s: %v\n", prefix, subKey, subValue))
				}
			}
		default:
			sb.WriteString(fmt.Sprintf("%s- %s: %v\n", prefix, key, v))
		}
	}

	return sb.String()
}

