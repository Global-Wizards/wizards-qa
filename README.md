# Wizards QA - AI-Powered Game Testing

**Version:** 0.44.6

AI-powered QA automation system for web games. An AI agent autonomously explores your game in a headless browser, understands its mechanics, generates test scenarios, and executes them -- all from a single URL.

## Features

- **AI Agent Exploration** -- An AI agent launches a headless Chrome browser, navigates your game, and autonomously discovers mechanics, UI elements, and user flows
- **Three Click Strategies** -- CDP mouse events for canvas/WebGL games, CDP touch events for mobile viewports, JS dispatch for HTML-only apps
- **Comprehensive Synthesis** -- Produces structured analysis: game info, mechanics, UI elements, user flows, edge cases, scenarios, UIUX analysis, wording checks, game design findings, GLI compliance, and navigation maps
- **Agent Test Execution** -- AI agent executes each test scenario using browser tools and reports pass/fail with evidence
- **Checkpoint & Resume** -- Saves progress after each pipeline stage so crashed runs resume without re-doing exploration
- **Web Dashboard** -- Vue.js frontend with real-time WebSocket updates, project management, and analysis history
- **Adaptive Exploration** -- Agent can request more steps or time when it discovers unexplored areas

## Quick Start

### Prerequisites

- **Go 1.25+** -- [Install Go](https://go.dev/dl/)
- **Node.js 18+** -- For the web frontend
- **Chrome/Chromium** -- Installed or available in PATH

### Build

```bash
# Clone the repository
git clone https://github.com/Global-Wizards/wizards-qa.git
cd wizards-qa

# Build CLI and backend
make build

# Build frontend
make frontend
```

### CLI Usage

```bash
# Scout a game (explore and analyze)
./bin/wizards-qa scout --url https://your-game.com

# With AI provider selection
./bin/wizards-qa scout --url https://your-game.com --provider anthropic --model claude-sonnet-4-5

# With custom viewport
./bin/wizards-qa scout --url https://your-game.com --viewport 1280x720
```

### Web Dashboard

```bash
# Start the backend server
./bin/wizards-qa-server

# Frontend dev server (for development)
cd web/frontend && npm run dev
```

## Architecture

```
                     ┌────────────────────────┐
                     │   CLI / Web Dashboard  │
                     └───────────┬────────────┘
                                 │
                  ┌──────────────┴──────────────┐
                  │                             │
                  v                             v
          ┌──────────────┐             ┌──────────────┐
          │    Scout     │             │  Web Backend  │
          │  (Headless   │             │  (Agent Test  │
          │   Chrome)    │             │   Executor)   │
          └──────┬───────┘             └──────┬───────┘
                 │                            │
                 v                            v
          ┌──────────────┐             ┌──────────────┐
          │  AI Agent    │             │  AI Agent    │
          │  Explorer    │             │  Executor    │
          │  (13 tools)  │             │  (+ report)  │
          └──────┬───────┘             └──────────────┘
                 │
                 v
          ┌──────────────┐
          │  Checkpoint  │
          │  & Resume    │
          └──────────────┘
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full technical deep-dive.

## Project Structure

```
wizards-qa/
├── cmd/                        # CLI commands (Cobra)
│   ├── main.go                 # Entry point
│   ├── scout.go                # Scout command (explore + analyze)
│   ├── test.go                 # Test command
│   ├── generate.go             # Generate command
│   ├── run.go                  # Run command
│   └── validate.go             # Validate command
│
├── pkg/                        # Core packages
│   ├── ai/                     # AI agent, tools, synthesis, prompts
│   ├── scout/                  # Headless Chrome, click strategies, viewports
│   ├── config/                 # Configuration loading
│   ├── maestro/                # Legacy Maestro executor
│   ├── flows/                  # Flow validation and parsing
│   ├── report/                 # Test report generation
│   ├── cache/                  # In-memory caching
│   ├── retry/                  # Exponential backoff retry
│   ├── parallel/               # Concurrent execution
│   └── util/                   # Shared utilities
│
├── web/                        # Web application
│   ├── backend/                # Go HTTP server, agent executor, store
│   └── frontend/               # Vue.js dashboard
│
├── docs/                       # Documentation
│   ├── ARCHITECTURE.md         # System architecture (current)
│   └── archive/                # Historical design documents
│
├── Makefile                    # Build automation
├── Dockerfile                  # Container build
├── fly.toml                    # Fly.io deployment
└── wizards-qa.yaml.example     # Example configuration
```

## Configuration

Copy the example config and set your API keys:

```bash
cp wizards-qa.yaml.example wizards-qa.yaml
```

```yaml
ai:
  provider: anthropic          # anthropic | google
  model: claude-sonnet-4-5
  apiKey: ${ANTHROPIC_API_KEY}

browser:
  headless: true
  viewport:
    width: 960
    height: 540
```

## Development

```bash
# Run all tests
make test

# Vet all code
make vet

# Full validation (vet + test + frontend build)
make validate

# Clean build artifacts
make clean
```

## Links

- **GitHub:** https://github.com/Global-Wizards/wizards-qa
- **Architecture:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## License

MIT License - see LICENSE file for details.
