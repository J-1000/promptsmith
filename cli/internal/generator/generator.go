package generator

import (
	"context"
	"fmt"
	"strings"

	"github.com/promptsmith/cli/internal/benchmark"
)

// GenerationType defines the type of generation to perform
type GenerationType string

const (
	TypeVariations GenerationType = "variations"
	TypeCompress   GenerationType = "compress"
	TypeExpand     GenerationType = "expand"
	TypeRephrase   GenerationType = "rephrase"
)

// GenerateRequest defines a request to generate prompt variations
type GenerateRequest struct {
	Type        GenerationType
	Prompt      string            // The original prompt content
	Count       int               // Number of variations to generate
	Goal        string            // Optional goal (e.g., "reduce tokens", "improve clarity")
	Model       string            // Model to use for generation
	Options     map[string]string // Additional options
}

// Variation represents a generated prompt variation
type Variation struct {
	Content     string `json:"content"`
	Description string `json:"description"`
	TokenDelta  int    `json:"token_delta,omitempty"` // Change in token count vs original
}

// GenerateResult holds the results of a generation request
type GenerateResult struct {
	Original    string      `json:"original"`
	Variations  []Variation `json:"variations"`
	Model       string      `json:"model"`
	Type        string      `json:"type"`
	Goal        string      `json:"goal,omitempty"`
}

// Generator generates prompt variations using an LLM
type Generator struct {
	provider benchmark.Provider
}

// New creates a new Generator with the given provider
func New(provider benchmark.Provider) *Generator {
	return &Generator{provider: provider}
}

// Generate creates prompt variations based on the request
func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	if req.Count <= 0 {
		req.Count = 3
	}
	if req.Count > 10 {
		req.Count = 10
	}

	systemPrompt := g.buildSystemPrompt(req)
	userPrompt := g.buildUserPrompt(req)

	fullPrompt := fmt.Sprintf("%s\n\n%s", systemPrompt, userPrompt)

	resp, err := g.provider.Complete(ctx, benchmark.CompletionRequest{
		Model:       req.Model,
		Prompt:      fullPrompt,
		MaxTokens:   4096,
		Temperature: 0.8, // Higher temperature for more creative variations
	})
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	variations := g.parseVariations(resp.Content, req.Count)

	return &GenerateResult{
		Original:   req.Prompt,
		Variations: variations,
		Model:      req.Model,
		Type:       string(req.Type),
		Goal:       req.Goal,
	}, nil
}

func (g *Generator) buildSystemPrompt(req GenerateRequest) string {
	switch req.Type {
	case TypeVariations:
		return `You are an expert prompt engineer. Generate variations of the given prompt that achieve the same goal but with different approaches, wording, or structure. Each variation should be complete and usable.`
	case TypeCompress:
		return `You are an expert prompt engineer specializing in token optimization. Compress the given prompt to use fewer tokens while preserving its core functionality and intent. Remove unnecessary words, combine instructions, and use concise language.`
	case TypeExpand:
		return `You are an expert prompt engineer. Expand the given prompt to be more detailed, comprehensive, and robust. Add clarifications, examples, edge case handling, and clearer instructions.`
	case TypeRephrase:
		return `You are an expert prompt engineer. Rephrase the given prompt using different wording while keeping the exact same meaning and functionality. Vary sentence structure and vocabulary.`
	default:
		return `You are an expert prompt engineer. Generate variations of the given prompt.`
	}
}

func (g *Generator) buildUserPrompt(req GenerateRequest) string {
	var sb strings.Builder

	sb.WriteString("Original prompt:\n```\n")
	sb.WriteString(req.Prompt)
	sb.WriteString("\n```\n\n")

	if req.Goal != "" {
		sb.WriteString(fmt.Sprintf("Goal: %s\n\n", req.Goal))
	}

	sb.WriteString(fmt.Sprintf("Generate exactly %d variations. ", req.Count))
	sb.WriteString("Format each variation as:\n")
	sb.WriteString("---VARIATION---\n")
	sb.WriteString("Description: [brief description of changes]\n")
	sb.WriteString("```\n[the variation content]\n```\n")

	return sb.String()
}

func (g *Generator) parseVariations(content string, expectedCount int) []Variation {
	variations := make([]Variation, 0, expectedCount)

	// Split by variation marker
	parts := strings.Split(content, "---VARIATION---")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var v Variation

		// Extract description
		if idx := strings.Index(part, "Description:"); idx != -1 {
			endIdx := strings.Index(part[idx:], "\n")
			if endIdx != -1 {
				v.Description = strings.TrimSpace(part[idx+12 : idx+endIdx])
			}
		}

		// Extract content between ``` markers
		if startIdx := strings.Index(part, "```"); startIdx != -1 {
			remaining := part[startIdx+3:]
			// Skip language identifier if present
			if newline := strings.Index(remaining, "\n"); newline != -1 {
				remaining = remaining[newline+1:]
			}
			if endIdx := strings.Index(remaining, "```"); endIdx != -1 {
				v.Content = strings.TrimSpace(remaining[:endIdx])
			}
		}

		if v.Content != "" {
			variations = append(variations, v)
		}

		if len(variations) >= expectedCount {
			break
		}
	}

	return variations
}

// EstimateTokens provides a rough token count estimate
func EstimateTokens(text string) int {
	// Rough estimate: ~4 characters per token on average
	return len(text) / 4
}
