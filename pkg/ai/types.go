package ai

import "time"

// AnalysisResult represents the result of analyzing a game
type AnalysisResult struct {
	GameInfo    GameInfo     `json:"gameInfo,omitempty"`
	Mechanics   []Mechanic   `json:"mechanics,omitempty"`
	UIElements  []UIElement  `json:"uiElements,omitempty"`
	UserFlows   []UserFlow   `json:"userFlows,omitempty"`
	EdgeCases   []EdgeCase   `json:"edgeCases,omitempty"`
	RawResponse string       `json:"rawResponse,omitempty"`
}

// ComprehensiveAnalysisResult combines game analysis with test scenarios in a
// single AI response. This avoids the lossy context degradation that occurs
// when analysis and scenario generation are separate calls.
type ComprehensiveAnalysisResult struct {
	GameInfo   GameInfo       `json:"gameInfo"`
	Mechanics  []Mechanic     `json:"mechanics"`
	UIElements []UIElement    `json:"uiElements"`
	UserFlows  []UserFlow     `json:"userFlows"`
	EdgeCases  []EdgeCase     `json:"edgeCases"`
	Scenarios  []TestScenario `json:"scenarios"`
}

// ToAnalysisResult converts a ComprehensiveAnalysisResult to the legacy AnalysisResult
// for backward compatibility with callers that expect the old type.
func (c *ComprehensiveAnalysisResult) ToAnalysisResult() *AnalysisResult {
	return &AnalysisResult{
		GameInfo:   c.GameInfo,
		Mechanics:  c.Mechanics,
		UIElements: c.UIElements,
		UserFlows:  c.UserFlows,
		EdgeCases:  c.EdgeCases,
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
	MaxSteps      int
	StepTimeout   time.Duration
	TotalTimeout  time.Duration
	UserMessages  <-chan string // Optional channel for user hints injected during exploration
	ScreenshotDir string       // Optional directory to write screenshots for live streaming
}

// DefaultAgentConfig returns sensible defaults for agent exploration.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		MaxSteps:     20,
		StepTimeout:  30 * time.Second,
		TotalTimeout: 5 * time.Minute,
	}
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

RULES:
1. Take a screenshot after each significant interaction to observe the result.
2. Use get_page_info to understand the page structure when screenshots are ambiguous.
3. Use console_logs when something seems wrong — errors often explain broken states.
4. Be systematic: don't click the same thing twice unless testing repeated interactions.
5. When you have thoroughly explored the game (usually 10-20 steps), output EXPLORATION_COMPLETE on its own line to signal you are done.
6. Focus on discovering testable behaviors through active interaction, not passive observation.

COORDINATE SYSTEM: The viewport is 1920x1080. Use absolute pixel coordinates for click and type_text tools.`

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

SCENARIO GENERATION INSTRUCTIONS:
6. Generate 3-6 test scenarios covering: happy path (main user flow), edge cases (boundary conditions), and failure scenarios (timeouts, disconnects).
7. Each scenario must have concrete, actionable steps with specific coordinates or selectors from the screenshots.
8. Include at least one happy-path scenario that exercises the core game loop end-to-end.

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
- openBrowser: {url: "..."} — navigate to the game URL
- waitFor: {visible: "text", timeout: 5000} — wait for element
- tapOn: "text" — tap on visible text element
- tapOn: {point: "50%,80%"} — tap at percentage coordinates on screen
- inputText: "..." — type text into focused field
- assertVisible: "text" — assert text is visible
- screenshot: "name" — capture screenshot

IMPORTANT RULES:
- For canvas games ({{framework}}), use coordinate-based tapOn with percentage points
- Always start with openBrowser and a waitFor to ensure the game loads
- Add screenshot commands after key interactions to capture state
- Use percentage-based coordinates that match what you see in the screenshots
- Generate 2-5 flows covering the most important scenarios

Respond with a JSON array of flows:
[
  {
    "name": "Flow name",
    "tags": ["smoke", "happy-path"],
    "commands": [
      {"openBrowser": {"url": "{{url}}"}},
      {"waitFor": {"visible": "text", "timeout": 5000}},
      {"tapOn": {"point": "50%,50%"}},
      {"screenshot": "after-tap"},
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
