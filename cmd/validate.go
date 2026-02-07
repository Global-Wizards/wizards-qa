package main

import (
	"fmt"

	"github.com/Global-Wizards/wizards-qa/pkg/flows"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var (
		flowFile string
		verbose  bool
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Maestro flow syntax",
		Long: `Check if a Maestro flow file has valid YAML syntax and structure.

This command validates flow files without executing them.

Example:
  wizards-qa validate --flow flows/my-game/gameplay.yaml
  wizards-qa validate --flow flows/templates/login.yaml
  wizards-qa validate --flow flows/my-game/gameplay.yaml --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flowFile == "" {
				return fmt.Errorf("--flow file is required")
			}

			fmt.Printf("🧙‍♂️ Wizards QA - Flow Validation\n\n")
			fmt.Printf("Flow File: %s\n\n", flowFile)

			// Create validator
			validator := flows.NewValidator()

			// Validate flow
			result, err := validator.ValidateFlow(flowFile)
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			// Print results
			fmt.Printf("Status: %s\n\n", result.Summary())

			if result.HasErrors() {
				fmt.Println("❌ Errors:")
				for _, err := range result.Errors {
					fmt.Printf("  • %s\n", err)
				}
				fmt.Println()
			}

			if result.HasWarnings() {
				fmt.Println("⚠️  Warnings:")
				for _, warn := range result.Warnings {
					fmt.Printf("  • %s\n", warn)
				}
				fmt.Println()
			}

			if result.Valid && !result.HasWarnings() {
				fmt.Println("✅ Flow is valid and ready to run!")
			} else if result.Valid {
				fmt.Println("⚠️  Flow is valid but has warnings. Review before running.")
			} else {
				fmt.Println("❌ Flow has errors and cannot be run. Fix errors and try again.")
				return fmt.Errorf("validation failed")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&flowFile, "flow", "f", "", "Maestro flow file to validate (required)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed validation output")
	cmd.MarkFlagRequired("flow")

	return cmd
}
