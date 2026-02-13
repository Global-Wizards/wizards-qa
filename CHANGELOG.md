# Changelog

All notable changes to wizards-qa will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.29.0] - 2026-02-13

### Added
- **Debug information in analysis progress log** — agent step summaries (`[Agent]`), test flow results (`[Test]`), and test step details now appear inline in the live progress log during analysis, making failures immediately diagnosable without clicking "Copy Debug Log".
- **Enriched failure broadcasts** — `analysis_failed` WebSocket messages now include exit code, last step, stderr line count, checkpoint availability, and the last 10 stderr lines, all rendered as `[Error]`/`[Stderr]` entries in the progress log.
- **Test data in debug log export** — "Copy Debug Log" now includes test flow results and test step details (command, status, reasoning) alongside existing agent step data.

## [0.28.1] - 2026-02-13

### Fixed
- **`testingDetail` computed broken** — added missing `.value` on `testStepScreenshots` ref inside computed, which caused step count to never display during the testing phase.
- **`runTests` module not persisted to database** — added `runTests` to the modules JSON serialization so the flag survives persistence and reload.
- **Test state leaking on resume** — `continueAnalysis()` now resets `testRunId`, `testStepScreenshots`, and `testFlowProgress` to prevent stale test data from a previous run.

### Changed
- **Capped `testStepScreenshots` array** — applied the same `MAX_LIVE_STEPS` (50) rotation used by `liveAgentSteps` to prevent unbounded memory growth from base64 screenshots.

## [0.28.0] - 2026-02-13

### Added
- **Integrated browser test execution in analysis** — new opt-in "Run Tests" toggle in Analysis Modules automatically runs browser tests after flow generation completes, with real-time progress shown inline via TestStepNavigator.
- **Testing progress phase in analysis panel** — new violet "Running browser tests" phase in the progress timeline with step screenshots, flow progress tracking, and live test WS event integration.
- **Re-run tests from analysis detail** — completed analyses with test plans show a "Run Tests" button and a "Test Results" tab displaying pass rate and per-flow results.
- **`lastTestRunId` on analysis records** — analyses now persist the ID of the last browser test run for quick access to results.

## [0.27.0] - 2026-02-12

### Added
- **`runFlow` command in browser mode** — flows using `runFlow: "00-setup.yaml"` now execute the referenced flow's commands in browser mode, enabling shared setup flows. Includes recursion guard and URL navigation from the referenced flow's metadata.
- **TestStepNavigator component** — agent-like step navigator for browser-mode test results with flow-grouped sidebar, base64 screenshot display, AI reasoning panel, prev/next navigation, lightbox dialog, and live auto-follow mode during execution.
- **AI reasoning in step data** — vision-powered commands (`tapOnText`, `assertVisible`, `assertNotVisible`, `extendedWaitUntil`) now return the AI's raw response as reasoning, broadcast via WebSocket and displayed in the step navigator.

### Fixed
- **Duplicate flow result broadcasts** — each flow completion was broadcast twice (once via `broadcastTestLog` and again via a separate `test_progress` message). Now uses inline log buffer update with a single broadcast.

### Changed
- **TestRunDetail step display** — replaced inline expandable screenshot sections with the new `TestStepNavigator` component for a richer step-by-step browsing experience.

## [0.26.1] - 2026-02-13

### Fixed
- **Browser executor: navigation failures now skip flow** — flows with invalid URLs are properly marked as failed instead of continuing execution on the wrong page.
- **Browser executor: deduplicate waitUntil prompt** — removed redundant identical prompt branch for visible/notVisible conditions in `executeWaitUntil`.

## [0.26.0] - 2026-02-13

### Added
- **Browser-based test execution** — run Maestro YAML test flows directly in headless Chrome using AI vision, bypassing the Maestro CLI entirely. Supports `openLink`, `tapOn` (text, point, ID), `inputText`, `scroll`, `extendedWaitUntil`, `assertVisible`, `assertNotVisible`, `takeScreenshot`, `back`, `repeat`, and more. Text-based commands use Claude vision to locate elements on screenshots.
- **Run mode selector** — test plans now have a dropdown Run button with "Run with Maestro" (existing) and "Run in Browser" (new headless Chrome + AI) modes, available in both Tests and EditTestPlan views.
- **Per-step screenshots in test runs** — browser mode broadcasts screenshots after each command via WebSocket (`test_step_screenshot`, `test_flow_started`, `test_command_progress`). TestRunDetail shows expandable flow results with inline step screenshots and a lightbox viewer.
- **Multi-device analysis** — analyze games on multiple device viewports simultaneously. Toggle "Multi-Device" mode on the Analyze page to select Desktop, iOS, and Android with configurable device presets. Each device runs as a separate analysis via the new `POST /api/analyze/batch` endpoint.
- **Batch analysis API** — `POST /api/analyze/batch` accepts a list of device configs and launches concurrent analyses (one per device), sharing the existing concurrency semaphore.

## [0.25.4] - 2026-02-13

### Fixed
- **Root cause fix for YAML "mapping values" errors** — AI-generated commands with spurious `visible:`/`notVisible:` as top-level sibling keys (e.g. `{openLink: "url", visible: "text"}`) are now split into two proper commands: the original + an `extendedWaitUntil`. Applied in both CLI and web backend flow generation/regeneration paths.
- **Regex fix for selectorVisible patterns** — now matches commands with values on the same line (e.g. `- openLink: url\n  visible:` was previously missed).
- **Auto-fix "Apply Fix" button now enables Save** — uses the same code path as manual editing so the save button activates correctly.

## [0.25.3] - 2026-02-13

### Fixed
- **YAML validation auto-fix** — when flow validation fails due to AI-generated patterns (e.g. `visible:` attached to `openLink` or blank lines in command blocks), the validator now runs `normalizeFlowYAML` and offers an "Apply Fix" button that replaces the editor content with the corrected YAML. Fixes the recurring "mapping values are not allowed in this context" error.

## [0.25.2] - 2026-02-12

### Added
- **YAML validation debug console** — when a flow fails validation in the test plan editor, a collapsible debug panel shows the raw content, parsed metadata/commands sections, separator detection, and all errors/warnings. Includes a "Copy" button that formats all debug context for easy sharing.

## [0.25.1] - 2026-02-12

### Fixed
- **AnalysisList URL overflow** — game URLs in the analyses table are now truncated with a max column width, with a tooltip showing the full URL and a copy button on hover.

## [0.25.0] - 2026-02-12

### Added
- **ShimmerButton component** (`ui/shimmer-button/`) — animated shimmer effect button from Inspira UI, used for primary CTAs in Dashboard onboarding and empty states.
- **DataTable `#empty` slot** — rich empty states with icons and CTA buttons, replacing plain text fallback.
- **DataTable `meta.class` support** — column definitions can now specify `meta: { class: 'hidden sm:table-cell' }` for responsive column hiding.
- **Reports list view** — grid/list view toggle on the Reports page with preference persisted via `useStorage('reports-view')`. List view uses DataTable with sortable columns.
- **AnalysisList auto-refresh** — analyses list now auto-polls every 5 seconds when any analysis has `running` status, using `useIntervalFn`.

### Changed
- **Dashboard recent tests** — replaced hand-rolled `<Table>` with `<DataTable>` component using TanStack column definitions, sortable headers, responsive column hiding, tooltip timestamps, and rich empty state with CTA.
- **AnalysisList** — replaced hand-rolled `<div>` list with `<DataTable>` component with sortable columns (Game, Status, Framework, Flows, Created, Actions), responsive column hiding, and rich empty state.
- **Dashboard.vue** — replaced manual `setInterval`/`clearInterval` with `useIntervalFn` from VueUse (auto-cleanup on unmount).
- **Tests.vue** — replaced manual debounce (`setTimeout`/`clearTimeout` + watcher) with `refDebounced` from VueUse.
- **Flows.vue** — switched from custom `useClipboard` to VueUse's `useClipboard`, added `refDebounced` for search filtering.
- **Sidebar.vue** — sidebar collapsed state now persists across page loads via `useStorage('sidebar-collapsed')`.
- **Analyze.vue** — analysis form preferences (agent mode, modules, viewport, profile) now persist across visits via `useStorage`. Reset ("Analyze Another") preserves user preferences instead of reverting to defaults.
- **Clipboard composable** — all consumers (Analyze.vue, Flows.vue, TestFlowsTab.vue) switched from custom `useClipboard` to VueUse's `useClipboard({ copiedDuring: 2000 })`.

### Removed
- **Custom `useClipboard.js` composable** — deleted in favor of VueUse's built-in `useClipboard`.

## [0.24.1] - 2026-02-12

### Fixed
- **Maestro YAML fixing** — `fixCommandData()` no longer replaces the entire command with the `visible` text when other keys (e.g. `point`) exist. `tapOn: {point: "88%,5%", visible: "Game Rules"}` now correctly becomes `tapOn: {point: "88%,5%"}` instead of `tapOn: "Game Rules"`.
- **Blank lines in YAML** — new `stripInvalidVisibleLines()` pre-pass removes blank lines from the commands section (which break YAML mapping blocks) and strips `visible:`/`notVisible:` lines from non-`extendedWaitUntil` command blocks.
- **AI prompt** — added explicit WRONG/RIGHT examples for `tapOn` with `point` + `visible` to reduce AI generation of invalid Maestro YAML.

## [0.24.0] - 2026-02-12

### Added
- **UI library integration** — integrated Inspira UI, Echo Editor (Tiptap), TanStack Table, Vee-Validate + Zod, date-fns, and VueUse Motion into the Vue frontend.
- **Textarea component** (`ui/textarea/`) — reusable Textarea matching the existing Input pattern, replacing inline `<textarea>` elements across ProjectForm, NewTestPlan, and EditTestPlan.
- **Form validation components** (`ui/form/`) — Vee-Validate wrappers (FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription) providing schema-driven validation with Zod and inline error display.
- **DataTable component** (`ui/data-table/`) — TanStack Table wrapper with sortable column headers, global filtering, and row selection, integrated with existing Table UI primitives.
- **NumberTicker component** (`ui/number-ticker/`) — animated counting component using `@vueuse/motion`, replaces hand-rolled requestAnimationFrame animation in StatCard.
- **AnimatedGradientText component** (`ui/animated-gradient-text/`) — shimmer gradient effect used for brand text on the Login page.
- **Zod form schemas** (`lib/formSchemas.js`) — centralized validation schemas for login, registration, project, and test plan forms.

### Changed
- **Tests.vue** — refactored test results table from manual sorting/selection/filtering logic to TanStack Table with `useVueTable()`, column definitions via `createColumnHelper()`, and `FlexRender`-based rendering. Supports sorting, global search filtering, and checkbox row selection.
- **Login.vue** — replaced individual refs with `useForm()` + Zod schema validation, added FormField/FormItem/FormMessage wrappers for inline error display, and replaced plain brand text with AnimatedGradientText.
- **ProjectForm.vue** — replaced reactive form with `useForm()` + Zod validation for name and gameUrl fields, added Echo Editor for rich text description editing.
- **NewTestPlan.vue** — wrapped Details step fields with `useForm()` + `testPlanDetailsSchema`, replaced `detailsValid` computed with `meta.valid`, and swapped inline textarea with Textarea component.
- **EditTestPlan.vue** — replaced inline `<textarea>` with Textarea component.
- **StatCard.vue** — replaced ~30-line manual requestAnimationFrame animation with NumberTicker component.
- **dateUtils.js** — rewritten with date-fns (`format`, `formatDistanceToNowStrict`), same API signatures, zero consumer changes.
- **tailwind.config.js** — added `@inspira-ui/plugins` plugin.
- **main.js** — registered MotionPlugin and EchoEditor as global Vue plugins.

## [0.23.6] - 2026-02-12

### Fixed
- **Generated flow YAML produces invalid syntax errors** — replaced hand-rolled `commandToYAML` string formatting with `fixCommandData` + `yaml.Marshal` in both `executor.go` and `analyzer.go`. This fixes: unescaped quotes in string values breaking YAML, embedded newlines splitting values across lines, `[]interface{}` sub-values rendering as Go debug strings (`[map[key:value]]`), and `visible`/`notVisible` fields leaking into commands like `tapOn` and `assertVisible` where they aren't valid. The new `fixCommandData` function fixes AI mistakes at the data level (alias translation, structure flattening, newline stripping) before delegating to Go's YAML library for correct serialization.

## [0.23.5] - 2026-02-12

### Fixed
- **Test plan editor "flow not found" after deploy** — analysis-linked test plans failed to load flows in the editor when generated flow files were lost on Fly.io's ephemeral storage after redeploy. The editor now automatically regenerates flows from the stored analysis result when they're missing, using the same `regenerateFlowsFromAnalysis` fallback already used by the test runner.

## [0.23.4] - 2026-02-12

### Added
- **Maestro YAML validation in test plan editor** — per-flow "Validate" button and "Validate All Flows" button in the Flows tab. Strict validation against Maestro CLI rules: checks YAML syntax, metadata/commands structure (`---` separator), allowed commands (19 core + extended set), command-specific argument validation (tapOn selectors, extendedWaitUntil requires visible/notVisible, openLink format, runFlow file extension, scroll direction, repeat structure), and detects deprecated command aliases (waitFor, screenshot, openBrowser). Errors and warnings displayed inline with color-coded icons per flow tab. New `POST /api/flows/validate` endpoint.

## [0.23.3] - 2026-02-12

### Added
- **Test Plan Editor** — full editing of test plans from the Tests view. Click any test plan row to open a dedicated editor page with metadata editing (name, description, game URL), a CodeMirror YAML editor for each flow, and a key/value variables editor. Changes are tracked with dirty detection and saved via `PUT /api/test-plans/{id}`. Flow YAML content is written directly back to disk. Auto-saves before running. New backend methods `UpdateTestPlan` and `SaveFlowContent` with path-traversal protection.

## [0.23.2] - 2026-02-12

### Fixed
- **Maestro flows failing — `visible`/`notVisible` leaking into element selectors** — the AI sometimes applied `extendedWaitUntil`'s `{visible: "..."}` / `{notVisible: "..."}` syntax to `tapOn`, `assertVisible`, `assertNotVisible`, and other commands, causing Maestro to reject flows with "Unrecognized field" errors. Generalized the existing `tapOn`-only defense to catch `visible`/`notVisible` in **all** selector commands:
  - **`commandToYAML()`** now flattens `{visible: "..."}` and `{notVisible: "..."}` from any command except `extendedWaitUntil` (previously only handled `tapOn` + `visible`).
  - **`normalizeFlowYAML()`** regex safety net expanded from `tapOnVisibleRegex` to two general regexes (`selectorVisibleRegex` / `selectorNotVisibleRegex`) covering all commands.
  - **AI prompt** strengthened to explicitly prohibit `visible`/`notVisible` on `tapOn`, `assertVisible`, `assertNotVisible`, and all other commands.
  - **Validator** now warns on `visible`/`notVisible` fields in `assertVisible`, `assertNotVisible`, and `tapOn` (added `notVisible` warning for `tapOn`).

## [0.23.1] - 2026-02-11

### Fixed
- **Analysis page stuck after exploration — missed WebSocket events** — the analysis page would get stuck showing exploration as complete but never transitioning to results when the WebSocket connection dropped during long silent periods (e.g., 3-minute synthesis). Three fixes:
  - **Server-side WebSocket pings** — `writePump()` now sends pings every 30 seconds with a 60-second pong deadline, keeping connections alive through Fly.io's idle timeout.
  - **Client-side status polling fallback** — `useAnalysis.js` now polls the analysis status API every 15 seconds during active states, catching any missed `analysis_completed` or `analysis_failed` WebSocket events.
  - **WebSocket listener error isolation** — `_emit()` now wraps each listener callback in its own try/catch, preventing one failing listener from blocking others. Parse errors and listener errors are now logged with distinct, accurate messages.

## [0.23.0] - 2026-02-11

### Added
- **Delete test results** — individual and bulk delete support for test results. New `DELETE /api/tests/{id}` and `POST /api/tests/delete-batch` endpoints. Per-row trash icon button and multi-select checkboxes with "Delete Selected" bulk action bar in the Tests view.
- **Click test result navigates to detail view** — clicking a test result row now navigates to the full `TestRunDetail` page (`/tests/run/:id`) with phase timeline, flow results, duration badges, stats strip, and logs, replacing the previous basic Sheet sidebar.

## [0.22.0] - 2026-02-11

### Added
- **Token usage & cost estimation** — the analysis pipeline now tracks cumulative token consumption (input, output, cache creation, cache read) across all API calls and emits a `cost_estimate` progress event at the end of every analysis. Both CLI and web frontend display the summary (e.g., "Tokens: 45000 in + 8000 out = 53000 total (12000 cached) | Est. cost: $0.2550 (3 API calls)"). Pricing tables included for Claude Sonnet 4.5, Claude Haiku 4.5, and Gemini models.

## [0.21.5] - 2026-02-11

### Fixed
- **CLI-generated Maestro flows have invalid YAML (`visible` field, bad structure)** — The CLI path (`WriteFlowsToFiles`) was missing the YAML normalization that the web backend already had. Added `tapOn: {visible}` flattening and command alias translation (`waitFor`→`extendedWaitUntil`, etc.) to the CLI's `commandToYAML()`, and added a `normalizeFlowYAML()` safety net with regex-based fixes (openLink object syntax, tapOn visible syntax, bare `visible:` lines, timeout-only extendedWaitUntil blocks) applied before writing flow files to disk.

## [0.21.4] - 2026-02-11

### Fixed
- **All Maestro flows failing with "Unknown option: '--no-shard'"** — Maestro 2.1.0 doesn't have a `--no-shard` flag; sharding is opt-in via `-s`/`--shards`, so the default already runs single-device. Removed the invalid flag from `RunFlow()` and `ValidateFlow()`.

## [0.21.3] - 2026-02-11

### Fixed
- **cache_control accumulation still broken past step 4** — the v0.21.2 cleanup in `addConversationCacheBreakpoint()` only handled `map[string]interface{}` blocks inside `[]interface{}` content, but agent tool results are stored as `ToolResultBlock` value types. Added a `ToolResultBlock` type assertion to the cleanup loop so `CacheControl` is properly nilled on those blocks.
- **Agent API errors not visible in progress log** — errors from the AI API call in the agent loop were returned but never emitted via the progress callback. Added `progress("agent_error", ...)` before the error return so failures appear in the live progress log.

## [0.21.2] - 2026-02-11

### Fixed
- **Agent exploration fails at step 4+ with "maximum of 4 blocks with cache_control"** — `addConversationCacheBreakpoint()` was adding `cache_control` to the second-to-last user message on each `CallWithTools` call but never removing it from previous messages, causing stale markers to accumulate past the Anthropic API limit of 4. Added a cleanup pass that strips `cache_control` from all user messages before adding the new one.

## [0.21.1] - 2026-02-11

### Fixed
- **All Maestro flows failing with "Not enough devices connected to run the requested number of shards"** — Maestro's `test` command defaults to sharding mode, which requires device orchestration. Added `--no-shard` flag to both `RunFlow()` and `ValidateFlow()` in `pkg/maestro/executor.go` for single-device execution.
- **Maestro flows failing with "Unrecognized field 'visible'" on tapOn commands** — the AI sometimes generates `tapOn: {visible: "text"}` (extendedWaitUntil syntax) instead of `tapOn: "text"`. Fixed across 4 layers:
  - **`commandToYAML()`** (`web/backend/executor.go`): flattens `tapOn: {visible: "..."}` → `tapOn: "..."` during YAML serialization, matching the existing `openLink` flattening pattern.
  - **`normalizeFlowYAML()`** (`web/backend/executor.go`): added regex to catch multi-line `tapOn:\n  visible: "text"` in raw YAML loaded from disk.
  - **Validator** (`pkg/flows/validator.go`): warns when `tapOn` contains a `visible` key, which is not a valid tapOn selector.
  - **AI prompt** (`pkg/ai/types.go`): added explicit rule forbidding `{visible: "..."}` with `tapOn`.

## [0.21.0] - 2026-02-11

### Optimize Agent Analysis Step Duration

#### Changed
- **Prompt caching on conversation messages** — added `cache_control: ephemeral` to the second-to-last user message in `CallWithTools()`, caching the entire conversation prefix across turns. After the first couple of steps, 80-90% of input tokens hit cache (~10x faster processing), reducing per-step API time from 60-120s to ~10-20s.
- **Reduced screenshot quality from 40 to 25** — WebP quality lowered in `CaptureScreenshot()`. AI vision works fine at q25 and images are ~40-50% smaller, reducing token count and transfer time per step.
- **Batch wait+screenshot prompt instruction** — added rule 7 to the agent system prompt instructing the AI to combine `wait` and `screenshot` tool calls in a single response, eliminating ~30-50% of redundant LLM round-trips for the common wait-then-screenshot pattern.
- **Cache token logging** — `CallWithTools` log line now includes `cache_creation` and `cache_read` token counts for monitoring cache hit rates.

#### Added
- **LLM thinking time tracking** — each agent step now records `thinkingMs` (time spent in the `CallWithTools()` API call). Displayed in the AgentExplorationPanel as a brain icon pill (`AI: 1m 30s`) next to each step, replacing the implicit gap indicator when available.
- **`thinking_ms` column** in `agent_steps` database table for persisting thinking time.
- **`CacheControl` field on `ToolResultBlock`** — enables per-message cache breakpoints for Anthropic prompt caching.
- **Cache stats in `ToolUseResponse.Usage`** — `CacheCreationInputTokens` and `CacheReadInputTokens` fields for monitoring cache effectiveness.

## [0.20.0] - 2026-02-11

### Fixed
- **Re-run button broken after page refresh** — `planId` was not persisted to the database with test results, so after navigating away and returning to a completed test, the Re-run button had no plan to re-run. Added `plan_id` column to `test_results` table, included it in INSERT/SELECT queries, and returned it in the completed-test API response.
- **`parseOutput()` overrides exit-code-based status incorrectly** — text matching for "PASSED", "FAILED", and "timeout" in Maestro output could match flow names or YAML content, overriding the correct exit-code-based status. Removed the unreliable text-matching logic; exit code is now the sole authority for pass/fail.
- **Plans list not updating when test starts** — the Tests.vue page only listened for `test_completed`/`test_failed` WebSocket events but not `test_started`, so the plan status wouldn't show "running" until the test finished. Added `test_started` WS listener to update plan status immediately.
- **Re-run button visible but silently fails when no planId** — the Re-run button was shown for all completed/failed tests even when `planId` was empty, and the `catch` block was empty. Now hidden when `planId` is missing and shows an alert on failure.
- **Test detail sheet silently swallows fetch errors** — `openDetail()` caught fetch errors and fell back to the summary object without any indication. Now shows a warning alert when full details couldn't be loaded.
- **Success rate displayed with excessive decimals** — values like `66.66666666666667%` were shown in the UI. Now rounded to the nearest integer with `Math.round()`.
- **Log truncation invisible to users** — when logs exceeded 500 lines, old lines were silently dropped. Now inserts a `[Truncated: showing last 500 lines]` indicator at the top.
- **Reconnect failure incorrectly marks test as "failed"** — when reconnecting to a test that no longer exists (e.g. after server restart), the status was set to "failed" with no explanation. Now adds an explanatory error message to the logs.
- **Phase detection allows backward transitions** — log lines matching "preparing" or "loading" could regress the phase from "executing" back to "preparing". Phase transitions are now forward-only using a phase order map.
- **Completed test response missing `totalFlows`** — the `handleGetLiveTest` endpoint omitted `totalFlows` for completed tests, causing the frontend to show 0 pending flow slots. Now returns `len(test.Flows)`.

## [0.19.6] - 2026-02-11

### Fixed
- **All Maestro flows failing with `extendedWaitUntil expects either visible or notVisible`** — the AI generates `extendedWaitUntil` with only `timeout` and no condition, but Maestro requires `visible` or `notVisible` (timeout is just the max wait duration, not a standalone "sleep"). Fixed across 4 layers:
  - **AI prompt** (`pkg/ai/types.go`): added `notVisible` variant to command reference and two explicit rules forbidding timeout-only usage.
  - **Example template** (`flows/templates/example-game.yaml`): replaced `visible: true` (boolean) with `visible: "Start Game"` (string) and added `visible:` conditions to 3 timeout-only blocks.
  - **Validator** (`pkg/flows/validator.go`): `extendedWaitUntil` with no `visible`/`notVisible` is now an error (was a warning that allowed timeout-only). Removed invalid `text` field check.
  - **Runtime fix for DB-stored flows**: `normalizeFlowYAML()` strips timeout-only `extendedWaitUntil` blocks via regex; `commandToYAML()` in both `executor.go` and `analyzer.go` skips `extendedWaitUntil` maps missing `visible`/`notVisible`.

## [0.19.5] - 2026-02-11

### Fixed
- **All Maestro flows failing with "exit status 1" due to invalid `openBrowser` command** — `openBrowser` is not a valid Maestro command; the correct command is `openLink`. Renamed in the AI prompt template, `flowToYAML()` serializer, `normalizeFlowYAML()` regex, `injectAppId()` detection, `commandToYAML()` flatten logic, and the flow validator's allowed commands. Added `"openBrowser": "openLink"` to `maestroCommandAliases` for runtime fix of flows already stored in the DB.
- **Maestro error details swallowed — only "exit status 1" shown** — `pkg/maestro/executor.go` now includes stderr content in `result.Error` when non-empty, surfacing Maestro's actual parsing/validation errors instead of the generic Go exit code.
- **Example template using invalid commands** — `flows/templates/example-game.yaml` still used `waitFor` and `captureScreenshot`; updated to `extendedWaitUntil` and `takeScreenshot`.

## [0.19.4] - 2026-02-11

### Fixed
- **Maestro flows failing with "Unrecognized field" errors for `waitFor` and `screenshot`** — the AI prompt taught two invalid Maestro command names (`waitFor` → should be `extendedWaitUntil`, `screenshot` → should be `takeScreenshot`). Fixed the `FlowGenerationPrompt` template, the validator's allowed commands map (also fixed `captureScreenshot` → `takeScreenshot` and added missing `openBrowser`), and added runtime command name aliasing in `normalizeFlowYAML()` and `commandToYAML()` to fix flows already stored in the DB.

## [0.19.3] - 2026-02-11

### Fixed
- **Flow names in test results showing "1", "2", "3" instead of real names** — `extractFlowNameAndDuration()` used `strings.IndexAny(line, "(.")` which matched the `.` in `"1. 00-setup.yaml"` at index 1, truncating to just the number. Replaced with a `leadingNumberRegex` to strip the number prefix and `TrimSuffix` to remove `.yaml`/`.yml` extensions. Flow results now show actual names like "00-setup".
- **Regenerated flows missing `appId`, `url`, and `---` separator** — `regenerateFlowsFromAnalysis()` only wrote `tags:` metadata from stored JSON, ignoring `appId` and `url`. The `---` separator was only emitted inside the tags block, so tagless flows got no separator between metadata and commands (invalid Maestro YAML). Now writes all metadata fields and emits `---` when any metadata is present.
- **`injectAppId` producing invalid YAML when content has no metadata section** — when content was pure commands (no existing `---` separator), `injectAppId` prepended `appId:` without a `---` separator. Now detects missing separator and injects `appId: com.android.chrome\n---\n`.
- **Duration regex not matching minute-format durations** — Go's `Duration.String()` produces `1m5.5s` for durations > 1 minute, but the regex only matched `ms`/`s` units. Expanded to handle compound durations like `1m5.5s` and `1h2m3s`.

## [0.19.2] - 2026-02-11

### Fixed
- **Test flows using `runFlow` missing `appId`, failing Maestro YAML parsing** — `injectAppId()` only recognized `openBrowser:` as a web flow marker, so the 5 test flows that use `runFlow: 00-setup.yaml` (instead of `openBrowser:` directly) were missing `appId: com.android.chrome`. Maestro requires `appId` in every flow file, not just the setup flow. Fixed both `injectAppId()` in `executor.go` and `flowToYAML()` in `analyzer.go` to also detect `runFlow:` commands as web flow indicators.

## [0.19.1] - 2026-02-10

### Fixed
- **Maestro flows failing due to missing `appId`** — all generated web flows failed with `Instantiation of YamlConfig value failed for JSON property appId due to missing (therefore NULL) value`. Maestro requires `appId` in every flow's YAML metadata. Added `injectAppId()` normalizer in `prepareFlowDir` and `regenerateFlowsFromAnalysis` to inject `appId: com.android.chrome` for web flows at execution time. Also updated `flowToYAML` to auto-set appId when a flow uses `openBrowser`, and updated the AI prompt example to include appId for future generations.

## [0.19.0] - 2026-02-10

### Added
- **Rich test execution detail page** — replaced the basic Sheet-based execution view with a full-page detail view at `/tests/run/:testId`. Features a gradient header with animated status icon and elapsed timer, a 3-phase vertical timeline (Preparing → Executing → Results) with animated nodes, per-flow result cards with duration badges, a 4-column stats strip (Total/Passed/Failed/Pass Rate), and collapsible color-coded logs. Design modeled after `AnalysisProgressPanel.vue`.
- **Reconnection support** — navigating away from a running test and returning (or clicking a running plan row) restores full live progress via the new `GET /api/tests/:id/live` endpoint, which returns in-memory state for running tests or completed results from the database.
- **Flow duration extraction** — `parseFlowLine()` now extracts per-flow duration from CLI output (e.g. `(234ms)`), populating `FlowResult.Duration` and including it in `test_progress` WebSocket events.
- **Total flow count** — `test_started` WebSocket event now includes `totalFlows` (counted from flow files in the run directory), enabling the frontend to show pending flow slots and accurate progress fractions.
- **Clickable running plan rows** — plan rows with `status === 'running'` now show a spinning indicator and are clickable, navigating to the live execution detail page. A "View" button also appears in the actions column.

### Changed
- **Test plan "Run" navigates to detail page** — clicking "Run" on a test plan now navigates to `/tests/run/:testId` instead of opening a Sheet.
- **Removed execution Sheet** — the `TestExecutionPanel.vue` component and its Sheet wrapper in `Tests.vue` have been removed in favor of the new full-page view.

## [0.18.6] - 2026-02-10

### Fixed
- **Regenerated flow YAML uses wrong format, breaking all test plans** — `regenerateFlowsFromAnalysis()` used `yaml.Marshal()` which produces standard Go YAML (unquoted special chars, float64 numbers, nested openBrowser objects, `comment:` keys instead of `#` comments). Replaced with a ported `commandToYAML()` that produces Maestro-compatible YAML matching the original serializer in `pkg/ai/analyzer.go`.

## [0.18.5] - 2026-02-10

### Fixed
- **Generated flows lost after container redeploy** — when the `generated/` directory is missing (e.g. ephemeral container storage), `prepareFlowDir()` now regenerates YAML flow files on-the-fly from the analysis result stored in the database, so test plans remain runnable without re-analyzing.

## [0.18.4] - 2026-02-10

### Fixed
- **Maestro flows fail with invalid YAML syntax** — `openBrowser` was generated as an object (`openBrowser: {url: "..."}`) but Maestro expects a simple string (`openBrowser: "..."`). Fixed the AI prompt, the `commandToYAML()` serializer, and added runtime normalization in `prepareFlowDir()` to fix existing flows on disk.

## [0.18.3] - 2026-02-10

### Fixed
- **Test plans not visible after analysis** — clicking "View Test Plan" after an analysis now navigates to `/tests?tab=plans`, pre-selecting the "Test Plans" tab instead of landing on the empty "Test Results" tab. Additionally, when navigating to the Tests page without a tab parameter, the UI auto-switches to the "Test Plans" tab if no test results exist but plans are available.

## [0.18.2] - 2026-02-10

### Added
- **Visible "Creating test plan" progress phase** — after flows are generated, a dedicated "Creating test plan" phase now appears in the progress timeline with sky-blue coloring and a ClipboardCheck icon. The backend emits granular sub-step progress events (`test_plan`, `test_plan_checking`, `test_plan_flows`, `test_plan_saving`, `test_plan_done`) so users see live messages like "Checking for existing test plan...", "Found N flow files: ...", "Saving test plan: GameName - Test Plan", and "Test plan created: GameName - Test Plan (N flows)". Sub-detail chips show flow count, individual flow names, and game name.

## [0.18.1] - 2026-02-10

### Added
- **Detailed progress during flow generation** — the "Generating test flows" phase now emits granular sub-step progress events (`flows_prompt`, `flows_calling`, `flows_parsing`, `flows_validating`) instead of showing only "Working..." until completion. Users see live messages like "Built prompt from N scenarios", "Sending to AI for flow generation", "Parsing AI response", and "Validated N flows from structured JSON".
- **Scenario names in flow generation message** — the initial "Converting N scenarios to Maestro flows" progress message now lists scenario names (e.g., "Converting 5 scenarios to Maestro flows: Login, Tutorial, Combat, Inventory, Settings").
- **Flow generation sub-detail chips** — the progress panel shows scenario count and individual scenario names as sub-detail chips during the flows phase.

## [0.18.0] - 2026-02-10

### Fix Test Plan Execution & Auto-Create from Analysis

#### Fixed
- **Flow name mismatch in test plans** — `prepareFlowDir()` now has a fast path for analysis-linked plans that copies flows directly from `generated/{analysisID}/` instead of matching by template name globally. This eliminates cross-analysis name collisions and ensures flows are always found when running a plan.
- **Flow pre-selection in NewTestPlan** — when navigating from an analysis, flow names are now fetched via `GET /api/analyses/{id}/flows` (filename-based names matching `ListTemplates()`), fixing the mismatch where human-readable names were passed but filename-based names were expected.

#### Added
- **Auto-create test plan on analysis completion** — when an analysis completes with generated flows, a draft test plan is automatically created and linked to the analysis via the new `analysis_id` column on `test_plans`. Idempotent: re-running an analysis for the same game does not create duplicate plans.
- **`analysis_id` column on test_plans** — links a test plan to its source analysis for direct flow resolution and idempotency.
- **`GET /api/analyses/{id}/flows` endpoint** — returns generated flow filenames for an analysis.
- **`testPlanId` in analysis responses** — `GET /api/analyses/{id}` and the `analysis_completed` WebSocket event now include the linked test plan ID when one exists.
- **"View Test Plan" button** — on the Analyze results page, shows "View Test Plan" when an auto-created plan exists, otherwise "Create Test Plan".
- **`GetTestPlanByAnalysis()` and `ListGeneratedFlowNames()` store methods** — support idempotency checks and flow enumeration for auto-creation.

## [0.17.1] - 2026-02-10

### Added
- **Resume running analysis from analyses list** — clicking a running analysis in the Analyses list now navigates to the Analyze page with full live progress (progress panel, agent exploration timeline, WebSocket events) instead of the detail view. `tryRecover()` accepts an explicit analysis ID to fetch state from the API, and persists to localStorage for page-refresh recovery. Directly navigating to `/analyses/:id` for a running analysis also redirects to the live progress view.

## [0.17.0] - 2026-02-10

### Changed
- **Redesigned agent exploration panel with rich timing display** — `AgentExplorationPanel.vue` overhauled with: SVG circular progress ring on the steps counter, primary-colored elapsed timer, computed avg/step stat in the banner; 5-column stats strip adding "Total Time" (sum of all step durations); per-step gap indicator showing AI "thinking time" between steps with color-coded thresholds (green < 2s, amber 2–5s, red > 5s), duration pills with speed-based coloring (green < 500ms, amber 500ms–2s, red > 2s), and cumulative `@M:SS` timestamps from analysis start; mini-map tooltips now include step duration; completion footer shows rich summary with avg, fastest, and slowest step times.

## [0.16.1] - 2026-02-10

### Fixed
- **Agent steps lost on reconnect to running analysis** — when navigating away from the Analyze page and returning during a running analysis, the UI now loads all persisted agent steps from the server, restoring the exploration timeline, screenshots, and step counter. Previously, `tryRecover` only reconnected the WebSocket for future events, leaving the timeline empty.
- **Duplicate agent steps on reconnect** — incoming WebSocket `agent_step_detail` events are now deduplicated against steps already loaded from the database, preventing duplicate entries in the timeline after reconnect.
- **Recovered step screenshots not displaying** — `AgentExplorationPanel` thumbnails and the full-size screenshot dialog now accept URL-based screenshots (`screenshotUrl`) in addition to base64 (`screenshotB64`), so screenshots loaded from the API render correctly.

## [0.16.0] - 2026-02-10

### Performance Optimization — Faster Screenshots, Prompt Caching, Auto-Screenshots, Device Viewports & Reduced Latency

#### Changed
- **Auto-screenshots on state-changing tools** — `click`, `type_text`, and `scroll` tools now automatically capture and return a screenshot after each action, eliminating the need for the AI to call `screenshot` separately. This cuts ~30-50% of exploration steps. The `screenshot` tool remains available for passive observation.
- **Anthropic prompt caching** — system prompt and tool definitions are now sent as cacheable content blocks with `cache_control: ephemeral`, reducing API latency by ~20% and cached token cost by ~80% on turns 2+.
- **Halved tool execution sleeps** — `click` reduced from 500ms to 250ms, `type_text` from 200ms to 100ms, `scroll` from 300ms to 150ms, saving ~5 seconds over a 20-step exploration.
- **Screenshot pruning reduced from 4 to 2** — with auto-screenshots providing more frequent visual feedback, keeping only the 2 most recent screenshots in conversation is sufficient, reducing API payload by ~200KB per call.
- **Adaptive canvas polling** — replaced fixed 500ms polling intervals with exponential backoff (100ms→150ms→225ms→337ms→500ms). Most games are ready within 2-3 iterations, saving 1-2 seconds on startup.
- **Shorter WaitIdle** — reduced from 5 seconds to 3 seconds across all three call sites (Navigate helper, ScoutURLHeadlessKeepAlive, ScoutURLHeadless).
- **Default viewport changed to 1280x720** — smaller viewport produces ~40% smaller screenshots, reducing API transfer size and encode time. The old 1920x1080 is still available as the "Desktop HD" preset.
- **Dynamic coordinate system in agent prompt** — the `COORDINATE SYSTEM` line in the agent system prompt now uses the actual viewport dimensions instead of hardcoded 1920x1080.
- **Dynamic tool descriptions** — `click` tool description now shows the actual viewport dimensions (e.g., "viewport is 1280x720") instead of hardcoded values.

#### Added
- **Device viewport presets** — 30 device presets across 4 categories (Desktop, iOS, Android tablets/phones) selectable from the Analyze page. Presets include accurate viewport dimensions and devicePixelRatio for realistic device emulation.
- **`--viewport` CLI flag** — select a device preset by name (e.g., `--viewport iphone-16-pro`) to set viewport dimensions and DPR.
- **Frontend device selector** — dropdown on the Analyze page showing all presets grouped by category with dimensions preview.
- **`DevicePixelRatio` in HeadlessConfig** — passed through to Chrome's `MustSetViewport` for accurate DPR emulation.
- **`ViewportWidth`/`ViewportHeight` in AgentConfig** — used by `BrowserTools()` and `BuildAgentSystemPrompt()` for dynamic descriptions.
- **`pkg/scout/viewports.go`** — Go-side viewport presets lookup table.
- **`web/frontend/src/lib/viewports.js`** — frontend viewport presets with category grouping.

## [0.15.0] - 2026-02-09

### Optimize Agent Exploration — WebP Screenshots, Faster Capture & Smarter Adaptive Expansion

#### Changed
- **WebP screenshots** — all three screenshot call sites (agent tool, initial scout, multi-screenshot capture) switched from JPEG to WebP format. WebP delivers better quality-per-byte, reducing screenshot payload sizes by ~30-50%.
- **Lowered screenshot quality** — agent tool screenshots reduced from quality 50 to 40, initial/multi-screenshot captures from 80 to 60. Combined with WebP, this significantly reduces encode time and API transfer size.
- **Aggressive adaptive prompts** — `AdaptiveExplorationPromptSuffix` and `DynamicTimeoutPromptSuffix` rewritten with concrete triggers (70% step budget threshold, 50% time budget threshold) and an assessment checklist the AI must run every 3 steps, replacing vague "if approaching your limit" language.
- **Budget status injection** — every 5 steps during adaptive exploration, a `[SYSTEM STATUS]` message is injected into the conversation with concrete step count and time remaining, enabling data-driven extension requests.
- **Raised profile defaults** — balanced profile now starts with 20 steps (was 15) with adaptive exploration enabled (was disabled), extending up to 25 steps and 20 minutes; thorough starts at 25 steps (was 20) extending to 50 (was 35); maximum starts at 30 (was 25) extending to 70 (was 50).
- **Raised timeout clamps** — CLI max clamp raised from 30→45 minutes (default) and 45→60 minutes (adaptive); `agentTotalTimeout` formula updated to `steps × 60s + 7min` clamped to 30 minutes (was `steps × 30s + 5min` clamped to 20 minutes).
- **Screenshot Content-Type** — server now detects `.webp` vs `.jpg` file extension for correct Content-Type header, maintaining backward compatibility with existing JPEG screenshots.

## [0.14.4] - 2026-02-09

### Changed
- **Redesigned analysis progress panel** — extracted `AnalysisProgressPanel.vue` to replace both the progress and error states with a unified polished component featuring: gradient header banner with animated status icon and segmented progress bar, vertical timeline with colored phase-specific nodes (Radar/Bot/Brain/ListTree/PlayCircle), duration badges, sub-detail chips with expand/collapse, collapsible line-numbered log section with color-coded entries (red errors, amber warnings) and auto-scroll, and a unified footer with phase completion count. Error mode shows destructive gradient overlay, auto-expanded logs, and Continue/Retry/Start Over buttons. Agent exploration panel renders inline via named slot on the timeline.
- **Retired `ProgressStep.vue`** — replaced by the phase timeline in `AnalysisProgressPanel.vue`.

## [0.14.3] - 2026-02-09

### Changed
- **Polished agent exploration panel** — extracted `AgentExplorationPanel.vue` from Analyze.vue with a redesigned 5-section layout: activity banner with breathing pulse animation, horizontal mini-map of color-coded step dots, live stats strip (steps/screenshots/actions/errors), vertical timeline with tool-specific icons, category-colored nodes, expandable step cards with entry animations, and a hint input bar with message icon prefix.
- **Tool classification system** — each agent tool (click, screenshot, navigate, etc.) now maps to a specific icon, color, and category (interaction/observation/navigation/meta) for visual differentiation across the mini-map, timeline nodes, and stats.

## [0.14.2] - 2026-02-09

### Changed
- **3-tier findings display** — FindingsTab redesigned with severity grouping: green "What's Working Well" (positive), amber "Suggestions" (suggestion + minor), and red "Bugs & Issues" (critical + major) sections with colored left-border accents and tier-matched badges.
- **Summary stat pills** — replaced flat severity badges with 3 clickable colored pills (Positive/Suggestions/Bugs) that double as tier filters.
- **Findings filters** — added tier toggle buttons, text search (debounced, matches description/suggestion/location), and active filter count indicator alongside the existing category dropdown.
- **Role-based action checklists** — collapsible "Action Checklist by Role" section with dynamic checkbox items for Developer, QA Engineer, Designer, and Product Manager, populated based on finding severity counts.
- **Expanded severity values** — UI/UX and Wording findings now support `positive` and `suggestion` severity in AI prompts and Go struct comments, matching Game Design findings.

## [0.14.1] - 2026-02-09

### Fixed
- **Agent screenshot performance** — switched all three screenshot call sites from full-page capture (`Screenshot(true, ...)`) to viewport-only (`Screenshot(false, ...)`) with `OptimizeForSpeed: true`. Full-page mode forced Chrome/SwiftShader to re-render the entire WebGL scene at enlarged CSS dimensions, causing each screenshot to take 60-70 seconds instead of 2-5 seconds.
- **Agent analysis timeouts** — increased timeout formula estimates (per-step from 40s→75s backend / 30s→60s CLI) and raised max clamps (backend: 30→45min default, 45→60min adaptive; CLI: 20→30min default, 30→45min adaptive) to prevent premature timeouts on complex WebGL games.
- **Synthesis reserve** — increased from 3 to 5 minutes to account for synthesis + flow generation retries.
- **Agent screenshot quality** — lowered `CaptureScreenshot()` JPEG quality from 80→50 for agent tool screenshots (used only for AI understanding, not stored as artifacts) to reduce payload size.

## [0.14.0] - 2026-02-09

### Chat-Style Agent Exploration Timeline

#### Changed
- **Live exploration panel redesigned as chat-style timeline** — the "Agent Exploring Game" panel now shows a scrollable chat-like history where each step displays its screenshot thumbnail inline, so you can scroll up and see all previous steps with their images instead of only the latest hero screenshot.
- **Screenshots stored per step** — `agent_screenshot` events now store the full base64 image data on the corresponding step object (`screenshotB64`) instead of just a `hasScreenshot` boolean flag. This enables inline thumbnail rendering for every step.
- **Reasoning text attached to steps** — `agent_reasoning` events now attach the reasoning text to the latest step, displayed inline below the tool name.
- **Taller timeline area** — timeline max height increased from 192px to 500px for comfortable scrolling through step history.
- **Click-to-expand thumbnails** — clicking any step's inline screenshot thumbnail opens the existing full-screen screenshot dialog with step details.

#### Removed
- **Hero screenshot + reasoning row** — the separate large screenshot and "Latest thinking" area above the timeline has been removed in favor of inline per-step display.

## [0.13.0] - 2026-02-09

### Agent Modules UI, Dynamic Timeout & Branching Test Flows

#### Added
- **Agent Modules UI** — dedicated "Agent Modules" section (visible when Agent Mode is enabled) with two toggles: **Dynamic Steps** (AI can request more exploration steps) and **Dynamic Timeout** (AI can extend exploration time). These replace the hidden adaptive checkbox that was previously only visible in Custom profile mode.
- **Dynamic Timeout** — new `request_more_time` pseudo-tool allows the AI agent to request additional exploration time when significant game areas remain unexplored. Controlled by `--adaptive-timeout` and `--max-total-timeout` CLI flags, with backend validation (1–60 minutes).
- **`DynamicTimeoutPromptSuffix()`** — appended to the agent system prompt when adaptive timeout is enabled, instructing the AI to proactively request more time.
- **`agent_timeout_extend` progress event** — streams timeout extension decisions to the frontend live timeline.
- **Branching test flows with `runFlow`** — generated Maestro flows now use a shared setup flow (`00-setup.yaml`) that other flows reference via Maestro's native `runFlow` command, eliminating redundant setup steps in every flow.
- **Profile-level adaptive timeout defaults** — Thorough (`adaptiveTimeout: true, maxTotalTimeout: 25`) and Maximum (`adaptiveTimeout: true, maxTotalTimeout: 40`) profiles now enable dynamic timeout by default.
- **Custom profile timeout controls** — "Max Total Timeout (minutes)" input visible in Custom mode when Dynamic Timeout is toggled on.

#### Changed
- **`AgentTools()` signature** — now accepts `AgentConfig` instead of `bool`, enabling both `request_more_steps` and `request_more_time` tools based on config.
- **`WriteFlowsToFiles()`** — setup flow is now sorted first (becomes `00-setup.yaml`), flows are 0-indexed instead of 1-indexed.
- **`FlowGenerationPrompt`** — updated with FLOW COMPOSITION instructions requiring a shared setup flow and `runFlow` references.
- **Timeout calculation** — backend process timeout accounts for `maxTotalTimeout` when adaptive timeout is enabled, with raised upper clamp (60min).
- **Profile sync** — selecting a profile now auto-syncs both Dynamic Steps and Dynamic Timeout toggles from profile defaults.

## [0.12.0] - 2026-02-09

### Adaptive Exploration — Dynamic Step Extension

#### Added
- **Adaptive exploration mode** — the AI agent can now dynamically request more exploration steps when it discovers a game needs more thorough testing, via a new `request_more_steps` pseudo-tool
- **`--adaptive` CLI flag** — enables adaptive exploration where the agent self-assesses coverage and can extend its step budget
- **`--max-total-steps` CLI flag** — hard cap on total exploration steps after adaptive extensions (prevents runaway exploration)
- **`AdaptiveExploration` and `MaxTotalSteps` fields** in `AgentConfig` — configurable adaptive exploration at the AI package level
- **`BuildAgentSystemPrompt()` function** — dynamically constructs the agent system prompt with adaptive exploration instructions when enabled
- **`AgentTools()` wrapper** — returns browser tools plus the `request_more_steps` tool when adaptive mode is active
- **`agent_adaptive` progress event** — streams adaptive extension decisions to the frontend live timeline
- **Frontend adaptive toggle** — custom profile mode exposes "Adaptive Exploration" checkbox and "Max Total Steps" field
- **Profile-level adaptive defaults** — Thorough (adaptive, up to 35 steps) and Maximum (adaptive, up to 50 steps) profiles now use adaptive exploration by default

#### Changed
- **Timeout calculation** — when adaptive mode is enabled, backend and CLI compute timeouts based on `maxTotalSteps` instead of `agentSteps`, with raised upper clamp (30min CLI, 45min backend)
- **Thorough profile description** updated to mention adaptive exploration
- **Maximum profile description** updated to mention adaptive exploration

## [0.11.0] - 2026-02-09

### Resumable Analyses — Continue from Timeout

#### Added
- **Checkpoint persistence** — the CLI writes checkpoint files after each major pipeline step (scouting, analysis/synthesis), enabling resume on failure
- **Continue Analysis button** — when an analysis times out or fails after completing intermediate steps, a "Continue Analysis" button appears that resumes from the last checkpoint instead of restarting from scratch
- **`POST /api/analyses/:id/continue` endpoint** — backend endpoint that reads stored checkpoint data, spawns a new CLI process with `--resume-from`/`--resume-data` flags, and completes the remaining pipeline steps
- **`--resume-from` / `--resume-data` CLI flags** — internal flags for the `scout` command to skip completed pipeline steps and resume from a checkpoint file
- **`partial_result` DB column** — stores checkpoint JSON on analysis failure for later resume
- **`agent_mode` and `profile` DB columns** — persist analysis configuration for accurate reconstruction on continue
- **`AnalyzeOption` pattern** in AI package — functional options (`WithResumeData`, `WithCheckpointDir`) for flexible pipeline configuration

#### Changed
- **Error handling** — on CLI failure/timeout, the backend now reads checkpoint files from tmpDir before cleanup and stores them as `partial_result`
- **Analysis list queries** — now include `partial_result` field so the frontend can determine continue eligibility
- **Retry vs Continue** — "Retry Analysis" button becomes secondary when "Continue Analysis" is available

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
