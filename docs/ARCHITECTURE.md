# Wizards QA - System Architecture

**Version:** 0.44.6
**Author:** Lia
**Last Updated:** 2026-02-14
**Status:** Current (reflects implemented system)

## Executive Summary

Wizards QA is an AI-powered QA automation system designed to intelligently test web games. The system accepts a game URL, launches a headless Chrome browser, uses an AI agent to autonomously explore and understand the game, synthesizes comprehensive test scenarios, and executes them via an AI agent with browser tools. A legacy Maestro YAML path is retained for backward compatibility.

## Architecture Overview

```
                        ┌──────────────────────────────┐
                        │       Wizards QA CLI / Web   │
                        │       (Go + Cobra + Web UI)  │
                        └──────────────┬───────────────┘
                                       │
                        ┌──────────────┴───────────────┐
                        │                              │
                        ▼                              ▼
                ┌──────────────┐              ┌──────────────┐
                │    Scout     │              │  Web Backend  │
                │  (Headless   │              │  (Agent Test  │
                │   Chrome)    │              │   Executor)   │
                └──────┬───────┘              └──────┬───────┘
                       │                             │
                       ▼                             ▼
                ┌──────────────┐              ┌──────────────┐
                │  AI Agent    │              │  AI Agent    │
                │  Explorer    │              │  Executor    │
                │              │              │              │
                │ - Browser    │              │ - Browser    │
                │   tools      │              │   tools      │
                │ - Screenshot │              │ - report_    │
                │ - Synthesis  │              │   result     │
                └──────┬───────┘              └──────────────┘
                       │
                       ▼
                ┌──────────────┐
                │  Checkpoint  │
                │  & Resume    │
                │  (JSON files)│
                └──────────────┘
```

## Pipeline Stages

### Stage 1: Browser Launch

**File:** `pkg/scout/headless.go` (`newHeadlessLauncher()`)

Chrome is launched via go-rod with `--headless=new` (Chrome 112+) and the following flags:

| Flag | Purpose |
|------|---------|
| `--disable-dev-shm-usage` | Use /tmp instead of limited /dev/shm |
| `--use-gl=angle --use-angle=swiftshader` | CPU-based WebGL rendering |
| `--enable-unsafe-swiftshader` | Enable SwiftShader explicitly |
| `--autoplay-policy=no-user-gesture-required` | Auto-play media without user click |
| `--font-render-hinting=none` | Consistent font rendering |
| `--in-process-gpu` | Reduce IPC overhead for SwiftShader |
| `--disable-hang-monitor` | Prevent killing "hung" renderers during slow SwiftShader frames |
| `--disable-background-timer-throttling` | Game loop timers run at full speed |
| `--disable-renderer-backgrounding` | Don't deprioritize renderer in headless |
| `--disable-backgrounding-occluded-windows` | Keep rendering in occluded windows |
| `--disable-ipc-flooding-protection` | Remove CDP message rate limiting |
| `--mute-audio` | Skip audio processing entirely |
| `--disable-extensions` | No extension overhead |
| `--disable-component-update` | No background component updates |
| `--disable-background-networking` | No safe-browsing or background network |
| `--disable-smooth-scrolling` | No scroll animations |
| `--no-first-run` | Skip first-run dialog |
| `--disable-sync` | No Chrome sync overhead |
| `--disable-default-apps` | No default app installs |
| `--single-process` | Merge browser + renderer for less IPC |
| `--no-sandbox` | Required for containerized environments |

**Ad/tracker blocking:** URLs matching analytics, tag managers, and telemetry services are blocked at the network level to reduce noise and speed up page loads.

### Stage 2: Screenshot Capture

**File:** `pkg/scout/headless.go` (`CaptureScreenshot()`)

Two paths, tried in order:

1. **Fast path (canvas):** Calls `canvas.toDataURL('image/jpeg', 0.10)` directly on the game canvas. For Phaser games, the game loop is paused first (`game.loop.sleep()`) to prevent SwiftShader contention, then resumed after capture.

2. **CDP fallback:** Uses `Page.captureScreenshot` via Chrome DevTools Protocol. Handles HTML overlays, no-canvas pages, and tainted canvases.

**Viewport downscaling:** If the canvas internal resolution differs from CSS display size, the fast path creates an offscreen canvas at CSS size and draws the game canvas onto it — ensuring the AI's pixel coordinates match the viewport coordinate space.

**Timeout protection:** Screenshots are wrapped with a timeout (20s for auto-screenshots, 30s for explicit screenshot tool calls) with one automatic retry, since SwiftShader can stall for 60+ seconds on complex WebGL frames.

### Stage 3: Click Strategies

**File:** `pkg/scout/click_strategy.go`

Three strategies, selected by `SelectClickStrategy()` based on device type and page content:

| Strategy | Mechanism | When Used |
|----------|-----------|-----------|
| `CDPMouseStrategy` | Trusted CDP mouse events (`InputDispatchMouseEvent`) + redundant JS pointer events on canvas | Canvas/WebGL games on desktop viewports |
| `CDPTouchStrategy` | Trusted CDP touch events (`InputDispatchTouchEvent`) | Mobile viewports (phones/tablets) |
| `JSDispatchStrategy` | JavaScript synthetic pointer + mouse events via `elementFromPoint` | HTML-only games/apps (no canvas) |

**Selection logic** (`SelectClickStrategy()`):
1. Touch device category (iPhone, Android, iPad, Android Tablet) → `CDPTouchStrategy`
2. Unknown category + viewport width <= 480px → `CDPTouchStrategy`
3. Canvas found or canvas framework detected → `CDPMouseStrategy`
4. Otherwise → `JSDispatchStrategy`

**Navigate re-detects:** After `navigate` tool navigates to a new URL, the system re-runs framework detection and selects a new click strategy appropriate for the new page.

**Overlay clearing:** `CDPMouseStrategy` checks if the click target is the canvas via `elementFromPoint`. If an overlay blocks the canvas, it sets `pointer-events: none` on up to 5 overlaying elements to let clicks through.

### Stage 4: Agent Exploration Loop

**File:** `pkg/ai/agent.go`, `pkg/ai/agent_tools.go`

The AI agent explores the game autonomously using browser tools. Each tool call is one "step."

#### Tool Table

| Tool | Description | Auto-Screenshot? |
|------|-------------|-------------------|
| `screenshot` | Capture current page state | Yes |
| `click` | Click at (x,y) pixel coordinates | Yes |
| `type_text` | Type text, optionally click first to focus | Yes |
| `scroll` | Scroll viewport in a direction | Yes |
| `press_key` | Keyboard key (Enter, Space, Escape, arrows...) | Yes |
| `navigate` | Go to URL, re-detect click strategy | Yes |
| `evaluate_js` | Run arbitrary JavaScript in page context | No |
| `wait` | Wait milliseconds or for a CSS selector | No |
| `get_page_info` | Page title, URL, visible text | No |
| `console_logs` | Last 50 browser console messages | No |
| `inspect_game_objects` | Query Phaser 3 / PixiJS scene graph for interactive objects with coordinates | No |
| `request_more_steps` | (Adaptive) Request more exploration budget | No |
| `request_more_time` | (Adaptive) Request more time before timeout | No |

#### Click Repetition Detection

**File:** `pkg/ai/agent_tools.go` (`checkClickRepetition()`)

Tracks the last 5 clicks. If the last 3 clicks are within 30px of each other, a warning is appended to the tool result advising the agent to try different coordinates, check game state via JS, or move on to explore other areas.

#### User Hint Injection

**File:** `pkg/ai/agent.go`

The web UI can send real-time guidance to the agent via a `UserMessages` channel (`<-chan string`). Hints are injected as `[USER HINT]: {text}` messages, prompting the agent to incorporate the guidance into its next action.

#### Budget Status Injection

Every 5 steps (when adaptive exploration or timeout is enabled), the system injects a `[SYSTEM STATUS]` message showing steps used/remaining and time elapsed/remaining, prompting the agent to use `request_more_steps` or `request_more_time` if needed.

#### Exploration Summary Injection

Every 3 steps, the system injects an `[EXPLORATION SUMMARY]` showing:
- Total steps completed
- Tool usage counts
- Unique click regions (bucketed to 50px grid)
- Last 5 non-trivial action results

This helps the agent avoid repeating actions and plan what to explore next.

#### Console Log Collection

**File:** `pkg/scout/headless.go`

Browser console messages (log, warn, error, etc.) are collected throughout the session, capped at 2,000 lines. The `console_logs` tool returns the last 50 lines.

#### Screenshot Context Management

To keep the AI context window manageable, only the 3 most recent screenshots are retained in the conversation. Before synthesis, all screenshots are stripped.

### Stage 5: Synthesis

**File:** `pkg/ai/analyzer.go`, `pkg/ai/types.go`

After exploration, the agent's observations are synthesized into a structured `ComprehensiveAnalysisResult`:

```json
{
  "gameInfo": {
    "name": "string",
    "technology": "string",
    "genre": "string",
    "features": ["string"]
  },
  "mechanics": [{
    "name": "string",
    "actions": ["string"],
    "expected": "string"
  }],
  "uiElements": [{
    "name": "string",
    "type": "string",
    "selector": "string",
    "location": { "x": 0, "y": 0 }
  }],
  "userFlows": [{
    "name": "string",
    "steps": ["string"],
    "expected": "string",
    "priority": "string"
  }],
  "edgeCases": [{
    "scenario": "string",
    "expected": "string"
  }],
  "scenarios": [{
    "name": "string",
    "type": "string",
    "steps": ["string"],
    "priority": "string",
    "tags": ["string"]
  }],
  "uiuxAnalysis": [{ "finding": "string", "severity": "string", "recommendation": "string" }],
  "wordingCheck": [{ "text": "string", "issue": "string", "suggestion": "string" }],
  "gameDesign": [{ "aspect": "string", "finding": "string", "impact": "string" }],
  "gliCompliance": [{ "requirement": "string", "status": "string", "notes": "string" }],
  "navigationMap": {
    "screens": [{ "name": "string", "description": "string" }],
    "transitions": [{ "from": "string", "to": "string", "action": "string" }],
    "entryScreen": "string"
  }
}
```

### Stage 6: Test Execution

Two execution modes are available:

#### Agent Mode (default since v0.35.0)

**File:** `web/backend/agent_executor.go`

No YAML flows are generated. Scenarios are stored in the database as structured `TestScenario` objects. For each scenario:

1. Agent launches headless Chrome and navigates to the game URL
2. Receives the scenario description and steps as context
3. Executes steps autonomously using browser tools (same toolbox as exploration)
4. Calls `report_result` tool to report pass/fail with evidence
5. Max 30 steps per scenario

Token usage and cost are tracked across all scenarios.

#### Legacy Maestro Mode (backward compatibility)

**File:** `pkg/maestro/executor.go`

Generates Maestro YAML flows from scenarios and executes them via the Maestro CLI subprocess. Retained for backward compatibility but not the default path.

### Stage 7: Checkpoint & Resume

**File:** `pkg/ai/types.go` (`CheckpointData`, `WriteCheckpoint`, `ReadLatestCheckpoint`, `ReadResumeData`)

Checkpoints are saved after each pipeline stage to enable crash recovery without re-doing expensive exploration:

| Checkpoint | Stage | Contents |
|------------|-------|----------|
| `checkpoint_scouted.json` | After page metadata collection | Page meta, framework detection |
| `checkpoint_analyzed.json` | After agent exploration | Page meta, agent steps, exploration data |
| `checkpoint_synthesized.json` | After synthesis | Full analysis result, module config |

**Checkpoint data structure:**
```go
type CheckpointData struct {
    Step       string          // "scouted", "analyzed", "synthesized"
    AgentMode  bool
    PageMeta   json.RawMessage // Page metadata
    Analysis   json.RawMessage // ComprehensiveAnalysisResult
    AgentSteps json.RawMessage // Saved exploration steps for resume
    Modules    AnalysisModules // Which analysis modules were enabled
    Timestamp  string
}
```

On resume, the system reads the latest checkpoint and skips completed stages.

## Technology Stack

### Core
- **Go** - CLI and core logic
- **Cobra** - CLI framework
- **go-rod** - Browser automation (Chrome DevTools Protocol)
- **SwiftShader** - CPU-based WebGL rendering for headless environments

### AI Integration
- **Anthropic Claude API** - Primary AI model
- **Google Gemini API** - Secondary model

### Web UI
- **Go net/http** - Backend API server
- **Agent executor** - Runs test scenarios via AI agent

## References

- [go-rod Documentation](https://go-rod.github.io/)
- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/)
- [Phaser 3 Docs](https://phaser.io/)
- [Claude API](https://docs.anthropic.com/)

---

**Architect:** Lia
**Last Reviewed:** 2026-02-14
