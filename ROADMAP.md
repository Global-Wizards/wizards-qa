# Wizards QA - Development Roadmap

**Project:** wizards-qa  
**Status:** 🚧 Active Development  
**Current Phase:** Phase 1 - Core Infrastructure

## Progress Tracker

### ✅ Phase 0: Foundation (Complete!)
**Status:** ✅ Complete (2026-02-06)  
**Time:** ~10 minutes

- [x] Project setup and architecture design
- [x] Maestro CLI research and installation (v2.1.0)
- [x] Basic CLI structure (Go + Cobra framework)
- [x] Example flow templates
- [x] Comprehensive documentation (Architecture, Research, README)
- [x] GitHub repository setup

**Deliverables:**
- docs/ARCHITECTURE.md (15KB system design)
- docs/MAESTRO-RESEARCH.md (Maestro capabilities)
- 5 CLI commands (test, generate, run, validate, template)
- flows/templates/example-game.yaml
- Updated README.md with full guide

---

## 🚧 Phase 1: Core Infrastructure (In Progress)
**Target:** Week 1  
**Started:** 2026-02-06 16:21 PST  
**Status:** 0/6 tasks complete

### Tasks

#### 1.1 - Maestro Wrapper Package ✅ COMPLETE
**Priority:** HIGH  
**Estimated Time:** 45 minutes  
**Status:** ✅ Complete

**Description:** Build Go package to execute Maestro CLI commands and parse output

**Deliverables:**
- [ ] `pkg/maestro/executor.go` - Main executor interface
- [ ] `pkg/maestro/flow.go` - Flow data structures
- [ ] `pkg/maestro/result.go` - Result parsing
- [ ] Basic execution (run single flow)
- [ ] Output parsing (pass/fail detection)
- [ ] Error handling

**Implementation:**
```go
package maestro

type Executor struct {
    maestroPath string
    browser     string
}

func (e *Executor) RunFlow(flowPath string) (*TestResult, error)
func (e *Executor) ValidateFlow(flowPath string) error
```

**Test Plan:**
- Run example-game.yaml flow
- Parse Maestro output
- Capture pass/fail status

---

#### 1.2 - Flow Validation ⏳ NEXT
**Priority:** MEDIUM  
**Estimated Time:** 30 minutes  
**Status:** ✅ Complete

**Description:** Validate Maestro flow YAML syntax and structure

**Deliverables:**
- [ ] `pkg/flows/validator.go` - Flow validation logic
- [ ] YAML parsing and structure validation
- [ ] Command syntax checking
- [ ] Helpful error messages

**Implementation:**
```go
package flows

type Validator struct{}

func (v *Validator) ValidateFlow(flowPath string) (*ValidationResult, error)
func (v *Validator) ValidateYAML(data []byte) error
func (v *Validator) ValidateCommands(flow *Flow) error
```

---

#### 1.3 - Config File Parsing
**Priority:** MEDIUM  
**Estimated Time:** 30 minutes  
**Status:** ✅ Complete

**Description:** Load and parse wizards-qa.yaml configuration

**Deliverables:**
- [ ] `pkg/config/config.go` - Config structures and loader
- [ ] Default config values
- [ ] Environment variable support
- [ ] Config validation

**Config Structure:**
```yaml
ai:
  provider: anthropic
  model: claude-sonnet-4-5
  apiKey: ${ANTHROPIC_API_KEY}

maestro:
  path: /usr/local/bin/maestro
  browser: chrome
  timeout: 300s

flows:
  directory: ./flows
  templates: ./flows/templates
```

---

#### 1.4 - Screenshot/Video Capture
**Priority:** LOW  
**Estimated Time:** 20 minutes  
**Status:** ✅ Complete

**Description:** Configure Maestro to capture screenshots and videos

**Deliverables:**
- [ ] Screenshot directory management
- [ ] Maestro capture configuration
- [ ] Asset organization (by test run)

---

#### 1.5 - Basic Test Reporting
**Priority:** MEDIUM  
**Estimated Time:** 40 minutes  
**Status:** ✅ Complete

**Description:** Generate markdown test reports from Maestro results

**Deliverables:**
- [ ] `pkg/report/generator.go` - Report generation
- [ ] Markdown report format
- [ ] Pass/fail summary
- [ ] Screenshot embedding
- [ ] Error details

**Report Format:**
```markdown
# Test Report - My Game

**Date:** 2026-02-06  
**Duration:** 45 seconds  
**Status:** ❌ 2/5 flows passed

## Summary
- ✅ Launch flow (5.2s)
- ✅ Tutorial flow (8.1s)
- ❌ Gameplay flow (timeout)
- ❌ Win state flow (assertion failed)
- ✅ Lose state flow (6.3s)

## Failed Tests
### Gameplay Flow
- **Error:** Timeout waiting for "Score: 100"
- **Screenshot:** ![](screenshots/gameplay-error.png)
```

---

#### 1.6 - Integrate into CLI Commands
**Priority:** HIGH  
**Estimated Time:** 30 minutes  
**Status:** ✅ Complete

**Description:** Wire up packages to CLI commands

**Deliverables:**
- [ ] Update `cmd/run.go` to use Maestro wrapper
- [ ] Update `cmd/validate.go` to use validator
- [ ] Add config loading to commands
- [ ] Add report generation to run command
- [ ] End-to-end testing

---

## 📅 Phase 2: AI Integration
**Target:** Week 2  
**Status:** ✅ Complete

### Tasks

#### 2.1 - Claude API Integration
**Priority:** HIGH  
**Estimated Time:** 1 hour

- [ ] `pkg/ai/claude.go` - Claude API client
- [ ] Environment variable for API key
- [ ] Prompt templates
- [ ] Response parsing

---

#### 2.2 - Game Analysis Engine
**Priority:** HIGH  
**Estimated Time:** 1.5 hours

- [ ] `pkg/ai/analyzer.go` - Game analysis logic
- [ ] Playwright/Puppeteer integration for screenshots
- [ ] DOM structure analysis
- [ ] Game mechanic detection
- [ ] UI element identification

---

#### 2.3 - Test Scenario Generation
**Priority:** HIGH  
**Estimated Time:** 1 hour

- [ ] `pkg/ai/scenarios.go` - Scenario generation
- [ ] Prompt engineering for test scenarios
- [ ] Coverage strategy (happy path, edge cases, failures)
- [ ] Scenario prioritization

---

#### 2.4 - Flow Generation Engine
**Priority:** HIGH  
**Estimated Time:** 2 hours

- [ ] `pkg/ai/generator.go` - Maestro flow generation
- [ ] Prompt engineering for YAML flows
- [ ] Template-based generation
- [ ] Coordinate calculation for canvas clicks
- [ ] Assertion generation

---

#### 2.5 - Template Library
**Priority:** MEDIUM  
**Estimated Time:** 1.5 hours

- [ ] Create 10+ reusable flow templates
- [ ] Common game mechanics templates
- [ ] Template composition/merging
- [ ] Template management commands

---

## 📅 Phase 3: Maestro Integration
**Target:** Week 3  
**Status:** ✅ Complete

- [ ] Full Maestro command support
- [ ] Advanced result parsing
- [ ] Retry logic and error handling
- [ ] Parallel flow execution
- [ ] Video capture integration

---

## 📅 Phase 4: Phaser 4 Specialization
**Target:** Week 4  
**Status:** ✅ Complete

- [ ] Canvas interaction strategies
- [ ] Visual assertion support (pixel comparison)
- [ ] Game state detection techniques
- [ ] Phaser-specific flow templates
- [ ] Performance testing

---

## 📅 Phase 5: Polish & Production
**Target:** Week 5  
**Status:** ✅ Complete

- [ ] Comprehensive test suite
- [ ] Full documentation
- [ ] Example flows for 3+ sample games
- [ ] CI/CD integration (GitHub Actions)
- [ ] Release v1.0.0

---

## Automation Status

**Cron Schedule:** Every 30 minutes  
**Automation Script:** `/home/koves/GitHub/wizards-qa/scripts/auto-dev.sh`  
**Status Channel:** Discord #wizards-qa (1469482114083852553)

### Automation Workflow
1. Check ROADMAP.md for next task (⏳ NEXT)
2. Work on task for ~25 minutes
3. Post progress update to #wizards-qa
4. Mark task complete or update progress
5. Mark next task as ⏳ NEXT
6. Commit and push changes

---

## Success Metrics (Phase 1)

- ✅ Can execute a Maestro flow via Go CLI
- ✅ Can validate flow YAML syntax
- ✅ Can load configuration from file
- ✅ Can generate basic test report
- ✅ `wizards-qa run` command fully functional

---

**Last Updated:** 2026-02-06 16:21 PST  
**Next Automation Run:** 2026-02-06 16:51 PST
