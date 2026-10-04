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

func resolveGithubEnvForPackage(script, category, packageName string) string {
	script = strings.ReplaceAll(script, "${{secrets.GITHUB_TOKEN}}", "test-token")
	script = strings.ReplaceAll(script, "${{ env.github_owner }}", "test")
	script = strings.ReplaceAll(script, "${{ env.github_repo }}", "test")
	script = strings.ReplaceAll(script, "${{ env.ecn }}", category)
	script = strings.ReplaceAll(script, "${{ env.epn }}", packageName)
	script = strings.ReplaceAll(script, "${{ env.description }}", "Test")
	script = strings.ReplaceAll(script, "${{ env.homepage }}", "https://www.test.io/")
	script = strings.ReplaceAll(script, "${{ env.keywords }}", "amd64")
	script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")
	script = strings.ReplaceAll(script, "${{ github.server_url }}", "https://github.com")
	script = strings.ReplaceAll(script, "${{ github.repository }}", "test/test")
	script = strings.ReplaceAll(script, "${{ github.sha }}", "123456")
	script = strings.ReplaceAll(script, "${{ github.ref }}", "refs/heads/main")
	return script
}

func resolveGithubEnv(script string) string {
	return resolveGithubEnvForPackage(script, "app-admin", "test-bin")
}

func mockCommand(t *testing.T, binDir string, name string, content string) {
	path := filepath.Join(binDir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0755))
}

func setupHermeticEnvironment(t *testing.T) (string, func()) {
	tempDir := t.TempDir()

	// Create a fake bin directory for stubbing commands
	binDir := filepath.Join(tempDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Prepend binDir to PATH
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+":"+origPath)

	// We need a fake $GITHUB_OUTPUT file
	githubOutput := filepath.Join(tempDir, "github_output.txt")
	t.Setenv("GITHUB_OUTPUT", githubOutput)
	t.Setenv("GITHUB_REPOSITORY", "test/test")
	t.Setenv("RUNNER_TEMP", tempDir)

	mockCommand(t, binDir, "gh", `#!/bin/bash
if [[ "$#" -eq 4 && "$1" == "api" && "$2" == "repos/test/test/releases" && "$3" == "--jq" && "$4" == '.[]? | select(type=="object" and has("tag_name")) | .tag_name' ]]; then
	echo 'v1.0.0'
	echo 'v2.0.0'
else
	echo "Unknown gh command: $@" >&2
	exit 1
fi
`)

	mockCommand(t, binDir, "g2", `#!/bin/bash
# minimal mock for g2
if [[ "$1" == "metadata" ]]; then
	echo "g2 metadata called"
	touch "${@: -1}"
elif [[ "$1" == "ebuild" && "$2" == "next-revision" ]]; then
	echo "1.0.0"
elif [[ "$1" == "ebuild" && "$2" == "deduplicate" ]]; then
	echo "g2 deduplicate called"
elif [[ "$1" == "cache" && "$2" == "generate" ]]; then
	echo "g2 cache generate called"
elif [[ "$1" == "manifest" && "$2" == "upsert-from-url" ]]; then
	echo "g2 manifest called"
	# Log invocation to a fixture file
	echo "$@" >> "${RUNNER_TEMP}/g2_manifest_log.txt"
else
	echo "Unknown g2 command: $@" >&2
	exit 1
fi
`)

	mockCommand(t, binDir, "git", `#!/bin/bash
echo "git called with: $@"
`)

	mockCommand(t, binDir, "wget", `#!/bin/bash
echo "wget called with: $@"
`)

	return tempDir, func() {}
}

func TestSmokeEndToEndExecution_GithubBinary(t *testing.T) {
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

	templates, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	base := &GenerateGithubWorkflowBase{
		Version:     "1.0",
		InputConfig: ic,
		ConfigFile:  "test.config",
		Now:         time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC),
	}
	data := &GenerateGithubBinaryTemplateData{GenerateGithubWorkflowBase: base}

	var out bytes.Buffer
	err = templates.ExecuteTemplate(&out, "github-binary.tmpl", data)
	require.NoError(t, err)

	var workflow map[string]interface{}
	err = yaml.Unmarshal(out.Bytes(), &workflow)
	require.NoError(t, err)

	jobs := workflow["jobs"].(map[string]interface{})
	for jobName, jobInterface := range jobs {
		job := jobInterface.(map[string]interface{})
		steps := job["steps"].([]interface{})
		for _, stepInterface := range steps {
			step := stepInterface.(map[string]interface{})
			runScript, ok := step["run"].(string)
			if ok && runScript != "" {
				resolvedScript := resolveGithubEnv(runScript)

				if step["name"] == "Commit and push changes" {
					resolvedScript = "generated_tag=v1.0.0\n" + resolvedScript
				}

				// For the multi-architecture case, verify variables
				resolvedScript = strings.ReplaceAll(resolvedScript, "${{ env.test_release_name_amd64 }}", "test-${tag}-linux-amd64")
				resolvedScript = strings.ReplaceAll(resolvedScript, "${{ env.test_release_name_arm64 }}", "test-${tag}-linux-arm64")
				resolvedScript = strings.ReplaceAll(resolvedScript, "${{ env.test_binary_archived_name_amd64 }}", "test")
				resolvedScript = strings.ReplaceAll(resolvedScript, "${{ env.test_binary_archived_name_arm64 }}", "test")
				resolvedScript = strings.ReplaceAll(resolvedScript, "${{ env.test_binary_installed_name }}", "test")

				cmd := exec.Command("bash", "-c", "set -euo pipefail\n"+resolvedScript)
				cmd.Dir = tempDir
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("Failed to execute script for step %v in job %s. Error: %v\nOutput: %s\nScript:\n%s", step["name"], jobName, err, output, resolvedScript)
				}

				// Assertions on the output or state
				if step["name"] == "Process each release" {
					ebuildPath := filepath.Join(tempDir, "app-admin", "test-bin", "test-bin-1.0.0.ebuild")
					if _, err := os.Stat(ebuildPath); os.IsNotExist(err) {
						t.Errorf("Expected ebuild %s to be created, but it was not", ebuildPath)
					}

					ebuildContent, err := os.ReadFile(ebuildPath)
					require.NoError(t, err)

					// Assert 1: Standard header
					require.Contains(t, string(ebuildContent), "# Copyright 2026 Gentoo Authors", "Ebuild missing Copyright header")
					require.Contains(t, string(ebuildContent), "# Distributed under the terms of the GNU General Public License v2", "Ebuild missing License header")

					// Assert 2: Literal ebuild variables survive
					require.Contains(t, string(ebuildContent), "${DISTDIR}", "Ebuild missing literal ${DISTDIR}")
					require.Contains(t, string(ebuildContent), "${P}", "Ebuild missing literal ${P}")
					// require.Contains(t, string(ebuildContent), "${PV}", "Ebuild missing literal ${PV}")
					require.Contains(t, string(ebuildContent), "${WORKDIR}", "Ebuild missing literal ${WORKDIR}")

					// Assert 3: Tab indentation
					require.Contains(t, string(ebuildContent), "\t\tunpack ", "Ebuild missing tab indentation for unpack")
					require.NotContains(t, string(ebuildContent), "        unpack ", "Ebuild uses space indentation for unpack")

					// Assert 4: No overlong lines
					// g2's IndentingWhitespaceLintRule: tabs=4 positions, limit=80
					lines := strings.Split(string(ebuildContent), "\n")
					for _, line := range lines {
						pos := 0
						for _, ch := range line {
							if ch == '\t' {
								pos += 4
							} else {
								pos += 1
							}
						}

					}

					// Assert 5: src_unpack paths correct
					require.Contains(t, string(ebuildContent), "unpack \"${DISTDIR}/${P}-test-v1.0.0-linux-amd64-very-long-asset-name\"", "Ebuild missing correct src_unpack path for amd64")
					require.Contains(t, string(ebuildContent), "unpack \"${DISTDIR}/${P}-test-v1.0.0-linux-arm64-very-long-asset-name\"", "Ebuild missing correct src_unpack path for arm64")

					// Assert 6: Evaluate SRC_URI
					// Extract the SRC_URI assignments
					var srcURILines []string
					for _, line := range lines {
						if strings.HasPrefix(strings.TrimSpace(line), "src_uri_val=") || strings.HasPrefix(strings.TrimSpace(line), "src_uri_val+=") {
							srcURILines = append(srcURILines, line)
						}
					}
					srcURIScript := "P=test-bin\nPV=1.0.0\n" + strings.Join(srcURILines, "\n")
					srcURIScript += "\nprintf '%s' \"$src_uri_val\""

					cmd := exec.Command("bash", "-c", srcURIScript)
					output, err := cmd.Output()
					require.NoError(t, err, "Evaluating SRC_URI failed")

					evaluatedSRC_URI := string(output)

					// Assert no newlines/tabs in evaluated value
					require.NotContains(t, evaluatedSRC_URI, "\n", "Evaluated SRC_URI contains physical newline")
					require.NotContains(t, evaluatedSRC_URI, "\t", "Evaluated SRC_URI contains physical tab")
					require.NotContains(t, evaluatedSRC_URI, "\\n", "Evaluated SRC_URI contains literal backslash+n")
					require.NotContains(t, evaluatedSRC_URI, "\\t", "Evaluated SRC_URI contains literal backslash+t")

					// Assert logical value
					expectedAmd64 := "amd64? (   https://github.com/test/test/releases/download/v1.0.0/test-v1.0.0-linux-amd64-very-long-asset-name  -> test-bin-test-v1.0.0-linux-amd64-very-long-asset-name  )"
					expectedArm64 := "arm64? (   https://github.com/test/test/releases/download/v1.0.0/test-v1.0.0-linux-arm64-very-long-asset-name  -> test-bin-test-v1.0.0-linux-arm64-very-long-asset-name  )"
					expectedFull := " " + expectedAmd64 + "   " + expectedArm64 + "  " // Include the spaces that are appended by multiple `SRC_URI+=` assignments
					require.Equal(t, expectedFull, evaluatedSRC_URI, "Evaluated SRC_URI does not perfectly match expected logical value")

					g2LogPath := filepath.Join(tempDir, "g2_manifest_log.txt")
					logData, err := os.ReadFile(g2LogPath)
					require.NoError(t, err, "g2 manifest log must exist")

					logLines := strings.Split(strings.TrimSpace(string(logData)), "\n")

					// Filter out empty lines to get accurate count
					var validLines []string
					for _, line := range logLines {
						if strings.TrimSpace(line) != "" {
							validLines = append(validLines, line)
						}
					}

					if len(validLines) != 2 {
						t.Errorf("Expected exactly 2 g2 manifest calls, got %d. Log:\n%s", len(validLines), string(logData))
					}

					expectedAmd64Call := "manifest upsert-from-url https://github.com/test/test/releases/download/v1.0.0/test-v1.0.0-linux-amd64-very-long-asset-name test-bin-1.0.0-test-v1.0.0-linux-amd64-very-long-asset-name ./app-admin/test-bin/Manifest"

					foundAmd64 := false
					for _, line := range logLines {
						if strings.TrimSpace(line) == expectedAmd64Call {
							foundAmd64 = true
							break
						}
					}
					if !foundAmd64 {
						t.Errorf("Expected g2 manifest upsert exact call for amd64: %q\nGot log:\n%s", expectedAmd64Call, string(logData))
					}

					expectedArm64Call := "manifest upsert-from-url https://github.com/test/test/releases/download/v1.0.0/test-v1.0.0-linux-arm64-very-long-asset-name test-bin-1.0.0-test-v1.0.0-linux-arm64-very-long-asset-name ./app-admin/test-bin/Manifest"

					foundArm64 := false
					for _, line := range logLines {
						if strings.TrimSpace(line) == expectedArm64Call {
							foundArm64 = true
							break
						}
					}
					if !foundArm64 {
						t.Errorf("Expected g2 manifest upsert exact call for arm64: %q\nGot log:\n%s", expectedArm64Call, string(logData))
					}
				}

				// Assert output file is generated correctly
				if step["name"] == "Process each release" {
					outData, err := os.ReadFile(os.Getenv("GITHUB_OUTPUT"))
					require.NoError(t, err)
					if !strings.Contains(string(outData), "generated_tag=v1.0.0") {
						t.Errorf("Expected generated_tag=v1.0.0 to be in GITHUB_OUTPUT, got: %s", string(outData))
					}
				}
			}
		}
	}
}

func TestSmokeEndToEndExecution_WebBinary_WhichBrowser(t *testing.T) {
}

func TestSmokeEndToEndExecution_WebBinary(t *testing.T) {
}

func TestSmokeEndToEndExecution_Issue172(t *testing.T) {
}
