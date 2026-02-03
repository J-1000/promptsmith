package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout <prompt> <version|tag>",
	Short: "Switch to a different version",
	Long: `Restore a prompt file to a specific version.

This updates the working file to match the specified version.
You can reference versions by version number, tag name, or HEAD notation.

Examples:
  promptsmith checkout summarizer 1.0.0      # Checkout version 1.0.0
  promptsmith checkout summarizer prod       # Checkout tagged version
  promptsmith checkout summarizer HEAD~2     # Checkout 2 versions back`,
	Args: cobra.ExactArgs(2),
	RunE: runCheckout,
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}

func runCheckout(cmd *cobra.Command, args []string) error {
	promptName := args[0]
	ref := args[1]

	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	p, err := database.GetPromptByName(promptName)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("prompt '%s' not found", promptName)
	}

	versions, err := database.ListVersions(p.ID)
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		return fmt.Errorf("no versions found for prompt '%s'", promptName)
	}

	// Try to resolve the reference
	targetVersion, err := resolveCheckoutRef(database, p.ID, versions, ref)
	if err != nil {
		return err
	}
	if targetVersion == nil {
		return fmt.Errorf("version or tag '%s' not found", ref)
	}

	// Get absolute path to prompt file
	absPath := filepath.Join(projectRoot, p.FilePath)

	// Check if file has uncommitted changes
	currentContent, err := os.ReadFile(absPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read current file: %w", err)
	}

	latest := versions[0]
	if err == nil && string(currentContent) != latest.Content {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("%s Warning: You have uncommitted changes in %s\n", yellow("!"), p.FilePath)
		fmt.Println("  Use 'promptsmith commit' to save changes before checkout,")
		fmt.Println("  or use '--force' to discard changes (not implemented yet).")
		return fmt.Errorf("uncommitted changes would be overwritten")
	}

	// Write the version content to file
	err = os.WriteFile(absPath, []byte(targetVersion.Content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s Checked out %s@%s\n", green("✓"), cyan(p.Name), targetVersion.Version)

	return nil
}

func resolveCheckoutRef(database *db.DB, promptID string, versions []*db.PromptVersion, ref string) (*db.PromptVersion, error) {
	// Try HEAD notation first
	headRegex := regexp.MustCompile(`^HEAD(~(\d+))?$`)
	if matches := headRegex.FindStringSubmatch(ref); matches != nil {
		offset := 0
		if matches[2] != "" {
			var err error
			offset, err = strconv.Atoi(matches[2])
			if err != nil {
				return nil, fmt.Errorf("invalid HEAD offset: %s", ref)
			}
		}
		if offset >= len(versions) {
			return nil, fmt.Errorf("HEAD~%d is beyond version history (only %d versions)", offset, len(versions))
		}
		return versions[offset], nil
	}

	// Try as version string
	v, err := database.GetVersionByString(promptID, ref)
	if err != nil {
		return nil, err
	}
	if v != nil {
		return v, nil
	}

	// Try as tag name
	tag, err := database.GetTagByName(promptID, ref)
	if err != nil {
		return nil, err
	}
	if tag != nil {
		return database.GetVersionByID(tag.VersionID)
	}

	return nil, nil
}
