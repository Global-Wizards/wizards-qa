package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the wizards-qa configuration
type Config struct {
	AI       AIConfig       `yaml:"ai"`
	Maestro  MaestroConfig  `yaml:"maestro"`
	Flows    FlowsConfig    `yaml:"flows"`
	Reporting ReportingConfig `yaml:"reporting"`
	Browser  BrowserConfig  `yaml:"browser"`
}

// AIConfig contains AI provider settings
type AIConfig struct {
	Provider    string  `yaml:"provider"`    // anthropic, google, openai
	Model       string  `yaml:"model"`       // claude-sonnet-4-5, gemini-pro, etc.
	APIKey      string  `yaml:"apiKey"`      // Can use ${ENV_VAR} syntax
	Temperature float64 `yaml:"temperature"` // 0.0 - 1.0
	MaxTokens   int     `yaml:"maxTokens"`   // Max response tokens
}

// MaestroConfig contains Maestro CLI settings
type MaestroConfig struct {
	Path          string        `yaml:"path"`          // Path to maestro binary
	Browser       string        `yaml:"browser"`       // chrome, firefox, safari
	Timeout       time.Duration `yaml:"timeout"`       // Test timeout
	ScreenshotDir string        `yaml:"screenshotDir"` // Screenshot output directory
	VideoCapture  bool          `yaml:"videoCapture"`  // Enable video recording
}

// FlowsConfig contains flow management settings
type FlowsConfig struct {
	Directory string `yaml:"directory"` // Flow storage directory
	Templates string `yaml:"templates"` // Template directory
	GitCommit bool   `yaml:"gitCommit"` // Auto-commit generated flows
	GitRepo   string `yaml:"gitRepo"`   // Git repository URL
}

// ReportingConfig contains test reporting settings
type ReportingConfig struct {
	Format             string `yaml:"format"`             // markdown, json, junit
	OutputDir          string `yaml:"outputDir"`          // Report output directory
	IncludeScreenshots bool   `yaml:"includeScreenshots"` // Embed screenshots in reports
	IncludeVideos      bool   `yaml:"includeVideos"`      // Embed videos in reports
}

// BrowserConfig contains browser automation settings (for game analysis)
type BrowserConfig struct {
	Headless bool `yaml:"headless"` // Run browser in headless mode
	Viewport struct {
		Width  int `yaml:"width"`
		Height int `yaml:"height"`
	} `yaml:"viewport"`
	Timeout time.Duration `yaml:"timeout"` // Page load timeout
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		AI: AIConfig{
			Provider:    "anthropic",
			Model:       "claude-sonnet-4-5",
			APIKey:      "${ANTHROPIC_API_KEY}",
			Temperature: 0.7,
			MaxTokens:   8000,
		},
		Maestro: MaestroConfig{
			Path:          "maestro",
			Browser:       "chrome",
			Timeout:       300 * time.Second,
			ScreenshotDir: "./screenshots",
			VideoCapture:  false,
		},
		Flows: FlowsConfig{
			Directory: "./flows",
			Templates: "./flows/templates",
			GitCommit: false,
			GitRepo:   "",
		},
		Reporting: ReportingConfig{
			Format:             "markdown",
			OutputDir:          "./reports",
			IncludeScreenshots: true,
			IncludeVideos:      false,
		},
		Browser: BrowserConfig{
			Headless: true,
			Viewport: struct {
				Width  int `yaml:"width"`
				Height int `yaml:"height"`
			}{
				Width:  1920,
				Height: 1080,
			},
			Timeout: 30 * time.Second,
		},
	}
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// If no path provided, try to find wizards-qa.yaml in current directory or home
	if path == "" {
		path = findConfigFile()
		if path == "" {
			// No config file found, use defaults
			return cfg, nil
		}
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand environment variables
	cfg.expandEnvVars()

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to a file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate AI config
	if c.AI.Provider == "" {
		return fmt.Errorf("ai.provider is required")
	}
	if c.AI.Model == "" {
		return fmt.Errorf("ai.model is required")
	}
	if c.AI.Temperature < 0 || c.AI.Temperature > 1 {
		return fmt.Errorf("ai.temperature must be between 0 and 1")
	}

	// Validate Maestro config
	if c.Maestro.Path == "" {
		return fmt.Errorf("maestro.path is required")
	}
	if c.Maestro.Browser == "" {
		return fmt.Errorf("maestro.browser is required")
	}
	validBrowsers := map[string]bool{"chrome": true, "firefox": true, "safari": true}
	if !validBrowsers[c.Maestro.Browser] {
		return fmt.Errorf("maestro.browser must be chrome, firefox, or safari")
	}

	// Validate reporting config
	validFormats := map[string]bool{"markdown": true, "json": true, "junit": true}
	if !validFormats[c.Reporting.Format] {
		return fmt.Errorf("reporting.format must be markdown, json, or junit")
	}

	return nil
}

// expandEnvVars expands environment variables in string fields
func (c *Config) expandEnvVars() {
	c.AI.APIKey = os.ExpandEnv(c.AI.APIKey)
	c.Maestro.Path = os.ExpandEnv(c.Maestro.Path)
	c.Maestro.ScreenshotDir = os.ExpandEnv(c.Maestro.ScreenshotDir)
	c.Flows.Directory = os.ExpandEnv(c.Flows.Directory)
	c.Flows.Templates = os.ExpandEnv(c.Flows.Templates)
	c.Flows.GitRepo = os.ExpandEnv(c.Flows.GitRepo)
	c.Reporting.OutputDir = os.ExpandEnv(c.Reporting.OutputDir)
}

// findConfigFile searches for wizards-qa.yaml in common locations
func findConfigFile() string {
	// Search order:
	// 1. Current directory
	// 2. Parent directories (up to 5 levels)
	// 3. Home directory

	candidates := []string{
		"wizards-qa.yaml",
		"wizards-qa.yml",
		".wizards-qa.yaml",
		".wizards-qa.yml",
	}

	// Try current directory
	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}

	// Try parent directories
	dir, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent

		for _, name := range candidates {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	// Try home directory
	home, err := os.UserHomeDir()
	if err == nil {
		for _, name := range candidates {
			path := filepath.Join(home, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}
