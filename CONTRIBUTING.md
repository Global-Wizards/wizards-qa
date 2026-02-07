# Contributing to Wizards QA

Thank you for your interest in contributing to Wizards QA! 🧙‍♂️

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue with:
- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Your environment (OS, Go version, Maestro version)
- Relevant logs or screenshots

### Suggesting Features

Feature requests are welcome! Please include:
- Clear description of the feature
- Use case and motivation
- Examples of how it would work
- Any implementation ideas

### Contributing Code

1. **Fork the repository**
2. **Create a feature branch:** `git checkout -b feature/amazing-feature`
3. **Make your changes**
4. **Test thoroughly**
5. **Commit with clear messages:** `git commit -m "feat: add amazing feature"`
6. **Push to your fork:** `git push origin feature/amazing-feature`
7. **Open a Pull Request**

## Development Setup

### Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/dl/)
- **Java 17+** - Required by Maestro
- **Maestro CLI** - [Installation guide](https://docs.maestro.dev/getting-started/installing-maestro)

### Setup

```bash
# Clone the repository
git clone https://github.com/Global-Wizards/wizards-qa.git
cd wizards-qa

# Install dependencies
go mod download

# Build the CLI
go build -o wizards-qa ./cmd

# Run tests (when available)
go test ./...
```

### Project Structure

```
wizards-qa/
├── cmd/                    # CLI commands
├── pkg/                    # Core packages
│   ├── ai/                # AI integration
│   ├── config/            # Configuration
│   ├── flows/             # Flow validation
│   ├── maestro/           # Maestro wrapper
│   └── report/            # Test reporting
├── flows/templates/        # Flow templates
├── examples/              # Example specs
└── docs/                  # Documentation
```

## Coding Guidelines

### Go Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and small
- Handle errors explicitly

### Example

```go
// AnalyzeGame analyzes a game from specification and URL
func (a *Analyzer) AnalyzeGame(specPath, gameURL string) (*AnalysisResult, error) {
    // Read game specification
    specContent, err := os.ReadFile(specPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read spec file: %w", err)
    }
    
    // ... rest of implementation
}
```

### Commit Messages

Use conventional commits format:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Formatting, missing semicolons, etc.
- `refactor:` - Code restructuring
- `test:` - Adding tests
- `chore:` - Maintenance tasks

**Examples:**
```
feat: add template apply command with variable substitution
fix: handle missing API key gracefully in generate command
docs: add template usage examples to README
```

## Adding New Features

### Flow Templates

To add a new flow template:

1. Create `.yaml` file in `flows/templates/` or subdirectory
2. Use `{{VARIABLE}}` syntax for customizable values
3. Add helpful comments
4. Capture screenshots at key moments
5. Document in `flows/templates/README.md`

### CLI Commands

To add a new CLI command:

1. Create `cmd/yourcommand.go`
2. Implement using Cobra framework
3. Add to `cmd/main.go` with `rootCmd.AddCommand()`
4. Update `README.md` with usage examples

### AI Prompts

To improve AI prompts:

1. Edit templates in `pkg/ai/types.go`
2. Test with real game specifications
3. Document changes and rationale
4. Include example outputs

## Testing

### Manual Testing

```bash
# Test flow validation
./wizards-qa validate --flow flows/templates/example-game.yaml

# Test configuration
./wizards-qa config show

# Test template system
./wizards-qa template list
./wizards-qa template show click-object
```

### Integration Testing

When testing with AI:

```bash
# Set API key
export ANTHROPIC_API_KEY=your_key_here

# Test generation (requires API key)
./wizards-qa generate --game https://game.com --spec examples/simple-platformer-spec.md

# Test E2E (requires API key + Maestro)
./wizards-qa test --game https://game.com --spec examples/simple-platformer-spec.md
```

## Documentation

### What to Document

- New features and commands
- Configuration options
- Template usage
- Architecture changes
- Breaking changes

### Where to Document

- `README.md` - User-facing guide
- `docs/ARCHITECTURE.md` - Technical design
- `CHANGELOG.md` - Changes by version
- Code comments - Implementation details
- Template README - Template usage

## Community

### Getting Help

- **Discord:** https://discord.com/invite/clawd
- **GitHub Issues:** For bugs and features
- **Discussions:** For questions and ideas

### Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Give constructive feedback
- Focus on the problem, not the person
- Assume good intentions

## Recognition

Contributors will be recognized in:
- README.md contributors section
- CHANGELOG.md for their changes
- GitHub contributors page

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to Wizards QA!** 🧙‍♂️🎮
