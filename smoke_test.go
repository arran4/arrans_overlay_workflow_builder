package arrans_overlay_workflow_builder

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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

					if !strings.Contains(string(ebuildContent), "amd64? (") {
						t.Errorf("Expected ebuild to contain amd64 USE condition but it didn't:\n%s", ebuildContent)
					}
					if !strings.Contains(string(ebuildContent), "arm64? (") {
						t.Errorf("Expected ebuild to contain arm64 USE condition but it didn't:\n%s", ebuildContent)
					}

					// Ensure SRC_URI has the exact resolved URLs
					expectedAmd64URI := "amd64? (  https://github.com/test/test/releases/download/v1.0.0/${PV} -> ${P}-test-v1.0.0-linux-amd64  )"
					if !strings.Contains(string(ebuildContent), expectedAmd64URI) {
						t.Errorf("Expected ebuild to have exact amd64 SRC_URI: %q, but got:\n%s", expectedAmd64URI, ebuildContent)
					}
					expectedArm64URI := "arm64? (  https://github.com/test/test/releases/download/v1.0.0/${PV} -> ${P}-test-v1.0.0-linux-arm64  )"
					if !strings.Contains(string(ebuildContent), expectedArm64URI) {
						t.Errorf("Expected ebuild to have exact arm64 SRC_URI: %q, but got:\n%s", expectedArm64URI, ebuildContent)
					}

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

					expectedAmd64Call := "manifest upsert-from-url https://github.com/test/test/releases/download/v1.0.0/1.0.0 test-bin-1.0.0-test-v1.0.0-linux-amd64 ./app-admin/test-bin/Manifest"

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

					expectedArm64Call := "manifest upsert-from-url https://github.com/test/test/releases/download/v1.0.0/1.0.0 test-bin-1.0.0-test-v1.0.0-linux-arm64 ./app-admin/test-bin/Manifest"

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
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body><a href="downloads/v0.2.6/which_browser-0.2.6+44-linux.deb">download</a></body></html>`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	configString := `Type Web Binary
EbuildName which-browser-bin
Category www-client
Description Test Web Binary
Homepage https://example.com
License MIT
DownloadBaseUrl ` + ts.URL + `/downloads/v${VERSION}/
VersionPipeline get(` + ts.URL + `/) | html_links | regex(which_browser-([^/]+)-linux[.]deb$) | exactly_one
DownloadPipeline get(` + ts.URL + `/) | html_links | regex(.*/downloads/v[^/]+/which_browser-[^/]+-linux[.]deb$) | exactly_one
Workaround Version Replacement => s/\+/_p/g
ProgramName which-browser
Binary amd64=>which_browser-${TAG}-linux.deb > which_browser > which-browser
`

	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	base := &GenerateGithubWorkflowBase{InputConfig: configs[0]}
	data := &GenerateWebBinaryTemplateData{
		GenerateGithubBinaryTemplateData: &GenerateGithubBinaryTemplateData{
			GenerateGithubWorkflowBase: base,
		},
	}
	data.Programs = map[string]*Program{
		"which-browser": {
			ProgramName: "which-browser",
			Binary: map[string][]string{
				"amd64": {"which_browser-${TAG}-linux.deb", "which_browser", "which-browser"},
			},
		},
	}

	tmpl, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "web-binary.tmpl", data)
	require.NoError(t, err)

	yamlOutput := buf.String()

	binDir := filepath.Join(tempDir, "bin")

	mockCommand(t, binDir, "python3", `#!/bin/bash
exec /usr/bin/python3 "$1" "${@:2}"
`)

	mockCommand(t, binDir, "g2", `#!/bin/bash
if [[ "$1" == "metadata" ]]; then
	echo "g2 metadata called"
	touch "${@: -1}"
elif [[ "$1" == "ebuild" && "$2" == "next-revision" ]]; then
	echo "${@: -1}"
elif [[ "$1" == "ebuild" && "$2" == "deduplicate" ]]; then
	echo "g2 deduplicate called"
elif [[ "$1" == "cache" && "$2" == "generate" ]]; then
	echo "g2 cache generate called"
elif [[ "$1" == "manifest" && "$2" == "upsert-from-url" ]]; then
	echo "g2 manifest called"
	echo "$@" >> "${RUNNER_TEMP}/g2_manifest_log.txt"
else
	echo "Unknown g2 command: $@" >&2
	exit 1
fi
`)

	workflow := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(yamlOutput), &workflow)
	require.NoError(t, err)

	var processReleasesScript string
	jobs := workflow["jobs"].(map[string]interface{})
	for _, jobInterface := range jobs {
		job := jobInterface.(map[string]interface{})
		steps := job["steps"].([]interface{})
		for _, stepInterface := range steps {
			step := stepInterface.(map[string]interface{})
			if step["name"] == "Process each release" || step["name"] == "Process releases" {
				processReleasesScript = step["run"].(string)
				break
			}
		}
	}

	processReleasesScript = resolveGithubEnvForPackage(processReleasesScript, "www-client", "which-browser-bin")
	processReleasesScript = strings.ReplaceAll(processReleasesScript, "${{ env.which-browser_binary_archived_name_amd64 }}", "which_browser")
	processReleasesScript = strings.ReplaceAll(processReleasesScript, "${{ env.which-browser_binary_installed_name }}", "which-browser")

	t.Setenv("RUNNER_TEMP", tempDir)
	t.Setenv("GITHUB_ENV", filepath.Join(tempDir, "github_env"))
	t.Setenv("GITHUB_OUTPUT", filepath.Join(tempDir, "github_output"))

	err = os.WriteFile(filepath.Join(tempDir, "github_env"), []byte(""), 0644)
	require.NoError(t, err)

	cmd := exec.Command("bash", "-c", processReleasesScript)
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Process release script failed with output: %s\nScript:\n%s", string(out), processReleasesScript)
	}
	require.NoError(t, err, "Process release script failed: %s", string(out))

	require.Contains(t, string(out), "Content changed or new version for 0.2.6_p44")
	require.Contains(t, string(out), "g2 manifest called")

	ebuildPath := filepath.Join(tempDir, "www-client", "which-browser-bin", "which-browser-bin-0.2.6_p44.ebuild")
	ebuildContent, err := os.ReadFile(ebuildPath)
	require.NoError(t, err, "Ebuild should have been generated")
	require.Contains(t, string(ebuildContent), ts.URL+"/downloads/v0.2.6/which_browser-0.2.6+44-linux.deb -> ${P}-which_browser-0.2.6+44-linux.deb", "SRC_URI must map precisely to Gentoo variable while preserving raw artifact name")

	manifestLogPath := filepath.Join(tempDir, "g2_manifest_log.txt")
	manifestLogContent, err := os.ReadFile(manifestLogPath)
	require.NoError(t, err)

	logLines := strings.Split(strings.TrimSpace(string(manifestLogContent)), "\n")
	var validLines []string
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			validLines = append(validLines, line)
		}
	}
	require.Equal(t, 1, len(validLines), "g2 manifest should be called exactly once")
	require.Equal(t, "manifest upsert-from-url "+ts.URL+"/downloads/v0.2.6/which_browser-0.2.6+44-linux.deb which-browser-bin-0.2.6_p44-which_browser-0.2.6+44-linux.deb ./www-client/which-browser-bin/Manifest", strings.TrimSpace(validLines[0]), "g2 manifest must receive exact resolved URL and expected distfile mapping")
}

func TestSmokeEndToEndExecution_WebBinary(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	binDir := filepath.Join(tempDir, "bin")
	mockCommand(t, binDir, "python3", `#!/bin/bash
EXPECTED_CMD="get(https://example.com) | rss"
if [[ "$#" -eq 2 && "$1" == "$RUNNER_TEMP/g2-pipeline.py" && "$2" == "$EXPECTED_CMD" ]]; then
	echo "2.0.0"
else
	echo "Unknown python3 command: $@" >&2
	exit 1
fi
`)

	configString := `Type Web Binary
DownloadBaseUrl https://example.com/download/${VERSION}/
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
VersionPipeline get(https://example.com) | rss
Binary amd64=>test-${VERSION}.tar.gz > test > test
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
				if step["name"] == "Commit and push changes" {
					resolvedScript = "generated_tag=2.0.0\n" + resolvedScript
				}
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

					if len(validLines) != 1 {
						t.Errorf("Expected exactly 1 g2 manifest call, got %d. Log:\n%s", len(validLines), string(logData))
					}

					expectedCall := "manifest upsert-from-url https://example.com/download/2.0.0/2.0.0 test-bin-2.0.0-test-2.0.0.tar.gz ./app-admin/test-bin/Manifest"

					found := false
					for _, line := range logLines {
						if strings.Contains(line, expectedCall) {
							found = true
							break
						}
					}

					if !found {
						t.Errorf("Expected g2 manifest upsert exact call for web binary: %q\nGot log:\n%s", expectedCall, string(logData))
					}

					outData, err := os.ReadFile(os.Getenv("GITHUB_OUTPUT"))
					require.NoError(t, err)
					if !strings.Contains(string(outData), "generated_tag=2.0.0") {
						t.Errorf("Expected generated_tag=2.0.0 to be in GITHUB_OUTPUT, got: %s", string(outData))
					}
				}
			}
		}
	}
}
