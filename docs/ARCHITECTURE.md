# Wizards QA - System Architecture

**Version:** 0.1.0 (Draft)  
**Author:** Lia 🌸  
**Date:** 2026-02-06  
**Status:** 🚧 Design Phase

## Executive Summary

Wizards QA is an AI-powered QA automation system designed to intelligently test Phaser 4 web games. The system accepts a game specification and live URL, uses AI to understand the game mechanics, generates comprehensive test flows in Maestro YAML format, and executes automated testing via Maestro CLI.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Wizards QA CLI                           │
│                      (Go + Cobra Framework)                      │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            │ Commands
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ wizards-qa   │    │ wizards-qa   │    │ wizards-qa   │
│   test       │    │  generate    │    │    run       │
│              │    │              │    │              │
│ Full E2E     │    │ Flow Only    │    │ Execute Only │
└──────┬───────┘    └──────┬───────┘    └──────┬───────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                           │
        ┌──────────────────┴──────────────────┐
        │                                     │
        ▼                                     ▼
┌──────────────────┐                 ┌──────────────────┐
│   AI Analysis    │                 │  Maestro Wrapper │
│     Engine       │                 │                  │
│                  │                 │  - Flow executor │
│ - Game analyzer  │                 │  - Result parser │
│ - Flow generator │                 │  - Screenshot    │
│ - Claude/Gemini  │                 │  - Reporting     │
└────────┬─────────┘                 └────────┬─────────┘
         │                                    │
         │ YAML Flows                         │
         └────────────────┬───────────────────┘
                          │
                          ▼
                ┌────────────────────┐
                │  Flow Repository   │
                │  (File System)     │
                │                    │
                │  flows/            │
                │  ├── game-1/       │
                │  ├── game-2/       │
                │  └── templates/    │
                └────────────────────┘
```

## Core Components

### 1. CLI Layer (Go + Cobra)

**Purpose:** User interface for all Wizards QA operations

**Commands:**

```bash
# Full end-to-end testing
wizards-qa test --game https://game.example.com --spec game-spec.md

# Generate flows only (no execution)
wizards-qa generate --game https://game.example.com --spec game-spec.md --output flows/

# Run existing flows
wizards-qa run --flows flows/my-game/ --report reports/

# Additional utilities
wizards-qa validate --flow flows/my-game/flow.yaml
wizards-qa template --list
wizards-qa template --create login-flow
```

**Structure:**
```
cmd/
├── main.go              # Entry point, Cobra setup
├── test.go              # Test command (E2E)
├── generate.go          # Flow generation command
├── run.go               # Flow execution command
├── validate.go          # Flow validation
└── template.go          # Template management
```

### 2. AI Analysis Engine

**Purpose:** Understand game mechanics and generate intelligent test flows

**Responsibilities:**
- Parse and understand game specifications
- Analyze live game via browser automation
- Identify game mechanics, UI elements, user flows
- Generate comprehensive test scenarios
- Create Maestro YAML flows from scenarios

**Technology Stack:**
- **Primary Model:** Claude Sonnet 4.5 (reasoning + generation)
- **Secondary Model:** Gemini Pro (alternative/comparison)
- **Context:** Game spec + screenshots + DOM analysis

**Key Functions:**
```go
// pkg/ai/analyzer.go
type GameAnalyzer interface {
    AnalyzeGame(spec GameSpec, url string) (*GameAnalysis, error)
    GenerateScenarios(analysis *GameAnalysis) ([]TestScenario, error)
    CreateFlows(scenarios []TestScenario) ([]MaestroFlow, error)
}

// pkg/ai/models.go
type GameAnalysis struct {
    Mechanics     []GameMechanic
    UIElements    []UIElement
    UserFlows     []UserFlow
    Requirements  []Requirement
    EdgeCases     []EdgeCase
}

type TestScenario struct {
    Name        string
    Description string
    Steps       []TestStep
    Expected    []Assertion
    Priority    Priority
}

type MaestroFlow struct {
    Name     string
    Metadata map[string]string
    Steps    []FlowStep
}
```

**AI Prompting Strategy:**

1. **Game Understanding Phase**
   ```
   System: You are a QA expert analyzing a Phaser 4 web game.
   
   User: Here's the game specification and live URL.
         Analyze the game mechanics, identify all interactive elements,
         and list all possible user actions.
   
   Output: Structured game analysis (JSON)
   ```

2. **Test Scenario Generation**
   ```
   System: You are a comprehensive QA test designer.
   
   User: Given this game analysis, create exhaustive test scenarios
         covering: happy paths, edge cases, failure modes, and UX flows.
   
   Output: Test scenario list (JSON)
   ```

3. **Maestro Flow Creation**
   ```
   System: You are a Maestro flow generator. Output only valid YAML.
   
   User: Convert these test scenarios into Maestro flow files.
         Use these commands: tapOn, inputText, assertVisible, etc.
   
   Output: Maestro YAML flows
   ```

### 3. Maestro Wrapper

**Purpose:** Execute Maestro flows and collect results

**Responsibilities:**
- Execute Maestro CLI commands
- Parse Maestro output
- Collect screenshots and videos
- Generate test reports
- Handle flow failures gracefully

**Key Functions:**
```go
// pkg/maestro/executor.go
type Executor interface {
    RunFlow(flow *MaestroFlow, options RunOptions) (*TestResult, error)
    RunFlows(flows []*MaestroFlow, options RunOptions) (*TestResults, error)
    ValidateFlow(flow *MaestroFlow) error
}

type RunOptions struct {
    Browser       string  // chrome, firefox, safari
    ScreenshotDir string
    VideoCapture  bool
    Timeout       time.Duration
}

type TestResult struct {
    FlowName     string
    Status       Status  // passed, failed, error
    Duration     time.Duration
    Screenshots  []string
    Video        string
    Errors       []Error
    Assertions   []AssertionResult
}
```

**Maestro CLI Integration:**
```bash
# Basic flow execution
maestro test flow.yaml

# With options
maestro test flow.yaml --format junit --output results/

# Multiple flows
maestro test flows/*.yaml

# Continuous mode (watch for changes)
maestro test flow.yaml --continuous
```

### 4. Flow Repository

**Purpose:** Store, version, and manage Maestro flows

**Structure:**
```
flows/
├── templates/           # Reusable flow templates
│   ├── login.yaml
│   ├── navigation.yaml
│   ├── form-fill.yaml
│   └── game-mechanics/
│       ├── click-object.yaml
│       ├── drag-drop.yaml
│       └── score-check.yaml
│
├── my-game/             # Game-specific flows
│   ├── metadata.json    # Game info, test config
│   ├── 01-launch.yaml
│   ├── 02-tutorial.yaml
│   ├── 03-gameplay.yaml
│   ├── 04-win-state.yaml
│   ├── 05-lose-state.yaml
│   └── 06-edge-cases.yaml
│
└── another-game/
    └── ...
```

**Metadata Format:**
```json
{
  "game": {
    "name": "My Phaser Game",
    "url": "https://game.example.com",
    "version": "1.0.0"
  },
  "flows": {
    "generated": "2026-02-06T15:00:00Z",
    "model": "claude-sonnet-4-5",
    "count": 6
  },
  "coverage": {
    "mechanics": ["click", "drag", "score", "timer"],
    "userFlows": ["launch", "tutorial", "gameplay", "win", "lose"],
    "edgeCases": ["network-error", "invalid-input"]
  }
}
```

## Data Flow

### Full E2E Test Flow

```
1. User runs: wizards-qa test --game URL --spec spec.md

2. CLI parses arguments and loads spec file

3. AI Analysis Engine:
   a. Reads game specification
   b. Opens game in browser (via Playwright/Puppeteer)
   c. Captures screenshots and DOM structure
   d. Sends to Claude/Gemini for analysis
   e. Receives game analysis (mechanics, UI, flows)

4. Flow Generation:
   a. AI generates test scenarios from analysis
   b. Converts scenarios to Maestro YAML flows
   c. Saves flows to flows/<game-name>/
   d. Creates metadata.json

5. Flow Execution (Maestro Wrapper):
   a. Validates all generated flows
   b. Executes flows sequentially (or parallel?)
   c. Captures screenshots at each step
   d. Records test results

6. Reporting:
   a. Collects all test results
   b. Generates markdown report
   c. Creates summary (pass/fail counts)
   d. Lists identified issues/bugs
   e. Outputs to console and file

7. Cleanup:
   a. Archives test artifacts
   b. Updates flow repository
   c. Optionally commits to Git
```

## Technology Stack

### Languages & Frameworks
- **Go 1.21+** - CLI and core logic
- **Cobra** - CLI framework
- **Viper** - Configuration management
- **Maestro CLI** - Test execution engine

### AI Integration
- **Anthropic Claude API** - Primary AI model
- **Google Gemini API** - Secondary model
- **OpenAI (optional)** - Future consideration

### Browser Automation (for analysis)
- **Playwright** - Game DOM analysis
- **Puppeteer** - Alternative option

### Testing & Quality
- **Go testing** - Unit tests
- **Testify** - Assertion library
- **golangci-lint** - Linting

## Configuration

### Config File Format (wizards-qa.yaml)

```yaml
# AI Configuration
ai:
  provider: anthropic  # anthropic | google | openai
  model: claude-sonnet-4-5
  apiKey: ${ANTHROPIC_API_KEY}  # From env
  temperature: 0.7
  maxTokens: 8000

# Maestro Configuration
maestro:
  path: /usr/local/bin/maestro
  browser: chrome
  timeout: 300s
  screenshotDir: ./screenshots
  videoCapture: true

# Flow Storage
flows:
  directory: ./flows
  templates: ./flows/templates
  gitCommit: true
  gitRepo: https://github.com/Global-Wizards/game-test-flows

# Reporting
reporting:
  format: markdown  # markdown | json | junit
  outputDir: ./reports
  includeScreenshots: true
  includeVideos: false

# Browser Automation (for analysis)
browser:
  headless: true
  viewport:
    width: 1920
    height: 1080
  timeout: 30s
```

## Phaser 4 & Canvas Interaction

### Challenge

Phaser 4 games render to HTML5 Canvas, which presents unique testing challenges:
- Canvas elements are not DOM elements (can't use text-based selectors)
- Game UI is rendered pixels, not HTML
- Interactions require coordinate-based clicking

### Strategy

1. **Coordinate-Based Interactions**
   ```yaml
   # Maestro flow for canvas clicking
   - tapOn:
       point: 50%,50%  # Center of screen
   - tapOn:
       point: 100,200  # Absolute coordinates
   ```

2. **Visual Assertions**
   ```yaml
   # Screenshot-based verification
   - assertVisible:
       image: expected-game-state.png
       threshold: 0.95
   ```

3. **AI-Assisted Element Identification**
   - Use AI to analyze game screenshots
   - Identify UI element positions from images
   - Generate coordinate-based clicks

4. **Game State Detection**
   - Analyze canvas pixels for specific patterns
   - Use console.log output from game
   - Monitor network requests (score updates, etc.)

### Maestro Commands for Games

```yaml
# Example: Phaser 4 game flow
url: https://my-game.example.com
---
# Launch and wait for game to load
- launchApp
- waitFor:
    text: "Start Game"  # On HTML overlay
    timeout: 10000

# Click "Start Game" button (HTML)
- tapOn: "Start Game"

# Wait for canvas to initialize
- waitFor:
    visible: true
    timeout: 5000

# Click center of game canvas
- tapOn:
    point: 50%,50%

# Wait and click specific coordinate (character)
- waitFor:
    timeout: 1000
- tapOn:
    point: 300,400

# Verify score (from HTML overlay or DOM)
- assertVisible: "Score: 100"

# Take screenshot for verification
- captureScreenshot: gameplay-state.png
```

## Development Roadmap

### Phase 1: Foundation (Week 1)
- [x] Project setup and architecture design
- [ ] Basic CLI structure (Cobra)
- [ ] Maestro CLI wrapper (basic execution)
- [ ] Config file parsing
- [ ] Simple flow validation

### Phase 2: AI Integration (Week 2)
- [ ] Claude API integration
- [ ] Game analysis prompts
- [ ] Scenario generation
- [ ] Flow generation (basic templates)
- [ ] Template library creation

### Phase 3: Maestro Integration (Week 3)
- [ ] Full Maestro command support
- [ ] Result parsing and reporting
- [ ] Screenshot/video capture
- [ ] Error handling and retries

### Phase 4: Phaser 4 Specialization (Week 4)
- [ ] Canvas interaction strategies
- [ ] Visual assertion support
- [ ] Game state detection
- [ ] Phaser-specific templates

### Phase 5: Polish & Production (Week 5)
- [ ] Comprehensive testing
- [ ] Documentation
- [ ] Example flows for sample games
- [ ] CI/CD integration
- [ ] GitHub release

## Open Questions & Decisions Needed

### 1. AI Model Selection
- **Question:** Claude vs Gemini vs both?
- **Consideration:** Cost, quality, speed, context window
- **Recommendation:** Start with Claude Sonnet 4.5, add Gemini as fallback

### 2. Browser Automation Library
- **Question:** Playwright vs Puppeteer for game analysis?
- **Consideration:** Feature set, performance, ecosystem
- **Recommendation:** Playwright (better API, more features)

### 3. Flow Storage Strategy
- **Question:** File system vs database vs Git?
- **Consideration:** Versioning, collaboration, simplicity
- **Recommendation:** File system + Git (simple, versionable)

### 4. Parallel Execution
- **Question:** Run flows sequentially or in parallel?
- **Consideration:** Speed vs reliability, resource usage
- **Recommendation:** Sequential initially, parallel as option

### 5. Canvas Interaction
- **Question:** How to reliably interact with Phaser games?
- **Consideration:** Coordinate precision, game state detection
- **Recommendation:** Hybrid approach (coordinates + visual + console)

### 6. Test Result Storage
- **Question:** Where to store test run results?
- **Consideration:** Historical tracking, trend analysis
- **Recommendation:** File system initially, database later

## Success Metrics

- ✅ Can analyze a simple Phaser 4 game from spec + URL
- ✅ Generates valid Maestro flows automatically
- ✅ Successfully executes flows with >80% success rate
- ✅ Identifies common game issues (broken mechanics, UI bugs)
- ✅ Produces actionable test reports
- ✅ Completes full cycle in <10 minutes for typical game

## References

- [Maestro Documentation](https://docs.maestro.dev/)
- [Phaser 4 Docs](https://phaser.io/)
- [Cobra CLI Framework](https://cobra.dev/)
- [Claude API](https://docs.anthropic.com/)
- [Playwright](https://playwright.dev/)

---

**Next Steps:**
1. Review and approve this architecture (Fernando + Nova)
2. Create detailed implementation plan (Forge)
3. Start Phase 1 development (CLI + Maestro wrapper)
4. Build prototype with simple test game

**Architect:** Lia 🌸  
**Status:** Ready for review and feedback!
