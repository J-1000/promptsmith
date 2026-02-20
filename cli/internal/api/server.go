package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

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
	s.mux.HandleFunc("/api/config/sync", s.corsMiddleware(s.handleSyncConfig))
	s.mux.HandleFunc("/api/tests", s.corsMiddleware(s.handleTests))
	s.mux.HandleFunc("/api/tests/", s.corsMiddleware(s.handleTestByName))
	s.mux.HandleFunc("/api/benchmarks", s.corsMiddleware(s.handleBenchmarks))
	s.mux.HandleFunc("/api/benchmarks/", s.corsMiddleware(s.handleBenchmarkByName))
	s.mux.HandleFunc("/api/generate", s.corsMiddleware(s.handleGenerate))
	s.mux.HandleFunc("/api/generate/", s.corsMiddleware(s.handleGenerateAlias))
	s.mux.HandleFunc("/api/comments/", s.corsMiddleware(s.handleCommentByID))
	s.mux.HandleFunc("/api/playground/run", s.corsMiddleware(s.handlePlaygroundRun))
	s.mux.HandleFunc("/api/providers/models", s.corsMiddleware(s.handleProviderModels))
	s.mux.HandleFunc("/api/dashboard/", s.corsMiddleware(s.handleDashboard))
	s.mux.HandleFunc("/api/chains", s.corsMiddleware(s.handleChains))
	s.mux.HandleFunc("/api/chains/", s.corsMiddleware(s.handleChainByName))
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

func safeJoinProjectPath(root, relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	cleaned := filepath.Clean(relPath)
	fullPath := filepath.Join(root, cleaned)

	relative, err := filepath.Rel(root, fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to validate path: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root")
	}

	return fullPath, nil
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

type SyncConfigResponse struct {
	Team     string `json:"team"`
	Remote   string `json:"remote"`
	AutoPush bool   `json:"auto_push"`
	Status   string `json:"status"`
}

func (s *Server) handleSyncConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	configPath := filepath.Join(s.root, ".promptsmith", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// No config file — return defaults
		writeJSON(w, http.StatusOK, SyncConfigResponse{Status: "not_configured"})
		return
	}

	// Simple YAML key-value parsing (avoids yaml dependency)
	cfg := SyncConfigResponse{Status: "configured"}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "team":
			cfg.Team = val
		case "remote":
			cfg.Remote = val
		case "auto_push":
			cfg.AutoPush = val == "true"
		}
	}

	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handlePrompts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPrompts(w, r)
	case http.MethodPost:
		s.createPrompt(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type CreatePromptRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
	Content     string `json:"content"`
}

func (s *Server) createPrompt(w http.ResponseWriter, r *http.Request) {
	var req CreatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Check for duplicate
	existing, err := s.db.GetPromptByName(req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("prompt '%s' already exists", req.Name))
		return
	}

	// Default file path
	if req.FilePath == "" {
		req.FilePath = fmt.Sprintf("prompts/%s.prompt", req.Name)
	}

	// Get project
	project, err := s.db.GetProject()
	if err != nil || project == nil {
		writeError(w, http.StatusInternalServerError, "no project found")
		return
	}

	filePath, err := safeJoinProjectPath(s.root, req.FilePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	prompt, err := s.db.CreatePrompt(project.ID, req.Name, req.Description, req.FilePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rollbackPrompt := func(cause error) {
		if delErr := s.db.DeletePrompt(prompt.ID); delErr != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("%v (rollback failed: %v)", cause, delErr))
			return
		}
		writeError(w, http.StatusInternalServerError, cause.Error())
	}

	// Write file to disk
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		rollbackPrompt(fmt.Errorf("failed to create directory: %w", err))
		return
	}

	content := req.Content
	if content == "" {
		content = fmt.Sprintf("# %s\n", req.Name)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		rollbackPrompt(fmt.Errorf("failed to write file: %w", err))
		return
	}

	// Create initial version if content provided
	var versionStr string
	if req.Content != "" {
		variables := extractVariables(req.Content)
		variablesJSON, _ := json.Marshal(variables)
		v, err := s.db.CreateVersion(prompt.ID, "1.0.0", req.Content, string(variablesJSON), "{}", "Initial version", "web", nil)
		if err == nil {
			versionStr = v.Version
		}
	}

	writeJSON(w, http.StatusCreated, PromptResponse{
		ID:          prompt.ID,
		Name:        prompt.Name,
		Description: prompt.Description,
		FilePath:    prompt.FilePath,
		Version:     versionStr,
		CreatedAt:   prompt.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
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
		case "tags":
			s.handleTags(w, r, promptID, parts[2:])
			return
		case "comments":
			s.handleComments(w, r, promptID)
			return
		}
	}

	// Get, update, or delete single prompt
	switch r.Method {
	case http.MethodGet:
		s.getPrompt(w, r, promptID)
	case http.MethodPut:
		s.updatePrompt(w, r, promptID)
	case http.MethodDelete:
		s.deletePrompt(w, r, promptID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type UpdatePromptRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) updatePrompt(w http.ResponseWriter, r *http.Request, promptName string) {
	prompt, err := s.db.GetPromptByName(promptName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", promptName))
		return
	}

	var req UpdatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Check for name conflict if renaming
	if req.Name != prompt.Name {
		existing, err := s.db.GetPromptByName(req.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if existing != nil {
			writeError(w, http.StatusConflict, fmt.Sprintf("prompt '%s' already exists", req.Name))
			return
		}
	}

	updated, err := s.db.UpdatePrompt(prompt.ID, req.Name, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	latestVersion, _ := s.db.GetLatestVersion(updated.ID)
	var versionStr string
	if latestVersion != nil {
		versionStr = latestVersion.Version
	}

	writeJSON(w, http.StatusOK, PromptResponse{
		ID:          updated.ID,
		Name:        updated.Name,
		Description: updated.Description,
		FilePath:    updated.FilePath,
		Version:     versionStr,
		CreatedAt:   updated.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (s *Server) deletePrompt(w http.ResponseWriter, r *http.Request, promptName string) {
	prompt, err := s.db.GetPromptByName(promptName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", promptName))
		return
	}

	if err := s.db.DeletePrompt(prompt.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type CreateTagRequest struct {
	Name      string `json:"name"`
	VersionID string `json:"version_id"`
}

func (s *Server) handleTags(w http.ResponseWriter, r *http.Request, promptName string, extra []string) {
	prompt, err := s.db.GetPromptByName(promptName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", promptName))
		return
	}

	// DELETE /api/prompts/:name/tags/:tagName
	if len(extra) > 0 && extra[0] != "" {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		tagName := extra[0]
		if err := s.db.DeleteTag(prompt.ID, tagName); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// POST /api/prompts/:name/tags
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.VersionID == "" {
		writeError(w, http.StatusBadRequest, "name and version_id are required")
		return
	}

	tag, err := s.db.CreateTag(prompt.ID, req.VersionID, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":         tag.ID,
		"name":       tag.Name,
		"version_id": tag.VersionID,
	})
}

func (s *Server) getPrompt(w http.ResponseWriter, r *http.Request, promptID string) {
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

type CreateVersionRequest struct {
	Content       string `json:"content"`
	CommitMessage string `json:"commit_message"`
}

func (s *Server) handleVersions(w http.ResponseWriter, r *http.Request, promptID string) {
	switch r.Method {
	case http.MethodGet:
		// continue below
	case http.MethodPost:
		s.createVersion(w, r, promptID)
		return
	default:
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

func (s *Server) createVersion(w http.ResponseWriter, r *http.Request, promptName string) {
	var req CreateVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.CommitMessage == "" {
		req.CommitMessage = "Updated via web editor"
	}

	// Find prompt
	prompt, err := s.db.GetPromptByName(promptName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", promptName))
		return
	}

	// Get latest version to compute next version
	latest, _ := s.db.GetLatestVersion(prompt.ID)
	nextVersion := "1.0.0"
	var parentID *string
	if latest != nil {
		nextVersion = bumpPatch(latest.Version)
		parentID = &latest.ID
	}

	// Extract variables from content ({{varName}} pattern)
	variables := extractVariables(req.Content)
	variablesJSON, _ := json.Marshal(variables)

	version, err := s.db.CreateVersion(
		prompt.ID,
		nextVersion,
		req.Content,
		string(variablesJSON),
		"{}",
		req.CommitMessage,
		"web",
		parentID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, VersionResponse{
		ID:            version.ID,
		Version:       version.Version,
		Content:       version.Content,
		CommitMessage: version.CommitMessage,
		CreatedAt:     version.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func bumpPatch(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "1.0.1"
	}
	patch := 0
	fmt.Sscanf(parts[2], "%d", &patch)
	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1)
}

func extractVariables(content string) []string {
	var vars []string
	seen := make(map[string]bool)
	i := 0
	for i < len(content)-3 {
		if content[i] == '{' && content[i+1] == '{' {
			end := strings.Index(content[i+2:], "}}")
			if end >= 0 {
				varName := strings.TrimSpace(content[i+2 : i+2+end])
				if varName != "" && !seen[varName] {
					vars = append(vars, varName)
					seen[varName] = true
				}
				i = i + 2 + end + 2
				continue
			}
		}
		i++
	}
	return vars
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
	result, err := runner.Run(context.Background(), suite)
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

func (s *Server) handleGenerateAlias(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/generate/")
	typeMap := map[string]string{
		"variations": "variations",
		"compress":   "compress",
		"expand":     "expand",
		"rephrase":   "rephrase",
	}

	genType, ok := typeMap[path]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown generate type: %s", path))
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Type = genType

	body, _ := json.Marshal(req)
	newReq, _ := http.NewRequest("POST", "/api/generate", strings.NewReader(string(body)))
	newReq.Header.Set("Content-Type", "application/json")

	s.handleGenerate(w, newReq)
}

// Comment handlers

type CommentResponse struct {
	ID         string `json:"id"`
	PromptID   string `json:"prompt_id"`
	VersionID  string `json:"version_id"`
	LineNumber int    `json:"line_number"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
}

type CreateCommentRequest struct {
	VersionID  string `json:"version_id"`
	LineNumber int    `json:"line_number"`
	Content    string `json:"content"`
}

func (s *Server) handleComments(w http.ResponseWriter, r *http.Request, promptName string) {
	prompt, err := s.db.GetPromptByName(promptName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prompt == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", promptName))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listComments(w, prompt.ID)
	case http.MethodPost:
		s.createComment(w, r, prompt.ID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) listComments(w http.ResponseWriter, promptID string) {
	comments, err := s.db.ListComments(promptID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		response = append(response, CommentResponse{
			ID:         c.ID,
			PromptID:   c.PromptID,
			VersionID:  c.VersionID,
			LineNumber: c.LineNumber,
			Content:    c.Content,
			CreatedAt:  c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) createComment(w http.ResponseWriter, r *http.Request, promptID string) {
	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.LineNumber < 0 {
		writeError(w, http.StatusBadRequest, "line_number must be >= 0")
		return
	}

	comment, err := s.db.CreateComment(promptID, req.VersionID, req.LineNumber, req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, CommentResponse{
		ID:         comment.ID,
		PromptID:   comment.PromptID,
		VersionID:  comment.VersionID,
		LineNumber: comment.LineNumber,
		Content:    comment.Content,
		CreatedAt:  comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (s *Server) handleCommentByID(w http.ResponseWriter, r *http.Request) {
	commentID := strings.TrimPrefix(r.URL.Path, "/api/comments/")
	if commentID == "" {
		writeError(w, http.StatusBadRequest, "comment id required")
		return
	}

	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := s.db.DeleteComment(commentID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Playground handlers

type PlaygroundRunRequest struct {
	PromptName  string         `json:"prompt_name,omitempty"`
	Content     string         `json:"content,omitempty"`
	Version     string         `json:"version,omitempty"`
	Model       string         `json:"model"`
	Variables   map[string]any `json:"variables,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
}

type PlaygroundRunResponse struct {
	Output         string  `json:"output"`
	RenderedPrompt string  `json:"rendered_prompt"`
	Model          string  `json:"model"`
	PromptTokens   int     `json:"prompt_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	LatencyMs      int64   `json:"latency_ms"`
	Cost           float64 `json:"cost"`
}

func (s *Server) handlePlaygroundRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req PlaygroundRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	// Resolve prompt content
	promptContent := req.Content
	if req.PromptName != "" && promptContent == "" {
		prompt, err := s.db.GetPromptByName(req.PromptName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if prompt == nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("prompt '%s' not found", req.PromptName))
			return
		}

		var version *db.PromptVersion
		if req.Version != "" {
			version, err = s.db.GetVersionByString(prompt.ID, req.Version)
		} else {
			version, err = s.db.GetLatestVersion(prompt.ID)
		}
		if err != nil || version == nil {
			writeError(w, http.StatusNotFound, "version not found")
			return
		}
		promptContent = version.Content
	}

	if promptContent == "" {
		writeError(w, http.StatusBadRequest, "prompt content or prompt_name is required")
		return
	}

	// Render variables into prompt
	rendered, err := renderPlaygroundPrompt(promptContent, req.Variables)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to render prompt: %v", err))
		return
	}

	// Create provider
	registry := benchmark.NewProviderRegistry()
	if openai, err := benchmark.NewOpenAIProvider(); err == nil {
		registry.Register(openai)
	}
	if anthropic, err := benchmark.NewAnthropicProvider(); err == nil {
		registry.Register(anthropic)
	}

	provider, err := registry.GetForModel(req.Model)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	temperature := 1.0
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	start := time.Now()
	resp, err := provider.Complete(context.Background(), benchmark.CompletionRequest{
		Model:       req.Model,
		Prompt:      rendered,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("completion failed: %v", err))
		return
	}
	latency := time.Since(start).Milliseconds()

	writeJSON(w, http.StatusOK, PlaygroundRunResponse{
		Output:         resp.Content,
		RenderedPrompt: rendered,
		Model:          resp.Model,
		PromptTokens:   resp.PromptTokens,
		OutputTokens:   resp.OutputTokens,
		LatencyMs:      latency,
		Cost:           resp.Cost,
	})
}

func renderPlaygroundPrompt(tmplBody string, vars map[string]any) (string, error) {
	if vars == nil || len(vars) == 0 {
		return tmplBody, nil
	}

	tmpl, err := template.New("prompt").Parse(tmplBody)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type ProviderModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

type ModelInfo struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
}

func (s *Server) handleProviderModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var models []ModelInfo

	if openai, err := benchmark.NewOpenAIProvider(); err == nil {
		for _, m := range openai.Models() {
			models = append(models, ModelInfo{ID: m, Provider: "openai"})
		}
	}

	if anthropic, err := benchmark.NewAnthropicProvider(); err == nil {
		for _, m := range anthropic.Models() {
			models = append(models, ModelInfo{ID: m, Provider: "anthropic"})
		}
	}

	writeJSON(w, http.StatusOK, ProviderModelsResponse{Models: models})
}

// Dashboard handlers

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/dashboard/")
	switch path {
	case "activity":
		s.handleDashboardActivity(w, r)
	case "health":
		s.handleDashboardHealth(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

type ActivityEventResponse struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Timestamp  string `json:"timestamp"`
	PromptName string `json:"prompt_name"`
}

func (s *Server) handleDashboardActivity(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	events, err := s.db.GetRecentActivity(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]ActivityEventResponse, 0, len(events))
	for _, e := range events {
		response = append(response, ActivityEventResponse{
			Type:       e.Type,
			Title:      e.Title,
			Detail:     e.Detail,
			Timestamp:  e.Timestamp.Format("2006-01-02T15:04:05Z"),
			PromptName: e.PromptName,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleDashboardHealth(w http.ResponseWriter, r *http.Request) {
	health, err := s.db.GetPromptHealth()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if health == nil {
		health = []db.PromptHealth{}
	}

	writeJSON(w, http.StatusOK, health)
}

// Chain handlers

type ChainResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	StepCount   int    `json:"step_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ChainDetailResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Steps       []ChainStepResponse `json:"steps"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

type ChainStepResponse struct {
	ID           string          `json:"id"`
	StepOrder    int             `json:"step_order"`
	PromptName   string          `json:"prompt_name"`
	InputMapping json.RawMessage `json:"input_mapping"`
	OutputKey    string          `json:"output_key"`
}

type CreateChainRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateChainRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SaveChainStepsRequest struct {
	Steps []ChainStepInput `json:"steps"`
}

type ChainStepInput struct {
	StepOrder    int             `json:"step_order"`
	PromptName   string          `json:"prompt_name"`
	InputMapping json.RawMessage `json:"input_mapping"`
	OutputKey    string          `json:"output_key"`
}

type RunChainRequest struct {
	Inputs map[string]string `json:"inputs"`
	Model  string            `json:"model"`
}

type ChainRunResponse struct {
	ID          string          `json:"id"`
	Status      string          `json:"status"`
	Inputs      json.RawMessage `json:"inputs"`
	Results     json.RawMessage `json:"results"`
	FinalOutput string          `json:"final_output"`
	StartedAt   string          `json:"started_at"`
	CompletedAt string          `json:"completed_at"`
}

type ChainStepRunResult struct {
	StepOrder      int    `json:"step_order"`
	PromptName     string `json:"prompt_name"`
	OutputKey      string `json:"output_key"`
	RenderedPrompt string `json:"rendered_prompt"`
	Output         string `json:"output"`
	DurationMs     int64  `json:"duration_ms"`
}

func (s *Server) handleChains(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listChains(w, r)
	case http.MethodPost:
		s.createChain(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) listChains(w http.ResponseWriter, r *http.Request) {
	chains, err := s.db.ListChains()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]ChainResponse, 0, len(chains))
	for _, c := range chains {
		steps, _ := s.db.ListChainSteps(c.ID)
		response = append(response, ChainResponse{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			StepCount:   len(steps),
			CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   c.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) createChain(w http.ResponseWriter, r *http.Request) {
	var req CreateChainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	existing, err := s.db.GetChainByName(req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("chain '%s' already exists", req.Name))
		return
	}

	project, err := s.db.GetProject()
	if err != nil || project == nil {
		writeError(w, http.StatusInternalServerError, "no project found")
		return
	}

	chain, err := s.db.CreateChain(project.ID, req.Name, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ChainResponse{
		ID:          chain.ID,
		Name:        chain.Name,
		Description: chain.Description,
		StepCount:   0,
		CreatedAt:   chain.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   chain.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (s *Server) handleChainByName(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/chains/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "chain name required")
		return
	}

	chainName := parts[0]

	if len(parts) >= 2 {
		switch parts[1] {
		case "steps":
			s.handleChainSteps(w, r, chainName)
			return
		case "run":
			s.handleChainRun(w, r, chainName)
			return
		case "runs":
			s.handleChainRuns(w, r, chainName)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		s.getChain(w, r, chainName)
	case http.MethodPut:
		s.updateChain(w, r, chainName)
	case http.MethodDelete:
		s.deleteChain(w, r, chainName)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) getChain(w http.ResponseWriter, r *http.Request, chainName string) {
	chain, err := s.db.GetChainByName(chainName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chain == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("chain '%s' not found", chainName))
		return
	}

	steps, _ := s.db.ListChainSteps(chain.ID)
	stepResponses := make([]ChainStepResponse, 0, len(steps))
	for _, st := range steps {
		stepResponses = append(stepResponses, ChainStepResponse{
			ID:           st.ID,
			StepOrder:    st.StepOrder,
			PromptName:   st.PromptName,
			InputMapping: json.RawMessage(st.InputMapping),
			OutputKey:    st.OutputKey,
		})
	}

	writeJSON(w, http.StatusOK, ChainDetailResponse{
		ID:          chain.ID,
		Name:        chain.Name,
		Description: chain.Description,
		Steps:       stepResponses,
		CreatedAt:   chain.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   chain.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (s *Server) updateChain(w http.ResponseWriter, r *http.Request, chainName string) {
	chain, err := s.db.GetChainByName(chainName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chain == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("chain '%s' not found", chainName))
		return
	}

	var req UpdateChainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.Name != chain.Name {
		existing, err := s.db.GetChainByName(req.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if existing != nil {
			writeError(w, http.StatusConflict, fmt.Sprintf("chain '%s' already exists", req.Name))
			return
		}
	}

	updated, err := s.db.UpdateChain(chain.ID, req.Name, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	steps, _ := s.db.ListChainSteps(updated.ID)
	writeJSON(w, http.StatusOK, ChainResponse{
		ID:          updated.ID,
		Name:        updated.Name,
		Description: updated.Description,
		StepCount:   len(steps),
		CreatedAt:   updated.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   updated.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (s *Server) deleteChain(w http.ResponseWriter, r *http.Request, chainName string) {
	chain, err := s.db.GetChainByName(chainName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chain == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("chain '%s' not found", chainName))
		return
	}

	if err := s.db.DeleteChain(chain.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleChainSteps(w http.ResponseWriter, r *http.Request, chainName string) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	chain, err := s.db.GetChainByName(chainName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chain == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("chain '%s' not found", chainName))
		return
	}

	var req SaveChainStepsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate step references
	outputKeys := make(map[string]int)
	for _, step := range req.Steps {
		if step.PromptName == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("step %d: prompt_name is required", step.StepOrder))
			return
		}
		if step.OutputKey == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("step %d: output_key is required", step.StepOrder))
			return
		}
		outputKeys[step.OutputKey] = step.StepOrder
	}

	// Convert to db structs
	dbSteps := make([]db.ChainStep, len(req.Steps))
	for i, step := range req.Steps {
		mappingJSON, _ := json.Marshal(step.InputMapping)
		dbSteps[i] = db.ChainStep{
			StepOrder:    step.StepOrder,
			PromptName:   step.PromptName,
			InputMapping: string(mappingJSON),
			OutputKey:    step.OutputKey,
		}
	}

	if err := s.db.ReplaceChainSteps(chain.ID, dbSteps); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return updated steps
	steps, _ := s.db.ListChainSteps(chain.ID)
	stepResponses := make([]ChainStepResponse, 0, len(steps))
	for _, st := range steps {
		stepResponses = append(stepResponses, ChainStepResponse{
			ID:           st.ID,
			StepOrder:    st.StepOrder,
			PromptName:   st.PromptName,
			InputMapping: json.RawMessage(st.InputMapping),
			OutputKey:    st.OutputKey,
		})
	}

	writeJSON(w, http.StatusOK, stepResponses)
}

func (s *Server) handleChainRun(w http.ResponseWriter, r *http.Request, chainName string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	chain, err := s.db.GetChainByName(chainName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chain == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("chain '%s' not found", chainName))
		return
	}

	var req RunChainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	steps, err := s.db.ListChainSteps(chain.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(steps) == 0 {
		writeError(w, http.StatusBadRequest, "chain has no steps")
		return
	}

	// Create provider
	registry := benchmark.NewProviderRegistry()
	if openai, err := benchmark.NewOpenAIProvider(); err == nil {
		registry.Register(openai)
	}
	if anthropic, err := benchmark.NewAnthropicProvider(); err == nil {
		registry.Register(anthropic)
	}

	provider, err := registry.GetForModel(req.Model)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Execute chain steps
	stepOutputs := make(map[string]string)
	var stepResults []ChainStepRunResult
	var finalOutput string

	for _, step := range steps {
		// Resolve input mapping
		var inputMap map[string]string
		if err := json.Unmarshal([]byte(step.InputMapping), &inputMap); err != nil {
			inputMap = map[string]string{}
		}

		resolvedVars := make(map[string]any)
		for varName, source := range inputMap {
			resolved := resolveChainInput(source, req.Inputs, stepOutputs)
			resolvedVars[varName] = resolved
		}

		// Load prompt and render
		prompt, err := s.db.GetPromptByName(step.PromptName)
		if err != nil || prompt == nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("step %d: prompt '%s' not found", step.StepOrder, step.PromptName))
			return
		}

		version, err := s.db.GetLatestVersion(prompt.ID)
		if err != nil || version == nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("step %d: no version for prompt '%s'", step.StepOrder, step.PromptName))
			return
		}

		rendered, err := renderPlaygroundPrompt(version.Content, resolvedVars)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("step %d: render failed: %v", step.StepOrder, err))
			return
		}

		start := time.Now()
		resp, err := provider.Complete(context.Background(), benchmark.CompletionRequest{
			Model:       req.Model,
			Prompt:      rendered,
			MaxTokens:   1024,
			Temperature: 1.0,
		})
		if err != nil {
			// Save failed run
			inputsJSON, _ := json.Marshal(req.Inputs)
			resultsJSON, _ := json.Marshal(stepResults)
			s.db.SaveChainRun(chain.ID, "failed", string(inputsJSON), string(resultsJSON), "")
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("step %d failed: %v", step.StepOrder, err))
			return
		}
		duration := time.Since(start).Milliseconds()

		stepOutputs[step.OutputKey] = resp.Content
		finalOutput = resp.Content

		stepResults = append(stepResults, ChainStepRunResult{
			StepOrder:      step.StepOrder,
			PromptName:     step.PromptName,
			OutputKey:      step.OutputKey,
			RenderedPrompt: rendered,
			Output:         resp.Content,
			DurationMs:     duration,
		})
	}

	// Save successful run
	inputsJSON, _ := json.Marshal(req.Inputs)
	resultsJSON, _ := json.Marshal(stepResults)
	run, err := s.db.SaveChainRun(chain.ID, "completed", string(inputsJSON), string(resultsJSON), finalOutput)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ChainRunResponse{
		ID:          run.ID,
		Status:      run.Status,
		Inputs:      json.RawMessage(inputsJSON),
		Results:     json.RawMessage(resultsJSON),
		FinalOutput: finalOutput,
		StartedAt:   run.StartedAt.Format("2006-01-02T15:04:05Z"),
		CompletedAt: run.CompletedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func resolveChainInput(source string, inputs map[string]string, stepOutputs map[string]string) string {
	if strings.HasPrefix(source, "{{input.") && strings.HasSuffix(source, "}}") {
		key := source[8 : len(source)-2]
		if v, ok := inputs[key]; ok {
			return v
		}
		return ""
	}
	if strings.HasPrefix(source, "{{steps.") && strings.HasSuffix(source, "}}") {
		inner := source[8 : len(source)-2]
		dotIdx := strings.Index(inner, ".")
		if dotIdx > 0 {
			stepKey := inner[:dotIdx]
			if v, ok := stepOutputs[stepKey]; ok {
				return v
			}
		}
		return ""
	}
	return source
}

func (s *Server) handleChainRuns(w http.ResponseWriter, r *http.Request, chainName string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	chain, err := s.db.GetChainByName(chainName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chain == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("chain '%s' not found", chainName))
		return
	}

	runs, err := s.db.ListChainRuns(chain.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]ChainRunResponse, 0, len(runs))
	for _, run := range runs {
		response = append(response, ChainRunResponse{
			ID:          run.ID,
			Status:      run.Status,
			Inputs:      json.RawMessage(run.Inputs),
			Results:     json.RawMessage(run.Results),
			FinalOutput: run.FinalOutput,
			StartedAt:   run.StartedAt.Format("2006-01-02T15:04:05Z"),
			CompletedAt: run.CompletedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, response)
}
