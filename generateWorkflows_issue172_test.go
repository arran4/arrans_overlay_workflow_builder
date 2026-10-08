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
	binDir := filepath.Join(tempDir, "bin")
	mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\n")
	mockCommand(t, binDir, "gh", "#!/bin/bash\necho v1.0.0\n")
	script = strings.ReplaceAll(script, "emit_metadata_field \"DESCRIPTION\" \"Test\"", "emit_metadata_field \"DESCRIPTION\" \"1234567890123456789012345678901234567890123456789012345678901234567890123\\${P}\"")
	script = strings.ReplaceAll(script, "DESCRIPTION+=\"Test\"", "DESCRIPTION+=\"1234567890123456789012345678901234567890123456789012345678901234567890123${P}\"")

	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=test-bin\nset -euo pipefail\n"+script)
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Bash failed: "+string(out))

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

	evalMetadata := func(varName string) string {
		var varLines []string
		for _, line := range ebuildLines {
			if strings.HasPrefix(strings.TrimSpace(line), varName+"=") || strings.HasPrefix(strings.TrimSpace(line), varName+"+=") {
				varLines = append(varLines, line)
			}
		}

		script := "P=test-bin\nPV=1.0.0\n" + strings.Join(varLines, "\n")
		script += "\nprintf '%s' \"$" + varName + "\""

		evalCmd := exec.Command("bash", "-c", script)
		output, evalErr := evalCmd.Output()
		require.NoError(t, evalErr, "Evaluating "+varName+" failed")

		evaluatedValue := string(output)
		require.NotContains(t, evaluatedValue, "\n")
		require.NotContains(t, evaluatedValue, "\t")
		return evaluatedValue
	}

	require.Equal(t, "1234567890123456789012345678901234567890123456789012345678901234567890123test-bin", evalMetadata("DESCRIPTION"), "DESCRIPTION variable expansion split or mangled")
	require.Equal(t, "https://www.test.io/", evalMetadata("HOMEPAGE"))
	require.Equal(t, "MIT License", evalMetadata("LICENSE"))
	require.Equal(t, "amd64", evalMetadata("KEYWORDS"))
	require.Equal(t, "", evalMetadata("IUSE"))
	require.Equal(t, "", evalMetadata("REQUIRED_USE"))
	require.Equal(t, "", evalMetadata("DEPEND"))
	require.Equal(t, "", evalMetadata("RDEPEND"))
	require.Equal(t, "", evalMetadata("BDEPEND"))
	require.Equal(t, " amd64? (   https://github.com/test/test/releases/download/v1.0.0/test-v1.0.0-linux-amd64-very-long-asset-name  -> test-bin-test-v1.0.0-linux-amd64-very-long-asset-name  )   arm64? (   https://github.com/test/test/releases/download/v1.0.0/test-v1.0.0-linux-arm64-very-long-asset-name  -> test-bin-test-v1.0.0-linux-arm64-very-long-asset-name  )  ", evalMetadata("SRC_URI"))
}
