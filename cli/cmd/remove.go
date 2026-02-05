package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:     "remove <prompt>",
	Aliases: []string{"rm"},
	Short:   "Remove a prompt from tracking",
	Long: `Stop tracking a prompt and optionally delete all its version history.

This does NOT delete the prompt file, only removes it from PromptSmith tracking.

Examples:
  promptsmith remove summarizer
  promptsmith rm summarizer
  promptsmith remove summarizer --force  # Skip confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "skip confirmation")
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
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

	// Get version count for confirmation message
	versions, err := database.ListVersions(p.ID)
	if err != nil {
		return err
	}

	// Confirm unless --force
	if !removeForce {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("%s This will remove '%s' from tracking.\n", yellow("⚠"), promptName)
		fmt.Printf("  %d version(s) will be deleted from the database.\n", len(versions))
		fmt.Printf("  The file '%s' will NOT be deleted.\n\n", p.FilePath)
		fmt.Print("Continue? [y/N] ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Delete tags first (foreign key constraint)
	_, err = database.Exec("DELETE FROM tags WHERE prompt_id = ?", p.ID)
	if err != nil {
		return fmt.Errorf("failed to delete tags: %w", err)
	}

	// Delete versions
	_, err = database.Exec("DELETE FROM prompt_versions WHERE prompt_id = ?", p.ID)
	if err != nil {
		return fmt.Errorf("failed to delete versions: %w", err)
	}

	// Delete prompt
	_, err = database.Exec("DELETE FROM prompts WHERE id = ?", p.ID)
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Removed '%s' from tracking\n", green("✓"), promptName)
	fmt.Printf("  %d version(s) deleted.\n", len(versions))

	return nil
}
