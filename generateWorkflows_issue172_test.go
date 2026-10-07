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

func TestIssue172MetadataWidth(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	configString := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description An extremely long description that will definitely exceed the eighty character limit that we have set for ebuilds to ensure they wrap correctly
Homepage https://www.test.io/
License MIT License
Binary amd64=>test-${TAG}-linux-amd64-very-long-asset-name > test > test
Binary arm64=>test-${TAG}-linux-arm64-very-long-asset-name > test > test
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

	script = resolveGithubEnvForPackage(script, "app-admin", "test-bin")

	cmd := exec.Command("bash", "-c", "set -euo pipefail\n"+script)
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	ebuildPath := filepath.Join(tempDir, "app-admin", "test-bin", "test-bin-1.0.0.ebuild")
	ebuildContent, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)

	lines := strings.Split(string(ebuildContent), "\n")
	for i, line := range lines {
		pos := 0
		for _, ch := range line {
			if ch == '\t' {
				pos += 4
			} else {
				pos += 1
			}
		}
		if pos > 80 {
			t.Errorf("Ebuild line %d exceeds 80 characters (width %d): %s", i+1, pos, line)
		}
	}
}

func TestIssue172ActualWidthCheck(t *testing.T) {}
