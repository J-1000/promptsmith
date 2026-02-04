package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"text/template"
	"time"

	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/prompt"
)

// Runner executes benchmark suites against prompts
type Runner struct {
	db       *db.DB
	registry *ProviderRegistry
}

// NewRunner creates a new benchmark runner
func NewRunner(database *db.DB, registry *ProviderRegistry) *Runner {
	if registry == nil {
		registry = NewProviderRegistry()
	}
	return &Runner{
		db:       database,
		registry: registry,
	}
}

// Run executes a benchmark suite and returns results
func (r *Runner) Run(ctx context.Context, suite *Suite) (*BenchmarkResult, error) {
	startTime := time.Now()

	result := &BenchmarkResult{
		SuiteName:  suite.Name,
		PromptName: suite.Prompt,
		StartedAt:  startTime.Format(time.RFC3339),
		Models:     make([]ModelResult, 0, len(suite.Models)),
		Runs:       make([]RunResult, 0),
	}

	// Get the prompt
	p, err := r.db.GetPromptByName(suite.Prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}
	if p == nil {
		return nil, fmt.Errorf("prompt '%s' not found", suite.Prompt)
	}

	// Get the version to benchmark
	var version *db.PromptVersion
	if suite.Version != "" {
		version, err = r.db.GetVersionByString(p.ID, suite.Version)
		if err != nil {
			return nil, err
		}
		if version == nil {
			return nil, fmt.Errorf("version '%s' not found for prompt '%s'", suite.Version, suite.Prompt)
		}
	} else {
		version, err = r.db.GetLatestVersion(p.ID)
		if err != nil {
			return nil, err
		}
		if version == nil {
			return nil, fmt.Errorf("no versions found for prompt '%s'", suite.Prompt)
		}
	}
	result.Version = version.Version

	// Parse the prompt template
	parsed, err := prompt.Parse(version.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt: %w", err)
	}

	// Render the prompt with any variables
	rendered, err := renderPrompt(parsed.Content, suite.Variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	// Run benchmarks for each model
	for _, model := range suite.Models {
		modelResult, runs := r.benchmarkModel(ctx, model, rendered, suite.RunsPerModel)
		result.Models = append(result.Models, modelResult)
		result.Runs = append(result.Runs, runs...)
	}

	result.DurationMs = time.Since(startTime).Milliseconds()
	result.CompletedAt = time.Now().Format(time.RFC3339)

	return result, nil
}

func (r *Runner) benchmarkModel(ctx context.Context, model, prompt string, runs int) (ModelResult, []RunResult) {
	result := ModelResult{
		Model: model,
		Runs:  runs,
	}

	runResults := make([]RunResult, 0, runs)
	latencies := make([]int64, 0, runs)
	var totalTokens, outputTokens, errors int
	var totalCost float64
	var promptTokens int

	provider, err := r.registry.GetForModel(model)
	if err != nil {
		// No provider registered, return empty results
		result.Errors = runs
		result.ErrorRate = 1.0
		for i := 0; i < runs; i++ {
			runResults = append(runResults, RunResult{
				Model: model,
				Error: err.Error(),
			})
		}
		return result, runResults
	}

	for i := 0; i < runs; i++ {
		req := CompletionRequest{
			Model:       model,
			Prompt:      prompt,
			MaxTokens:   1024,
			Temperature: 0.7,
		}

		resp, err := provider.Complete(ctx, req)
		runResult := RunResult{Model: model}

		if err != nil {
			runResult.Error = err.Error()
			errors++
		} else {
			runResult.LatencyMs = resp.LatencyMs
			runResult.PromptTokens = resp.PromptTokens
			runResult.OutputTokens = resp.OutputTokens
			runResult.TotalTokens = resp.TotalTokens
			runResult.Cost = resp.Cost
			runResult.Output = resp.Content

			latencies = append(latencies, resp.LatencyMs)
			promptTokens = resp.PromptTokens // same for all runs
			outputTokens += resp.OutputTokens
			totalTokens += resp.TotalTokens
			totalCost += resp.Cost
		}

		runResults = append(runResults, runResult)
	}

	successfulRuns := runs - errors
	if successfulRuns > 0 {
		// Calculate percentiles
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		result.LatencyP50Ms = float64(percentile(latencies, 50))
		result.LatencyP99Ms = float64(percentile(latencies, 99))
		result.LatencyAvgMs = avg(latencies)

		result.PromptTokens = promptTokens
		result.TotalTokensAvg = float64(totalTokens) / float64(successfulRuns)
		result.OutputTokensAvg = float64(outputTokens) / float64(successfulRuns)
		result.CostPerRequest = totalCost / float64(successfulRuns)
		result.TotalCost = totalCost
	}

	result.Errors = errors
	result.ErrorRate = float64(errors) / float64(runs)

	return result, runResults
}

func renderPrompt(tmplBody string, vars map[string]any) (string, error) {
	if vars == nil || len(vars) == 0 {
		return tmplBody, nil
	}

	tmpl, err := template.New("prompt").Parse(tmplBody)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func avg(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum int64
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}
