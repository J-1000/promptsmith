package benchmark

import (
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
