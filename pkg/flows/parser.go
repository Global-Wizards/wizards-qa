package flows

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseMaestroFlow parses a Maestro flow file
// Maestro flows have format:
// ---
// metadata (appId, url, etc.)
// ---
// - command1
// - command2
func ParseMaestroFlow(data []byte) (*Flow, error) {
	flow := &Flow{
		Commands: make([]interface{}, 0),
	}

	// Split on "---" separator
	parts := bytes.Split(data, []byte("\n---\n"))

	if len(parts) == 1 {
		// No separator, treat as single YAML doc
		// Try to parse as flow with commands array
		var rawFlow struct {
			AppId    string        `yaml:"appId"`
			URL      string        `yaml:"url"`
			Name     string        `yaml:"name"`
			Tags     []string      `yaml:"tags"`
			Commands []interface{} `yaml:",inline"`
		}
		if err := yaml.Unmarshal(data, &rawFlow); err != nil {
			return nil, fmt.Errorf("failed to parse flow: %w", err)
		}

		flow.AppId = rawFlow.AppId
		flow.URL = rawFlow.URL
		flow.Name = rawFlow.Name
		flow.Tags = rawFlow.Tags

		// Try to extract commands as array
		// Maestro flows are typically just an array after metadata
		var commands []interface{}
		if err := yaml.Unmarshal(data, &commands); err == nil {
			flow.Commands = commands
		}

		return flow, nil
	}

	// Parse metadata (before "---")
	if len(parts) > 0 && len(parts[0]) > 0 {
		var metadata map[string]interface{}
		if err := yaml.Unmarshal(parts[0], &metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		if appId, ok := metadata["appId"].(string); ok {
			flow.AppId = appId
		}
		if url, ok := metadata["url"].(string); ok {
			flow.URL = url
		}
		if name, ok := metadata["name"].(string); ok {
			flow.Name = name
		}
		if tags, ok := metadata["tags"].([]interface{}); ok {
			flow.Tags = make([]string, len(tags))
			for i, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					flow.Tags[i] = tagStr
				}
			}
		}
	}

	// Parse commands (after "---")
	if len(parts) > 1 && len(parts[1]) > 0 {
		// Commands are a YAML array
		var commands []interface{}
		
		// Clean up the commands section
		commandsStr := string(parts[1])
		commandsStr = strings.TrimSpace(commandsStr)
		
		if err := yaml.Unmarshal([]byte(commandsStr), &commands); err != nil {
			return nil, fmt.Errorf("failed to parse commands: %w", err)
		}

		flow.Commands = commands
	}

	return flow, nil
}
