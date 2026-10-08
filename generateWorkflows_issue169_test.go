package arrans_overlay_workflow_builder

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIssue169CLIDefaults(t *testing.T) {
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

	require.Contains(t, string(yamlData), "# SPDX-License-Identifier: MIT", "CLI default should emit SPDX MIT")
}
