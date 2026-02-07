package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Global-Wizards/wizards-qa/pkg/util"
	"github.com/spf13/cobra"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage flow templates",
		Long: `List, show, and apply reusable Maestro flow templates.

Templates are pre-built flow patterns for common game testing scenarios.

Example:
  wizards-qa template list              # List all templates
  wizards-qa template show click-object # Show template content
  wizards-qa template apply click-object --output flows/my-game/ \\
    --var GAME_URL=https://game.com \\
    --var X_COORD=50%`,
	}

	cmd.AddCommand(newTemplateListCmd())
	cmd.AddCommand(newTemplateShowCmd())
	cmd.AddCommand(newTemplateApplyCmd())

	return cmd
}

func newTemplateListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			templatesDir := "flows/templates"
			
			fmt.Printf("%s Available Templates:\n\n", util.EmojiWizard)

			// Find all template files
			templates, err := findTemplates(templatesDir)
			if err != nil {
				return fmt.Errorf("failed to find templates: %w", err)
			}

			if len(templates) == 0 {
				fmt.Println("No templates found in", templatesDir)
				return nil
			}

			// Group by category
			categories := make(map[string][]string)
			for _, t := range templates {
				dir := filepath.Dir(t)
				category := filepath.Base(dir)
				if category == "templates" {
					category = "Main"
				}
				categories[category] = append(categories[category], filepath.Base(t))
			}

			// Print grouped templates
			for category, files := range categories {
				fmt.Printf("%s %s\n", util.EmojiFolder, category)
				for _, file := range files {
					fmt.Printf("   • %s\n", file)
				}
				fmt.Println()
			}

			fmt.Println("Use 'wizards-qa template show <name>' to view a template")
			fmt.Println("See flows/templates/README.md for detailed documentation")

			return nil
		},
	}

	return cmd
}

func newTemplateShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <template-name>",
		Short: "Show template content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]
			templatesDir := "flows/templates"

			// Find template file
			templatePath, err := findTemplate(templatesDir, templateName)
			if err != nil {
				return fmt.Errorf("template not found: %s", templateName)
			}

			// Read and display content
			content, err := os.ReadFile(templatePath)
			if err != nil {
				return fmt.Errorf("failed to read template: %w", err)
			}

			fmt.Printf("%s Template: %s\n", util.EmojiWizard, templateName)
			fmt.Printf("%s Path: %s\n\n", util.EmojiClip, templatePath)
			fmt.Println(string(content))

			return nil
		},
	}

	return cmd
}

func newTemplateApplyCmd() *cobra.Command {
	var outputDir string
	var outputFile string
	var variables []string

	cmd := &cobra.Command{
		Use:   "apply <template-name>",
		Short: "Apply a template with variable substitution",
		Long: `Apply a template and replace variables with actual values.

Variables are specified with --var KEY=VALUE and replace {{KEY}} in the template.

Example:
  wizards-qa template apply click-object \\
    --output flows/my-game/click-button.yaml \\
    --var GAME_URL=https://game.com \\
    --var BUTTON_TEXT="Start Game" \\
    --var X_COORD=50% \\
    --var Y_COORD=50% \\
    --var OBJECT_NAME=start-button \\
    --var EXPECTED_TEXT="Playing..."`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]
			templatesDir := "flows/templates"

			// Find template
			templatePath, err := findTemplate(templatesDir, templateName)
			if err != nil {
				return fmt.Errorf("template not found: %s", templateName)
			}

			// Read template
			content, err := os.ReadFile(templatePath)
			if err != nil {
				return fmt.Errorf("failed to read template: %w", err)
			}

			// Parse variables
			varMap := make(map[string]string)
			for _, v := range variables {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid variable format: %s (use KEY=VALUE)", v)
				}
				varMap[parts[0]] = parts[1]
			}

			// Replace variables
			result := string(content)
			for key, value := range varMap {
				placeholder := "{{" + key + "}}"
				result = strings.ReplaceAll(result, placeholder, value)
			}

			// Check for unreplaced variables
			if strings.Contains(result, "{{") {
				fmt.Printf("%s Warning: Some variables may not have been replaced:\n", util.EmojiWarning)
				// Extract unreplaced variables
				lines := strings.Split(result, "\n")
				for _, line := range lines {
					if strings.Contains(line, "{{") {
						fmt.Printf("   %s\n", strings.TrimSpace(line))
					}
				}
				fmt.Println()
			}

			// Determine output path
			var outputPath string
			if outputFile != "" {
				outputPath = outputFile
			} else if outputDir != "" {
				filename := strings.TrimSuffix(filepath.Base(templatePath), ".yaml") + "-applied.yaml"
				outputPath = filepath.Join(outputDir, filename)
			} else {
				return fmt.Errorf("either --output or --output-dir must be specified")
			}

			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			// Write result
			if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}

			fmt.Printf("%s Template applied successfully!\n", util.EmojiPassed)
			fmt.Printf("   Output: %s\n\n", outputPath)
			fmt.Println("You can now run this flow with:")
			fmt.Printf("  wizards-qa run --flows %s\n", filepath.Dir(outputPath))

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "d", "", "Output directory")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path")
	cmd.Flags().StringArrayVarP(&variables, "var", "v", []string{}, "Template variable (KEY=VALUE)")

	return cmd
}

// findTemplates finds all template files in a directory
func findTemplates(dir string) ([]string, error) {
	var templates []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			templates = append(templates, path)
		}
		return nil
	})

	return templates, err
}

// findTemplate finds a specific template by name (with or without extension)
func findTemplate(dir, name string) (string, error) {
	// Try exact match first
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Try with .yaml extension
	path = filepath.Join(dir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Try in subdirectories
	templates, err := findTemplates(dir)
	if err != nil {
		return "", err
	}

	for _, t := range templates {
		base := strings.TrimSuffix(filepath.Base(t), filepath.Ext(t))
		if base == name {
			return t, nil
		}
	}

	return "", fmt.Errorf("template not found: %s", name)
}
