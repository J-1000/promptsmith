package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/promptsmith/cli/internal/db"
)

// Test helper to set up a test project
func setupTestProject(t *testing.T) (string, *db.DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "promptsmith-api-test-*")
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

	// Create directories
	for _, dir := range []string{"prompts", "tests", "benchmarks"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			database.Close()
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create %s dir: %v", dir, err)
		}
	}

	// Create a test prompt
	_, err = database.CreatePrompt(project.ID, "summarizer", "Summarizes text", "prompts/summarizer.prompt")
	if err != nil {
		database.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create prompt: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return tmpDir, database, cleanup
}

func TestServerRoutes(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	tests := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{"GET", "/api/project", http.StatusOK},
		{"GET", "/api/prompts", http.StatusOK},
		{"GET", "/api/tests", http.StatusOK},
		{"GET", "/api/benchmarks", http.StatusOK},
		{"POST", "/api/project", http.StatusMethodNotAllowed},
		{"OPTIONS", "/api/prompts", http.StatusOK}, // CORS preflight
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestGetProject(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/project", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response ProjectResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "test-project" {
		t.Errorf("project name = %q, want %q", response.Name, "test-project")
	}
}

func TestListPrompts(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/prompts", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response []PromptResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("got %d prompts, want 1", len(response))
	}

	if len(response) > 0 && response[0].Name != "summarizer" {
		t.Errorf("prompt name = %q, want %q", response[0].Name, "summarizer")
	}
}

func TestCreatePromptRejectsPathTraversal(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	body := `{"name":"escape","file_path":"../../outside.prompt","content":"test"}`
	req := httptest.NewRequest("POST", "/api/prompts", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	outsidePath := filepath.Join(filepath.Dir(tmpDir), "outside.prompt")
	if _, err := os.Stat(outsidePath); !os.IsNotExist(err) {
		t.Fatalf("outside file should not exist, err=%v", err)
	}
}

func TestCreatePromptRollsBackOnFileWriteFailure(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	blockingPath := filepath.Join(tmpDir, "blocked")
	if err := os.WriteFile(blockingPath, []byte("not-a-directory"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	server := NewServer(database, tmpDir)

	body := `{"name":"rollback-prompt","file_path":"blocked/rollback.prompt","content":"x"}`
	req := httptest.NewRequest("POST", "/api/prompts", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	prompt, err := database.GetPromptByName("rollback-prompt")
	if err != nil {
		t.Fatalf("failed to query prompt: %v", err)
	}
	if prompt != nil {
		t.Fatal("prompt row should be rolled back on file write failure")
	}
}

func TestGetPromptByName(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Test existing prompt
	req := httptest.NewRequest("GET", "/api/prompts/summarizer", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response PromptResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "summarizer" {
		t.Errorf("name = %q, want %q", response.Name, "summarizer")
	}

	// Test non-existent prompt
	req = httptest.NewRequest("GET", "/api/prompts/nonexistent", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetPromptVersions(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create versions
	prompt, _ := database.GetPromptByName("summarizer")
	v1, _ := database.CreateVersion(prompt.ID, "1.0.0", "content v1", "[]", "{}", "First", "user", nil)
	database.CreateVersion(prompt.ID, "1.0.1", "content v2", "[]", "{}", "Second", "user", &v1.ID)
	database.CreateTag(prompt.ID, v1.ID, "prod")

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/prompts/summarizer/versions", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response []VersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("got %d versions, want 2", len(response))
	}

	// Check that tags are included
	foundProdTag := false
	for _, v := range response {
		if v.Version == "1.0.0" {
			for _, tag := range v.Tags {
				if tag == "prod" {
					foundProdTag = true
				}
			}
		}
	}
	if !foundProdTag {
		t.Error("expected 'prod' tag on version 1.0.0")
	}
}

func TestGetPromptDiff(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create versions
	prompt, _ := database.GetPromptByName("summarizer")
	v1, _ := database.CreateVersion(prompt.ID, "1.0.0", "content v1", "[]", "{}", "First", "user", nil)
	database.CreateVersion(prompt.ID, "1.0.1", "content v2", "[]", "{}", "Second", "user", &v1.ID)

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/prompts/summarizer/diff?v1=1.0.0&v2=1.0.1", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["prompt"] != "summarizer" {
		t.Errorf("prompt = %v, want %q", response["prompt"], "summarizer")
	}

	v1Data, ok := response["v1"].(map[string]interface{})
	if !ok {
		t.Fatal("v1 data missing")
	}
	if v1Data["content"] != "content v1" {
		t.Errorf("v1 content = %v, want %q", v1Data["content"], "content v1")
	}

	v2Data, ok := response["v2"].(map[string]interface{})
	if !ok {
		t.Fatal("v2 data missing")
	}
	if v2Data["content"] != "content v2" {
		t.Errorf("v2 content = %v, want %q", v2Data["content"], "content v2")
	}
}

func TestDiffMissingParams(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Missing both params
	req := httptest.NewRequest("GET", "/api/prompts/summarizer/diff", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Missing v2
	req = httptest.NewRequest("GET", "/api/prompts/summarizer/diff?v1=1.0.0", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCORSHeaders(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/prompts", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	// Check CORS headers
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing Access-Control-Allow-Origin header")
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing Access-Control-Allow-Methods header")
	}
}

func TestListTests(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a test file
	testContent := `name: test-suite
prompt: summarizer
tests:
  - name: basic-test
    inputs:
      text: "hello"
    assertions:
      - type: not_empty
`
	testPath := filepath.Join(tmpDir, "tests", "summarizer.test.yaml")
	if err := os.WriteFile(testPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/tests", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response []TestSuiteResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("got %d test suites, want 1", len(response))
	}

	if len(response) > 0 {
		if response[0].Name != "test-suite" {
			t.Errorf("name = %q, want %q", response[0].Name, "test-suite")
		}
		if response[0].TestCount != 1 {
			t.Errorf("test_count = %d, want 1", response[0].TestCount)
		}
	}
}

func TestCreateTestSuiteRejectsPathTraversal(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	body := `{"name":"../../escape","prompt":"summarizer"}`
	req := httptest.NewRequest("POST", "/api/tests", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	outsidePath := filepath.Join(filepath.Dir(tmpDir), "escape.test.yaml")
	if _, err := os.Stat(outsidePath); !os.IsNotExist(err) {
		t.Fatalf("outside file should not exist, err=%v", err)
	}
}

func TestListBenchmarks(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a benchmark file
	benchContent := `name: bench-suite
prompt: summarizer
models:
  - gpt-4o
  - gpt-4o-mini
runs_per_model: 3
`
	benchPath := filepath.Join(tmpDir, "benchmarks", "summarizer.bench.yaml")
	if err := os.WriteFile(benchPath, []byte(benchContent), 0644); err != nil {
		t.Fatalf("failed to write benchmark file: %v", err)
	}

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/benchmarks", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response []BenchmarkSuiteResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("got %d benchmark suites, want 1", len(response))
	}

	if len(response) > 0 {
		if response[0].Name != "bench-suite" {
			t.Errorf("name = %q, want %q", response[0].Name, "bench-suite")
		}
		if len(response[0].Models) != 2 {
			t.Errorf("models count = %d, want 2", len(response[0].Models))
		}
		if response[0].RunsPerModel != 3 {
			t.Errorf("runs_per_model = %d, want 3", response[0].RunsPerModel)
		}
	}
}

func TestCreateBenchmarkSuiteRejectsPathTraversal(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	body := `{"name":"../../escape","prompt":"summarizer"}`
	req := httptest.NewRequest("POST", "/api/benchmarks", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	outsidePath := filepath.Join(filepath.Dir(tmpDir), "escape.bench.yaml")
	if _, err := os.Stat(outsidePath); !os.IsNotExist(err) {
		t.Fatalf("outside file should not exist, err=%v", err)
	}
}

func TestEmptyTestsAndBenchmarks(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Empty tests
	req := httptest.NewRequest("GET", "/api/tests", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("tests status = %d, want %d", rec.Code, http.StatusOK)
	}

	var tests []TestSuiteResponse
	json.NewDecoder(rec.Body).Decode(&tests)
	if len(tests) != 0 {
		t.Errorf("expected 0 tests, got %d", len(tests))
	}

	// Empty benchmarks
	req = httptest.NewRequest("GET", "/api/benchmarks", nil)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("benchmarks status = %d, want %d", rec.Code, http.StatusOK)
	}

	var benchmarks []BenchmarkSuiteResponse
	json.NewDecoder(rec.Body).Decode(&benchmarks)
	if len(benchmarks) != 0 {
		t.Errorf("expected 0 benchmarks, got %d", len(benchmarks))
	}
}

func TestGetTestByName(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a test file
	testContent := `name: my-test-suite
prompt: summarizer
description: Tests for summarizer
tests:
  - name: basic-test
    inputs:
      text: "hello"
    assertions:
      - type: not_empty
  - name: advanced-test
    inputs:
      text: "world"
    assertions:
      - type: contains
        value: "result"
`
	testPath := filepath.Join(tmpDir, "tests", "summarizer.test.yaml")
	if err := os.WriteFile(testPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	server := NewServer(database, tmpDir)

	// Test getting existing suite
	req := httptest.NewRequest("GET", "/api/tests/my-test-suite", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["name"] != "my-test-suite" {
		t.Errorf("name = %v, want %q", response["name"], "my-test-suite")
	}

	if response["prompt"] != "summarizer" {
		t.Errorf("prompt = %v, want %q", response["prompt"], "summarizer")
	}

	// Test non-existent suite
	req = httptest.NewRequest("GET", "/api/tests/nonexistent", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetBenchmarkByName(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a benchmark file
	benchContent := `name: my-bench-suite
prompt: summarizer
description: Benchmark for summarizer
models:
  - gpt-4o
  - gpt-4o-mini
  - claude-sonnet
runs_per_model: 5
`
	benchPath := filepath.Join(tmpDir, "benchmarks", "summarizer.bench.yaml")
	if err := os.WriteFile(benchPath, []byte(benchContent), 0644); err != nil {
		t.Fatalf("failed to write benchmark file: %v", err)
	}

	server := NewServer(database, tmpDir)

	// Test getting existing suite
	req := httptest.NewRequest("GET", "/api/benchmarks/my-bench-suite", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["name"] != "my-bench-suite" {
		t.Errorf("name = %v, want %q", response["name"], "my-bench-suite")
	}

	if response["prompt"] != "summarizer" {
		t.Errorf("prompt = %v, want %q", response["prompt"], "summarizer")
	}

	models := response["models"].([]interface{})
	if len(models) != 3 {
		t.Errorf("models count = %d, want 3", len(models))
	}

	// Test non-existent suite
	req = httptest.NewRequest("GET", "/api/benchmarks/nonexistent", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestTestRunEndpoint(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a test file
	testContent := `name: runnable-test
prompt: summarizer
tests:
  - name: simple-test
    inputs:
      text: "hello"
    assertions:
      - type: not_empty
`
	testPath := filepath.Join(tmpDir, "tests", "runnable.test.yaml")
	if err := os.WriteFile(testPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	server := NewServer(database, tmpDir)

	// POST to run endpoint
	req := httptest.NewRequest("POST", "/api/tests/runnable-test/run", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	// Should succeed (even without actual executor)
	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Errorf("unexpected status = %d", rec.Code)
	}

	// Test run on non-existent test
	req = httptest.NewRequest("POST", "/api/tests/nonexistent/run", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Test wrong method on run endpoint
	req = httptest.NewRequest("GET", "/api/tests/runnable-test/run", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGetTestRunRejectsMismatchedSuite(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	run, err := database.SaveTestRun("suite-a", "", "passed", `{"ok":true}`)
	if err != nil {
		t.Fatalf("failed to save run: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/tests/suite-b/runs/"+run.ID, nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestBenchmarkRunEndpoint(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a benchmark file
	benchContent := `name: runnable-bench
prompt: summarizer
models:
  - gpt-4o-mini
runs_per_model: 1
`
	benchPath := filepath.Join(tmpDir, "benchmarks", "runnable.bench.yaml")
	if err := os.WriteFile(benchPath, []byte(benchContent), 0644); err != nil {
		t.Fatalf("failed to write benchmark file: %v", err)
	}

	server := NewServer(database, tmpDir)

	// Test run on non-existent benchmark
	req := httptest.NewRequest("POST", "/api/benchmarks/nonexistent/run", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Test wrong method on run endpoint
	req = httptest.NewRequest("GET", "/api/benchmarks/runnable-bench/run", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestMissingPromptID(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Test missing prompt ID
	req := httptest.NewRequest("GET", "/api/prompts/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMissingTestName(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Test missing test name
	req := httptest.NewRequest("GET", "/api/tests/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMissingBenchmarkName(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Test missing benchmark name
	req := httptest.NewRequest("GET", "/api/benchmarks/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGenerateEndpointValidation(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Test wrong method
	req := httptest.NewRequest("GET", "/api/generate", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}

	// Test empty body
	req = httptest.NewRequest("POST", "/api/generate", strings.NewReader("{}"))
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/api/generate", strings.NewReader("invalid json"))
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateVersion(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create an initial version
	prompt, _ := database.GetPromptByName("summarizer")
	database.CreateVersion(prompt.ID, "1.0.0", "initial content", "[]", "{}", "Initial", "user", nil)

	server := NewServer(database, tmpDir)

	// Create new version via API
	body := `{"content": "Hello {{name}}, welcome to {{place}}!", "commit_message": "Add variables"}`
	req := httptest.NewRequest("POST", "/api/prompts/summarizer/versions", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response VersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Version != "1.0.1" {
		t.Errorf("version = %q, want %q", response.Version, "1.0.1")
	}
	if response.Content != "Hello {{name}}, welcome to {{place}}!" {
		t.Errorf("content mismatch")
	}
	if response.CommitMessage != "Add variables" {
		t.Errorf("commit_message = %q, want %q", response.CommitMessage, "Add variables")
	}
}

func TestCreateVersionFirstVersion(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Create first version (no existing versions)
	body := `{"content": "new prompt content"}`
	req := httptest.NewRequest("POST", "/api/prompts/summarizer/versions", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response VersionResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", response.Version, "1.0.0")
	}
	if response.CommitMessage != "Updated via web editor" {
		t.Errorf("commit_message = %q, want default message", response.CommitMessage)
	}
}

func TestCreateVersionValidation(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Empty content
	body := `{"content": ""}`
	req := httptest.NewRequest("POST", "/api/prompts/summarizer/versions", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Invalid JSON
	req = httptest.NewRequest("POST", "/api/prompts/summarizer/versions", strings.NewReader("not json"))
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Non-existent prompt
	body = `{"content": "test"}`
	req = httptest.NewRequest("POST", "/api/prompts/nonexistent/versions", strings.NewReader(body))
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestBumpPatch(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "1.0.1"},
		{"1.0.9", "1.0.10"},
		{"2.3.5", "2.3.6"},
		{"0.0.0", "0.0.1"},
		{"invalid", "1.0.1"},
	}

	for _, tt := range tests {
		result := bumpPatch(tt.input)
		if result != tt.expected {
			t.Errorf("bumpPatch(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Hello {{name}}", []string{"name"}},
		{"{{a}} and {{b}}", []string{"a", "b"}},
		{"{{ name }} with spaces", []string{"name"}},
		{"no variables here", nil},
		{"{{a}} then {{a}} again", []string{"a"}}, // dedup
		{"{{x}} {{y}} {{z}}", []string{"x", "y", "z"}},
	}

	for _, tt := range tests {
		result := extractVariables(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("extractVariables(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("extractVariables(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

func TestCreatePrompt(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Create a new prompt
	body := `{"name": "translator", "description": "Translates text", "content": "Translate {{text}} to {{language}}"}`
	req := httptest.NewRequest("POST", "/api/prompts", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response PromptResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "translator" {
		t.Errorf("name = %q, want %q", response.Name, "translator")
	}
	if response.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", response.Version, "1.0.0")
	}
}

func TestCreatePromptValidation(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Empty name
	body := `{"name": ""}`
	req := httptest.NewRequest("POST", "/api/prompts", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Duplicate name
	body = `{"name": "summarizer"}`
	req = httptest.NewRequest("POST", "/api/prompts", strings.NewReader(body))
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestDeletePrompt(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Delete existing prompt
	req := httptest.NewRequest("DELETE", "/api/prompts/summarizer", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	// Verify it's gone
	req = httptest.NewRequest("GET", "/api/prompts/summarizer", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Delete non-existent
	req = httptest.NewRequest("DELETE", "/api/prompts/nonexistent", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("not found: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCreateTag(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	prompt, _ := database.GetPromptByName("summarizer")
	v, _ := database.CreateVersion(prompt.ID, "1.0.0", "content", "[]", "{}", "Initial", "user", nil)

	server := NewServer(database, tmpDir)

	body := `{"name": "prod", "version_id": "` + v.ID + `"}`
	req := httptest.NewRequest("POST", "/api/prompts/summarizer/tags", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["name"] != "prod" {
		t.Errorf("name = %q, want %q", response["name"], "prod")
	}
}

func TestDeleteTag(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	prompt, _ := database.GetPromptByName("summarizer")
	v, _ := database.CreateVersion(prompt.ID, "1.0.0", "content", "[]", "{}", "Initial", "user", nil)
	database.CreateTag(prompt.ID, v.ID, "staging")

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("DELETE", "/api/prompts/summarizer/tags/staging", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	// Delete non-existent tag
	req = httptest.NewRequest("DELETE", "/api/prompts/summarizer/tags/nonexistent", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("not found: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCreateBenchmarkSuite(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	body := `{"name": "new-bench", "prompt": "summarizer", "models": ["gpt-4o", "claude-sonnet"], "runs_per_model": 5}`
	req := httptest.NewRequest("POST", "/api/benchmarks", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response BenchmarkSuiteResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "new-bench" {
		t.Errorf("name = %q, want %q", response.Name, "new-bench")
	}
	if len(response.Models) != 2 {
		t.Errorf("models count = %d, want 2", len(response.Models))
	}
	if response.RunsPerModel != 5 {
		t.Errorf("runs_per_model = %d, want 5", response.RunsPerModel)
	}

	// Verify file exists
	filePath := filepath.Join(tmpDir, "benchmarks", "new-bench.bench.yaml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("benchmark file was not created")
	}

	// Duplicate should fail
	req = httptest.NewRequest("POST", "/api/benchmarks", strings.NewReader(body))
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate: status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestCreateTestSuite(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Create test suite
	body := `{"name": "new-test", "prompt": "summarizer", "description": "My test suite"}`
	req := httptest.NewRequest("POST", "/api/tests", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response TestSuiteResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "new-test" {
		t.Errorf("name = %q, want %q", response.Name, "new-test")
	}
	if response.Prompt != "summarizer" {
		t.Errorf("prompt = %q, want %q", response.Prompt, "summarizer")
	}

	// Verify file was written
	filePath := filepath.Join(tmpDir, "tests", "new-test.test.yaml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("test file was not created")
	}

	// Duplicate should fail
	req = httptest.NewRequest("POST", "/api/tests", strings.NewReader(body))
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate: status = %d, want %d", rec.Code, http.StatusConflict)
	}

	// Missing name
	body = `{"prompt": "summarizer"}`
	req = httptest.NewRequest("POST", "/api/tests", strings.NewReader(body))
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing name: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Non-existent prompt
	body = `{"name": "test2", "prompt": "nonexistent"}`
	req = httptest.NewRequest("POST", "/api/tests", strings.NewReader(body))
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("bad prompt: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestListBenchmarkRuns(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	database.SaveBenchmarkRun("my-bench", "", `{"models":[]}`)
	database.SaveBenchmarkRun("my-bench", "", `{"models":[{"model":"gpt-4o"}]}`)

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/benchmarks/my-bench/runs", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response []BenchmarkRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("got %d runs, want 2", len(response))
	}
}

func TestListTestRuns(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Save some test runs
	database.SaveTestRun("my-suite", "", "passed", `{"passed":2}`)
	database.SaveTestRun("my-suite", "", "failed", `{"failed":1}`)

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/tests/my-suite/runs", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response []TestRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("got %d runs, want 2", len(response))
	}
}

func TestGetTestRun(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	run, _ := database.SaveTestRun("my-suite", "", "passed", `{"passed":2}`)

	server := NewServer(database, tmpDir)

	// Get existing run
	req := httptest.NewRequest("GET", "/api/tests/my-suite/runs/"+run.ID, nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response TestRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != run.ID {
		t.Errorf("id = %q, want %q", response.ID, run.ID)
	}
	if response.Status != "passed" {
		t.Errorf("status = %q, want %q", response.Status, "passed")
	}

	// Get non-existent run
	req = httptest.NewRequest("GET", "/api/tests/my-suite/runs/nonexistent", nil)
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUpdatePrompt(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Update existing prompt
	body := `{"name": "summarizer-v2", "description": "Updated description"}`
	req := httptest.NewRequest("PUT", "/api/prompts/summarizer", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response PromptResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "summarizer-v2" {
		t.Errorf("name = %q, want %q", response.Name, "summarizer-v2")
	}
	if response.Description != "Updated description" {
		t.Errorf("description = %q, want %q", response.Description, "Updated description")
	}
}

func TestUpdatePromptValidation(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Empty name
	body := `{"name": ""}`
	req := httptest.NewRequest("PUT", "/api/prompts/summarizer", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// Non-existent prompt
	body = `{"name": "new-name"}`
	req = httptest.NewRequest("PUT", "/api/prompts/nonexistent", strings.NewReader(body))
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// Name conflict: create another prompt first
	project, _ := database.GetProject()
	database.CreatePrompt(project.ID, "other-prompt", "", "prompts/other.prompt")

	body = `{"name": "other-prompt"}`
	req = httptest.NewRequest("PUT", "/api/prompts/summarizer", strings.NewReader(body))
	rec = httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestGenerateAliasEndpoints(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// Test each alias endpoint parses correctly (they'll fail with no API key, but validate routing)
	aliases := []string{"variations", "compress", "expand", "rephrase"}
	for _, alias := range aliases {
		body := `{"prompt": "Test prompt content"}`
		req := httptest.NewRequest("POST", "/api/generate/"+alias, strings.NewReader(body))
		rec := httptest.NewRecorder()

		server.ServeHTTP(rec, req)

		// Should get 500 (no API key) not 404
		if rec.Code == http.StatusNotFound {
			t.Errorf("/api/generate/%s returned 404, expected route to exist", alias)
		}
	}

	// Unknown type should 404
	body := `{"prompt": "test"}`
	req := httptest.NewRequest("POST", "/api/generate/unknown", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown type: status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// GET should be method not allowed
	req = httptest.NewRequest("GET", "/api/generate/compress", nil)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGenerateEndpointDefaults(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	// This will fail because no API key is set, but we can verify the request parsing works
	body := `{"prompt": "Test prompt content"}`
	req := httptest.NewRequest("POST", "/api/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	// Without API key, should return internal server error
	// This validates the request was parsed and defaults were applied
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d (expected provider error without API key)", rec.Code, http.StatusInternalServerError)
	}
}

func TestSyncConfigNotConfigured(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/config/sync", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp SyncConfigResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "not_configured" {
		t.Errorf("status = %q, want %q", resp.Status, "not_configured")
	}
}

func TestDashboardActivity(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create some versions to generate activity
	prompt, _ := database.GetPromptByName("summarizer")
	database.CreateVersion(prompt.ID, "1.0.0", "content v1", "[]", "{}", "First version", "user", nil)
	database.CreateVersion(prompt.ID, "1.0.1", "content v2", "[]", "{}", "Second version", "user", nil)

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/dashboard/activity", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response []ActivityEventResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("got %d events, want 2", len(response))
	}

	if len(response) > 0 && response[0].Type != "version" {
		t.Errorf("first event type = %q, want %q", response[0].Type, "version")
	}
}

func TestDashboardActivityWithLimit(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	prompt, _ := database.GetPromptByName("summarizer")
	database.CreateVersion(prompt.ID, "1.0.0", "v1", "[]", "{}", "First", "user", nil)
	database.CreateVersion(prompt.ID, "1.0.1", "v2", "[]", "{}", "Second", "user", nil)
	database.CreateVersion(prompt.ID, "1.0.2", "v3", "[]", "{}", "Third", "user", nil)

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/dashboard/activity?limit=2", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response []ActivityEventResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if len(response) != 2 {
		t.Errorf("got %d events, want 2 (limited)", len(response))
	}
}

func TestDashboardActivityEmpty(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/dashboard/activity", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response []ActivityEventResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response != nil && len(response) != 0 {
		t.Errorf("expected empty activity, got %d events", len(response))
	}
}

func TestDashboardHealth(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a version for the existing prompt
	prompt, _ := database.GetPromptByName("summarizer")
	database.CreateVersion(prompt.ID, "1.0.0", "content", "[]", "{}", "Initial", "user", nil)

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/dashboard/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("got %d prompts, want 1", len(response))
	}

	if response[0]["prompt_name"] != "summarizer" {
		t.Errorf("prompt_name = %v, want %q", response[0]["prompt_name"], "summarizer")
	}
	if response[0]["version_count"].(float64) != 1 {
		t.Errorf("version_count = %v, want 1", response[0]["version_count"])
	}
	if response[0]["last_test_status"] != "none" {
		t.Errorf("last_test_status = %v, want %q", response[0]["last_test_status"], "none")
	}
}

func TestDashboardHealthEmpty(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Delete the default prompt
	prompt, _ := database.GetPromptByName("summarizer")
	database.DeletePrompt(prompt.ID)

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/dashboard/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response []map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	if len(response) != 0 {
		t.Errorf("expected empty health, got %d entries", len(response))
	}
}

func TestDashboardMethodNotAllowed(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("POST", "/api/dashboard/activity", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestDashboardNotFound(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/dashboard/unknown", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSyncConfigConfigured(t *testing.T) {
	tmpDir, database, cleanup := setupTestProject(t)
	defer cleanup()

	// Write config file
	configDir := filepath.Join(tmpDir, ".promptsmith")
	os.MkdirAll(configDir, 0o755)
	configContent := "team: acme-team\nremote: https://sync.example.com\nauto_push: true\n"
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644)

	server := NewServer(database, tmpDir)

	req := httptest.NewRequest("GET", "/api/config/sync", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp SyncConfigResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "configured" {
		t.Errorf("status = %q, want %q", resp.Status, "configured")
	}
	if resp.Team != "acme-team" {
		t.Errorf("team = %q, want %q", resp.Team, "acme-team")
	}
	if resp.Remote != "https://sync.example.com" {
		t.Errorf("remote = %q, want %q", resp.Remote, "https://sync.example.com")
	}
	if !resp.AutoPush {
		t.Error("auto_push = false, want true")
	}
}
