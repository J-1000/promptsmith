package testing

import (
	"testing"
)

func TestParseSuite(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid suite",
			yaml: `
name: test-suite
prompt: summarizer
tests:
  - name: basic-test
    inputs:
      article: "Test article"
    assertions:
      - type: contains
        value: "summary"
`,
			wantErr: false,
		},
		{
			name: "missing name",
			yaml: `
prompt: summarizer
tests:
  - name: test
    assertions:
      - type: not_empty
`,
			wantErr: true,
			errMsg:  "test suite requires a name",
		},
		{
			name: "missing prompt",
			yaml: `
name: test-suite
tests:
  - name: test
    assertions:
      - type: not_empty
`,
			wantErr: true,
			errMsg:  "test suite requires a prompt name",
		},
		{
			name: "no tests",
			yaml: `
name: test-suite
prompt: summarizer
tests: []
`,
			wantErr: true,
			errMsg:  "test suite requires at least one test",
		},
		{
			name: "test without name",
			yaml: `
name: test-suite
prompt: summarizer
tests:
  - assertions:
      - type: not_empty
`,
			wantErr: true,
			errMsg:  "test 1 requires a name",
		},
		{
			name: "test without assertions",
			yaml: `
name: test-suite
prompt: summarizer
tests:
  - name: empty-test
`,
			wantErr: true,
			errMsg:  "test 'empty-test' requires at least one assertion",
		},
		{
			name: "skipped test without assertions is ok",
			yaml: `
name: test-suite
prompt: summarizer
tests:
  - name: skipped-test
    skip: true
`,
			wantErr: false,
		},
		{
			name: "invalid assertion type",
			yaml: `
name: test-suite
prompt: summarizer
tests:
  - name: test
    assertions:
      - type: invalid_type
`,
			wantErr: true,
			errMsg:  "test 'test' assertion 1: unknown assertion type: invalid_type",
		},
		{
			name: "contains without value",
			yaml: `
name: test-suite
prompt: summarizer
tests:
  - name: test
    assertions:
      - type: contains
`,
			wantErr: true,
			errMsg:  "test 'test' assertion 1: contains requires a value",
		},
		{
			name: "json_path without path",
			yaml: `
name: test-suite
prompt: summarizer
tests:
  - name: test
    assertions:
      - type: json_path
`,
			wantErr: true,
			errMsg:  "test 'test' assertion 1: json_path requires a path",
		},
		{
			name: "full suite with multiple tests",
			yaml: `
name: summarizer-tests
prompt: summarizer
description: Tests for the summarizer prompt
version: "1.0.0"
tests:
  - name: basic-summary
    inputs:
      article: "Long article text here"
      max_points: 3
    assertions:
      - type: not_empty
      - type: max_length
        value: 500
      - type: min_lines
        value: 3
  - name: json-output
    inputs:
      article: "Another article"
    assertions:
      - type: json_valid
      - type: json_path
        path: "summary"
        value: "exists"
    tags:
      - json
      - format
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite, err := ParseSuite([]byte(tt.yaml))
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if suite == nil {
					t.Error("expected suite, got nil")
				}
			}
		})
	}
}

func TestParseSuiteFields(t *testing.T) {
	yaml := `
name: test-suite
prompt: my-prompt
description: A test suite
version: "2.0.0"
tests:
  - name: test-one
    inputs:
      key: value
      num: 42
    assertions:
      - type: contains
        value: "expected"
        message: "custom message"
    tags:
      - tag1
      - tag2
`

	suite, err := ParseSuite([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suite.Name != "test-suite" {
		t.Errorf("expected name 'test-suite', got %q", suite.Name)
	}
	if suite.Prompt != "my-prompt" {
		t.Errorf("expected prompt 'my-prompt', got %q", suite.Prompt)
	}
	if suite.Description != "A test suite" {
		t.Errorf("expected description 'A test suite', got %q", suite.Description)
	}
	if suite.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %q", suite.Version)
	}
	if len(suite.Tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(suite.Tests))
	}

	tc := suite.Tests[0]
	if tc.Name != "test-one" {
		t.Errorf("expected test name 'test-one', got %q", tc.Name)
	}
	if tc.Inputs["key"] != "value" {
		t.Errorf("expected input key='value', got %v", tc.Inputs["key"])
	}
	if len(tc.Assertions) != 1 {
		t.Fatalf("expected 1 assertion, got %d", len(tc.Assertions))
	}
	if tc.Assertions[0].Type != AssertContains {
		t.Errorf("expected assertion type 'contains', got %q", tc.Assertions[0].Type)
	}
	if tc.Assertions[0].Message != "custom message" {
		t.Errorf("expected message 'custom message', got %q", tc.Assertions[0].Message)
	}
	if len(tc.Tags) != 2 || tc.Tags[0] != "tag1" || tc.Tags[1] != "tag2" {
		t.Errorf("expected tags [tag1, tag2], got %v", tc.Tags)
	}
}
