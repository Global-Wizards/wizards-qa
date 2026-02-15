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
	callAPIOnce(ctx context.Context, prompt string) (string, error)
}

// ImageAnalyzer is an optional interface for AI clients that support multimodal (image+text) analysis.
type ImageAnalyzer interface {
	AnalyzeWithImage(ctx context.Context, prompt string, imageB64 string) (string, error)
	// AnalyzeWithImages sends a multimodal request with multiple images and an optional system prompt.
	AnalyzeWithImages(ctx context.Context, systemPrompt string, prompt string, imagesB64 []string) (string, error)
}

// ToolUseAgent is an optional interface for AI clients that support tool use (agentic) interactions.
type ToolUseAgent interface {
	CallWithTools(ctx context.Context, systemPrompt string, messages []AgentMessage, tools []ToolDefinition) (*ToolUseResponse, error)
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
func (b *BaseClient) Analyze(ctx context.Context, prompt string, ctxMap map[string]interface{}) (*AnalysisResult, error) {
	fullPrompt := buildLegacyAnalysisPrompt(prompt, ctxMap)

	response, err := b.callAPI(ctx, fullPrompt)
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
func (b *BaseClient) Generate(ctx context.Context, prompt string, ctxMap map[string]interface{}) (string, error) {
	fullPrompt := BuildGenerationPrompt(prompt, ctxMap)

	response, err := b.callAPI(ctx, fullPrompt)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}

	return response, nil
}

// callAPI makes an HTTP request with retry logic, delegating to the provider's callAPIOnce.
func (b *BaseClient) callAPI(ctx context.Context, prompt string) (string, error) {
	var response string
	err := retry.DoWithRetryable(ctx, retry.DefaultConfig(), IsRetryableAPIError, func() error {
		resp, err := b.caller.callAPIOnce(ctx, prompt)
		if err != nil {
			return err
		}
		response = resp
		return nil
	})

	return response, err
}
