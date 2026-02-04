package benchmark

import (
	"testing"
)

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		sorted []int64
		p      int
		want   int64
	}{
		{"empty", []int64{}, 50, 0},
		{"single", []int64{100}, 50, 100},
		{"p50 of 10", []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, 50, 60},
		{"p99 of 10", []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, 99, 100},
		{"p0 of 10", []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.sorted, tt.p)
			if got != tt.want {
				t.Errorf("percentile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAvg(t *testing.T) {
	tests := []struct {
		name   string
		values []int64
		want   float64
	}{
		{"empty", []int64{}, 0},
		{"single", []int64{100}, 100},
		{"multiple", []int64{10, 20, 30}, 20},
		{"with zeros", []int64{0, 0, 30}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := avg(tt.values)
			if got != tt.want {
				t.Errorf("avg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderPrompt(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]any
		want     string
		wantErr  bool
	}{
		{
			name:     "no variables",
			template: "Hello world",
			vars:     nil,
			want:     "Hello world",
		},
		{
			name:     "with variable",
			template: "Hello {{.name}}",
			vars:     map[string]any{"name": "Alice"},
			want:     "Hello Alice",
		},
		{
			name:     "multiple variables",
			template: "{{.greeting}} {{.name}}, you have {{.count}} messages",
			vars:     map[string]any{"greeting": "Hi", "name": "Bob", "count": 5},
			want:     "Hi Bob, you have 5 messages",
		},
		{
			name:     "empty vars map",
			template: "Hello world",
			vars:     map[string]any{},
			want:     "Hello world",
		},
		{
			name:     "invalid template",
			template: "Hello {{.name",
			vars:     map[string]any{"name": "Alice"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderPrompt(tt.template, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("renderPrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRunner(t *testing.T) {
	// Test with nil registry
	runner := NewRunner(nil, nil)
	if runner == nil {
		t.Error("expected non-nil runner")
	}
	if runner.registry == nil {
		t.Error("expected non-nil registry")
	}

	// Test with provided registry
	registry := NewProviderRegistry()
	runner2 := NewRunner(nil, registry)
	if runner2.registry != registry {
		t.Error("expected registry to be set")
	}
}

func TestBenchmarkModelNoProvider(t *testing.T) {
	runner := NewRunner(nil, NewProviderRegistry())

	// Benchmark a model with no registered provider
	modelResult, runs := runner.benchmarkModel(nil, "unknown-model", "test prompt", 3)

	if modelResult.Errors != 3 {
		t.Errorf("expected 3 errors, got %d", modelResult.Errors)
	}
	if modelResult.ErrorRate != 1.0 {
		t.Errorf("expected error rate 1.0, got %f", modelResult.ErrorRate)
	}
	if len(runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(runs))
	}
	for _, run := range runs {
		if run.Error == "" {
			t.Error("expected error in run result")
		}
	}
}
