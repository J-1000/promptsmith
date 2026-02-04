package testing

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// TestSuite defines a collection of tests for a prompt
type TestSuite struct {
	Name        string     `yaml:"name" json:"name"`
	Prompt      string     `yaml:"prompt" json:"prompt"`
	Description string     `yaml:"description,omitempty" json:"description,omitempty"`
	Version     string     `yaml:"version,omitempty" json:"version,omitempty"` // Optional: pin to specific version
	Tests       []TestCase `yaml:"tests" json:"tests"`
}

// TestCase defines a single test with inputs and assertions
type TestCase struct {
	Name       string            `yaml:"name" json:"name"`
	Inputs     map[string]any    `yaml:"inputs" json:"inputs"`
	Assertions []Assertion       `yaml:"assertions" json:"assertions"`
	Skip       bool              `yaml:"skip,omitempty" json:"skip,omitempty"`
	Tags       []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Assertion defines an expected condition on the output
type Assertion struct {
	Type    AssertionType `yaml:"type" json:"type"`
	Value   any           `yaml:"value,omitempty" json:"value,omitempty"`
	Path    string        `yaml:"path,omitempty" json:"path,omitempty"`       // For json_path assertions
	Message string        `yaml:"message,omitempty" json:"message,omitempty"` // Custom failure message
}

// AssertionType defines the type of assertion
type AssertionType string

const (
	AssertContains      AssertionType = "contains"
	AssertNotContains   AssertionType = "not_contains"
	AssertEquals        AssertionType = "equals"
	AssertMatches       AssertionType = "matches"        // regex
	AssertStartsWith    AssertionType = "starts_with"
	AssertEndsWith      AssertionType = "ends_with"
	AssertMinLength     AssertionType = "min_length"
	AssertMaxLength     AssertionType = "max_length"
	AssertJSONPath      AssertionType = "json_path"      // JSONPath query
	AssertJSONValid     AssertionType = "json_valid"
	AssertNotEmpty      AssertionType = "not_empty"
	AssertLineCount     AssertionType = "line_count"     // exact line count
	AssertMinLines      AssertionType = "min_lines"
	AssertMaxLines      AssertionType = "max_lines"
	AssertWordCount     AssertionType = "word_count"
	AssertSentiment     AssertionType = "sentiment"      // positive, negative, neutral
	AssertLanguage      AssertionType = "language"       // e.g., "en", "es"
)

// TestResult holds the result of running a single test
type TestResult struct {
	TestName  string            `json:"test_name"`
	Passed    bool              `json:"passed"`
	Skipped   bool              `json:"skipped"`
	Output    string            `json:"output,omitempty"`
	Failures  []AssertionResult `json:"failures,omitempty"`
	Error     string            `json:"error,omitempty"`
	DurationMs int64            `json:"duration_ms"`
}

// AssertionResult holds the result of a single assertion
type AssertionResult struct {
	Type     AssertionType `json:"type"`
	Passed   bool          `json:"passed"`
	Expected string        `json:"expected"`
	Actual   string        `json:"actual"`
	Message  string        `json:"message,omitempty"`
}

// SuiteResult holds the result of running an entire test suite
type SuiteResult struct {
	SuiteName  string       `json:"suite_name"`
	PromptName string       `json:"prompt_name"`
	Version    string       `json:"version"`
	Passed     int          `json:"passed"`
	Failed     int          `json:"failed"`
	Skipped    int          `json:"skipped"`
	Total      int          `json:"total"`
	Results    []TestResult `json:"results"`
	DurationMs int64        `json:"duration_ms"`
}

// ParseSuiteFile reads and parses a test suite from a YAML file
func ParseSuiteFile(path string) (*TestSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test suite: %w", err)
	}
	return ParseSuite(data)
}

// ParseSuite parses a test suite from YAML data
func ParseSuite(data []byte) (*TestSuite, error) {
	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse test suite: %w", err)
	}

	if suite.Name == "" {
		return nil, fmt.Errorf("test suite requires a name")
	}
	if suite.Prompt == "" {
		return nil, fmt.Errorf("test suite requires a prompt name")
	}
	if len(suite.Tests) == 0 {
		return nil, fmt.Errorf("test suite requires at least one test")
	}

	// Validate each test
	for i, tc := range suite.Tests {
		if tc.Name == "" {
			return nil, fmt.Errorf("test %d requires a name", i+1)
		}
		if len(tc.Assertions) == 0 && !tc.Skip {
			return nil, fmt.Errorf("test '%s' requires at least one assertion", tc.Name)
		}
		for j, a := range tc.Assertions {
			if err := validateAssertion(a); err != nil {
				return nil, fmt.Errorf("test '%s' assertion %d: %w", tc.Name, j+1, err)
			}
		}
	}

	return &suite, nil
}

func validateAssertion(a Assertion) error {
	switch a.Type {
	case AssertContains, AssertNotContains, AssertEquals, AssertMatches,
		AssertStartsWith, AssertEndsWith:
		if a.Value == nil {
			return fmt.Errorf("%s requires a value", a.Type)
		}
	case AssertMinLength, AssertMaxLength, AssertLineCount, AssertMinLines,
		AssertMaxLines, AssertWordCount:
		if a.Value == nil {
			return fmt.Errorf("%s requires a value", a.Type)
		}
	case AssertJSONPath:
		if a.Path == "" {
			return fmt.Errorf("json_path requires a path")
		}
	case AssertJSONValid, AssertNotEmpty:
		// No value required
	case AssertSentiment:
		if a.Value == nil {
			return fmt.Errorf("sentiment requires a value (positive, negative, neutral)")
		}
	case AssertLanguage:
		if a.Value == nil {
			return fmt.Errorf("language requires a value (e.g., 'en', 'es')")
		}
	case "":
		return fmt.Errorf("assertion type is required")
	default:
		return fmt.Errorf("unknown assertion type: %s", a.Type)
	}
	return nil
}
