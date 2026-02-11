package flows

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Validator validates Maestro flow files
type Validator struct {
	AllowedCommands map[string]bool
}

// NewValidator creates a new flow validator
func NewValidator() *Validator {
	return &Validator{
		AllowedCommands: map[string]bool{
			"launchApp":         true,
			"tapOn":             true,
			"inputText":         true,
			"assertVisible":     true,
			"assertNotVisible":  true,
			"extendedWaitUntil": true,
			"scroll":            true,
			"swipe":             true,
			"back":              true,
			"takeScreenshot":    true,
			"openBrowser":       true,
			"hideKeyboard":      true,
			"pressKey":          true,
			"eraseText":         true,
			"clearState":        true,
			"stopApp":           true,
			"runFlow":           true,
			"repeat":            true,
			"evalScript":        true,
		},
	}
}

// ValidateFlow validates a Maestro flow file
func (v *Validator) ValidateFlow(flowPath string) (*ValidationResult, error) {
	result := &ValidationResult{
		FlowPath: flowPath,
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Check if file exists
	absPath, err := filepath.Abs(flowPath)
	if err != nil {
		return nil, fmt.Errorf("invalid flow path: %w", err)
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("file not found: %s", absPath))
		return result, nil
	}

	if fileInfo.IsDir() {
		result.Valid = false
		result.Errors = append(result.Errors, "path is a directory, not a file")
		return result, nil
	}

	// Check file extension
	if !strings.HasSuffix(strings.ToLower(flowPath), ".yaml") && !strings.HasSuffix(strings.ToLower(flowPath), ".yml") {
		result.Warnings = append(result.Warnings, "file does not have .yaml or .yml extension")
	}

	// Read file content
	data, err := os.ReadFile(absPath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to read file: %v", err))
		return result, nil
	}

	// Validate YAML syntax
	if err := v.ValidateYAML(data); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("YAML syntax error: %v", err))
		return result, nil
	}

	// Parse flow structure
	flow, err := ParseMaestroFlow(data)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("flow structure error: %v", err))
		return result, nil
	}

	// Validate flow structure
	if err := v.ValidateFlowStructure(flow, result); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result, nil
	}

	// Validate commands
	v.ValidateCommands(flow, result)

	return result, nil
}

// ValidateYAML checks if the data is valid YAML
func (v *Validator) ValidateYAML(data []byte) error {
	var obj interface{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	return nil
}

// ParseFlow parses a Maestro flow file
func (v *Validator) ParseFlow(data []byte) (*Flow, error) {
	flow := &Flow{}
	if err := yaml.Unmarshal(data, flow); err != nil {
		return nil, fmt.Errorf("failed to parse flow: %w", err)
	}
	return flow, nil
}

// ValidateFlowStructure validates the overall flow structure
func (v *Validator) ValidateFlowStructure(flow *Flow, result *ValidationResult) error {
	// Check for app configuration (appId or url)
	if flow.AppId == "" && flow.URL == "" {
		result.Warnings = append(result.Warnings, "neither 'appId' nor 'url' specified - flow may not launch correctly")
	}

	// Check if there are any commands
	if len(flow.Commands) == 0 {
		result.Valid = false
		return fmt.Errorf("flow has no commands")
	}

	return nil
}

// ValidateCommands validates individual flow commands
func (v *Validator) ValidateCommands(flow *Flow, result *ValidationResult) {
	for i, cmd := range flow.Commands {
		cmdNum := i + 1

		// Check if command is a map (single command) or string (simple command)
		switch c := cmd.(type) {
		case string:
			// Simple command like "launchApp", "back", etc.
			if !v.AllowedCommands[c] {
				result.Warnings = append(result.Warnings, fmt.Sprintf("command %d: unknown command '%s'", cmdNum, c))
			}

		case map[string]interface{}:
			// Complex command like "tapOn: ..."
			if len(c) == 0 {
				result.Errors = append(result.Errors, fmt.Sprintf("command %d: empty command object", cmdNum))
				result.Valid = false
				continue
			}

			// Get command name (first key)
			for cmdName := range c {
				if !v.AllowedCommands[cmdName] {
					result.Warnings = append(result.Warnings, fmt.Sprintf("command %d: unknown command '%s'", cmdNum, cmdName))
				}

				// Validate specific commands
				v.validateSpecificCommand(cmdName, c[cmdName], cmdNum, result)
				break // Only check first key
			}

		default:
			result.Errors = append(result.Errors, fmt.Sprintf("command %d: invalid command type", cmdNum))
			result.Valid = false
		}
	}
}

// validateSpecificCommand validates specific command parameters
func (v *Validator) validateSpecificCommand(cmdName string, value interface{}, cmdNum int, result *ValidationResult) {
	switch cmdName {
	case "tapOn":
		// tapOn should have either a string (text) or map (point, etc.)
		switch val := value.(type) {
		case string:
			if val == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("command %d (tapOn): empty text selector", cmdNum))
			}
		case map[string]interface{}:
			// Check for point coordinates
			if point, ok := val["point"]; ok {
				if pointStr, ok := point.(string); ok {
					// Validate point format (e.g., "50%,50%" or "100,200")
					if !strings.Contains(pointStr, ",") {
						result.Warnings = append(result.Warnings, fmt.Sprintf("command %d (tapOn): point should be 'x,y' format", cmdNum))
					}
				}
			}
		}

	case "inputText":
		// inputText should have a string value
		if str, ok := value.(string); ok {
			if str == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("command %d (inputText): empty text", cmdNum))
			}
		}

	case "assertVisible", "assertNotVisible":
		// Should have text or selector
		if str, ok := value.(string); ok {
			if str == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("command %d (%s): empty assertion", cmdNum, cmdName))
				result.Valid = false
			}
		}

	case "extendedWaitUntil":
		// Should have visible, text, or timeout
		if m, ok := value.(map[string]interface{}); ok {
			hasCondition := false
			if _, ok := m["visible"]; ok {
				hasCondition = true
			}
			if _, ok := m["text"]; ok {
				hasCondition = true
			}
			if _, ok := m["timeout"]; ok {
				// timeout is fine alone
			} else if !hasCondition {
				result.Warnings = append(result.Warnings, fmt.Sprintf("command %d (extendedWaitUntil): should have 'visible' or 'text' condition", cmdNum))
			}
		}
	}
}

// ValidateFlows validates multiple flow files
func (v *Validator) ValidateFlows(flowPaths []string) (*MultiValidationResult, error) {
	multiResult := &MultiValidationResult{
		Total:   len(flowPaths),
		Valid:   0,
		Invalid: 0,
		Results: make([]*ValidationResult, 0, len(flowPaths)),
	}

	for _, flowPath := range flowPaths {
		result, err := v.ValidateFlow(flowPath)
		if err != nil {
			return multiResult, fmt.Errorf("validation error for %s: %w", flowPath, err)
		}

		multiResult.Results = append(multiResult.Results, result)

		if result.Valid {
			multiResult.Valid++
		} else {
			multiResult.Invalid++
		}
	}

	return multiResult, nil
}
