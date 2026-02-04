package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/benchmark"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/testing"
	"github.com/spf13/cobra"
)

var (
	testFilter  string
	testVersion string
	testOutput  string
	testLive    bool
	testModel   string
)

var testCmd = &cobra.Command{
	Use:   "test [suite-file...]",
	Short: "Run prompt tests",
	Long: `Run test suites against prompts.

Test suites are YAML files that define test cases with inputs and assertions.
If no files are specified, runs all .test.yaml files in the tests/ directory.

By default, tests run with mock outputs (the rendered prompt).
Use --live to run tests against real LLMs (requires API keys).

Examples:
  promptsmith test                           # Run all tests in tests/
  promptsmith test tests/summarizer.test.yaml
  promptsmith test --filter "basic"          # Run tests matching filter
  promptsmith test --version 1.0.0           # Test specific prompt version
  promptsmith test --live                    # Run with real LLM
  promptsmith test --live --model gpt-4o     # Use specific model`,
	RunE: runTest,
}

func init() {
	testCmd.Flags().StringVarP(&testFilter, "filter", "f", "", "only run tests matching this pattern")
	testCmd.Flags().StringVarP(&testVersion, "version", "v", "", "test against specific prompt version")
	testCmd.Flags().StringVarP(&testOutput, "output", "o", "", "write results to file (JSON format)")
	testCmd.Flags().BoolVar(&testLive, "live", false, "run tests against real LLMs (requires API keys)")
	testCmd.Flags().StringVarP(&testModel, "model", "m", "gpt-4o-mini", "model to use for live testing")
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	// Find test suite files
	var suiteFiles []string
	if len(args) > 0 {
		suiteFiles = args
	} else {
		// Look for *.test.yaml in tests/ directory
		testsDir := filepath.Join(projectRoot, "tests")
		if _, err := os.Stat(testsDir); err == nil {
			matches, err := filepath.Glob(filepath.Join(testsDir, "*.test.yaml"))
			if err != nil {
				return fmt.Errorf("failed to find test files: %w", err)
			}
			suiteFiles = matches
		}
	}

	if len(suiteFiles) == 0 {
		fmt.Println("No test suites found.")
		fmt.Println("Create test files in tests/*.test.yaml or specify files directly.")
		return nil
	}

	// Set up executor
	var executor testing.OutputExecutor
	if testLive {
		// Use real LLM executor
		registry := benchmark.NewProviderRegistry()

		// Register OpenAI if API key available
		if os.Getenv("OPENAI_API_KEY") != "" {
			if p, err := benchmark.NewOpenAIProvider(); err == nil {
				registry.Register(p)
			}
		}

		// Register Anthropic if API key available
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			if p, err := benchmark.NewAnthropicProvider(); err == nil {
				registry.Register(p)
			}
		}

		executor = testing.NewLLMExecutor(registry, testing.WithModel(testModel))
		if !jsonOut {
			fmt.Printf("Running tests with live LLM (%s)\n", testModel)
		}
	}

	// Parse and run all suites
	runner := testing.NewRunner(database, executor)
	var allResults []*testing.SuiteResult
	totalPassed, totalFailed, totalSkipped := 0, 0, 0

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	for _, file := range suiteFiles {
		suite, err := testing.ParseSuiteFile(file)
		if err != nil {
			fmt.Printf("%s Error parsing %s: %v\n", red("✗"), file, err)
			continue
		}

		// Override version if specified
		if testVersion != "" {
			suite.Version = testVersion
		}

		// Apply filter if specified
		if testFilter != "" {
			filtered := make([]testing.TestCase, 0)
			for _, tc := range suite.Tests {
				if strings.Contains(tc.Name, testFilter) {
					filtered = append(filtered, tc)
				}
			}
			suite.Tests = filtered
		}

		if len(suite.Tests) == 0 {
			continue
		}

		result, err := runner.Run(suite)
		if err != nil {
			fmt.Printf("%s Error running %s: %v\n", red("✗"), file, err)
			continue
		}

		allResults = append(allResults, result)
		totalPassed += result.Passed
		totalFailed += result.Failed
		totalSkipped += result.Skipped

		// Print results
		if !jsonOut {
			fmt.Printf("\n%s %s@%s\n", cyan("▶"), result.PromptName, result.Version)

			for _, tr := range result.Results {
				if tr.Skipped {
					fmt.Printf("  %s %s %s\n", yellow("○"), tr.TestName, dim("(skipped)"))
				} else if tr.Passed {
					fmt.Printf("  %s %s %s\n", green("✓"), tr.TestName, dim(fmt.Sprintf("%dms", tr.DurationMs)))
				} else {
					fmt.Printf("  %s %s\n", red("✗"), tr.TestName)
					if tr.Error != "" {
						fmt.Printf("    %s\n", red(tr.Error))
					}
					for _, f := range tr.Failures {
						fmt.Printf("    %s %s\n", dim("├"), f.Message)
						if verbose {
							fmt.Printf("    %s expected: %s\n", dim("│"), f.Expected)
							fmt.Printf("    %s actual: %s\n", dim("└"), f.Actual)
						}
					}
				}
			}
		}
	}

	// Summary
	if jsonOut {
		output := struct {
			Suites  []*testing.SuiteResult `json:"suites"`
			Summary struct {
				Passed  int `json:"passed"`
				Failed  int `json:"failed"`
				Skipped int `json:"skipped"`
				Total   int `json:"total"`
			} `json:"summary"`
		}{
			Suites: allResults,
		}
		output.Summary.Passed = totalPassed
		output.Summary.Failed = totalFailed
		output.Summary.Skipped = totalSkipped
		output.Summary.Total = totalPassed + totalFailed + totalSkipped

		data, _ := json.MarshalIndent(output, "", "  ")

		if testOutput != "" {
			if err := os.WriteFile(testOutput, data, 0644); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
			fmt.Printf("Results written to %s\n", testOutput)
		} else {
			fmt.Println(string(data))
		}
	} else {
		total := totalPassed + totalFailed + totalSkipped
		fmt.Printf("\n%s\n", strings.Repeat("─", 40))
		if totalFailed == 0 {
			fmt.Printf("%s %d passed", green("✓"), totalPassed)
		} else {
			fmt.Printf("%s %d passed, %s %d failed", green("✓"), totalPassed, red("✗"), totalFailed)
		}
		if totalSkipped > 0 {
			fmt.Printf(", %s %d skipped", yellow("○"), totalSkipped)
		}
		fmt.Printf(" %s\n", dim(fmt.Sprintf("(%d total)", total)))

		if testOutput != "" {
			output := struct {
				Suites  []*testing.SuiteResult `json:"suites"`
				Summary struct {
					Passed  int `json:"passed"`
					Failed  int `json:"failed"`
					Skipped int `json:"skipped"`
					Total   int `json:"total"`
				} `json:"summary"`
			}{
				Suites: allResults,
			}
			output.Summary.Passed = totalPassed
			output.Summary.Failed = totalFailed
			output.Summary.Skipped = totalSkipped
			output.Summary.Total = total

			data, _ := json.MarshalIndent(output, "", "  ")
			if err := os.WriteFile(testOutput, data, 0644); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
			fmt.Printf("Results written to %s\n", testOutput)
		}
	}

	// Exit with error code if tests failed
	if totalFailed > 0 {
		os.Exit(1)
	}

	return nil
}
