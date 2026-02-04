package testing

import (
	"context"

	"github.com/promptsmith/cli/internal/benchmark"
)

// LLMExecutor executes prompts using real LLM providers
type LLMExecutor struct {
	registry    *benchmark.ProviderRegistry
	model       string
	maxTokens   int
	temperature float64
}

// LLMExecutorOption configures the LLM executor
type LLMExecutorOption func(*LLMExecutor)

// WithModel sets the model to use
func WithModel(model string) LLMExecutorOption {
	return func(e *LLMExecutor) {
		e.model = model
	}
}

// WithMaxTokens sets the max tokens for completions
func WithMaxTokens(maxTokens int) LLMExecutorOption {
	return func(e *LLMExecutor) {
		e.maxTokens = maxTokens
	}
}

// WithTemperature sets the temperature for completions
func WithTemperature(temp float64) LLMExecutorOption {
	return func(e *LLMExecutor) {
		e.temperature = temp
	}
}

// NewLLMExecutor creates a new LLM executor
func NewLLMExecutor(registry *benchmark.ProviderRegistry, opts ...LLMExecutorOption) *LLMExecutor {
	e := &LLMExecutor{
		registry:    registry,
		model:       "gpt-4o-mini",
		maxTokens:   1024,
		temperature: 0.7,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Execute sends the prompt to an LLM and returns the response
func (e *LLMExecutor) Execute(renderedPrompt string, inputs map[string]any) (string, error) {
	provider, err := e.registry.GetForModel(e.model)
	if err != nil {
		return "", err
	}

	req := benchmark.CompletionRequest{
		Model:       e.model,
		Prompt:      renderedPrompt,
		MaxTokens:   e.maxTokens,
		Temperature: e.temperature,
		Variables:   inputs,
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}
