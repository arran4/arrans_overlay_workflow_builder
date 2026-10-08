package arrans_overlay_workflow_builder

import (
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIssue170ProvenanceWidth(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	configString := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
Binary amd64=>test-${TAG}-linux-amd64-very-long-asset-name > test > test
`
	inputConfigs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	ic := inputConfigs[0]

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)
	err = ic.GenerateGithubWorkflow("test.config", time.Now(), tmpls, tempDir, "1.0.0")
	require.NoError(t, err)

	yamlPath := filepath.Join(tempDir, "app-admin-test-bin-update.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	var workflow map[string]any
	err = yaml.Unmarshal(yamlData, &workflow)
	require.NoError(t, err)

	jobs := workflow["jobs"].(map[string]any)
	job := jobs["check-and-create-ebuild"].(map[string]any)
	steps := job["steps"].([]any)

	var script string
	for _, stepItem := range steps {
		step := stepItem.(map[string]any)
		if step["name"] == "Process each release" {
			script = step["run"].(string)
			break
		}
	}
	require.NotEmpty(t, script, "Process each release step not found")

	// Do manual replacements first to test long values
	script = strings.ReplaceAll(script, "${{ github.server_url }}", "https://github.com")
	script = strings.ReplaceAll(script, "${{ github.repository }}", "a-very-very-very-very-very-very-very-very-very-very-very-long-repo-name/test")
	script = strings.ReplaceAll(script, "${{ github.sha }}", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "app-admin-arrans-overlay-workflow-builder-bin-update-with-an-extremely-long-suffix-to-force-wrapping-to-multiple-lines.yaml")
	script = resolveGithubEnvForPackage(script, "app-admin", "test-bin")

	cmd := exec.Command("bash", "-c", "P=test-bin\nset -euo pipefail\n"+script)
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	ebuildPath := filepath.Join(tempDir, "app-admin", "test-bin", "test-bin-1.0.0.ebuild")
	ebuildContent, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)

	ebuildLines := strings.Split(string(ebuildContent), "\n")

	// Check physical widths
	for i, line := range ebuildLines {
		pos := 0
		for _, ch := range line {
			if ch == '	' {
				pos += 4
			} else {
				pos += 1
			}
		}
		if pos > 80 {
			t.Errorf("Ebuild line %d exceeds 80 characters (width %d): %s", i+1, pos, line)
		}
	}

	// Reconstruct provenance
	var repo, commit, workflowPath string
	for _, line := range ebuildLines {
		if strings.HasPrefix(line, "#   repository: ") || (repo != "" && strings.HasPrefix(line, "#               ") && commit == "") {
			repo += strings.TrimSpace(strings.TrimPrefix(line, "#"))
		} else if strings.HasPrefix(line, "#   commit: ") || (commit != "" && strings.HasPrefix(line, "#           ") && workflowPath == "") {
			commit += strings.TrimSpace(strings.TrimPrefix(line, "#"))
		} else if strings.HasPrefix(line, "#   workflow: ") || (workflowPath != "" && strings.HasPrefix(line, "#             ")) {
			workflowPath += strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
	}

	require.Equal(t, "repository: https://github.com/a-very-very-very-very-very-very-very-very-very-very-very-long-repo-name/test", repo)
	require.Equal(t, "commit: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", commit)
	require.Equal(t, "workflow: .github/workflows/app-admin-arrans-overlay-workflow-builder-bin-update-with-an-extremely-long-suffix-to-force-wrapping-to-multiple-lines.yaml", workflowPath)
}
