package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var (
		flowFile string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Maestro flow syntax",
		Long: `Check if a Maestro flow file has valid YAML syntax and structure.

This command validates flow files without executing them.

Example:
  wizards-qa validate --flow flows/my-game/gameplay.yaml
  wizards-qa validate --flow flows/templates/login.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flowFile == "" {
				return fmt.Errorf("--flow file is required")
			}

			fmt.Printf("🧙‍♂️ Wizards QA - Flow Validation\n\n")
			fmt.Printf("Flow File: %s\n\n", flowFile)

			// TODO: Implement flow validation
			// 1. Load flow file
			// 2. Parse YAML
			// 3. Validate structure
			// 4. Check command syntax

			fmt.Println("⚠️  Flow validation not yet implemented")
			fmt.Println("Coming soon in Phase 1!")

			return nil
		},
	}

	cmd.Flags().StringVarP(&flowFile, "flow", "f", "", "Maestro flow file to validate (required)")
	cmd.MarkFlagRequired("flow")

	return cmd
}
