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

func init() {
	benchmarkCmd.Flags().StringVarP(&benchModels, "models", "m", "", "comma-separated list of models to benchmark")
	benchmarkCmd.Flags().IntVarP(&benchRuns, "runs", "r", 0, "number of runs per model (overrides suite config)")
	benchmarkCmd.Flags().StringVarP(&benchVersion, "version", "v", "", "benchmark against specific prompt version")
	benchmarkCmd.Flags().StringVarP(&benchOutput, "output", "o", "", "write results to file (JSON format)")
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

	// Create runner with no registered providers (dry run mode)
	// Actual LLM providers will be registered when API keys are configured
	runner := benchmark.NewRunner(database, nil)
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
