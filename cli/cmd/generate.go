package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/benchmark"
	"github.com/promptsmith/cli/internal/db"
	"github.com/promptsmith/cli/internal/generator"
	"github.com/promptsmith/cli/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	genCount   int
	genGoal    string
	genModel   string
	genType    string
	genOutput  string
	genVersion string
)

var generateCmd = &cobra.Command{
	Use:   "generate <prompt-name>",
	Short: "Generate prompt variations using AI",
	Long: `Generate variations of a prompt using an LLM.

Supports different generation types:
  variations - Create alternative versions with different approaches
  compress   - Reduce token count while preserving functionality
  expand     - Add more detail, examples, and edge case handling
  rephrase   - Reword while keeping the same meaning

Examples:
  promptsmith generate summarizer                    # Generate 3 variations
  promptsmith generate summarizer --count 5         # Generate 5 variations
  promptsmith generate summarizer --type compress   # Compress the prompt
  promptsmith generate summarizer --goal "be concise"
  promptsmith generate summarizer --model gpt-4o`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().IntVarP(&genCount, "count", "c", 3, "number of variations to generate")
	generateCmd.Flags().StringVarP(&genGoal, "goal", "g", "", "specific goal for generation")
	generateCmd.Flags().StringVarP(&genModel, "model", "m", "gpt-4o-mini", "model to use for generation")
	generateCmd.Flags().StringVarP(&genType, "type", "t", "variations", "generation type: variations, compress, expand, rephrase")
	generateCmd.Flags().StringVarP(&genOutput, "output", "o", "", "write results to file (JSON format)")
	generateCmd.Flags().StringVarP(&genVersion, "version", "v", "", "generate from specific prompt version")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	promptName := args[0]

	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	// Get the prompt
	p, err := database.GetPromptByName(promptName)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("prompt '%s' not found", promptName)
	}

	// Get the version
	var version *db.PromptVersion
	if genVersion != "" {
		version, err = database.GetVersionByString(p.ID, genVersion)
		if err != nil {
			return err
		}
		if version == nil {
			return fmt.Errorf("version '%s' not found for prompt '%s'", genVersion, promptName)
		}
	} else {
		version, err = database.GetLatestVersion(p.ID)
		if err != nil {
			return err
		}
		if version == nil {
			return fmt.Errorf("no versions found for prompt '%s'", promptName)
		}
	}

	// Parse the prompt to get content
	parsed, err := prompt.Parse(version.Content)
	if err != nil {
		return fmt.Errorf("failed to parse prompt: %w", err)
	}

	// Get provider
	provider, err := getProvider(genModel)
	if err != nil {
		return err
	}

	// Create generator
	gen := generator.New(provider)

	// Parse generation type
	var genTypeEnum generator.GenerationType
	switch strings.ToLower(genType) {
	case "variations":
		genTypeEnum = generator.TypeVariations
	case "compress":
		genTypeEnum = generator.TypeCompress
	case "expand":
		genTypeEnum = generator.TypeExpand
	case "rephrase":
		genTypeEnum = generator.TypeRephrase
	default:
		return fmt.Errorf("unknown generation type: %s", genType)
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	if !jsonOut {
		fmt.Printf("\n%s Generating %s for %s@%s\n", cyan("▶"), genType, promptName, version.Version)
		fmt.Printf("  Model: %s\n", genModel)
		fmt.Printf("  Count: %d\n", genCount)
		if genGoal != "" {
			fmt.Printf("  Goal: %s\n", genGoal)
		}
		fmt.Println()
	}

	// Generate variations
	result, err := gen.Generate(context.Background(), generator.GenerateRequest{
		Type:   genTypeEnum,
		Prompt: parsed.Content,
		Count:  genCount,
		Goal:   genGoal,
		Model:  genModel,
	})
	if err != nil {
		return err
	}

	// Output results
	if jsonOut {
		data, _ := json.MarshalIndent(result, "", "  ")
		if genOutput != "" {
			if err := os.WriteFile(genOutput, data, 0644); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
			fmt.Printf("Results written to %s\n", genOutput)
		} else {
			fmt.Println(string(data))
		}
	} else {
		originalTokens := generator.EstimateTokens(result.Original)

		for i, v := range result.Variations {
			tokens := generator.EstimateTokens(v.Content)
			tokenDelta := tokens - originalTokens
			deltaStr := ""
			if tokenDelta > 0 {
				deltaStr = fmt.Sprintf(" (+%d tokens)", tokenDelta)
			} else if tokenDelta < 0 {
				deltaStr = fmt.Sprintf(" (%d tokens)", tokenDelta)
			}

			fmt.Printf("%s Variation %d%s\n", green("●"), i+1, dim(deltaStr))
			if v.Description != "" {
				fmt.Printf("  %s\n", dim(v.Description))
			}
			fmt.Println()
			fmt.Println("  " + strings.ReplaceAll(v.Content, "\n", "\n  "))
			fmt.Println()
			fmt.Println(dim(strings.Repeat("─", 60)))
			fmt.Println()
		}

		fmt.Printf("Generated %d %s using %s\n", len(result.Variations), genType, genModel)

		if genOutput != "" {
			data, _ := json.MarshalIndent(result, "", "  ")
			if err := os.WriteFile(genOutput, data, 0644); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
			fmt.Printf("Results written to %s\n", genOutput)
		}
	}

	return nil
}

func getProvider(model string) (benchmark.Provider, error) {
	providerName := benchmark.GetProviderForModel(model)

	switch providerName {
	case "openai":
		return benchmark.NewOpenAIProvider()
	case "anthropic":
		return benchmark.NewAnthropicProvider()
	default:
		return nil, fmt.Errorf("unsupported model: %s (provider: %s)", model, providerName)
	}
}
