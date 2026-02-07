package main

import (
	"fmt"
	"path/filepath"

	"github.com/Global-Wizards/wizards-qa/pkg/ai"
	"github.com/Global-Wizards/wizards-qa/pkg/util"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var (
		gameURL    string
		specFile   string
		output     string
		configPath string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Maestro test flows using AI (no execution)",
		Long: `Analyze a game and generate Maestro test flows without executing them.

This command only performs analysis and flow generation:
1. Load game specification
2. Analyze game with AI
3. Generate Maestro YAML flows
4. Save flows to output directory

Useful when you want to review flows before execution or generate flows for later use.

Example:
  wizards-qa generate --game https://game.example.com --spec spec.md
  wizards-qa generate --game https://game.example.com --spec spec.md --output custom-flows/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if gameURL == "" {
				return fmt.Errorf("--game URL is required")
			}
			if specFile == "" {
				return fmt.Errorf("--spec file is required")
			}

			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			fmt.Printf("%s Wizards QA - Flow Generation\n\n", util.EmojiWizard)
			fmt.Printf("Game URL:  %s\n", gameURL)
			fmt.Printf("Spec File: %s\n", specFile)
			fmt.Printf("AI Model:  %s\n", cfg.AI.Model)
			fmt.Printf("Output:    %s\n\n", output)

			if err := validateAPIKey(cfg); err != nil {
				return err
			}

			analyzer, err := newAnalyzer(cfg)
			if err != nil {
				return err
			}

			// Step 1: Analyze game
			fmt.Printf("%s Analyzing game...\n", util.EmojiClip)
			analysis, err := analyzer.AnalyzeGame(specFile, gameURL)
			if err != nil {
				return fmt.Errorf("game analysis failed: %w", err)
			}

			if analysis.GameInfo.Name != "" {
				fmt.Printf("   Game: %s\n", analysis.GameInfo.Name)
				fmt.Printf("   Genre: %s\n", analysis.GameInfo.Genre)
			}
			fmt.Printf("   Mechanics: %d\n", len(analysis.Mechanics))
			fmt.Printf("   UI Elements: %d\n", len(analysis.UIElements))
			fmt.Printf("   User Flows: %d\n\n", len(analysis.UserFlows))

			// Step 2: Generate scenarios
			fmt.Printf("%s Generating test scenarios...\n", util.EmojiTarget)
			scenarios, err := analyzer.GenerateScenarios(analysis)
			if err != nil {
				return fmt.Errorf("scenario generation failed: %w", err)
			}
			fmt.Printf("   Generated %d test scenario(s)\n\n", len(scenarios))

			// Step 3: Generate Maestro flows
			fmt.Printf("%s Generating Maestro flows...\n", util.EmojiHammer)
			flows, err := analyzer.GenerateFlows(scenarios)
			if err != nil {
				return fmt.Errorf("flow generation failed: %w", err)
			}
			fmt.Printf("   Generated %d flow(s)\n\n", len(flows))

			// Step 4: Save flows
			fmt.Printf("%s Saving flows...\n", util.EmojiDisk)
			gameName := deriveGameName(gameURL)
			flowsDir := filepath.Join(output, gameName)

			if err := ai.WriteFlowsToFiles(flows, flowsDir); err != nil {
				return fmt.Errorf("failed to save flows: %w", err)
			}

			fmt.Printf("%s Flows saved to: %s\n\n", util.EmojiPassed, flowsDir)
			fmt.Println("You can now run these flows with:")
			fmt.Printf("  wizards-qa run --flows %s\n", flowsDir)

			return nil
		},
	}

	cmd.Flags().StringVarP(&gameURL, "game", "g", "", "Live game URL (required)")
	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "Game specification file (required)")
	cmd.Flags().StringVarP(&output, "output", "o", "./flows", "Output directory for generated flows")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file path")

	cmd.MarkFlagRequired("game")
	cmd.MarkFlagRequired("spec")

	return cmd
}
