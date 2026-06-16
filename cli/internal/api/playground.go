package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/promptsmith/cli/internal/benchmark"
	"github.com/promptsmith/cli/internal/db"
)

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

	ctx, cancel := llmContext(r)
	defer cancel()
	start := time.Now()
	resp, err := provider.Complete(ctx, benchmark.CompletionRequest{
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
