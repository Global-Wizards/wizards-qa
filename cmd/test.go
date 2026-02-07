package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/ai"
	"github.com/Global-Wizards/wizards-qa/pkg/config"
	"github.com/Global-Wizards/wizards-qa/pkg/maestro"
	"github.com/Global-Wizards/wizards-qa/pkg/report"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	var (
		gameURL    string
		specFile   string
		output     string
		browser    string
		configPath string
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run full E2E test on a Phaser 4 game",
		Long: `Analyze a game, generate test flows with AI, and execute them via Maestro.

This command performs the complete Wizards QA workflow:
1. Load game specification
2. Analyze game with AI (Claude/Gemini)
3. Generate Maestro test flows
4. Execute flows and collect results
5. Generate test report

Example:
  wizards-qa test --game https://game.example.com --spec game-spec.md
  wizards-qa test --game https://game.example.com --spec spec.md --browser firefox`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if gameURL == "" {
				return fmt.Errorf("--game URL is required")
			}
			if specFile == "" {
				return fmt.Errorf("--spec file is required")
			}

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if browser != "" {
				cfg.Maestro.Browser = browser
			}

			fmt.Printf("🧙‍♂️ Wizards QA - Full E2E Test\n\n")
			fmt.Printf("Game URL:  %s\n", gameURL)
			fmt.Printf("Spec File: %s\n", specFile)
			fmt.Printf("AI Model:  %s\n", cfg.AI.Model)
			fmt.Printf("Browser:   %s\n\n", cfg.Maestro.Browser)

			// Validate API key
			if cfg.AI.APIKey == "" || cfg.AI.APIKey == "${ANTHROPIC_API_KEY}" {
				return fmt.Errorf("AI API key not configured. Set ANTHROPIC_API_KEY environment variable")
			}

			// Create AI client and analyzer
			client := ai.NewClaudeClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.Temperature, cfg.AI.MaxTokens)
			analyzer := ai.NewAnalyzer(client)

			// Step 1: Analyze game
			fmt.Println("📋 Step 1: Analyzing game...")
			analysis, err := analyzer.AnalyzeGame(specFile, gameURL)
			if err != nil {
				return fmt.Errorf("game analysis failed: %w", err)
			}

			if analysis.GameInfo.Name != "" {
				fmt.Printf("   Game: %s\n", analysis.GameInfo.Name)
			}
			fmt.Printf("   Mechanics: %d | UI Elements: %d | User Flows: %d\n\n", 
				len(analysis.Mechanics), len(analysis.UIElements), len(analysis.UserFlows))

			// Step 2: Generate scenarios
			fmt.Println("🎯 Step 2: Generating test scenarios...")
			scenarios, err := analyzer.GenerateScenarios(analysis)
			if err != nil {
				return fmt.Errorf("scenario generation failed: %w", err)
			}
			fmt.Printf("   Generated %d scenario(s)\n\n", len(scenarios))

			// Step 3: Generate flows
			fmt.Println("🔨 Step 3: Generating Maestro flows...")
			flows, err := analyzer.GenerateFlows(scenarios)
			if err != nil {
				return fmt.Errorf("flow generation failed: %w", err)
			}
			fmt.Printf("   Generated %d flow(s)\n\n", len(flows))

			// Step 4: Save flows
			gameName := filepath.Base(filepath.Dir(gameURL))
			if gameName == "" || gameName == "." {
				gameName = "game"
			}
			flowsDir := filepath.Join(output, gameName)

			fmt.Println("💾 Step 4: Saving flows...")
			if err := ai.WriteFlowsToFiles(flows, flowsDir); err != nil {
				return fmt.Errorf("failed to save flows: %w", err)
			}
			fmt.Printf("   Saved to: %s\n\n", flowsDir)

			// Step 5: Execute flows
			fmt.Println("🚀 Step 5: Executing flows with Maestro...")
			
			// Find flow files
			flowFiles, err := findFlowFiles(flowsDir)
			if err != nil {
				return fmt.Errorf("failed to find flows: %w", err)
			}

			// Set up capture manager
			captureManager := maestro.NewCaptureManager(cfg.Maestro.ScreenshotDir)
			if err := captureManager.PrepareDirectories(); err != nil {
				return fmt.Errorf("failed to prepare directories: %w", err)
			}

			// Execute
			executor := maestro.NewExecutor(cfg.Maestro.Path, cfg.Maestro.Browser, cfg.Maestro.Timeout)
			results, err := executor.RunFlows(flowFiles)
			if err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}

			// Print quick results
			for i, result := range results.Flows {
				status := "✅"
				if result.Status != maestro.StatusPassed {
					status = "❌"
				}
				fmt.Printf("   %s %d. %s (%s)\n", status, i+1, result.FlowName, result.Duration.Round(time.Millisecond))
			}
			fmt.Printf("\n   Success Rate: %.1f%% (%d/%d passed)\n\n", results.SuccessRate(), results.Passed, results.Total)

			// Step 6: Generate report
			fmt.Println("📊 Step 6: Generating test report...")
			generator := report.NewGenerator(
				cfg.Reporting.Format,
				cfg.Reporting.OutputDir,
				cfg.Reporting.IncludeScreenshots,
				cfg.Reporting.IncludeVideos,
			)

			reportPath, err := generator.Generate(results, gameName)
			if err != nil {
				return fmt.Errorf("failed to generate report: %w", err)
			}

			fmt.Printf("   Report: %s\n\n", reportPath)

			// Final summary
			if results.SuccessRate() == 100 {
				fmt.Println("✅ All tests passed! Game is working as expected.")
			} else {
				fmt.Printf("⚠️  %d/%d tests failed. Review the report for details.\n", 
					results.Failed+results.Timeout, results.Total)
			}

			// Exit with error if tests failed
			if results.Failed > 0 || results.Timeout > 0 {
				return fmt.Errorf("some tests failed")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&gameURL, "game", "g", "", "Live game URL (required)")
	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "Game specification file (required)")
	cmd.Flags().StringVarP(&output, "output", "o", "./flows", "Output directory for generated flows")
	cmd.Flags().StringVarP(&browser, "browser", "b", "", "Browser to use (overrides config)")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file path")

	cmd.MarkFlagRequired("game")
	cmd.MarkFlagRequired("spec")

	return cmd
}
