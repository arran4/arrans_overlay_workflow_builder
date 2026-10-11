package arrans_overlay_workflow_builder

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Exercise the generated workflow's actual shell step and inspect the ebuild.
// A template-source substring check cannot detect truncation, lost characters
// or overlong physical provenance lines.
func TestIssue170ProvenanceWidth(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT
Binary amd64=>test-${TAG}-linux-amd64-very-long-asset-name > test > test
Binary arm64=>test-${TAG}-linux-arm64-very-long-asset-name > test > test
`
	configs, err := ParseInputConfigReader(strings.NewReader(cfg))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)
	err = configs[0].GenerateGithubWorkflow(
		"test.config",
		time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC),
		tmpls, tempDir, "1.0.0",
	)
	require.NoError(t, err)

	yamlData, err := os.ReadFile(filepath.Join(tempDir, "app-admin-test-bin-update.yaml"))
	require.NoError(t, err)
	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(yamlData, &workflow))
	jobs, ok := workflow["jobs"].(map[string]any)
	require.True(t, ok)
	job, ok := jobs["check-and-create-ebuild"].(map[string]any)
	require.True(t, ok)
	steps, ok := job["steps"].([]any)
	require.True(t, ok)

	var script string
	for _, raw := range steps {
		step := raw.(map[string]any)
		if step["name"] == "Process each release" {
			script, _ = step["run"].(string)
			break
		}
	}
	require.NotEmpty(t, script)

	longRepo := "very-long-organisation/" + strings.Repeat("repository", 14)
	longSHA := strings.Repeat("0123456789", 15)
	longWorkflow := "very-long-" + strings.Repeat("workflow", 14) + ".yaml"
	expected := map[string]string{
		"repository": "https://github.com/" + longRepo,
		"commit":     longSHA,
		"workflow":   ".github/workflows/" + longWorkflow,
	}

	// Substitute the real workflow expressions before resolving the other
	// fixture variables, to preserve the deliberately long provenance values.
	script = strings.ReplaceAll(script, "${{ github.repository }}", longRepo)
	script = strings.ReplaceAll(script, "${{ github.sha }}", longSHA)
	script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", longWorkflow)
	script = resolveGithubEnvForPackage(script, "app-admin", "test-bin")
	script = strings.ReplaceAll(script, "${{ env.test_release_name_amd64 }}", "test-${tag}-linux-amd64")
	script = strings.ReplaceAll(script, "${{ env.test_release_name_arm64 }}", "test-${tag}-linux-arm64")
	script = strings.ReplaceAll(script, "${{ env.test_binary_archived_name_amd64 }}", "test")
	script = strings.ReplaceAll(script, "${{ env.test_binary_archived_name_arm64 }}", "test")
	script = strings.ReplaceAll(script, "${{ env.test_binary_installed_name }}", "test")

	binDir := filepath.Join(tempDir, "bin")
	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nset -euo pipefail\n"+script)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "generated workflow failed:\n%s\nScript:\n%s", output, script)

	ebuild, err := os.ReadFile(filepath.Join(tempDir, "app-admin", "test-bin", "test-bin-1.0.0.ebuild"))
	require.NoError(t, err)
	lines := strings.Split(string(ebuild), "\n")

	for label, want := range expected {
		prefix := "#   " + label + ": "
		continuation := "#" + strings.Repeat(" ", len(prefix)-1)
		index := -1
		for i, line := range lines {
			if strings.HasPrefix(line, prefix) {
				require.Equal(t, -1, index, "duplicate provenance label %s", label)
				index = i
			}
		}
		require.NotEqual(t, -1, index, "missing provenance field %s", label)
		var reconstructed strings.Builder
		reconstructed.WriteString(strings.TrimPrefix(lines[index], prefix))
		wrapped := 0
		for i := index + 1; i < len(lines) && strings.HasPrefix(lines[i], continuation); i++ {
			reconstructed.WriteString(strings.TrimPrefix(lines[i], continuation))
			wrapped++
		}
		require.Greater(t, wrapped, 0, "fixture for %s must wrap", label)
		require.Equal(t, want, reconstructed.String(), "%s lost/reordered provenance characters", label)
	}

	inProvenance := false
	for i, line := range lines {
		if line == "# Generated via:" {
			inProvenance = true
			continue
		}
		if inProvenance && line == "EAPI=8" {
			break
		}
		if inProvenance {
			require.LessOrEqualf(t, len(line), 80, "provenance line %d is over 80 columns: %q", i+1, line)
		}
	}
	require.True(t, inProvenance, "generated ebuild has no provenance section")
}
