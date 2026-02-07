package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/benchmark"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var (
	benchModels  string
	benchRuns    int
	benchVersion string
	benchOutput  string
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark [suite-file...]",
	Short: "Run model benchmarks",
	Long: `Run benchmark suites to compare prompt performance across models.

Benchmark suites are YAML files that define models to test and metrics to collect.
If no files are specified, runs all .bench.yaml files in the benchmarks/ directory.

Examples:
  promptsmith benchmark                              # Run all benchmarks
  promptsmith benchmark benchmarks/summarizer.bench.yaml
  promptsmith benchmark --models gpt-4o,claude-sonnet
  promptsmith benchmark --runs 10                    # 10 runs per model
  promptsmith benchmark -o results.json              # Save results`,
	RunE: runBenchmark,
}

var benchmarkCompareCmd = &cobra.Command{
	Use:   "compare <file1.json> <file2.json>",
	Short: "Compare two benchmark result files",
	Long: `Compare two JSON benchmark result files and show a delta table.

Examples:
  promptsmith benchmark compare baseline.json latest.json`,
	Args: cobra.ExactArgs(2),
	RunE: runBenchmarkCompare,
}

func init() {
	benchmarkCmd.Flags().StringVarP(&benchModels, "models", "m", "", "comma-separated list of models to benchmark")
	benchmarkCmd.Flags().IntVarP(&benchRuns, "runs", "r", 0, "number of runs per model (overrides suite config)")
	benchmarkCmd.Flags().StringVarP(&benchVersion, "version", "v", "", "benchmark against specific prompt version")
	benchmarkCmd.Flags().StringVarP(&benchOutput, "output", "o", "", "write results to file (JSON format)")
	benchmarkCmd.AddCommand(benchmarkCompareCmd)
	rootCmd.AddCommand(benchmarkCmd)
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	// Find benchmark suite files
	var suiteFiles []string
	if len(args) > 0 {
		suiteFiles = args
	} else {
		// Look for *.bench.yaml in benchmarks/ directory
		benchDir := filepath.Join(projectRoot, "benchmarks")
		if _, err := os.Stat(benchDir); err == nil {
			matches, err := filepath.Glob(filepath.Join(benchDir, "*.bench.yaml"))
			if err != nil {
				return fmt.Errorf("failed to find benchmark files: %w", err)
			}
			suiteFiles = matches
		}
	}

	if len(suiteFiles) == 0 {
		fmt.Println("No benchmark suites found.")
		fmt.Println("Create benchmark files in benchmarks/*.bench.yaml or specify files directly.")
		return nil
	}

	// Create provider registry and register available providers
	registry := benchmark.NewProviderRegistry()
	if openai, err := benchmark.NewOpenAIProvider(); err == nil {
		registry.Register(openai)
	}
	if anthropic, err := benchmark.NewAnthropicProvider(); err == nil {
		registry.Register(anthropic)
	}

	runner := benchmark.NewRunner(database, registry)
	var allResults []*benchmark.BenchmarkResult

	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	for _, file := range suiteFiles {
		suite, err := benchmark.ParseSuiteFile(file)
		if err != nil {
			fmt.Printf("%s Error parsing %s: %v\n", color.RedString("✗"), file, err)
			continue
		}

		// Override version if specified
		if benchVersion != "" {
			suite.Version = benchVersion
		}

		// Override runs if specified
		if benchRuns > 0 {
			suite.RunsPerModel = benchRuns
		}

		// Override models if specified
		if benchModels != "" {
			suite.Models = strings.Split(benchModels, ",")
			for i := range suite.Models {
				suite.Models[i] = strings.TrimSpace(suite.Models[i])
			}
		}

		if !jsonOut {
			fmt.Printf("\n%s %s@%s\n", cyan("▶"), suite.Prompt, suite.Version)
			fmt.Printf("  Models: %s\n", strings.Join(suite.Models, ", "))
			fmt.Printf("  Runs per model: %d\n", suite.RunsPerModel)
		}

		result, err := runner.Run(context.Background(), suite)
		if err != nil {
			fmt.Printf("%s Error running %s: %v\n", color.RedString("✗"), file, err)
			continue
		}

		allResults = append(allResults, result)

		// Print results table
		if !jsonOut {
			printBenchmarkTable(result)
		}
	}

	// Output JSON if requested
	if jsonOut {
		data, _ := json.MarshalIndent(allResults, "", "  ")
		if benchOutput != "" {
			if err := os.WriteFile(benchOutput, data, 0644); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
			fmt.Printf("Results written to %s\n", benchOutput)
		} else {
			fmt.Println(string(data))
		}
	} else if benchOutput != "" {
		data, _ := json.MarshalIndent(allResults, "", "  ")
		if err := os.WriteFile(benchOutput, data, 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf("\n%s Results written to %s\n", dim("→"), benchOutput)
	}

	// Print recommendation
	if !jsonOut && len(allResults) > 0 {
		for _, result := range allResults {
			printRecommendation(result, yellow)
		}
	}

	return nil
}

func printBenchmarkTable(result *benchmark.BenchmarkResult) {
	dim := color.New(color.Faint).SprintFunc()

	// Table header
	fmt.Println()
	fmt.Printf("  %-20s %10s %10s %12s %10s\n",
		"Model", "Latency", "Tokens", "Cost/Req", "Errors")
	fmt.Printf("  %s\n", dim(strings.Repeat("─", 66)))

	// Table rows
	for _, m := range result.Models {
		latency := fmt.Sprintf("%.0fms", m.LatencyP50Ms)
		if m.LatencyP50Ms == 0 && m.Errors > 0 {
			latency = "-"
		}

		tokens := fmt.Sprintf("%.0f", m.TotalTokensAvg)
		if m.TotalTokensAvg == 0 {
			tokens = "-"
		}

		cost := fmt.Sprintf("$%.4f", m.CostPerRequest)
		if m.CostPerRequest == 0 {
			cost = "-"
		}

		errors := "-"
		if m.Errors > 0 {
			errors = fmt.Sprintf("%d (%.0f%%)", m.Errors, m.ErrorRate*100)
		}

		fmt.Printf("  %-20s %10s %10s %12s %10s\n",
			m.Model, latency, tokens, cost, errors)
	}

	fmt.Printf("  %s\n", dim(strings.Repeat("─", 66)))
	fmt.Printf("  %s %dms\n", dim("Total time:"), result.DurationMs)
}

func runBenchmarkCompare(cmd *cobra.Command, args []string) error {
	file1, file2 := args[0], args[1]

	data1, err := os.ReadFile(file1)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", file1, err)
	}

	data2, err := os.ReadFile(file2)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", file2, err)
	}

	var results1, results2 []*benchmark.BenchmarkResult
	if err := json.Unmarshal(data1, &results1); err != nil {
		return fmt.Errorf("failed to parse %s: %w", file1, err)
	}
	if err := json.Unmarshal(data2, &results2); err != nil {
		return fmt.Errorf("failed to parse %s: %w", file2, err)
	}

	if len(results1) == 0 || len(results2) == 0 {
		return fmt.Errorf("both files must contain at least one benchmark result")
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Printf("\n%s Comparing %s vs %s\n", cyan("▶"), file1, file2)

	// Build model map from both results
	r1 := results1[0]
	r2 := results2[0]

	modelMap1 := map[string]*benchmark.ModelResult{}
	for i := range r1.Models {
		modelMap1[r1.Models[i].Model] = &r1.Models[i]
	}

	fmt.Printf("\n  %-20s %12s %12s %12s\n", "Model", "Latency Δ", "Cost Δ", "Errors Δ")
	fmt.Printf("  %s\n", dim(strings.Repeat("─", 60)))

	for _, m2 := range r2.Models {
		m1, ok := modelMap1[m2.Model]
		if !ok {
			fmt.Printf("  %-20s %12s %12s %12s\n", m2.Model, "new", "new", "new")
			continue
		}

		latDelta := m2.LatencyP50Ms - m1.LatencyP50Ms
		costDelta := m2.CostPerRequest - m1.CostPerRequest
		errDelta := m2.Errors - m1.Errors

		latStr := formatDelta(latDelta, "ms", true, green, red)
		costStr := formatCostDelta(costDelta, green, red)
		errStr := formatIntDelta(errDelta, green, red)

		fmt.Printf("  %-20s %12s %12s %12s\n", m2.Model, latStr, costStr, errStr)
	}

	fmt.Printf("  %s\n", dim(strings.Repeat("─", 60)))
	return nil
}

func formatDelta(delta float64, unit string, lowerBetter bool, green, red func(a ...interface{}) string) string {
	if delta == 0 {
		return fmt.Sprintf("0%s", unit)
	}
	sign := "+"
	colorFn := red
	if (lowerBetter && delta < 0) || (!lowerBetter && delta > 0) {
		colorFn = green
	}
	if delta < 0 {
		sign = ""
	}
	return colorFn(fmt.Sprintf("%s%.0f%s", sign, delta, unit))
}

func formatCostDelta(delta float64, green, red func(a ...interface{}) string) string {
	if delta == 0 {
		return "$0"
	}
	sign := "+"
	colorFn := red
	if delta < 0 {
		sign = ""
		colorFn = green
	}
	return colorFn(fmt.Sprintf("%s$%.4f", sign, delta))
}

func formatIntDelta(delta int, green, red func(a ...interface{}) string) string {
	if delta == 0 {
		return "0"
	}
	sign := "+"
	colorFn := red
	if delta < 0 {
		sign = ""
		colorFn = green
	}
	return colorFn(fmt.Sprintf("%s%d", sign, delta))
}

func printRecommendation(result *benchmark.BenchmarkResult, yellow func(a ...interface{}) string) {
	if len(result.Models) < 2 {
		return
	}

	// Find best latency and best cost
	var bestLatency, bestCost *benchmark.ModelResult
	for i := range result.Models {
		m := &result.Models[i]
		if m.ErrorRate >= 1.0 {
			continue
		}
		if bestLatency == nil || (m.LatencyP50Ms > 0 && m.LatencyP50Ms < bestLatency.LatencyP50Ms) {
			bestLatency = m
		}
		if bestCost == nil || (m.CostPerRequest > 0 && m.CostPerRequest < bestCost.CostPerRequest) {
			bestCost = m
		}
	}

	fmt.Println()
	if bestLatency != nil && bestCost != nil {
		if bestLatency.Model == bestCost.Model {
			fmt.Printf("  %s %s (best latency & cost)\n", yellow("★"), bestLatency.Model)
		} else {
			fmt.Printf("  %s %s for speed, %s for cost\n",
				yellow("★"), bestLatency.Model, bestCost.Model)
		}
	}
}
