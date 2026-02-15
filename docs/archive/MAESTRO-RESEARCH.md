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
- evalScript: |
    ${window.game.scene.scenes[0].player.hp > 0} # Inline JS state check
- assertWithAI:
    assertion: "The player character is standing next to the treasure chest."
```

## Advanced Features for Game QA
- **`assertWithAI`**: This is the "Game Changer." Since games are often visually complex with zero DOM accessibility, `assertWithAI` allows us to describe the intended visual state. Maestro takes a screenshot and uses an LLM to verify if the description matches the visual (e.g., "The boss health bar is at 50%").
- **`tapOn: { x: 100, y: 200 }`**: Direct coordinate tapping is essential for interacting with specific regions of the Phaser canvas when text-based selection fails.
- **`extractTextWithAI`**: Useful for reading non-DOM text (bitmaps, custom fonts) inside the game world for verification.

## Recommendation
Maestro is excellent for high-level "flow" testing (menus, transitions, loading). For deep in-game logic, we should leverage **`assertWithAI`** for visual validation and **`evalScript`** to query the Phaser 4 engine state directly. This hybrid approach removes the need for expensive "DOM Proxying" in many cases.

---
*Source: Sage 🔮 (Research Cycle 17)*
