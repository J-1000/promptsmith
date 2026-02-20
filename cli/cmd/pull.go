package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/sync"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Fetch latest from cloud",
	Long: `Download remote changes from PromptSmith cloud.

This command pulls all prompts, versions, and tags from the remote server,
updating your local project with any changes from collaborators.

Examples:
  promptsmith pull              # Pull all changes
  promptsmith pull --force      # Force pull, overwriting local changes`,
	RunE: runPull,
}

var (
	pullForce bool
)

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().BoolVar(&pullForce, "force", false, "Force pull, overwriting local changes")
}

func runPull(cmd *cobra.Command, args []string) error {
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	config, err := loadConfig(projectRoot)
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	// Determine remote URL
	remote := sync.DefaultRemote
	if config.Sync.Remote != "" {
		remote = config.Sync.Remote
	}

	client := sync.NewClient(remote)

	// Load auth token
	configDir := getGlobalConfigDir()
	if err := client.LoadToken(configDir); err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Printf("Pulling from %s...\n\n", cyan(remote))

	// Get local project
	project, err := database.GetProject()
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return fmt.Errorf("no project found")
	}

	// Pull from remote
	resp, err := client.Pull(project.ID, nil)
	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	// Track counts
	var promptsAdded, promptsUpdated int
	var versionsAdded int
	var tagsAdded int

	// Sync prompts
	for _, rp := range resp.Prompts {
		safeFilePath, err := safeProjectPath(projectRoot, rp.FilePath)
		if err != nil {
			return fmt.Errorf("invalid remote path for prompt %s: %w", rp.Name, err)
		}

		localPrompt, err := database.GetPromptByName(rp.Name)
		if err != nil {
			return fmt.Errorf("failed to check prompt %s: %w", rp.Name, err)
		}

		if localPrompt == nil {
			// Create new prompt
			_, err := database.CreatePrompt(project.ID, rp.Name, rp.Description, rp.FilePath)
			if err != nil {
				return fmt.Errorf("failed to create prompt %s: %w", rp.Name, err)
			}
			promptsAdded++

			// Create the prompt file if it doesn't exist
			if _, err := os.Stat(safeFilePath); os.IsNotExist(err) {
				// Find the latest version content
				for _, v := range resp.Versions {
					if v.PromptID == rp.ID {
						if err := os.MkdirAll(filepath.Dir(safeFilePath), 0755); err != nil {
							return fmt.Errorf("failed to create directory for %s: %w", rp.Name, err)
						}
						if err := os.WriteFile(safeFilePath, []byte(v.Content), 0644); err != nil {
							return fmt.Errorf("failed to write prompt file %s: %w", rp.Name, err)
						}
						break
					}
				}
			}
		} else {
			promptsUpdated++
		}
	}

	// Sync versions
	for _, rv := range resp.Versions {
		// Get local prompt by finding matching remote prompt
		var promptName string
		for _, rp := range resp.Prompts {
			if rp.ID == rv.PromptID {
				promptName = rp.Name
				break
			}
		}
		if promptName == "" {
			continue
		}

		localPrompt, err := database.GetPromptByName(promptName)
		if err != nil || localPrompt == nil {
			continue
		}

		// Check if version already exists
		existingVersion, err := database.GetVersionByString(localPrompt.ID, rv.Version)
		if err != nil {
			return fmt.Errorf("failed to check version %s: %w", rv.Version, err)
		}

		if existingVersion == nil {
			// Create version
			_, err := database.CreateVersion(
				localPrompt.ID,
				rv.Version,
				rv.Content,
				rv.Variables,
				rv.Metadata,
				rv.CommitMessage,
				rv.CreatedBy,
				rv.ParentVersionID,
			)
			if err != nil {
				return fmt.Errorf("failed to create version %s: %w", rv.Version, err)
			}
			versionsAdded++
		}
	}

	// Sync tags
	for _, rt := range resp.Tags {
		// Get local prompt by finding matching remote prompt
		var promptName string
		for _, rp := range resp.Prompts {
			if rp.ID == rt.PromptID {
				promptName = rp.Name
				break
			}
		}
		if promptName == "" {
			continue
		}

		localPrompt, err := database.GetPromptByName(promptName)
		if err != nil || localPrompt == nil {
			continue
		}

		// Check if tag already exists
		existingTag, err := database.GetTagByName(localPrompt.ID, rt.Name)
		if err != nil {
			return fmt.Errorf("failed to check tag %s: %w", rt.Name, err)
		}

		if existingTag == nil {
			// Find matching local version
			var versionName string
			for _, rv := range resp.Versions {
				if rv.ID == rt.VersionID {
					versionName = rv.Version
					break
				}
			}
			if versionName == "" {
				continue
			}

			localVersion, err := database.GetVersionByString(localPrompt.ID, versionName)
			if err != nil || localVersion == nil {
				continue
			}

			_, err = database.CreateTag(localPrompt.ID, localVersion.ID, rt.Name)
			if err != nil {
				return fmt.Errorf("failed to create tag %s: %w", rt.Name, err)
			}
			tagsAdded++
		}
	}

	// Report results
	if promptsAdded == 0 && versionsAdded == 0 && tagsAdded == 0 {
		fmt.Printf("%s Already up to date\n", green("✓"))
	} else {
		fmt.Printf("%s Pulled changes:\n", green("✓"))
		if promptsAdded > 0 {
			fmt.Printf("  %d new prompt(s)\n", promptsAdded)
		}
		if promptsUpdated > 0 {
			fmt.Printf("  %d prompt(s) checked\n", promptsUpdated)
		}
		if versionsAdded > 0 {
			fmt.Printf("  %d new version(s)\n", versionsAdded)
		}
		if tagsAdded > 0 {
			fmt.Printf("  %d new tag(s)\n", tagsAdded)
		}
	}

	if resp.Message != "" {
		fmt.Printf("\n%s\n", dim(resp.Message))
	}

	return nil
}
