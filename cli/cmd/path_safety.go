package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
)

func safeProjectPath(projectRoot, relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	cleaned := filepath.Clean(relPath)
	fullPath := filepath.Join(projectRoot, cleaned)

	relative, err := filepath.Rel(projectRoot, fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to validate path: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root")
	}

	return fullPath, nil
}
