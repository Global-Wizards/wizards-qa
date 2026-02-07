package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Global-Wizards/wizards-qa/pkg/ai"
	"github.com/Global-Wizards/wizards-qa/pkg/scout"
	"github.com/Global-Wizards/wizards-qa/pkg/util"
	"github.com/spf13/cobra"
)

func newScoutCmd() *cobra.Command {
	var (
		gameURL    string
		output     string
		jsonOutput bool
		saveFlows  bool
		configPath string
	)

	cmd := &cobra.Command{
		Use:   "scout",
		Short: "Scout a game URL and auto-generate Maestro test flows",
		Long: `Automatically scout a game page, detect its framework, analyze game mechanics
with AI, and generate ready-to-run Maestro test flows — no spec file needed.

Pipeline:
1. Fetch page and extract metadata (framework, canvas, scripts, structure)
2. Analyze game mechanics and UI with AI
3. Generate test scenarios and Maestro YAML flows

Example:
  wizards-qa scout --game https://game.example.com
  wizards-qa scout --game https://game.example.com --json
  wizards-qa scout --game https://game.example.com --output ./my-flows`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if gameURL == "" {
				return fmt.Errorf("--game URL is required")
			}

			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			if !jsonOutput {
				fmt.Printf("%s Scouting %s...\n", util.EmojiTarget, gameURL)
			}

			// Step 1: Scout the page
			ctx := context.Background()
			pageMeta, err := scout.ScoutURL(ctx, gameURL)
			if err != nil {
				return fmt.Errorf("page scout failed: %w", err)
			}

			if !jsonOutput {
				fmt.Printf("   Title: %s\n", pageMeta.Title)
				fmt.Printf("   Framework: %s\n", pageMeta.Framework)
				fmt.Printf("   Canvas: %v\n", pageMeta.CanvasFound)
				fmt.Printf("   Scripts: %d\n\n", len(pageMeta.ScriptSrcs))
			}

			if err := validateAPIKey(cfg); err != nil {
				return err
			}

			analyzer, err := newAnalyzer(cfg)
			if err != nil {
				return err
			}

			if !jsonOutput {
				fmt.Printf("%s Analyzing game with AI...\n", util.EmojiClip)
			}

			// Step 2: Full pipeline
			_, result, flows, err := analyzer.AnalyzeFromURL(ctx, gameURL)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			if !jsonOutput {
				if result.GameInfo.Name != "" {
					fmt.Printf("   Game: %s\n", result.GameInfo.Name)
					fmt.Printf("   Genre: %s\n", result.GameInfo.Genre)
				}
				fmt.Printf("   Mechanics: %d\n", len(result.Mechanics))
				fmt.Printf("   UI Elements: %d\n", len(result.UIElements))
				fmt.Printf("   User Flows: %d\n\n", len(result.UserFlows))
				fmt.Printf("%s Generating %d test flow(s)...\n", util.EmojiHammer, len(flows))
			}

			// Step 3: Save flows
			if saveFlows {
				gameName := deriveGameName(gameURL)
				flowsDir := fmt.Sprintf("%s/%s", output, gameName)
				if err := ai.WriteFlowsToFiles(flows, flowsDir); err != nil {
					return fmt.Errorf("failed to save flows: %w", err)
				}
				if !jsonOutput {
					fmt.Printf("%s Flows saved to: %s\n\n", util.EmojiPassed, flowsDir)
				}
			}

			// Step 4: JSON output
			if jsonOutput {
				out := map[string]interface{}{
					"pageMeta": pageMeta,
					"analysis": result,
					"flows":    flows,
				}
				data, err := json.Marshal(out)
				if err != nil {
					return fmt.Errorf("JSON marshal failed: %w", err)
				}
				fmt.Println(string(data))
			} else {
				fmt.Printf("%s Done! %d flow(s) generated.\n", util.EmojiRocket, len(flows))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&gameURL, "game", "g", "", "Game URL to analyze (required)")
	cmd.Flags().StringVarP(&output, "output", "o", "./flows", "Output directory for generated flows")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output full result as JSON to stdout")
	cmd.Flags().BoolVar(&saveFlows, "save-flows", true, "Write generated flows to disk")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file path")

	cmd.MarkFlagRequired("game")

	return cmd
}
