package db

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) (*DB, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "promptsmith-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	db, err := Initialize(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to initialize db: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, tmpDir, cleanup
}

func TestInitialize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "promptsmith-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := Initialize(tmpDir)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	defer db.Close()

	// Check that config dir was created
	configDir := filepath.Join(tmpDir, ConfigDir)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("config directory was not created")
	}

	// Check that database file was created
	dbFile := filepath.Join(configDir, DBFile)
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestCreateAndGetProject(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Create project
	project, err := db.CreateProject("test-project")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if project.Name != "test-project" {
		t.Errorf("expected name 'test-project', got '%s'", project.Name)
	}
	if project.ID == "" {
		t.Error("project ID should not be empty")
	}

	// Get project
	retrieved, err := db.GetProject()
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if retrieved.ID != project.ID {
		t.Errorf("expected ID '%s', got '%s'", project.ID, retrieved.ID)
	}
	if retrieved.Name != project.Name {
		t.Errorf("expected name '%s', got '%s'", project.Name, retrieved.Name)
	}
}

func TestCreateAndGetPrompt(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Create project first
	project, _ := db.CreateProject("test-project")

	// Create prompt
	prompt, err := db.CreatePrompt(project.ID, "summarizer", "A summarization prompt", "prompts/summarizer.prompt")
	if err != nil {
		t.Fatalf("CreatePrompt failed: %v", err)
	}

	if prompt.Name != "summarizer" {
		t.Errorf("expected name 'summarizer', got '%s'", prompt.Name)
	}

	// Get by name
	byName, err := db.GetPromptByName("summarizer")
	if err != nil {
		t.Fatalf("GetPromptByName failed: %v", err)
	}
	if byName.ID != prompt.ID {
		t.Errorf("expected ID '%s', got '%s'", prompt.ID, byName.ID)
	}

	// Get by path
	byPath, err := db.GetPromptByPath("prompts/summarizer.prompt")
	if err != nil {
		t.Fatalf("GetPromptByPath failed: %v", err)
	}
	if byPath.ID != prompt.ID {
		t.Errorf("expected ID '%s', got '%s'", prompt.ID, byPath.ID)
	}

	// Get non-existent
	notFound, err := db.GetPromptByName("nonexistent")
	if err != nil {
		t.Fatalf("GetPromptByName failed: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent prompt")
	}
}

func TestListPrompts(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")

	// Create multiple prompts
	db.CreatePrompt(project.ID, "alpha", "", "prompts/alpha.prompt")
	db.CreatePrompt(project.ID, "beta", "", "prompts/beta.prompt")
	db.CreatePrompt(project.ID, "gamma", "", "prompts/gamma.prompt")

	prompts, err := db.ListPrompts()
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	if len(prompts) != 3 {
		t.Errorf("expected 3 prompts, got %d", len(prompts))
	}

	// Should be ordered by name
	if prompts[0].Name != "alpha" {
		t.Errorf("expected first prompt 'alpha', got '%s'", prompts[0].Name)
	}
}

func TestCreateAndGetVersions(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	prompt, _ := db.CreatePrompt(project.ID, "summarizer", "", "prompts/summarizer.prompt")

	// Create first version
	v1, err := db.CreateVersion(prompt.ID, "1.0.0", "Content v1", "[]", "{}", "Initial version", "testuser", nil)
	if err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	if v1.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", v1.Version)
	}

	// Create second version
	v2, err := db.CreateVersion(prompt.ID, "1.0.1", "Content v2", "[]", "{}", "Bug fix", "testuser", &v1.ID)
	if err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	if v2.ParentVersionID == nil || *v2.ParentVersionID != v1.ID {
		t.Error("parent version ID not set correctly")
	}

	// Get latest version
	latest, err := db.GetLatestVersion(prompt.ID)
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}
	if latest.Version != "1.0.1" {
		t.Errorf("expected latest version '1.0.1', got '%s'", latest.Version)
	}

	// Get version by string
	v1Retrieved, err := db.GetVersionByString(prompt.ID, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersionByString failed: %v", err)
	}
	if v1Retrieved.ID != v1.ID {
		t.Errorf("expected ID '%s', got '%s'", v1.ID, v1Retrieved.ID)
	}

	// List versions
	versions, err := db.ListVersions(prompt.ID)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
	// Should be ordered by created_at DESC
	if versions[0].Version != "1.0.1" {
		t.Errorf("expected first version '1.0.1', got '%s'", versions[0].Version)
	}
}

func TestTags(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	prompt, _ := db.CreatePrompt(project.ID, "summarizer", "", "prompts/summarizer.prompt")
	v1, _ := db.CreateVersion(prompt.ID, "1.0.0", "Content v1", "[]", "{}", "Initial", "testuser", nil)
	v2, _ := db.CreateVersion(prompt.ID, "1.0.1", "Content v2", "[]", "{}", "Update", "testuser", &v1.ID)

	// Create tag
	tag, err := db.CreateTag(prompt.ID, v1.ID, "prod")
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}
	if tag.Name != "prod" {
		t.Errorf("expected tag name 'prod', got '%s'", tag.Name)
	}

	// Get tag by name
	retrieved, err := db.GetTagByName(prompt.ID, "prod")
	if err != nil {
		t.Fatalf("GetTagByName failed: %v", err)
	}
	if retrieved.VersionID != v1.ID {
		t.Errorf("expected version ID '%s', got '%s'", v1.ID, retrieved.VersionID)
	}

	// Update existing tag (move to new version)
	updated, err := db.CreateTag(prompt.ID, v2.ID, "prod")
	if err != nil {
		t.Fatalf("CreateTag (update) failed: %v", err)
	}
	if updated.VersionID != v2.ID {
		t.Errorf("expected updated version ID '%s', got '%s'", v2.ID, updated.VersionID)
	}

	// Create another tag
	db.CreateTag(prompt.ID, v1.ID, "staging")

	// List tags
	tags, err := db.ListTags(prompt.ID)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	// Delete tag
	err = db.DeleteTag(prompt.ID, "staging")
	if err != nil {
		t.Fatalf("DeleteTag failed: %v", err)
	}

	tags, _ = db.ListTags(prompt.ID)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag after delete, got %d", len(tags))
	}

	// Delete non-existent tag
	err = db.DeleteTag(prompt.ID, "nonexistent")
	if err == nil {
		t.Error("expected error when deleting non-existent tag")
	}
}

func TestDeletePrompt(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	prompt, _ := db.CreatePrompt(project.ID, "to-delete", "Will be deleted", "prompts/delete.prompt")
	v1, _ := db.CreateVersion(prompt.ID, "1.0.0", "Content v1", "[]", "{}", "Initial", "user", nil)
	db.CreateVersion(prompt.ID, "1.0.1", "Content v2", "[]", "{}", "Update", "user", &v1.ID)
	db.CreateTag(prompt.ID, v1.ID, "prod")

	// Delete prompt (should cascade)
	err := db.DeletePrompt(prompt.ID)
	if err != nil {
		t.Fatalf("DeletePrompt failed: %v", err)
	}

	// Verify prompt is gone
	found, err := db.GetPromptByName("to-delete")
	if err != nil {
		t.Fatalf("GetPromptByName failed: %v", err)
	}
	if found != nil {
		t.Error("expected prompt to be deleted")
	}

	// Verify versions are gone
	versions, err := db.ListVersions(prompt.ID)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}

	// Verify tags are gone
	tags, err := db.ListTags(prompt.ID)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}

	// Delete non-existent prompt
	err = db.DeletePrompt("nonexistent-id")
	if err == nil {
		t.Error("expected error when deleting non-existent prompt")
	}
}

func TestFindProjectRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "promptsmith-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve any symlinks (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// Initialize promptsmith in tmpDir
	_, err = Initialize(tmpDir)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Change to nested directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	os.Chdir(nestedDir)

	// Find project root should find parent
	root, err := FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}

	// Resolve symlinks for comparison
	root, _ = filepath.EvalSymlinks(root)

	if root != tmpDir {
		t.Errorf("expected root '%s', got '%s'", tmpDir, root)
	}
}

func TestSaveAndListTestRuns(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	prompt, _ := db.CreatePrompt(project.ID, "summarizer", "", "prompts/summarizer.prompt")
	v, _ := db.CreateVersion(prompt.ID, "1.0.0", "Content", "[]", "{}", "Init", "user", nil)

	// Save test run
	run, err := db.SaveTestRun("suite-1", v.ID, "passed", `{"passed": 3, "failed": 0}`)
	if err != nil {
		t.Fatalf("SaveTestRun failed: %v", err)
	}
	if run.ID == "" {
		t.Error("run ID should not be empty")
	}
	if run.Status != "passed" {
		t.Errorf("expected status 'passed', got '%s'", run.Status)
	}

	// Save another run
	db.SaveTestRun("suite-1", v.ID, "failed", `{"passed": 2, "failed": 1}`)

	// List test runs
	runs, err := db.ListTestRuns("suite-1")
	if err != nil {
		t.Fatalf("ListTestRuns failed: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}

	// Get single test run
	retrieved, err := db.GetTestRun(run.ID)
	if err != nil {
		t.Fatalf("GetTestRun failed: %v", err)
	}
	if retrieved.ID != run.ID {
		t.Errorf("expected ID '%s', got '%s'", run.ID, retrieved.ID)
	}

	// Get non-existent run
	notFound, err := db.GetTestRun("nonexistent")
	if err != nil {
		t.Fatalf("GetTestRun failed: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent run")
	}
}

func TestSaveAndListBenchmarkRuns(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	prompt, _ := db.CreatePrompt(project.ID, "summarizer", "", "prompts/summarizer.prompt")
	v, _ := db.CreateVersion(prompt.ID, "1.0.0", "Content", "[]", "{}", "Init", "user", nil)

	// Save benchmark run
	run, err := db.SaveBenchmarkRun("bench-1", v.ID, `{"models": []}`)
	if err != nil {
		t.Fatalf("SaveBenchmarkRun failed: %v", err)
	}
	if run.ID == "" {
		t.Error("run ID should not be empty")
	}

	// Save another run
	db.SaveBenchmarkRun("bench-1", v.ID, `{"models": [{"model": "gpt-4o"}]}`)

	// List benchmark runs
	runs, err := db.ListBenchmarkRuns("bench-1")
	if err != nil {
		t.Fatalf("ListBenchmarkRuns failed: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}

	// Verify ordering (newest first)
	if runs[0].CreatedAt.Before(runs[1].CreatedAt) {
		t.Error("expected runs ordered newest first")
	}
}

func TestGetVersionByID(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	prompt, _ := db.CreatePrompt(project.ID, "test", "", "test.prompt")
	v, _ := db.CreateVersion(prompt.ID, "1.0.0", "Content", "[]", "{}", "Test", "user", nil)

	// Get by ID
	retrieved, err := db.GetVersionByID(v.ID)
	if err != nil {
		t.Fatalf("GetVersionByID failed: %v", err)
	}
	if retrieved.ID != v.ID {
		t.Errorf("expected ID '%s', got '%s'", v.ID, retrieved.ID)
	}

	// Get non-existent
	notFound, err := db.GetVersionByID("nonexistent-id")
	if err != nil {
		t.Fatalf("GetVersionByID failed: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent version")
	}
}
