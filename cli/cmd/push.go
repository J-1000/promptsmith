package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/sync"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Sync local changes to cloud",
	Long: `Upload local prompts, versions, and tags to PromptSmith cloud.

This command pushes all tracked prompts and their versions to the remote server,
enabling collaboration and backup.

Examples:
  promptsmith push              # Push all changes
  promptsmith push --force      # Force push, overwriting remote conflicts`,
	RunE: runPush,
}

var (
	pushForce bool
)

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVar(&pushForce, "force", false, "Force push, overwriting remote conflicts")
}

func runPush(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("Pushing to %s...\n\n", cyan(remote))

	// Get project
	project, err := database.GetProject()
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return fmt.Errorf("no project found")
	}

	// Get all prompts
	prompts, err := database.ListPrompts()
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	if len(prompts) == 0 {
		fmt.Println("No prompts to push")
		return nil
	}

	// Build push request
	req := &sync.PushRequest{
		Project: sync.Project{
			ID:        project.ID,
			Name:      project.Name,
			Team:      config.Sync.Team,
			CreatedAt: project.CreatedAt,
			UpdatedAt: project.UpdatedAt,
		},
		Prompts:  make([]sync.Prompt, 0),
		Versions: make([]sync.PromptVersion, 0),
		Tags:     make([]sync.Tag, 0),
	}

	// Collect prompts, versions, and tags
	for _, p := range prompts {
		req.Prompts = append(req.Prompts, sync.Prompt{
			ID:          p.ID,
			ProjectID:   p.ProjectID,
			Name:        p.Name,
			Description: p.Description,
			FilePath:    p.FilePath,
			CreatedAt:   p.CreatedAt,
		})

		// Get versions for this prompt
		versions, err := database.ListVersions(p.ID)
		if err != nil {
			return fmt.Errorf("failed to list versions for %s: %w", p.Name, err)
		}

		for _, v := range versions {
			req.Versions = append(req.Versions, sync.PromptVersion{
				ID:              v.ID,
				PromptID:        v.PromptID,
				Version:         v.Version,
				Content:         v.Content,
				Variables:       v.Variables,
				Metadata:        v.Metadata,
				ParentVersionID: v.ParentVersionID,
				CommitMessage:   v.CommitMessage,
				CreatedAt:       v.CreatedAt,
				CreatedBy:       v.CreatedBy,
			})
		}

		// Get tags for this prompt
		tags, err := database.ListTags(p.ID)
		if err != nil {
			return fmt.Errorf("failed to list tags for %s: %w", p.Name, err)
		}

		for _, t := range tags {
			req.Tags = append(req.Tags, sync.Tag{
				ID:        t.ID,
				PromptID:  t.PromptID,
				VersionID: t.VersionID,
				Name:      t.Name,
				CreatedAt: t.CreatedAt,
			})
		}
	}

	// Push to remote
	resp, err := client.Push(req)
	if err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	// Report results
	fmt.Printf("%s Pushed %d prompt(s) with %d version(s) and %d tag(s)\n",
		green("✓"), len(req.Prompts), len(req.Versions), len(req.Tags))

	if len(resp.Conflicts) > 0 {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("\n%s Conflicts detected:\n", yellow("⚠"))
		for _, conflict := range resp.Conflicts {
			fmt.Printf("  %s\n", conflict)
		}
		fmt.Printf("\nUse %s to overwrite remote changes\n", cyan("promptsmith push --force"))
	}

	if resp.Message != "" {
		fmt.Printf("\n%s\n", dim(resp.Message))
	}

	return nil
}
