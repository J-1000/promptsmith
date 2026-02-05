package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/promptsmith/cli/internal/db"
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
