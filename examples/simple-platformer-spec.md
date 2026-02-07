# Simple Platformer Game - Test Specification

## Overview
A 2D side-scrolling platformer game built with Phaser 4. Players control a character that can jump and run to collect coins while avoiding obstacles.

## Game Mechanics

### Movement
- **Left/Right Arrow Keys**: Move character horizontally
- **Space Bar or Up Arrow**: Jump
- **Character Speed**: Consistent movement speed
- **Jump Height**: Fixed jump arc

### Objectives
- **Collect Coins**: Scattered throughout the level
- **Avoid Obstacles**: Red enemy sprites that cause damage
- **Reach Goal**: Flag at the end of the level triggers victory

### Game States
1. **Main Menu**: Title screen with "Start Game" button
2. **Gameplay**: Active play with character control
3. **Victory**: Reached the flag successfully
4. **Game Over**: Hit 3 obstacles (lost all lives)

## UI Elements

### Main Menu
- **Title**: "Simple Platformer" (centered top)
- **Start Button**: Green button with text "Start Game"
- **Instructions**: Brief control guide below button

### HUD (Heads-Up Display)
- **Score**: Top-left corner, format "Score: XXX"
- **Lives**: Top-right corner, heart icons (3 hearts)
- **Coins Collected**: Below score, format "Coins: X/10"

### Game Over Screen
- **Message**: "Game Over!" centered
- **Final Score**: Display total score
- **Retry Button**: "Try Again" button

### Victory Screen
- **Message**: "You Win!" centered
- **Completion Time**: Display time taken
- **Next Level Button**: "Continue" button

## Technical Details

### Technology Stack
- **Engine**: Phaser 4
- **Rendering**: HTML5 Canvas
- **Platform**: Web browser

### Game URL
- Development: `http://localhost:8000`
- Production: `https://games.example.com/simple-platformer`

### Canvas Coordinates (approximate)
- **Canvas Size**: 800x600 pixels
- **Start Button**: ~50%, 60% (center-ish)
- **Player Start Position**: ~10%, 80%
- **First Coin**: ~25%, 50%
- **First Enemy**: ~40%, 75%
- **Goal Flag**: ~90%, 80%

## Test Scenarios

### Critical User Flows
1. **Complete Game Flow**: Menu → Gameplay → Collect all coins → Reach flag → Victory
2. **Game Over Flow**: Menu → Gameplay → Hit 3 enemies → Game over → Retry
3. **Basic Movement**: Jump, run left/right, basic physics

### Edge Cases
- **Double Jump Attempt**: Verify player can't jump while in air
- **Boundary Detection**: Player can't walk off screen edges
- **Coin Collection**: Each coin increments counter correctly
- **Enemy Collision**: Taking damage reduces lives properly

### Performance
- **Load Time**: Game should load within 5 seconds
- **FPS**: Maintain 60 FPS during gameplay
- **Responsiveness**: Input lag < 100ms

## Expected Behavior

### Valid Actions
- ✅ Player can jump on platforms
- ✅ Coins disappear when collected
- ✅ Score increases with each coin
- ✅ Lives decrease when hit by enemy
- ✅ Victory screen shows on goal reached

### Invalid/Error States
- ❌ Player should not clip through walls
- ❌ Enemies should not move off-screen
- ❌ Score should not decrease
- ❌ Game should not freeze on enemy collision

## Acceptance Criteria

A successful test run should verify:
1. All UI elements are visible and functional
2. Player controls work correctly
3. Collision detection works (coins, enemies, platforms)
4. Game state transitions occur properly
5. Victory and game over conditions trigger correctly
6. No visual glitches or rendering issues
7. Performance meets targets (FPS, load time)
