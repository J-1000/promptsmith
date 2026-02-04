package benchmark

import (
	"os"
	"testing"
)

func TestNewOpenAIProvider_NoAPIKey(t *testing.T) {
	// Save and clear the API key
	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		}
	}()

	_, err := NewOpenAIProvider()
	if err == nil {
		t.Error("expected error when OPENAI_API_KEY is not set")
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	// Temporarily set a fake key for testing
	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	p, err := NewOpenAIProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Name() != "openai" {
		t.Errorf("expected name 'openai', got '%s'", p.Name())
	}
}

func TestOpenAIProvider_Models(t *testing.T) {
	// Temporarily set a fake key for testing
	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	p, err := NewOpenAIProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	models := p.Models()
	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Check for common models
	expectedModels := []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo"}
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

func TestOpenAIProvider_SupportsModel(t *testing.T) {
	// Temporarily set a fake key for testing
	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	p, err := NewOpenAIProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		model    string
		expected bool
	}{
		{"gpt-4o", true},
		{"gpt-4o-mini", true},
		{"gpt-4-turbo", true},
		{"claude-sonnet", false},
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
