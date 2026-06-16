package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/promptsmith/cli/internal/db"
)

// Prompt, version, tag, and diff handlers

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
	prompts, err := s.db.ListPromptsWithLatestVersion()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]PromptResponse, 0, len(prompts))
	for _, p := range prompts {
		response = append(response, PromptResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			FilePath:    p.FilePath,
			Version:     p.LatestVersion,
			CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
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
