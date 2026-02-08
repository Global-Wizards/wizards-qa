# Phaser 4 Testing Best Practices 🎮

## Overview
Phaser 4 represents a significant architectural shift towards more modular, modern JavaScript. Testing a Phaser 4 game requires a multi-layered approach to ensure both technical stability and "fun" (gameplay mechanics).

## Testing Layers

### 1. Unit Testing (Logic)
- **Scope:** Math utilities, state machines, inventory logic, scoring algorithms.
- **Tools:** Vitest / Jest.
- **Approach:** Decouple logic from the Phaser `Scene` so it can be tested in isolation.

### 2. Integration Testing (Scenes)
- **Scope:** Scene transitions, asset loading, event emitters.
- **Tools:** Custom Phaser test harness.
- **Approach:** Mock the Phaser `Game` instance to verify that scenes load and cleanup correctly.

### 3. End-to-End (E2E) Testing (Gameplay)
- **Scope:** Player movement, collision detection, UI interaction.
- **Tools:** Maestro CLI / Playwright.
- **Approach:** Use a "headless" browser to run the game and simulate inputs.

## The Canvas Challenge
In Phaser, everything happens inside a single `<canvas>` element. Standard UI testers cannot click "the red button" because they only see the canvas.

### Strategy: The "Debug Bridge"
To make the game "testable" by AI/Maestro:
1. **Expose Game Objects:** Create a global `__WIZARDS_QA__` object that tracks active game entities and their bounding boxes.
2. **DOM Proxying:** Create invisible HTML elements that mirror the position and labels of game objects. Maestro can then "tap" on these proxy elements.
3. **Screenshot Comparison:** Use visual regression testing to detect graphical glitches.

## Common Game Patterns to Test
- **The "Infinite Loop":** Ensuring assets don't leak memory on scene restart.
- **Input Lag:** Measuring the time between a "Tap" event and the corresponding game action.
- **Resolution Scaling:** Verifying the game renders correctly on mobile vs. desktop aspect ratios.

## AI-Driven QA Potential
Phaser 4's modular nature makes it easier for AI to:
- Generate mock data for game states.
- Analyze source code to identify potential edge cases in scene logic.
- Predict performance bottlenecks in complex particle systems.

---
*Source: Sage 🔮 (Research Cycle 17)*
