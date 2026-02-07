package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/maestro"
	"github.com/Global-Wizards/wizards-qa/pkg/report"
	"github.com/Global-Wizards/wizards-qa/pkg/util"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var (
		flowsDir   string
		browser    string
		gameName   string
		configPath string
		timeout    string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute existing Maestro test flows",
		Long: `Run pre-generated Maestro flows and collect results.

This command executes existing flows without AI analysis:
1. Load flows from directory
2. Execute each flow via Maestro CLI
3. Collect screenshots and results
4. Generate test report

Useful when flows are already generated or maintained manually.

Example:
  wizards-qa run --flows flows/my-game/
  wizards-qa run --flows flows/my-game/ --browser firefox
  wizards-qa run --flows flows/my-game/ --name "My Game"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flowsDir == "" {
				return fmt.Errorf("--flows directory is required")
			}

			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			if browser != "" {
				cfg.Maestro.Browser = browser
			}
			if timeout != "" {
				duration, err := time.ParseDuration(timeout)
				if err != nil {
					return fmt.Errorf("invalid timeout: %w", err)
				}
				cfg.Maestro.Timeout = duration
			}

			fmt.Printf("%s Wizards QA - Flow Execution\n\n", util.EmojiWizard)
			fmt.Printf("Flows Dir: %s\n", flowsDir)
			fmt.Printf("Browser:   %s\n", cfg.Maestro.Browser)
			fmt.Printf("Timeout:   %s\n\n", cfg.Maestro.Timeout)

			// Find all flow files (recursive)
			flowFiles, err := findFlowFiles(flowsDir)
			if err != nil {
				return fmt.Errorf("failed to find flows: %w", err)
			}

			if len(flowFiles) == 0 {
				return fmt.Errorf("no flow files found in %s", flowsDir)
			}

			fmt.Printf("Found %d flow file(s)\n\n", len(flowFiles))

			// Set up capture manager
			captureManager := maestro.NewCaptureManager(cfg.Maestro.ScreenshotDir)
			if err := captureManager.PrepareDirectories(); err != nil {
				return fmt.Errorf("failed to prepare capture directories: %w", err)
			}

			fmt.Printf("Capture directory: %s\n\n", captureManager.GetRunDir())

			// Execute flows
			executor := maestro.NewExecutor(cfg.Maestro.Path, cfg.Maestro.Browser, cfg.Maestro.Timeout)

			fmt.Println("Executing flows...")
			fmt.Println()
			results, err := executor.RunFlows(flowFiles)
			if err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}

			// Print results
			fmt.Println("\n--- Results ---")
			printResults(results)

			fmt.Printf("\n--- Summary ---\n")
			fmt.Printf("Total:   %d\n", results.Total)
			fmt.Printf("Passed:  %d\n", results.Passed)
			fmt.Printf("Failed:  %d\n", results.Failed)
			fmt.Printf("Timeout: %d\n", results.Timeout)
			fmt.Printf("Success: %.1f%%\n\n", results.SuccessRate())

			// Generate report
			if gameName == "" {
				gameName = filepath.Base(flowsDir)
			}

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

			fmt.Printf("%s Report generated: %s\n", util.EmojiPassed, reportPath)

			if results.Failed > 0 || results.Timeout > 0 {
				return fmt.Errorf("some tests failed")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&flowsDir, "flows", "f", "", "Directory containing Maestro flows (required)")
	cmd.Flags().StringVarP(&browser, "browser", "b", "", "Browser to use (overrides config)")
	cmd.Flags().StringVarP(&gameName, "name", "n", "", "Game name for report (default: directory name)")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file path")
	cmd.Flags().StringVarP(&timeout, "timeout", "t", "", "Test timeout (e.g. 5m, 300s)")

	cmd.MarkFlagRequired("flows")

	return cmd
}

// findFlowFiles finds all .yaml/.yml files in a directory, searching recursively.
func findFlowFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(info.Name())
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}
