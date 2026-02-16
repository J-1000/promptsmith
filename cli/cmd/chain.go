package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/benchmark"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var (
	chainDescription string
	chainInputs      []string
	chainModel       string
)

var chainCmd = &cobra.Command{
	Use:   "chain",
	Short: "Manage prompt chains / pipelines",
	Long: `Create and run prompt chains where the output of one step feeds into the next.

Examples:
  promptsmith chain list
  promptsmith chain create summarize-translate --description "Summarize then translate"
  promptsmith chain show summarize-translate
  promptsmith chain run summarize-translate --input text="Hello world" --model gpt-4o-mini`,
}

var chainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all chains",
	RunE:  runChainList,
}

var chainCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new chain",
	Args:  cobra.ExactArgs(1),
	RunE:  runChainCreate,
}

var chainShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show chain details and steps",
	Args:  cobra.ExactArgs(1),
	RunE:  runChainShow,
}

var chainRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Execute a chain",
	Args:  cobra.ExactArgs(1),
	RunE:  runChainRun,
}

func init() {
	chainCreateCmd.Flags().StringVarP(&chainDescription, "description", "d", "", "chain description")
	chainRunCmd.Flags().StringSliceVarP(&chainInputs, "input", "i", nil, "input key=value pairs")
	chainRunCmd.Flags().StringVarP(&chainModel, "model", "m", "gpt-4o-mini", "model to use for all steps")

	chainCmd.AddCommand(chainListCmd)
	chainCmd.AddCommand(chainCreateCmd)
	chainCmd.AddCommand(chainShowCmd)
	chainCmd.AddCommand(chainRunCmd)
	rootCmd.AddCommand(chainCmd)
}

func runChainList(cmd *cobra.Command, args []string) error {
	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	chains, err := database.ListChains()
	if err != nil {
		return err
	}

	if jsonOut {
		data, _ := json.MarshalIndent(chains, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(chains) == 0 {
		fmt.Println("No chains found.")
		fmt.Println("Create one with: promptsmith chain create <name>")
		return nil
	}

	dim := color.New(color.Faint).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("\n%s\n", cyan("Chains"))
	fmt.Printf("%s\n", dim(strings.Repeat("─", 50)))

	for _, c := range chains {
		steps, _ := database.ListChainSteps(c.ID)
		desc := c.Description
		if desc == "" {
			desc = dim("no description")
		}
		fmt.Printf("  %-25s %s  (%d steps)\n", cyan(c.Name), desc, len(steps))
	}
	fmt.Println()
	return nil
}

func runChainCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	existing, err := database.GetChainByName(name)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("chain '%s' already exists", name)
	}

	project, err := database.GetProject()
	if err != nil {
		return err
	}
	if project == nil {
		return fmt.Errorf("no project found — run 'promptsmith init' first")
	}

	chain, err := database.CreateChain(project.ID, name, chainDescription)
	if err != nil {
		return err
	}

	if jsonOut {
		data, _ := json.MarshalIndent(chain, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Created chain '%s'\n", green("✓"), chain.Name)
	return nil
}

func runChainShow(cmd *cobra.Command, args []string) error {
	name := args[0]

	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	chain, err := database.GetChainByName(name)
	if err != nil {
		return err
	}
	if chain == nil {
		return fmt.Errorf("chain '%s' not found", name)
	}

	steps, err := database.ListChainSteps(chain.ID)
	if err != nil {
		return err
	}

	if jsonOut {
		out := map[string]interface{}{
			"chain": chain,
			"steps": steps,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Printf("\n%s %s\n", cyan("Chain:"), chain.Name)
	if chain.Description != "" {
		fmt.Printf("  %s\n", chain.Description)
	}
	fmt.Printf("  %s\n\n", dim(fmt.Sprintf("Created: %s", chain.CreatedAt.Format("2006-01-02 15:04"))))

	if len(steps) == 0 {
		fmt.Println("  No steps configured.")
		fmt.Println("  Add steps via the web UI or API.")
	} else {
		fmt.Printf("  %s (%d)\n", cyan("Steps"), len(steps))
		fmt.Printf("  %s\n", dim(strings.Repeat("─", 40)))
		for _, s := range steps {
			fmt.Printf("  %d. %s → %s\n", s.StepOrder, cyan(s.PromptName), s.OutputKey)
		}
	}
	fmt.Println()
	return nil
}

func runChainRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	projectRoot, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	database, err := db.Open(projectRoot)
	if err != nil {
		return err
	}
	defer database.Close()

	chain, err := database.GetChainByName(name)
	if err != nil {
		return err
	}
	if chain == nil {
		return fmt.Errorf("chain '%s' not found", name)
	}

	steps, err := database.ListChainSteps(chain.ID)
	if err != nil {
		return err
	}
	if len(steps) == 0 {
		return fmt.Errorf("chain '%s' has no steps — add steps first", name)
	}

	// Parse inputs
	inputs := make(map[string]string)
	for _, kv := range chainInputs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid input format '%s' — use key=value", kv)
		}
		inputs[parts[0]] = parts[1]
	}

	// Create provider
	registry := benchmark.NewProviderRegistry()
	if openai, err := benchmark.NewOpenAIProvider(); err == nil {
		registry.Register(openai)
	}
	if anthropic, err := benchmark.NewAnthropicProvider(); err == nil {
		registry.Register(anthropic)
	}

	provider, err := registry.GetForModel(chainModel)
	if err != nil {
		return fmt.Errorf("model error: %w", err)
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	if !jsonOut {
		fmt.Printf("\n%s Running chain '%s' with %d steps\n", cyan("▶"), chain.Name, len(steps))
		fmt.Printf("  Model: %s\n\n", chainModel)
	}

	stepOutputs := make(map[string]string)
	type stepResult struct {
		Step    int    `json:"step"`
		Prompt  string `json:"prompt"`
		Output  string `json:"output"`
		Key     string `json:"output_key"`
	}
	var results []stepResult

	for _, step := range steps {
		// Resolve inputs
		var inputMap map[string]string
		if err := json.Unmarshal([]byte(step.InputMapping), &inputMap); err != nil {
			inputMap = map[string]string{}
		}

		resolvedVars := make(map[string]any)
		for varName, source := range inputMap {
			resolvedVars[varName] = resolveInput(source, inputs, stepOutputs)
		}

		// Load prompt
		prompt, err := database.GetPromptByName(step.PromptName)
		if err != nil || prompt == nil {
			return fmt.Errorf("step %d: prompt '%s' not found", step.StepOrder, step.PromptName)
		}

		version, err := database.GetLatestVersion(prompt.ID)
		if err != nil || version == nil {
			return fmt.Errorf("step %d: no version for prompt '%s'", step.StepOrder, step.PromptName)
		}

		// Simple template rendering
		rendered := version.Content
		for k, v := range resolvedVars {
			rendered = strings.ReplaceAll(rendered, "{{"+k+"}}", fmt.Sprint(v))
		}

		if !jsonOut {
			fmt.Printf("  %s Step %d: %s\n", dim("→"), step.StepOrder, cyan(step.PromptName))
		}

		resp, err := provider.Complete(context.Background(), benchmark.CompletionRequest{
			Model:       chainModel,
			Prompt:      rendered,
			MaxTokens:   1024,
			Temperature: 1.0,
		})
		if err != nil {
			return fmt.Errorf("step %d failed: %w", step.StepOrder, err)
		}

		stepOutputs[step.OutputKey] = resp.Content
		results = append(results, stepResult{
			Step:   step.StepOrder,
			Prompt: step.PromptName,
			Output: resp.Content,
			Key:    step.OutputKey,
		})

		if !jsonOut {
			// Show truncated output
			output := resp.Content
			if len(output) > 200 {
				output = output[:200] + "..."
			}
			fmt.Printf("    %s\n\n", dim(output))
		}
	}

	// Save run
	inputsJSON, _ := json.Marshal(inputs)
	resultsJSON, _ := json.Marshal(results)
	finalOutput := stepOutputs[steps[len(steps)-1].OutputKey]
	database.SaveChainRun(chain.ID, "completed", string(inputsJSON), string(resultsJSON), finalOutput)

	if jsonOut {
		out := map[string]interface{}{
			"status":       "completed",
			"steps":        results,
			"final_output": finalOutput,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("  %s Chain completed\n", green("✓"))
		fmt.Printf("  %s\n\n", dim("Final output:"))
		fmt.Println(finalOutput)
	}

	return nil
}

func resolveInput(source string, inputs map[string]string, stepOutputs map[string]string) string {
	if strings.HasPrefix(source, "{{input.") && strings.HasSuffix(source, "}}") {
		key := source[8 : len(source)-2]
		return inputs[key]
	}
	if strings.HasPrefix(source, "{{steps.") && strings.HasSuffix(source, "}}") {
		inner := source[8 : len(source)-2]
		if dotIdx := strings.Index(inner, "."); dotIdx > 0 {
			return stepOutputs[inner[:dotIdx]]
		}
	}
	return source
}
