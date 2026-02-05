package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/sync"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user info",
	Long:  `Display the currently logged in user's information.`,
	RunE:  runWhoami,
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

func runWhoami(cmd *cobra.Command, args []string) error {
	configDir := getGlobalConfigDir()

	// Determine remote URL
	remote := sync.DefaultRemote
	if projectRoot, err := db.FindProjectRoot(); err == nil {
		if config, err := loadConfig(projectRoot); err == nil && config.Sync.Remote != "" {
			remote = config.Sync.Remote
		}
	}

	client := sync.NewClient(remote)

	if err := client.LoadToken(configDir); err != nil {
		dim := color.New(color.Faint).SprintFunc()
		fmt.Printf("Not logged in\n")
		fmt.Printf("\n%s\n", dim("Run 'promptsmith login' to authenticate"))
		return nil
	}

	user, err := client.WhoAmI()
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Printf("Logged in as %s\n", cyan(user.Name))
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  ID:    %s\n", dim(user.ID))

	return nil
}
