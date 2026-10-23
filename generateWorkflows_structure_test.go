package arrans_overlay_workflow_builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGeneratedGithubBinaryWorkflowStructure(t *testing.T) {
	configString := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
Category app-admin
EbuildName test-bin
Description Test
Homepage https://example.test/
License BSD-2-Clause
Binary amd64=>test-${VERSION}.tar.gz > test > test
`
	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	err = configs[0].GenerateGithubWorkflow("test.config", time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), tmpls, tempDir, "1.0.0")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tempDir, "app-admin-test-bin-update.yaml"))
	require.NoError(t, err)

	// A generated file existing on disk says nothing about whether Actions can
	// execute it. Parse the actual YAML and assert the build pipeline's shape.
	var workflow struct {
		Name        string `yaml:"name"`
		Permissions struct {
			Contents string `yaml:"contents"`
		} `yaml:"permissions"`
		On  map[string]any `yaml:"on"`
		Env struct {
			Category string `yaml:"ecn"`
			Package  string `yaml:"epn"`
		} `yaml:"env"`
		Jobs map[string]struct {
			RunsOn string `yaml:"runs-on"`
			Steps  []struct {
				Name string `yaml:"name"`
				ID   string `yaml:"id"`
				Uses string `yaml:"uses"`
				Run  string `yaml:"run"`
				If   string `yaml:"if"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(content, &workflow))
	require.NotEmpty(t, workflow.Name)
	require.Equal(t, "write", workflow.Permissions.Contents)
	require.Contains(t, workflow.On, "workflow_dispatch")
	require.Contains(t, workflow.On, "schedule")
	require.Equal(t, "app-admin", workflow.Env.Category)
	require.Equal(t, "test-bin", workflow.Env.Package)

	job, exists := workflow.Jobs["check-and-create-ebuild"]
	require.True(t, exists, "missing primary generation job")
	require.Equal(t, "ubuntu-latest", job.RunsOn)
	require.NotEmpty(t, job.Steps)

	positions := make(map[string]int, len(job.Steps))
	for i, step := range job.Steps {
		require.NotEmpty(t, step.Name, "unnamed step at index %d", i)
		_, seen := positions[step.Name]
		require.False(t, seen, "duplicate workflow step %q", step.Name)
		positions[step.Name] = i
		require.NotEqual(t, step.Run != "", step.Uses != "",
			"step %q must have exactly one executable run or uses", step.Name)
	}
	for _, name := range []string{
		"Checkout repository", "Install g2", "Process each release",
		"Deduplicate ebuilds", "Lint output", "Commit and push changes",
	} {
		_, present := positions[name]
		require.True(t, present, "required workflow step %q absent", name)
	}
	require.Less(t, positions["Checkout repository"], positions["Process each release"])
	require.Less(t, positions["Install g2"], positions["Process each release"])
	require.Less(t, positions["Process each release"], positions["Deduplicate ebuilds"])
	require.Less(t, positions["Deduplicate ebuilds"], positions["Lint output"])
	require.Less(t, positions["Lint output"], positions["Commit and push changes"])
	require.Equal(t, "process_releases", job.Steps[positions["Process each release"]].ID)
	require.NotEmpty(t, job.Steps[positions["Process each release"]].Run)
	require.Contains(t, job.Steps[positions["Lint output"]].If, "process_releases.outputs.generated_tag")
	require.Contains(t, job.Steps[positions["Commit and push changes"]].If, "process_releases.outputs.generated_tag")
}
