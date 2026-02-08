# Maestro CLI Research 🧪

## Overview
Maestro is a next-generation UI automation tool known for its simplicity and YAML-based "flows." While originally mobile-focused (iOS/Android), it has expanded to support web applications, making it a candidate for game testing.

## Key Capabilities
- **YAML Syntax:** Tests are written in declarative YAML, making them readable for both humans and AI.
- **Auto-Wait:** Maestro automatically waits for elements to appear, reducing flakiness.
- **Native/Web Support:** Can interact with native apps and web browsers.
- **Screenshots & Video:** Built-in commands to capture visual state.
- **Custom Commands:** Ability to extend Maestro with custom logic.

## Maestro for Web Games (Phaser 4)
### Strengths
- **Low Barrier to Entry:** AI agents can easily generate YAML flows.
- **Stable Execution:** Better than Selenium/Appium for modern web stacks.
- **Visual Validation:** Screen capture helps in detecting rendering issues.

### Challenges
- **Canvas Interaction:** Maestro (like most UI tools) sees the `<canvas>` as a single element. It cannot "see" individual sprites or game objects natively.
- **Workaround:** Requires a "bridge" or "debug mode" in the Phaser game that exposes object coordinates/states to the DOM or via a global variable that Maestro can query.

## Core Commands for Game Testing
```yaml
appId: com.example.game
---
- launchApp
- tapOn: "Start Game"
- assertVisible: "Level 1"
- takeScreenshot: "game-start"
- runScript: "scripts/check-game-state.js" # Custom JS for deep state inspection
```

## Recommendation
Maestro is excellent for high-level "flow" testing (menus, transitions, loading). For deep in-game logic, we must combine it with a custom Phaser plugin that exposes game state to the testing environment.

---
*Source: Sage 🔮 (Research Cycle 17)*
