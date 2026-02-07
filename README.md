# 🧙‍♂️ Wizards QA - AI-Powered Game Testing

**Version:** 0.1.0 (Alpha)  
**Status:** 🚧 Active Development

AI-powered QA automation system for Phaser 4 web games. Analyzes games, generates intelligent test flows, and executes comprehensive testing via Maestro CLI.

## 🎯 Vision

Send a game spec + URL → Get comprehensive automated test coverage powered by AI.

## ✨ Features

- 🤖 **AI-Powered Analysis** - Claude/Gemini analyzes game mechanics and generates test scenarios
- 📝 **Maestro Flow Generation** - Automatically creates executable test flows in YAML
- ⚡ **Automated Execution** - Runs tests via Maestro CLI with screenshot/video capture
- 📊 **Detailed Reports** - Comprehensive test results with identified issues
- 🎮 **Phaser 4 Optimized** - Canvas interaction strategies for game testing
- 🔧 **Template Library** - Reusable flow patterns for common game mechanics

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/dl/)
- **Java 17+** - Required by Maestro
- **Maestro CLI** - Installed automatically or manually

### Installation

```bash
# Clone the repository
git clone https://github.com/Global-Wizards/wizards-qa.git
cd wizards-qa

# Install dependencies
go mod download

# Build the CLI
go build -o wizards-qa ./cmd

# Verify installation
./wizards-qa --version
```

### Maestro CLI Installation

Maestro is required for test execution:

```bash
# Install via script (macOS, Linux, WSL)
curl -fsSL "https://get.maestro.mobile.dev" | bash

# Or via Homebrew (macOS)
brew tap mobile-dev-inc/tap
brew install mobile-dev-inc/tap/maestro

# Verify installation
maestro --version
```

## 📖 Usage

### Full E2E Testing

Analyze a game, generate flows, and execute tests:

```bash
./wizards-qa test \
  --game https://your-game.example.com \
  --spec game-spec.md \
  --model claude-sonnet-4-5
```

### Generate Flows Only

Create test flows without executing them:

```bash
./wizards-qa generate \
  --game https://your-game.example.com \
  --spec game-spec.md \
  --output flows/my-game/
```

### Run Existing Flows

Execute pre-generated flows:

```bash
./wizards-qa run \
  --flows flows/my-game/ \
  --report reports/test-001.md
```

### Validate Flow Syntax

Check flow file syntax before execution:

```bash
./wizards-qa validate --flow flows/my-game/gameplay.yaml
```

### Manage Templates

List and manage reusable flow templates:

```bash
./wizards-qa template --list
./wizards-qa template --show login
```

## 📋 Game Specification Format

Create a markdown file describing your game:

```markdown
# My Phaser Game

## Overview
A 2D platformer where players collect coins and avoid enemies.

## Mechanics
- Click/tap to jump
- Collect coins for points
- Avoid red enemies
- Reach the flag to win

## UI Elements
- Score display (top-left)
- Lives counter (top-right)
- Pause button (top-center)
- Start button (main menu)

## User Flows
1. Main menu → Click "Start Game"
2. Tutorial → Follow on-screen instructions
3. Gameplay → Jump, collect, avoid
4. Win state → Reach flag, see victory screen
5. Lose state → Lose all lives, see game over screen
```

## 🏗️ Architecture

```
┌─────────────────┐
│  Wizards QA CLI │
│   (Go + Cobra)  │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌─────────┐ ┌──────────┐
│   AI    │ │ Maestro  │
│ Engine  │ │ Wrapper  │
│         │ │          │
│ Claude/ │ │ Executor │
│ Gemini  │ │ Reporter │
└─────────┘ └──────────┘
    │           │
    └─────┬─────┘
          ▼
    ┌────────────┐
    │   Flows    │
    │ Repository │
    └────────────┘
```

## 📂 Project Structure

```
wizards-qa/
├── cmd/                    # CLI commands
│   ├── main.go            # Entry point
│   ├── test.go            # Test command
│   ├── generate.go        # Generate command
│   ├── run.go             # Run command
│   ├── validate.go        # Validate command
│   └── template.go        # Template management
│
├── pkg/                    # Core packages
│   ├── ai/                # AI integration
│   ├── maestro/           # Maestro wrapper
│   ├── phaser/            # Phaser 4 utilities
│   └── flows/             # Flow management
│
├── flows/                  # Flow repository
│   ├── templates/         # Reusable templates
│   └── games/             # Game-specific flows
│
├── docs/                   # Documentation
│   ├── PROJECT-BRIEF.md   # Project overview
│   ├── ARCHITECTURE.md    # System architecture
│   ├── MAESTRO-RESEARCH.md # Maestro research
│   └── PHASER4-TESTING.md # Phaser testing guide
│
└── tests/                  # Test suite
    └── fixtures/          # Test fixtures
```

## 🔧 Configuration

Create `wizards-qa.yaml`:

```yaml
ai:
  provider: anthropic
  model: claude-sonnet-4-5
  apiKey: ${ANTHROPIC_API_KEY}

maestro:
  browser: chrome
  timeout: 300s
  screenshotDir: ./screenshots

flows:
  directory: ./flows
  templates: ./flows/templates
```

## 🎮 Phaser 4 & Canvas Testing

Phaser games render to HTML5 Canvas, requiring special interaction strategies:

### Coordinate-Based Clicking

```yaml
# Click center of game screen
- tapOn:
    point: 50%,50%

# Click specific coordinate
- tapOn:
    point: 400,300
```

### Visual Assertions

```yaml
# Screenshot-based verification
- captureScreenshot: game-state.png
- assertVisible: "Score: 100"
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed strategies.

## 📊 Example Flow

```yaml
# flows/templates/example-game.yaml
url: https://game.example.com
---
- launchApp
- waitFor:
    visible: true
    timeout: 10000
- assertVisible: "Start Game"
- tapOn: "Start Game"
- waitFor:
    timeout: 3000
- tapOn:
    point: 50%,50%
- assertVisible: "Score:"
- captureScreenshot: gameplay.png
```

## 🗺️ Development Roadmap

### ✅ Phase 0: Foundation (Current)
- [x] Project setup and architecture design
- [x] Maestro CLI integration
- [x] Basic CLI structure (Cobra)
- [x] Example flow templates
- [ ] Maestro wrapper (basic execution)

### 🚧 Phase 1: Core Infrastructure (Week 1)
- [ ] Config file parsing
- [ ] Flow validation
- [ ] Maestro execution wrapper
- [ ] Screenshot/video capture
- [ ] Basic reporting

### 📅 Phase 2: AI Integration (Week 2)
- [ ] Claude API integration
- [ ] Game analysis prompts
- [ ] Scenario generation
- [ ] Flow generation engine
- [ ] Template library

### 📅 Phase 3: Maestro Integration (Week 3)
- [ ] Full command support
- [ ] Result parsing
- [ ] Error handling
- [ ] Parallel execution

### 📅 Phase 4: Phaser 4 Specialization (Week 4)
- [ ] Canvas interaction strategies
- [ ] Visual assertion support
- [ ] Game state detection
- [ ] Phaser-specific templates

### 📅 Phase 5: Polish & Production (Week 5)
- [ ] Comprehensive testing
- [ ] Documentation
- [ ] Example flows for sample games
- [ ] CI/CD integration
- [ ] GitHub release

## 📚 Documentation

- [Project Brief](docs/PROJECT-BRIEF.md) - Vision and requirements
- [Architecture](docs/ARCHITECTURE.md) - System design and components
- [Maestro Research](docs/MAESTRO-RESEARCH.md) - Maestro CLI capabilities
- [Phaser 4 Testing](docs/PHASER4-TESTING.md) - Game testing strategies (coming soon)

## 🤝 Contributing

Contributions welcome! This is an open-source project by [Global Wizards](https://github.com/Global-Wizards).

## 📄 License

MIT License - see LICENSE file for details

## 🔗 Links

- **GitHub:** https://github.com/Global-Wizards/wizards-qa
- **Maestro Docs:** https://docs.maestro.dev/
- **Phaser 4:** https://phaser.io/
- **Global Wizards:** https://github.com/Global-Wizards

---

**Built with** 🧙‍♂️ **by the Kingdom** | Powered by AI 🤖

**Status:** Alpha - Active Development 🚧
