package main

import (
	"fmt"
	"os"

	"github.com/Global-Wizards/wizards-qa/pkg/config"
	"github.com/Global-Wizards/wizards-qa/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage wizards-qa configuration",
		Long: `View, validate, or initialize wizards-qa configuration.

Configuration is loaded from wizards-qa.yaml in the current directory,
parent directories, or home directory.

Example:
  wizards-qa config show      # Display current configuration
  wizards-qa config init      # Create example config file
  wizards-qa config validate  # Validate configuration`,
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigValidateCmd())

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Display as YAML
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			fmt.Printf("%s Current Configuration:\n\n", util.EmojiWizard)
			fmt.Println(string(data))

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file path")

	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create example configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "wizards-qa.yaml"

			// Check if file exists
			if !force {
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("config file already exists: %s (use --force to overwrite)", path)
				}
			}

			// Create default config
			cfg := config.DefaultConfig()

			// Save to file
			if err := cfg.Save(path); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("%s Created configuration file: %s\n\n", util.EmojiPassed, path)
			fmt.Println("Edit this file to customize your wizards-qa settings.")
			fmt.Println("Don't forget to set your AI API key!")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing config file")

	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				fmt.Printf("%s Configuration is invalid:\n\n", util.EmojiFailed)
				fmt.Printf("  %v\n\n", err)
				return fmt.Errorf("validation failed")
			}

			fmt.Printf("%s Configuration is valid!\n\n", util.EmojiPassed)

			// Show key settings
			fmt.Printf("AI Provider:  %s (%s)\n", cfg.AI.Provider, cfg.AI.Model)
			fmt.Printf("Maestro:      %s (browser: %s)\n", cfg.Maestro.Path, cfg.Maestro.Browser)
			fmt.Printf("Flows Dir:    %s\n", cfg.Flows.Directory)
			fmt.Printf("Reports:      %s (%s format)\n", cfg.Reporting.OutputDir, cfg.Reporting.Format)

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Config file path")

	return cmd
}
