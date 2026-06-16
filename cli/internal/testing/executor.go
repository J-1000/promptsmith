package testing

import (
	"context"
	"time"

	"github.com/promptsmith/cli/internal/benchmark"
)

// defaultExecuteTimeout bounds a single LLM call so a hung provider cannot
// block a test run indefinitely.
const defaultExecuteTimeout = 60 * time.Second

// LLMExecutor executes prompts using real LLM providers
type LLMExecutor struct {
	registry    *benchmark.ProviderRegistry
	model       string
	maxTokens   int
	temperature float64
	timeout     time.Duration
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

// WithTimeout sets the per-call timeout for completions. A non-positive value
// disables the deadline.
func WithTimeout(timeout time.Duration) LLMExecutorOption {
	return func(e *LLMExecutor) {
		e.timeout = timeout
	}
}

// NewLLMExecutor creates a new LLM executor
func NewLLMExecutor(registry *benchmark.ProviderRegistry, opts ...LLMExecutorOption) *LLMExecutor {
	e := &LLMExecutor{
		registry:    registry,
		model:       "gpt-4o-mini",
		maxTokens:   1024,
		temperature: 0.7,
		timeout:     defaultExecuteTimeout,
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

	ctx := context.Background()
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}
