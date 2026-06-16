package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

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
