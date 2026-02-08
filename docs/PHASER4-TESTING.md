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
- **Tools:** Maestro CLI / Playwright / Gemini Vision.
- **Approach:** Use a "headless" browser to run the game and simulate inputs. Leverage AI Vision to "see" game state when the engine isn't easily queryable.

## The Canvas Challenge
In Phaser, everything happens inside a single `<canvas>` element. Standard UI testers cannot click "the red button" because they only see the canvas.

### Strategy 1: The "Debug Bridge" (Engine Query)
1. **Expose Game Objects:** Create a global `__WIZARDS_QA__` object that tracks active game entities and their bounding boxes.
2. **Maestro `evalScript`**: Query the bridge: `output.playerPos = evalScript("window.__WIZARDS_QA__.getPlayerPosition()")`.

### Strategy 2: AI Vision (The "Maestro" Way)
1. **`assertWithAI`**: Instead of querying memory, ask the AI: "Is the player sprite overlapping the spike trap?"
2. **`extractTextWithAI`**: Read high scores or character names directly from the canvas pixels.

### Strategy 3: Coordinate-Based Interaction
Mapping game world coordinates to screen space for Maestro's `tapOn: { x, y }` command. Phaser 4's modular camera system makes this transformation straightforward for the automation layer.

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
