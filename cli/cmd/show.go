package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/prompt"
	"github.com/spf13/cobra"
)

var showVersion string

var showCmd = &cobra.Command{
	Use:   "show <prompt>",
	Short: "Show prompt details",
	Long: `Display detailed information about a prompt including its content,
variables, and metadata.

Examples:
  promptsmith show summarizer
  promptsmith show summarizer --version 1.0.0
  promptsmith show summarizer --json`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func init() {
	showCmd.Flags().StringVarP(&showVersion, "version", "v", "", "show specific version")
	rootCmd.AddCommand(showCmd)
}

type showOutput struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Version     string         `json:"version"`
	FilePath    string         `json:"file_path"`
	Variables   []variableInfo `json:"variables,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Content     string         `json:"content"`
	CreatedAt   string         `json:"created_at,omitempty"`
	CreatedBy   string         `json:"created_by,omitempty"`
}

type variableInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Required bool   `json:"required,omitempty"`
	Default  any    `json:"default,omitempty"`
}

func runShow(cmd *cobra.Command, args []string) error {
	promptName := args[0]

	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	// Get the prompt
	p, err := database.GetPromptByName(promptName)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("prompt '%s' not found", promptName)
	}

	// Get version
	var version *db.PromptVersion
	if showVersion != "" {
		version, err = database.GetVersionByString(p.ID, showVersion)
		if err != nil {
			return err
		}
		if version == nil {
			return fmt.Errorf("version '%s' not found for prompt '%s'", showVersion, promptName)
		}
	} else {
		version, err = database.GetLatestVersion(p.ID)
		if err != nil {
			return err
		}
		if version == nil {
			return fmt.Errorf("no versions found for prompt '%s'", promptName)
		}
	}

	// Get tags
	tags, err := database.ListTags(p.ID)
	if err != nil {
		tags = []*db.Tag{}
	}

	// Filter tags for this version
	var versionTags []string
	for _, t := range tags {
		if t.VersionID == version.ID {
			versionTags = append(versionTags, t.Name)
		}
	}

	// Parse the prompt content to get variables
	parsed, _ := prompt.Parse(version.Content)

	// Use parsed template content if available, otherwise raw content
	content := version.Content
	if parsed != nil && parsed.Content != "" {
		content = parsed.Content
	}

	output := showOutput{
		Name:        p.Name,
		Description: p.Description,
		Version:     version.Version,
		FilePath:    p.FilePath,
		Tags:        versionTags,
		Content:     content,
		CreatedAt:   version.CreatedAt.Format("2006-01-02T15:04:05Z"),
		CreatedBy:   version.CreatedBy,
	}

	if parsed != nil && parsed.Frontmatter != nil {
		output.Variables = make([]variableInfo, len(parsed.Frontmatter.Variables))
		for i, v := range parsed.Frontmatter.Variables {
			output.Variables[i] = variableInfo{
				Name:     v.Name,
				Type:     v.Type,
				Required: v.Required,
				Default:  v.Default,
			}
		}
	}

	// JSON output
	if jsonOut {
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Text output
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("%s %s@%s\n", cyan("▶"), output.Name, output.Version)
	fmt.Printf("  File: %s\n", dim(output.FilePath))
	if output.Description != "" {
		fmt.Printf("  Description: %s\n", output.Description)
	}
	if len(output.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", green(strings.Join(output.Tags, ", ")))
	}
	fmt.Printf("  Created: %s by %s\n", dim(output.CreatedAt), output.CreatedBy)

	if len(output.Variables) > 0 {
		fmt.Printf("\n%s\n", yellow("Variables:"))
		for _, v := range output.Variables {
			req := ""
			if v.Required {
				req = " (required)"
			}
			def := ""
			if v.Default != nil {
				def = fmt.Sprintf(" [default: %v]", v.Default)
			}
			fmt.Printf("  - %s: %s%s%s\n", v.Name, v.Type, req, def)
		}
	}

	fmt.Printf("\n%s\n", yellow("Content:"))
	fmt.Printf("%s\n", dim(strings.Repeat("─", 60)))
	fmt.Println(output.Content)
	fmt.Printf("%s\n", dim(strings.Repeat("─", 60)))

	return nil
}
