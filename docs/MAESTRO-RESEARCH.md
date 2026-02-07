# Maestro CLI Research

**Updated:** 2026-02-06  
**Researcher:** Lia 🌸

## Overview

Maestro is an open-source framework for mobile **and web** UI testing. Built on learnings from predecessors like Appium, Espresso, Selenium, and Playwright.

**Official Docs:** https://docs.maestro.dev/

## Key Features

### 1. **Built-in Flakiness Tolerance**
- UI elements won't always be where you expect
- Screen taps won't always go through immediately
- Maestro embraces instability and counters it automatically

### 2. **Built-in Delay Tolerance**
- No need for `sleep()` calls in tests
- Automatically waits for content to load (network, animations, etc.)
- Waits only as long as required

### 3. **Blazingly Fast Iteration**
- Tests are **interpreted**, no compilation needed
- Can continuously monitor test files and rerun on changes
- Perfect for rapid development

### 4. **Declarative YAML Syntax**
- Simple, readable test definitions
- Easy to generate programmatically (perfect for AI!)
- Example flow structure

### 5. **Simple Setup**
- Single binary that works anywhere
- Cross-platform support

## Platform Support

✅ **Android** (Views + Jetpack Compose)  
✅ **iOS** (UIKit + SwiftUI)  
✅ **React Native**  
✅ **Flutter**  
✅ **Web Views**  
✅ **Web (Desktop Browser)** ← **PERFECT FOR US!**  
✅ **.NET MAUI** (iOS + Android)

## YAML Flow Structure

Maestro flows are defined in YAML files with simple, declarative commands:

```yaml
# Example: Web flow
url: https://example.com
---
- launchApp
- tapOn: "Login"
- inputText: "username@example.com"
- tapOn: "Password"
- inputText: "password123"
- tapOn: "Submit"
- assertVisible: "Welcome back"
```

### Common Commands (from examples)
- `launchApp` - Start the application/browser
- `tapOn: "text"` - Click on element with text
- `inputText: "value"` - Type text into focused element
- `assertVisible: "text"` - Verify element is visible
- More to discover...

## Web Testing Specifics

**✅ Confirmed:** Maestro supports web browser testing!

Key considerations:
- Uses `url:` parameter instead of `appId:`
- Launches desktop browser for web flows
- Can interact with canvas elements (needs verification)
- Can capture screenshots and videos

## Installation

**Status:** Not yet installed on this system

**Next Step:** Check installation guide at:
https://docs.maestro.dev/getting-started/installing-maestro

## Open Questions

1. **Canvas Interaction:**
   - Can Maestro interact with HTML5 Canvas elements?
   - Phaser 4 games run in canvas - how to target game elements?
   - Coordinate-based clicking? Pixel analysis?

2. **Advanced Commands:**
   - What's the full command set for web flows?
   - Custom commands/plugins available?
   - JavaScript injection support?

3. **AI-Generated Flows:**
   - How complex can flows get?
   - Best practices for reusable flow components?
   - Template patterns for game testing?

4. **Test Results:**
   - Screenshot/video capture configuration
   - Test report format
   - Pass/fail criteria definition

5. **Performance:**
   - How fast can it run tests?
   - Parallel execution support?
   - Resource requirements?

## Implications for Wizards QA

### ✅ Good News
- Maestro is **perfect** for our use case!
- Web support means we can test Phaser 4 games directly
- YAML format is ideal for AI generation
- Built-in reliability features reduce flakiness
- Simple syntax = easier to generate from AI

### 🚧 Challenges
- Canvas element interaction may require workarounds
- Game-specific testing patterns need to be developed
- May need coordinate-based clicking for game UI
- Screenshot analysis might be needed for game state verification

### 💡 Strategy
1. **Install Maestro CLI** on development machine
2. **Create test flows** for a simple Phaser 4 game manually
3. **Identify patterns** that work for game testing
4. **Build AI agent** to generate these patterns from game specs
5. **Create reusable templates** for common game mechanics

## Next Steps

1. **Install Maestro** - Follow official guide
2. **Explore web commands** - Full command reference
3. **Test with simple game** - Validate canvas interaction
4. **Document patterns** - Build library of game test flows
5. **Build AI integration** - Connect to Claude/Gemini for flow generation

## Resources

- Official Docs: https://docs.maestro.dev/
- GitHub: https://github.com/mobile-dev-inc/maestro
- Web Testing Docs: https://docs.maestro.dev/platform-support/web
- Installation Guide: https://docs.maestro.dev/getting-started/installing-maestro

---

**Researcher Notes:**
This initial research is very promising! Maestro seems like exactly the right tool for our use case. The declarative YAML syntax will be perfect for AI generation, and the built-in reliability features will help with the inherent flakiness of game testing.

Need to dive deeper into canvas interaction and coordinate-based clicking next! 🔍
