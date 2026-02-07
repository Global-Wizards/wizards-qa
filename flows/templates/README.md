# Maestro Flow Templates

Reusable flow templates for common game testing patterns.

## Template Variables

Templates use `{{VARIABLE}}` syntax for customization. Replace these with actual values when using a template.

Common variables:
- `{{GAME_URL}}` - URL of the game
- `{{BUTTON_TEXT}}` - Text on HTML buttons
- `{{X_COORD}}`, `{{Y_COORD}}` - Canvas coordinates (e.g., "50%,50%" or "400,300")
- `{{EXPECTED_TEXT}}` - Text to verify
- `{{OBJECT_NAME}}` - Name for screenshots

## Available Templates

### Game Mechanics

#### `game-mechanics/click-object.yaml`
Test clicking on canvas objects using coordinates.

**Variables:**
- `GAME_URL`, `BUTTON_TEXT`, `X_COORD`, `Y_COORD`, `OBJECT_NAME`, `EXPECTED_TEXT`

**Use case:** Click-based games, button interactions

---

#### `game-mechanics/collect-items.yaml`
Test item collection mechanics with score verification.

**Variables:**
- `GAME_URL`, `ITEM1_X`, `ITEM1_Y`, `ITEM2_X`, `ITEM2_Y`, `ITEM3_X`, `ITEM3_Y`

**Use case:** Platformers, arcade games with collectibles

---

#### `game-mechanics/character-movement.yaml`
Test player movement in different directions.

**Variables:**
- `GAME_URL`

**Use case:** Platformers, top-down games, any game with player movement

---

#### `game-mechanics/enemy-collision.yaml`
Test collision detection with enemies/obstacles.

**Variables:**
- `GAME_URL`, `LIVES_TEXT`, `ENEMY_X`, `ENEMY_Y`

**Use case:** Action games, platformers with enemies

---

#### `game-mechanics/victory-condition.yaml`
Test win state and victory screens.

**Variables:**
- `GAME_URL`, `GOAL_X`, `GOAL_Y`, `VICTORY_TEXT`, `SCORE_TEXT`, `TIME_TEXT`, `CONTINUE_BUTTON`

**Use case:** Any game with win conditions

---

#### `game-mechanics/game-over.yaml`
Test failure conditions and game over screens.

**Variables:**
- `GAME_URL`, `ENEMY_X`, `ENEMY_Y`, `GAME_OVER_TEXT`, `RETRY_BUTTON`

**Use case:** Games with lives/health systems

---

### Main Flow

#### `example-game.yaml`
Complete example flow showing all common patterns.

**Use case:** Reference for building comprehensive test flows

---

## How to Use Templates

### Manual Usage

1. Copy a template file
2. Replace all `{{VARIABLES}}` with actual values
3. Save with a descriptive name
4. Run with `wizards-qa run --flows <directory>`

### Example

```bash
# Copy template
cp flows/templates/game-mechanics/click-object.yaml flows/my-game/click-button.yaml

# Edit the file and replace variables:
# {{GAME_URL}} → https://my-game.example.com
# {{BUTTON_TEXT}} → "Start Game"
# {{X_COORD}} → 50%
# {{Y_COORD}} → 50%
# etc.

# Run the flow
wizards-qa run --flows flows/my-game/
```

### AI-Powered Usage

The `wizards-qa generate` command uses these templates as reference when generating flows:

```bash
wizards-qa generate --game https://game.example.com --spec game-spec.md
```

The AI will automatically:
1. Analyze the game
2. Select appropriate templates
3. Fill in variables based on game analysis
4. Generate customized flows

## Creating Custom Templates

To create your own templates:

1. Create a new `.yaml` file in `flows/templates/`
2. Use `{{VARIABLE}}` syntax for customizable values
3. Add helpful comments
4. Include screenshot captures for key moments
5. Document the template in this README

### Template Best Practices

✅ **Do:**
- Use descriptive variable names in UPPER_CASE
- Add comments explaining each step
- Capture screenshots at key moments
- Wait for animations/loading
- Verify state changes with assertions

❌ **Don't:**
- Hard-code game-specific values
- Skip error checking
- Assume instant state changes
- Forget to capture evidence (screenshots)

## Contributing

To contribute new templates:

1. Test your template on at least one game
2. Add clear documentation
3. Use consistent variable naming
4. Follow the existing template structure
5. Submit a PR with examples

## Support

For questions or issues with templates:
- GitHub Issues: https://github.com/Global-Wizards/wizards-qa/issues
- Discord: https://discord.com/invite/clawd

---

**Happy Testing!** 🧙‍♂️🎮
