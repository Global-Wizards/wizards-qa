package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "0.2.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "wizards-qa",
		Short: "AI-powered QA automation for Phaser 4 games",
		Long: `Wizards QA is an intelligent testing system that analyzes Phaser 4 web games
and generates comprehensive Maestro test flows using AI.

Usage:
  wizards-qa test --game URL --spec spec.md    # Full E2E testing
  wizards-qa generate --game URL --spec spec.md # Generate flows only
  wizards-qa run --flows flows/                # Execute existing flows
  wizards-qa validate --flow flow.yaml         # Validate flow syntax`,
		Version: version,
	}

	// Add commands
	rootCmd.AddCommand(newTestCmd())
	rootCmd.AddCommand(newGenerateCmd())
	rootCmd.AddCommand(newScoutCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newTemplateCmd())
	rootCmd.AddCommand(newConfigCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
