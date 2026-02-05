package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Manage project configuration",
	Long: `View and modify PromptSmith project configuration.

Examples:
  promptsmith config                    # List all config
  promptsmith config get defaults.model # Get specific value
  promptsmith config set defaults.model gpt-4o-mini
  promptsmith config set defaults.temperature 0.5`,
	Args: cobra.MaximumNArgs(2),
	RunE: runConfig,
}

var (
	configGetFlag bool
	configSetFlag bool
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVar(&configGetFlag, "get", false, "Get a config value")
	configCmd.Flags().BoolVar(&configSetFlag, "set", false, "Set a config value")
}

func loadConfig(projectRoot string) (*Config, error) {
	configPath := filepath.Join(projectRoot, db.ConfigDir, db.ConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func saveConfig(projectRoot string, config *Config) error {
	configPath := filepath.Join(projectRoot, db.ConfigDir, db.ConfigFile)
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func getConfigValue(config *Config, key string) (string, error) {
	parts := strings.Split(key, ".")

	switch parts[0] {
	case "version":
		return strconv.Itoa(config.Version), nil
	case "project":
		if len(parts) < 2 {
			return "", fmt.Errorf("specify project.name or project.id")
		}
		switch parts[1] {
		case "name":
			return config.Project.Name, nil
		case "id":
			return config.Project.ID, nil
		default:
			return "", fmt.Errorf("unknown project key: %s", parts[1])
		}
	case "prompts_dir":
		return config.PromptsDir, nil
	case "tests_dir":
		return config.TestsDir, nil
	case "benchmarks_dir":
		return config.BenchmarksDir, nil
	case "defaults":
		if len(parts) < 2 {
			return "", fmt.Errorf("specify defaults.model or defaults.temperature")
		}
		switch parts[1] {
		case "model":
			return config.Defaults.Model, nil
		case "temperature":
			return fmt.Sprintf("%.1f", config.Defaults.Temperature), nil
		default:
			return "", fmt.Errorf("unknown defaults key: %s", parts[1])
		}
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func setConfigValue(config *Config, key, value string) error {
	parts := strings.Split(key, ".")

	switch parts[0] {
	case "project":
		if len(parts) < 2 {
			return fmt.Errorf("specify project.name")
		}
		switch parts[1] {
		case "name":
			config.Project.Name = value
		default:
			return fmt.Errorf("cannot set project.%s", parts[1])
		}
	case "prompts_dir":
		config.PromptsDir = value
	case "tests_dir":
		config.TestsDir = value
	case "benchmarks_dir":
		config.BenchmarksDir = value
	case "defaults":
		if len(parts) < 2 {
			return fmt.Errorf("specify defaults.model or defaults.temperature")
		}
		switch parts[1] {
		case "model":
			config.Defaults.Model = value
		case "temperature":
			temp, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid temperature value: %s", value)
			}
			if temp < 0 || temp > 2 {
				return fmt.Errorf("temperature must be between 0 and 2")
			}
			config.Defaults.Temperature = temp
		default:
			return fmt.Errorf("unknown defaults key: %s", parts[1])
		}
	default:
		return fmt.Errorf("unknown or read-only config key: %s", key)
	}

	return nil
}

func runConfig(cmd *cobra.Command, args []string) error {
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	config, err := loadConfig(projectRoot)
	if err != nil {
		return err
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	// No args: list all config
	if len(args) == 0 {
		fmt.Printf("%s\n", cyan("Project Configuration"))
		fmt.Printf("  project.name:       %s\n", config.Project.Name)
		fmt.Printf("  project.id:         %s\n", dim(config.Project.ID))
		fmt.Printf("  prompts_dir:        %s\n", config.PromptsDir)
		fmt.Printf("  tests_dir:          %s\n", config.TestsDir)
		fmt.Printf("  benchmarks_dir:     %s\n", config.BenchmarksDir)
		fmt.Printf("\n%s\n", cyan("Defaults"))
		fmt.Printf("  defaults.model:       %s\n", config.Defaults.Model)
		fmt.Printf("  defaults.temperature: %.1f\n", config.Defaults.Temperature)
		return nil
	}

	// One arg: get value
	if len(args) == 1 {
		value, err := getConfigValue(config, args[0])
		if err != nil {
			return err
		}
		fmt.Println(value)
		return nil
	}

	// Two args: set value
	if len(args) == 2 {
		if err := setConfigValue(config, args[0], args[1]); err != nil {
			return err
		}
		if err := saveConfig(projectRoot, config); err != nil {
			return err
		}
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Set %s = %s\n", green("✓"), args[0], args[1])
		return nil
	}

	return nil
}
