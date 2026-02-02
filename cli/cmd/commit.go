package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/prompt"
	"github.com/promptsmith/cli/internal/scanner"
	"github.com/spf13/cobra"
)

var (
	commitMessage string
	commitAll     bool
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Record changes to prompts",
	Long:  `Create a new version for all prompts that have changed since the last commit.`,
	RunE:  runCommit,
}

func init() {
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "commit message (required)")
	commitCmd.Flags().BoolVarP(&commitAll, "all", "a", false, "commit all tracked prompts")
	commitCmd.MarkFlagRequired("message")
	rootCmd.AddCommand(commitCmd)
}

func runCommit(cmd *cobra.Command, args []string) error {
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

	// Get all tracked prompts
	prompts, err := database.ListPrompts()
	if err != nil {
		return err
	}

	if len(prompts) == 0 {
		return fmt.Errorf("no prompts tracked. Use 'promptsmith add <file>' to track a prompt")
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	var committed int
	secretScanner := scanner.New()

	for _, p := range prompts {
		// Read current file content
		absPath := filepath.Join(projectRoot, p.FilePath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("%s %s: file not found\n", yellow("!"), p.Name)
				continue
			}
			return fmt.Errorf("failed to read %s: %w", p.FilePath, err)
		}

		// Get latest version
		latest, err := database.GetLatestVersion(p.ID)
		if err != nil {
			return err
		}

		// Check if content changed
		if latest != nil && latest.Content == string(content) {
			if verbose {
				fmt.Printf("  %s: no changes\n", p.Name)
			}
			continue
		}

		// Scan for secrets
		secrets := secretScanner.Scan(string(content))
		if len(secrets) > 0 {
			fmt.Printf("\n%s Potential secrets in %s:\n", yellow("⚠"), p.Name)
			for _, s := range secrets {
				fmt.Printf("  Line %d: %s - %s\n", s.Line, s.Type, s.Match)
			}
			fmt.Println()
		}

		// Parse prompt
		parsed, err := prompt.Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", p.FilePath, err)
		}

		// Calculate new version
		newVersion := "1.0.0"
		var parentID *string
		if latest != nil {
			newVersion = bumpVersion(latest.Version)
			parentID = &latest.ID
		}

		// Get current user
		user := os.Getenv("USER")
		if user == "" {
			user = "unknown"
		}

		// Create version
		v, err := database.CreateVersion(
			p.ID,
			newVersion,
			string(content),
			parsed.VariablesJSON(),
			parsed.MetadataJSON(),
			commitMessage,
			user,
			parentID,
		)
		if err != nil {
			return err
		}

		fmt.Printf("%s %s@%s\n", green("✓"), cyan(p.Name), v.Version)
		committed++
	}

	if committed == 0 {
		fmt.Println("No changes to commit.")
	} else {
		fmt.Printf("\n%d prompt(s) committed.\n", committed)
	}

	return nil
}

func bumpVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "1.0.0"
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "1.0.0"
	}

	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts, ".")
}
