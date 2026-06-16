package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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

type ProjectResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
