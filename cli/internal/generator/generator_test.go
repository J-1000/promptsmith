package generator

import (
	"context"
	"testing"

	"github.com/promptsmith/cli/internal/benchmark"
)

// MockProvider for testing
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) Complete(ctx context.Context, req benchmark.CompletionRequest) (*benchmark.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &benchmark.CompletionResponse{
		Content:      m.response,
		Model:        req.Model,
		PromptTokens: 100,
		OutputTokens: 200,
		TotalTokens:  300,
		LatencyMs:    500,
		Cost:         0.01,
	}, nil
}

func (m *mockProvider) Models() []string {
	return []string{"mock-model"}
}

func (m *mockProvider) SupportsModel(model string) bool {
	return true
}

func TestGenerator_Generate(t *testing.T) {
	mockResponse := `Here are the variations:

---VARIATION---
Description: More concise version
` + "```" + `
Summarize this text briefly.
` + "```" + `

---VARIATION---
Description: Added context
` + "```" + `
You are an expert summarizer. Please summarize the following text.
` + "```" + `

---VARIATION---
Description: Bullet point format
` + "```" + `
Create a bullet-point summary of this text.
` + "```" + `
`

	provider := &mockProvider{response: mockResponse}
	gen := New(provider)

	result, err := gen.Generate(context.Background(), GenerateRequest{
		Type:   TypeVariations,
		Prompt: "Summarize this text.",
		Count:  3,
		Model:  "mock-model",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Original != "Summarize this text." {
		t.Errorf("expected original prompt, got %s", result.Original)
	}

	if len(result.Variations) != 3 {
		t.Errorf("expected 3 variations, got %d", len(result.Variations))
	}

	// Check first variation
	if result.Variations[0].Description != "More concise version" {
		t.Errorf("unexpected description: %s", result.Variations[0].Description)
	}
	if result.Variations[0].Content != "Summarize this text briefly." {
		t.Errorf("unexpected content: %s", result.Variations[0].Content)
	}
}

func TestGenerator_DefaultCount(t *testing.T) {
	mockResponse := `---VARIATION---
Description: V1
` + "```" + `
Content 1
` + "```" + `

---VARIATION---
Description: V2
` + "```" + `
Content 2
` + "```" + `

---VARIATION---
Description: V3
` + "```" + `
Content 3
` + "```" + `
`

	provider := &mockProvider{response: mockResponse}
	gen := New(provider)

	result, err := gen.Generate(context.Background(), GenerateRequest{
		Type:   TypeVariations,
		Prompt: "Test",
		Count:  0, // Should default to 3
		Model:  "mock-model",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Variations) != 3 {
		t.Errorf("expected 3 variations (default), got %d", len(result.Variations))
	}
}

func TestGenerator_MaxCount(t *testing.T) {
	provider := &mockProvider{response: ""}
	gen := New(provider)

	// Request more than max (10)
	_, err := gen.Generate(context.Background(), GenerateRequest{
		Type:   TypeVariations,
		Prompt: "Test",
		Count:  20, // Should be capped to 10
		Model:  "mock-model",
	})

	// Should not error, just cap the count
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	provider := &mockProvider{}
	gen := New(provider)

	tests := []struct {
		genType  GenerationType
		contains string
	}{
		{TypeVariations, "variations"},
		{TypeCompress, "token optimization"},
		{TypeExpand, "Expand"},
		{TypeRephrase, "Rephrase"},
	}

	for _, tt := range tests {
		t.Run(string(tt.genType), func(t *testing.T) {
			prompt := gen.buildSystemPrompt(GenerateRequest{Type: tt.genType})
			if prompt == "" {
				t.Error("expected non-empty system prompt")
			}
		})
	}
}

func TestBuildUserPrompt(t *testing.T) {
	provider := &mockProvider{}
	gen := New(provider)

	prompt := gen.buildUserPrompt(GenerateRequest{
		Type:   TypeVariations,
		Prompt: "Test prompt content",
		Count:  5,
		Goal:   "improve clarity",
	})

	if prompt == "" {
		t.Error("expected non-empty user prompt")
	}

	// Should contain the original prompt
	if !containsString(prompt, "Test prompt content") {
		t.Error("user prompt should contain original prompt")
	}

	// Should contain the goal
	if !containsString(prompt, "improve clarity") {
		t.Error("user prompt should contain goal")
	}

	// Should specify count
	if !containsString(prompt, "5 variations") {
		t.Error("user prompt should specify count")
	}
}

func TestParseVariations(t *testing.T) {
	provider := &mockProvider{}
	gen := New(provider)

	content := `---VARIATION---
Description: First one
` + "```" + `
First content here
` + "```" + `

---VARIATION---
Description: Second one
` + "```" + `
Second content here
` + "```" + `
`

	variations := gen.parseVariations(content, 5)

	if len(variations) != 2 {
		t.Errorf("expected 2 variations, got %d", len(variations))
	}

	if variations[0].Description != "First one" {
		t.Errorf("expected 'First one', got '%s'", variations[0].Description)
	}
	if variations[0].Content != "First content here" {
		t.Errorf("expected 'First content here', got '%s'", variations[0].Content)
	}
}

func TestParseVariations_LimitsCount(t *testing.T) {
	provider := &mockProvider{}
	gen := New(provider)

	content := `---VARIATION---
Description: V1
` + "```" + `C1` + "```" + `
---VARIATION---
Description: V2
` + "```" + `C2` + "```" + `
---VARIATION---
Description: V3
` + "```" + `C3` + "```" + `
`

	variations := gen.parseVariations(content, 2) // Only want 2

	if len(variations) != 2 {
		t.Errorf("expected 2 variations (limited), got %d", len(variations))
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"hello world", 2},  // 11 chars / 4 = 2
		{"This is a longer text with more words", 9}, // 38 chars / 4 = 9
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got != tt.expected {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.text, got, tt.expected)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
