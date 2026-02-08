package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is an interface for AI providers
type Client interface {
	Analyze(prompt string, context map[string]interface{}) (*AnalysisResult, error)
	Generate(prompt string, context map[string]interface{}) (string, error)
}

// ClaudeClient implements the Client interface for Anthropic Claude
type ClaudeClient struct {
	BaseClient
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(apiKey, model string, temperature float64, maxTokens int) *ClaudeClient {
	c := &ClaudeClient{}
	c.BaseClient = NewBaseClient(apiKey, model, temperature, maxTokens, c)
	return c
}

// claudeRequest represents the request to Claude API
type claudeRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature,omitempty"`
	System      string    `json:"system,omitempty"`
	Messages    []message `json:"messages"`
}

// message represents a message in the conversation (text-only)
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// contentBlock represents a content block in a multimodal message (text or image)
type contentBlock struct {
	Type   string       `json:"type"`
	Text   string       `json:"text,omitempty"`
	Source *imageSource `json:"source,omitempty"`
}

// imageSource represents a base64-encoded image for the Claude API
type imageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/png"
	Data      string `json:"data"`       // base64 data
}

// multimodalMessage supports both text and image content
type multimodalMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

// claudeMultimodalRequest is like claudeRequest but uses multimodal messages
type claudeMultimodalRequest struct {
	Model       string              `json:"model"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature,omitempty"`
	System      string              `json:"system,omitempty"`
	Messages    []multimodalMessage `json:"messages"`
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

// callAPIOnce makes a single API request to Claude
func (c *ClaudeClient) callAPIOnce(prompt string) (string, error) {
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

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return claudeResp.Content[0].Text, nil
}

// AnalyzeWithImages sends a multimodal request with multiple images and an optional system prompt.
func (c *ClaudeClient) AnalyzeWithImages(systemPrompt string, prompt string, imagesB64 []string) (string, error) {
	var content []contentBlock

	// Add all images first
	for _, imgB64 := range imagesB64 {
		content = append(content, contentBlock{
			Type: "image",
			Source: &imageSource{
				Type:      "base64",
				MediaType: "image/jpeg",
				Data:      imgB64,
			},
		})
	}

	// Add text prompt
	content = append(content, contentBlock{
		Type: "text",
		Text: prompt,
	})

	req := claudeMultimodalRequest{
		Model:       c.Model,
		MaxTokens:   c.MaxTokens,
		Temperature: c.Temperature,
		System:      systemPrompt,
		Messages: []multimodalMessage{
			{
				Role:    "user",
				Content: content,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal multimodal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return claudeResp.Content[0].Text, nil
}

// --- Tool Use types ---

// ToolDefinition defines a tool that Claude can call.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// AgentMessage uses interface{} Content to support text, images, tool_use, and tool_result blocks.
type AgentMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ResponseContentBlock is a polymorphic content block: text or tool_use.
type ResponseContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ToolUseResponse extends response to include StopReason for tool_use detection.
type ToolUseResponse struct {
	Content    []ResponseContentBlock `json:"content"`
	StopReason string                 `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ToolResultBlock is sent in user messages to return tool results.
type ToolResultBlock struct {
	Type      string      `json:"type"`    // "tool_result"
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content"` // string or []contentBlock
	IsError   bool        `json:"is_error,omitempty"`
}

// claudeToolUseRequest is the request body for tool use calls.
type claudeToolUseRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	Temperature float64          `json:"temperature,omitempty"`
	System      string           `json:"system,omitempty"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Messages    []AgentMessage   `json:"messages"`
}

// CallWithTools sends a tool-use request to the Claude API.
func (c *ClaudeClient) CallWithTools(systemPrompt string, messages []AgentMessage, tools []ToolDefinition) (*ToolUseResponse, error) {
	req := claudeToolUseRequest{
		Model:       c.Model,
		MaxTokens:   c.MaxTokens,
		Temperature: c.Temperature,
		System:      systemPrompt,
		Tools:       tools,
		Messages:    messages,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool use request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var toolResp ToolUseResponse
	if err := json.Unmarshal(body, &toolResp); err != nil {
		return nil, fmt.Errorf("failed to parse tool use response: %w", err)
	}

	return &toolResp, nil
}

// AnalyzeWithImage sends a multimodal request with an image and text prompt to the Claude API.
func (c *ClaudeClient) AnalyzeWithImage(prompt string, imageB64 string) (string, error) {
	req := claudeMultimodalRequest{
		Model:       c.Model,
		MaxTokens:   c.MaxTokens,
		Temperature: c.Temperature,
		Messages: []multimodalMessage{
			{
				Role: "user",
				Content: []contentBlock{
					{
						Type: "image",
						Source: &imageSource{
							Type:      "base64",
							MediaType: "image/jpeg",
							Data:      imageB64,
						},
					},
					{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal multimodal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return claudeResp.Content[0].Text, nil
}
