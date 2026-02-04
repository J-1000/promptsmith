package testing

import (
	"testing"
)

func TestAssertionEvaluate(t *testing.T) {
	tests := []struct {
		name       string
		assertion  Assertion
		output     string
		wantPassed bool
	}{
		// Contains
		{
			name:       "contains - pass",
			assertion:  Assertion{Type: AssertContains, Value: "hello"},
			output:     "hello world",
			wantPassed: true,
		},
		{
			name:       "contains - fail",
			assertion:  Assertion{Type: AssertContains, Value: "goodbye"},
			output:     "hello world",
			wantPassed: false,
		},
		// Not Contains
		{
			name:       "not_contains - pass",
			assertion:  Assertion{Type: AssertNotContains, Value: "goodbye"},
			output:     "hello world",
			wantPassed: true,
		},
		{
			name:       "not_contains - fail",
			assertion:  Assertion{Type: AssertNotContains, Value: "hello"},
			output:     "hello world",
			wantPassed: false,
		},
		// Equals
		{
			name:       "equals - pass",
			assertion:  Assertion{Type: AssertEquals, Value: "hello world"},
			output:     "hello world",
			wantPassed: true,
		},
		{
			name:       "equals - pass with whitespace trimming",
			assertion:  Assertion{Type: AssertEquals, Value: "hello"},
			output:     "  hello  ",
			wantPassed: true,
		},
		{
			name:       "equals - fail",
			assertion:  Assertion{Type: AssertEquals, Value: "hello"},
			output:     "world",
			wantPassed: false,
		},
		// Matches (regex)
		{
			name:       "matches - pass",
			assertion:  Assertion{Type: AssertMatches, Value: `\d+`},
			output:     "test 123 test",
			wantPassed: true,
		},
		{
			name:       "matches - fail",
			assertion:  Assertion{Type: AssertMatches, Value: `^\d+$`},
			output:     "test 123 test",
			wantPassed: false,
		},
		// Starts With
		{
			name:       "starts_with - pass",
			assertion:  Assertion{Type: AssertStartsWith, Value: "Hello"},
			output:     "Hello world",
			wantPassed: true,
		},
		{
			name:       "starts_with - fail",
			assertion:  Assertion{Type: AssertStartsWith, Value: "World"},
			output:     "Hello world",
			wantPassed: false,
		},
		// Ends With
		{
			name:       "ends_with - pass",
			assertion:  Assertion{Type: AssertEndsWith, Value: "world"},
			output:     "Hello world",
			wantPassed: true,
		},
		{
			name:       "ends_with - fail",
			assertion:  Assertion{Type: AssertEndsWith, Value: "Hello"},
			output:     "Hello world",
			wantPassed: false,
		},
		// Min Length
		{
			name:       "min_length - pass",
			assertion:  Assertion{Type: AssertMinLength, Value: 5},
			output:     "hello world",
			wantPassed: true,
		},
		{
			name:       "min_length - fail",
			assertion:  Assertion{Type: AssertMinLength, Value: 100},
			output:     "hello",
			wantPassed: false,
		},
		// Max Length
		{
			name:       "max_length - pass",
			assertion:  Assertion{Type: AssertMaxLength, Value: 100},
			output:     "hello",
			wantPassed: true,
		},
		{
			name:       "max_length - fail",
			assertion:  Assertion{Type: AssertMaxLength, Value: 3},
			output:     "hello",
			wantPassed: false,
		},
		// Not Empty
		{
			name:       "not_empty - pass",
			assertion:  Assertion{Type: AssertNotEmpty},
			output:     "hello",
			wantPassed: true,
		},
		{
			name:       "not_empty - fail with empty",
			assertion:  Assertion{Type: AssertNotEmpty},
			output:     "",
			wantPassed: false,
		},
		{
			name:       "not_empty - fail with whitespace",
			assertion:  Assertion{Type: AssertNotEmpty},
			output:     "   ",
			wantPassed: false,
		},
		// JSON Valid
		{
			name:       "json_valid - pass",
			assertion:  Assertion{Type: AssertJSONValid},
			output:     `{"key": "value"}`,
			wantPassed: true,
		},
		{
			name:       "json_valid - fail",
			assertion:  Assertion{Type: AssertJSONValid},
			output:     `{invalid json}`,
			wantPassed: false,
		},
		// JSON Path
		{
			name:       "json_path - exists pass",
			assertion:  Assertion{Type: AssertJSONPath, Path: "name"},
			output:     `{"name": "test"}`,
			wantPassed: true,
		},
		{
			name:       "json_path - exists fail",
			assertion:  Assertion{Type: AssertJSONPath, Path: "missing"},
			output:     `{"name": "test"}`,
			wantPassed: false,
		},
		{
			name:       "json_path - value match pass",
			assertion:  Assertion{Type: AssertJSONPath, Path: "name", Value: "test"},
			output:     `{"name": "test"}`,
			wantPassed: true,
		},
		{
			name:       "json_path - value match fail",
			assertion:  Assertion{Type: AssertJSONPath, Path: "name", Value: "other"},
			output:     `{"name": "test"}`,
			wantPassed: false,
		},
		{
			name:       "json_path - nested path",
			assertion:  Assertion{Type: AssertJSONPath, Path: "data.items.0.id", Value: "1"},
			output:     `{"data": {"items": [{"id": "1"}]}}`,
			wantPassed: true,
		},
		// Line Count
		{
			name:       "line_count - pass",
			assertion:  Assertion{Type: AssertLineCount, Value: 3},
			output:     "line1\nline2\nline3",
			wantPassed: true,
		},
		{
			name:       "line_count - fail",
			assertion:  Assertion{Type: AssertLineCount, Value: 5},
			output:     "line1\nline2\nline3",
			wantPassed: false,
		},
		// Min Lines
		{
			name:       "min_lines - pass",
			assertion:  Assertion{Type: AssertMinLines, Value: 2},
			output:     "line1\nline2\nline3",
			wantPassed: true,
		},
		{
			name:       "min_lines - fail",
			assertion:  Assertion{Type: AssertMinLines, Value: 5},
			output:     "line1\nline2",
			wantPassed: false,
		},
		// Max Lines
		{
			name:       "max_lines - pass",
			assertion:  Assertion{Type: AssertMaxLines, Value: 5},
			output:     "line1\nline2\nline3",
			wantPassed: true,
		},
		{
			name:       "max_lines - fail",
			assertion:  Assertion{Type: AssertMaxLines, Value: 2},
			output:     "line1\nline2\nline3",
			wantPassed: false,
		},
		// Word Count
		{
			name:       "word_count - pass",
			assertion:  Assertion{Type: AssertWordCount, Value: 3},
			output:     "one two three",
			wantPassed: true,
		},
		{
			name:       "word_count - fail",
			assertion:  Assertion{Type: AssertWordCount, Value: 5},
			output:     "one two three",
			wantPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.assertion.Evaluate(tt.output)
			if result.Passed != tt.wantPassed {
				t.Errorf("expected passed=%v, got passed=%v, message: %s", tt.wantPassed, result.Passed, result.Message)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{"hello", "hello"},
		{42, "42"},
		{3.14, "3.14"},
		{float64(10), "10"},
		{true, "true"},
		{nil, ""},
	}

	for _, tt := range tests {
		result := toString(tt.input)
		if result != tt.expected {
			t.Errorf("toString(%v): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"one line", 1},
		{"line1\nline2", 2},
		{"line1\nline2\nline3", 3},
		{"line1\nline2\n", 2}, // trailing newline doesn't add line
	}

	for _, tt := range tests {
		result := countLines(tt.input)
		if result != tt.expected {
			t.Errorf("countLines(%q): expected %d, got %d", tt.input, tt.expected, result)
		}
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  multiple   spaces  ", 2},
		{"line1\nline2", 2},
	}

	for _, tt := range tests {
		result := countWords(tt.input)
		if result != tt.expected {
			t.Errorf("countWords(%q): expected %d, got %d", tt.input, tt.expected, result)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"short", 100, "short"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d): expected %q, got %q", tt.input, tt.maxLen, tt.expected, result)
		}
	}
}
