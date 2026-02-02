package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool
	jsonOut bool
)

var rootCmd = &cobra.Command{
	Use:   "promptsmith",
	Short: "The GitHub Copilot for Prompt Engineering",
	Long: `PromptSmith brings software engineering best practices to prompt engineering.
Version, test, iterate, and benchmark your LLM prompts with the same rigor you apply to code.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output as JSON")
}
