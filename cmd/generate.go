package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var (
		gameURL  string
		specFile string
		output   string
		model    string
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

			fmt.Printf("🧙‍♂️ Wizards QA - Flow Generation\n\n")
			fmt.Printf("Game URL:  %s\n", gameURL)
			fmt.Printf("Spec File: %s\n", specFile)
			fmt.Printf("AI Model:  %s\n", model)
			fmt.Printf("Output:    %s\n\n", output)

			// TODO: Implement flow generation
			// 1. Load spec file
			// 2. Analyze game with AI
			// 3. Generate flows
			// 4. Save to output directory

			fmt.Println("⚠️  Flow generation not yet implemented")
			fmt.Println("Coming soon in Phase 2!")

			return nil
		},
	}

	cmd.Flags().StringVarP(&gameURL, "game", "g", "", "Live game URL (required)")
	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "Game specification file (required)")
	cmd.Flags().StringVarP(&output, "output", "o", "./flows", "Output directory for generated flows")
	cmd.Flags().StringVarP(&model, "model", "m", "claude-sonnet-4-5", "AI model to use")

	cmd.MarkFlagRequired("game")
	cmd.MarkFlagRequired("spec")

	return cmd
}
