package benchmark

import (
	"context"
	"fmt"
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

// mockBenchmarkProvider implements Provider for testing
// Uses "openai" as name so it matches the GetForModel lookup for gpt-* models
type mockBenchmarkProvider struct {
	responses []*CompletionResponse
	errors    []error
	callCount int
}

func (m *mockBenchmarkProvider) Name() string {
	return "openai" // Use openai so GetProviderForModel finds us for gpt-* models
}

func (m *mockBenchmarkProvider) Models() []string {
	return []string{"gpt-4o", "gpt-4o-mini"}
}

func (m *mockBenchmarkProvider) SupportsModel(model string) bool {
	return model == "gpt-4o" || model == "gpt-4o-mini"
}

func (m *mockBenchmarkProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	idx := m.callCount
	m.callCount++

	if idx < len(m.errors) && m.errors[idx] != nil {
		return nil, m.errors[idx]
	}

	if idx < len(m.responses) {
		return m.responses[idx], nil
	}

	// Default response
	return &CompletionResponse{
		Content:      "default response",
		Model:        req.Model,
		PromptTokens: 100,
		OutputTokens: 50,
		TotalTokens:  150,
		LatencyMs:    200,
		Cost:         0.005,
	}, nil
}

func TestBenchmarkModelWithMockProvider(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &mockBenchmarkProvider{
		responses: []*CompletionResponse{
			{LatencyMs: 100, PromptTokens: 100, OutputTokens: 50, TotalTokens: 150, Cost: 0.003},
			{LatencyMs: 200, PromptTokens: 100, OutputTokens: 60, TotalTokens: 160, Cost: 0.004},
			{LatencyMs: 150, PromptTokens: 100, OutputTokens: 55, TotalTokens: 155, Cost: 0.0035},
		},
	}
	registry.Register(provider)

	runner := NewRunner(nil, registry)
	modelResult, runs := runner.benchmarkModel(nil, "gpt-4o", "test prompt", 3)

	if modelResult.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", modelResult.Errors)
	}
	if modelResult.ErrorRate != 0 {
		t.Errorf("expected error rate 0, got %f", modelResult.ErrorRate)
	}
	if len(runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(runs))
	}

	// Check latency calculations
	if modelResult.LatencyAvgMs != 150 {
		t.Errorf("expected avg latency 150, got %f", modelResult.LatencyAvgMs)
	}

	// Verify all runs have correct data
	for _, run := range runs {
		if run.Error != "" {
			t.Errorf("unexpected error: %s", run.Error)
		}
		if run.PromptTokens != 100 {
			t.Errorf("expected 100 prompt tokens, got %d", run.PromptTokens)
		}
	}
}

func TestBenchmarkModelMixedResults(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &mockBenchmarkProvider{
		responses: []*CompletionResponse{
			{LatencyMs: 100, PromptTokens: 100, OutputTokens: 50, TotalTokens: 150, Cost: 0.003},
			nil, // This will use error
			{LatencyMs: 200, PromptTokens: 100, OutputTokens: 50, TotalTokens: 150, Cost: 0.003},
		},
		errors: []error{
			nil,
			fmt.Errorf("API error"),
			nil,
		},
	}
	registry.Register(provider)

	runner := NewRunner(nil, registry)
	modelResult, runs := runner.benchmarkModel(nil, "gpt-4o-mini", "test prompt", 3)

	if modelResult.Errors != 1 {
		t.Errorf("expected 1 error, got %d", modelResult.Errors)
	}

	expectedErrorRate := float64(1) / float64(3)
	if modelResult.ErrorRate != expectedErrorRate {
		t.Errorf("expected error rate %f, got %f", expectedErrorRate, modelResult.ErrorRate)
	}

	// Check that we have mix of success and failure
	successCount := 0
	errorCount := 0
	for _, run := range runs {
		if run.Error != "" {
			errorCount++
		} else {
			successCount++
		}
	}

	if successCount != 2 {
		t.Errorf("expected 2 successes, got %d", successCount)
	}
	if errorCount != 1 {
		t.Errorf("expected 1 error, got %d", errorCount)
	}
}

func TestBenchmarkModelCostCalculation(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &mockBenchmarkProvider{
		responses: []*CompletionResponse{
			{LatencyMs: 100, Cost: 0.01},
			{LatencyMs: 200, Cost: 0.02},
			{LatencyMs: 150, Cost: 0.03},
		},
	}
	registry.Register(provider)

	runner := NewRunner(nil, registry)
	modelResult, _ := runner.benchmarkModel(nil, "gpt-4o", "test prompt", 3)

	// Total cost should be 0.06
	if modelResult.TotalCost != 0.06 {
		t.Errorf("expected total cost 0.06, got %f", modelResult.TotalCost)
	}

	// Average cost per request should be 0.02
	if modelResult.CostPerRequest != 0.02 {
		t.Errorf("expected cost per request 0.02, got %f", modelResult.CostPerRequest)
	}
}

func TestPercentileEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		sorted []int64
		p      int
		want   int64
	}{
		{"p100", []int64{10, 20, 30, 40, 50}, 100, 50},
		{"two elements p50", []int64{10, 20}, 50, 20},
		{"three elements p50", []int64{10, 20, 30}, 50, 20},
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
