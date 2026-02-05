package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/promptsmith/cli/internal/benchmark"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/generator"
	"github.com/promptsmith/cli/internal/testing"
)

type Server struct {
	db   *db.DB
	root string
	mux  *http.ServeMux
}

func NewServer(database *db.DB, projectRoot string) *Server {
	s := &Server{
		db:   database,
		root: projectRoot,
		mux:  http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Enable CORS for all routes
	s.mux.HandleFunc("/api/prompts", s.corsMiddleware(s.handlePrompts))
	s.mux.HandleFunc("/api/prompts/", s.corsMiddleware(s.handlePromptByID))
	s.mux.HandleFunc("/api/project", s.corsMiddleware(s.handleProject))
	s.mux.HandleFunc("/api/tests", s.corsMiddleware(s.handleTests))
	s.mux.HandleFunc("/api/tests/", s.corsMiddleware(s.handleTestByName))
	s.mux.HandleFunc("/api/benchmarks", s.corsMiddleware(s.handleBenchmarks))
	s.mux.HandleFunc("/api/benchmarks/", s.corsMiddleware(s.handleBenchmarkByName))
	s.mux.HandleFunc("/api/generate", s.corsMiddleware(s.handleGenerate))
}

func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// API Response types

type PromptResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
	Version     string `json:"version,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type VersionResponse struct {
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	Content       string   `json:"content"`
	CommitMessage string   `json:"commit_message"`
	CreatedAt     string   `json:"created_at"`
	Tags          []string `json:"tags,omitempty"`
}

type ProjectResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TestSuiteResponse struct {
	Name        string `json:"name"`
	FilePath    string `json:"file_path"`
	Prompt      string `json:"prompt"`
	Description string `json:"description,omitempty"`
	TestCount   int    `json:"test_count"`
}

type BenchmarkSuiteResponse struct {
	Name        string   `json:"name"`
	FilePath    string   `json:"file_path"`
	Prompt      string   `json:"prompt"`
	Description string   `json:"description,omitempty"`
	Models      []string `json:"models"`
	RunsPerModel int     `json:"runs_per_model"`
}

// Handlers

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	project, err := s.db.GetProject()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, "no project found")
		return
	}

	writeJSON(w, http.StatusOK, ProjectResponse{
		ID:   project.ID,
		Name: project.Name,
	})
}

func (s *Server) handlePrompts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPrompts(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) listPrompts(w http.ResponseWriter, r *http.Request) {
	prompts, err := s.db.ListPrompts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]PromptResponse, 0, len(prompts))
	for _, p := range prompts {
		pr := PromptResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			FilePath:    p.FilePath,
			CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		// Get latest version
		latestVersion, err := s.db.GetLatestVersion(p.ID)
		if err == nil && latestVersion != nil {
			pr.Version = latestVersion.Version
		}

		response = append(response, pr)
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handlePromptByID(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/prompts/{id} or /api/prompts/{id}/versions
	path := strings.TrimPrefix(r.URL.Path, "/api/prompts/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "prompt id required")
		return
	}

	promptID := parts[0]

	if len(parts) >= 2 {
		switch parts[1] {
		case "versions":
			s.handleVersions(w, r, promptID)
			return
		case "diff":
			s.handleDiff(w, r, promptID)
			return
		}
	}

	// Get single prompt
	s.getPrompt(w, r, promptID)
}

func (s *Server) getPrompt(w http.ResponseWriter, r *http.Request, promptID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Try to find prompt by ID first, then by name
	prompt, err := s.db.GetPromptByName(promptID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", promptID))
		return
	}

	latestVersion, _ := s.db.GetLatestVersion(prompt.ID)

	response := PromptResponse{
		ID:          prompt.ID,
		Name:        prompt.Name,
		Description: prompt.Description,
		FilePath:    prompt.FilePath,
		CreatedAt:   prompt.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if latestVersion != nil {
		response.Version = latestVersion.Version
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleVersions(w http.ResponseWriter, r *http.Request, promptID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Find prompt by name
	prompt, err := s.db.GetPromptByName(promptID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", promptID))
		return
	}

	versions, err := s.db.ListVersions(prompt.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get tags for each version
	tags, err := s.db.ListTags(prompt.ID)
	if err != nil {
		tags = []*db.Tag{}
	}

	// Build tag map
	tagMap := make(map[string][]string)
	for _, t := range tags {
		tagMap[t.VersionID] = append(tagMap[t.VersionID], t.Name)
	}

	response := make([]VersionResponse, 0, len(versions))
	for _, v := range versions {
		vr := VersionResponse{
			ID:            v.ID,
			Version:       v.Version,
			Content:       v.Content,
			CommitMessage: v.CommitMessage,
			CreatedAt:     v.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Tags:          tagMap[v.ID],
		}
		response = append(response, vr)
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request, promptID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	v1 := r.URL.Query().Get("v1")
	v2 := r.URL.Query().Get("v2")

	if v1 == "" || v2 == "" {
		writeError(w, http.StatusBadRequest, "v1 and v2 query parameters required")
		return
	}

	// Find prompt by name
	prompt, err := s.db.GetPromptByName(promptID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", promptID))
		return
	}

	// Get versions
	version1, err := s.db.GetVersionByString(prompt.ID, v1)
	if err != nil || version1 == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("version '%s' not found", v1))
		return
	}

	version2, err := s.db.GetVersionByString(prompt.ID, v2)
	if err != nil || version2 == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("version '%s' not found", v2))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"prompt": promptID,
		"v1": map[string]string{
			"version": version1.Version,
			"content": version1.Content,
		},
		"v2": map[string]string{
			"version": version2.Version,
			"content": version2.Content,
		},
	})
}

// Test handlers

func (s *Server) handleTests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	// Check for /run endpoint
	if len(parts) >= 2 && parts[1] == "run" {
		s.runTest(w, r, testName)
		return
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

	writeJSON(w, http.StatusOK, result)
}

// Benchmark handlers

func (s *Server) handleBenchmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	benchDir := filepath.Join(s.root, "benchmarks")
	if _, err := os.Stat(benchDir); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, []BenchmarkSuiteResponse{})
		return
	}

	matches, err := filepath.Glob(filepath.Join(benchDir, "*.bench.yaml"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]BenchmarkSuiteResponse, 0, len(matches))
	for _, file := range matches {
		suite, err := benchmark.ParseSuiteFile(file)
		if err != nil {
			continue // Skip invalid files
		}

		relPath, _ := filepath.Rel(s.root, file)
		response = append(response, BenchmarkSuiteResponse{
			Name:         suite.Name,
			FilePath:     relPath,
			Prompt:       suite.Prompt,
			Description:  suite.Description,
			Models:       suite.Models,
			RunsPerModel: suite.RunsPerModel,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleBenchmarkByName(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/benchmarks/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "benchmark name required")
		return
	}

	benchName := parts[0]

	// Check for /run endpoint
	if len(parts) >= 2 && parts[1] == "run" {
		s.runBenchmark(w, r, benchName)
		return
	}

	// Get single benchmark suite info
	s.getBenchmark(w, r, benchName)
}

func (s *Server) getBenchmark(w http.ResponseWriter, r *http.Request, benchName string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	benchDir := filepath.Join(s.root, "benchmarks")
	matches, err := filepath.Glob(filepath.Join(benchDir, "*.bench.yaml"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, file := range matches {
		suite, err := benchmark.ParseSuiteFile(file)
		if err != nil {
			continue
		}
		if suite.Name == benchName {
			writeJSON(w, http.StatusOK, suite)
			return
		}
	}

	writeError(w, http.StatusNotFound, fmt.Sprintf("benchmark suite '%s' not found", benchName))
}

func (s *Server) runBenchmark(w http.ResponseWriter, r *http.Request, benchName string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	benchDir := filepath.Join(s.root, "benchmarks")
	matches, err := filepath.Glob(filepath.Join(benchDir, "*.bench.yaml"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var suite *benchmark.Suite
	for _, file := range matches {
		bs, err := benchmark.ParseSuiteFile(file)
		if err != nil {
			continue
		}
		if bs.Name == benchName {
			suite = bs
			break
		}
	}

	if suite == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("benchmark suite '%s' not found", benchName))
		return
	}

	// Create provider registry
	registry := benchmark.NewProviderRegistry()
	if openai, err := benchmark.NewOpenAIProvider(); err == nil {
		registry.Register(openai)
	}
	if anthropic, err := benchmark.NewAnthropicProvider(); err == nil {
		registry.Register(anthropic)
	}

	// Run the benchmark suite
	runner := benchmark.NewRunner(s.db, registry)
	result, err := runner.Run(context.Background(), suite)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Generate handlers

type GenerateRequest struct {
	Type   string `json:"type"`   // variations, compress, expand, rephrase
	Prompt string `json:"prompt"` // The prompt content to generate from
	Count  int    `json:"count"`  // Number of variations (default 3)
	Goal   string `json:"goal"`   // Optional goal
	Model  string `json:"model"`  // Model to use (default gpt-4o-mini)
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	if req.Type == "" {
		req.Type = "variations"
	}

	if req.Count <= 0 {
		req.Count = 3
	}

	if req.Model == "" {
		req.Model = "gpt-4o-mini"
	}

	// Create provider based on model
	var provider benchmark.Provider
	var err error

	if strings.HasPrefix(req.Model, "gpt-") || strings.HasPrefix(req.Model, "o1") {
		provider, err = benchmark.NewOpenAIProvider()
	} else if strings.HasPrefix(req.Model, "claude") {
		provider, err = benchmark.NewAnthropicProvider()
	} else {
		// Default to OpenAI
		provider, err = benchmark.NewOpenAIProvider()
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create provider: %v", err))
		return
	}

	gen := generator.New(provider)
	result, err := gen.Generate(context.Background(), generator.GenerateRequest{
		Type:   generator.GenerationType(req.Type),
		Prompt: req.Prompt,
		Count:  req.Count,
		Goal:   req.Goal,
		Model:  req.Model,
	})

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
