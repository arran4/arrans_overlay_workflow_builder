package arrans_overlay_workflow_builder

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arran4/arrans_overlay_workflow_builder/util"
)

func TestTagsCommandRendering(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected string
	}{
		{
			name:     "AppImage",
			config:   "testdata/config/tags_command_appimage.config",
			expected: "tags=$(cat tags.txt)",
		},
		{
			name:     "Binary",
			config:   "testdata/config/tags_command_binary.config",
			expected: "tags=$(curl -s https://raw.githubusercontent.com/JetBrains/junie/main/update-info.jsonl | jq -r '.version' | sort -u -V -r)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := os.ReadFile(tt.config)
			if err != nil {
				t.Fatalf("ReadFile(%q): %v", tt.config, err)
			}

			fsys := util.NewMockFS()
			if err := fsys.WriteFile("input.config", config, 0644); err != nil {
				t.Fatalf("WriteFile(input.config): %v", err)
			}

			now := time.Date(2026, time.July, 12, 0, 0, 0, 0, time.UTC)
			if err := GenerateGithubWorkflows("input.config", "output", "1.0.0", fsys, now); err != nil {
				t.Fatalf("GenerateGithubWorkflows(): %v", err)
			}

			var workflow string
			for name, contents := range fsys.Files {
				if strings.HasPrefix(name, "output/") {
					workflow = string(contents)
					break
				}
			}
			if workflow == "" {
				t.Fatal("GenerateGithubWorkflows() did not write a workflow")
			}
			if !strings.Contains(workflow, tt.expected) {
				t.Errorf("generated workflow does not contain %q", tt.expected)
			}
		})
	}
}
