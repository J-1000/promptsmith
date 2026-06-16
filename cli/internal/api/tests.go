package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/promptsmith/cli/internal/testing"
)

// Test handlers

func (s *Server) handleTests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// continue below
	case http.MethodPost:
		s.createTestSuite(w, r)
		return
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	testsDir := filepath.Join(s.root, "tests")
	if _, err := os.Stat(testsDir); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, []TestSuiteResponse{})
		return
	}

	matches, err := filepath.Glob(filepath.Join(testsDir, "*.test.yaml"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]TestSuiteResponse, 0, len(matches))
	for _, file := range matches {
		suite, err := testing.ParseSuiteFile(file)
		if err != nil {
			continue // Skip invalid files
		}

		relPath, _ := filepath.Rel(s.root, file)
		response = append(response, TestSuiteResponse{
			Name:        suite.Name,
			FilePath:    relPath,
			Prompt:      suite.Prompt,
			Description: suite.Description,
			TestCount:   len(suite.Tests),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleTestByName(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tests/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "test name required")
		return
	}

	testName := parts[0]

	// Check for sub-endpoints
	if len(parts) >= 2 {
		switch parts[1] {
		case "run":
			s.runTest(w, r, testName)
			return
		case "runs":
			if len(parts) >= 3 && parts[2] != "" {
				s.getTestRun(w, r, testName, parts[2])
			} else {
				s.listTestRuns(w, r, testName)
			}
			return
		}
	}

	// Get single test suite info
	s.getTest(w, r, testName)
}

func (s *Server) getTest(w http.ResponseWriter, r *http.Request, testName string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	testsDir := filepath.Join(s.root, "tests")
	matches, err := filepath.Glob(filepath.Join(testsDir, "*.test.yaml"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, file := range matches {
		suite, err := testing.ParseSuiteFile(file)
		if err != nil {
			continue
		}
		if suite.Name == testName {
			writeJSON(w, http.StatusOK, suite)
			return
		}
	}

	writeError(w, http.StatusNotFound, fmt.Sprintf("test suite '%s' not found", testName))
}

func (s *Server) runTest(w http.ResponseWriter, r *http.Request, testName string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	testsDir := filepath.Join(s.root, "tests")
	matches, err := filepath.Glob(filepath.Join(testsDir, "*.test.yaml"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var suite *testing.TestSuite
	for _, file := range matches {
		s, err := testing.ParseSuiteFile(file)
		if err != nil {
			continue
		}
		if s.Name == testName {
			suite = s
			break
		}
	}

	if suite == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("test suite '%s' not found", testName))
		return
	}

	// Run the test suite
	runner := testing.NewRunner(s.db, nil) // Using mock executor
	result, err := runner.Run(suite)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Persist run results
	status := "passed"
	if result.Failed > 0 {
		status = "failed"
	}
	prompt, err := s.db.GetPromptByName(suite.Prompt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", suite.Prompt))
		return
	}
	if err := s.db.EnsureTestSuite(testName, prompt.ID, testName, "{}"); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resultsJSON, _ := json.Marshal(result)
	if _, err := s.db.SaveTestRun(testName, "", status, string(resultsJSON)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

type CreateTestSuiteRequest struct {
	Name        string `json:"name"`
	Prompt      string `json:"prompt"`
	Description string `json:"description"`
}

func (s *Server) createTestSuite(w http.ResponseWriter, r *http.Request) {
	var req CreateTestSuiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	// Check prompt exists
	prompt, err := s.db.GetPromptByName(req.Prompt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", req.Prompt))
		return
	}

	// Write YAML file
	testsDir := filepath.Join(s.root, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create tests dir: %v", err))
		return
	}

	filePath, err := safeJoinProjectPath(s.root, filepath.Join("tests", req.Name+".test.yaml"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check for existing file
	if _, err := os.Stat(filePath); err == nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("test suite '%s' already exists", req.Name))
		return
	}

	desc := ""
	if req.Description != "" {
		desc = fmt.Sprintf("description: %s\n", req.Description)
	}

	content := fmt.Sprintf(`name: %s
prompt: %s
%stests:
  - name: example-test
    inputs:
      text: "hello"
    assertions:
      - type: not_empty
`, req.Name, req.Prompt, desc)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write file: %v", err))
		return
	}
	if err := s.db.EnsureTestSuite(req.Name, prompt.ID, req.Name, content); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	relPath, _ := filepath.Rel(s.root, filePath)
	writeJSON(w, http.StatusCreated, TestSuiteResponse{
		Name:        req.Name,
		FilePath:    relPath,
		Prompt:      req.Prompt,
		Description: req.Description,
		TestCount:   1,
	})
}

type TestRunResponse struct {
	ID          string          `json:"id"`
	SuiteID     string          `json:"suite_id"`
	Status      string          `json:"status"`
	Results     json.RawMessage `json:"results"`
	StartedAt   string          `json:"started_at"`
	CompletedAt string          `json:"completed_at"`
}

func (s *Server) listTestRuns(w http.ResponseWriter, r *http.Request, testName string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	runs, err := s.db.ListTestRuns(testName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]TestRunResponse, 0, len(runs))
	for _, run := range runs {
		response = append(response, TestRunResponse{
			ID:          run.ID,
			SuiteID:     run.SuiteID,
			Status:      run.Status,
			Results:     json.RawMessage(run.Results),
			StartedAt:   run.StartedAt.Format("2006-01-02T15:04:05Z"),
			CompletedAt: run.CompletedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) getTestRun(w http.ResponseWriter, r *http.Request, testName string, runID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	run, err := s.db.GetTestRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("test run '%s' not found", runID))
		return
	}
	if run.SuiteID != testName {
		writeError(w, http.StatusNotFound, fmt.Sprintf("test run '%s' not found in suite '%s'", runID, testName))
		return
	}

	writeJSON(w, http.StatusOK, TestRunResponse{
		ID:          run.ID,
		SuiteID:     run.SuiteID,
		Status:      run.Status,
		Results:     json.RawMessage(run.Results),
		StartedAt:   run.StartedAt.Format("2006-01-02T15:04:05Z"),
		CompletedAt: run.CompletedAt.Format("2006-01-02T15:04:05Z"),
	})
}

type TestSuiteResponse struct {
	Name        string `json:"name"`
	FilePath    string `json:"file_path"`
	Prompt      string `json:"prompt"`
	Description string `json:"description,omitempty"`
	TestCount   int    `json:"test_count"`
}
