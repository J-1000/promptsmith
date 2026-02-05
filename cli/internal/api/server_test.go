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
