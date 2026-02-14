package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/ai"
	"github.com/Global-Wizards/wizards-qa/pkg/config"
	"github.com/Global-Wizards/wizards-qa/pkg/maestro"
	"github.com/Global-Wizards/wizards-qa/pkg/util"
)

// loadConfig loads the configuration from the given path, returning a helpful error.
func loadConfig(configPath string) (*config.Config, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

// validateAPIKey checks that the AI API key is configured.
func validateAPIKey(cfg *config.Config) error {
	if cfg.AI.APIKey == "" || cfg.AI.APIKey == "${ANTHROPIC_API_KEY}" {
		return fmt.Errorf("AI API key not configured. Set ANTHROPIC_API_KEY environment variable or update wizards-qa.yaml")
	}
	return nil
}

// detectProviderAndKey auto-detects the AI provider and API key from a model name.
// When the --model flag overrides the config model, the provider may need to change too
// (e.g. config says "google" but --model is "claude-sonnet-4-5-20250929").
func detectProviderAndKey(model, cfgProvider, cfgAPIKey string) (provider, apiKey string) {
	provider = cfgProvider
	apiKey = cfgAPIKey

	switch {
	case strings.HasPrefix(model, "claude"):
		provider = "anthropic"
		// Use ANTHROPIC_API_KEY from env if the config key is for a different provider
		if envKey := os.Getenv("ANTHROPIC_API_KEY"); envKey != "" {
			apiKey = envKey
		}
	case strings.HasPrefix(model, "gemini"):
		provider = "google"
		if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
			apiKey = envKey
		}
	case strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3"):
		provider = "openai"
		if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
			apiKey = envKey
		}
	}
	return provider, apiKey
}

// newAnalyzer creates an AI analyzer from the configuration, respecting the provider setting.
// Optional overrides (model, maxTokens, temperature) take precedence over config values
// when non-zero/non-empty. Use temperature < 0 to indicate "no override".
// When --model overrides the model, provider and API key are auto-detected from the model name.
func newAnalyzer(cfg *config.Config, model string, maxTokens int, temperature float64) (*ai.Analyzer, error) {
	m := cfg.AI.Model
	if model != "" {
		m = model
	}
	mt := cfg.AI.MaxTokens
	if maxTokens > 0 {
		mt = maxTokens
	}
	t := cfg.AI.Temperature
	if temperature >= 0 {
		t = temperature
	}

	// Auto-detect provider + API key from model name (handles --model overrides)
	provider, apiKey := detectProviderAndKey(m, cfg.AI.Provider, cfg.AI.APIKey)

	analyzer, err := ai.NewAnalyzerFromConfig(provider, apiKey, m, t, mt)
	if err != nil {
		return nil, err
	}

	// Set up secondary client for synthesis/flow generation if configured
	if cfg.AI.SynthesisModel != "" {
		synthProvider, synthAPIKey := detectProviderAndKey(cfg.AI.SynthesisModel, cfg.AI.Provider, cfg.AI.APIKey)
		if cfg.AI.SynthesisProvider != "" {
			synthProvider = cfg.AI.SynthesisProvider
		}
		if cfg.AI.SynthesisAPIKey != "" {
			synthAPIKey = cfg.AI.SynthesisAPIKey
		}
		secondaryClient, synthErr := ai.NewClientFromConfig(synthProvider, synthAPIKey, cfg.AI.SynthesisModel, t, mt)
		if synthErr != nil {
			return nil, fmt.Errorf("failed to create synthesis client: %w", synthErr)
		}
		analyzer.SetSecondaryClient(secondaryClient)
	}

	return analyzer, nil
}

// deriveGameName extracts a game name from a URL, falling back to "game".
func deriveGameName(gameURL string) string {
	gameName := filepath.Base(filepath.Dir(gameURL))
	if gameName == "" || gameName == "." {
		gameName = "game"
	}
	return gameName
}

// printResults prints a summary of test results to stdout.
func printResults(results *maestro.TestResults) {
	for i, result := range results.Flows {
		status := util.EmojiPassed
		if result.Status != maestro.StatusPassed {
			status = util.EmojiFailed
		}
		fmt.Printf("   %s %d. %s (%s)\n", status, i+1, result.FlowName, result.Duration.Round(time.Millisecond))
		if result.Error != "" {
			fmt.Printf("      Error: %s\n", result.Error)
		}
	}
	fmt.Printf("\n   Success Rate: %.1f%% (%d/%d passed)\n", results.SuccessRate(), results.Passed, results.Total)
}
