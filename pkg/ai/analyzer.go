package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
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

	// Add token expiry info
	tokenStatuses := checkURLTokenExpiry(gameURL)
	if len(tokenStatuses) > 0 {
		var statusParts []string
		var expiredNames []string
		for param, ts := range tokenStatuses {
			if ts.Expired {
				ago := time.Since(ts.ExpiresAt).Truncate(time.Minute)
				statusParts = append(statusParts, fmt.Sprintf("%s expired %s ago", param, ago))
				expiredNames = append(expiredNames, param)
			} else {
				remaining := time.Until(ts.ExpiresAt).Truncate(time.Minute)
				statusParts = append(statusParts, fmt.Sprintf("%s valid (%s remaining)", param, remaining))
			}
		}
		hints["tokenStatus"] = strings.Join(statusParts, ", ")
		if len(expiredNames) > 0 {
			hints["expiredTokens"] = strings.Join(expiredNames, ", ")
		}
	} else {
		hints["tokenStatus"] = "no tokens found"
	}

	return hints
}

// tokenExpiryInfo holds expiry information for a single JWT token found in URL parameters.
type tokenExpiryInfo struct {
	Expired          bool
	ExpiresAt        time.Time
	SecondsRemaining int
}

// checkURLTokenExpiry inspects URL query parameters for JWT-shaped values and
// returns expiry information for any tokens that contain an "exp" claim.
// A value is considered JWT-shaped if it has exactly 3 dot-separated segments
// where the middle segment is valid base64.
func checkURLTokenExpiry(gameURL string) map[string]tokenExpiryInfo {
	u, err := url.Parse(gameURL)
	if err != nil {
		return nil
	}

	result := map[string]tokenExpiryInfo{}
	for param, values := range u.Query() {
		if len(values) == 0 {
			continue
		}
		val := values[0]
		parts := strings.Split(val, ".")
		if len(parts) != 3 {
			continue
		}

		// Try to decode the payload (middle segment)
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			continue
		}

		// Parse as JSON and look for "exp" claim
		var claims map[string]interface{}
		if err := json.Unmarshal(payload, &claims); err != nil {
			continue
		}

		expVal, ok := claims["exp"]
		if !ok {
			continue
		}

		// exp should be a number (Unix timestamp)
		expFloat, ok := expVal.(float64)
		if !ok {
			continue
		}

		expiresAt := time.Unix(int64(expFloat), 0)
		now := time.Now()
		secondsRemaining := int(math.Max(0, expiresAt.Sub(now).Seconds()))

		result[param] = tokenExpiryInfo{
			Expired:          now.After(expiresAt),
			ExpiresAt:        expiresAt,
			SecondsRemaining: secondsRemaining,
		}
	}

	return result
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

// AnalyzeOption configures the analysis pipeline (checkpoint writing, resume).
type AnalyzeOption func(*analyzeOptions)

type analyzeOptions struct {
	resumeData    *CheckpointData
	checkpointDir string
}

// WithResumeData attaches checkpoint data to resume from.
func WithResumeData(rd *CheckpointData) AnalyzeOption {
	return func(o *analyzeOptions) { o.resumeData = rd }
}

// WithCheckpointDir enables checkpoint writing to the given directory.
func WithCheckpointDir(dir string) AnalyzeOption {
	return func(o *analyzeOptions) { o.checkpointDir = dir }
}

// AnalyzeFromURLWithMeta performs analysis and flow generation using pre-fetched page metadata.
// Use this to avoid double-fetching when the caller already has PageMeta.
func (a *Analyzer) AnalyzeFromURLWithMeta(ctx context.Context, gameURL string, pageMeta *scout.PageMeta) (*scout.PageMeta, *AnalysisResult, []*MaestroFlow, error) {
	return a.AnalyzeFromURLWithMetaProgress(ctx, gameURL, pageMeta, DefaultAnalysisModules(), nil /* no options */)
}

// collectScreenshots gathers all available screenshots from PageMeta.
// Returns the list of base64-encoded images (may be empty).
func collectScreenshots(pageMeta *scout.PageMeta) []string {
	if len(pageMeta.Screenshots) > 0 {
		return pageMeta.Screenshots
	}
	// Fall back to single legacy screenshot
	if pageMeta.ScreenshotB64 != "" {
		return []string{pageMeta.ScreenshotB64}
	}
	return nil
}

// buildPageMetaJSON creates a JSON representation of PageMeta for inclusion
// in text prompts, stripping out the large screenshot fields.
func buildPageMetaJSON(pageMeta *scout.PageMeta) []byte {
	type pageMetaForPrompt struct {
		Title       string            `json:"title"`
		Description string            `json:"description"`
		Framework   string            `json:"framework"`
		CanvasFound bool              `json:"canvasFound"`
		ScriptSrcs  []string          `json:"scriptSrcs"`
		MetaTags    map[string]string `json:"metaTags"`
		BodySnippet string            `json:"bodySnippet"`
		Links       []string          `json:"links"`
		JSGlobals   []string          `json:"jsGlobals,omitempty"`
	}
	promptMeta := pageMetaForPrompt{
		Title:       pageMeta.Title,
		Description: pageMeta.Description,
		Framework:   pageMeta.Framework,
		CanvasFound: pageMeta.CanvasFound,
		ScriptSrcs:  pageMeta.ScriptSrcs,
		MetaTags:    pageMeta.MetaTags,
		BodySnippet: pageMeta.BodySnippet,
		Links:       pageMeta.Links,
		JSGlobals:   pageMeta.JSGlobals,
	}
	data, _ := json.MarshalIndent(promptMeta, "", "  ")
	return data
}

// AnalyzeFromURLWithMetaProgress is the main analysis pipeline.
//
// Pipeline (2 AI calls max):
//  1. Comprehensive analysis + scenarios (single multimodal call with all screenshots)
//  2. Flow generation (multimodal call with screenshots + full structured JSON from step 1)
func (a *Analyzer) AnalyzeFromURLWithMetaProgress(
	ctx context.Context, gameURL string, pageMeta *scout.PageMeta,
	modules AnalysisModules, onProgress ProgressFunc,
	optFns ...AnalyzeOption,
) (*scout.PageMeta, *AnalysisResult, []*MaestroFlow, error) {
	var opts analyzeOptions
	for _, fn := range optFns {
		if fn != nil {
			fn(&opts)
		}
	}

	progress := func(step, message string) {
		if onProgress != nil {
			onProgress(step, message)
		}
	}

	// --- Resume path: skip AI Call #1 if checkpoint has analysis ---
	if opts.resumeData != nil && opts.resumeData.Step == "analyzed" && len(opts.resumeData.Analysis) > 0 {
		var comprehensiveResult ComprehensiveAnalysisResult
		if err := json.Unmarshal(opts.resumeData.Analysis, &comprehensiveResult); err == nil {
			progress("analyzed", "Resumed from checkpoint — skipping analysis")
			result := comprehensiveResult.ToAnalysisResult()
			scenarios := comprehensiveResult.Scenarios

			if !modules.TestFlows {
				progress("flows_done", "Test flow generation skipped (module disabled)")
				return pageMeta, result, nil, nil
			}

			progress("flows", fmt.Sprintf("Converting %d scenarios to Maestro flows...", len(scenarios)))
			// Flow generation without screenshots (text-only fallback)
			flows, err := a.generateFlowsStructured(gameURL, pageMeta.Framework, result, scenarios, nil)
			if err != nil {
				return pageMeta, result, nil, fmt.Errorf("flow generation failed: %w", err)
			}
			for _, flow := range flows {
				flow.URL = gameURL
			}
			totalCommands := 0
			for _, f := range flows {
				totalCommands += len(f.Commands)
			}
			progress("flows_done", fmt.Sprintf("Generated %d flows with %d total commands", len(flows), totalCommands))
			return pageMeta, result, flows, nil
		}
	}

	// Collect all screenshots for multimodal calls
	screenshots := collectScreenshots(pageMeta)
	pageMetaJSON := buildPageMetaJSON(pageMeta)

	urlHints := parseURLHints(gameURL)
	urlHintsJSON, _ := json.Marshal(urlHints)

	screenshotSection := ""
	switch len(screenshots) {
	case 0:
		// no screenshots
	case 1:
		screenshotSection = "A screenshot of the game is attached. Describe what you see and use it to identify UI elements, buttons, game state, and interactive regions."
	default:
		screenshotSection = fmt.Sprintf("%d screenshots of the game are attached, showing different states (initial load, after interactions). Examine ALL screenshots to understand the game's UI, state transitions, and interactive elements.", len(screenshots))
	}

	// --- AI Call #1: Comprehensive analysis + scenarios ---
	promptTemplate := BuildAnalysisPrompt(modules)
	prompt := fillTemplate(promptTemplate, map[string]string{
		"url":               gameURL,
		"pageMeta":          string(pageMetaJSON),
		"urlHints":          string(urlHintsJSON),
		"screenshotSection": screenshotSection,
	})

	// Progress reporting
	promptTokenEstimate := len(prompt) / 4
	mode := "text-only"
	if len(screenshots) > 0 {
		totalKB := 0
		for _, s := range screenshots {
			totalKB += len(s) * 3 / 4 / 1024
		}
		mode = fmt.Sprintf("multimodal (%d screenshots, ~%d KB)", len(screenshots), totalKB)
	}
	progress("analyzing", fmt.Sprintf("Sending to AI (%s, ~%dk prompt tokens)...", mode, promptTokenEstimate/1000))

	var comprehensiveResult *ComprehensiveAnalysisResult
	var result *AnalysisResult
	var scenarios []TestScenario

	// Try multimodal with AnalyzeWithImages (preferred path — uses system prompt + all screenshots)
	if len(screenshots) > 0 {
		if imgAnalyzer, ok := a.Client.(ImageAnalyzer); ok {
			response, imgErr := imgAnalyzer.AnalyzeWithImages(AnalysisSystemPrompt, prompt, screenshots)
			if imgErr != nil {
				return pageMeta, nil, nil, fmt.Errorf("AI multimodal analysis failed: %w", imgErr)
			}
			parsed, parseErr := parseComprehensiveJSON(response)
			if parseErr == nil {
				comprehensiveResult = parsed
			} else {
				// Try parsing as legacy AnalysisResult (AI may not include scenarios)
				var legacyResult AnalysisResult
				if jsonErr := json.Unmarshal([]byte(stripCodeFences(response)), &legacyResult); jsonErr == nil && len(legacyResult.Mechanics) > 0 {
					result = &legacyResult
				} else {
					result = &AnalysisResult{RawResponse: response}
				}
			}
		}
	}

	// Fallback to text-only if multimodal was not used or not available
	if comprehensiveResult == nil && result == nil {
		progress("analyzing", "Sending to AI (text-only fallback)...")
		var err error
		result, err = a.Client.Analyze(prompt, map[string]interface{}{
			"url":      gameURL,
			"pageMeta": string(pageMetaJSON),
		})
		if err != nil {
			return pageMeta, nil, nil, fmt.Errorf("AI analysis failed: %w", err)
		}
	}

	// Extract result and scenarios from comprehensive response
	if comprehensiveResult != nil {
		result = comprehensiveResult.ToAnalysisResult()
		scenarios = comprehensiveResult.Scenarios
	}

	// Retry with more aggressive prompt if analysis found no mechanics and we have screenshots
	if len(result.Mechanics) == 0 && len(screenshots) > 0 {
		progress("analyzing", "No mechanics found — retrying with focused screenshot analysis...")
		if imgAnalyzer, ok := a.Client.(ImageAnalyzer); ok {
			retryPrompt := fmt.Sprintf(`The previous analysis found no game mechanics. Look at the screenshots more carefully.

Game URL: %s
URL Hints: %s

This is likely a %s game. Even if the page source is minimal (SPA/JS-rendered),
you MUST identify mechanics from the game type, URL parameters, and what you see in the screenshots.

Generate at least 3 mechanics, 3 UI elements, and 2 user flows.

Respond with structured JSON matching the ComprehensiveAnalysisResult format (gameInfo, mechanics, uiElements, userFlows, edgeCases, scenarios).`,
				gameURL, string(urlHintsJSON), urlHints["gameType"])

			retryResp, retryErr := imgAnalyzer.AnalyzeWithImages(AnalysisSystemPrompt, retryPrompt, screenshots)
			if retryErr == nil {
				if parsed, parseErr := parseComprehensiveJSON(retryResp); parseErr == nil && len(parsed.Mechanics) > 0 {
					comprehensiveResult = parsed
					result = parsed.ToAnalysisResult()
					scenarios = parsed.Scenarios
				}
			}
		}
	}

	// Report analysis results
	analysisDetail := fmt.Sprintf("Found %d mechanics, %d UI elements, %d user flows, %d edge cases",
		len(result.Mechanics), len(result.UIElements), len(result.UserFlows), len(result.EdgeCases))
	if result.GameInfo.Name != "" {
		analysisDetail = fmt.Sprintf("%s — %s (%s)", result.GameInfo.Name, result.GameInfo.Genre, analysisDetail)
	}
	progress("analyzed", analysisDetail)

	// Write checkpoint after analysis succeeds
	if opts.checkpointDir != "" && comprehensiveResult != nil {
		pageMetaJSON, _ := json.Marshal(pageMeta)
		analysisJSON, _ := json.Marshal(comprehensiveResult)
		if cpErr := WriteCheckpoint(opts.checkpointDir, CheckpointData{
			Step:     "analyzed",
			PageMeta: pageMetaJSON,
			Analysis: analysisJSON,
			Modules:  modules,
		}); cpErr != nil {
			progress("checkpoint", fmt.Sprintf("Warning: failed to write checkpoint: %v", cpErr))
		}
	}

	// If comprehensive call produced scenarios, report them
	if len(scenarios) > 0 {
		scenarioTypes := map[string]int{}
		for _, s := range scenarios {
			scenarioTypes[s.Type]++
		}
		typeSummary := ""
		for t, c := range scenarioTypes {
			if typeSummary != "" {
				typeSummary += ", "
			}
			typeSummary += fmt.Sprintf("%d %s", c, t)
		}
		if typeSummary == "" {
			typeSummary = fmt.Sprintf("%d total", len(scenarios))
		}
		progress("scenarios_done", fmt.Sprintf("Generated %d scenarios (%s) — included in analysis call", len(scenarios), typeSummary))
	} else {
		// Fallback: generate scenarios in a separate call (legacy path)
		progress("scenarios", "Generating test scenarios from analysis...")
		var err error
		scenarios, err = a.GenerateScenarios(result)
		if err != nil {
			return pageMeta, result, nil, fmt.Errorf("scenario generation failed: %w", err)
		}
		scenarioTypes := map[string]int{}
		for _, s := range scenarios {
			scenarioTypes[s.Type]++
		}
		typeSummary := ""
		for t, c := range scenarioTypes {
			if typeSummary != "" {
				typeSummary += ", "
			}
			typeSummary += fmt.Sprintf("%d %s", c, t)
		}
		if typeSummary == "" {
			typeSummary = fmt.Sprintf("%d total", len(scenarios))
		}
		progress("scenarios_done", fmt.Sprintf("Generated %d scenarios (%s)", len(scenarios), typeSummary))
	}

	// Skip flow generation if test flows module is disabled
	if !modules.TestFlows {
		progress("flows_done", "Test flow generation skipped (module disabled)")
		return pageMeta, result, nil, nil
	}

	progress("flows", fmt.Sprintf("Converting %d scenarios to Maestro flows...", len(scenarios)))

	// --- AI Call #2: Flow generation (multimodal with screenshots + full structured JSON) ---
	flows, err := a.generateFlowsStructured(gameURL, pageMeta.Framework, result, scenarios, screenshots)
	if err != nil {
		return pageMeta, result, nil, fmt.Errorf("flow generation failed: %w", err)
	}

	// Set URL on each flow
	for _, flow := range flows {
		flow.URL = gameURL
	}

	// Summarize flows for progress
	totalCommands := 0
	for _, f := range flows {
		totalCommands += len(f.Commands)
	}
	progress("flows_done", fmt.Sprintf("Generated %d flows with %d total commands", len(flows), totalCommands))

	return pageMeta, result, flows, nil
}

// generateFlowsStructured generates Maestro flows using structured JSON input and multimodal screenshots.
// This replaces the old GenerateFlows which used lossy text conversion and YAML output.
func (a *Analyzer) generateFlowsStructured(gameURL, framework string, result *AnalysisResult, scenarios []TestScenario, screenshots []string) ([]*MaestroFlow, error) {
	// Build the full analysis JSON to pass to the flow generation prompt
	analysisForFlow := struct {
		GameInfo   GameInfo       `json:"gameInfo"`
		Mechanics  []Mechanic     `json:"mechanics"`
		UIElements []UIElement    `json:"uiElements"`
		UserFlows  []UserFlow     `json:"userFlows"`
		EdgeCases  []EdgeCase     `json:"edgeCases"`
		Scenarios  []TestScenario `json:"scenarios"`
	}{
		GameInfo:   result.GameInfo,
		Mechanics:  result.Mechanics,
		UIElements: result.UIElements,
		UserFlows:  result.UserFlows,
		EdgeCases:  result.EdgeCases,
		Scenarios:  scenarios,
	}
	analysisJSON, _ := json.MarshalIndent(analysisForFlow, "", "  ")

	screenshotSection := ""
	switch len(screenshots) {
	case 0:
		// no screenshots
	case 1:
		screenshotSection = "A screenshot of the game is attached. Use it to ground coordinate-based interactions in what you actually see."
	default:
		screenshotSection = fmt.Sprintf("%d screenshots of the game are attached showing different states. Use them to ground coordinate-based interactions in what you actually see.", len(screenshots))
	}

	prompt := fillTemplate(FlowGenerationPrompt.Template, map[string]string{
		"url":               gameURL,
		"framework":         framework,
		"analysisJSON":      string(analysisJSON),
		"screenshotSection": screenshotSection,
	})

	var response string

	// Prefer multimodal with screenshots for flow generation too
	if len(screenshots) > 0 {
		if imgAnalyzer, ok := a.Client.(ImageAnalyzer); ok {
			var err error
			response, err = imgAnalyzer.AnalyzeWithImages(AnalysisSystemPrompt, prompt, screenshots)
			if err != nil {
				return nil, fmt.Errorf("AI multimodal flow generation failed: %w", err)
			}
		}
	}

	// Fallback to text-only
	if response == "" {
		var err error
		response, err = a.Client.Generate(prompt, map[string]interface{}{
			"analysisJSON": string(analysisJSON),
		})
		if err != nil {
			return nil, fmt.Errorf("flow generation failed: %w", err)
		}
	}

	// Try parsing as structured JSON first (preferred)
	flows := parseFlowsJSON(response)
	if len(flows) > 0 {
		return flows, nil
	}

	// Fallback to YAML parsing for backward compatibility
	yamlFlows := a.parseFlowsFromResponse(response, scenarios)
	return yamlFlows, nil
}

// parseFlowsJSON attempts to parse the AI response as a JSON array of MaestroFlow.
func parseFlowsJSON(response string) []*MaestroFlow {
	flows, err := parseJSONArrayFromAI[*MaestroFlow](response)
	if err != nil {
		return nil
	}
	return flows
}

// parseComprehensiveJSON attempts to parse the AI response as a ComprehensiveAnalysisResult.
func parseComprehensiveJSON(response string) (*ComprehensiveAnalysisResult, error) {
	return parseJSONObjectFromAI[ComprehensiveAnalysisResult](response, func(r *ComprehensiveAnalysisResult) bool {
		return len(r.Mechanics) > 0
	})
}

// GenerateScenarios generates test scenarios from game analysis (legacy path).
// The primary pipeline now generates scenarios as part of the comprehensive analysis call.
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

// GenerateFlows generates Maestro flows from test scenarios (legacy path).
// The primary pipeline now uses generateFlowsStructured.
func (a *Analyzer) GenerateFlows(scenarios []TestScenario) ([]*MaestroFlow, error) {
	// Convert scenarios to string
	scenariosStr := a.scenariosToString(scenarios)

	// Build prompt using legacy template format
	prompt := fillTemplate(ScenarioGenerationPrompt.Template, map[string]string{
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

// scenariosToString converts scenarios to readable string (used by legacy GenerateFlows path)
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

// parseFlowsFromResponse extracts Maestro flows from AI response (YAML fallback path)
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

// parseJSONArrayFromAI strips code fences and parses a JSON array from an AI response.
// Falls back to extracting the substring between the first '[' and last ']'.
func parseJSONArrayFromAI[T any](response string) ([]T, error) {
	cleaned := stripCodeFences(response)

	var result []T
	if err := json.Unmarshal([]byte(cleaned), &result); err == nil && len(result) > 0 {
		return result, nil
	}

	start := strings.Index(cleaned, "[")
	end := strings.LastIndex(cleaned, "]")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(cleaned[start:end+1]), &result); err == nil && len(result) > 0 {
			return result, nil
		}
	}

	return nil, fmt.Errorf("could not parse JSON array from response")
}

// parseJSONObjectFromAI strips code fences and parses a JSON object from an AI response.
// Falls back to extracting the substring between the first '{' and last '}'.
// The validate function is called to confirm the parsed result is meaningful (e.g., has required fields).
func parseJSONObjectFromAI[T any](response string, validate func(*T) bool) (*T, error) {
	cleaned := stripCodeFences(response)

	var result T
	if err := json.Unmarshal([]byte(cleaned), &result); err == nil && validate(&result) {
		return &result, nil
	}

	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(cleaned[start:end+1]), &result); err == nil && validate(&result) {
			return &result, nil
		}
	}

	return nil, fmt.Errorf("could not parse JSON object from response")
}

// parseScenarioJSON attempts to parse the AI response as a JSON array of TestScenario.
func parseScenarioJSON(response string) ([]TestScenario, error) {
	return parseJSONArrayFromAI[TestScenario](response)
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
			// Simple string command like "launchApp" -> {"launchApp": ""}
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
