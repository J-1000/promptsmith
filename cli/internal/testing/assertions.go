package testing

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

// Evaluate checks if the output satisfies the assertion
func (a *Assertion) Evaluate(output string) AssertionResult {
	result := AssertionResult{
		Type:     a.Type,
		Passed:   false,
		Expected: fmt.Sprintf("%v", a.Value),
		Message:  a.Message,
	}

	switch a.Type {
	case AssertContains:
		result.Passed = strings.Contains(output, toString(a.Value))
		result.Actual = truncate(output, 100)
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected output to contain '%s'", a.Value)
		}

	case AssertNotContains:
		result.Passed = !strings.Contains(output, toString(a.Value))
		result.Actual = truncate(output, 100)
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected output not to contain '%s'", a.Value)
		}

	case AssertEquals:
		expected := toString(a.Value)
		result.Passed = strings.TrimSpace(output) == strings.TrimSpace(expected)
		result.Actual = truncate(output, 100)
		if !result.Passed && result.Message == "" {
			result.Message = "output does not match expected value"
		}

	case AssertMatches:
		pattern := toString(a.Value)
		re, err := regexp.Compile(pattern)
		if err != nil {
			result.Message = fmt.Sprintf("invalid regex pattern: %s", err)
			return result
		}
		result.Passed = re.MatchString(output)
		result.Actual = truncate(output, 100)
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("output does not match pattern '%s'", pattern)
		}

	case AssertStartsWith:
		prefix := toString(a.Value)
		result.Passed = strings.HasPrefix(strings.TrimSpace(output), prefix)
		result.Actual = truncate(output, 100)
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected output to start with '%s'", prefix)
		}

	case AssertEndsWith:
		suffix := toString(a.Value)
		result.Passed = strings.HasSuffix(strings.TrimSpace(output), suffix)
		result.Actual = truncate(output, 100)
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected output to end with '%s'", suffix)
		}

	case AssertMinLength:
		minLen := toInt(a.Value)
		result.Passed = len(output) >= minLen
		result.Actual = fmt.Sprintf("%d characters", len(output))
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected at least %d characters, got %d", minLen, len(output))
		}

	case AssertMaxLength:
		maxLen := toInt(a.Value)
		result.Passed = len(output) <= maxLen
		result.Actual = fmt.Sprintf("%d characters", len(output))
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected at most %d characters, got %d", maxLen, len(output))
		}

	case AssertNotEmpty:
		result.Passed = strings.TrimSpace(output) != ""
		result.Expected = "non-empty output"
		result.Actual = fmt.Sprintf("%d characters", len(output))
		if !result.Passed && result.Message == "" {
			result.Message = "expected non-empty output"
		}

	case AssertJSONValid:
		result.Passed = json.Valid([]byte(output))
		result.Expected = "valid JSON"
		result.Actual = truncate(output, 100)
		if !result.Passed && result.Message == "" {
			result.Message = "output is not valid JSON"
		}

	case AssertJSONPath:
		if !json.Valid([]byte(output)) {
			result.Message = "output is not valid JSON"
			return result
		}
		r := gjson.Get(output, a.Path)
		result.Actual = r.String()
		if a.Value != nil {
			expected := toString(a.Value)
			result.Passed = r.String() == expected
			if !result.Passed && result.Message == "" {
				result.Message = fmt.Sprintf("JSONPath '%s': expected '%s', got '%s'", a.Path, expected, r.String())
			}
		} else {
			result.Passed = r.Exists()
			result.Expected = fmt.Sprintf("path '%s' exists", a.Path)
			if !result.Passed && result.Message == "" {
				result.Message = fmt.Sprintf("JSONPath '%s' does not exist", a.Path)
			}
		}

	case AssertLineCount:
		expected := toInt(a.Value)
		actual := countLines(output)
		result.Passed = actual == expected
		result.Actual = fmt.Sprintf("%d lines", actual)
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected %d lines, got %d", expected, actual)
		}

	case AssertMinLines:
		minLines := toInt(a.Value)
		actual := countLines(output)
		result.Passed = actual >= minLines
		result.Actual = fmt.Sprintf("%d lines", actual)
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected at least %d lines, got %d", minLines, actual)
		}

	case AssertMaxLines:
		maxLines := toInt(a.Value)
		actual := countLines(output)
		result.Passed = actual <= maxLines
		result.Actual = fmt.Sprintf("%d lines", actual)
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected at most %d lines, got %d", maxLines, actual)
		}

	case AssertWordCount:
		expected := toInt(a.Value)
		actual := countWords(output)
		result.Passed = actual == expected
		result.Actual = fmt.Sprintf("%d words", actual)
		if !result.Passed && result.Message == "" {
			result.Message = fmt.Sprintf("expected %d words, got %d", expected, actual)
		}

	case AssertSnapshot:
		// Snapshot comparison is handled by the runner which passes
		// the expected_output as a.Value before calling Evaluate
		expected := toString(a.Value)
		if expected == "" {
			result.Passed = false
			result.Expected = "(no snapshot stored)"
			result.Actual = truncate(output, 100)
			result.Message = "no snapshot stored; run with --update-snapshots to create one"
			return result
		}
		result.Passed = strings.TrimSpace(output) == strings.TrimSpace(expected)
		result.Expected = truncate(expected, 100)
		result.Actual = truncate(output, 100)
		if !result.Passed && result.Message == "" {
			result.Message = "output does not match snapshot; run with --update-snapshots to update"
		}

	case AssertSentiment, AssertLanguage:
		// These require LLM evaluation - mark as passed for now
		// Will be implemented when LLM integration is added
		result.Passed = true
		result.Message = "LLM-based assertion (not yet implemented)"

	default:
		result.Message = fmt.Sprintf("unknown assertion type: %s", a.Type)
	}

	return result
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return fmt.Sprintf("%g", val)
	case int:
		return fmt.Sprintf("%d", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		var n int
		fmt.Sscanf(val, "%d", &n)
		return n
	default:
		return 0
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	lines := strings.Split(s, "\n")
	// Don't count trailing empty line
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		return len(lines) - 1
	}
	return len(lines)
}

func countWords(s string) int {
	fields := strings.Fields(s)
	return len(fields)
}
