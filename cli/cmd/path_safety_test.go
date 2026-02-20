package cmd

import (
	"path/filepath"
	"testing"
)

func TestSafeProjectPath(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "project")

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantPath string
	}{
		{name: "simple relative path", input: "prompts/a.prompt", wantPath: filepath.Join(root, "prompts", "a.prompt")},
		{name: "normalized path", input: "prompts/../prompts/a.prompt", wantPath: filepath.Join(root, "prompts", "a.prompt")},
		{name: "traversal parent", input: "../outside.prompt", wantErr: true},
		{name: "deep traversal", input: "prompts/../../outside.prompt", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "absolute", input: filepath.Join(string(filepath.Separator), "etc", "passwd"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safeProjectPath(root, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got path %q", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantPath {
				t.Fatalf("path = %q, want %q", got, tt.wantPath)
			}
		})
	}
}
