package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// AnthropicProvider implements the Provider interface for Anthropic
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// Anthropic API types
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Model        string `json:"model"`
	Content      []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider() (*AnthropicProvider, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1",
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Models returns supported models
func (p *AnthropicProvider) Models() []string {
	return []string{
		"claude-sonnet-4-20250514",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-sonnet",
		"claude-haiku",
		"claude-opus",
	}
}

// SupportsModel checks if model is supported
func (p *AnthropicProvider) SupportsModel(model string) bool {
	for _, m := range p.Models() {
		if m == model {
			return true
		}
	}
	return false
}

// Complete sends a completion request to Anthropic
func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	startTime := time.Now()

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	// Map shorthand model names to full names
	model := p.mapModelName(req.Model)

	anthropicReq := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []anthropicMessage{
			{Role: "user", Content: req.Prompt},
		},
		Temperature: temperature,
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if anthropicResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", anthropicResp.Error.Message)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	latencyMs := time.Since(startTime).Milliseconds()
	cost := CalculateCost(req.Model, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens)

	return &CompletionResponse{
		Content:      anthropicResp.Content[0].Text,
		Model:        anthropicResp.Model,
		PromptTokens: anthropicResp.Usage.InputTokens,
		OutputTokens: anthropicResp.Usage.OutputTokens,
		TotalTokens:  anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		LatencyMs:    latencyMs,
		Cost:         cost,
	}, nil
}

// mapModelName converts shorthand names to full model names
func (p *AnthropicProvider) mapModelName(model string) string {
	switch model {
	case "claude-sonnet":
		return "claude-sonnet-4-20250514"
	case "claude-haiku":
		return "claude-3-5-haiku-20241022"
	case "claude-opus":
		return "claude-3-opus-20240229"
	default:
		return model
	}
}
