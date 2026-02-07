package testing

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/prompt"
)

// Runner executes test suites against prompts
type Runner struct {
	db              *db.DB
	executor        OutputExecutor
	UpdateSnapshots bool
}

// OutputExecutor generates output for a rendered prompt
// For now, we use mock outputs; LLM integration comes in Phase 4
type OutputExecutor interface {
	Execute(renderedPrompt string, inputs map[string]any) (string, error)
}

// MockExecutor uses expected outputs defined in test cases
type MockExecutor struct {
	outputs map[string]string // testName -> expected output
}

func NewMockExecutor(outputs map[string]string) *MockExecutor {
	return &MockExecutor{outputs: outputs}
}

func (m *MockExecutor) Execute(renderedPrompt string, inputs map[string]any) (string, error) {
	// For mock testing, we just return the renderedPrompt as "output"
	// In real usage, the test file would specify expected_output
	return renderedPrompt, nil
}

// NewRunner creates a new test runner
func NewRunner(database *db.DB, executor OutputExecutor) *Runner {
	if executor == nil {
		executor = &MockExecutor{}
	}
	return &Runner{
		db:       database,
		executor: executor,
	}
}

// Run executes a test suite and returns results
func (r *Runner) Run(suite *TestSuite) (*SuiteResult, error) {
	startTime := time.Now()

	result := &SuiteResult{
		SuiteName:  suite.Name,
		PromptName: suite.Prompt,
		Results:    make([]TestResult, 0, len(suite.Tests)),
	}

	// Get the prompt
	p, err := r.db.GetPromptByName(suite.Prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}
	if p == nil {
		return nil, fmt.Errorf("prompt '%s' not found", suite.Prompt)
	}

	// Get the version to test
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

	// Run each test
	for _, tc := range suite.Tests {
		testResult := r.runTest(tc, parsed, suite.FilePath)
		result.Results = append(result.Results, testResult)

		if testResult.Skipped {
			result.Skipped++
		} else if testResult.Passed {
			result.Passed++
		} else {
			result.Failed++
		}
		result.Total++
	}

	result.DurationMs = time.Since(startTime).Milliseconds()
	return result, nil
}

func (r *Runner) runTest(tc TestCase, parsed *prompt.ParsedPrompt, suiteFile string) TestResult {
	testStart := time.Now()
	result := TestResult{
		TestName: tc.Name,
		Failures: make([]AssertionResult, 0),
	}

	if tc.Skip {
		result.Skipped = true
		result.DurationMs = time.Since(testStart).Milliseconds()
		return result
	}

	// Render the prompt with test inputs
	rendered, err := renderPrompt(parsed.Content, tc.Inputs)
	if err != nil {
		result.Error = fmt.Sprintf("failed to render prompt: %s", err)
		result.DurationMs = time.Since(testStart).Milliseconds()
		return result
	}

	// Get output (for now, use the rendered prompt or mock)
	output, err := r.executor.Execute(rendered, tc.Inputs)
	if err != nil {
		result.Error = fmt.Sprintf("execution failed: %s", err)
		result.DurationMs = time.Since(testStart).Milliseconds()
		return result
	}
	result.Output = output

	// Run assertions
	result.Passed = true
	for _, assertion := range tc.Assertions {
		// For snapshot assertions, inject the expected_output as the value
		if assertion.Type == AssertSnapshot {
			if r.UpdateSnapshots && suiteFile != "" {
				// Update mode: store current output as the new snapshot
				if err := UpdateSnapshot(suiteFile, tc.Name, output); err != nil {
					result.Error = fmt.Sprintf("failed to update snapshot: %s", err)
					result.DurationMs = time.Since(testStart).Milliseconds()
					return result
				}
				// Mark as passed since we just updated
				continue
			}
			assertion.Value = tc.ExpectedOutput
		}
		ar := assertion.Evaluate(output)
		if !ar.Passed {
			result.Passed = false
			result.Failures = append(result.Failures, ar)
		}
	}

	result.DurationMs = time.Since(testStart).Milliseconds()
	return result
}

func renderPrompt(tmplBody string, inputs map[string]any) (string, error) {
	tmpl, err := template.New("prompt").Parse(tmplBody)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, inputs); err != nil {
		return "", err
	}
	return buf.String(), nil
}
