package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/promptsmith/cli/internal/benchmark"
)

// Benchmark handlers

func (s *Server) handleBenchmarks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// continue below
	case http.MethodPost:
		s.createBenchmarkSuite(w, r)
		return
	default:
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

type CreateBenchmarkSuiteRequest struct {
	Name         string   `json:"name"`
	Prompt       string   `json:"prompt"`
	Description  string   `json:"description"`
	Models       []string `json:"models"`
	RunsPerModel int      `json:"runs_per_model"`
}

func (s *Server) createBenchmarkSuite(w http.ResponseWriter, r *http.Request) {
	var req CreateBenchmarkSuiteRequest
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

	// Defaults
	if len(req.Models) == 0 {
		req.Models = []string{"gpt-4o-mini"}
	}
	if req.RunsPerModel <= 0 {
		req.RunsPerModel = 3
	}

	// Write YAML file
	benchDir := filepath.Join(s.root, "benchmarks")
	if err := os.MkdirAll(benchDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create benchmarks dir: %v", err))
		return
	}

	filePath, err := safeJoinProjectPath(s.root, filepath.Join("benchmarks", req.Name+".bench.yaml"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := os.Stat(filePath); err == nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("benchmark suite '%s' already exists", req.Name))
		return
	}

	desc := ""
	if req.Description != "" {
		desc = fmt.Sprintf("description: %s\n", req.Description)
	}

	modelsYAML := ""
	for _, m := range req.Models {
		modelsYAML += fmt.Sprintf("  - %s\n", m)
	}

	content := fmt.Sprintf(`name: %s
prompt: %s
%smodels:
%sruns_per_model: %d
`, req.Name, req.Prompt, desc, modelsYAML, req.RunsPerModel)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write file: %v", err))
		return
	}
	if err := s.db.EnsureBenchmark(req.Name, prompt.ID, content); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	relPath, _ := filepath.Rel(s.root, filePath)
	writeJSON(w, http.StatusCreated, BenchmarkSuiteResponse{
		Name:         req.Name,
		FilePath:     relPath,
		Prompt:       req.Prompt,
		Description:  req.Description,
		Models:       req.Models,
		RunsPerModel: req.RunsPerModel,
	})
}

func (s *Server) handleBenchmarkByName(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/benchmarks/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "benchmark name required")
		return
	}

	benchName := parts[0]

	// Check for sub-endpoints
	if len(parts) >= 2 {
		switch parts[1] {
		case "run":
			s.runBenchmark(w, r, benchName)
			return
		case "runs":
			s.listBenchmarkRuns(w, r, benchName)
			return
		}
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
	ctx, cancel := llmContext(r)
	defer cancel()
	result, err := runner.Run(ctx, suite)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Persist run results
	prompt, err := s.db.GetPromptByName(suite.Prompt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", suite.Prompt))
		return
	}
	if err := s.db.EnsureBenchmark(benchName, prompt.ID, "{}"); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resultsJSON, _ := json.Marshal(result)
	if _, err := s.db.SaveBenchmarkRun(benchName, "", string(resultsJSON)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

type BenchmarkRunResponse struct {
	ID          string          `json:"id"`
	BenchmarkID string          `json:"benchmark_id"`
	Results     json.RawMessage `json:"results"`
	CreatedAt   string          `json:"created_at"`
}

func (s *Server) listBenchmarkRuns(w http.ResponseWriter, r *http.Request, benchName string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	runs, err := s.db.ListBenchmarkRuns(benchName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]BenchmarkRunResponse, 0, len(runs))
	for _, run := range runs {
		response = append(response, BenchmarkRunResponse{
			ID:          run.ID,
			BenchmarkID: run.BenchmarkID,
			Results:     json.RawMessage(run.Results),
			CreatedAt:   run.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

type BenchmarkSuiteResponse struct {
	Name         string   `json:"name"`
	FilePath     string   `json:"file_path"`
	Prompt       string   `json:"prompt"`
	Description  string   `json:"description,omitempty"`
	Models       []string `json:"models"`
	RunsPerModel int      `json:"runs_per_model"`
}
