package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/retry"
)

// APICallerOnce is the interface that provider-specific clients must implement.
type APICallerOnce interface {
	callAPIOnce(prompt string) (string, error)
}

// ImageAnalyzer is an optional interface for AI clients that support multimodal (image+text) analysis.
type ImageAnalyzer interface {
	AnalyzeWithImage(prompt string, imageB64 string) (string, error)
	// AnalyzeWithImages sends a multimodal request with multiple images and an optional system prompt.
	AnalyzeWithImages(systemPrompt string, prompt string, imagesB64 []string) (string, error)
}

// ToolUseAgent is an optional interface for AI clients that support tool use (agentic) interactions.
type ToolUseAgent interface {
	CallWithTools(systemPrompt string, messages []AgentMessage, tools []ToolDefinition) (*ToolUseResponse, error)
}

// BaseClient contains shared fields and methods for AI provider clients.
type BaseClient struct {
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
	HTTPClient  *http.Client
	caller      APICallerOnce
	OnUsage     func(input, output, cacheCreate, cacheRead int) // optional callback for token usage tracking
}

// NewBaseClient creates a BaseClient with standard HTTP settings.
func NewBaseClient(apiKey, model string, temperature float64, maxTokens int, caller APICallerOnce) BaseClient {
	return BaseClient{
		APIKey:      apiKey,
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		HTTPClient: &http.Client{
			Timeout: 180 * time.Second, // 3 min per API call; agent loop manages total budget
		},
		caller: caller,
	}
}

// Analyze sends a prompt and returns structured analysis.
func (b *BaseClient) Analyze(prompt string, ctx map[string]interface{}) (*AnalysisResult, error) {
	fullPrompt := buildLegacyAnalysisPrompt(prompt, ctx)

	response, err := b.callAPI(fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return &AnalysisResult{
			RawResponse: response,
		}, nil
	}

	return &result, nil
}

// Generate sends a prompt and returns raw text response.
func (b *BaseClient) Generate(prompt string, ctx map[string]interface{}) (string, error) {
	fullPrompt := BuildGenerationPrompt(prompt, ctx)

	response, err := b.callAPI(fullPrompt)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}

	return response, nil
}

// callAPI makes an HTTP request with retry logic, delegating to the provider's callAPIOnce.
func (b *BaseClient) callAPI(prompt string) (string, error) {
	var response string
	err := retry.DoWithRetryable(context.Background(), retry.DefaultConfig(), IsRetryableAPIError, func() error {
		resp, err := b.caller.callAPIOnce(prompt)
		if err != nil {
			return err
		}
		response = resp
		return nil
	})

	return response, err
}
