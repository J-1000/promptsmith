package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status",
	Long: `Show the current status of the PromptSmith project.

Displays tracked prompts, their versions, and whether they have
uncommitted changes.

Examples:
  promptsmith status`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

type promptStatus struct {
	Name        string `json:"name"`
	FilePath    string `json:"file_path"`
	Version     string `json:"version"`
	Status      string `json:"status"` // clean, modified, untracked
	Description string `json:"description,omitempty"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	project, err := database.GetProject()
	if err != nil {
		return err
	}

	// Get all tracked prompts
	prompts, err := database.ListPrompts()
	if err != nil {
		return err
	}

	// Find prompt files in prompts/ directory
	promptsDir := filepath.Join(projectRoot, "prompts")
	var untrackedFiles []string

	if _, err := os.Stat(promptsDir); err == nil {
		matches, _ := filepath.Glob(filepath.Join(promptsDir, "*.prompt"))
		for _, m := range matches {
			relPath, _ := filepath.Rel(projectRoot, m)
			found := false
			for _, p := range prompts {
				if p.FilePath == relPath || p.FilePath == m {
					found = true
					break
				}
			}
			if !found {
				untrackedFiles = append(untrackedFiles, relPath)
			}
		}
	}

	var statuses []promptStatus

	// Check each tracked prompt
	for _, p := range prompts {
		ps := promptStatus{
			Name:        p.Name,
			FilePath:    p.FilePath,
			Description: p.Description,
			Status:      "clean",
		}

		// Get latest version
		latestVersion, err := database.GetLatestVersion(p.ID)
		if err == nil && latestVersion != nil {
			ps.Version = latestVersion.Version

			// Check if file has changed
			fullPath := filepath.Join(projectRoot, p.FilePath)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				ps.Status = "deleted"
			} else {
				// Read current file content
				fileContent, err := os.ReadFile(fullPath)
				if err == nil {
					// Compare content hashes (full file content)
					currentHash := hashContent(string(fileContent))
					storedHash := hashContent(latestVersion.Content)
					if currentHash != storedHash {
						ps.Status = "modified"
					}
				}
			}
		} else {
			ps.Version = "0.0.0"
			ps.Status = "new"
		}

		statuses = append(statuses, ps)
	}

	// JSON output
	if jsonOut {
		output := struct {
			Project   string         `json:"project"`
			Prompts   []promptStatus `json:"prompts"`
			Untracked []string       `json:"untracked,omitempty"`
		}{
			Project:   project.Name,
			Prompts:   statuses,
			Untracked: untrackedFiles,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Text output
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Printf("Project: %s\n\n", cyan(project.Name))

	if len(statuses) == 0 && len(untrackedFiles) == 0 {
		fmt.Println("No prompts tracked yet.")
		fmt.Printf("Use %s to start tracking prompts.\n", cyan("promptsmith add <file>"))
		return nil
	}

	if len(statuses) > 0 {
		fmt.Println("Tracked prompts:")
		for _, ps := range statuses {
			var statusIcon, statusColor string
			switch ps.Status {
			case "clean":
				statusIcon = green("✓")
				statusColor = ""
			case "modified":
				statusIcon = yellow("M")
				statusColor = yellow(ps.Status)
			case "deleted":
				statusIcon = red("D")
				statusColor = red(ps.Status)
			case "new":
				statusIcon = green("N")
				statusColor = green(ps.Status)
			}

			fmt.Printf("  %s %s@%s", statusIcon, ps.Name, dim(ps.Version))
			if statusColor != "" {
				fmt.Printf(" %s", statusColor)
			}
			fmt.Println()
		}
	}

	if len(untrackedFiles) > 0 {
		fmt.Println("\nUntracked files:")
		for _, f := range untrackedFiles {
			fmt.Printf("  %s %s\n", red("?"), f)
		}
		fmt.Printf("\nUse %s to track files.\n", cyan("promptsmith add <file>"))
	}

	// Count modified
	modified := 0
	for _, ps := range statuses {
		if ps.Status == "modified" || ps.Status == "new" {
			modified++
		}
	}

	if modified > 0 {
		fmt.Printf("\n%d prompt(s) with uncommitted changes.\n", modified)
		fmt.Printf("Use %s to commit.\n", cyan("promptsmith commit -m \"message\""))
	}

	return nil
}

func hashContent(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}
