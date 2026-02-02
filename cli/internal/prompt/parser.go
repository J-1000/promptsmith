package prompt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const frontmatterDelimiter = "---"

type Variable struct {
	Name     string   `yaml:"name" json:"name"`
	Type     string   `yaml:"type" json:"type"`
	Required bool     `yaml:"required" json:"required"`
	Default  any      `yaml:"default,omitempty" json:"default,omitempty"`
	Values   []string `yaml:"values,omitempty" json:"values,omitempty"` // for enum type
}

type Frontmatter struct {
	Name        string     `yaml:"name" json:"name"`
	Description string     `yaml:"description" json:"description"`
	ModelHint   string     `yaml:"model_hint" json:"model_hint"`
	Variables   []Variable `yaml:"variables" json:"variables"`
}

type ParsedPrompt struct {
	Frontmatter       *Frontmatter
	Content           string
	ExtractedVars     []string // Variables found in template ({{var}})
	RawContent        string   // Original file content
	HasFrontmatter    bool
}

func Parse(content string) (*ParsedPrompt, error) {
	parsed := &ParsedPrompt{
		RawContent: content,
	}

	// Check for frontmatter
	if strings.HasPrefix(strings.TrimSpace(content), frontmatterDelimiter) {
		parts := strings.SplitN(content, frontmatterDelimiter, 3)
		if len(parts) >= 3 {
			// Has frontmatter
			parsed.HasFrontmatter = true
			frontmatterStr := strings.TrimSpace(parts[1])
			parsed.Content = strings.TrimSpace(parts[2])

			var fm Frontmatter
			if err := yaml.Unmarshal([]byte(frontmatterStr), &fm); err != nil {
				return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
			}
			parsed.Frontmatter = &fm
		} else {
			// No valid frontmatter, treat entire content as prompt
			parsed.Content = content
		}
	} else {
		parsed.Content = content
	}

	// Extract Mustache variables from content
	parsed.ExtractedVars = extractMustacheVars(parsed.Content)

	return parsed, nil
}

func extractMustacheVars(content string) []string {
	// Match {{variable}} patterns, excluding {{#section}} and {{/section}}
	re := regexp.MustCompile(`\{\{([^#/}][^}]*)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	var vars []string
	for _, match := range matches {
		if len(match) > 1 {
			varName := strings.TrimSpace(match[1])
			if !seen[varName] {
				seen[varName] = true
				vars = append(vars, varName)
			}
		}
	}
	return vars
}

func (p *ParsedPrompt) VariablesJSON() string {
	vars := p.ExtractedVars
	if p.Frontmatter != nil && len(p.Frontmatter.Variables) > 0 {
		// Use frontmatter variables as authoritative
		data, _ := json.Marshal(p.Frontmatter.Variables)
		return string(data)
	}
	// Fall back to extracted variables
	extracted := make([]Variable, len(vars))
	for i, v := range vars {
		extracted[i] = Variable{
			Name:     v,
			Type:     "string",
			Required: true,
		}
	}
	data, _ := json.Marshal(extracted)
	return string(data)
}

func (p *ParsedPrompt) MetadataJSON() string {
	metadata := map[string]any{}
	if p.Frontmatter != nil {
		if p.Frontmatter.ModelHint != "" {
			metadata["model_hint"] = p.Frontmatter.ModelHint
		}
	}
	if len(metadata) == 0 {
		return "{}"
	}
	data, _ := json.Marshal(metadata)
	return string(data)
}

func (p *ParsedPrompt) Name() string {
	if p.Frontmatter != nil && p.Frontmatter.Name != "" {
		return p.Frontmatter.Name
	}
	return ""
}

func (p *ParsedPrompt) Description() string {
	if p.Frontmatter != nil {
		return p.Frontmatter.Description
	}
	return ""
}
