package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all tracked prompts",
	Long: `List all prompts tracked in the current project.

Shows each prompt with its current version, description, and tags.

Examples:
  promptsmith list
  promptsmith ls
  promptsmith list --json`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

type listItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version"`
	FilePath    string   `json:"file_path"`
	Tags        []string `json:"tags,omitempty"`
}

func runList(cmd *cobra.Command, args []string) error {
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	prompts, err := database.ListPrompts()
	if err != nil {
		return err
	}

	if len(prompts) == 0 {
		fmt.Println("No prompts tracked yet.")
		fmt.Printf("Use %s to start tracking prompts.\n", "promptsmith add <file>")
		return nil
	}

	items := make([]listItem, 0, len(prompts))

	for _, p := range prompts {
		item := listItem{
			Name:        p.Name,
			Description: p.Description,
			FilePath:    p.FilePath,
		}

		// Get latest version
		latestVersion, err := database.GetLatestVersion(p.ID)
		if err == nil && latestVersion != nil {
			item.Version = latestVersion.Version
		} else {
			item.Version = "0.0.0"
		}

		// Get tags
		tags, err := database.ListTags(p.ID)
		if err == nil && len(tags) > 0 {
			item.Tags = make([]string, len(tags))
			for i, t := range tags {
				item.Tags[i] = t.Name
			}
		}

		items = append(items, item)
	}

	// JSON output
	if jsonOut {
		data, _ := json.MarshalIndent(items, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Text output
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("Found %d prompt(s):\n\n", len(items))

	for _, item := range items {
		fmt.Printf("  %s@%s\n", cyan(item.Name), item.Version)
		if item.Description != "" {
			fmt.Printf("    %s\n", dim(item.Description))
		}
		if len(item.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", green(strings.Join(item.Tags, ", ")))
		}
	}

	return nil
}
