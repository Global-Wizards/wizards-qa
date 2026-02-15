# Wizards QA - Project Brief

**Project:** wizards-qa  
**Created:** 2026-02-06  
**Owner:** Fernando (Global Wizards)  
**Status:** 🚧 Design Phase

## Vision

Create a fully automated QA system that can intelligently test Phaser 4 web games by:
1. Understanding game requirements and specifications
2. Generating comprehensive test flows using AI
3. Executing tests via Maestro CLI
4. Providing detailed test results and insights

## Core Requirements

### Input
- **Game Description:** Markdown spec with game details, mechanics, features
- **Live URL:** Public web link to the deployed game
- **Optional:** Screenshots, videos, or additional context

### Processing
- **AI QA Agent:** Analyzes the game and generates test scenarios
- **Flow Generation:** Creates Maestro test scripts for comprehensive coverage
- **Test Orchestration:** Manages test execution and result collection

### Output
- **Maestro Test Flows:** Ready-to-run test scripts
- **Test Reports:** Detailed results with screenshots/videos
- **Issues/Bugs:** Identified problems with reproduction steps

## Technical Stack

### CLI (Go + Cobra)
```
wizards-qa/
├── cmd/
│   ├── main.go           # CLI entry point
│   ├── test.go           # Test command
│   ├── generate.go       # Generate flows command
│   └── run.go            # Run flows command
├── pkg/
│   ├── ai/               # AI agent integration
│   ├── maestro/          # Maestro CLI wrapper
│   ├── phaser/           # Phaser 4 game analysis
│   └── flows/            # Flow generation logic
└── tests/
    └── fixtures/         # Test game examples
```

### Integration Points
1. **Maestro CLI** - Test execution engine
2. **AI Model** (Claude/Gemini) - Test flow generation
3. **Browser/Headless** - Game interaction via Maestro
4. **Git/GitHub** - Flow storage and versioning

## Workflow

```
┌─────────────────┐
│ Game Spec + URL │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   AI Analysis   │  ← Understand game mechanics
│  (Claude/Gemini) │     Identify test scenarios
└────────┬────────┘     Define success criteria
         │
         ▼
┌─────────────────┐
│ Flow Generation │  ← Create Maestro YAML flows
│   (wizards-qa)  │     Generate test steps
└────────┬────────┘     Add assertions
         │
         ▼
┌─────────────────┐
│  Maestro CLI    │  ← Execute tests in browser
│  (Test Runner)  │     Capture screenshots/video
└────────┬────────┘     Record results
         │
         ▼
┌─────────────────┐
│  Test Report    │  ← Pass/fail status
│   (Markdown)    │     Bug reports
└─────────────────┘     Recommendations
```

## Open Questions

1. **AI Integration:**
   - Which model(s) to use? (Claude Sonnet, Gemini Pro)
   - Prompt engineering for game analysis
   - Context window management for large games

2. **Maestro Integration:**
   - How to structure flows for maximum reusability?
   - Custom Maestro commands/plugins needed?
   - Screenshot/video capture strategy

3. **Game Analysis:**
   - How to detect game mechanics automatically?
   - Canvas element interaction via Maestro
   - Phaser 4-specific considerations

4. **Test Coverage:**
   - What level of coverage is realistic?
   - User journey vs. component testing
   - Performance/load testing scope

5. **Storage:**
   - Where to store generated flows? (Git repo, database)
   - Test result history tracking
   - CI/CD integration strategy

## Success Criteria

- [ ] Can accept game spec + URL as input
- [ ] Generates valid Maestro test flows
- [ ] Successfully tests a simple Phaser 4 game end-to-end
- [ ] Produces useful test reports with actionable insights
- [ ] Reusable flows for common game patterns

## Next Steps

1. **Architecture Design** (Nova 🌟)
   - System architecture
   - Component design
   - Integration patterns

2. **Research** (Sage 🔮)
   - Maestro CLI capabilities
   - Phaser 4 testing best practices
   - Similar tools/approaches

3. **QA Strategy** (Sentinel 🛡️)
   - Test flow design patterns
   - Coverage strategies
   - Quality gates

4. **Implementation** (Forge 🔨)
   - Go CLI skeleton
   - Maestro wrapper
   - Flow generator

---

**Fernando's Request:**
> "I want to send a requirement of the game, with the description of the game, etc, and the link where the game is live. The system would simply create flows with AI as an QA agent to go and check the full game, and create flows of test to send to maestro to test the complete game."
