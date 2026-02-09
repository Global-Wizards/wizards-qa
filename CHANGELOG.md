# Changelog

All notable changes to wizards-qa will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
