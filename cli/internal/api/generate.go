package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/promptsmith/cli/internal/benchmark"
	"github.com/promptsmith/cli/internal/generator"
)

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
	ctx, cancel := llmContext(r)
	defer cancel()
	result, err := gen.Generate(ctx, generator.GenerateRequest{
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
