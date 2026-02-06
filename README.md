# Wizards QA 🧙‍♂️🔍

AI-powered QA automation system for Phaser 4 web games using Maestro CLI.

## Overview

Wizards QA is an intelligent testing framework that:
- Accepts game requirements and live URLs
- Uses AI to analyze and understand the game
- Generates comprehensive Maestro test flows
- Automates end-to-end testing of web games built with Phaser 4

## Tech Stack

- **Go** + **Cobra** - CLI framework
- **Maestro CLI** - Test execution engine
- **AI QA Agent** - Intelligent test flow generation
- **Phaser 4** - Target game framework

## Project Status

🚧 **In Design Phase** - Architecture being developed by Kingdom agents

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed system design.

## Usage (Planned)

```bash
# Submit a game for testing
wizards-qa test --game "https://example.com/game" --requirements game-spec.md

# Generate test flows only
wizards-qa generate --game "https://example.com/game" --output flows/

# Run existing test suite
wizards-qa run --flows flows/game-tests/
```

## Development

```bash
# Install dependencies
go mod download

# Build CLI
go build -o wizards-qa cmd/main.go

# Run tests
go test ./...
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and components
- [Maestro Integration](docs/MAESTRO.md) - Maestro CLI integration guide
- [AI Agent](docs/AI-AGENT.md) - QA agent design and prompting
- [Contributing](CONTRIBUTING.md) - How to contribute

## License

MIT License - See [LICENSE](LICENSE) for details

---

**Created by:** Global Wizards  
**Maintained by:** The Kingdom 👑
