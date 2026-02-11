package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AnalysisResult represents the result of analyzing a game
type AnalysisResult struct {
	GameInfo     GameInfo            `json:"gameInfo,omitempty"`
	Mechanics    []Mechanic          `json:"mechanics,omitempty"`
	UIElements   []UIElement         `json:"uiElements,omitempty"`
	UserFlows    []UserFlow          `json:"userFlows,omitempty"`
	EdgeCases    []EdgeCase          `json:"edgeCases,omitempty"`
	UIUXAnalysis []UIUXFinding       `json:"uiuxAnalysis,omitempty"`
	WordingCheck []WordingFinding    `json:"wordingCheck,omitempty"`
	GameDesign   []GameDesignFinding `json:"gameDesign,omitempty"`
	RawResponse  string              `json:"rawResponse,omitempty"`
}

// ComprehensiveAnalysisResult combines game analysis with test scenarios in a
// single AI response. This avoids the lossy context degradation that occurs
// when analysis and scenario generation are separate calls.
type ComprehensiveAnalysisResult struct {
	GameInfo     GameInfo            `json:"gameInfo"`
	Mechanics    []Mechanic          `json:"mechanics"`
	UIElements   []UIElement         `json:"uiElements"`
	UserFlows    []UserFlow          `json:"userFlows"`
	EdgeCases    []EdgeCase          `json:"edgeCases"`
	Scenarios    []TestScenario      `json:"scenarios"`
	UIUXAnalysis []UIUXFinding       `json:"uiuxAnalysis,omitempty"`
	WordingCheck []WordingFinding    `json:"wordingCheck,omitempty"`
	GameDesign   []GameDesignFinding `json:"gameDesign,omitempty"`
}

// ToAnalysisResult converts a ComprehensiveAnalysisResult to the legacy AnalysisResult
// for backward compatibility with callers that expect the old type.
func (c *ComprehensiveAnalysisResult) ToAnalysisResult() *AnalysisResult {
	return &AnalysisResult{
		GameInfo:     c.GameInfo,
		Mechanics:    c.Mechanics,
		UIElements:   c.UIElements,
		UserFlows:    c.UserFlows,
		EdgeCases:    c.EdgeCases,
		UIUXAnalysis: c.UIUXAnalysis,
		WordingCheck: c.WordingCheck,
		GameDesign:   c.GameDesign,
	}
}

// GameInfo represents basic game information
type GameInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Genre       string   `json:"genre"`
	Technology  string   `json:"technology"` // Phaser 4, etc.
	Features    []string `json:"features"`
}

// Mechanic represents a game mechanic that needs testing
type Mechanic struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`     // User actions required
	Expected    string   `json:"expected"`    // Expected outcome
	Priority    string   `json:"priority"`    // high, medium, low
}

// UIElement represents a UI element to interact with
type UIElement struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`     // button, input, canvas, etc.
	Selector string            `json:"selector"` // Text or coordinate
	Location map[string]string `json:"location"` // x, y, or percentage
}

// UserFlow represents a complete user journey
type UserFlow struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	Expected    string   `json:"expected"`
	Priority    string   `json:"priority"`
}

// EdgeCase represents an edge case or failure scenario
type EdgeCase struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Scenario    string `json:"scenario"`
	Expected    string `json:"expected"`
}

// UIUXFinding represents a single UI/UX observation from the analysis.
type UIUXFinding struct {
	Category    string `json:"category"`    // alignment, spacing, color, typography, responsive, hierarchy, accessibility, animation
	Description string `json:"description"`
	Severity    string `json:"severity"`    // critical, major, minor, suggestion, positive
	Location    string `json:"location"`    // Where in the UI this was observed
	Suggestion  string `json:"suggestion"`  // Recommended fix
}

// WordingFinding represents a single wording/translation issue.
type WordingFinding struct {
	Category    string `json:"category"`    // grammar, spelling, consistency, tone, truncation, placeholder, translation, overflow
	Text        string `json:"text"`        // The problematic text
	Description string `json:"description"`
	Severity    string `json:"severity"`    // critical, major, minor, suggestion, positive
	Location    string `json:"location"`    // Where in the UI this text appears
	Suggestion  string `json:"suggestion"`  // Corrected text or fix
}

// GameDesignFinding represents a single game design observation.
type GameDesignFinding struct {
	Category    string `json:"category"`    // rewards, balance, progression, psychology, difficulty, monetization, tutorial, feedback
	Description string `json:"description"`
	Severity    string `json:"severity"`    // critical, major, minor, positive
	Impact      string `json:"impact"`      // How this affects the player experience
	Suggestion  string `json:"suggestion"`  // Recommended improvement
}

// TestScenario represents a test scenario generated from analysis
type TestScenario struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"` // happy-path, edge-case, failure
	Steps       []Step   `json:"steps"`
	Priority    string   `json:"priority"`
	Tags        []string `json:"tags"`
}

// Step represents a single test step
type Step struct {
	Action      string            `json:"action"`
	Target      string            `json:"target"`
	Value       string            `json:"value,omitempty"`
	Expected    string            `json:"expected,omitempty"`
	Screenshot  bool              `json:"screenshot,omitempty"`
	Coordinates map[string]string `json:"coordinates,omitempty"` // For canvas clicks
}

// MaestroFlow represents a complete Maestro YAML flow
type MaestroFlow struct {
	Name     string                   `json:"name"`
	AppId    string                   `json:"appId,omitempty"`
	URL      string                   `json:"url,omitempty"`
	Tags     []string                 `json:"tags,omitempty"`
	Commands []map[string]interface{} `json:"commands"`
}

// PromptTemplate represents a reusable prompt template
type PromptTemplate struct {
	Name        string
	Description string
	Template    string
	Variables   []string
}

// BrowserPage is an interface for interacting with a live browser page.
// The ai package defines this interface so the agent doesn't import go-rod directly.
type BrowserPage interface {
	CaptureScreenshot() (b64 string, err error)
	Click(x, y int) error
	TypeText(text string) error
	Scroll(dx, dy float64) error
	EvalJS(expr string) (string, error)
	WaitVisible(selector string, timeout time.Duration) error
	GetPageInfo() (title, url, visibleText string, err error)
	GetConsoleLogs() ([]string, error)
	Navigate(url string) error
}

// AnalysisModules controls which optional analysis sections are enabled.
type AnalysisModules struct {
	UIUX       bool
	Wording    bool
	GameDesign bool
	TestFlows  bool
}

// DefaultAnalysisModules returns modules with everything enabled.
func DefaultAnalysisModules() AnalysisModules {
	return AnalysisModules{UIUX: true, Wording: true, GameDesign: true, TestFlows: true}
}

// BuildAnalysisPrompt constructs the comprehensive analysis prompt with only
// the enabled module sections included. This reduces token usage when modules
// are disabled.
func BuildAnalysisPrompt(modules AnalysisModules) string {
	tmpl := `Analyze this web-based game for automated QA testing. You are provided with screenshots of the game in different states and page metadata.

Game URL: {{url}}
URL Hints: {{urlHints}}

Page metadata (auto-detected):
{{pageMeta}}

{{screenshotSection}}

ANALYSIS INSTRUCTIONS:
1. URL parameters often reveal critical game info (e.g., game_type=SLOTS, mode=demo). Use the domain and path to infer the game studio/platform.
2. If page metadata is minimal (just a JS loader), this is a JS-rendered SPA. Focus on what the screenshots and URL parameters tell you.
3. For canvas-based games (Phaser, PIXI, etc.): game interactions are coordinate-based taps on the canvas. HTML overlays use text selectors.
4. Describe ONLY mechanics, UI elements, and flows that are evidenced by the screenshots or metadata. Do not guess.
5. For UI elements, provide percentage-based coordinates from the screenshots (e.g., "50%,80%").
`
	if modules.UIUX {
		tmpl += `
UI/UX ANALYSIS:
6. Evaluate visual design — alignments, layout consistency, color palette harmony, spacing, typography, visual hierarchy, accessibility (contrast ratios), animation quality. Report issues and strengths. Include 'positive' severity for things done well (good alignment, strong visual hierarchy, etc.).
`
	}
	if modules.Wording {
		tmpl += `
WORDING/TRANSLATION CHECK:
7. Examine all visible text for grammar, spelling, inconsistent terminology, tone, truncated text, placeholder text (e.g., "Lorem ipsum"), translation completeness, text overflow. Include 'positive' severity for well-written text, and 'suggestion' for non-bug improvements.
`
	}
	if modules.GameDesign {
		tmpl += `
GAME DESIGN ANALYSIS:
8. Analyze game design — reward systems, balance, progression, player engagement, difficulty curve, monetization fairness, tutorial/onboarding quality, feedback systems.
`
	}

	tmpl += `
SCENARIO GENERATION INSTRUCTIONS:
9. Generate 3-6 test scenarios covering: happy path (main user flow), edge cases (boundary conditions), and failure scenarios (timeouts, disconnects).
10. Each scenario must have concrete, actionable steps with specific coordinates or selectors from the screenshots.
11. Include at least one happy-path scenario that exercises the core game loop end-to-end.

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
  ]`

	if modules.UIUX {
		tmpl += `,
  "uiuxAnalysis": [
    {
      "category": "alignment|spacing|color|typography|responsive|hierarchy|accessibility|animation",
      "description": "...",
      "severity": "critical|major|minor|suggestion|positive",
      "location": "where in the UI",
      "suggestion": "recommended fix"
    }
  ]`
	}
	if modules.Wording {
		tmpl += `,
  "wordingCheck": [
    {
      "category": "grammar|spelling|consistency|tone|truncation|placeholder|translation|overflow",
      "text": "the problematic text",
      "description": "what is wrong",
      "severity": "critical|major|minor|suggestion|positive",
      "location": "where this text appears",
      "suggestion": "corrected text"
    }
  ]`
	}
	if modules.GameDesign {
		tmpl += `,
  "gameDesign": [
    {
      "category": "rewards|balance|progression|psychology|difficulty|monetization|tutorial|feedback",
      "description": "...",
      "severity": "critical|major|minor|positive",
      "impact": "how this affects player experience",
      "suggestion": "recommended improvement"
    }
  ]`
	}

	tmpl += `
}`

	return tmpl
}

// BuildSynthesisPrompt constructs the agent synthesis prompt with only
// the enabled module sections included.
func BuildSynthesisPrompt(modules AnalysisModules) string {
	prompt := `Based on your exploration of this game, provide a comprehensive QA analysis as a single JSON object.

You interacted with the game and observed its behavior through screenshots. Now produce a structured analysis based on ONLY what you actually observed during exploration.
`

	if modules.UIUX || modules.Wording || modules.GameDesign {
		prompt += "\nIn addition to functional QA, also analyze:\n"
		if modules.UIUX {
			prompt += "- UI/UX quality: alignments, spacing, color harmony, typography, visual hierarchy, accessibility, animations\n"
		}
		if modules.Wording {
			prompt += "- Wording/translation: grammar, spelling, consistency, tone, truncation, placeholder text, text overflow\n"
		}
		if modules.GameDesign {
			prompt += "- Game design: rewards, balance, progression, engagement, difficulty, monetization, tutorial, feedback\n"
		}
	}

	prompt += `
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
  ]`

	if modules.UIUX {
		prompt += `,
  "uiuxAnalysis": [
    {
      "category": "alignment|spacing|color|typography|responsive|hierarchy|accessibility|animation",
      "description": "...",
      "severity": "critical|major|minor|suggestion|positive",
      "location": "where in the UI",
      "suggestion": "recommended fix"
    }
  ]`
	}
	if modules.Wording {
		prompt += `,
  "wordingCheck": [
    {
      "category": "grammar|spelling|consistency|tone|truncation|placeholder|translation|overflow",
      "text": "the problematic text",
      "description": "what is wrong",
      "severity": "critical|major|minor|suggestion|positive",
      "location": "where this text appears",
      "suggestion": "corrected text"
    }
  ]`
	}
	if modules.GameDesign {
		prompt += `,
  "gameDesign": [
    {
      "category": "rewards|balance|progression|psychology|difficulty|monetization|tutorial|feedback",
      "description": "...",
      "severity": "critical|major|minor|positive",
      "impact": "how this affects player experience",
      "suggestion": "recommended improvement"
    }
  ]`
	}

	prompt += `
}

IMPORTANT: Base your analysis on what you actually observed during exploration. Include specific coordinates you discovered. Respond with valid JSON only.`

	return prompt
}

// AgentStep records a single step in the agent exploration loop.
type AgentStep struct {
	StepNumber    int    `json:"stepNumber"`
	ToolName      string `json:"toolName"`
	Input         string `json:"input"`
	Result        string `json:"result"`
	ScreenshotB64 string `json:"screenshotB64,omitempty"`
	DurationMs    int    `json:"durationMs"`
	Error         string `json:"error,omitempty"`
}

// AgentConfig controls the agent exploration loop.
type AgentConfig struct {
	MaxSteps            int
	StepTimeout         time.Duration
	TotalTimeout        time.Duration
	UserMessages        <-chan string // Optional channel for user hints injected during exploration
	ScreenshotDir       string       // Optional directory to write screenshots for live streaming
	SynthesisMaxTokens  int          // Override maxTokens for synthesis call (0 = use client default)
	AdaptiveExploration bool         // Enable request_more_steps tool for dynamic step extension
	MaxTotalSteps       int          // Hard cap on total steps after adaptive extensions
	AdaptiveTimeout     bool         // Enable request_more_time tool for dynamic timeout extension
	MaxTotalTimeout     time.Duration // Hard cap on total exploration time after extensions
	ViewportWidth       int          // Browser viewport width (for tool descriptions)
	ViewportHeight      int          // Browser viewport height (for tool descriptions)
}

// CheckpointData wraps the state written to checkpoint files after each pipeline step.
type CheckpointData struct {
	Step      string          `json:"step"`
	AgentMode bool            `json:"agentMode"`
	PageMeta  json.RawMessage `json:"pageMeta,omitempty"`
	Analysis  json.RawMessage `json:"analysis,omitempty"` // ComprehensiveAnalysisResult
	Modules   AnalysisModules `json:"modules"`
	Timestamp string          `json:"timestamp"`
}

// WriteCheckpoint serialises checkpoint data to a file in dir.
func WriteCheckpoint(dir string, data CheckpointData) error {
	data.Timestamp = time.Now().Format(time.RFC3339)
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, fmt.Sprintf("checkpoint_%s.json", data.Step)), b, 0644)
}

// ReadLatestCheckpoint returns the most-advanced checkpoint found in dir, or nil.
func ReadLatestCheckpoint(dir string) (*CheckpointData, error) {
	for _, step := range []string{"synthesized", "analyzed", "scouted"} {
		data, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("checkpoint_%s.json", step)))
		if err != nil {
			continue
		}
		var cp CheckpointData
		if json.Unmarshal(data, &cp) == nil {
			return &cp, nil
		}
	}
	return nil, nil
}

// ReadResumeData reads a checkpoint from the given path.
func ReadResumeData(path string) (*CheckpointData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cp CheckpointData
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

// DefaultAgentConfig returns sensible defaults for agent exploration.
func DefaultAgentConfig() AgentConfig {
	steps := 20
	return AgentConfig{
		MaxSteps:     steps,
		StepTimeout:  30 * time.Second,
		TotalTimeout: agentTotalTimeout(steps),
	}
}

// agentTotalTimeout scales exploration timeout with step count:
// steps × 60s avg + 7min buffer, clamped to [5min, 30min].
func agentTotalTimeout(steps int) time.Duration {
	t := time.Duration(steps)*60*time.Second + 7*time.Minute
	if t < 5*time.Minute {
		t = 5 * time.Minute
	}
	if t > 30*time.Minute {
		t = 30 * time.Minute
	}
	return t
}

// AgentSystemPrompt is the system prompt for the agent exploration loop.
const AgentSystemPrompt = `You are an expert QA engineer exploring a web-based game through a browser. You have access to browser tools that let you interact with the page.

YOUR GOAL: Thoroughly explore the game to understand its mechanics, UI elements, state transitions, and interactive features. You will use this information to generate comprehensive QA test scenarios.

INITIAL LOAD CHECK (do this first):
1. Examine the initial screenshot for loading screens, error dialogs, or blank pages.
2. Use console_logs to check for JavaScript errors or warnings.
3. If the page shows a loading spinner, error message, or the game hasn't rendered:
   a. Check console_logs for errors (e.g., failed asset loads, JS exceptions).
   b. Wait a few seconds and take another screenshot.
   c. If still broken after 2 attempts, use navigate to reload the game URL.
   d. Do NOT spend more than 3 steps diagnosing a broken loading screen — reload and move on.

EXPLORATION STRATEGY:
1. Once the game is loaded, identify clickable elements, buttons, and interactive regions.
2. Actively interact with the game: click buttons, spin reels, place bets, type text — don't just observe.
3. After each interaction, take a screenshot to observe the result.
4. Try to trigger different game states: loading, playing, paused, game over, bonus rounds, etc.
5. For canvas games, try clicking different regions (center, corners, common button positions like bottom-center for spin).
6. Note any animations, transitions, error states, or popups you encounter.

LOADING FAILURE RECOVERY:
- Signs of loading failure: blank canvas, "loading" text stuck for multiple screenshots, error dialogs (even hidden ones), console errors about failed network requests.
- If you see an error dialog, try clicking its OK/Close button or dismissing it.
- If the game is stuck loading, use navigate with the game URL to reload.
- After reloading, wait 3-5 seconds then take a screenshot to check the new state.

SESSION-GATED GAMES:
- Many games require valid session tokens passed via URL parameters (sessionToken, game_instance_token).
- If the initial message indicates tokens are expired, the game WILL NOT load. Do not waste steps trying to interact with a loading screen.
- Instead: take 1 screenshot to confirm the loading/error state, check console_logs for auth errors, then output EXPLORATION_COMPLETE.
- In your synthesis, note that the game could not be analyzed due to expired session tokens and that fresh tokens are needed.

RULES:
1. The click, type_text, scroll, and navigate tools automatically return screenshots. You only need the screenshot tool for observing the page without interacting.
2. Use get_page_info to understand the page structure when screenshots are ambiguous.
3. Use console_logs when something seems wrong — errors often explain broken states.
4. Be systematic: don't click the same thing twice unless testing repeated interactions.
5. When you have thoroughly explored the game (usually 10-20 steps), output EXPLORATION_COMPLETE on its own line to signal you are done.
6. Focus on discovering testable behaviors through active interaction, not passive observation.`

// AdaptiveExplorationPromptSuffix returns the system prompt addition for adaptive exploration mode.
func AdaptiveExplorationPromptSuffix(maxTotalSteps int) string {
	return fmt.Sprintf(`

ADAPTIVE EXPLORATION:
You MUST call request_more_steps when BOTH conditions are true:
1. You have used 70%% or more of your current step budget
2. You have NOT yet explored all major interactive elements (buttons, menus, game states, bonus features)

Assessment checklist — run through this every 3 steps:
- Have I explored all visible buttons and menus?
- Have I triggered all reachable game states (play, pause, settings, bonus, game over)?
- Have I tested edge cases (rapid clicks, boundary values, unusual inputs)?
- If ANY answer is "no" and I'm past 70%% of my steps, call request_more_steps immediately.

When calling request_more_steps, request at least 5-10 additional steps. Your exploration can extend up to %d total steps.

IMPORTANT: Do NOT output EXPLORATION_COMPLETE until you have explored at least 80%% of the game's interactive surface. If in doubt, request more steps rather than stopping early.`, maxTotalSteps)
}

// DynamicTimeoutPromptSuffix returns the system prompt addition for adaptive timeout mode.
func DynamicTimeoutPromptSuffix(maxMinutes int) string {
	return fmt.Sprintf(`

DYNAMIC TIMEOUT:
You MUST call request_more_time if you are less than 50%% through your exploration goals and more than 50%% through your time budget. The system will periodically inject [SYSTEM STATUS] messages showing your elapsed and remaining time.

When you see remaining time is under 3 minutes and you still have significant areas to explore, call request_more_time immediately with at least 5 additional minutes. Your exploration can extend up to %d minutes total.

Do NOT wait until you're about to be cut off — request proactively.`, maxMinutes)
}

// BuildAgentSystemPrompt constructs the agent system prompt, optionally appending
// adaptive exploration instructions when the feature is enabled.
func BuildAgentSystemPrompt(cfg AgentConfig) string {
	prompt := AgentSystemPrompt
	// Add dynamic coordinate system based on actual viewport
	w := cfg.ViewportWidth
	h := cfg.ViewportHeight
	if w <= 0 {
		w = 1280
	}
	if h <= 0 {
		h = 720
	}
	prompt += fmt.Sprintf("\n\nCOORDINATE SYSTEM: The viewport is %dx%d. Use absolute pixel coordinates for click and type_text tools.", w, h)
	if cfg.AdaptiveExploration && cfg.MaxTotalSteps > 0 {
		prompt += AdaptiveExplorationPromptSuffix(cfg.MaxTotalSteps)
	}
	if cfg.AdaptiveTimeout && cfg.MaxTotalTimeout > 0 {
		prompt += DynamicTimeoutPromptSuffix(int(cfg.MaxTotalTimeout.Minutes()))
	}
	return prompt
}

// AnalysisSystemPrompt is the system prompt used for all analysis AI calls.
// It establishes the AI's role and enforces grounding constraints.
const AnalysisSystemPrompt = `You are an expert QA engineer specializing in automated testing of web-based games.

CRITICAL RULES:
1. ONLY report what you can actually see in the provided screenshots or infer from the page metadata. Do NOT invent or hallucinate game mechanics, UI elements, or features that are not evidenced by the screenshots or metadata.
2. For canvas-based games, all game interactions use coordinate-based taps. HTML overlay elements (buttons, dialogs) use text-based selectors.
3. When multiple screenshots are provided, they show the game in different states (e.g., loading, after clicking canvas, after clicking a button). Use ALL screenshots to understand the game's state transitions.
4. Be specific about coordinates. Reference UI element positions using percentage-based coordinates (e.g., "50%,80%") based on what you see in the screenshots.
5. Always respond with valid JSON only — no markdown, no code fences, no explanatory text outside the JSON.`

// Common prompt templates
var (
	// GameAnalysisPrompt is the legacy prompt for spec-based analysis (kept for backward compat).
	GameAnalysisPrompt = PromptTemplate{
		Name:        "game-analysis",
		Description: "Analyze a game from specification and URL",
		Template: `Analyze this game and provide a comprehensive breakdown.

Game Specification:
{{spec}}

Game URL: {{url}}

Provide your analysis in the following JSON format:
{
  "gameInfo": {
    "name": "...",
    "description": "...",
    "genre": "...",
    "technology": "Phaser 4",
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
      "selector": "text or coordinate",
      "location": {"x": "...", "y": "..."}
    }
  ],
  "userFlows": [
    {
      "name": "...",
      "description": "...",
      "steps": ["...", "..."],
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
  ]
}`,
		Variables: []string{"spec", "url"},
	}

	// ComprehensiveAnalysisPrompt combines analysis + scenario generation into a single call.
	// This replaces the old URLAnalysisPrompt + ScenarioGenerationPrompt two-call pipeline.
	ComprehensiveAnalysisPrompt = PromptTemplate{
		Name:        "comprehensive-analysis",
		Description: "Analyze a game and generate test scenarios in one call",
		Template: `Analyze this web-based game for automated QA testing. You are provided with screenshots of the game in different states and page metadata.

Game URL: {{url}}
URL Hints: {{urlHints}}

Page metadata (auto-detected):
{{pageMeta}}

{{screenshotSection}}

ANALYSIS INSTRUCTIONS:
1. URL parameters often reveal critical game info (e.g., game_type=SLOTS, mode=demo). Use the domain and path to infer the game studio/platform.
2. If page metadata is minimal (just a JS loader), this is a JS-rendered SPA. Focus on what the screenshots and URL parameters tell you.
3. For canvas-based games (Phaser, PIXI, etc.): game interactions are coordinate-based taps on the canvas. HTML overlays use text selectors.
4. Describe ONLY mechanics, UI elements, and flows that are evidenced by the screenshots or metadata. Do not guess.
5. For UI elements, provide percentage-based coordinates from the screenshots (e.g., "50%,80%").

UI/UX ANALYSIS:
6. Evaluate visual design — alignments, layout consistency, color palette harmony, spacing, typography, visual hierarchy, accessibility (contrast ratios), animation quality. Report issues and strengths. Include 'positive' severity for things done well (good alignment, strong visual hierarchy, etc.).
7. WORDING/TRANSLATION CHECK: Examine all visible text for grammar, spelling, inconsistent terminology, tone, truncated text, placeholder text (e.g., "Lorem ipsum"), translation completeness, text overflow. Include 'positive' severity for well-written text, and 'suggestion' for non-bug improvements.
8. GAME DESIGN ANALYSIS: Analyze game design — reward systems, balance, progression, player engagement, difficulty curve, monetization fairness, tutorial/onboarding quality, feedback systems.

SCENARIO GENERATION INSTRUCTIONS:
9. Generate 3-6 test scenarios covering: happy path (main user flow), edge cases (boundary conditions), and failure scenarios (timeouts, disconnects).
10. Each scenario must have concrete, actionable steps with specific coordinates or selectors from the screenshots.
11. Include at least one happy-path scenario that exercises the core game loop end-to-end.

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
  ],
  "uiuxAnalysis": [
    {
      "category": "alignment|spacing|color|typography|responsive|hierarchy|accessibility|animation",
      "description": "...",
      "severity": "critical|major|minor|suggestion|positive",
      "location": "where in the UI",
      "suggestion": "recommended fix"
    }
  ],
  "wordingCheck": [
    {
      "category": "grammar|spelling|consistency|tone|truncation|placeholder|translation|overflow",
      "text": "the problematic text",
      "description": "what is wrong",
      "severity": "critical|major|minor|suggestion|positive",
      "location": "where this text appears",
      "suggestion": "corrected text"
    }
  ],
  "gameDesign": [
    {
      "category": "rewards|balance|progression|psychology|difficulty|monetization|tutorial|feedback",
      "description": "...",
      "severity": "critical|major|minor|positive",
      "impact": "how this affects player experience",
      "suggestion": "recommended improvement"
    }
  ]
}`,
		Variables: []string{"url", "pageMeta", "urlHints", "screenshotSection"},
	}

	// FlowGenerationPrompt generates Maestro YAML flows from structured analysis+scenario JSON.
	// This version receives the full JSON analysis (not lossy text) and screenshots.
	FlowGenerationPrompt = PromptTemplate{
		Name:        "flow-generation",
		Description: "Generate Maestro flows from structured analysis JSON",
		Template: `Convert the following game analysis and test scenarios into Maestro test flows.

Game URL: {{url}}
Framework: {{framework}}

Full analysis (JSON):
{{analysisJSON}}

{{screenshotSection}}

Generate Maestro flows as a JSON array. Each flow should be a complete, runnable test.

Maestro command reference:
- openLink: "..." — open a URL (string value, NOT an object)
- extendedWaitUntil: {visible: "text", timeout: 5000} — wait for element
- tapOn: "text" — tap on visible text element
- tapOn: {point: "50%,80%"} — tap at percentage coordinates on screen
- inputText: "..." — type text into focused field
- assertVisible: "text" — assert text is visible
- takeScreenshot: "name" — capture screenshot

IMPORTANT RULES:
- For canvas games ({{framework}}), use coordinate-based tapOn with percentage points
- Always start with openLink and an extendedWaitUntil to ensure the game loads
- Add takeScreenshot commands after key interactions to capture state
- Use percentage-based coordinates that match what you see in the screenshots

FLOW COMPOSITION — SHARED SETUP:
- The FIRST flow MUST be a "setup" flow named exactly "setup". It contains the common steps that every test needs: opening the browser, waiting for the game to load, dismissing any splash screens, skipping tutorials, and reaching the main game state.
- All subsequent flows MUST start with: {"runFlow": "00-setup.yaml"} as their first command, then branch into their specific test scenario.
- Do NOT repeat setup steps in individual test flows — use runFlow instead.
- Generate 1 setup flow + 2-4 test flows that branch from it.

Respond with a JSON array of flows:
[
  {
    "name": "setup",
    "appId": "com.android.chrome",
    "tags": ["setup"],
    "commands": [
      {"openLink": "{{url}}"},
      {"extendedWaitUntil": {"visible": "Play", "timeout": 10000}},
      {"tapOn": "OK"},
      {"tapOn": "Play"},
      {"takeScreenshot": "game-ready"}
    ]
  },
  {
    "name": "basic-gameplay",
    "appId": "com.android.chrome",
    "tags": ["smoke", "happy-path"],
    "commands": [
      {"runFlow": "00-setup.yaml"},
      {"tapOn": {"point": "50%,80%"}},
      {"takeScreenshot": "after-action"},
      {"assertVisible": "expected text"}
    ]
  }
]`,
		Variables: []string{"url", "framework", "analysisJSON", "screenshotSection"},
	}

	// ScenarioGenerationPrompt is kept for backward compatibility but is no longer
	// used in the primary pipeline (scenarios are now part of ComprehensiveAnalysisPrompt).
	ScenarioGenerationPrompt = PromptTemplate{
		Name:        "scenario-generation",
		Description: "Generate test scenarios from game analysis (legacy)",
		Template: `Based on this game analysis, generate comprehensive test scenarios.

Analysis:
{{analysis}}

Generate test scenarios covering:
1. Happy path (main user flow)
2. Edge cases (boundary conditions, invalid inputs)
3. Failure scenarios (network errors, timeouts)

Respond with JSON array of scenarios:
[
  {
    "name": "...",
    "description": "...",
    "type": "happy-path|edge-case|failure",
    "steps": [
      {
        "action": "launch|click|input|wait|assert",
        "target": "...",
        "value": "...",
        "expected": "...",
        "coordinates": {"x": "...", "y": "..."}
      }
    ],
    "priority": "high|medium|low",
    "tags": ["smoke", "regression", etc.]
  }
]`,
		Variables: []string{"analysis"},
	}

	// URLAnalysisPrompt is kept for backward compatibility but is replaced by
	// ComprehensiveAnalysisPrompt in the primary pipeline.
	URLAnalysisPrompt = PromptTemplate{
		Name:        "url-analysis",
		Description: "Analyze a game from URL with auto-detected page metadata (legacy)",
		Template: `You are an expert QA engineer analyzing a web-based game for automated testing.

Game URL: {{url}}
URL Hints: {{urlHints}}

Page metadata (auto-detected):
{{pageMeta}}

{{screenshotSection}}

IMPORTANT INSTRUCTIONS:
1. The URL parameters often reveal critical game info. For example:
   - game_type=LOTTERY -> This is a lottery/scratch card game
   - game_type=SLOTS -> This is a slot machine game
   - mode=demo -> Running in demo/free-play mode
   - Use the domain and path to infer the game studio/platform

2. If the page metadata shows minimal content (e.g., just a JS loader),
   the game is a JS-rendered SPA. Focus on what the URL parameters and
   any screenshot tell you about the game type.

3. For canvas-based games (Phaser, PIXI, etc.):
   - All game interactions are coordinate-based taps on the canvas
   - HTML overlays (buttons, dialogs) use text-based selectors

4. Describe ONLY what you can see in the screenshots or infer from the metadata.

Respond with structured JSON matching this format:
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
      "selector": "text or coordinate",
      "location": {"x": "...", "y": "..."}
    }
  ],
  "userFlows": [
    {
      "name": "...",
      "description": "...",
      "steps": ["...", "..."],
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
  ]
}`,
		Variables: []string{"url", "pageMeta", "framework", "urlHints", "screenshotSection"},
	}
)
