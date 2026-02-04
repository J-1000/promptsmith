package benchmark

import (
	"context"
	"fmt"
	"strings"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider name (e.g., "openai", "anthropic")
	Name() string
	// Complete sends a completion request and returns the response
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	// Models returns the list of supported models
	Models() []string
	// SupportsModel returns true if the provider supports the given model
	SupportsModel(model string) bool
}

// CompletionRequest represents a request to an LLM
type CompletionRequest struct {
	Model       string
	Prompt      string
	MaxTokens   int
	Temperature float64
	Variables   map[string]any
}

// CompletionResponse represents a response from an LLM
type CompletionResponse struct {
	Content      string
	Model        string
	PromptTokens int
	OutputTokens int
	TotalTokens  int
	LatencyMs    int64
	Cost         float64
}

// ModelPricing defines token pricing for a model
type ModelPricing struct {
	InputPer1M  float64 // Cost per 1M input tokens
	OutputPer1M float64 // Cost per 1M output tokens
}

// Known model pricing (approximate, as of Feb 2025)
var modelPricing = map[string]ModelPricing{
	// OpenAI
	"gpt-4o":          {InputPer1M: 2.50, OutputPer1M: 10.00},
	"gpt-4o-mini":     {InputPer1M: 0.15, OutputPer1M: 0.60},
	"gpt-4-turbo":     {InputPer1M: 10.00, OutputPer1M: 30.00},
	"o1":              {InputPer1M: 15.00, OutputPer1M: 60.00},
	"o1-mini":         {InputPer1M: 3.00, OutputPer1M: 12.00},
	// Anthropic
	"claude-3-5-sonnet-20241022": {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-sonnet-4-20250514":   {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-3-5-haiku-20241022":  {InputPer1M: 0.80, OutputPer1M: 4.00},
	"claude-3-opus-20240229":     {InputPer1M: 15.00, OutputPer1M: 75.00},
	// Shortcuts
	"claude-sonnet": {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-haiku":  {InputPer1M: 0.80, OutputPer1M: 4.00},
	"claude-opus":   {InputPer1M: 15.00, OutputPer1M: 75.00},
	// Google
	"gemini-1.5-pro":   {InputPer1M: 1.25, OutputPer1M: 5.00},
	"gemini-1.5-flash": {InputPer1M: 0.075, OutputPer1M: 0.30},
	"gemini-2.0-flash": {InputPer1M: 0.10, OutputPer1M: 0.40},
}

// CalculateCost calculates the cost for a completion
func CalculateCost(model string, promptTokens, outputTokens int) float64 {
	pricing, ok := modelPricing[model]
	if !ok {
		// Try prefix matching for versioned models
		for m, p := range modelPricing {
			if strings.HasPrefix(model, m) {
				pricing = p
				ok = true
				break
			}
		}
	}
	if !ok {
		return 0 // Unknown model
	}

	inputCost := float64(promptTokens) * pricing.InputPer1M / 1_000_000
	outputCost := float64(outputTokens) * pricing.OutputPer1M / 1_000_000
	return inputCost + outputCost
}

// GetProviderForModel returns the provider name for a given model
func GetProviderForModel(model string) string {
	model = strings.ToLower(model)
	switch {
	case strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1"):
		return "openai"
	case strings.HasPrefix(model, "claude"):
		return "anthropic"
	case strings.HasPrefix(model, "gemini"):
		return "google"
	case strings.HasPrefix(model, "llama") || strings.HasPrefix(model, "mixtral"):
		return "groq"
	default:
		return "unknown"
	}
}

// ProviderRegistry holds registered providers
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Get returns a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// GetForModel returns the provider that supports the given model
func (r *ProviderRegistry) GetForModel(model string) (Provider, error) {
	providerName := GetProviderForModel(model)
	p, ok := r.Get(providerName)
	if !ok {
		return nil, fmt.Errorf("no provider registered for model %s (provider: %s)", model, providerName)
	}
	return p, nil
}
