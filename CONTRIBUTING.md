# Contributing to Wizards QA

## How to Contribute

### Reporting Bugs

Create an issue with:
- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Your environment (OS, Go version, Chrome version)
- Relevant logs or screenshots

### Suggesting Features

Feature requests are welcome! Please include:
- Clear description of the feature
- Use case and motivation
- Examples of how it would work

### Contributing Code

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Run tests: `make test`
5. Commit with clear messages: `git commit -m "feat: add amazing feature"`
6. Push to your fork: `git push origin feature/amazing-feature`
7. Open a Pull Request

## Development Setup

### Prerequisites

- **Go 1.25+** -- [Install Go](https://go.dev/dl/)
- **Node.js 18+** -- For the web frontend
- **Chrome/Chromium** -- Required for headless browser automation

### Setup

```bash
git clone https://github.com/Global-Wizards/wizards-qa.git
cd wizards-qa
go mod download
make build
make test
```

### Project Structure

```
wizards-qa/
├── cmd/                    # CLI commands (Cobra)
├── pkg/                    # Core packages
│   ├── ai/                 # AI agent, tools, synthesis
│   ├── scout/              # Headless Chrome, click strategies
│   ├── config/             # Configuration
│   ├── flows/              # Flow validation
│   ├── maestro/            # Legacy Maestro wrapper
│   └── report/             # Test reporting
├── web/                    # Web application
│   ├── backend/            # Go HTTP server
│   └── frontend/           # Vue.js dashboard
└── docs/                   # Documentation
```

## Coding Guidelines

### Go Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors explicitly

### Commit Messages

Use conventional commits format:

- `feat:` -- New feature
- `fix:` -- Bug fix
- `docs:` -- Documentation changes
- `refactor:` -- Code restructuring
- `test:` -- Adding tests
- `chore:` -- Maintenance tasks

### Release Process

- Update `VERSION` and `CHANGELOG.md` with every commit that changes functionality
- Bump patch for fixes, minor for features, major for breaking changes

## Testing

```bash
# Run all tests
make test

# Run CLI tests only
make test-cli

# Run backend tests only
make test-backend

# Vet all code
make vet

# Full validation
make validate
```

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
