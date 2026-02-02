package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var (
	logLimit  int
	logPrompt string
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show commit history",
	Long:  `Display the version history of prompts with commit messages and timestamps.`,
	RunE:  runLog,
}

func init() {
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 10, "number of entries to show")
	logCmd.Flags().StringVarP(&logPrompt, "prompt", "p", "", "filter by prompt name")
	rootCmd.AddCommand(logCmd)
}

type logEntry struct {
	PromptName    string `json:"prompt_name"`
	Version       string `json:"version"`
	CommitMessage string `json:"commit_message"`
	CreatedAt     string `json:"created_at"`
	CreatedBy     string `json:"created_by"`
}

func runLog(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	// Open database
	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	if logPrompt != "" {
		// Show history for specific prompt
		p, err := database.GetPromptByName(logPrompt)
		if err != nil {
			return err
		}
		if p == nil {
			return fmt.Errorf("prompt %s not found", logPrompt)
		}

		versions, err := database.ListVersions(p.ID)
		if err != nil {
			return err
		}

		if jsonOut {
			entries := make([]logEntry, 0, len(versions))
			for i, v := range versions {
				if i >= logLimit {
					break
				}
				entries = append(entries, logEntry{
					PromptName:    p.Name,
					Version:       v.Version,
					CommitMessage: v.CommitMessage,
					CreatedAt:     v.CreatedAt.Format("2006-01-02 15:04:05"),
					CreatedBy:     v.CreatedBy,
				})
			}
			data, _ := json.MarshalIndent(entries, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("History for %s:\n\n", cyan(p.Name))
		for i, v := range versions {
			if i >= logLimit {
				break
			}
			fmt.Printf("%s %s\n", yellow(v.Version), v.CommitMessage)
			fmt.Printf("    %s by %s\n\n", dim(v.CreatedAt.Format("2006-01-02 15:04:05")), v.CreatedBy)
		}
		return nil
	}

	// Show all versions across all prompts
	results, err := database.GetAllVersionsForLog()
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("No commits yet.")
		return nil
	}

	if jsonOut {
		entries := make([]logEntry, 0, len(results))
		for i, r := range results {
			if i >= logLimit {
				break
			}
			entries = append(entries, logEntry{
				PromptName:    r.Prompt.Name,
				Version:       r.Version.Version,
				CommitMessage: r.Version.CommitMessage,
				CreatedAt:     r.Version.CreatedAt.Format("2006-01-02 15:04:05"),
				CreatedBy:     r.Version.CreatedBy,
			})
		}
		data, _ := json.MarshalIndent(entries, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	for i, r := range results {
		if i >= logLimit {
			break
		}
		fmt.Printf("%s@%s %s\n", cyan(r.Prompt.Name), yellow(r.Version.Version), r.Version.CommitMessage)
		fmt.Printf("    %s by %s\n\n", dim(r.Version.CreatedAt.Format("2006-01-02 15:04:05")), r.Version.CreatedBy)
	}

	return nil
}
