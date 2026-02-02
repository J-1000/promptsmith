package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/prompt"
	"github.com/promptsmith/cli/internal/scanner"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <file>",
	Short: "Track a new prompt file",
	Long:  `Add a prompt file to PromptSmith tracking. The file will be parsed and an initial version will be created.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	filePath := args[0]

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

	// Get project
	project, err := database.GetProject()
	if err != nil {
		return err
	}
	if project == nil {
		return fmt.Errorf("no project found in database")
	}

	// Resolve file path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Make path relative to project root
	relPath, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return fmt.Errorf("failed to make path relative: %w", err)
	}

	// Check if file exists
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Check if already tracked
	existing, err := database.GetPromptByPath(relPath)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("prompt %s is already tracked", relPath)
	}

	// Parse prompt file
	parsed, err := prompt.Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse prompt: %w", err)
	}

	// Scan for secrets
	secretScanner := scanner.New()
	secrets := secretScanner.Scan(string(content))
	if len(secrets) > 0 {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("\n%s Potential secrets detected:\n", yellow("⚠"))
		for _, s := range secrets {
			fmt.Printf("  Line %d: %s - %s\n", s.Line, s.Type, s.Match)
		}
		fmt.Printf("\nConsider removing sensitive data before committing.\n\n")
	}

	// Determine prompt name
	promptName := parsed.Name()
	if promptName == "" {
		// Use filename without extension
		base := filepath.Base(relPath)
		ext := filepath.Ext(base)
		promptName = base[:len(base)-len(ext)]
	}

	// Check for name collision
	existingByName, err := database.GetPromptByName(promptName)
	if err != nil {
		return err
	}
	if existingByName != nil {
		return fmt.Errorf("a prompt named %s already exists", promptName)
	}

	// Create prompt entry
	p, err := database.CreatePrompt(project.ID, promptName, parsed.Description(), relPath)
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s Added prompt %s\n", green("✓"), cyan(promptName))
	fmt.Printf("  File: %s\n", relPath)
	if parsed.HasFrontmatter {
		fmt.Printf("  Frontmatter: detected\n")
	}
	if len(parsed.ExtractedVars) > 0 {
		fmt.Printf("  Variables: %v\n", parsed.ExtractedVars)
	}
	fmt.Printf("\nRun %s to create the first version.\n", cyan("promptsmith commit -m \"message\""))

	_ = p // Silence unused warning
	return nil
}
