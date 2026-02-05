package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/api"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long: `Start a local HTTP server that exposes the PromptSmith API.

This allows the web UI to connect to your local project and display
real data instead of mock data.

Examples:
  promptsmith serve              # Start on default port 3001
  promptsmith serve --port 8080  # Start on custom port`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 3001, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	server := api.NewServer(database, projectRoot)

	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	addr := fmt.Sprintf(":%d", servePort)
	fmt.Printf("%s API server started\n", cyan("▶"))
	fmt.Printf("  Local:   %s\n", cyan(fmt.Sprintf("http://localhost:%d", servePort)))
	fmt.Printf("  Project: %s\n", dim(projectRoot))
	fmt.Printf("\n%s\n", dim("Press Ctrl+C to stop"))

	return server.ListenAndServe(addr)
}
