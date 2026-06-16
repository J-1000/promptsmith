package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/promptsmith/cli/internal/db"
)

type Server struct {
	db   *db.DB
	root string
	mux  *http.ServeMux
}

const maxRequestBodyBytes int64 = 10 << 20 // 10 MiB

// llmRequestTimeout bounds long-running LLM operations (benchmarks, generation,
// playground and chain runs) triggered by an API request.
const llmRequestTimeout = 5 * time.Minute

// llmContext derives a bounded context from the request so that work is
// cancelled when the client disconnects or the deadline elapses, rather than
// running unbounded with context.Background().
func llmContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), llmRequestTimeout)
}

var allowedCORSOrigins = map[string]struct{}{
	"http://localhost:8080": {},
	"http://127.0.0.1:8080": {},
	"http://localhost:8081": {},
	"http://127.0.0.1:8081": {},
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
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowedCORSOrigins[origin]; !ok {
				writeError(w, http.StatusForbidden, "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}

		next(w, r)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(addr string) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return server.ListenAndServe()
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
	Name         string   `json:"name"`
	FilePath     string   `json:"file_path"`
	Prompt       string   `json:"prompt"`
	Description  string   `json:"description,omitempty"`
	Models       []string `json:"models"`
	RunsPerModel int      `json:"runs_per_model"`
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
