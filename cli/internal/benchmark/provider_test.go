package benchmark

import (
	"context"
	"testing"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		promptTokens int
		outputTokens int
		wantMin      float64
		wantMax      float64
	}{
		{
			name:         "gpt-4o cost",
			model:        "gpt-4o",
			promptTokens: 1000,
			outputTokens: 500,
			wantMin:      0.007,
			wantMax:      0.008,
		},
		{
			name:         "gpt-4o-mini cost",
			model:        "gpt-4o-mini",
			promptTokens: 1000,
			outputTokens: 500,
			wantMin:      0.0004,
			wantMax:      0.0005,
		},
		{
			name:         "claude-sonnet cost",
			model:        "claude-sonnet",
			promptTokens: 1000,
			outputTokens: 500,
			wantMin:      0.010,
			wantMax:      0.011,
		},
		{
			name:         "unknown model returns 0",
			model:        "unknown-model",
			promptTokens: 1000,
			outputTokens: 500,
			wantMin:      0,
			wantMax:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateCost(tt.model, tt.promptTokens, tt.outputTokens)
			if cost < tt.wantMin || cost > tt.wantMax {
				t.Errorf("CalculateCost() = %v, want between %v and %v", cost, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGetProviderForModel(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		{"gpt-4o", "openai"},
		{"gpt-4o-mini", "openai"},
		{"gpt-4-turbo", "openai"},
		{"o1", "openai"},
		{"o1-mini", "openai"},
		{"claude-sonnet", "anthropic"},
		{"claude-3-5-sonnet-20241022", "anthropic"},
		{"claude-haiku", "anthropic"},
		{"claude-opus", "anthropic"},
		{"gemini-1.5-pro", "google"},
		{"gemini-2.0-flash", "google"},
		{"llama-3.1-70b", "groq"},
		{"mixtral-8x7b", "groq"},
		{"unknown-model", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := GetProviderForModel(tt.model)
			if got != tt.expected {
				t.Errorf("GetProviderForModel(%s) = %s, want %s", tt.model, got, tt.expected)
			}
		})
	}
}

// MockProvider for testing
type MockProvider struct {
	name     string
	models   []string
	response *CompletionResponse
	err      error
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *MockProvider) Models() []string {
	return m.models
}

func (m *MockProvider) SupportsModel(model string) bool {
	for _, supported := range m.models {
		if supported == model {
			return true
		}
	}
	return false
}

func TestProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	mockOpenAI := &MockProvider{
		name:   "openai",
		models: []string{"gpt-4o", "gpt-4o-mini"},
	}

	mockAnthropic := &MockProvider{
		name:   "anthropic",
		models: []string{"claude-sonnet", "claude-opus"},
	}

	registry.Register(mockOpenAI)
	registry.Register(mockAnthropic)

	t.Run("get registered provider", func(t *testing.T) {
		p, ok := registry.Get("openai")
		if !ok {
			t.Error("expected to find openai provider")
		}
		if p.Name() != "openai" {
			t.Errorf("expected openai, got %s", p.Name())
		}
	})

	t.Run("get unregistered provider", func(t *testing.T) {
		_, ok := registry.Get("google")
		if ok {
			t.Error("expected not to find google provider")
		}
	})

	t.Run("get provider for model", func(t *testing.T) {
		p, err := registry.GetForModel("gpt-4o")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if p.Name() != "openai" {
			t.Errorf("expected openai, got %s", p.Name())
		}
	})

	t.Run("get provider for unregistered model provider", func(t *testing.T) {
		_, err := registry.GetForModel("gemini-1.5-pro")
		if err == nil {
			t.Error("expected error for unregistered provider")
		}
	})
}
