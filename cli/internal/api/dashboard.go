package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/promptsmith/cli/internal/db"
)

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
