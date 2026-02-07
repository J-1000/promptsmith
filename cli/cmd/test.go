package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/promptsmith/cli/internal/benchmark"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/testing"
	"github.com/spf13/cobra"
)

var (
	testFilter          string
	testVersion         string
	testOutput          string
	testLive            bool
	testModel           string
	testWatch           bool
	testUpdateSnapshots bool
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
  promptsmith test --live --model gpt-4o     # Use specific model
  promptsmith test --watch                   # Re-run tests on file changes
  promptsmith test --update-snapshots        # Update snapshot assertions`,
	RunE: runTest,
}

func init() {
	testCmd.Flags().StringVarP(&testFilter, "filter", "f", "", "only run tests matching this pattern")
	testCmd.Flags().StringVarP(&testVersion, "version", "v", "", "test against specific prompt version")
	testCmd.Flags().StringVarP(&testOutput, "output", "o", "", "write results to file (JSON format)")
	testCmd.Flags().BoolVar(&testLive, "live", false, "run tests against real LLMs (requires API keys)")
	testCmd.Flags().StringVarP(&testModel, "model", "m", "gpt-4o-mini", "model to use for live testing")
	testCmd.Flags().BoolVarP(&testWatch, "watch", "w", false, "watch for file changes and re-run tests")
	testCmd.Flags().BoolVar(&testUpdateSnapshots, "update-snapshots", false, "update snapshot assertions with current output")
	rootCmd.AddCommand(testCmd)
}

type testRunContext struct {
	projectRoot string
	database    *db.DB
	suiteFiles  []string
	executor    testing.OutputExecutor
}

func setupTestContext(args []string) (*testRunContext, error) {
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return nil, err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return nil, err
	}

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
				return nil, fmt.Errorf("failed to find test files: %w", err)
			}
			suiteFiles = matches
		}
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
	}

	return &testRunContext{
		projectRoot: projectRoot,
		database:    database,
		suiteFiles:  suiteFiles,
		executor:    executor,
	}, nil
}

func executeTests(ctx *testRunContext) (passed, failed, skipped int, results []*testing.SuiteResult) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	runner := testing.NewRunner(ctx.database, ctx.executor)
	runner.UpdateSnapshots = testUpdateSnapshots

	for _, file := range ctx.suiteFiles {
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

		results = append(results, result)
		passed += result.Passed
		failed += result.Failed
		skipped += result.Skipped

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

	return passed, failed, skipped, results
}

func printTestSummary(passed, failed, skipped int, results []*testing.SuiteResult) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	total := passed + failed + skipped

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
			Suites: results,
		}
		output.Summary.Passed = passed
		output.Summary.Failed = failed
		output.Summary.Skipped = skipped
		output.Summary.Total = total

		data, _ := json.MarshalIndent(output, "", "  ")

		if testOutput != "" {
			if err := os.WriteFile(testOutput, data, 0644); err != nil {
				fmt.Printf("Failed to write output: %v\n", err)
			} else {
				fmt.Printf("Results written to %s\n", testOutput)
			}
		} else {
			fmt.Println(string(data))
		}
	} else {
		fmt.Printf("\n%s\n", strings.Repeat("─", 40))
		if failed == 0 {
			fmt.Printf("%s %d passed", green("✓"), passed)
		} else {
			fmt.Printf("%s %d passed, %s %d failed", green("✓"), passed, red("✗"), failed)
		}
		if skipped > 0 {
			fmt.Printf(", %s %d skipped", yellow("○"), skipped)
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
				Suites: results,
			}
			output.Summary.Passed = passed
			output.Summary.Failed = failed
			output.Summary.Skipped = skipped
			output.Summary.Total = total

			data, _ := json.MarshalIndent(output, "", "  ")
			if err := os.WriteFile(testOutput, data, 0644); err != nil {
				fmt.Printf("Failed to write output: %v\n", err)
			} else {
				fmt.Printf("Results written to %s\n", testOutput)
			}
		}
	}
}

func runTestWatch(ctx *testRunContext) error {
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the tests directory
	testsDir := filepath.Join(ctx.projectRoot, "tests")
	if err := watcher.Add(testsDir); err != nil {
		return fmt.Errorf("failed to watch tests directory: %w", err)
	}

	// Watch the prompts directory
	promptsDir := filepath.Join(ctx.projectRoot, "prompts")
	if err := watcher.Add(promptsDir); err != nil {
		// Prompts dir might not exist, that's okay
		_ = err
	}

	// Initial run
	fmt.Printf("%s Watching for changes... %s\n", cyan("👁"), dim("(Ctrl+C to stop)"))
	passed, failed, skipped, results := executeTests(ctx)
	printTestSummary(passed, failed, skipped, results)

	// Debounce timer to avoid multiple rapid triggers
	var debounce <-chan time.Time

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only react to writes and creates
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// Check if it's a relevant file
				ext := filepath.Ext(event.Name)
				if ext == ".yaml" || ext == ".yml" || ext == ".prompt" {
					// Debounce - wait 100ms before running
					debounce = time.After(100 * time.Millisecond)
				}
			}

		case <-debounce:
			// Clear screen and re-run
			fmt.Print("\033[H\033[2J")
			fmt.Printf("%s File changed, re-running tests...\n", cyan("↻"))
			passed, failed, skipped, results := executeTests(ctx)
			printTestSummary(passed, failed, skipped, results)
			fmt.Printf("\n%s Watching for changes... %s\n", cyan("👁"), dim("(Ctrl+C to stop)"))

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

func runTest(cmd *cobra.Command, args []string) error {
	ctx, err := setupTestContext(args)
	if err != nil {
		return err
	}
	defer ctx.database.Close()

	if len(ctx.suiteFiles) == 0 {
		fmt.Println("No test suites found.")
		fmt.Println("Create test files in tests/*.test.yaml or specify files directly.")
		return nil
	}

	if testLive && !jsonOut {
		fmt.Printf("Running tests with live LLM (%s)\n", testModel)
	}

	// Watch mode
	if testWatch {
		return runTestWatch(ctx)
	}

	// Single run mode
	passed, failed, skipped, results := executeTests(ctx)
	printTestSummary(passed, failed, skipped, results)

	// Exit with error code if tests failed
	if failed > 0 {
		os.Exit(1)
	}

	return nil
}
