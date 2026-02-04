package testing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/promptsmith/cli/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "runner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	database, err := db.Initialize(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to initialize db: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

func TestNewRunner(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// With nil executor
	runner := NewRunner(database, nil)
	if runner == nil {
		t.Fatal("expected runner, got nil")
	}
	if runner.executor == nil {
		t.Fatal("expected default executor, got nil")
	}

	// With custom executor
	mockExec := NewMockExecutor(map[string]string{})
	runner = NewRunner(database, mockExec)
	if runner.executor != mockExec {
		t.Fatal("expected custom executor")
	}
}

func TestRunnerRun(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create project and prompt
	project, err := database.CreateProject("test-project")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	prompt, err := database.CreatePrompt(project.ID, "greeting", "A greeting prompt", "prompts/greeting.prompt")
	if err != nil {
		t.Fatalf("failed to create prompt: %v", err)
	}

	// Create a version
	_, err = database.CreateVersion(
		prompt.ID,
		"1.0.0",
		"Hello {{.name}}!",
		"[]",
		"{}",
		"Initial version",
		"test",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create version: %v", err)
	}

	runner := NewRunner(database, nil)

	t.Run("prompt not found", func(t *testing.T) {
		suite := &TestSuite{
			Name:   "test-suite",
			Prompt: "nonexistent",
			Tests: []TestCase{
				{Name: "test1", Assertions: []Assertion{{Type: AssertNotEmpty}}},
			},
		}
		_, err := runner.Run(suite)
		if err == nil {
			t.Fatal("expected error for nonexistent prompt")
		}
	})

	t.Run("version not found", func(t *testing.T) {
		suite := &TestSuite{
			Name:    "test-suite",
			Prompt:  "greeting",
			Version: "9.9.9",
			Tests: []TestCase{
				{Name: "test1", Assertions: []Assertion{{Type: AssertNotEmpty}}},
			},
		}
		_, err := runner.Run(suite)
		if err == nil {
			t.Fatal("expected error for nonexistent version")
		}
	})

	t.Run("successful run with passing tests", func(t *testing.T) {
		suite := &TestSuite{
			Name:   "test-suite",
			Prompt: "greeting",
			Tests: []TestCase{
				{
					Name:   "basic-test",
					Inputs: map[string]any{"name": "World"},
					Assertions: []Assertion{
						{Type: AssertContains, Value: "Hello"},
						{Type: AssertContains, Value: "World"},
						{Type: AssertNotEmpty},
					},
				},
			},
		}

		result, err := runner.Run(suite)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Passed != 1 {
			t.Errorf("expected 1 passed, got %d", result.Passed)
		}
		if result.Failed != 0 {
			t.Errorf("expected 0 failed, got %d", result.Failed)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 total, got %d", result.Total)
		}
		if result.Version != "1.0.0" {
			t.Errorf("expected version 1.0.0, got %s", result.Version)
		}
	})

	t.Run("run with failing tests", func(t *testing.T) {
		suite := &TestSuite{
			Name:   "test-suite",
			Prompt: "greeting",
			Tests: []TestCase{
				{
					Name:   "failing-test",
					Inputs: map[string]any{"name": "World"},
					Assertions: []Assertion{
						{Type: AssertContains, Value: "Goodbye"}, // Will fail
					},
				},
			},
		}

		result, err := runner.Run(suite)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Passed != 0 {
			t.Errorf("expected 0 passed, got %d", result.Passed)
		}
		if result.Failed != 1 {
			t.Errorf("expected 1 failed, got %d", result.Failed)
		}
		if len(result.Results[0].Failures) == 0 {
			t.Error("expected failures to be recorded")
		}
	})

	t.Run("run with skipped tests", func(t *testing.T) {
		suite := &TestSuite{
			Name:   "test-suite",
			Prompt: "greeting",
			Tests: []TestCase{
				{
					Name: "skipped-test",
					Skip: true,
				},
			},
		}

		result, err := runner.Run(suite)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Skipped != 1 {
			t.Errorf("expected 1 skipped, got %d", result.Skipped)
		}
		if !result.Results[0].Skipped {
			t.Error("expected test to be marked as skipped")
		}
	})

	t.Run("run with specific version", func(t *testing.T) {
		// Create another version
		_, err = database.CreateVersion(
			prompt.ID,
			"1.0.1",
			"Hi {{.name}}!",
			"[]",
			"{}",
			"Updated greeting",
			"test",
			nil,
		)
		if err != nil {
			t.Fatalf("failed to create version: %v", err)
		}

		suite := &TestSuite{
			Name:    "test-suite",
			Prompt:  "greeting",
			Version: "1.0.0",
			Tests: []TestCase{
				{
					Name:   "version-test",
					Inputs: map[string]any{"name": "World"},
					Assertions: []Assertion{
						{Type: AssertContains, Value: "Hello"}, // Only in 1.0.0
					},
				},
			},
		}

		result, err := runner.Run(suite)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Version != "1.0.0" {
			t.Errorf("expected version 1.0.0, got %s", result.Version)
		}
		if result.Passed != 1 {
			t.Errorf("expected test to pass with version 1.0.0")
		}
	})

	t.Run("multiple tests mixed results", func(t *testing.T) {
		suite := &TestSuite{
			Name:   "test-suite",
			Prompt: "greeting",
			Tests: []TestCase{
				{
					Name:       "pass",
					Inputs:     map[string]any{"name": "World"},
					Assertions: []Assertion{{Type: AssertNotEmpty}},
				},
				{
					Name:       "fail",
					Inputs:     map[string]any{"name": "World"},
					Assertions: []Assertion{{Type: AssertContains, Value: "NOTFOUND"}},
				},
				{
					Name: "skip",
					Skip: true,
				},
			},
		}

		result, err := runner.Run(suite)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Passed != 1 {
			t.Errorf("expected 1 passed, got %d", result.Passed)
		}
		if result.Failed != 1 {
			t.Errorf("expected 1 failed, got %d", result.Failed)
		}
		if result.Skipped != 1 {
			t.Errorf("expected 1 skipped, got %d", result.Skipped)
		}
		if result.Total != 3 {
			t.Errorf("expected 3 total, got %d", result.Total)
		}
	})
}

func TestRenderPrompt(t *testing.T) {
	tests := []struct {
		name     string
		template string
		inputs   map[string]any
		expected string
		wantErr  bool
	}{
		{
			name:     "simple variable",
			template: "Hello {{.name}}!",
			inputs:   map[string]any{"name": "World"},
			expected: "Hello World!",
		},
		{
			name:     "multiple variables",
			template: "{{.greeting}} {{.name}}!",
			inputs:   map[string]any{"greeting": "Hi", "name": "Alice"},
			expected: "Hi Alice!",
		},
		{
			name:     "no variables",
			template: "Static text",
			inputs:   map[string]any{},
			expected: "Static text",
		},
		{
			name:     "missing variable",
			template: "Hello {{.name}}!",
			inputs:   map[string]any{},
			expected: "Hello <no value>!",
		},
		{
			name:     "invalid template",
			template: "Hello {{.name",
			inputs:   map[string]any{"name": "World"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderPrompt(tt.template, tt.inputs)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMockExecutor(t *testing.T) {
	exec := NewMockExecutor(map[string]string{"test": "output"})

	result, err := exec.Execute("prompt text", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// MockExecutor returns the rendered prompt as output
	if result != "prompt text" {
		t.Errorf("expected 'prompt text', got %q", result)
	}
}

func TestRunnerNoVersions(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create project and prompt without any versions
	project, _ := database.CreateProject("test-project")
	database.CreatePrompt(project.ID, "empty", "Empty prompt", "prompts/empty.prompt")

	runner := NewRunner(database, nil)
	suite := &TestSuite{
		Name:   "test-suite",
		Prompt: "empty",
		Tests: []TestCase{
			{Name: "test1", Assertions: []Assertion{{Type: AssertNotEmpty}}},
		},
	}

	_, err := runner.Run(suite)
	if err == nil {
		t.Fatal("expected error for prompt with no versions")
	}
}

func TestRunnerTemplateError(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := database.CreateProject("test-project")
	prompt, _ := database.CreatePrompt(project.ID, "bad", "Bad template", "prompts/bad.prompt")

	// Create version with invalid Go template
	database.CreateVersion(
		prompt.ID,
		"1.0.0",
		"Hello {{.name", // Unclosed template
		"[]",
		"{}",
		"Bad template",
		"test",
		nil,
	)

	runner := NewRunner(database, nil)
	suite := &TestSuite{
		Name:   "test-suite",
		Prompt: "bad",
		Tests: []TestCase{
			{
				Name:       "test1",
				Inputs:     map[string]any{"name": "World"},
				Assertions: []Assertion{{Type: AssertNotEmpty}},
			},
		},
	}

	result, err := runner.Run(suite)
	if err != nil {
		t.Fatalf("Run should not error, but got: %v", err)
	}

	// The test should have an error recorded
	if result.Results[0].Error == "" {
		t.Error("expected error in test result for bad template")
	}
}

// Ensure temp dir path doesn't depend on working directory
func init() {
	// Get absolute path for temp directory
	tmpBase := os.TempDir()
	if !filepath.IsAbs(tmpBase) {
		abs, err := filepath.Abs(tmpBase)
		if err == nil {
			os.Setenv("TMPDIR", abs)
		}
	}
}
