package testing

import (
	"context"
	"testing"

	"github.com/promptsmith/cli/internal/benchmark"
)

// mockProvider implements benchmark.Provider for testing
type mockProvider struct {
	name     string
	response *benchmark.CompletionResponse
	err      error
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Models() []string { return []string{"gpt-4o-mini"} }
func (m *mockProvider) SupportsModel(model string) bool { return true }
func (m *mockProvider) Complete(ctx context.Context, req benchmark.CompletionRequest) (*benchmark.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestLLMExecutor_Execute(t *testing.T) {
	registry := benchmark.NewProviderRegistry()
	registry.Register(&mockProvider{
		name: "openai",
		response: &benchmark.CompletionResponse{
			Content:      "Hello, world!",
			Model:        "gpt-4o-mini",
			PromptTokens: 10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
	})

	executor := NewLLMExecutor(registry, WithModel("gpt-4o-mini"))

	output, err := executor.Execute("Test prompt", nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", output)
	}
}

func TestLLMExecutor_Options(t *testing.T) {
	registry := benchmark.NewProviderRegistry()

	executor := NewLLMExecutor(registry,
		WithModel("gpt-4o"),
		WithMaxTokens(2048),
		WithTemperature(0.5),
	)

	if executor.model != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got '%s'", executor.model)
	}
	if executor.maxTokens != 2048 {
		t.Errorf("Expected maxTokens 2048, got %d", executor.maxTokens)
	}
	if executor.temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", executor.temperature)
	}
}

func TestLLMExecutor_Defaults(t *testing.T) {
	registry := benchmark.NewProviderRegistry()
	executor := NewLLMExecutor(registry)

	if executor.model != "gpt-4o-mini" {
		t.Errorf("Expected default model 'gpt-4o-mini', got '%s'", executor.model)
	}
	if executor.maxTokens != 1024 {
		t.Errorf("Expected default maxTokens 1024, got %d", executor.maxTokens)
	}
	if executor.temperature != 0.7 {
		t.Errorf("Expected default temperature 0.7, got %f", executor.temperature)
	}
}
