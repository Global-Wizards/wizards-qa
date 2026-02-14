package main

import (
	"fmt"
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

// newAnalyzer creates an AI analyzer from the configuration, respecting the provider setting.
// Optional overrides (model, maxTokens, temperature) take precedence over config values
// when non-zero/non-empty. Use temperature < 0 to indicate "no override".
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
	analyzer, err := ai.NewAnalyzerFromConfig(cfg.AI.Provider, cfg.AI.APIKey, m, t, mt)
	if err != nil {
		return nil, err
	}

	// Set up secondary client for synthesis/flow generation if configured
	if cfg.AI.SynthesisModel != "" {
		synthProvider := cfg.AI.SynthesisProvider
		if synthProvider == "" {
			// Auto-detect provider from model name
			if strings.HasPrefix(cfg.AI.SynthesisModel, "gemini") {
				synthProvider = "google"
			} else if strings.HasPrefix(cfg.AI.SynthesisModel, "claude") {
				synthProvider = "anthropic"
			} else {
				synthProvider = cfg.AI.Provider
			}
		}
		synthAPIKey := cfg.AI.SynthesisAPIKey
		if synthAPIKey == "" {
			synthAPIKey = cfg.AI.APIKey
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
