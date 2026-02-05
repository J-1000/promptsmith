package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/sync"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with PromptSmith cloud",
	Long: `Log in to PromptSmith cloud to enable sync features.

You can authenticate using:
  1. Interactive email/password login
  2. API token (--token flag or PROMPTSMITH_TOKEN env var)

Examples:
  promptsmith login                    # Interactive login
  promptsmith login --token <token>    # Token-based login`,
	RunE: runLogin,
}

var (
	loginToken string
)

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&loginToken, "token", "", "API token for authentication")
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Get config directory (user's home .promptsmith or project-local)
	configDir := getGlobalConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Determine remote URL
	remote := sync.DefaultRemote
	if projectRoot, err := db.FindProjectRoot(); err == nil {
		if config, err := loadConfig(projectRoot); err == nil && config.Sync.Remote != "" {
			remote = config.Sync.Remote
		}
	}

	client := sync.NewClient(remote)

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Token-based login
	if loginToken != "" {
		user, err := client.LoginWithToken(loginToken)
		if err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}

		if err := client.SaveToken(configDir, loginToken); err != nil {
			return err
		}

		fmt.Printf("%s Logged in as %s (%s)\n", green("✓"), cyan(user.Name), user.Email)
		return nil
	}

	// Interactive login
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Log in to %s\n\n", cyan(remote))

	fmt.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read email: %w", err)
	}
	email = strings.TrimSpace(email)

	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()
	password := string(passwordBytes)

	auth, err := client.Login(email, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if err := client.SaveToken(configDir, auth.Token); err != nil {
		return err
	}

	fmt.Printf("\n%s Logged in as %s (%s)\n", green("✓"), cyan(auth.User.Name), auth.User.Email)
	return nil
}

func getGlobalConfigDir() string {
	// Try to use project-local config dir first
	if projectRoot, err := db.FindProjectRoot(); err == nil {
		return filepath.Join(projectRoot, db.ConfigDir)
	}

	// Fall back to user home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return ".promptsmith"
	}
	return filepath.Join(home, ".promptsmith")
}
