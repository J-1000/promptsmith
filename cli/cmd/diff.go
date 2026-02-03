package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/promptsmith/cli/internal/db"
	"github.com/spf13/cobra"
)

var (
	diffFormat string
)

var diffCmd = &cobra.Command{
	Use:   "diff <prompt> [version1] [version2]",
	Short: "Show changes between versions",
	Long: `Show differences between prompt versions.

Examples:
  promptsmith diff summarizer              # Compare working file vs latest
  promptsmith diff summarizer 1.0.0 1.0.1  # Compare two versions
  promptsmith diff summarizer HEAD~1 HEAD  # Compare using HEAD notation`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVar(&diffFormat, "format", "unified", "output format: unified, side-by-side")
	rootCmd.AddCommand(diffCmd)
}

type diffOutput struct {
	Prompt   string   `json:"prompt"`
	Version1 string   `json:"version1"`
	Version2 string   `json:"version2"`
	Hunks    []hunk   `json:"hunks"`
}

type hunk struct {
	OldStart int      `json:"old_start"`
	OldCount int      `json:"old_count"`
	NewStart int      `json:"new_start"`
	NewCount int      `json:"new_count"`
	Lines    []string `json:"lines"`
}

func runDiff(cmd *cobra.Command, args []string) error {
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

	var content1, content2 string
	var label1, label2 string

	versions, err := database.ListVersions(p.ID)
	if err != nil {
		return err
	}

	switch len(args) {
	case 1:
		// Compare working file vs latest version
		if len(versions) == 0 {
			return fmt.Errorf("no versions found for prompt '%s'", promptName)
		}
		latest := versions[0]
		content1 = latest.Content
		label1 = fmt.Sprintf("%s@%s", promptName, latest.Version)

		absPath := filepath.Join(projectRoot, p.FilePath)
		data, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("failed to read working file: %w", err)
		}
		content2 = string(data)
		label2 = fmt.Sprintf("%s (working)", promptName)

	case 2:
		// Single version argument - compare vs latest
		v1, err := resolveVersion(database, p.ID, versions, args[1])
		if err != nil {
			return err
		}
		if v1 == nil {
			return fmt.Errorf("version '%s' not found", args[1])
		}

		if len(versions) == 0 {
			return fmt.Errorf("no versions found for prompt '%s'", promptName)
		}
		latest := versions[0]

		content1 = v1.Content
		label1 = fmt.Sprintf("%s@%s", promptName, v1.Version)
		content2 = latest.Content
		label2 = fmt.Sprintf("%s@%s", promptName, latest.Version)

	case 3:
		// Compare two specific versions
		v1, err := resolveVersion(database, p.ID, versions, args[1])
		if err != nil {
			return err
		}
		if v1 == nil {
			return fmt.Errorf("version '%s' not found", args[1])
		}

		v2, err := resolveVersion(database, p.ID, versions, args[2])
		if err != nil {
			return err
		}
		if v2 == nil {
			return fmt.Errorf("version '%s' not found", args[2])
		}

		content1 = v1.Content
		label1 = fmt.Sprintf("%s@%s", promptName, v1.Version)
		content2 = v2.Content
		label2 = fmt.Sprintf("%s@%s", promptName, v2.Version)
	}

	if content1 == content2 {
		fmt.Println("No differences.")
		return nil
	}

	lines1 := strings.Split(content1, "\n")
	lines2 := strings.Split(content2, "\n")
	hunks := computeDiff(lines1, lines2)

	if jsonOut {
		output := diffOutput{
			Prompt:   promptName,
			Version1: label1,
			Version2: label2,
			Hunks:    hunks,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	printUnifiedDiff(label1, label2, hunks)
	return nil
}

func resolveVersion(database *db.DB, promptID string, versions []*db.PromptVersion, ref string) (*db.PromptVersion, error) {
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

func computeDiff(lines1, lines2 []string) []hunk {
	// Simple LCS-based diff algorithm
	m, n := len(lines1), len(lines2)

	// Build LCS table
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if lines1[i-1] == lines2[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else {
				if lcs[i-1][j] > lcs[i][j-1] {
					lcs[i][j] = lcs[i-1][j]
				} else {
					lcs[i][j] = lcs[i][j-1]
				}
			}
		}
	}

	// Backtrack to find diff
	var diffLines []struct {
		op   rune
		line string
		old  int
		new  int
	}

	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && lines1[i-1] == lines2[j-1] {
			diffLines = append([]struct {
				op   rune
				line string
				old  int
				new  int
			}{{' ', lines1[i-1], i, j}}, diffLines...)
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			diffLines = append([]struct {
				op   rune
				line string
				old  int
				new  int
			}{{'+', lines2[j-1], 0, j}}, diffLines...)
			j--
		} else if i > 0 {
			diffLines = append([]struct {
				op   rune
				line string
				old  int
				new  int
			}{{'-', lines1[i-1], i, 0}}, diffLines...)
			i--
		}
	}

	// Group into hunks with context
	const contextLines = 3
	var hunks []hunk
	var currentHunk *hunk

	for idx, dl := range diffLines {
		if dl.op != ' ' {
			// Start or extend hunk
			if currentHunk == nil {
				currentHunk = &hunk{
					OldStart: max(1, dl.old),
					NewStart: max(1, dl.new),
				}
				// Add preceding context
				start := max(0, idx-contextLines)
				for k := start; k < idx; k++ {
					if diffLines[k].op == ' ' {
						currentHunk.Lines = append(currentHunk.Lines, " "+diffLines[k].line)
						if currentHunk.OldStart == 0 || diffLines[k].old < currentHunk.OldStart {
							currentHunk.OldStart = diffLines[k].old
						}
						if currentHunk.NewStart == 0 || diffLines[k].new < currentHunk.NewStart {
							currentHunk.NewStart = diffLines[k].new
						}
					}
				}
			}

			switch dl.op {
			case '+':
				currentHunk.Lines = append(currentHunk.Lines, "+"+dl.line)
				currentHunk.NewCount++
			case '-':
				currentHunk.Lines = append(currentHunk.Lines, "-"+dl.line)
				currentHunk.OldCount++
			}
		} else if currentHunk != nil {
			// Context line after change
			currentHunk.Lines = append(currentHunk.Lines, " "+dl.line)
			currentHunk.OldCount++
			currentHunk.NewCount++

			// Check if we should close hunk
			nextChange := -1
			for k := idx + 1; k < len(diffLines) && k <= idx+contextLines+1; k++ {
				if diffLines[k].op != ' ' {
					nextChange = k
					break
				}
			}
			if nextChange == -1 || nextChange > idx+contextLines*2 {
				// Add trailing context up to contextLines
				added := 1 // We already added current
				for k := idx + 1; k < len(diffLines) && added < contextLines; k++ {
					if diffLines[k].op == ' ' {
						currentHunk.Lines = append(currentHunk.Lines, " "+diffLines[k].line)
						currentHunk.OldCount++
						currentHunk.NewCount++
						added++
					} else {
						break
					}
				}
				hunks = append(hunks, *currentHunk)
				currentHunk = nil
			}
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

func printUnifiedDiff(label1, label2 string, hunks []hunk) {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s %s\n", red("---"), label1)
	fmt.Printf("%s %s\n", green("+++"), label2)

	for _, h := range hunks {
		fmt.Printf("%s\n", cyan(fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OldStart, h.OldCount, h.NewStart, h.NewCount)))
		for _, line := range h.Lines {
			if len(line) == 0 {
				fmt.Println()
				continue
			}
			switch line[0] {
			case '+':
				fmt.Println(green(line))
			case '-':
				fmt.Println(red(line))
			default:
				fmt.Println(line)
			}
		}
	}
}
