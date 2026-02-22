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

func TestListPromptsWithLatestVersion(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	alpha, _ := db.CreatePrompt(project.ID, "alpha", "", "prompts/alpha.prompt")
	beta, _ := db.CreatePrompt(project.ID, "beta", "", "prompts/beta.prompt")

	v1, _ := db.CreateVersion(alpha.ID, "1.0.0", "alpha v1", "[]", "{}", "Initial", "user", nil)
	db.CreateVersion(alpha.ID, "1.0.1", "alpha v2", "[]", "{}", "Update", "user", &v1.ID)
	db.CreateVersion(beta.ID, "2.0.0", "beta v1", "[]", "{}", "Initial", "user", nil)

	prompts, err := db.ListPromptsWithLatestVersion()
	if err != nil {
		t.Fatalf("ListPromptsWithLatestVersion failed: %v", err)
	}
	if len(prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(prompts))
	}

	if prompts[0].Name != "alpha" || prompts[0].LatestVersion != "1.0.1" {
		t.Fatalf("alpha latest = %q, want %q", prompts[0].LatestVersion, "1.0.1")
	}
	if prompts[1].Name != "beta" || prompts[1].LatestVersion != "2.0.0" {
		t.Fatalf("beta latest = %q, want %q", prompts[1].LatestVersion, "2.0.0")
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

func TestEnsureTestSuiteAndBenchmark(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	prompt, _ := db.CreatePrompt(project.ID, "summarizer", "", "prompts/summarizer.prompt")

	if err := db.EnsureTestSuite("suite-1", prompt.ID, "suite-1", "{}"); err != nil {
		t.Fatalf("EnsureTestSuite failed: %v", err)
	}
	if err := db.EnsureTestSuite("suite-1", prompt.ID, "suite-1-renamed", `{"x":1}`); err != nil {
		t.Fatalf("EnsureTestSuite update failed: %v", err)
	}

	var suiteName, suiteCfg string
	if err := db.QueryRow("SELECT name, config FROM test_suites WHERE id = ?", "suite-1").Scan(&suiteName, &suiteCfg); err != nil {
		t.Fatalf("failed to read test suite row: %v", err)
	}
	if suiteName != "suite-1-renamed" {
		t.Fatalf("suite name = %q, want %q", suiteName, "suite-1-renamed")
	}
	if suiteCfg != `{"x":1}` {
		t.Fatalf("suite config = %q, want %q", suiteCfg, `{"x":1}`)
	}

	if err := db.EnsureBenchmark("bench-1", prompt.ID, "{}"); err != nil {
		t.Fatalf("EnsureBenchmark failed: %v", err)
	}
	if err := db.EnsureBenchmark("bench-1", prompt.ID, `{"models":["gpt-4o-mini"]}`); err != nil {
		t.Fatalf("EnsureBenchmark update failed: %v", err)
	}

	var benchCfg string
	if err := db.QueryRow("SELECT config FROM benchmarks WHERE id = ?", "bench-1").Scan(&benchCfg); err != nil {
		t.Fatalf("failed to read benchmark row: %v", err)
	}
	if benchCfg != `{"models":["gpt-4o-mini"]}` {
		t.Fatalf("benchmark config = %q, want %q", benchCfg, `{"models":["gpt-4o-mini"]}`)
	}
}

func TestChainCRUD(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")

	// Create chain
	chain, err := db.CreateChain(project.ID, "summarize-translate", "Summarize then translate")
	if err != nil {
		t.Fatalf("CreateChain failed: %v", err)
	}
	if chain.Name != "summarize-translate" {
		t.Errorf("expected name 'summarize-translate', got '%s'", chain.Name)
	}

	// Get by name
	byName, err := db.GetChainByName("summarize-translate")
	if err != nil {
		t.Fatalf("GetChainByName failed: %v", err)
	}
	if byName.ID != chain.ID {
		t.Errorf("expected ID '%s', got '%s'", chain.ID, byName.ID)
	}

	// Get by ID
	byID, err := db.GetChainByID(chain.ID)
	if err != nil {
		t.Fatalf("GetChainByID failed: %v", err)
	}
	if byID.Name != chain.Name {
		t.Errorf("expected name '%s', got '%s'", chain.Name, byID.Name)
	}

	// List chains
	db.CreateChain(project.ID, "alpha-chain", "")
	chains, err := db.ListChains()
	if err != nil {
		t.Fatalf("ListChains failed: %v", err)
	}
	if len(chains) != 2 {
		t.Errorf("expected 2 chains, got %d", len(chains))
	}
	if chains[0].Name != "alpha-chain" {
		t.Errorf("expected first chain 'alpha-chain', got '%s'", chains[0].Name)
	}

	// Update
	updated, err := db.UpdateChain(chain.ID, "new-name", "new desc")
	if err != nil {
		t.Fatalf("UpdateChain failed: %v", err)
	}
	if updated.Name != "new-name" {
		t.Errorf("expected name 'new-name', got '%s'", updated.Name)
	}

	// Not found
	notFound, err := db.GetChainByName("nonexistent")
	if err != nil {
		t.Fatalf("GetChainByName failed: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent chain")
	}
}

func TestChainSteps(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	chain, _ := db.CreateChain(project.ID, "my-chain", "")

	// Create steps
	s1, err := db.CreateChainStep(chain.ID, 1, "summarize", `{"text":"{{input.text}}"}`, "summary")
	if err != nil {
		t.Fatalf("CreateChainStep failed: %v", err)
	}
	if s1.PromptName != "summarize" {
		t.Errorf("expected prompt 'summarize', got '%s'", s1.PromptName)
	}

	db.CreateChainStep(chain.ID, 2, "translate", `{"text":"{{steps.summary.output}}"}`, "translation")

	// List steps
	steps, err := db.ListChainSteps(chain.ID)
	if err != nil {
		t.Fatalf("ListChainSteps failed: %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].StepOrder != 1 {
		t.Errorf("expected first step order 1, got %d", steps[0].StepOrder)
	}

	// Replace steps
	newSteps := []ChainStep{
		{StepOrder: 1, PromptName: "expand", InputMapping: `{"text":"{{input.text}}"}`, OutputKey: "expanded"},
	}
	err = db.ReplaceChainSteps(chain.ID, newSteps)
	if err != nil {
		t.Fatalf("ReplaceChainSteps failed: %v", err)
	}

	steps, _ = db.ListChainSteps(chain.ID)
	if len(steps) != 1 {
		t.Errorf("expected 1 step after replace, got %d", len(steps))
	}
	if steps[0].PromptName != "expand" {
		t.Errorf("expected prompt 'expand', got '%s'", steps[0].PromptName)
	}
}

func TestChainRuns(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	chain, _ := db.CreateChain(project.ID, "my-chain", "")

	// Save run
	run, err := db.SaveChainRun(chain.ID, "completed", `{"text":"hello"}`, `[{"step":"s1","output":"hi"}]`, "hi")
	if err != nil {
		t.Fatalf("SaveChainRun failed: %v", err)
	}
	if run.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", run.Status)
	}

	db.SaveChainRun(chain.ID, "failed", `{}`, `[]`, "")

	// List runs
	runs, err := db.ListChainRuns(chain.ID)
	if err != nil {
		t.Fatalf("ListChainRuns failed: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

func TestDeleteChainCascade(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	project, _ := db.CreateProject("test-project")
	chain, _ := db.CreateChain(project.ID, "to-delete", "")
	db.CreateChainStep(chain.ID, 1, "prompt", `{}`, "out")
	db.SaveChainRun(chain.ID, "completed", `{}`, `[]`, "result")

	err := db.DeleteChain(chain.ID)
	if err != nil {
		t.Fatalf("DeleteChain failed: %v", err)
	}

	// Verify all deleted
	found, _ := db.GetChainByID(chain.ID)
	if found != nil {
		t.Error("expected chain to be deleted")
	}

	steps, _ := db.ListChainSteps(chain.ID)
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}

	runs, _ := db.ListChainRuns(chain.ID)
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}

	// Delete non-existent
	err = db.DeleteChain("nonexistent")
	if err == nil {
		t.Error("expected error when deleting non-existent chain")
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
