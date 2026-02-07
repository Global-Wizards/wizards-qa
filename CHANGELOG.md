# Changelog

All notable changes to wizards-qa will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
