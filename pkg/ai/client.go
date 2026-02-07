package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/retry"
)

// Client is an interface for AI providers
type Client interface {
	Analyze(prompt string, context map[string]interface{}) (*AnalysisResult, error)
	Generate(prompt string, context map[string]interface{}) (string, error)
}

// ClaudeClient implements the Client interface for Anthropic Claude
type ClaudeClient struct {
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
	HTTPClient  *http.Client
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(apiKey, model string, temperature float64, maxTokens int) *ClaudeClient {
	return &ClaudeClient{
		APIKey:      apiKey,
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// claudeRequest represents the request to Claude API
type claudeRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature,omitempty"`
	Messages    []message `json:"messages"`
}

// message represents a message in the conversation
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse represents the response from Claude API
type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Analyze sends a prompt to Claude and returns structured analysis
func (c *ClaudeClient) Analyze(prompt string, context map[string]interface{}) (*AnalysisResult, error) {
	// Build the full prompt with context
	fullPrompt := c.buildAnalysisPrompt(prompt, context)

	// Call Claude API
	response, err := c.callAPI(fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("claude API call failed: %w", err)
	}

	// Parse response as JSON (structured analysis)
	var result AnalysisResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If JSON parsing fails, return raw response
		return &AnalysisResult{
			RawResponse: response,
		}, nil
	}

	return &result, nil
}

// Generate sends a prompt to Claude and returns raw text response
func (c *ClaudeClient) Generate(prompt string, context map[string]interface{}) (string, error) {
	// Build the full prompt with context
	fullPrompt := c.buildGenerationPrompt(prompt, context)

	// Call Claude API
	response, err := c.callAPI(fullPrompt)
	if err != nil {
		return "", fmt.Errorf("claude API call failed: %w", err)
	}

	return response, nil
}

// callAPI makes the actual HTTP request to Claude with retry logic
func (c *ClaudeClient) callAPI(prompt string) (string, error) {
	var response string
	err := retry.Do(context.Background(), retry.DefaultConfig(), func() error {
		resp, err := c.callAPIOnce(prompt)
		if err != nil {
			return err
		}
		response = resp
		return nil
	})
	
	return response, err
}

// callAPIOnce makes a single API request
func (c *ClaudeClient) callAPIOnce(prompt string) (string, error) {
	// Build request
	req := claudeRequest{
		Model:       c.Model,
		MaxTokens:   c.MaxTokens,
		Temperature: c.Temperature,
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Marshal to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from content
	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return claudeResp.Content[0].Text, nil
}

// buildAnalysisPrompt constructs the prompt for game analysis
func (c *ClaudeClient) buildAnalysisPrompt(prompt string, context map[string]interface{}) string {
	var fullPrompt string

	fullPrompt += "You are an expert QA engineer specializing in game testing.\n"
	fullPrompt += "Analyze the provided game and respond with structured JSON.\n\n"

	// Add context if provided
	if spec, ok := context["spec"].(string); ok {
		fullPrompt += "Game Specification:\n"
		fullPrompt += spec + "\n\n"
	}

	if url, ok := context["url"].(string); ok {
		fullPrompt += "Game URL: " + url + "\n\n"
	}

	fullPrompt += prompt

	return fullPrompt
}

// buildGenerationPrompt constructs the prompt for flow generation
func (c *ClaudeClient) buildGenerationPrompt(prompt string, context map[string]interface{}) string {
	var fullPrompt string

	fullPrompt += "You are an expert at creating Maestro test flows for game testing.\n"
	fullPrompt += "Generate valid Maestro YAML flows based on the provided requirements.\n\n"

	// Add context if provided
	if analysis, ok := context["analysis"].(string); ok {
		fullPrompt += "Game Analysis:\n"
		fullPrompt += analysis + "\n\n"
	}

	fullPrompt += prompt

	return fullPrompt
}
