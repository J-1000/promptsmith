package benchmark

import (
	"os"
	"testing"
)

func TestNewAnthropicProvider_NoAPIKey(t *testing.T) {
	// Save and clear the API key
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		}
	}()

	_, err := NewAnthropicProvider()
	if err == nil {
		t.Error("expected error when ANTHROPIC_API_KEY is not set")
	}
}

func TestAnthropicProvider_Name(t *testing.T) {
	// Temporarily set a fake key for testing
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
	}()

	p, err := NewAnthropicProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Name() != "anthropic" {
		t.Errorf("expected name 'anthropic', got '%s'", p.Name())
	}
}

func TestAnthropicProvider_Models(t *testing.T) {
	// Temporarily set a fake key for testing
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
	}()

	p, err := NewAnthropicProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	models := p.Models()
	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Check for common models
	expectedModels := []string{"claude-sonnet", "claude-haiku", "claude-opus"}
	for _, expected := range expectedModels {
		found := false
		for _, m := range models {
			if m == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected model %s not found", expected)
		}
	}
}

func TestAnthropicProvider_SupportsModel(t *testing.T) {
	// Temporarily set a fake key for testing
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
	}()

	p, err := NewAnthropicProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		model    string
		expected bool
	}{
		{"claude-sonnet", true},
		{"claude-haiku", true},
		{"claude-opus", true},
		{"claude-3-5-sonnet-20241022", true},
		{"gpt-4o", false},
		{"unknown-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := p.SupportsModel(tt.model)
			if got != tt.expected {
				t.Errorf("SupportsModel(%s) = %v, want %v", tt.model, got, tt.expected)
			}
		})
	}
}

func TestAnthropicProvider_MapModelName(t *testing.T) {
	// Temporarily set a fake key for testing
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
	}()

	p, err := NewAnthropicProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"claude-sonnet", "claude-sonnet-4-20250514"},
		{"claude-haiku", "claude-3-5-haiku-20241022"},
		{"claude-opus", "claude-3-opus-20240229"},
		{"claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20241022"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := p.mapModelName(tt.input)
			if got != tt.expected {
				t.Errorf("mapModelName(%s) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
