package ai

// AnalysisResult represents the result of analyzing a game
type AnalysisResult struct {
	GameInfo    GameInfo     `json:"gameInfo,omitempty"`
	Mechanics   []Mechanic   `json:"mechanics,omitempty"`
	UIElements  []UIElement  `json:"uiElements,omitempty"`
	UserFlows   []UserFlow   `json:"userFlows,omitempty"`
	EdgeCases   []EdgeCase   `json:"edgeCases,omitempty"`
	RawResponse string       `json:"rawResponse,omitempty"`
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

// Common prompt templates
var (
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

	ScenarioGenerationPrompt = PromptTemplate{
		Name:        "scenario-generation",
		Description: "Generate test scenarios from game analysis",
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

	FlowGenerationPrompt = PromptTemplate{
		Name:        "flow-generation",
		Description: "Generate Maestro YAML flows from scenarios",
		Template: `Convert these test scenarios into Maestro YAML flows.

Scenarios:
{{scenarios}}

Generate Maestro flows using these commands:
- launchApp
- tapOn: "text" or tapOn: {point: "x,y"}
- inputText: "..."
- assertVisible: "..."
- waitFor: {visible: true, timeout: ms}
- captureScreenshot: "filename.png"

Important for Phaser 4 games:
- Use coordinate-based clicking for canvas: tapOn: {point: "50%,50%"}
- Wait for game to load before interactions
- Use assertVisible for HTML overlays, not canvas content

Respond with valid Maestro YAML for each flow.`,
		Variables: []string{"scenarios"},
	}

	URLAnalysisPrompt = PromptTemplate{
		Name:        "url-analysis",
		Description: "Analyze a game from URL with auto-detected page metadata",
		Template: `You are an expert QA engineer analyzing a web-based game for automated testing.

Game URL: {{url}}
URL Hints: {{urlHints}}

Page metadata (auto-detected):
{{pageMeta}}

{{screenshotSection}}

IMPORTANT INSTRUCTIONS:
1. The URL parameters often reveal critical game info. For example:
   - game_type=LOTTERY → This is a lottery/scratch card game
   - game_type=SLOTS → This is a slot machine game
   - mode=demo → Running in demo/free-play mode
   - Use the domain and path to infer the game studio/platform

2. If the page metadata shows minimal content (e.g., just a JS loader),
   the game is a JS-rendered SPA. Focus on what the URL parameters and
   any screenshot tell you about the game type.

3. For canvas-based games (Phaser, PIXI, etc.):
   - All game interactions are coordinate-based taps on the canvas
   - HTML overlays (buttons, dialogs) use text-based selectors
   - Common patterns: "Play" button, "Spin" button, bet controls,
     balance display, settings gear icon

4. Generate REALISTIC mechanics based on the game type. For a LOTTERY game:
   - Scratch/reveal mechanics, number selection, draw animations
   - Bet amount controls, auto-play, balance display
   - Win/loss states, prize tiers, bonus features

5. For a SLOTS game:
   - Spin mechanic, reel animations, payline display
   - Bet controls, auto-spin, turbo mode
   - Free spins, bonus rounds, scatter/wild symbols

6. If the framework is "phaser" or uses HTML5 canvas, focus on coordinate-based interactions.
   If the framework is "unity", note that WebGL interactions may need special handling.

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
