package flows

// Flow represents a Maestro flow file structure
type Flow struct {
	AppId    string        `yaml:"appId,omitempty"`
	URL      string        `yaml:"url,omitempty"`
	Name     string        `yaml:"name,omitempty"`
	Tags     []string      `yaml:"tags,omitempty"`
	Commands []interface{} `yaml:"commands,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for Flow
// Maestro flows have a special format where metadata is before "---" and commands after
func (f *Flow) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Parse as generic map first
	var data map[string]interface{}
	if err := unmarshal(&data); err != nil {
		return err
	}

	// Extract known fields
	if appId, ok := data["appId"].(string); ok {
		f.AppId = appId
	}
	if url, ok := data["url"].(string); ok {
		f.URL = url
	}
	if name, ok := data["name"].(string); ok {
		f.Name = name
	}
	if tags, ok := data["tags"].([]interface{}); ok {
		f.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				f.Tags[i] = tagStr
			}
		}
	}

	// Everything else goes into commands (this is simplified)
	// Real Maestro parsing would split on "---" separator
	f.Commands = make([]interface{}, 0)
	for key, value := range data {
		// Skip known metadata fields
		if key == "appId" || key == "url" || key == "name" || key == "tags" {
			continue
		}

		// Everything else is a command
		f.Commands = append(f.Commands, map[string]interface{}{key: value})
	}

	return nil
}

// ValidationResult represents the result of validating a single flow
type ValidationResult struct {
	FlowPath string   `json:"flowPath"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// MultiValidationResult represents results from validating multiple flows
type MultiValidationResult struct {
	Total   int                 `json:"total"`
	Valid   int                 `json:"valid"`
	Invalid int                 `json:"invalid"`
	Results []*ValidationResult `json:"results"`
}

// HasErrors returns true if any validation errors exist
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if any validation warnings exist
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// Summary returns a human-readable summary
func (r *ValidationResult) Summary() string {
	if r.Valid && !r.HasWarnings() {
		return "✅ Valid"
	}
	if r.Valid && r.HasWarnings() {
		return "⚠️  Valid with warnings"
	}
	return "❌ Invalid"
}
