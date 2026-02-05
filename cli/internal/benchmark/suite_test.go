package benchmark

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSuite(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(*Suite) bool
	}{
		{
			name: "valid minimal suite",
			yaml: `
name: test-benchmark
prompt: summarizer
models:
  - gpt-4o
  - claude-sonnet
`,
			wantErr: false,
			check: func(s *Suite) bool {
				return s.Name == "test-benchmark" &&
					s.Prompt == "summarizer" &&
					len(s.Models) == 2 &&
					s.RunsPerModel == 3 // default
			},
		},
		{
			name: "valid full suite",
			yaml: `
name: full-benchmark
prompt: summarizer
description: Benchmark summarizer across models
version: "1.0.0"
models:
  - gpt-4o
  - gpt-4o-mini
  - claude-sonnet
runs_per_model: 10
dataset: fixtures/articles.jsonl
metrics:
  - name: latency
    type: latency_p50
  - name: cost
    type: cost_per_request
variables:
  max_tokens: 500
`,
			wantErr: false,
			check: func(s *Suite) bool {
				return s.Name == "full-benchmark" &&
					s.Description == "Benchmark summarizer across models" &&
					s.Version == "1.0.0" &&
					len(s.Models) == 3 &&
					s.RunsPerModel == 10 &&
					s.Dataset == "fixtures/articles.jsonl" &&
					len(s.Metrics) == 2
			},
		},
		{
			name:    "missing name",
			yaml:    "prompt: summarizer\nmodels:\n  - gpt-4o",
			wantErr: true,
		},
		{
			name:    "missing prompt",
			yaml:    "name: test\nmodels:\n  - gpt-4o",
			wantErr: true,
		},
		{
			name:    "missing models",
			yaml:    "name: test\nprompt: summarizer",
			wantErr: true,
		},
		{
			name:    "empty model in list",
			yaml:    "name: test\nprompt: summarizer\nmodels:\n  - gpt-4o\n  - \"\"",
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			yaml:    "name: test\n  invalid: indentation",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite, err := ParseSuite([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSuite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(suite) {
				t.Errorf("ParseSuite() check failed for %+v", suite)
			}
		})
	}
}

func TestModelResult(t *testing.T) {
	// Verify ModelResult struct works correctly
	result := ModelResult{
		Model:          "gpt-4o",
		Runs:           10,
		LatencyP50Ms:   150.5,
		LatencyP99Ms:   350.2,
		LatencyAvgMs:   175.3,
		TotalTokensAvg: 847.5,
		PromptTokens:   200,
		OutputTokensAvg: 647.5,
		CostPerRequest: 0.0042,
		TotalCost:      0.042,
		Errors:         1,
		ErrorRate:      0.1,
	}

	if result.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", result.Model)
	}
	if result.Runs != 10 {
		t.Errorf("expected 10 runs, got %d", result.Runs)
	}
}

func TestParseSuiteFile(t *testing.T) {
	// Create a temp file
	tmpDir, err := os.MkdirTemp("", "suite-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Valid suite file
	validContent := `name: test-suite
prompt: test-prompt
models:
  - gpt-4o
`
	validPath := filepath.Join(tmpDir, "valid.bench.yaml")
	if err := os.WriteFile(validPath, []byte(validContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	suite, err := ParseSuiteFile(validPath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if suite.Name != "test-suite" {
		t.Errorf("expected name 'test-suite', got '%s'", suite.Name)
	}

	// Non-existent file
	_, err = ParseSuiteFile(filepath.Join(tmpDir, "nonexistent.yaml"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}

	// Invalid YAML file
	invalidPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte("name: test\n  invalid: yaml"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	_, err = ParseSuiteFile(invalidPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseSuiteRunsPerModelDefault(t *testing.T) {
	yaml := `name: test
prompt: test
models:
  - gpt-4o
`
	suite, err := ParseSuite([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suite.RunsPerModel != 3 {
		t.Errorf("expected default runs_per_model 3, got %d", suite.RunsPerModel)
	}
}

func TestParseSuiteWithVariables(t *testing.T) {
	yaml := `name: test
prompt: test
models:
  - gpt-4o
variables:
  max_tokens: 500
  temperature: 0.7
`
	suite, err := ParseSuite([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suite.Variables == nil {
		t.Fatal("expected variables to be set")
	}

	if suite.Variables["max_tokens"] != 500 {
		t.Errorf("expected max_tokens 500, got %v", suite.Variables["max_tokens"])
	}
}

func TestBenchmarkResultStruct(t *testing.T) {
	result := BenchmarkResult{
		SuiteName:   "test-suite",
		PromptName:  "test-prompt",
		Version:     "1.0.0",
		Models:      []ModelResult{{Model: "gpt-4o"}},
		Runs:        []RunResult{{Model: "gpt-4o", LatencyMs: 100}},
		DurationMs:  5000,
		StartedAt:   "2025-01-01T00:00:00Z",
		CompletedAt: "2025-01-01T00:00:05Z",
	}

	if result.SuiteName != "test-suite" {
		t.Errorf("expected suite name 'test-suite', got '%s'", result.SuiteName)
	}
	if len(result.Models) != 1 {
		t.Errorf("expected 1 model result, got %d", len(result.Models))
	}
	if len(result.Runs) != 1 {
		t.Errorf("expected 1 run result, got %d", len(result.Runs))
	}
}
