package prompt

import (
	"encoding/json"
	"testing"
)

func TestParseWithFrontmatter(t *testing.T) {
	content := `---
name: article-summarizer
description: Summarizes news articles
model_hint: gpt-4o-mini

variables:
  - name: article
    type: string
    required: true
  - name: max_points
    type: number
    default: 5
---

Summarize the following article into {{max_points}} bullet points.

Article:
{{article}}
`

	parsed, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parsed.HasFrontmatter {
		t.Error("expected HasFrontmatter to be true")
	}

	if parsed.Frontmatter == nil {
		t.Fatal("expected Frontmatter to be non-nil")
	}

	if parsed.Frontmatter.Name != "article-summarizer" {
		t.Errorf("expected name 'article-summarizer', got '%s'", parsed.Frontmatter.Name)
	}

	if parsed.Frontmatter.Description != "Summarizes news articles" {
		t.Errorf("expected description 'Summarizes news articles', got '%s'", parsed.Frontmatter.Description)
	}

	if parsed.Frontmatter.ModelHint != "gpt-4o-mini" {
		t.Errorf("expected model_hint 'gpt-4o-mini', got '%s'", parsed.Frontmatter.ModelHint)
	}

	if len(parsed.Frontmatter.Variables) != 2 {
		t.Errorf("expected 2 variables, got %d", len(parsed.Frontmatter.Variables))
	}

	// Check first variable
	if parsed.Frontmatter.Variables[0].Name != "article" {
		t.Errorf("expected first variable name 'article', got '%s'", parsed.Frontmatter.Variables[0].Name)
	}
	if parsed.Frontmatter.Variables[0].Type != "string" {
		t.Errorf("expected first variable type 'string', got '%s'", parsed.Frontmatter.Variables[0].Type)
	}
	if !parsed.Frontmatter.Variables[0].Required {
		t.Error("expected first variable to be required")
	}

	// Check extracted vars
	if len(parsed.ExtractedVars) != 2 {
		t.Errorf("expected 2 extracted vars, got %d", len(parsed.ExtractedVars))
	}
}

func TestParseWithoutFrontmatter(t *testing.T) {
	content := `You are a helpful assistant.

Please summarize {{text}} in {{language}}.
`

	parsed, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.HasFrontmatter {
		t.Error("expected HasFrontmatter to be false")
	}

	if parsed.Frontmatter != nil {
		t.Error("expected Frontmatter to be nil")
	}

	if len(parsed.ExtractedVars) != 2 {
		t.Errorf("expected 2 extracted vars, got %d", len(parsed.ExtractedVars))
	}

	// Check extracted variables
	expectedVars := map[string]bool{"text": true, "language": true}
	for _, v := range parsed.ExtractedVars {
		if !expectedVars[v] {
			t.Errorf("unexpected variable: %s", v)
		}
	}
}

func TestParseEmptyContent(t *testing.T) {
	parsed, err := Parse("")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.HasFrontmatter {
		t.Error("expected HasFrontmatter to be false for empty content")
	}

	if len(parsed.ExtractedVars) != 0 {
		t.Errorf("expected 0 extracted vars, got %d", len(parsed.ExtractedVars))
	}
}

func TestExtractMustacheVars(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "simple variables",
			content:  "Hello {{name}}, welcome to {{place}}!",
			expected: []string{"name", "place"},
		},
		{
			name:     "duplicate variables",
			content:  "{{name}} says hello to {{name}}",
			expected: []string{"name"},
		},
		{
			name:     "variables with whitespace",
			content:  "{{ name }} and {{  place  }}",
			expected: []string{"name", "place"},
		},
		{
			name:     "no variables",
			content:  "Just plain text",
			expected: nil,
		},
		{
			name:     "section tags excluded",
			content:  "{{#items}}{{name}}{{/items}}",
			expected: []string{"name"},
		},
		{
			name:     "nested object access",
			content:  "{{user.name}} lives in {{user.address.city}}",
			expected: []string{"user.name", "user.address.city"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := Parse(tt.content)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(parsed.ExtractedVars) != len(tt.expected) {
				t.Errorf("expected %d vars, got %d: %v", len(tt.expected), len(parsed.ExtractedVars), parsed.ExtractedVars)
				return
			}

			expectedMap := make(map[string]bool)
			for _, v := range tt.expected {
				expectedMap[v] = true
			}

			for _, v := range parsed.ExtractedVars {
				if !expectedMap[v] {
					t.Errorf("unexpected variable: %s", v)
				}
			}
		})
	}
}

func TestVariablesJSON(t *testing.T) {
	// Test with frontmatter variables
	contentWithFM := `---
name: test
variables:
  - name: foo
    type: string
    required: true
---
{{foo}} {{bar}}
`

	parsed, _ := Parse(contentWithFM)
	varsJSON := parsed.VariablesJSON()

	var vars []Variable
	if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
		t.Fatalf("failed to unmarshal variables JSON: %v", err)
	}

	// Should use frontmatter variables, not extracted
	if len(vars) != 1 {
		t.Errorf("expected 1 variable from frontmatter, got %d", len(vars))
	}

	// Test without frontmatter - should use extracted vars
	contentNoFM := "{{foo}} {{bar}}"
	parsed, _ = Parse(contentNoFM)
	varsJSON = parsed.VariablesJSON()

	if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
		t.Fatalf("failed to unmarshal variables JSON: %v", err)
	}

	if len(vars) != 2 {
		t.Errorf("expected 2 extracted variables, got %d", len(vars))
	}

	// All should be string type and required
	for _, v := range vars {
		if v.Type != "string" {
			t.Errorf("expected type 'string', got '%s'", v.Type)
		}
		if !v.Required {
			t.Error("expected extracted variables to be required")
		}
	}
}

func TestMetadataJSON(t *testing.T) {
	// With model hint
	contentWithHint := `---
name: test
model_hint: gpt-4o
---
content
`

	parsed, _ := Parse(contentWithHint)
	metaJSON := parsed.MetadataJSON()

	var meta map[string]any
	if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
		t.Fatalf("failed to unmarshal metadata JSON: %v", err)
	}

	if meta["model_hint"] != "gpt-4o" {
		t.Errorf("expected model_hint 'gpt-4o', got '%v'", meta["model_hint"])
	}

	// Without model hint
	contentNoHint := `---
name: test
---
content
`

	parsed, _ = Parse(contentNoHint)
	metaJSON = parsed.MetadataJSON()

	if metaJSON != "{}" {
		t.Errorf("expected empty metadata '{}', got '%s'", metaJSON)
	}
}

func TestNameAndDescription(t *testing.T) {
	content := `---
name: my-prompt
description: A test prompt
---
content
`

	parsed, _ := Parse(content)

	if parsed.Name() != "my-prompt" {
		t.Errorf("expected name 'my-prompt', got '%s'", parsed.Name())
	}

	if parsed.Description() != "A test prompt" {
		t.Errorf("expected description 'A test prompt', got '%s'", parsed.Description())
	}

	// Without frontmatter
	parsed, _ = Parse("no frontmatter")

	if parsed.Name() != "" {
		t.Errorf("expected empty name, got '%s'", parsed.Name())
	}

	if parsed.Description() != "" {
		t.Errorf("expected empty description, got '%s'", parsed.Description())
	}
}

func TestParseInvalidFrontmatter(t *testing.T) {
	content := `---
name: [invalid yaml
---
content
`

	_, err := Parse(content)
	if err == nil {
		t.Error("expected error for invalid YAML frontmatter")
	}
}

func TestParseEnumVariable(t *testing.T) {
	content := `---
name: test
variables:
  - name: tone
    type: enum
    values: [formal, casual, technical]
    default: formal
---
Write in a {{tone}} tone.
`

	parsed, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Frontmatter.Variables) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(parsed.Frontmatter.Variables))
	}

	v := parsed.Frontmatter.Variables[0]
	if v.Type != "enum" {
		t.Errorf("expected type 'enum', got '%s'", v.Type)
	}

	if len(v.Values) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(v.Values))
	}

	if v.Default != "formal" {
		t.Errorf("expected default 'formal', got '%v'", v.Default)
	}
}
