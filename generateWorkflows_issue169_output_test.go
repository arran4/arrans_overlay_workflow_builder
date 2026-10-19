package arrans_overlay_workflow_builder

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"
)

func TestIssue169PhysicalEbuildOutput(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	configString := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT
GeneratedEbuildGentooAuthors True
GeneratedEbuildSourceLicense GPL-2.0-only
Binary amd64=>test-${TAG}-linux-amd64 > test > test
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

	script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")
	binDir := filepath.Join(tempDir, "bin")
	mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\n")
	mockCommand(t, binDir, "gh", "#!/bin/bash\necho v1.0.0\n")

	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=test-bin\nset -euo pipefail\n"+script)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "generated workflow failed:\n%s", string(output))

	ebuildPath := filepath.Join(tempDir, "app-admin", "test-bin", "test-bin-1.0.0.ebuild")
	ebuildContent, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)

	contentStr := string(ebuildContent)
	require.Contains(t, contentStr, "# Copyright 2026 Gentoo Authors")
	require.Contains(t, contentStr, "# SPDX-License-Identifier: GPL-2.0-only")
    require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"MIT\"")
}
