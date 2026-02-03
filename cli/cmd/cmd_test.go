package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/promptsmith/cli/internal/db"
)

// Test helper to set up a test project
func setupTestProject(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "promptsmith-cmd-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize project
	database, err := db.Initialize(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to initialize db: %v", err)
	}

	// Create project
	project, err := database.CreateProject("test-project")
	if err != nil {
		database.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create project: %v", err)
	}

	// Create prompts directory
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		database.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	// Create a test prompt file
	promptContent := `---
name: summarizer
description: Summarizes text
model_hint: gpt-4o
---

Summarize the following text in {{max_points}} bullet points:

{{text}}
`
	promptPath := filepath.Join(promptsDir, "summarizer.prompt")
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		database.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Track the prompt
	_, err = database.CreatePrompt(project.ID, "summarizer", "Summarizes text", "prompts/summarizer.prompt")
	if err != nil {
		database.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create prompt: %v", err)
	}

	database.Close()

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestBumpVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "1.0.1"},
		{"1.0.9", "1.0.10"},
		{"1.2.3", "1.2.4"},
		{"0.0.0", "0.0.1"},
		{"invalid", "1.0.0"},
		{"1.0", "1.0.0"},
	}

	for _, tt := range tests {
		result := bumpVersion(tt.input)
		if result != tt.expected {
			t.Errorf("bumpVersion(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestResolveVersion(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Change to test directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("summarizer")

	// Create some versions
	v1, _ := database.CreateVersion(prompt.ID, "1.0.0", "v1 content", "[]", "{}", "First", "user", nil)
	v2, _ := database.CreateVersion(prompt.ID, "1.0.1", "v2 content", "[]", "{}", "Second", "user", &v1.ID)
	_, _ = database.CreateVersion(prompt.ID, "1.0.2", "v3 content", "[]", "{}", "Third", "user", &v2.ID)

	versions, _ := database.ListVersions(prompt.ID)

	tests := []struct {
		ref         string
		expectedVer string
		shouldFail  bool
	}{
		{"HEAD", "1.0.2", false},
		{"HEAD~0", "1.0.2", false},
		{"HEAD~1", "1.0.1", false},
		{"HEAD~2", "1.0.0", false},
		{"HEAD~3", "", true}, // Beyond history
		{"1.0.0", "1.0.0", false},
		{"1.0.1", "1.0.1", false},
		{"1.0.2", "1.0.2", false},
		{"nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			v, err := resolveVersion(database, prompt.ID, versions, tt.ref)

			if tt.shouldFail {
				if err == nil && v != nil {
					t.Errorf("expected failure for ref %q, but got version %s", tt.ref, v.Version)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for ref %q: %v", tt.ref, err)
					return
				}
				if v == nil {
					t.Errorf("expected version for ref %q, got nil", tt.ref)
					return
				}
				if v.Version != tt.expectedVer {
					t.Errorf("resolveVersion(%q) = %s, want %s", tt.ref, v.Version, tt.expectedVer)
				}
			}
		})
	}
}

func TestComputeDiff(t *testing.T) {
	tests := []struct {
		name     string
		lines1   []string
		lines2   []string
		hasHunks bool
	}{
		{
			name:     "identical",
			lines1:   []string{"line 1", "line 2", "line 3"},
			lines2:   []string{"line 1", "line 2", "line 3"},
			hasHunks: false,
		},
		{
			name:     "added line",
			lines1:   []string{"line 1", "line 2"},
			lines2:   []string{"line 1", "line 2", "line 3"},
			hasHunks: true,
		},
		{
			name:     "removed line",
			lines1:   []string{"line 1", "line 2", "line 3"},
			lines2:   []string{"line 1", "line 3"},
			hasHunks: true,
		},
		{
			name:     "changed line",
			lines1:   []string{"line 1", "OLD", "line 3"},
			lines2:   []string{"line 1", "NEW", "line 3"},
			hasHunks: true,
		},
		{
			name:     "empty to content",
			lines1:   []string{},
			lines2:   []string{"new content"},
			hasHunks: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunks := computeDiff(tt.lines1, tt.lines2)

			if tt.hasHunks && len(hunks) == 0 {
				t.Error("expected hunks but got none")
			}
			if !tt.hasHunks && len(hunks) > 0 {
				t.Errorf("expected no hunks but got %d", len(hunks))
			}
		})
	}
}

func TestComputeDiffContent(t *testing.T) {
	lines1 := []string{"line 1", "line 2", "line 3"}
	lines2 := []string{"line 1", "modified line 2", "line 3"}

	hunks := computeDiff(lines1, lines2)

	if len(hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}

	// Check that the hunk contains the expected changes
	hunkContent := strings.Join(hunks[0].Lines, "\n")

	if !strings.Contains(hunkContent, "-line 2") {
		t.Error("expected hunk to contain removed line")
	}
	if !strings.Contains(hunkContent, "+modified line 2") {
		t.Error("expected hunk to contain added line")
	}
}

func TestResolveCheckoutRef(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("summarizer")

	// Create versions
	v1, _ := database.CreateVersion(prompt.ID, "1.0.0", "v1", "[]", "{}", "First", "user", nil)
	_, _ = database.CreateVersion(prompt.ID, "1.0.1", "v2", "[]", "{}", "Second", "user", &v1.ID)

	// Create a tag
	database.CreateTag(prompt.ID, v1.ID, "prod")

	versions, _ := database.ListVersions(prompt.ID)

	tests := []struct {
		ref         string
		expectedVer string
		shouldFail  bool
	}{
		{"HEAD", "1.0.1", false},
		{"HEAD~1", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
		{"1.0.1", "1.0.1", false},
		{"prod", "1.0.0", false}, // Tag reference
		{"nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			v, err := resolveCheckoutRef(database, prompt.ID, versions, tt.ref)

			if tt.shouldFail {
				if err == nil && v != nil {
					t.Errorf("expected failure for ref %q, got version %s", tt.ref, v.Version)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for ref %q: %v", tt.ref, err)
					return
				}
				if v == nil {
					t.Errorf("expected version for ref %q, got nil", tt.ref)
					return
				}
				if v.Version != tt.expectedVer {
					t.Errorf("resolveCheckoutRef(%q) = %s, want %s", tt.ref, v.Version, tt.expectedVer)
				}
			}
		})
	}
}

func TestResolveVersionForTag(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("summarizer")

	// Create versions
	v1, _ := database.CreateVersion(prompt.ID, "1.0.0", "v1", "[]", "{}", "First", "user", nil)
	_, _ = database.CreateVersion(prompt.ID, "1.0.1", "v2", "[]", "{}", "Second", "user", &v1.ID)

	versions, _ := database.ListVersions(prompt.ID)

	tests := []struct {
		ref         string
		expectedVer string
		shouldFail  bool
	}{
		{"HEAD", "1.0.1", false},
		{"HEAD~1", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
		{"HEAD~5", "", true}, // Beyond history
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			v, err := resolveVersionForTag(database, prompt.ID, versions, tt.ref)

			if tt.shouldFail {
				if err == nil && v != nil {
					t.Errorf("expected failure for ref %q", tt.ref)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if v == nil {
					t.Errorf("expected version, got nil")
					return
				}
				if v.Version != tt.expectedVer {
					t.Errorf("got %s, want %s", v.Version, tt.expectedVer)
				}
			}
		})
	}
}
