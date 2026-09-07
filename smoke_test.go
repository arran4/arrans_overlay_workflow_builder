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

func resolveGithubEnv(script string) string {
	script = strings.ReplaceAll(script, "${{secrets.GITHUB_TOKEN}}", "test-token")
	script = strings.ReplaceAll(script, "${{ env.github_owner }}", "test")
	script = strings.ReplaceAll(script, "${{ env.github_repo }}", "test")
	script = strings.ReplaceAll(script, "${{ env.ecn }}", "app-admin")
	script = strings.ReplaceAll(script, "${{ env.epn }}", "test-bin")
	script = strings.ReplaceAll(script, "${{ env.description }}", "Test")
	script = strings.ReplaceAll(script, "${{ env.homepage }}", "https://www.test.io/")
	script = strings.ReplaceAll(script, "${{ env.keywords }}", "amd64")
	script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")
	script = strings.ReplaceAll(script, "${{ github.server_url }}", "https://github.com")
	script = strings.ReplaceAll(script, "${{ github.repository }}", "test/test")
	script = strings.ReplaceAll(script, "${{ github.sha }}", "123456")
	script = strings.ReplaceAll(script, "${{ github.ref }}", "refs/heads/main")

	// Add mock outputs generated previously
	script = "generated_tag=v2.0.0\n" + script
	return script
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
	os.Setenv("PATH", binDir+":"+origPath)

	// We need a fake $GITHUB_OUTPUT file
	githubOutput := filepath.Join(tempDir, "github_output.txt")
	os.Setenv("GITHUB_OUTPUT", githubOutput)
	os.Setenv("GITHUB_REPOSITORY", "test/test")
	os.Setenv("RUNNER_TEMP", tempDir)

	mockCommand(t, binDir, "gh", `#!/bin/bash
if [[ "$1" == "api" && "$2" == "repos/test/test/releases" ]]; then
	echo 'v1.0.0'
	echo 'v2.0.0'
else
	echo "Unknown gh command: $@" >&2
	# exit fake replaced
fi
`)

	mockCommand(t, binDir, "g2", `#!/bin/bash
# minimal mock for g2
if [[ "$1" == "metadata" ]]; then
	echo "g2 metadata called"
	# Ignore all flags and just touch the last argument which is the file
	touch "${@: -1}"
elif [[ "$1" == "ebuild" && "$2" == "next-revision" ]]; then
	echo "1.0.0"
elif [[ "$1" == "ebuild" && "$2" == "deduplicate" ]]; then
	echo "g2 deduplicate called"
elif [[ "$1" == "cache" && "$2" == "generate" ]]; then
	echo "g2 cache generate called"
elif [[ "$1" == "manifest" && "$2" == "upsert-from-url" ]]; then
	echo "g2 manifest called"
else
	echo "Unknown g2 command: $@" >&2
fi
`)

	mockCommand(t, binDir, "git", `#!/bin/bash
echo "git called with: $@"
`)

	mockCommand(t, binDir, "wget", `#!/bin/bash
echo "wget called with: $@"
`)

	cleanup := func() {
		os.Setenv("PATH", origPath)
		os.Unsetenv("GITHUB_OUTPUT")
		os.Unsetenv("GITHUB_REPOSITORY")
		os.Unsetenv("RUNNER_TEMP")
	}

	return tempDir, cleanup
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
Binary amd64=>test-${TAG}-linux-amd64 > test > test
Binary arm64=>test-${TAG}-linux-arm64 > test > test
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
		Now:         time.Now(),
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

					if !strings.Contains(string(ebuildContent), "amd64? (") {
						t.Errorf("Expected ebuild to contain amd64 USE condition but it didn't:\n%s", ebuildContent)
					}
					if !strings.Contains(string(ebuildContent), "arm64? (") {
						t.Errorf("Expected ebuild to contain arm64 USE condition but it didn't:\n%s", ebuildContent)
					}
				}
			}
		}
	}
}

func TestSmokeEndToEndExecution_WebBinary(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	binDir := filepath.Join(tempDir, "bin")
	mockCommand(t, binDir, "python3", `#!/bin/bash
if [[ "$1" == "$RUNNER_TEMP/g2-pipeline.py" ]]; then
	echo "v2.0.0"
else
	# Fallback to system python3?
	/usr/bin/python3 "$@"
fi
`)

	configString := `Type Web Binary
DownloadBaseUrl https://example.com/download/
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
VersionPipeline get(https://example.com) | rss
`
	inputConfigs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	ic := inputConfigs[0]
	ic.Programs = map[string]*Program{}

	templates, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	base := &GenerateGithubWorkflowBase{
		Version:     "1.0",
		InputConfig: ic,
		ConfigFile:  "test.config",
		Now:         time.Now(),
	}

	ghBase := &GenerateGithubBinaryTemplateData{GenerateGithubWorkflowBase: base}
	data := &GenerateWebBinaryTemplateData{GenerateGithubBinaryTemplateData: ghBase}

	var out bytes.Buffer
	err = templates.ExecuteTemplate(&out, "web-binary.tmpl", data)
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

				// Make sure we have the pipeline partials available
				err := os.MkdirAll(filepath.Join(tempDir, "templates", "_partials"), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(tempDir, "templates", "_partials", "pipeline.py"), []byte("print('2.0.0')"), 0755)
				require.NoError(t, err)

				cmd := exec.Command("bash", "-c", "set -euo pipefail\n"+resolvedScript)
				cmd.Dir = tempDir
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("Failed to execute script for step %v in job %s. Error: %v\nOutput: %s\nScript:\n%s", step["name"], jobName, err, output, resolvedScript)
				}

				if step["name"] == "Process each release" {
					ebuildPath := filepath.Join(tempDir, "app-admin", "test-bin", "test-bin-1.0.0.ebuild")
					if _, err := os.Stat(ebuildPath); os.IsNotExist(err) {
						t.Errorf("Expected ebuild %s to be created, but it was not", ebuildPath)
					}
				}
			}
		}
	}
}
