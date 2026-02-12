package main

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// maestroAllowedCommands is the set of valid Maestro CLI commands.
var maestroAllowedCommands = map[string]bool{
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
	"openLink":          true,
	"hideKeyboard":      true,
	"pressKey":          true,
	"eraseText":         true,
	"clearState":        true,
	"stopApp":           true,
	"runFlow":           true,
	"repeat":            true,
	"evalScript":        true,
	"copyTextFrom":      true,
	"inputRandomText":   true,
	"inputRandomNumber": true,
	"inputRandomEmail":  true,
	"inputRandomPersonName": true,
	"waitForAnimationToEnd":  true,
	"assertTrue":        true,
	"setLocation":       true,
	"travel":            true,
	"startRecording":    true,
	"stopRecording":     true,
	"addMedia":          true,
}

// deprecatedCommands maps old/alias names to their correct Maestro equivalents.
var deprecatedCommands = map[string]string{
	"waitFor":     "extendedWaitUntil",
	"screenshot":  "takeScreenshot",
	"openBrowser": "openLink",
	"wait":        "extendedWaitUntil",
}

// flowValidationResult is the JSON response for the validate endpoint.
type flowValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// validateMaestroYAML validates raw YAML content as a Maestro flow.
func validateMaestroYAML(content string) *flowValidationResult {
	result := &flowValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	content = strings.TrimSpace(content)
	if content == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Empty YAML content")
		return result
	}

	// Phase 1: Basic YAML syntax check
	var syntaxCheck interface{}
	if err := yaml.Unmarshal([]byte(content), &syntaxCheck); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("YAML syntax error: %v", err))
		return result
	}

	// Phase 2: Parse Maestro flow structure (metadata + commands split by ---)
	appId, url, commands, parseErrors := parseMaestroContent(content)
	if len(parseErrors) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, parseErrors...)
		return result
	}

	// Phase 3: Structure validation
	if appId == "" && url == "" {
		result.Warnings = append(result.Warnings, "Neither 'appId' nor 'url' specified — flow may not launch correctly")
	}

	if len(commands) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Flow has no commands — Maestro requires at least one command after the '---' separator")
		return result
	}

	// Phase 4: Command validation
	for i, cmd := range commands {
		cmdNum := i + 1
		validateCommand(cmd, cmdNum, result)
	}

	return result
}

// parseMaestroContent splits a Maestro flow on "---" and parses metadata + commands.
func parseMaestroContent(content string) (appId, url string, commands []interface{}, errors []string) {
	data := []byte(content)
	parts := bytes.Split(data, []byte("\n---\n"))

	// Also handle "---\n" at the very start of the file
	if bytes.HasPrefix(data, []byte("---\n")) {
		rest := data[4:]
		innerParts := bytes.SplitN(rest, []byte("\n---\n"), 2)
		if len(innerParts) == 2 {
			parts = [][]byte{innerParts[0], innerParts[1]}
		} else {
			parts = [][]byte{innerParts[0]}
		}
	}

	if len(parts) == 1 {
		// No --- separator: try to parse as a single document
		// Could be just a command list, or metadata + commands mixed
		var rawList []interface{}
		if err := yaml.Unmarshal(data, &rawList); err == nil && len(rawList) > 0 {
			// It's a bare command list (no metadata)
			commands = rawList
			return
		}

		// Try as a map (metadata only or metadata + inline commands)
		var rawMap map[string]interface{}
		if err := yaml.Unmarshal(data, &rawMap); err == nil {
			if v, ok := rawMap["appId"].(string); ok {
				appId = v
			}
			if v, ok := rawMap["url"].(string); ok {
				url = v
			}
			// No commands section found
			errors = append(errors, "No '---' separator found — Maestro flows require metadata (appId/url) above '---' and commands below it")
			return
		}

		errors = append(errors, "Could not parse flow structure — expected a YAML list of commands or a metadata map")
		return
	}

	// Parse metadata (before ---)
	if len(parts[0]) > 0 {
		var metadata map[string]interface{}
		if err := yaml.Unmarshal(parts[0], &metadata); err != nil {
			errors = append(errors, fmt.Sprintf("Metadata section (above '---') has invalid YAML: %v", err))
			return
		}
		if v, ok := metadata["appId"].(string); ok {
			appId = v
		}
		if v, ok := metadata["url"].(string); ok {
			url = v
		}
	}

	// Parse commands (after ---)
	if len(parts) > 1 && len(bytes.TrimSpace(parts[1])) > 0 {
		var rawCmds []interface{}
		if err := yaml.Unmarshal(parts[1], &rawCmds); err != nil {
			errors = append(errors, fmt.Sprintf("Commands section (below '---') has invalid YAML: %v", err))
			return
		}
		commands = rawCmds
	}

	return
}

// validateCommand checks a single Maestro command for validity.
func validateCommand(cmd interface{}, cmdNum int, result *flowValidationResult) {
	switch c := cmd.(type) {
	case string:
		// Simple string command like "back" or "hideKeyboard"
		if !maestroAllowedCommands[c] {
			if replacement, ok := deprecatedCommands[c]; ok {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d: '%s' is not a valid Maestro command — use '%s' instead", cmdNum, c, replacement))
				result.Valid = false
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d: unknown command '%s'", cmdNum, c))
				result.Valid = false
			}
		}

	case map[string]interface{}:
		if len(c) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Command %d: empty command object", cmdNum))
			result.Valid = false
			return
		}

		for cmdName, value := range c {
			// Check for deprecated/alias commands
			if replacement, ok := deprecatedCommands[cmdName]; ok {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d: '%s' is not a valid Maestro command — use '%s' instead", cmdNum, cmdName, replacement))
				result.Valid = false
				break
			}

			if !maestroAllowedCommands[cmdName] {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d: unknown command '%s'", cmdNum, cmdName))
				result.Valid = false
				break
			}

			// Command-specific validation
			validateCommandValue(cmdName, value, cmdNum, result)
			break // Only the first key is the command
		}

	default:
		result.Errors = append(result.Errors, fmt.Sprintf("Command %d: invalid command type (expected string or map)", cmdNum))
		result.Valid = false
	}
}

// validateCommandValue does deep validation of specific command arguments.
func validateCommandValue(cmdName string, value interface{}, cmdNum int, result *flowValidationResult) {
	switch cmdName {
	case "tapOn":
		switch val := value.(type) {
		case string:
			if val == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Command %d (tapOn): empty text selector", cmdNum))
			}
		case map[string]interface{}:
			if _, has := val["visible"]; has {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d (tapOn): 'visible' is not valid here — use tapOn: \"text\" directly or extendedWaitUntil for waiting", cmdNum))
				result.Valid = false
			}
			if _, has := val["notVisible"]; has {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d (tapOn): 'notVisible' is not valid here — use tapOn: \"text\" directly", cmdNum))
				result.Valid = false
			}
			if point, ok := val["point"]; ok {
				if pointStr, ok := point.(string); ok {
					if !strings.Contains(pointStr, ",") {
						result.Errors = append(result.Errors, fmt.Sprintf("Command %d (tapOn): point must be 'x,y' format (e.g. \"50%%,50%%\"), got '%s'", cmdNum, pointStr))
						result.Valid = false
					}
				}
			}
			// Check for at least one valid selector
			_, hasText := val["text"]
			_, hasId := val["id"]
			_, hasPoint := val["point"]
			_, hasIndex := val["index"]
			if !hasText && !hasId && !hasPoint && !hasIndex {
				// Check if the only keys are invalid ones
				validKeys := map[string]bool{"text": true, "id": true, "point": true, "index": true, "retryTapIfNoChange": true, "longPressTimeout": true, "repeat": true, "delay": true, "waitUntilVisible": true, "label": true, "optional": true, "enabled": true, "selected": true, "checked": true, "focused": true, "childOf": true, "containsChild": true, "containsDescendants": true, "below": true, "above": true}
				hasAnyValid := false
				for k := range val {
					if validKeys[k] {
						hasAnyValid = true
						break
					}
				}
				if !hasAnyValid && len(val) > 0 {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Command %d (tapOn): no recognizable selector (text, id, or point) found", cmdNum))
				}
			}
		case nil:
			result.Errors = append(result.Errors, fmt.Sprintf("Command %d (tapOn): missing selector — provide a text string or map with text/id/point", cmdNum))
			result.Valid = false
		}

	case "inputText":
		if str, ok := value.(string); ok {
			if str == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Command %d (inputText): empty text value", cmdNum))
			}
		} else if value == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Command %d (inputText): missing text value", cmdNum))
			result.Valid = false
		}

	case "assertVisible", "assertNotVisible":
		switch val := value.(type) {
		case string:
			if val == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d (%s): empty assertion text", cmdNum, cmdName))
				result.Valid = false
			}
		case map[string]interface{}:
			if _, has := val["visible"]; has {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d (%s): 'visible' is not valid here — use %s: \"text\" directly", cmdNum, cmdName, cmdName))
				result.Valid = false
			}
			if _, has := val["notVisible"]; has {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d (%s): 'notVisible' is not valid here — use %s: \"text\" directly", cmdNum, cmdName, cmdName))
				result.Valid = false
			}
		case nil:
			result.Errors = append(result.Errors, fmt.Sprintf("Command %d (%s): missing assertion value", cmdNum, cmdName))
			result.Valid = false
		}

	case "extendedWaitUntil":
		if m, ok := value.(map[string]interface{}); ok {
			_, hasVisible := m["visible"]
			_, hasNotVisible := m["notVisible"]
			if !hasVisible && !hasNotVisible {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d (extendedWaitUntil): requires 'visible' or 'notVisible' condition — timeout alone is invalid in Maestro", cmdNum))
				result.Valid = false
			}
		} else if value == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Command %d (extendedWaitUntil): missing configuration — needs visible/notVisible and optional timeout", cmdNum))
			result.Valid = false
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("Command %d (extendedWaitUntil): must be a map with visible/notVisible, not a %T", cmdNum, value))
			result.Valid = false
		}

	case "openLink":
		if str, ok := value.(string); ok {
			if str == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d (openLink): empty URL", cmdNum))
				result.Valid = false
			}
		} else if m, ok := value.(map[string]interface{}); ok {
			// openLink: {url: "..."} is a common AI mistake — should be openLink: "..."
			if _, has := m["url"]; has {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Command %d (openLink): use openLink: \"url\" format, not openLink: {url: \"...\"}", cmdNum))
			}
		} else if value == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Command %d (openLink): missing URL", cmdNum))
			result.Valid = false
		}

	case "runFlow":
		if str, ok := value.(string); ok {
			if str == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d (runFlow): empty flow path", cmdNum))
				result.Valid = false
			} else if !strings.HasSuffix(str, ".yaml") && !strings.HasSuffix(str, ".yml") {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Command %d (runFlow): '%s' does not end with .yaml — Maestro expects a YAML file path", cmdNum, str))
			}
		}

	case "scroll":
		// scroll should be a map or a string
		if m, ok := value.(map[string]interface{}); ok {
			// Validate direction if present
			if dir, ok := m["direction"].(string); ok {
				validDirs := map[string]bool{"UP": true, "DOWN": true, "LEFT": true, "RIGHT": true}
				if !validDirs[strings.ToUpper(dir)] {
					result.Errors = append(result.Errors, fmt.Sprintf("Command %d (scroll): invalid direction '%s' — use UP, DOWN, LEFT, or RIGHT", cmdNum, dir))
					result.Valid = false
				}
			}
		}

	case "swipe":
		if m, ok := value.(map[string]interface{}); ok {
			_, hasStart := m["start"]
			_, hasEnd := m["end"]
			_, hasDirection := m["direction"]
			if !hasDirection && (!hasStart || !hasEnd) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Command %d (swipe): should have either 'direction' or both 'start' and 'end' points", cmdNum))
			}
		}

	case "repeat":
		if m, ok := value.(map[string]interface{}); ok {
			_, hasTimes := m["times"]
			_, hasWhileVisible := m["whileVisible"]
			_, hasWhileNotVisible := m["whileNotVisible"]
			_, hasWhileTrue := m["whileTrue"]
			if !hasTimes && !hasWhileVisible && !hasWhileNotVisible && !hasWhileTrue {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Command %d (repeat): should have 'times', 'whileVisible', 'whileNotVisible', or 'whileTrue'", cmdNum))
			}
			if cmds, ok := m["commands"]; ok {
				if cmdList, ok := cmds.([]interface{}); ok {
					for j, subCmd := range cmdList {
						validateCommand(subCmd, cmdNum*100+j+1, result)
					}
				}
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("Command %d (repeat): missing 'commands' list", cmdNum))
				result.Valid = false
			}
		}
	}
}
