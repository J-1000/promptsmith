package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/promptsmith/cli/internal/benchmark"
	"github.com/promptsmith/cli/internal/db"
)

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
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Steps       []ChainStepResponse `json:"steps"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
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
	chains, err := s.db.ListChainsWithStepCounts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]ChainResponse, 0, len(chains))
	for _, c := range chains {
		response = append(response, ChainResponse{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			StepCount:   c.StepCount,
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

	// Execute chain steps. A single bounded context spans the whole run so a
	// hung step cannot block the request indefinitely.
	ctx, cancel := llmContext(r)
	defer cancel()

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
		resp, err := provider.Complete(ctx, benchmark.CompletionRequest{
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
