# Changelog

All notable changes to wizards-qa will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.10.0] - 2026-02-09

### Toggleable Analysis Modules & Analysis List View

#### Added
- **Toggleable analysis modules** — UI/UX Analysis, Wording Check, Game Design Analysis, and Test Flow generation can now be individually enabled/disabled before starting an analysis, reducing token usage and focusing results
- **Analysis List view** (`/analyses`) — dedicated full-page list for browsing all past analyses with search, status filters, module badge filters, re-analyze, and delete actions
- **Module badges** — analysis list items show colored pills indicating which modules were enabled (UI/UX, Wording, Design, Flows)
- **CLI flags** — `--no-uiux`, `--no-wording`, `--no-game-design`, `--no-test-flows` flags for the `scout` command to disable specific analysis modules
- **Conditional AI prompts** — `BuildAnalysisPrompt()` and `BuildSynthesisPrompt()` dynamically construct prompts with only enabled module sections, saving tokens
- **`modules` column** in analyses DB table — persists which modules were enabled for each analysis

#### Changed
- **Sidebar navigation** — "Analyze" renamed to "Analyses" and now links to the list view; "New Analysis" button at the top of the list view navigates to the analyze page
- **Recent analyses** on Analyze page — slimmed from 5 to 3 items with a "View All Analyses" link
- **Analysis Detail view** — tabs for disabled modules are now hidden entirely (vs previously just disabled when empty)
- **Back button** in Analysis Detail now navigates to `/analyses` list instead of `/analyze`

## [0.9.0] - 2026-02-09

### Dedicated Analysis Detail View with Tabbed Navigation

#### Added
- **Analysis Detail View** (`/analyses/:id`) — dedicated page for viewing analysis results with rich tabbed navigation instead of inline collapsible sections
- **Overview Tab** — game info card, 8-stat grid (mechanics, UI elements, user flows, edge cases, UI/UX issues, wording issues, game design findings, test flows), page metadata, and screenshot thumbnail
- **Functional QA Tab** — tables for mechanics (with actions and priority), UI elements (with selectors), and cards for user flows (with numbered steps) and edge cases
- **Findings Tab** (reusable) — severity summary bar, severity toggle filters, category dropdown filter, and finding cards with severity badges; used for UI/UX Analysis, Wording Check, and Game Design Analysis tabs
- **Test Flows Tab** — grid of flow cards with tag badges and command counts, click-to-preview YAML dialog with copy button
- **Exploration Tab** (agent mode only) — embedded AgentStepNavigator for reviewing agent exploration steps
- **`severityVariant()` shared utility** — extracted from Analyze.vue to `lib/utils.js` for reuse across components
- **"View Full Analysis" button** — in Analyze.vue completed state, navigates to the detail view
- **Project-scoped routing** — detail view accessible at both `/analyses/:id` and `/projects/:projectId/analyses/:id`

#### Changed
- **Recent analyses list** — clicking an analysis now navigates to the detail view instead of loading results inline

## [0.8.0] - 2026-02-09

### UI/UX Analysis, Wording Check & Game Design Analysis

#### Added
- **UI/UX Analysis section** — AI now evaluates visual design quality (alignments, spacing, color harmony, typography, visual hierarchy, accessibility, animations) and reports findings with severity levels and fix suggestions
- **Wording/Translation Check section** — AI examines all visible text for grammar, spelling, inconsistent terminology, tone, truncated text, placeholder text, and text overflow issues
- **Game Design Analysis section** — AI analyzes game design quality including reward systems, balance, progression, player engagement, difficulty curve, monetization fairness, tutorial quality, and feedback systems
- **Frontend display** — three new collapsible sections in analysis results with severity badges, category tags, and detailed findings
- **Markdown export** — new sections included in markdown export format

#### Changed
- **Token budgets raised** — profiles now use higher maxTokens (debug: 4096, quick/balanced: 8192, thorough/maximum: 16384) to accommodate the expanded analysis output
- **SynthesisMaxTokens floor raised** from 8192 to 16384 to prevent truncation of the larger JSON output
- **Custom max tokens ceiling** raised from 16384 to 32768

## [0.7.3] - 2026-02-09

### Fix 401 Unauthorized Errors After 15 Minutes

#### Fixed
- **Access token expiring during active sessions** — increased access token TTL from 15 minutes to 24 hours. Users working on analyses (which can run 10-30 minutes) were hitting 401 errors that appeared in the browser console when the token expired mid-session. The refresh token remains at 7 days, and the axios interceptor still handles edge cases.

## [0.7.2] - 2026-02-09

### Fix Synthesis Failure — Context Too Large, Truncation, Error Messages

#### Fixed
- **Synthesis failing with "CLI exited with code 1"** — the synthesis API call included all base64 screenshots (~1.6MB) from exploration. Now strips ALL screenshots before synthesis since the AI already observed them during exploration.
- **Truncated synthesis JSON silently failing** — if `stop_reason=max_tokens`, the incomplete JSON now gets auto-repaired by closing open brackets/braces before parsing. Logs a warning when truncation occurs.
- **Cryptic "CLI exited with code 1" error message** — exit-code errors now include `lastKnownStep` (e.g., "failed during: agent_synthesize") and the last meaningful stderr line for debugging context.
- **SynthesisMaxTokens floor too low** — raised from 4096 to 8192. The comprehensive JSON output (gameInfo + mechanics + uiElements + userFlows + edgeCases + scenarios) routinely needs 4000–6000 tokens; 8192 provides safe headroom.

## [0.7.1] - 2026-02-09

### Code Quality Audit — DRY, N+1 Queries, Transaction Safety

#### Fixed
- **N+1 stats queries** — `GetStats()` and `GetStatsByProject()` consolidated from 6 sequential queries to 2, reducing database round-trips.
- **DeleteProject missing transaction** — multi-step delete (unassign analyses, plans, results + delete) now wrapped in a single SQL transaction with proper rollback on failure.
- **Unbounded query results** — added LIMIT clauses to all project-scoped list queries (200) and user/project listings (500/100) to prevent memory issues with large datasets.
- **ID collision risk** — all entity IDs (analysis, user, project, test, plan, member) now include a random suffix via `crypto/rand` for collision resistance.

#### Added
- **`useClipboard` composable** — extracted duplicated clipboard copy+timeout pattern from Analyze.vue (×2) and Flows.vue into a reusable composable.
- **`newID()` helper** — centralized ID generation with `prefix-timestamp-randomhex` format, replacing 8 inline `fmt.Sprintf` patterns.
- **`authTokenResponse()` helper** — extracted triplicated auth token response struct from register/login/refresh handlers.
- **`marshalToPtr()` applied to migrations** — replaced 4 remaining inline JSON marshal patterns in db.go migration functions.
- **Missing database indexes** — added `idx_test_results_status` and `idx_project_members_user` for frequently-queried columns.

#### Changed
- **Consistent date formatting** — Tests.vue, Reports.vue, ProjectSettings.vue now use `formatDate()` from dateUtils instead of inline `new Date().toLocaleString()`.

#### Removed
- **Dead code** — deleted unused `useApiLoader.js` composable.

## [0.7.0] - 2026-02-09

### Complete Profiles System — Optimize Agent Token Usage & Timeouts

#### Changed
- **CLI TotalTimeout scales with agentSteps** — exploration timeout now uses `steps × 30s + 5min buffer` (clamped 5–20min) instead of hardcoded 12min. Debug (3 steps) → 5min, quick (8) → 9min, balanced (15) → 12.5min, thorough (20) → 15min, maximum (25) → 17.5min.
- **Default config MaxTokens lowered from 16000 to 8192** — no profile uses more than 8192; the old value encouraged overly verbose exploration output.
- **Profile temperatures lowered for reliable JSON** — thorough/maximum dropped from 0.7 → 0.3/0.2; balanced from 0.5 → 0.3; quick from 0.3 → 0.2. Structured JSON output is far more reliable at low temperatures.
- **Quick profile maxTokens raised from 2048 → 4096** — synthesis needs at least 4096 tokens for full JSON output.
- **Debug profile maxTokens raised from 1024 → 2048, agentSteps reduced from 5 → 3** — faster pipeline debugging.
- **Maximum profile description updated** to mention extensive exploration.

#### Added
- **SynthesisMaxTokens in AgentConfig** — new field overrides maxTokens for the synthesis call only, ensuring low-token profiles (quick, debug) don't truncate the synthesis JSON. Automatically set to 4096 when the profile's maxTokens is below that threshold.
- **Cost/time indicators in profile selector** — each profile now shows estimated cost tier and time range (e.g., "medium cost · ~5–10 min") in the Analyze page UI.
- **`agentTotalTimeout()` helper** — reusable timeout formula shared between `DefaultAgentConfig()` and `cmd/scout.go`.

## [0.6.1] - 2026-02-09

### Fix Analysis Timeouts — Per-Phase Retry & Dynamic Timeouts

#### Fixed
- **Synthesis and flow generation failures losing all exploration work** — both calls now auto-retry up to 3 times with exponential backoff (5s → 10s → 20s). A single transient API error no longer wastes 12+ minutes of exploration.
- **Fixed 15-minute timeout too short for thorough analyses, too long for quick ones** — backend timeout now scales dynamically with agent steps (e.g., 5 steps → 11min, 20 steps → 21min, 25 steps → 25min), clamped between 10–30 minutes.
- **Exploration starving synthesis of time** — exploration loop now reserves 3 minutes for synthesis by stopping early when the time budget runs low, ensuring synthesis always has time to complete.
- **Timeout errors lacking context** — timeout error messages now include the last known step (e.g., "Analysis timed out after 25 minutes (last step: agent_synthesize)").

#### Added
- **Retry progress events** — new `synthesis_retry` and `flows_retry` progress events stream to the frontend so users see "Retrying synthesis (attempt 2/3)..." in real time.
- **Failed phase indicator** — error state now shows which phase failed (e.g., "Failed during: Synthesis") below the error message.

## [0.6.0] - 2026-02-09

### Analysis Profiles — Configurable Model, Tokens & Steps

#### Added
- **Analysis Profiles** — 5 presets (Quick Scan, Balanced, Thorough, Maximum, Debug) that configure model, max tokens, agent steps, and temperature in one click
- **Profile selector UI** on the Analyze page with a dropdown below the Agent Mode toggle; selecting a profile shows a summary of its settings
- **Custom profile mode** — selecting "Custom" expands individual fields (model, max tokens, agent steps, temperature) for full manual control
- **CLI flags** — `--model`, `--max-tokens`, `--temperature` flags on the `scout` command for direct override of AI parameters
- **Backend passthrough** — `AnalysisRequest` now accepts `model`, `maxTokens`, `agentSteps`, `temperature` fields and passes them as CLI flags to the scout subprocess

## [0.5.2] - 2026-02-09

### Fix Agent Timeout — Sliding Window Screenshot Pruning

#### Fixed
- **Agent exploration exhausting timeout before synthesis** — every `CallWithTools` sent the full conversation including ALL accumulated base64 screenshots (~100-200KB each). By step 17, API calls took 50-72 seconds each, consuming the entire timeout budget before synthesis could run. Added `pruneOldScreenshots()` sliding window that keeps only the last 4 screenshots in the conversation, replacing older ones with text placeholders. API calls now stay consistently fast regardless of exploration length.
- **Backend timeout too tight for full agent pipeline** — increased backend context timeout from 10 to 15 minutes for agent mode, giving enough headroom for exploration + synthesis + flow generation.
- **Agent exploration timeout too tight** — increased `TotalTimeout` from 8 to 12 minutes since the synthesis and flow generation calls happen inside the same context.

## [0.5.1] - 2026-02-08

### Fix Agent Timeouts on Multimodal API Calls

#### Fixed
- **Agent API calls timing out** — HTTP client timeout was 120s but multimodal API calls (screenshot images + growing conversation) routinely exceed this. Increased to 300s to match agent total timeout.
- **Backend context too short for agent mode** — the 5-minute `context.WithTimeout` killed the CLI subprocess before the agent could finish. Now uses 10 minutes for agent mode, 5 minutes for standard mode.
- **Agent total timeout too tight** — increased from 5 to 8 minutes to account for browser startup (~30s), canvas readiness polling (~20s), and multiple slow API calls with image context.

#### Added
- **API call timing logs** — `CallWithTools` now logs request size, elapsed time, token usage, and stop reason for each Claude API call, enabling diagnosis of slow calls.

## [0.5.0] - 2026-02-08

### Persist Agent Steps, Fix Errors, Enhanced Logging & Step Navigator

#### Added
- **Agent steps persistence** — agent exploration steps are now saved to the `agent_steps` database table as they arrive, surviving analysis failures and server restarts
- **Step Navigator UI** — new `AgentStepNavigator` component with left panel step list, right panel detail view, prev/next navigation, and full-screen screenshot dialog
- **Persisted screenshots** — agent screenshots are saved to `/app/data/screenshots/{analysisID}/` and served via REST API instead of ephemeral base64
- **New API endpoints** — `GET /api/analyses/{id}/steps` returns all persisted agent steps; `GET /api/analyses/{id}/steps/{stepNumber}/screenshot` serves screenshot JPEGs
- **Enhanced debug log** — "Copy Full Log" now includes agent step details (tool name, input, result, reasoning, duration, errors) and last agent reasoning text
- **Retry Analysis button** — error state now shows a "Retry Analysis" button that re-runs with the same URL and agent mode setting
- **Agent steps visible on failure** — error state shows the step navigator so users can see what the agent did before the failure occurred

#### Fixed
- **Error classification** — analysis errors now show concise messages ("Analysis timed out after 5 minutes", "CLI exited with code N") instead of dumping raw agent reasoning text from stderr
- **Full stderr preserved** — the complete stderr output is saved in the `error_message` database column for debugging, separate from the user-visible error
- **Live steps preserved on failure** — `liveAgentSteps` are no longer cleared when analysis fails, keeping them available for the debug log and step navigator

#### Changed
- **Agent step reasoning tracking** — each persisted step captures the agent's latest reasoning text at the time the step was recorded
- **Delete analysis cleanup** — deleting an analysis now also removes its persisted screenshots directory

## [0.4.6] - 2026-02-09

### Fix Analysis Failures — SQLite Locking & OOM Kills

#### Fixed
- **SQLITE_BUSY errors during agent analysis** — Go's `database/sql` connection pool created multiple connections to SQLite, but PRAGMAs (including `busy_timeout=5000`) are per-connection; new pool connections had no timeout and failed immediately on contention. Fixed with `SetMaxOpenConns(1)` which serializes all DB access through a single connection where PRAGMAs persist.
- **Chrome processes OOM-killed during agent analysis** — headless Chrome with SwiftShader WebGL rendering exceeded the 1GB VM memory limit, causing SIGKILL of all Chrome child processes mid-analysis. Bumped Fly.io VM to 2GB RAM / 2 shared CPUs.

## [0.4.5] - 2026-02-09

### Headless Chrome Hardening for Phaser/WebGL Game Testing

#### Fixed
- **Old headless mode had degraded WebGL** — switched from `Headless(true)` (legacy `--headless`) to `HeadlessNew(true)` (`--headless=new`, Chrome 112+) which shares the full browser rendering pipeline for proper WebGL/canvas support
- **SwiftShader libraries missing from Docker image** — added `chromium-swiftshader` package (CPU-based Vulkan for WebGL without a real GPU) and `ttf-freefont` for complete font coverage
- **Game audio blocking page load** — added `--autoplay-policy=no-user-gesture-required` to prevent Phaser games from hanging on Web Audio API initialization
- **Inconsistent screenshot font rendering** — added `--font-render-hinting=none` for predictable text rendering across environments

## [0.4.4] - 2026-02-09

### WebGL Support for Phaser 4 Games

#### Fixed
- **WebGL completely broken in headless Chrome** (CRITICAL) — the combination of `--disable-gpu` + `--disable-software-rasterizer` flags eliminated all WebGL rendering paths, causing Phaser 4 games (and Phaser 3 WebGL games) to render black or fail entirely
- **Replaced with SwiftShader software rendering** — now uses `--use-gl=angle --use-angle=swiftshader --enable-unsafe-swiftshader` for CPU-based WebGL in both `ScoutURLHeadlessKeepAlive` and `ScoutURLHeadless`
- **Added missing OpenGL/EGL libraries to Docker image** — `mesa-egl`, `mesa-gl`, `libxcomposite`, `libxdamage` for reliable SwiftShader operation

## [0.4.3] - 2026-02-09

### WebSocket Fix & Project Rules

#### Fixed
- **WebSocket connections failing in production** — the logging middleware's `statusWriter` wrapper did not implement `http.Hijacker`, causing gorilla/websocket to reject every `/ws` upgrade with "response does not implement http.Hijacker"

#### Added
- **CLAUDE.md** — project rules for Claude Code (always update VERSION + CHANGELOG on functional commits)

## [0.4.2] - 2026-02-08

### JWT Token Expiration Detection & Session-Gated Game Handling

#### Added
- **JWT token expiry detection** — `checkURLTokenExpiry()` in `pkg/ai/analyzer.go` scans URL query params for JWT-shaped values, decodes the payload, and extracts the `exp` claim
- **Token status in URL hints** — `parseURLHints()` now includes `tokenStatus` and `expiredTokens` keys, which flow automatically into analysis prompts
- **Token status in agent initial message** — `AgentExplore()` includes token expiry info (e.g., "sessionToken expired 2h ago") so the agent knows immediately whether the game can load
- **SESSION-GATED GAMES system prompt** — new section in `AgentSystemPrompt` instructs the agent to abort quickly (1 screenshot + console check + EXPLORATION_COMPLETE) when tokens are expired
- **Frontend expired token warning** — `tokenWarning` computed property on Analyze page parses the URL for expired JWTs and shows an `<Alert>` below the URL input (warning only, does not block submission)
- **Token info in debug log** — `buildDebugLogText()` includes the token warning in the clipboard diagnostic output

## [0.4.1] - 2026-02-08

### Audit Remediation — Security, Race Conditions & UX Fixes

#### Fixed
- **Path traversal in screenshot filename** (CRITICAL) — sanitize with `filepath.Base()` and reject path separators in `web/backend/analyze.go`
- **Race condition in agent hint sender** (HIGH) — move stdin write inside mutex-protected section to prevent write-to-closed-pipe crash
- **Rate limit set before write succeeds** (MEDIUM) — `lastHintAt` now only updated after a successful stdin write
- **Unchecked `os.WriteFile` for screenshots** (MEDIUM) — log error in `pkg/ai/agent.go` when screenshot write fails
- **Unchecked `os.MkdirAll` for screenshot dir** (MEDIUM) — log error in `cmd/scout.go` when directory creation fails
- **Unbounded `liveAgentSteps` memory growth** (HIGH) — cap at 50 entries; store `hasScreenshot` flag instead of full base64 strings
- **Auto-scroll overrides manual scrolling** (LOW) — only auto-scroll timeline and log panels when user is near the bottom

#### Removed
- Dead `saveToLocalStorage` function in `useAnalysis.js`

## [0.4.0] - 2026-02-08

### Agent Mode - Interactive Game Exploration (2026-02-08)

#### Added
- **Optional Agent Mode** - AI actively explores games through browser interactions
  - Agentic loop of 10-20 steps: Claude uses browser tools (click, scroll, type, screenshot, eval JS) to explore the game
  - Synthesis call produces structured analysis grounded in actual observations
  - Last 5 agent screenshots passed to flow generation for coordinate-grounded Maestro flows
  - `--agent` CLI flag and `--agent-steps` (default 20) to control exploration depth
- **Tool Use API Support** - `CallWithTools` method on `ClaudeClient` for Claude tool use protocol
  - `ToolDefinition`, `AgentMessage`, `ResponseContentBlock`, `ToolUseResponse`, `ToolResultBlock` types
  - `ToolUseAgent` interface in `pkg/ai/base.go`
- **BrowserPage Interface** - Decouples AI package from go-rod
  - `BrowserPage` interface in `pkg/ai/types.go` with 7 methods (CaptureScreenshot, Click, TypeText, Scroll, EvalJS, WaitVisible, GetPageInfo)
  - `RodBrowserPage` adapter in `pkg/scout/headless.go` implementing the interface
  - `ScoutURLHeadlessKeepAlive` returns a live browser page + cleanup function for agent mode
- **7 Browser Tools** - `pkg/ai/agent_tools.go`
  - `screenshot`, `click`, `type_text`, `scroll`, `evaluate_js`, `wait`, `get_page_info`
  - `BrowserToolExecutor` maps tool names to `BrowserPage` method calls
- **Agent Exploration Loop** - `pkg/ai/agent.go`
  - `AgentExplore` runs the agentic loop with progress events
  - `AnalyzeFromURLWithAgent` integrates agent exploration with existing flow generation
- **Frontend Agent Mode Toggle**
  - "Agent Mode" checkbox on the Analyze page
  - Agent exploration progress step with real-time step updates
  - Agent results section showing step-by-step actions with clickable screenshot thumbnails
  - Full-screen screenshot dialog for agent step screenshots
- **Backend Agent Mode** - `agentMode` field in analysis request, passes `--agent` to CLI

#### Design Decisions
- Agent mode is fully optional — the 2-call pipeline remains the default and is untouched
- `BrowserPage` interface in `pkg/ai` keeps the AI package decoupled from go-rod
- Synthesis is a separate final call (no tools) to ensure clean JSON output
- Agent steps and mode are included in JSON output and persisted through the full stack

## [0.3.0] - 2026-02-08

### Phase 3 Complete - Projects & Organization (2026-02-08)

#### Added
- **Projects as Top-Level Entity** - Group analyses, test plans, and test results by game
  - Full CRUD for projects with rich fields (name, URL, description, color, tags)
  - Nested routing (`/projects/:id/...`) with project-scoped sidebar
  - Project dashboard with scoped stats, recent tests, and quick actions
  - Team member management (add by email, roles: owner/admin/member)
  - Project settings with danger zone (delete)
- **Auto-Migration** - Existing data automatically grouped into projects by `game_url`
- **Project-Aware Views** - Analyze, Tests, and New Test Plan views adapt to project context
  - Game URL auto-filled from project settings
  - API calls scoped to project when inside one
- **Dashboard Integration** - "Your Projects" section on global dashboard
- **Backend**
  - `projects` and `project_members` database tables with indexes
  - 14 new API endpoints for project CRUD, stats, scoped entities, and members
  - Idempotent database migrations for `project_id` columns
  - Project ID propagation through analysis and test execution flows
- **Frontend**
  - `useProject` composable for singleton project state
  - 6 new views: ProjectList, ProjectForm, ProjectLayout, ProjectDashboard, ProjectSettings, ProjectMembers
  - Dual-mode sidebar (global navigation vs project-scoped navigation)

## [Unreleased]

### Phase 2 Complete - AI Integration (2026-02-07)

#### Added
- **Complete AI Integration** with Claude API
  - `pkg/ai/client.go` - Claude API client with structured analysis
  - `pkg/ai/analyzer.go` - Game analysis engine with 3-phase workflow
  - `pkg/ai/types.go` - Comprehensive data structures and prompt templates
- **AI-Powered CLI Commands**
  - `wizards-qa generate` - Analyze games and generate flows automatically
  - `wizards-qa test` - Full E2E testing with AI + execution + reporting
- **Template Library** - Reusable flow patterns
  - 6 game mechanic templates (click, collect, movement, collision, victory, game-over)
  - Template management commands (list, show, apply)
  - Variable substitution system
  - Comprehensive template documentation
- **Example Game Spec** - `examples/simple-platformer-spec.md`
- **Documentation** - Template README with usage guide

#### Changed
- Updated `cmd/generate.go` with full AI workflow
- Updated `cmd/test.go` with 6-step E2E process
- Enhanced `cmd/template.go` with list/show/apply commands

### Phase 1 Complete - Core Infrastructure (2026-02-06)

#### Added
- **Maestro Wrapper** - Flow execution engine
  - `pkg/maestro/executor.go` - Single & multi-flow execution
  - `pkg/maestro/types.go` - Result data structures
  - `pkg/maestro/capture.go` - Screenshot/video asset management
- **Flow Validation** - Comprehensive YAML validation
  - `pkg/flows/validator.go` - Validates 20+ Maestro commands
  - `pkg/flows/parser.go` - Maestro flow parser
  - `pkg/flows/types.go` - Flow data structures
- **Configuration System** - Full config management
  - `pkg/config/config.go` - Config loading with environment variables
  - `cmd/config.go` - Config CLI commands (show, init, validate)
  - `wizards-qa.yaml.example` - Example configuration
- **Test Reporting** - Markdown report generation
  - `pkg/report/generator.go` - Beautiful test reports with statistics
- **CLI Commands**
  - `wizards-qa validate` - Validate Maestro flows
  - `wizards-qa config` - Manage configuration
  - `wizards-qa run` - Execute flows and generate reports

#### Changed
- Enhanced `cmd/run.go` with full E2E execution
- Improved error handling across all commands

### Phase 0 Complete - Foundation (2026-02-06)

#### Added
- **Project Structure** - Complete Go + Cobra CLI framework
  - 5 main commands: test, generate, run, validate, template
  - Modular package structure (pkg/maestro, pkg/flows, pkg/config, pkg/ai, pkg/report)
- **Maestro CLI Integration**
  - Maestro v2.1.0 installed and configured
  - Research documentation on capabilities
- **Documentation**
  - `README.md` - Complete usage guide
  - `docs/ARCHITECTURE.md` - System architecture (15KB)
  - `docs/MAESTRO-RESEARCH.md` - Maestro capabilities
  - `docs/PROJECT-BRIEF.md` - Vision and requirements
  - `ROADMAP.md` - Development roadmap
- **Example Flows**
  - `flows/templates/example-game.yaml` - Complete example flow
- **Build System**
  - Go modules setup
  - Build scripts and automation

## Statistics

### Code Metrics
- **Total Go Files:** 17
- **Total Lines of Code:** ~2,800
- **Packages:** 6 (maestro, flows, config, ai, report + cmd)
- **Templates:** 7 flow templates
- **Documentation:** 8 markdown files

### Commits
- **Phase 0:** 3 commits (foundation)
- **Phase 1:** 4 commits (core infrastructure)
- **Phase 2:** 2 commits (AI integration + templates)
- **Total:** 9 commits

### Features Completed
- ✅ Flow validation (20+ Maestro commands)
- ✅ Configuration management
- ✅ Maestro execution with timeout
- ✅ Screenshot/video capture
- ✅ Markdown test reports
- ✅ Claude AI integration
- ✅ Game analysis and understanding
- ✅ Test scenario generation
- ✅ Maestro flow generation
- ✅ Template library with 6 patterns
- ✅ Template management CLI
- ✅ End-to-end automation

## Links

- **GitHub:** https://github.com/Global-Wizards/wizards-qa
- **Issues:** https://github.com/Global-Wizards/wizards-qa/issues
- **Discord:** https://discord.com/invite/clawd
