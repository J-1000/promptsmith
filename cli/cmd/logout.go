package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/sync"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from PromptSmith cloud",
	Long:  `Remove stored authentication credentials.`,
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	configDir := getGlobalConfigDir()

	// Determine remote URL
	remote := sync.DefaultRemote
	if projectRoot, err := db.FindProjectRoot(); err == nil {
		if config, err := loadConfig(projectRoot); err == nil && config.Sync.Remote != "" {
			remote = config.Sync.Remote
		}
	}

	client := sync.NewClient(remote)

	// Try to load token and logout from server
	if err := client.LoadToken(configDir); err == nil {
		_ = client.Logout() // Ignore errors, we're logging out anyway
	}

	// Delete local token
	if err := client.DeleteToken(configDir); err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Logged out successfully\n", green("✓"))
	return nil
}
