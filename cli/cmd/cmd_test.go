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

// ============================================================================
// Config Command Tests
// ============================================================================

func setupTestProjectWithConfig(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, cleanup := setupTestProject(t)

	// Create config file
	configContent := `version: 1
project:
  name: test-project
  id: test-id-123
prompts_dir: ./prompts
tests_dir: ./tests
benchmarks_dir: ./benchmarks
defaults:
  model: gpt-4o
  temperature: 0.7
`
	configPath := filepath.Join(tmpDir, db.ConfigDir, db.ConfigFile)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		cleanup()
		t.Fatalf("failed to write config: %v", err)
	}

	return tmpDir, cleanup
}

func TestGetConfigValue(t *testing.T) {
	config := &Config{
		Version: 1,
		Project: ProjectConfig{
			Name: "my-project",
			ID:   "proj-123",
		},
		PromptsDir:    "./prompts",
		TestsDir:      "./tests",
		BenchmarksDir: "./benchmarks",
		Defaults: DefaultsConfig{
			Model:       "gpt-4o",
			Temperature: 0.7,
		},
	}

	tests := []struct {
		key        string
		expected   string
		shouldFail bool
	}{
		{"version", "1", false},
		{"project.name", "my-project", false},
		{"project.id", "proj-123", false},
		{"prompts_dir", "./prompts", false},
		{"tests_dir", "./tests", false},
		{"benchmarks_dir", "./benchmarks", false},
		{"defaults.model", "gpt-4o", false},
		{"defaults.temperature", "0.7", false},
		{"unknown", "", true},
		{"project", "", true},  // Missing subkey
		{"defaults", "", true}, // Missing subkey
		{"project.unknown", "", true},
		{"defaults.unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value, err := getConfigValue(config, tt.key)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("expected error for key %q, got value %q", tt.key, value)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for key %q: %v", tt.key, err)
					return
				}
				if value != tt.expected {
					t.Errorf("getConfigValue(%q) = %q, want %q", tt.key, value, tt.expected)
				}
			}
		})
	}
}

func TestSetConfigValue(t *testing.T) {
	tests := []struct {
		key        string
		value      string
		shouldFail bool
		validate   func(*Config) bool
	}{
		{
			key:        "project.name",
			value:      "new-name",
			shouldFail: false,
			validate:   func(c *Config) bool { return c.Project.Name == "new-name" },
		},
		{
			key:        "prompts_dir",
			value:      "./new-prompts",
			shouldFail: false,
			validate:   func(c *Config) bool { return c.PromptsDir == "./new-prompts" },
		},
		{
			key:        "tests_dir",
			value:      "./new-tests",
			shouldFail: false,
			validate:   func(c *Config) bool { return c.TestsDir == "./new-tests" },
		},
		{
			key:        "benchmarks_dir",
			value:      "./new-benchmarks",
			shouldFail: false,
			validate:   func(c *Config) bool { return c.BenchmarksDir == "./new-benchmarks" },
		},
		{
			key:        "defaults.model",
			value:      "claude-sonnet",
			shouldFail: false,
			validate:   func(c *Config) bool { return c.Defaults.Model == "claude-sonnet" },
		},
		{
			key:        "defaults.temperature",
			value:      "0.5",
			shouldFail: false,
			validate:   func(c *Config) bool { return c.Defaults.Temperature == 0.5 },
		},
		{
			key:        "defaults.temperature",
			value:      "invalid",
			shouldFail: true,
		},
		{
			key:        "defaults.temperature",
			value:      "3.0", // Out of range
			shouldFail: true,
		},
		{
			key:        "version",
			value:      "2",
			shouldFail: true, // Read-only
		},
		{
			key:        "project.id",
			value:      "new-id",
			shouldFail: true, // Cannot set
		},
		{
			key:        "unknown",
			value:      "value",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			config := &Config{
				Version: 1,
				Project: ProjectConfig{Name: "original", ID: "id"},
				Defaults: DefaultsConfig{
					Model:       "gpt-4o",
					Temperature: 0.7,
				},
			}

			err := setConfigValue(config, tt.key, tt.value)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("expected error for key %q", tt.key)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if tt.validate != nil && !tt.validate(config) {
					t.Errorf("validation failed for key %q", tt.key)
				}
			}
		})
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tmpDir, cleanup := setupTestProjectWithConfig(t)
	defer cleanup()

	// Test load
	config, err := loadConfig(tmpDir)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	if config.Project.Name != "test-project" {
		t.Errorf("project.name = %q, want %q", config.Project.Name, "test-project")
	}
	if config.Defaults.Model != "gpt-4o" {
		t.Errorf("defaults.model = %q, want %q", config.Defaults.Model, "gpt-4o")
	}

	// Modify and save
	config.Project.Name = "modified-project"
	config.Defaults.Temperature = 0.9

	if err := saveConfig(tmpDir, config); err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	// Reload and verify
	reloaded, err := loadConfig(tmpDir)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if reloaded.Project.Name != "modified-project" {
		t.Errorf("reloaded project.name = %q, want %q", reloaded.Project.Name, "modified-project")
	}
	if reloaded.Defaults.Temperature != 0.9 {
		t.Errorf("reloaded temperature = %f, want %f", reloaded.Defaults.Temperature, 0.9)
	}
}

// ============================================================================
// Status Command Tests
// ============================================================================

func TestHashContent(t *testing.T) {
	tests := []struct {
		content1 string
		content2 string
		sameHash bool
	}{
		{"hello", "hello", true},
		{"hello", "world", false},
		{"", "", true},
		{"test\nwith\nnewlines", "test\nwith\nnewlines", true},
		{"test\nwith\nnewlines", "test\nwith\ndifferent", false},
	}

	for i, tt := range tests {
		hash1 := hashContent(tt.content1)
		hash2 := hashContent(tt.content2)

		if tt.sameHash && hash1 != hash2 {
			t.Errorf("test %d: expected same hash for %q and %q", i, tt.content1, tt.content2)
		}
		if !tt.sameHash && hash1 == hash2 {
			t.Errorf("test %d: expected different hash for %q and %q", i, tt.content1, tt.content2)
		}
	}

	// Verify hash is 64 chars (SHA256 hex)
	hash := hashContent("test")
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}
}

func TestStatusDetection(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	prompt, _ := database.GetPromptByName("summarizer")

	// Read the original content
	promptPath := filepath.Join(tmpDir, "prompts", "summarizer.prompt")
	originalContent, _ := os.ReadFile(promptPath)

	// Create a version with the original content
	_, err = database.CreateVersion(prompt.ID, "1.0.0", string(originalContent), "[]", "{}", "Initial", "user", nil)
	if err != nil {
		t.Fatalf("failed to create version: %v", err)
	}

	database.Close()

	// Test 1: File unchanged - should be clean
	database, _ = db.Open(tmpDir)
	latestVersion, _ := database.GetLatestVersion(prompt.ID)
	currentContent, _ := os.ReadFile(promptPath)

	currentHash := hashContent(string(currentContent))
	storedHash := hashContent(latestVersion.Content)

	if currentHash != storedHash {
		t.Error("unchanged file should have same hash")
	}

	// Test 2: Modify file - should be modified
	modifiedContent := string(originalContent) + "\n# Modified"
	os.WriteFile(promptPath, []byte(modifiedContent), 0644)

	newContent, _ := os.ReadFile(promptPath)
	newHash := hashContent(string(newContent))

	if newHash == storedHash {
		t.Error("modified file should have different hash")
	}

	database.Close()
}

// ============================================================================
// List Command Tests
// ============================================================================

func TestListPrompts(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	// Get the prompt we created in setup
	prompts, err := database.ListPrompts()
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	if len(prompts) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(prompts))
	}

	if prompts[0].Name != "summarizer" {
		t.Errorf("prompt name = %q, want %q", prompts[0].Name, "summarizer")
	}

	// Add another prompt
	project, _ := database.GetProject()
	_, err = database.CreatePrompt(project.ID, "translator", "Translates text", "prompts/translator.prompt")
	if err != nil {
		t.Fatalf("CreatePrompt failed: %v", err)
	}

	prompts, _ = database.ListPrompts()
	if len(prompts) != 2 {
		t.Errorf("expected 2 prompts, got %d", len(prompts))
	}

	database.Close()
}

// ============================================================================
// Show Command Tests
// ============================================================================

func TestShowPromptDetails(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("summarizer")

	// Create a version
	promptPath := filepath.Join(tmpDir, "prompts", "summarizer.prompt")
	content, _ := os.ReadFile(promptPath)
	v1, _ := database.CreateVersion(prompt.ID, "1.0.0", string(content), "[]", "{}", "Initial", "testuser", nil)

	// Create a tag
	database.CreateTag(prompt.ID, v1.ID, "prod")

	// Verify we can get the prompt
	p, err := database.GetPromptByName("summarizer")
	if err != nil || p == nil {
		t.Fatal("failed to get prompt by name")
	}

	// Verify we can get the latest version
	latestVersion, err := database.GetLatestVersion(p.ID)
	if err != nil || latestVersion == nil {
		t.Fatal("failed to get latest version")
	}

	if latestVersion.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", latestVersion.Version, "1.0.0")
	}

	if latestVersion.CreatedBy != "testuser" {
		t.Errorf("created_by = %q, want %q", latestVersion.CreatedBy, "testuser")
	}

	// Verify tags
	tags, _ := database.ListTags(p.ID)
	if len(tags) != 1 || tags[0].Name != "prod" {
		t.Error("expected 'prod' tag")
	}
}
