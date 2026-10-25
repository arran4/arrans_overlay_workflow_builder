package arrans_overlay_workflow_builder

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIssue171DosymIndentation(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/derailed/k9s
EbuildName k9s-bin
Category app-misc
Description test
License MIT
ProgramName k9s
Symlink k9s=>/usr/bin/k9s
Binary amd64=>k9s_Linux_amd64.tar.gz > k9s > k9s`

	data := NewGenerateGithubBinaryTemplateDataFromString(cfg)

	templates, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	var buf bytes.Buffer
	err = templates.ExecuteTemplate(&buf, "github-binary.tmpl", data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "echo '\tdosym \"/usr/bin/k9s\" \"k9s\"'") {
		t.Errorf("Expected dosym to be indented with a tab and double quotes around args, but got output: %s", output)
	}
}

func TestIssue171PhysicalEbuildDosymBothGenerators(t *testing.T) {
	testDate := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	expectedDosymLine := "\tdosym \"/usr/bin/k9s\" \"k9s\""

	type testCase struct {
		generatorType string
		configString  string
		mockSetup     func(t *testing.T, binDir string)
	}

	cases := []testCase{
		{
			generatorType: "Github Binary Release",
			configString: `Type Github Binary Release
GithubProjectUrl https://github.com/derailed/k9s
EbuildName k9s-bin
Category app-misc
Description test k9s
Homepage https://k9scli.io/
License Apache-2.0
ProgramName k9s
Symlink k9s=>/usr/bin/k9s
Binary amd64=>k9s_Linux_amd64.tar.gz > k9s > k9s
`,
			mockSetup: func(t *testing.T, binDir string) {
				mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\nexit 0\n")
				mockCommand(t, binDir, "gh", "#!/bin/bash\necho v1.0.0\nexit 0\n")
			},
		},
		{
			generatorType: "Web Binary",
			configString: `Type Web Binary
DownloadBaseUrl https://example.com/downloads/
TagsCommand echo "1.0.0"
EbuildName k9s-bin
Category app-misc
Description test k9s
Homepage https://k9scli.io/
License Apache-2.0
ProgramName k9s
Symlink k9s=>/usr/bin/k9s
Binary amd64=>k9s_Linux_amd64.tar.gz > k9s > k9s
`,
			mockSetup: func(t *testing.T, binDir string) {
				mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\nexit 0\n")
				mockCommand(t, binDir, "curl", "#!/bin/bash\nexit 0\n")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.generatorType, func(t *testing.T) {
			tempDir, cleanup := setupHermeticEnvironment(t)
			defer cleanup()

			configs, err := ParseInputConfigReader(strings.NewReader(tc.configString))
			require.NoError(t, err)
			require.Len(t, configs, 1)

			tmpls, err := ParseWorkflowTemplates()
			require.NoError(t, err)

			err = configs[0].GenerateGithubWorkflow("k9s.config", testDate, tmpls, tempDir, "1.0.0")
			require.NoError(t, err)

			yamlPath := filepath.Join(tempDir, "app-misc-k9s-bin-update.yaml")
			yamlData, err := os.ReadFile(yamlPath)
			require.NoError(t, err)

			var workflow map[string]any
			err = yaml.Unmarshal(yamlData, &workflow)
			require.NoError(t, err)

			steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
			var script string
			var stepEnvMap map[string]any
			for _, stepItem := range steps {
				step := stepItem.(map[string]any)
				if step["name"] == "Process each release" {
					script = step["run"].(string)
					if envVal, ok := step["env"].(map[string]any); ok {
						stepEnvMap = envVal
					}
					break
				}
			}
			require.NotEmpty(t, script)

			binDir := filepath.Join(tempDir, "bin")
			tc.mockSetup(t, binDir)

			script = resolveGithubEnvForPackage(script, "app-misc", "k9s-bin")
			script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")
			script = strings.ReplaceAll(script, "${{ env.k9s_release_name_amd64 }}", "k9s_Linux_amd64.tar.gz")
			script = strings.ReplaceAll(script, "${{ env.k9s_binary_installed_name }}", "k9s")
			script = strings.ReplaceAll(script, "${{ env.k9s_binary_archived_name_amd64 }}", "k9s")

			githubOutput := filepath.Join(tempDir, "github_output")
			_ = os.WriteFile(githubOutput, []byte(""), 0644)
			githubEnv := filepath.Join(tempDir, "github_env")
			_ = os.WriteFile(githubEnv, []byte(""), 0644)

			cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=k9s-bin\nPV=1.0.0\nset -euo pipefail\n"+script)
			cmd.Dir = tempDir
			cmd.Env = append(os.Environ(),
				"RUNNER_TEMP="+tempDir,
				"GITHUB_OUTPUT="+githubOutput,
				"GITHUB_ENV="+githubEnv,
			)
			for k, v := range stepEnvMap {
				if s, ok := v.(string); ok {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
				}
			}
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s execution failed: %s", tc.generatorType, string(out))

			ebuildPath := filepath.Join(tempDir, "app-misc", "k9s-bin", "k9s-bin-1.0.0.ebuild")
			ebuildBytes, err := os.ReadFile(ebuildPath)
			require.NoError(t, err, "physical ebuild on disk must exist: %s", ebuildPath)

			ebuildContent := string(ebuildBytes)
			require.Contains(t, ebuildContent, expectedDosymLine,
				"physical ebuild on disk for %s must contain tab-indented dosym directive", tc.generatorType)

			lines := strings.Split(ebuildContent, "\n")
			var foundDosym string
			for _, line := range lines {
				if strings.Contains(line, "dosym") {
					foundDosym = line
					break
				}
			}
			require.Equal(t, expectedDosymLine, foundDosym,
				"dosym line on disk must have exact tab indentation and arguments")
		})
	}
}
