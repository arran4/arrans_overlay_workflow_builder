package arrans_overlay_workflow_builder

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestGeneratedBinarySRCURICacheValue uses the real workflow-to-ebuild
// generation path. The g2 executable is mocked by the shared smoke harness:
// this verifies its input boundary, not the behavior of g2 cache generate.
func TestGeneratedBinarySRCURICacheValue(t *testing.T) {
	dir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	config := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
Binary amd64=>test-${TAG}-linux-amd64-very-long-filename-that-will-cause-line-wrapping-if-not-careful.tar.gz > test > test
Binary arm64=>test-${TAG}-linux-arm64-very-long-filename-that-will-cause-line-wrapping-if-not-careful.tar.gz > test > test
`
	configs, err := ParseInputConfigReader(strings.NewReader(config))
	require.NoError(t, err)
	require.Len(t, configs, 1)
	templates, err := ParseWorkflowTemplates()
	require.NoError(t, err)
	base := &GenerateGithubWorkflowBase{
		Version: "1.0", InputConfig: configs[0], ConfigFile: "test.config",
		Now: time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC),
	}
	var rendered bytes.Buffer
	require.NoError(t, templates.ExecuteTemplate(&rendered, "github-binary.tmpl", &GenerateGithubBinaryTemplateData{GenerateGithubWorkflowBase: base}))

	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string `yaml:"name"`
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(rendered.Bytes(), &workflow))
	var processScript string
	for _, job := range workflow.Jobs {
		for _, step := range job.Steps {
			if step.Name == "Process each release" {
				processScript = step.Run
			}
		}
	}
	require.NotEmpty(t, processScript, "generated workflow needs a release-processing step")
	processScript = resolveGithubEnv(processScript)
	for expression, replacement := range map[string]string{
		"${{ env.binary_archived_name_amd64 }}": "test",
		"${{ env.binary_archived_name_arm64 }}": "test",
		"${{ env.binary_installed_name }}":     "test",
		"${{ env.release_name_amd64 }}":       "test-${tag}-linux-amd64",
		"${{ env.release_name_arm64 }}":       "test-${tag}-linux-arm64",
	} {
		processScript = strings.ReplaceAll(processScript, expression, replacement)
	}
	require.NotContains(t, processScript, "${{", "unresolved GitHub expression in executable test script")
	cmd := exec.Command("bash", "-c", "set -euo pipefail\n"+processScript)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "generated release script failed: %s", output)

	ebuildPath := filepath.Join(dir, "app-admin", "test-bin", "test-bin-1.0.0.ebuild")
	ebuild, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)
	srcURI := evaluateEbuildSRCURI(t, string(ebuild), "test-bin-1.0.0", "1.0.0")
	require.NotContains(t, srcURI, "\n", "actual newlines make a malformed physical md5-cache record")
	require.NotContains(t, srcURI, `\n`, "literal backslash-n artifact")
	require.NotContains(t, srcURI, `\t`, "literal backslash-t artifact")
	require.NotContains(t, srcURI, "${", "ebuild variables must be expanded in SRC_URI")

	// Assert architecture grouping, URL/rename token boundaries, and both full
	// rename filenames. Compare tokens only after checking for embedded newlines.
	const longName = "very-long-filename-that-will-cause-line-wrapping-if-not-careful.tar.gz"
	expected := "amd64? ( https://github.com/test/test/releases/download/v1.0.0/1.0.0 -> " +
		"test-bin-1.0.0-test-v1.0.0-linux-amd64-" + longName + " ) " +
		"arm64? ( https://github.com/test/test/releases/download/v1.0.0/1.0.0 -> " +
		"test-bin-1.0.0-test-v1.0.0-linux-arm64-" + longName + " )"
	require.Equal(t, strings.Fields(expected), strings.Fields(srcURI), "evaluated SRC_URI must preserve complete tokens and architecture conditionals")

	cacheRecord := "SRC_URI=" + srcURI + "\n"
	require.Equal(t, 1, strings.Count(cacheRecord, "\n"), "SRC_URI must fit in one physical cache line")
}

// Evaluate only top-level SRC_URI assignments from the actual ebuild. Running
// the entire file would require the Portage environment and eclass helpers.
func evaluateEbuildSRCURI(t *testing.T, ebuild, p, pv string) string {
	t.Helper()
	var assignments []string
	for _, line := range strings.Split(ebuild, "\n") {
		if strings.HasPrefix(line, "SRC_URI=") || strings.HasPrefix(line, "SRC_URI+=") {
			assignments = append(assignments, line)
		}
	}
	require.Greater(t, len(assignments), 1, "expected multiple SRC_URI assignments in generated ebuild")
	// P and PV are fixed test-fixture values; the ebuild text is generated
	// locally, not supplied by an external service.
	script := "set -euo pipefail\nP=" + p + "\nPV=" + pv + "\n" + strings.Join(assignments, "\n") + "\nprintf '%s' \"$SRC_URI\"\n"
	cmd := exec.Command("bash", "-c", script)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "evaluating generated SRC_URI failed: %s", output)
	return string(output)
}
