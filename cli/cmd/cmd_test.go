package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
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

// ============================================================================
// Remove Command Tests
// ============================================================================

func TestRemovePrompt(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	prompt, _ := database.GetPromptByName("summarizer")

	// Create versions and tags
	v1, _ := database.CreateVersion(prompt.ID, "1.0.0", "v1", "[]", "{}", "First", "user", nil)
	database.CreateVersion(prompt.ID, "1.0.1", "v2", "[]", "{}", "Second", "user", &v1.ID)
	database.CreateTag(prompt.ID, v1.ID, "prod")

	// Verify data exists
	versions, _ := database.ListVersions(prompt.ID)
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}

	tags, _ := database.ListTags(prompt.ID)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(tags))
	}

	// Remove the prompt (simulating what remove command does)
	_, err = database.Exec("DELETE FROM tags WHERE prompt_id = ?", prompt.ID)
	if err != nil {
		t.Fatalf("failed to delete tags: %v", err)
	}

	_, err = database.Exec("DELETE FROM prompt_versions WHERE prompt_id = ?", prompt.ID)
	if err != nil {
		t.Fatalf("failed to delete versions: %v", err)
	}

	_, err = database.Exec("DELETE FROM prompts WHERE id = ?", prompt.ID)
	if err != nil {
		t.Fatalf("failed to delete prompt: %v", err)
	}

	// Verify prompt is gone
	p, _ := database.GetPromptByName("summarizer")
	if p != nil {
		t.Error("prompt should be deleted")
	}

	// Verify versions are gone
	versions, _ = database.ListVersions(prompt.ID)
	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}

	// Verify tags are gone
	tags, _ = database.ListTags(prompt.ID)
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}

	// Verify the file still exists
	promptPath := filepath.Join(tmpDir, "prompts", "summarizer.prompt")
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		t.Error("prompt file should NOT be deleted")
	}

	database.Close()
}

// ============================================================================
// Config Command Edge Case Tests
// ============================================================================

func TestLoadConfigMissingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "promptsmith-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .promptsmith directory but no config file
	configDir := filepath.Join(tmpDir, db.ConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	_, err = loadConfig(tmpDir)
	if err == nil {
		t.Error("expected error when loading missing config")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "promptsmith-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .promptsmith directory with invalid YAML
	configDir := filepath.Join(tmpDir, db.ConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, db.ConfigFile)
	invalidYAML := `{invalid yaml content: [}`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	_, err = loadConfig(tmpDir)
	if err == nil {
		t.Error("expected error when loading invalid YAML config")
	}
}

func TestConfigValueEdgeCases(t *testing.T) {
	config := &Config{
		Version: 1,
		Project: ProjectConfig{
			Name: "",
			ID:   "",
		},
		PromptsDir:    "",
		TestsDir:      "",
		BenchmarksDir: "",
		Defaults: DefaultsConfig{
			Model:       "",
			Temperature: 0,
		},
	}

	// Test getting empty values
	name, err := getConfigValue(config, "project.name")
	if err != nil {
		t.Errorf("unexpected error getting empty project.name: %v", err)
	}
	if name != "" {
		t.Errorf("expected empty string, got %q", name)
	}

	// Test setting empty string values
	err = setConfigValue(config, "defaults.model", "")
	if err != nil {
		t.Errorf("unexpected error setting empty model: %v", err)
	}
	if config.Defaults.Model != "" {
		t.Errorf("expected empty model, got %q", config.Defaults.Model)
	}

	// Test temperature boundary values
	err = setConfigValue(config, "defaults.temperature", "0")
	if err != nil {
		t.Errorf("unexpected error setting temperature=0: %v", err)
	}
	if config.Defaults.Temperature != 0 {
		t.Errorf("expected temperature 0, got %f", config.Defaults.Temperature)
	}

	err = setConfigValue(config, "defaults.temperature", "2")
	if err != nil {
		t.Errorf("unexpected error setting temperature=2: %v", err)
	}
	if config.Defaults.Temperature != 2 {
		t.Errorf("expected temperature 2, got %f", config.Defaults.Temperature)
	}

	// Test negative temperature (should fail)
	err = setConfigValue(config, "defaults.temperature", "-0.1")
	if err == nil {
		t.Error("expected error for negative temperature")
	}

	// Test temperature > 2 (should fail)
	err = setConfigValue(config, "defaults.temperature", "2.1")
	if err == nil {
		t.Error("expected error for temperature > 2")
	}
}

func TestConfigPathWithSpaces(t *testing.T) {
	config := &Config{
		Version:       1,
		PromptsDir:    "./my prompts",
		TestsDir:      "./my tests",
		BenchmarksDir: "./my benchmarks",
	}

	// Test getting paths with spaces
	promptsDir, err := getConfigValue(config, "prompts_dir")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if promptsDir != "./my prompts" {
		t.Errorf("expected './my prompts', got %q", promptsDir)
	}

	// Test setting paths with spaces
	err = setConfigValue(config, "prompts_dir", "./path with/many spaces/here")
	if err != nil {
		t.Errorf("unexpected error setting path with spaces: %v", err)
	}
	if config.PromptsDir != "./path with/many spaces/here" {
		t.Errorf("expected path with spaces, got %q", config.PromptsDir)
	}
}

func TestConfigDeeplyNestedKey(t *testing.T) {
	config := &Config{
		Project: ProjectConfig{Name: "test"},
		Defaults: DefaultsConfig{Model: "gpt-4o"},
	}

	// Note: The current implementation ignores extra parts after the second level.
	// This is acceptable behavior - it just uses the first two parts.

	// Test keys with more than 2 parts - implementation extracts first two parts
	val, err := getConfigValue(config, "project.name.extra.parts")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "test" {
		t.Errorf("expected 'test', got %q", val)
	}

	// Test invalid top-level keys
	_, err = getConfigValue(config, "completely.unknown.path")
	if err == nil {
		t.Error("expected error for unknown top-level key")
	}
}

// ============================================================================
// Init Command Integration Tests
// ============================================================================

// executeCommand runs a cobra command and captures output
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestInitCommand(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "promptsmith-init-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Run init command
	err = runInit(&cobra.Command{}, []string{"test-project"})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify .promptsmith directory was created
	configDir := filepath.Join(tmpDir, db.ConfigDir)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("expected .promptsmith directory to exist")
	}

	// Verify config file was created
	configPath := filepath.Join(configDir, db.ConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config.yaml to exist")
	}

	// Verify database was created
	dbPath := filepath.Join(configDir, "promptsmith.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected promptsmith.db to exist")
	}

	// Verify directories were created
	expectedDirs := []string{"prompts", "tests", "benchmarks"}
	for _, dir := range expectedDirs {
		dirPath := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("expected %s directory to exist", dir)
		}
	}

	// Verify .gitignore was created
	gitignorePath := filepath.Join(configDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Error("expected .gitignore to exist")
	}
}

func TestInitCommandDefaultName(t *testing.T) {
	// Create temp directory with specific name
	tmpDir, err := os.MkdirTemp("", "my-awesome-project-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Run init without project name - should use directory name
	err = runInit(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Load config and verify project name
	config, err := loadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Project name should be the directory base name
	expectedPrefix := "my-awesome-project-"
	if !strings.HasPrefix(config.Project.Name, expectedPrefix) {
		t.Errorf("expected project name to start with %q, got %q", expectedPrefix, config.Project.Name)
	}
}

func TestInitCommandAlreadyInitialized(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "promptsmith-init-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Initialize first time
	err = runInit(&cobra.Command{}, []string{"test-project"})
	if err != nil {
		t.Fatalf("first runInit failed: %v", err)
	}

	// Try to initialize again - should fail
	err = runInit(&cobra.Command{}, []string{"test-project"})
	if err == nil {
		t.Error("expected error when initializing already initialized project")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("expected 'already initialized' error, got: %v", err)
	}
}

// ============================================================================
// Add Command Integration Tests
// ============================================================================

// initTestProject initializes a project and returns the temp dir and cleanup func
func initTestProject(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "promptsmith-cmd-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)

	err = runInit(&cobra.Command{}, []string{"test-project"})
	if err != nil {
		os.Chdir(originalWd)
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to initialize project: %v", err)
	}

	cleanup := func() {
		os.Chdir(originalWd)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestAddCommand(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Create a prompt file
	promptContent := `---
name: greeting
description: A greeting prompt
model_hint: gpt-4o
---

Hello {{name}}, welcome to PromptSmith!
`
	promptPath := filepath.Join(tmpDir, "prompts", "greeting.prompt")
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Run add command
	err := runAdd(&cobra.Command{}, []string{"prompts/greeting.prompt"})
	if err != nil {
		t.Fatalf("runAdd failed: %v", err)
	}

	// Verify prompt was added
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, err := database.GetPromptByName("greeting")
	if err != nil {
		t.Fatalf("failed to get prompt: %v", err)
	}
	if prompt == nil {
		t.Fatal("expected prompt to be tracked")
	}
	if prompt.Description != "A greeting prompt" {
		t.Errorf("expected description 'A greeting prompt', got %q", prompt.Description)
	}
}

func TestAddCommandNoFrontmatter(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Create a prompt file without frontmatter
	promptContent := `Hello {{name}}, welcome!`
	promptPath := filepath.Join(tmpDir, "prompts", "simple.prompt")
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Run add command
	err := runAdd(&cobra.Command{}, []string{"prompts/simple.prompt"})
	if err != nil {
		t.Fatalf("runAdd failed: %v", err)
	}

	// Verify prompt was added with filename as name
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, err := database.GetPromptByName("simple")
	if err != nil {
		t.Fatalf("failed to get prompt: %v", err)
	}
	if prompt == nil {
		t.Fatal("expected prompt to be tracked")
	}
}

func TestAddCommandFileNotFound(t *testing.T) {
	_, cleanup := initTestProject(t)
	defer cleanup()

	// Try to add a file that doesn't exist
	err := runAdd(&cobra.Command{}, []string{"prompts/nonexistent.prompt"})
	if err == nil {
		t.Error("expected error when adding non-existent file")
	}
}

func TestAddCommandAlreadyTracked(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Create a prompt file
	promptContent := `---
name: duplicate
description: Test duplicate
---
Hello!
`
	promptPath := filepath.Join(tmpDir, "prompts", "duplicate.prompt")
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Add it once
	err := runAdd(&cobra.Command{}, []string{"prompts/duplicate.prompt"})
	if err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	// Try to add again
	err = runAdd(&cobra.Command{}, []string{"prompts/duplicate.prompt"})
	if err == nil {
		t.Error("expected error when adding already tracked file")
	}
	if !strings.Contains(err.Error(), "already tracked") {
		t.Errorf("expected 'already tracked' error, got: %v", err)
	}
}

func TestAddCommandNameCollision(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Create first prompt
	promptContent1 := `---
name: collision
description: First prompt
---
Hello!
`
	promptPath1 := filepath.Join(tmpDir, "prompts", "first.prompt")
	if err := os.WriteFile(promptPath1, []byte(promptContent1), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Add first prompt
	err := runAdd(&cobra.Command{}, []string{"prompts/first.prompt"})
	if err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	// Create second prompt with same name in frontmatter
	promptContent2 := `---
name: collision
description: Second prompt with same name
---
World!
`
	promptPath2 := filepath.Join(tmpDir, "prompts", "second.prompt")
	if err := os.WriteFile(promptPath2, []byte(promptContent2), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Try to add - should fail due to name collision
	err = runAdd(&cobra.Command{}, []string{"prompts/second.prompt"})
	if err == nil {
		t.Error("expected error when adding prompt with duplicate name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

// ============================================================================
// Commit Command Integration Tests
// ============================================================================

// addTestPrompt is a helper to add a prompt to a project
func addTestPrompt(t *testing.T, tmpDir, name, content string) {
	t.Helper()
	promptPath := filepath.Join(tmpDir, "prompts", name+".prompt")
	if err := os.WriteFile(promptPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}
	err := runAdd(&cobra.Command{}, []string{"prompts/" + name + ".prompt"})
	if err != nil {
		t.Fatalf("failed to add prompt: %v", err)
	}
}

func TestCommitCommand(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add a prompt
	addTestPrompt(t, tmpDir, "greeting", `---
name: greeting
description: A greeting prompt
---
Hello {{name}}!
`)

	// Set commit message
	commitMessage = "Initial commit"

	// Run commit
	err := runCommit(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runCommit failed: %v", err)
	}

	// Verify version was created
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("greeting")
	versions, err := database.ListVersions(prompt.ID)
	if err != nil {
		t.Fatalf("failed to list versions: %v", err)
	}

	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}

	if versions[0].Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", versions[0].Version)
	}

	if versions[0].CommitMessage != "Initial commit" {
		t.Errorf("expected commit message 'Initial commit', got %s", versions[0].CommitMessage)
	}
}

func TestCommitCommandNoChanges(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add a prompt
	addTestPrompt(t, tmpDir, "nochange", `Hello!`)

	// Commit first time
	commitMessage = "First commit"
	err := runCommit(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("first commit failed: %v", err)
	}

	// Commit again without changes - should succeed but not create new version
	commitMessage = "Second commit"
	err = runCommit(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("second commit failed: %v", err)
	}

	// Verify only one version exists
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("nochange")
	versions, _ := database.ListVersions(prompt.ID)

	if len(versions) != 1 {
		t.Errorf("expected 1 version (no changes), got %d", len(versions))
	}
}

func TestCommitCommandVersionBump(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "versioned.prompt")

	// Add and commit first version
	if err := os.WriteFile(promptPath, []byte("Version 1"), 0644); err != nil {
		t.Fatalf("failed to write prompt: %v", err)
	}
	runAdd(&cobra.Command{}, []string{"prompts/versioned.prompt"})
	commitMessage = "Version 1"
	runCommit(&cobra.Command{}, []string{})

	// Modify and commit second version
	if err := os.WriteFile(promptPath, []byte("Version 2"), 0644); err != nil {
		t.Fatalf("failed to write prompt: %v", err)
	}
	commitMessage = "Version 2"
	runCommit(&cobra.Command{}, []string{})

	// Modify and commit third version
	if err := os.WriteFile(promptPath, []byte("Version 3"), 0644); err != nil {
		t.Fatalf("failed to write prompt: %v", err)
	}
	commitMessage = "Version 3"
	runCommit(&cobra.Command{}, []string{})

	// Verify versions
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("versioned")
	versions, _ := database.ListVersions(prompt.ID)

	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}

	// Versions are returned newest first
	expectedVersions := []string{"1.0.2", "1.0.1", "1.0.0"}
	for i, v := range versions {
		if v.Version != expectedVersions[i] {
			t.Errorf("expected version %s at index %d, got %s", expectedVersions[i], i, v.Version)
		}
	}
}

func TestCommitCommandNoPrompts(t *testing.T) {
	_, cleanup := initTestProject(t)
	defer cleanup()

	// Try to commit with no prompts tracked
	commitMessage = "Empty commit"
	err := runCommit(&cobra.Command{}, []string{})
	if err == nil {
		t.Error("expected error when committing with no prompts")
	}
	if !strings.Contains(err.Error(), "no prompts tracked") {
		t.Errorf("expected 'no prompts tracked' error, got: %v", err)
	}
}

func TestCommitCommandMultiplePrompts(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add multiple prompts
	addTestPrompt(t, tmpDir, "prompt1", "Content 1")
	addTestPrompt(t, tmpDir, "prompt2", "Content 2")
	addTestPrompt(t, tmpDir, "prompt3", "Content 3")

	// Commit all
	commitMessage = "Initial commit for all"
	err := runCommit(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Verify all have versions
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	for _, name := range []string{"prompt1", "prompt2", "prompt3"} {
		prompt, _ := database.GetPromptByName(name)
		versions, _ := database.ListVersions(prompt.ID)
		if len(versions) != 1 {
			t.Errorf("expected 1 version for %s, got %d", name, len(versions))
		}
	}
}

// ============================================================================
// Log Command Integration Tests
// ============================================================================

func TestLogCommand(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add and commit a prompt
	addTestPrompt(t, tmpDir, "logtest", "Content v1")
	commitMessage = "First version"
	runCommit(&cobra.Command{}, []string{})

	// Run log command - should not error
	logPrompt = ""
	logLimit = 10
	err := runLog(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runLog failed: %v", err)
	}
}

func TestLogCommandSpecificPrompt(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add and commit multiple versions
	promptPath := filepath.Join(tmpDir, "prompts", "multilog.prompt")
	if err := os.WriteFile(promptPath, []byte("V1"), 0644); err != nil {
		t.Fatalf("failed to write prompt: %v", err)
	}
	runAdd(&cobra.Command{}, []string{"prompts/multilog.prompt"})

	commitMessage = "Version 1"
	runCommit(&cobra.Command{}, []string{})

	os.WriteFile(promptPath, []byte("V2"), 0644)
	commitMessage = "Version 2"
	runCommit(&cobra.Command{}, []string{})

	os.WriteFile(promptPath, []byte("V3"), 0644)
	commitMessage = "Version 3"
	runCommit(&cobra.Command{}, []string{})

	// Run log for specific prompt
	logPrompt = "multilog"
	logLimit = 10
	err := runLog(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runLog failed: %v", err)
	}

	// Verify 3 versions exist
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("multilog")
	versions, _ := database.ListVersions(prompt.ID)
	if len(versions) != 3 {
		t.Errorf("expected 3 versions, got %d", len(versions))
	}
}

func TestLogCommandPromptNotFound(t *testing.T) {
	_, cleanup := initTestProject(t)
	defer cleanup()

	// Try to log non-existent prompt
	logPrompt = "nonexistent"
	logLimit = 10
	err := runLog(&cobra.Command{}, []string{})
	if err == nil {
		t.Error("expected error for non-existent prompt")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestLogCommandNoCommits(t *testing.T) {
	_, cleanup := initTestProject(t)
	defer cleanup()

	// Run log with no commits
	logPrompt = ""
	logLimit = 10
	err := runLog(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runLog failed: %v", err)
	}
	// Should print "No commits yet." but not error
}

func TestLogCommandLimit(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Create multiple versions
	promptPath := filepath.Join(tmpDir, "prompts", "limited.prompt")
	os.WriteFile(promptPath, []byte("V1"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/limited.prompt"})

	for i := 1; i <= 5; i++ {
		os.WriteFile(promptPath, []byte(fmt.Sprintf("V%d", i)), 0644)
		commitMessage = fmt.Sprintf("Version %d", i)
		runCommit(&cobra.Command{}, []string{})
	}

	// Test with limit
	logPrompt = "limited"
	logLimit = 2
	err := runLog(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runLog failed: %v", err)
	}
	// The log should only show 2 entries (limit applies to display, not verification)
}

// ============================================================================
// Diff Command Integration Tests
// ============================================================================

func TestDiffCommand(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Create and commit a prompt
	promptPath := filepath.Join(tmpDir, "prompts", "difftest.prompt")
	os.WriteFile(promptPath, []byte("Line 1\nLine 2\nLine 3"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/difftest.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// Modify the file
	os.WriteFile(promptPath, []byte("Line 1\nModified Line 2\nLine 3"), 0644)

	// Run diff (working vs latest)
	err := runDiff(&cobra.Command{}, []string{"difftest"})
	if err != nil {
		t.Fatalf("runDiff failed: %v", err)
	}
}

func TestDiffCommandTwoVersions(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "twover.prompt")

	// Create v1
	os.WriteFile(promptPath, []byte("Version 1 content"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/twover.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// Create v2
	os.WriteFile(promptPath, []byte("Version 2 content"), 0644)
	commitMessage = "V2"
	runCommit(&cobra.Command{}, []string{})

	// Diff between two versions
	err := runDiff(&cobra.Command{}, []string{"twover", "1.0.0", "1.0.1"})
	if err != nil {
		t.Fatalf("runDiff failed: %v", err)
	}
}

func TestDiffCommandHeadNotation(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "headtest.prompt")

	// Create multiple versions
	os.WriteFile(promptPath, []byte("V1"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/headtest.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	os.WriteFile(promptPath, []byte("V2"), 0644)
	commitMessage = "V2"
	runCommit(&cobra.Command{}, []string{})

	os.WriteFile(promptPath, []byte("V3"), 0644)
	commitMessage = "V3"
	runCommit(&cobra.Command{}, []string{})

	// Diff using HEAD notation
	err := runDiff(&cobra.Command{}, []string{"headtest", "HEAD~2", "HEAD"})
	if err != nil {
		t.Fatalf("runDiff failed: %v", err)
	}
}

func TestDiffCommandNoDifferences(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "nodiff.prompt")
	os.WriteFile(promptPath, []byte("Same content"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/nodiff.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// File unchanged, diff should show "No differences"
	err := runDiff(&cobra.Command{}, []string{"nodiff"})
	if err != nil {
		t.Fatalf("runDiff failed: %v", err)
	}
}

func TestDiffCommandPromptNotFound(t *testing.T) {
	_, cleanup := initTestProject(t)
	defer cleanup()

	err := runDiff(&cobra.Command{}, []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for non-existent prompt")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestDiffCommandVersionNotFound(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "vernotfound.prompt")
	os.WriteFile(promptPath, []byte("Content"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/vernotfound.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// Try to diff with non-existent version
	err := runDiff(&cobra.Command{}, []string{"vernotfound", "9.9.9"})
	if err == nil {
		t.Error("expected error for non-existent version")
	}
}

// ============================================================================
// Tag Command Integration Tests
// ============================================================================

func TestTagCommand(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "tagtest.prompt")
	os.WriteFile(promptPath, []byte("Content"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/tagtest.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// Create a tag
	tagList = false
	tagDelete = false
	err := runTag(&cobra.Command{}, []string{"tagtest", "prod"})
	if err != nil {
		t.Fatalf("runTag failed: %v", err)
	}

	// Verify tag was created
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("tagtest")
	tags, _ := database.ListTags(prompt.ID)
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].Name != "prod" {
		t.Errorf("expected tag 'prod', got %s", tags[0].Name)
	}
}

func TestTagCommandWithVersion(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "tagver.prompt")

	// Create multiple versions
	os.WriteFile(promptPath, []byte("V1"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/tagver.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	os.WriteFile(promptPath, []byte("V2"), 0644)
	commitMessage = "V2"
	runCommit(&cobra.Command{}, []string{})

	// Tag specific version
	tagList = false
	tagDelete = false
	err := runTag(&cobra.Command{}, []string{"tagver", "stable", "1.0.0"})
	if err != nil {
		t.Fatalf("runTag failed: %v", err)
	}

	// Verify tag points to correct version
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("tagver")
	tags, _ := database.ListTags(prompt.ID)
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}

	// Get version for tag
	version, _ := database.GetVersionByID(tags[0].VersionID)
	if version.Version != "1.0.0" {
		t.Errorf("expected tag to point to 1.0.0, got %s", version.Version)
	}
}

func TestTagCommandList(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "taglist.prompt")
	os.WriteFile(promptPath, []byte("Content"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/taglist.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// Create multiple tags
	tagList = false
	tagDelete = false
	runTag(&cobra.Command{}, []string{"taglist", "prod"})
	runTag(&cobra.Command{}, []string{"taglist", "staging"})

	// List tags
	tagList = true
	err := runTag(&cobra.Command{}, []string{"taglist"})
	if err != nil {
		t.Fatalf("runTag --list failed: %v", err)
	}
}

func TestTagCommandDelete(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "tagdel.prompt")
	os.WriteFile(promptPath, []byte("Content"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/tagdel.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// Create tag
	tagList = false
	tagDelete = false
	runTag(&cobra.Command{}, []string{"tagdel", "temp"})

	// Delete tag
	tagDelete = true
	err := runTag(&cobra.Command{}, []string{"tagdel", "temp"})
	if err != nil {
		t.Fatalf("runTag --delete failed: %v", err)
	}

	// Verify tag was deleted
	database, err := db.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	prompt, _ := database.GetPromptByName("tagdel")
	tags, _ := database.ListTags(prompt.ID)
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after deletion, got %d", len(tags))
	}
}

func TestTagCommandPromptNotFound(t *testing.T) {
	_, cleanup := initTestProject(t)
	defer cleanup()

	tagList = false
	tagDelete = false
	err := runTag(&cobra.Command{}, []string{"nonexistent", "tag"})
	if err == nil {
		t.Error("expected error for non-existent prompt")
	}
}

// ============================================================================
// Checkout Command Integration Tests
// ============================================================================

func TestCheckoutCommand(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "checkout.prompt")

	// Create multiple versions
	os.WriteFile(promptPath, []byte("Version 1 content"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/checkout.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	os.WriteFile(promptPath, []byte("Version 2 content"), 0644)
	commitMessage = "V2"
	runCommit(&cobra.Command{}, []string{})

	// Checkout first version
	err := runCheckout(&cobra.Command{}, []string{"checkout", "1.0.0"})
	if err != nil {
		t.Fatalf("runCheckout failed: %v", err)
	}

	// Verify file content was restored
	content, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "Version 1 content" {
		t.Errorf("expected 'Version 1 content', got %q", string(content))
	}
}

func TestCheckoutCommandByTag(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "checkoutag.prompt")

	// Create versions
	os.WriteFile(promptPath, []byte("Production content"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/checkoutag.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// Tag as prod
	tagList = false
	tagDelete = false
	runTag(&cobra.Command{}, []string{"checkoutag", "prod"})

	// Create another version
	os.WriteFile(promptPath, []byte("Development content"), 0644)
	commitMessage = "V2"
	runCommit(&cobra.Command{}, []string{})

	// Checkout by tag
	err := runCheckout(&cobra.Command{}, []string{"checkoutag", "prod"})
	if err != nil {
		t.Fatalf("runCheckout failed: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "Production content" {
		t.Errorf("expected 'Production content', got %q", string(content))
	}
}

func TestCheckoutCommandHeadNotation(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "checkhead.prompt")

	// Create multiple versions
	os.WriteFile(promptPath, []byte("V1"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/checkhead.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	os.WriteFile(promptPath, []byte("V2"), 0644)
	commitMessage = "V2"
	runCommit(&cobra.Command{}, []string{})

	os.WriteFile(promptPath, []byte("V3"), 0644)
	commitMessage = "V3"
	runCommit(&cobra.Command{}, []string{})

	// Checkout HEAD~2 (first version)
	err := runCheckout(&cobra.Command{}, []string{"checkhead", "HEAD~2"})
	if err != nil {
		t.Fatalf("runCheckout failed: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "V1" {
		t.Errorf("expected 'V1', got %q", string(content))
	}
}

func TestCheckoutCommandPromptNotFound(t *testing.T) {
	_, cleanup := initTestProject(t)
	defer cleanup()

	err := runCheckout(&cobra.Command{}, []string{"nonexistent", "1.0.0"})
	if err == nil {
		t.Error("expected error for non-existent prompt")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestCheckoutCommandVersionNotFound(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "checkver.prompt")
	os.WriteFile(promptPath, []byte("Content"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/checkver.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	err := runCheckout(&cobra.Command{}, []string{"checkver", "9.9.9"})
	if err == nil {
		t.Error("expected error for non-existent version")
	}
}

// ============================================================================
// Test Command Integration Tests
// ============================================================================

// createTestSuite creates a test suite YAML file for testing
func createTestSuite(t *testing.T, tmpDir, name, content string) {
	t.Helper()
	testsDir := filepath.Join(tmpDir, "tests")
	suitePath := filepath.Join(testsDir, name+".test.yaml")
	if err := os.WriteFile(suitePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test suite file: %v", err)
	}
}

func TestTestCommand(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add and commit a prompt (Go templates use .field syntax)
	addTestPrompt(t, tmpDir, "greeting", `---
name: greeting
description: A greeting prompt
---
Hello {{.name}}! Welcome to PromptSmith.
`)
	commitMessage = "Initial commit"
	runCommit(&cobra.Command{}, []string{})

	// Create a test suite
	createTestSuite(t, tmpDir, "greeting", `
name: greeting-tests
prompt: greeting
tests:
  - name: basic-test
    inputs:
      name: World
    assertions:
      - type: not_empty
      - type: contains
        value: Hello
      - type: contains
        value: World
`)

	// Reset flags
	testFilter = ""
	testVersion = ""
	testOutput = ""
	testLive = false
	testWatch = false

	// Run test command
	err := runTest(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runTest failed: %v", err)
	}
}

func TestTestCommandWithFilter(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add and commit a prompt (Go templates use .field syntax)
	addTestPrompt(t, tmpDir, "filtered", `---
name: filtered
---
Hello {{.name}}!
`)
	commitMessage = "Initial commit"
	runCommit(&cobra.Command{}, []string{})

	// Create a test suite with multiple tests
	createTestSuite(t, tmpDir, "filtered", `
name: filtered-tests
prompt: filtered
tests:
  - name: basic-hello
    inputs:
      name: Alice
    assertions:
      - type: not_empty
  - name: basic-world
    inputs:
      name: Bob
    assertions:
      - type: not_empty
  - name: advanced-check
    inputs:
      name: Charlie
    assertions:
      - type: not_empty
`)

	// Reset and set filter
	testFilter = "basic"
	testVersion = ""
	testOutput = ""
	testLive = false
	testWatch = false

	// Run test command with filter - should only run "basic" tests
	err := runTest(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runTest with filter failed: %v", err)
	}
}

func TestTestCommandWithVersion(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "versioned.prompt")

	// Create v1
	os.WriteFile(promptPath, []byte("---\nname: versioned\n---\nVersion ONE"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/versioned.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// Create v2
	os.WriteFile(promptPath, []byte("---\nname: versioned\n---\nVersion TWO"), 0644)
	commitMessage = "V2"
	runCommit(&cobra.Command{}, []string{})

	// Create test suite
	createTestSuite(t, tmpDir, "versioned", `
name: versioned-tests
prompt: versioned
tests:
  - name: version-test
    assertions:
      - type: not_empty
`)

	// Test against specific version
	testFilter = ""
	testVersion = "1.0.0"
	testOutput = ""
	testLive = false
	testWatch = false

	err := runTest(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runTest with version failed: %v", err)
	}
}

func TestTestCommandNoSuites(t *testing.T) {
	_, cleanup := initTestProject(t)
	defer cleanup()

	// Reset flags
	testFilter = ""
	testVersion = ""
	testOutput = ""
	testLive = false
	testWatch = false

	// Run test command with no suites - should not error
	err := runTest(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runTest with no suites failed: %v", err)
	}
}

func TestTestCommandWithOutput(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add and commit a prompt
	addTestPrompt(t, tmpDir, "output", `---
name: output
---
Hello!
`)
	commitMessage = "Initial commit"
	runCommit(&cobra.Command{}, []string{})

	// Create a test suite
	createTestSuite(t, tmpDir, "output", `
name: output-tests
prompt: output
tests:
  - name: output-test
    assertions:
      - type: not_empty
`)

	// Set output file
	outputPath := filepath.Join(tmpDir, "results.json")
	testFilter = ""
	testVersion = ""
	testOutput = outputPath
	testLive = false
	testWatch = false

	// Run test command
	err := runTest(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runTest with output failed: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected results.json to be created")
	}
}

func TestTestCommandPromptNotFound(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Create a test suite for non-existent prompt
	createTestSuite(t, tmpDir, "missing", `
name: missing-tests
prompt: nonexistent
tests:
  - name: test
    assertions:
      - type: not_empty
`)

	// Reset flags
	testFilter = ""
	testVersion = ""
	testOutput = ""
	testLive = false
	testWatch = false

	// Run test command - should handle missing prompt gracefully
	err := runTest(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runTest should handle missing prompt gracefully: %v", err)
	}
}

// ============================================================================
// Benchmark Command Integration Tests
// ============================================================================

// createBenchmarkSuite creates a benchmark suite YAML file for testing
func createBenchmarkSuite(t *testing.T, tmpDir, name, content string) {
	t.Helper()
	benchDir := filepath.Join(tmpDir, "benchmarks")
	suitePath := filepath.Join(benchDir, name+".bench.yaml")
	if err := os.WriteFile(suitePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write benchmark suite file: %v", err)
	}
}

func TestBenchmarkCommandNoSuites(t *testing.T) {
	_, cleanup := initTestProject(t)
	defer cleanup()

	// Reset flags
	benchModels = ""
	benchRuns = 0
	benchVersion = ""
	benchOutput = ""

	// Run benchmark command with no suites - should not error
	err := runBenchmark(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runBenchmark with no suites failed: %v", err)
	}
}

func TestBenchmarkCommandSuiteDiscovery(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add and commit a prompt
	addTestPrompt(t, tmpDir, "benchable", `---
name: benchable
---
Hello!
`)
	commitMessage = "Initial commit"
	runCommit(&cobra.Command{}, []string{})

	// Create a benchmark suite
	createBenchmarkSuite(t, tmpDir, "benchable", `
name: benchable-benchmark
prompt: benchable
models:
  - gpt-4o-mini
runs_per_model: 1
`)

	// Reset flags
	benchModels = ""
	benchRuns = 0
	benchVersion = ""
	benchOutput = ""

	// Run benchmark command
	// Note: This will fail gracefully since we don't have API keys
	err := runBenchmark(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runBenchmark failed: %v", err)
	}
}

func TestBenchmarkCommandModelOverride(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add and commit a prompt
	addTestPrompt(t, tmpDir, "override", `---
name: override
---
Hello!
`)
	commitMessage = "Initial commit"
	runCommit(&cobra.Command{}, []string{})

	// Create a benchmark suite
	createBenchmarkSuite(t, tmpDir, "override", `
name: override-benchmark
prompt: override
models:
  - gpt-4o
runs_per_model: 1
`)

	// Override models via flag
	benchModels = "gpt-4o-mini"
	benchRuns = 0
	benchVersion = ""
	benchOutput = ""

	// Run benchmark command
	err := runBenchmark(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runBenchmark with model override failed: %v", err)
	}
}

func TestBenchmarkCommandRunsOverride(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Add and commit a prompt
	addTestPrompt(t, tmpDir, "runs", `---
name: runs
---
Hello!
`)
	commitMessage = "Initial commit"
	runCommit(&cobra.Command{}, []string{})

	// Create a benchmark suite
	createBenchmarkSuite(t, tmpDir, "runs", `
name: runs-benchmark
prompt: runs
models:
  - gpt-4o-mini
runs_per_model: 5
`)

	// Override runs via flag
	benchModels = ""
	benchRuns = 2
	benchVersion = ""
	benchOutput = ""

	// Run benchmark command
	err := runBenchmark(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runBenchmark with runs override failed: %v", err)
	}
}

func TestBenchmarkCommandVersionOverride(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	promptPath := filepath.Join(tmpDir, "prompts", "version.prompt")

	// Create v1
	os.WriteFile(promptPath, []byte("---\nname: version\n---\nV1"), 0644)
	runAdd(&cobra.Command{}, []string{"prompts/version.prompt"})
	commitMessage = "V1"
	runCommit(&cobra.Command{}, []string{})

	// Create v2
	os.WriteFile(promptPath, []byte("---\nname: version\n---\nV2"), 0644)
	commitMessage = "V2"
	runCommit(&cobra.Command{}, []string{})

	// Create a benchmark suite
	createBenchmarkSuite(t, tmpDir, "version", `
name: version-benchmark
prompt: version
models:
  - gpt-4o-mini
runs_per_model: 1
`)

	// Override version via flag
	benchModels = ""
	benchRuns = 0
	benchVersion = "1.0.0"
	benchOutput = ""

	// Run benchmark command
	err := runBenchmark(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runBenchmark with version override failed: %v", err)
	}
}

func TestBenchmarkCommandPromptNotFound(t *testing.T) {
	tmpDir, cleanup := initTestProject(t)
	defer cleanup()

	// Create a benchmark suite for non-existent prompt
	createBenchmarkSuite(t, tmpDir, "missing", `
name: missing-benchmark
prompt: nonexistent
models:
  - gpt-4o-mini
runs_per_model: 1
`)

	// Reset flags
	benchModels = ""
	benchRuns = 0
	benchVersion = ""
	benchOutput = ""

	// Run benchmark command - should handle missing prompt gracefully
	err := runBenchmark(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("runBenchmark should handle missing prompt gracefully: %v", err)
	}
}
