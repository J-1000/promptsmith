package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var (
	tagDelete bool
	tagList   bool
)

var tagCmd = &cobra.Command{
	Use:   "tag <prompt> <tag-name> [version]",
	Short: "Create, list, or delete tags",
	Long: `Manage tags for prompt versions.

Tags are named references to specific versions, useful for marking
releases or environments (prod, staging, etc.).

Examples:
  promptsmith tag summarizer prod              # Tag latest version as 'prod'
  promptsmith tag summarizer v1.0 1.0.0        # Tag version 1.0.0 as 'v1.0'
  promptsmith tag summarizer staging HEAD~1   # Tag previous version
  promptsmith tag summarizer --list            # List all tags
  promptsmith tag summarizer prod --delete     # Delete tag`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runTag,
}

func init() {
	tagCmd.Flags().BoolVarP(&tagDelete, "delete", "d", false, "delete the specified tag")
	tagCmd.Flags().BoolVarP(&tagList, "list", "l", false, "list all tags for the prompt")
	rootCmd.AddCommand(tagCmd)
}

type tagOutput struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	CreatedAt string `json:"created_at"`
}

func runTag(cmd *cobra.Command, args []string) error {
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

	p, err := database.GetPromptByName(promptName)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("prompt '%s' not found", promptName)
	}

	// List tags
	if tagList {
		return listTags(database, p)
	}

	// Need tag name for create/delete
	if len(args) < 2 {
		return fmt.Errorf("tag name required")
	}
	tagName := args[1]

	// Delete tag
	if tagDelete {
		return deleteTag(database, p, tagName)
	}

	// Create/update tag
	versions, err := database.ListVersions(p.ID)
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		return fmt.Errorf("no versions found for prompt '%s'", promptName)
	}

	var targetVersion *db.PromptVersion
	if len(args) == 3 {
		// Specific version provided
		targetVersion, err = resolveVersionForTag(database, p.ID, versions, args[2])
		if err != nil {
			return err
		}
		if targetVersion == nil {
			return fmt.Errorf("version '%s' not found", args[2])
		}
	} else {
		// Default to latest
		targetVersion = versions[0]
	}

	return createTag(database, p, tagName, targetVersion)
}

func listTags(database *db.DB, p *db.Prompt) error {
	tags, err := database.ListTags(p.ID)
	if err != nil {
		return err
	}

	if len(tags) == 0 {
		fmt.Printf("No tags for %s\n", p.Name)
		return nil
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	if jsonOut {
		var outputs []tagOutput
		for _, t := range tags {
			v, _ := database.GetVersionByID(t.VersionID)
			version := "unknown"
			if v != nil {
				version = v.Version
			}
			outputs = append(outputs, tagOutput{
				Name:      t.Name,
				Version:   version,
				CreatedAt: t.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		data, _ := json.MarshalIndent(outputs, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Tags for %s:\n\n", cyan(p.Name))
	for _, t := range tags {
		v, _ := database.GetVersionByID(t.VersionID)
		version := "unknown"
		if v != nil {
			version = v.Version
		}
		fmt.Printf("  %s -> %s  %s\n", yellow(t.Name), version, dim(t.CreatedAt.Format("2006-01-02")))
	}
	return nil
}

func deleteTag(database *db.DB, p *db.Prompt, tagName string) error {
	err := database.DeleteTag(p.ID, tagName)
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Deleted tag '%s' from %s\n", green("✓"), tagName, p.Name)
	return nil
}

func createTag(database *db.DB, p *db.Prompt, tagName string, v *db.PromptVersion) error {
	_, err := database.CreateTag(p.ID, v.ID, tagName)
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s Tagged %s@%s as '%s'\n", green("✓"), cyan(p.Name), v.Version, tagName)
	return nil
}

func resolveVersionForTag(database *db.DB, promptID string, versions []*db.PromptVersion, ref string) (*db.PromptVersion, error) {
	// Handle HEAD notation
	headRegex := regexp.MustCompile(`^HEAD(~(\d+))?$`)
	if matches := headRegex.FindStringSubmatch(ref); matches != nil {
		offset := 0
		if matches[2] != "" {
			var err error
			offset, err = strconv.Atoi(matches[2])
			if err != nil {
				return nil, fmt.Errorf("invalid HEAD offset: %s", ref)
			}
		}
		if offset >= len(versions) {
			return nil, fmt.Errorf("HEAD~%d is beyond version history (only %d versions)", offset, len(versions))
		}
		return versions[offset], nil
	}

	// Try as version string
	v, err := database.GetVersionByString(promptID, ref)
	if err != nil {
		return nil, err
	}
	return v, nil
}
