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
	"sort"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/scout"
	"github.com/Global-Wizards/wizards-qa/pkg/util"
	"gopkg.in/yaml.v3"
)

// Analyzer analyzes games and generates test scenarios
type Analyzer struct {
	Client Client
	Usage  TokenUsage
}

// NewAnalyzer creates a new game analyzer
func NewAnalyzer(client Client) *Analyzer {
	a := &Analyzer{Client: client}
	if bc := baseClientOf(client); bc != nil {
		bc.OnUsage = func(input, output, cacheCreate, cacheRead int) {
			a.Usage.Add(input, output, cacheCreate, cacheRead)
		}
	}
	return a
}

// baseClientOf extracts the *BaseClient from a Client implementation.
func baseClientOf(client Client) *BaseClient {
	switch c := client.(type) {
	case *ClaudeClient:
		return &c.BaseClient
	case *GeminiClient:
		return &c.BaseClient
	default:
		return nil
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

	return NewAnalyzer(client), nil
}

// emitCostEstimate sends a cost_estimate progress event with accumulated token usage as structured JSON.
func (a *Analyzer) emitCostEstimate(progress func(step, message string)) {
	if a.Usage.APICallCount == 0 {
		return
	}
	model := ""
	if bc := baseClientOf(a.Client); bc != nil {
		model = bc.Model
	}
	cost := a.Usage.EstimatedCost(model)
	data := map[string]interface{}{
		"inputTokens":       a.Usage.InputTokens,
		"outputTokens":      a.Usage.OutputTokens,
		"cacheReadTokens":   a.Usage.CacheReadInputTokens,
		"cacheCreateTokens": a.Usage.CacheCreationInputTokens,
		"totalTokens":       a.Usage.TotalTokens,
		"apiCallCount":      a.Usage.APICallCount,
		"costUsd":           cost,
		"credits":           int(math.Ceil(cost * 100)),
		"model":             model,
	}
	jsonBytes, _ := json.Marshal(data)
	progress("cost_estimate", string(jsonBytes))
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
	defer a.emitCostEstimate(progress)

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

			scenarioNames := make([]string, 0, len(scenarios))
			for _, s := range scenarios {
				scenarioNames = append(scenarioNames, s.Name)
			}
			progress("flows", fmt.Sprintf("Converting %d scenarios to Maestro flows: %s", len(scenarios), strings.Join(scenarioNames, ", ")))
			// Flow generation without screenshots (text-only fallback)
			flows, err := a.generateFlowsStructured(gameURL, pageMeta.Framework, result, scenarios, nil, progress)
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

	scenarioNames := make([]string, 0, len(scenarios))
	for _, s := range scenarios {
		scenarioNames = append(scenarioNames, s.Name)
	}
	progress("flows", fmt.Sprintf("Converting %d scenarios to Maestro flows: %s", len(scenarios), strings.Join(scenarioNames, ", ")))

	// --- AI Call #2: Flow generation (multimodal with screenshots + full structured JSON) ---
	flows, err := a.generateFlowsStructured(gameURL, pageMeta.Framework, result, scenarios, screenshots, progress)
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
func (a *Analyzer) generateFlowsStructured(gameURL, framework string, result *AnalysisResult, scenarios []TestScenario, screenshots []string, onProgress ProgressFunc) ([]*MaestroFlow, error) {
	progress := func(step, message string) {
		if onProgress != nil {
			onProgress(step, message)
		}
	}

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

	progress("flows_prompt", fmt.Sprintf("Built prompt from %d scenarios and %d screenshots", len(scenarios), len(screenshots)))

	var response string

	// Prefer multimodal with screenshots for flow generation too
	if len(screenshots) > 0 {
		if imgAnalyzer, ok := a.Client.(ImageAnalyzer); ok {
			progress("flows_calling", "Sending to AI for flow generation (this may take 30-60s)...")
			var err error
			response, err = imgAnalyzer.AnalyzeWithImages(AnalysisSystemPrompt, prompt, screenshots)
			if err != nil {
				return nil, fmt.Errorf("AI multimodal flow generation failed: %w", err)
			}
		}
	}

	// Fallback to text-only
	if response == "" {
		progress("flows_calling", "Sending to AI for flow generation (this may take 30-60s)...")
		var err error
		response, err = a.Client.Generate(prompt, map[string]interface{}{
			"analysisJSON": string(analysisJSON),
		})
		if err != nil {
			return nil, fmt.Errorf("flow generation failed: %w", err)
		}
	}

	progress("flows_parsing", fmt.Sprintf("Parsing AI response (%d chars)...", len(response)))

	// Try parsing as structured JSON first (preferred)
	flows := parseFlowsJSON(response)
	if len(flows) > 0 {
		progress("flows_validating", fmt.Sprintf("Validated %d flows from structured JSON", len(flows)))
		return flows, nil
	}

	// Fallback to YAML parsing for backward compatibility
	progress("flows_parsing", "Trying YAML fallback parser...")
	yamlFlows := a.parseFlowsFromResponse(response, scenarios)
	progress("flows_validating", fmt.Sprintf("Parsed %d flows via YAML fallback", len(yamlFlows)))
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

// WriteFlowsToFiles writes Maestro flows to YAML files.
// The "setup" flow (if present) is sorted first to become 00-setup.yaml,
// matching the runFlow references in branching test flows.
func WriteFlowsToFiles(flows []*MaestroFlow, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Sort: setup flow first, others after
	sort.SliceStable(flows, func(i, j int) bool {
		iSetup := strings.EqualFold(flows[i].Name, "setup")
		jSetup := strings.EqualFold(flows[j].Name, "setup")
		if iSetup != jSetup {
			return iSetup
		}
		return false
	})

	for i, flow := range flows {
		filename := fmt.Sprintf("%02d-%s.yaml", i, util.SanitizeFilename(flow.Name))
		filepath := fmt.Sprintf("%s/%s", outputDir, filename)

		// Convert flow to YAML and normalize to fix common AI generation mistakes
		yamlContent := normalizeFlowYAML(flowToYAML(flow))

		if err := os.WriteFile(filepath, []byte(yamlContent), 0644); err != nil {
			return fmt.Errorf("failed to write flow file: %w", err)
		}
	}

	return nil
}

// --- CLI-side YAML normalization (mirrors web/backend/executor.go) ---

// openLinkObjRegexCLI matches multi-line openLink blocks: openLink:\n  url: "..."
var openLinkObjRegexCLI = regexp.MustCompile(`(?m)^(\s*- openLink):\s*\n\s+url:\s*"?([^"\n]+)"?\s*$`)

// extendedWaitTimeoutOnlyRegexCLI matches extendedWaitUntil blocks with only a timeout (invalid).
var extendedWaitTimeoutOnlyRegexCLI = regexp.MustCompile(`(?m)^[ \t]*- extendedWaitUntil:\s*\n[ \t]+timeout:\s*\d+\s*$`)

// selectorVisibleRegexCLI matches any command followed by visible: on the next indented line.
// The replacement function skips extendedWaitUntil (which legitimately uses visible:).
// Handles both "- cmd:\n  visible:" and "- cmd: value\n  visible:" patterns.
var selectorVisibleRegexCLI = regexp.MustCompile(`(?m)^(\s*- \w+):.*\n\s+visible:\s*"?([^"\n]+)"?\s*$`)

// selectorNotVisibleRegexCLI — same but for notVisible:
var selectorNotVisibleRegexCLI = regexp.MustCompile(`(?m)^(\s*- \w+):.*\n\s+notVisible:\s*"?([^"\n]+)"?\s*$`)

// bareVisibleRegexCLI matches a bare "visible:" line NOT preceded by extendedWaitUntil or tapOn,
// wrapping it in an extendedWaitUntil block.
var bareVisibleRegexCLI = regexp.MustCompile(`(?m)^(\s*)- visible:\s*"?([^"\n]+)"?\s*$`)

// maestroCommandAliasesCLI maps invalid/old command names to correct Maestro names.
var maestroCommandAliasesCLI = map[string]string{
	"waitFor":     "extendedWaitUntil",
	"screenshot":  "takeScreenshot",
	"openBrowser": "openLink",
}

// stripInvalidVisibleLinesCLI removes blank lines from the commands section and
// strips visible:/notVisible: lines from non-extendedWaitUntil command blocks.
// This catches AI patterns where blank lines between mapping fields break YAML
// parsing and where visible: is sprinkled after arbitrary commands.
func stripInvalidVisibleLinesCLI(content string) string {
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

// normalizeFlowYAML applies regex-based safety-net fixes to generated YAML.
// This catches issues that the structured commandToYAML serialization may miss.
func normalizeFlowYAML(content string) string {
	// First pass: strip blank lines and invalid visible/notVisible lines
	result := stripInvalidVisibleLinesCLI(content)
	result = openLinkObjRegexCLI.ReplaceAllString(result, `$1: "$2"`)
	skipExtended := func(match string) string {
		if strings.Contains(match, "extendedWaitUntil") {
			return match
		}
		return selectorVisibleRegexCLI.ReplaceAllString(match, `$1: "$2"`)
	}
	result = selectorVisibleRegexCLI.ReplaceAllStringFunc(result, skipExtended)
	skipExtendedNV := func(match string) string {
		if strings.Contains(match, "extendedWaitUntil") {
			return match
		}
		return selectorNotVisibleRegexCLI.ReplaceAllString(match, `$1: "$2"`)
	}
	result = selectorNotVisibleRegexCLI.ReplaceAllStringFunc(result, skipExtendedNV)
	for old, correct := range maestroCommandAliasesCLI {
		result = strings.ReplaceAll(result, "- "+old+":", "- "+correct+":")
	}
	result = extendedWaitTimeoutOnlyRegexCLI.ReplaceAllString(result, "")
	// Wrap bare "visible:" lines in extendedWaitUntil blocks
	result = bareVisibleRegexCLI.ReplaceAllString(result, "${1}- extendedWaitUntil:\n${1}    visible: \"$2\"")
	return result
}

// flowToYAML converts a MaestroFlow to YAML string
func flowToYAML(flow *MaestroFlow) string {
	var sb strings.Builder

	// Auto-set appId for web flows that use openLink or runFlow (which delegates to a web setup flow)
	if flow.AppId == "" {
		for _, cmd := range flow.Commands {
			if _, ok := cmd["openLink"]; ok {
				flow.AppId = "com.android.chrome"
				break
			}
			if _, ok := cmd["runFlow"]; ok {
				flow.AppId = "com.android.chrome"
				break
			}
		}
	}

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

	// Write commands — serialize using yaml.Marshal for correct escaping
	for _, cmd := range flow.Commands {
		// Extract comment before fixing
		if comment, ok := cmd["comment"].(string); ok && comment != "" {
			sb.WriteString(fmt.Sprintf("# %s\n", strings.ReplaceAll(comment, "\n", " ")))
		}
		// Split out spurious visible/notVisible into separate extendedWaitUntil commands
		for _, splitCmd := range splitVisibleFromCommand(cmd) {
			if fixed := fixCommandData(splitCmd, maestroCommandAliasesCLI); fixed != nil {
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

	return sb.String()
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
