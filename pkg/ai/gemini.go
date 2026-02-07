package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GeminiClient implements the Client interface for Google Gemini
type GeminiClient struct {
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
	HTTPClient  *http.Client
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(apiKey, model string, temperature float64, maxTokens int) *GeminiClient {
	if model == "" {
		model = "gemini-pro"
	}
	
	return &GeminiClient{
		APIKey:      apiKey,
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// geminiRequest represents the request to Gemini API
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature  float64 `json:"temperature,omitempty"`
	MaxOutputTokens int  `json:"maxOutputTokens,omitempty"`
}

// geminiResponse represents the response from Gemini API
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// Analyze sends a prompt to Gemini and returns structured analysis
func (g *GeminiClient) Analyze(prompt string, context map[string]interface{}) (*AnalysisResult, error) {
	// Build the full prompt with context
	fullPrompt := g.buildAnalysisPrompt(prompt, context)

	// Call Gemini API
	response, err := g.callAPI(fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("gemini API call failed: %w", err)
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

// Generate sends a prompt to Gemini and returns raw text response
func (g *GeminiClient) Generate(prompt string, context map[string]interface{}) (string, error) {
	// Build the full prompt with context
	fullPrompt := g.buildGenerationPrompt(prompt, context)

	// Call Gemini API
	response, err := g.callAPI(fullPrompt)
	if err != nil {
		return "", fmt.Errorf("gemini API call failed: %w", err)
	}

	return response, nil
}

// callAPI makes the actual HTTP request to Gemini
func (g *GeminiClient) callAPI(prompt string) (string, error) {
	// Build API URL
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		g.Model, g.APIKey)

	// Build request
	req := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     g.Temperature,
			MaxOutputTokens: g.MaxTokens,
		},
	}

	// Marshal to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := g.HTTPClient.Do(httpReq)
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
	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from response
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// buildAnalysisPrompt constructs the prompt for game analysis
func (g *GeminiClient) buildAnalysisPrompt(prompt string, context map[string]interface{}) string {
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
func (g *GeminiClient) buildGenerationPrompt(prompt string, context map[string]interface{}) string {
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
