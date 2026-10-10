package arrans_overlay_workflow_builder

import (
	"strings"
	"testing"
	"time"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/require"
)

func TestEmptyStructureTestThatNeedToBeReplaced(t *testing.T) {
	// Ensure the config file parsing is valid
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

	yamlPath := filepath.Join(tempDir, "app-admin-test-bin-update.yaml")
	_, err = os.ReadFile(yamlPath)
	require.NoError(t, err)
}
