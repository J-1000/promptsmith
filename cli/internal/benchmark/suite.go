package benchmark

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Suite defines a benchmark configuration for a prompt
type Suite struct {
	Name        string   `yaml:"name" json:"name"`
	Prompt      string   `yaml:"prompt" json:"prompt"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Version     string   `yaml:"version,omitempty" json:"version,omitempty"`
	Models      []string `yaml:"models" json:"models"`
	Dataset     string   `yaml:"dataset,omitempty" json:"dataset,omitempty"`
	RunsPerModel int     `yaml:"runs_per_model,omitempty" json:"runs_per_model,omitempty"`
	Metrics     []Metric `yaml:"metrics,omitempty" json:"metrics,omitempty"`
	Variables   map[string]any `yaml:"variables,omitempty" json:"variables,omitempty"`
}

// Metric defines what to measure in the benchmark
type Metric struct {
	Name   string         `yaml:"name" json:"name"`
	Type   MetricType     `yaml:"type" json:"type"`
	Config map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

// MetricType defines the type of metric to collect
type MetricType string

const (
	MetricLatency       MetricType = "latency"
	MetricTokens        MetricType = "tokens"
	MetricCost          MetricType = "cost"
	MetricQuality       MetricType = "quality"
	MetricLatencyP50    MetricType = "latency_p50"
	MetricLatencyP99    MetricType = "latency_p99"
	MetricTotalTokens   MetricType = "total_tokens"
	MetricPromptTokens  MetricType = "prompt_tokens"
	MetricOutputTokens  MetricType = "output_tokens"
	MetricCostPerReq    MetricType = "cost_per_request"
)

// ModelResult holds benchmark results for a single model
type ModelResult struct {
	Model          string  `json:"model"`
	Runs           int     `json:"runs"`
	LatencyP50Ms   float64 `json:"latency_p50_ms"`
	LatencyP99Ms   float64 `json:"latency_p99_ms"`
	LatencyAvgMs   float64 `json:"latency_avg_ms"`
	TotalTokensAvg float64 `json:"total_tokens_avg"`
	PromptTokens   int     `json:"prompt_tokens"`
	OutputTokensAvg float64 `json:"output_tokens_avg"`
	CostPerRequest float64 `json:"cost_per_request"`
	TotalCost      float64 `json:"total_cost"`
	Errors         int     `json:"errors"`
	ErrorRate      float64 `json:"error_rate"`
}

// RunResult holds individual run data
type RunResult struct {
	Model        string  `json:"model"`
	LatencyMs    int64   `json:"latency_ms"`
	PromptTokens int     `json:"prompt_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	Cost         float64 `json:"cost"`
	Output       string  `json:"output,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// BenchmarkResult holds the complete benchmark results
type BenchmarkResult struct {
	SuiteName   string         `json:"suite_name"`
	PromptName  string         `json:"prompt_name"`
	Version     string         `json:"version"`
	Models      []ModelResult  `json:"models"`
	Runs        []RunResult    `json:"runs,omitempty"`
	DurationMs  int64          `json:"duration_ms"`
	StartedAt   string         `json:"started_at"`
	CompletedAt string         `json:"completed_at"`
}

// ParseSuiteFile reads and parses a benchmark suite from a YAML file
func ParseSuiteFile(path string) (*Suite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read benchmark suite: %w", err)
	}
	return ParseSuite(data)
}

// ParseSuite parses a benchmark suite from YAML data
func ParseSuite(data []byte) (*Suite, error) {
	var suite Suite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse benchmark suite: %w", err)
	}

	if suite.Name == "" {
		return nil, fmt.Errorf("benchmark suite requires a name")
	}
	if suite.Prompt == "" {
		return nil, fmt.Errorf("benchmark suite requires a prompt name")
	}
	if len(suite.Models) == 0 {
		return nil, fmt.Errorf("benchmark suite requires at least one model")
	}

	// Set defaults
	if suite.RunsPerModel == 0 {
		suite.RunsPerModel = 3
	}

	// Validate models
	for i, model := range suite.Models {
		if model == "" {
			return nil, fmt.Errorf("model at index %d is empty", i)
		}
	}

	return &suite, nil
}
