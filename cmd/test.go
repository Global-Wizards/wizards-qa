package main

import (
	"fmt"
	"path/filepath"

	"github.com/Global-Wizards/wizards-qa/pkg/ai"
	"github.com/Global-Wizards/wizards-qa/pkg/config"
	"github.com/Global-Wizards/wizards-qa/pkg/maestro"
	"github.com/Global-Wizards/wizards-qa/pkg/report"
	"github.com/Global-Wizards/wizards-qa/pkg/util"
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

			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			if browser != "" {
				cfg.Maestro.Browser = browser
			}

			fmt.Printf("%s Wizards QA - Full E2E Test\n\n", util.EmojiWizard)
			fmt.Printf("Game URL:  %s\n", gameURL)
			fmt.Printf("Spec File: %s\n", specFile)
			fmt.Printf("AI Model:  %s\n", cfg.AI.Model)
			fmt.Printf("Browser:   %s\n\n", cfg.Maestro.Browser)

			if err := validateAPIKey(cfg); err != nil {
				return err
			}

			analyzer, err := newAnalyzer(cfg)
			if err != nil {
				return err
			}

			// Step 1: Analyze game
			analysis, err := analyzeGame(analyzer, specFile, gameURL)
			if err != nil {
				return err
			}

			// Step 2: Generate scenarios
			scenarios, err := generateScenarios(analyzer, analysis)
			if err != nil {
				return err
			}

			// Step 3: Generate flows
			flows, err := generateFlows(analyzer, scenarios)
			if err != nil {
				return err
			}

			// Step 4: Save flows
			gameName := deriveGameName(gameURL)
			flowsDir := filepath.Join(output, gameName)
			if err := saveFlows(flows, flowsDir); err != nil {
				return err
			}

			// Step 5: Execute flows
			results, err := executeFlows(cfg, flowsDir)
			if err != nil {
				return err
			}

			printResults(results)
			fmt.Println()

			// Step 6: Generate report
			reportPath, err := generateReport(cfg, results, gameName)
			if err != nil {
				return err
			}
			fmt.Printf("   Report: %s\n\n", reportPath)

			// Final summary
			if results.SuccessRate() == 100 {
				fmt.Printf("%s All tests passed! Game is working as expected.\n", util.EmojiPassed)
			} else {
				fmt.Printf("%s %d/%d tests failed. Review the report for details.\n",
					util.EmojiWarning, results.Failed+results.Timeout, results.Total)
			}

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

// analyzeGame runs AI analysis on the game spec and URL.
func analyzeGame(analyzer *ai.Analyzer, specFile, gameURL string) (*ai.AnalysisResult, error) {
	fmt.Printf("%s Step 1: Analyzing game...\n", util.EmojiClip)
	analysis, err := analyzer.AnalyzeGame(specFile, gameURL)
	if err != nil {
		return nil, fmt.Errorf("game analysis failed: %w", err)
	}

	if analysis.GameInfo.Name != "" {
		fmt.Printf("   Game: %s\n", analysis.GameInfo.Name)
	}
	fmt.Printf("   Mechanics: %d | UI Elements: %d | User Flows: %d\n\n",
		len(analysis.Mechanics), len(analysis.UIElements), len(analysis.UserFlows))
	return analysis, nil
}

// generateScenarios generates test scenarios from game analysis.
func generateScenarios(analyzer *ai.Analyzer, analysis *ai.AnalysisResult) ([]ai.TestScenario, error) {
	fmt.Printf("%s Step 2: Generating test scenarios...\n", util.EmojiTarget)
	scenarios, err := analyzer.GenerateScenarios(analysis)
	if err != nil {
		return nil, fmt.Errorf("scenario generation failed: %w", err)
	}
	fmt.Printf("   Generated %d scenario(s)\n\n", len(scenarios))
	return scenarios, nil
}

// generateFlows generates Maestro flows from test scenarios.
func generateFlows(analyzer *ai.Analyzer, scenarios []ai.TestScenario) ([]*ai.MaestroFlow, error) {
	fmt.Printf("%s Step 3: Generating Maestro flows...\n", util.EmojiHammer)
	flows, err := analyzer.GenerateFlows(scenarios)
	if err != nil {
		return nil, fmt.Errorf("flow generation failed: %w", err)
	}
	fmt.Printf("   Generated %d flow(s)\n\n", len(flows))
	return flows, nil
}

// saveFlows writes generated flows to disk.
func saveFlows(flows []*ai.MaestroFlow, flowsDir string) error {
	fmt.Printf("%s Step 4: Saving flows...\n", util.EmojiDisk)
	if err := ai.WriteFlowsToFiles(flows, flowsDir); err != nil {
		return fmt.Errorf("failed to save flows: %w", err)
	}
	fmt.Printf("   Saved to: %s\n\n", flowsDir)
	return nil
}

// executeFlows runs all Maestro flows in a directory.
func executeFlows(cfg *config.Config, flowsDir string) (*maestro.TestResults, error) {
	fmt.Printf("%s Step 5: Executing flows with Maestro...\n", util.EmojiRocket)

	flowFiles, err := findFlowFiles(flowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find flows: %w", err)
	}

	captureManager := maestro.NewCaptureManager(cfg.Maestro.ScreenshotDir)
	if err := captureManager.PrepareDirectories(); err != nil {
		return nil, fmt.Errorf("failed to prepare directories: %w", err)
	}

	executor := maestro.NewExecutor(cfg.Maestro.Path, cfg.Maestro.Browser, cfg.Maestro.Timeout)
	results, err := executor.RunFlows(flowFiles)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	return results, nil
}

// generateReport creates a test report from results.
func generateReport(cfg *config.Config, results *maestro.TestResults, gameName string) (string, error) {
	fmt.Printf("%s Step 6: Generating test report...\n", util.EmojiChart)
	generator := report.NewGenerator(
		cfg.Reporting.Format,
		cfg.Reporting.OutputDir,
		cfg.Reporting.IncludeScreenshots,
		cfg.Reporting.IncludeVideos,
	)

	reportPath, err := generator.Generate(results, gameName)
	if err != nil {
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	return reportPath, nil
}
