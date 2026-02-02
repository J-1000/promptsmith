package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new PromptSmith project",
	Long:  `Creates a new PromptSmith project in the current directory with version control for prompts.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

type Config struct {
	Version     int            `yaml:"version"`
	Project     ProjectConfig  `yaml:"project"`
	PromptsDir  string         `yaml:"prompts_dir"`
	TestsDir    string         `yaml:"tests_dir"`
	BenchmarksDir string       `yaml:"benchmarks_dir"`
	Defaults    DefaultsConfig `yaml:"defaults"`
}

type ProjectConfig struct {
	Name string `yaml:"name"`
	ID   string `yaml:"id"`
}

type DefaultsConfig struct {
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if already initialized
	configDir := filepath.Join(cwd, db.ConfigDir)
	if _, err := os.Stat(configDir); err == nil {
		return fmt.Errorf("project already initialized in %s", cwd)
	}

	// Determine project name
	projectName := filepath.Base(cwd)
	if len(args) > 0 {
		projectName = args[0]
	}

	// Initialize database
	database, err := db.Initialize(cwd)
	if err != nil {
		return err
	}
	defer database.Close()

	// Create project
	project, err := database.CreateProject(projectName)
	if err != nil {
		return err
	}

	// Create config file
	config := Config{
		Version: 1,
		Project: ProjectConfig{
			Name: projectName,
			ID:   project.ID,
		},
		PromptsDir:    "./prompts",
		TestsDir:      "./tests",
		BenchmarksDir: "./benchmarks",
		Defaults: DefaultsConfig{
			Model:       "gpt-4o",
			Temperature: 0.7,
		},
	}

	configPath := filepath.Join(configDir, db.ConfigFile)
	configData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Create default directories
	dirs := []string{
		filepath.Join(cwd, "prompts"),
		filepath.Join(cwd, "tests"),
		filepath.Join(cwd, "benchmarks"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create .gitignore for .promptsmith
	gitignorePath := filepath.Join(configDir, ".gitignore")
	gitignoreContent := "# PromptSmith database\npromptsmith.db\n"
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	// Output success
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s Initialized PromptSmith project %s\n", green("✓"), cyan(projectName))
	fmt.Printf("\nCreated:\n")
	fmt.Printf("  %s/\n", db.ConfigDir)
	fmt.Printf("  prompts/\n")
	fmt.Printf("  tests/\n")
	fmt.Printf("  benchmarks/\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Create a prompt file in prompts/\n")
	fmt.Printf("  2. Run %s to track it\n", cyan("promptsmith add <file>"))
	fmt.Printf("  3. Run %s to commit changes\n", cyan("promptsmith commit -m \"message\""))

	return nil
}
