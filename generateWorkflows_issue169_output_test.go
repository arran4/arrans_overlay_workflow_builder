package arrans_overlay_workflow_builder

import (
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

func TestIssue169PhysicalEbuildOutputAllFamilies(t *testing.T) {
	testDate := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)

	type familyTest struct {
		name         string
		configString string
		category     string
		ebuildName   string
		version      string
		runScript    func(t *testing.T, tempDir string, workflow map[string]any)
	}

	families := []familyTest{
		{
			name: "Github Binary Release",
			configString: `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test Binary
Homepage https://www.test.io/
License Proprietary
GeneratedEbuildGentooAuthors True
GeneratedEbuildSourceLicense GPL-2.0-only
Binary amd64=>test-${TAG}-linux-amd64 > test > test
`,
			category:   "app-admin",
			ebuildName: "test-bin",
			version:    "1.0.0",
			runScript: func(t *testing.T, tempDir string, workflow map[string]any) {
				binDir := filepath.Join(tempDir, "bin")
				mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\nexit 0\n")
				mockCommand(t, binDir, "gh", "#!/bin/bash\necho v1.0.0\nexit 0\n")

				steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
				var script string
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						break
					}
				}
				require.NotEmpty(t, script)

				script = resolveGithubEnvForPackage(script, "app-admin", "test-bin")
				script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")

				githubOutput := filepath.Join(tempDir, "github_output")
				_ = os.WriteFile(githubOutput, []byte(""), 0644)
				githubEnv := filepath.Join(tempDir, "github_env")
				_ = os.WriteFile(githubEnv, []byte(""), 0644)

				cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=test-bin\nset -euo pipefail\n"+script)
				cmd.Dir = tempDir
				cmd.Env = append(os.Environ(),
					"RUNNER_TEMP="+tempDir,
					"GITHUB_OUTPUT="+githubOutput,
					"GITHUB_ENV="+githubEnv,
				)
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "Github Binary execution failed: %s", string(out))
			},
		},
		{
			name: "Web Binary",
			configString: `Type Web Binary
DownloadBaseUrl https://example.com/downloads/
TagsCommand echo "1.2.3"
EbuildName test-webbin-bin
Category app-admin
Description Test Web Binary
Homepage https://www.test.io/
License Apache-2.0
GentooAuthors true
Binary amd64=>test-${TAG}-x86_64.tar.gz > test > test
`,
			category:   "app-admin",
			ebuildName: "test-webbin-bin",
			version:    "1.2.3",
			runScript: func(t *testing.T, tempDir string, workflow map[string]any) {
				binDir := filepath.Join(tempDir, "bin")
				mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\nexit 0\n")
				mockCommand(t, binDir, "curl", "#!/bin/bash\nexit 0\n")

				steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
				var script string
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						break
					}
				}
				require.NotEmpty(t, script)

				script = resolveGithubEnvForPackage(script, "app-admin", "test-webbin-bin")
				script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")

				githubOutput := filepath.Join(tempDir, "github_output")
				_ = os.WriteFile(githubOutput, []byte(""), 0644)
				githubEnv := filepath.Join(tempDir, "github_env")
				_ = os.WriteFile(githubEnv, []byte(""), 0644)

				cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=test-webbin-bin\nset -euo pipefail\n"+script)
				cmd.Dir = tempDir
				cmd.Env = append(os.Environ(),
					"RUNNER_TEMP="+tempDir,
					"GITHUB_OUTPUT="+githubOutput,
					"GITHUB_ENV="+githubEnv,
				)
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "Web Binary execution failed: %s", string(out))
			},
		},
		{
			name: "Github AppImage Release",
			configString: `Type Github AppImage Release
GithubProjectUrl https://github.com/test/test
EbuildName test-appimage
Category app-admin
Description Test AppImage
Homepage https://www.test.io/
License MIT
ProgramName testapp
Binary amd64=>test-${TAG}.AppImage > test.AppImage
`,
			category:   "app-admin",
			ebuildName: "test-appimage",
			version:    "1.0.0",
			runScript: func(t *testing.T, tempDir string, workflow map[string]any) {
				binDir := filepath.Join(tempDir, "bin")
				mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\nexit 0\n")
				mockCommand(t, binDir, "gh", "#!/bin/bash\necho v1.0.0\nexit 0\n")

				steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
				var script string
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						break
					}
				}
				require.NotEmpty(t, script)

				script = resolveGithubEnvForPackage(script, "app-admin", "test-appimage")
				script = strings.ReplaceAll(script, "${{ env.testapp_release_name_amd64 }}", "test-${tag}.AppImage")
				script = strings.ReplaceAll(script, "${{ env.testapp_appimage_installed_name }}", "test.AppImage")
				script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")

				githubOutput := filepath.Join(tempDir, "github_output")
				_ = os.WriteFile(githubOutput, []byte(""), 0644)
				githubEnv := filepath.Join(tempDir, "github_env")
				_ = os.WriteFile(githubEnv, []byte(""), 0644)

				cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=test-appimage\nset -euo pipefail\n"+script)
				cmd.Dir = tempDir
				cmd.Env = append(os.Environ(),
					"RUNNER_TEMP="+tempDir,
					"GITHUB_OUTPUT="+githubOutput,
					"GITHUB_ENV="+githubEnv,
				)
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "Github AppImage execution failed: %s", string(out))
			},
		},
		{
			name: "Web AppImage",
			configString: `Type Web AppImage
DownloadPageUrl https://example.com/downloads/
CustomDownloadUrl https://example.com/downloads/test-webapp-1.0.0.AppImage
CustomVersionSource echo "1.0.0"
EbuildName test-webapp-appimage
Category app-admin
Description Test Web AppImage
Homepage https://www.test.io/
License BSD-2-Clause
GeneratedEbuildSourceLicense BSD-2-Clause
ProgramName testapp
Binary amd64=>test-webapp-1.0.0.AppImage > testapp.AppImage
`,
			category:   "app-admin",
			ebuildName: "test-webapp-appimage",
			version:    "1.0.0",
			runScript: func(t *testing.T, tempDir string, workflow map[string]any) {
				binDir := filepath.Join(tempDir, "bin")
				mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\nexit 0\n")
				mockCommand(t, binDir, "curl", "#!/bin/bash\nexit 0\n")

				jobs := workflow["jobs"].(map[string]any)
				var steps []any
				if job, ok := jobs["fetch-latest"].(map[string]any); ok {
					steps = job["steps"].([]any)
				} else if job, ok := jobs["check-and-create-ebuild"].(map[string]any); ok {
					steps = job["steps"].([]any)
				}
				require.NotEmpty(t, steps, "workflow steps missing")

				var findStep, updateStep string
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Determine latest AppImage" {
						findStep = step["run"].(string)
					}
					if step["name"] == "Update ebuild" {
						updateStep = step["run"].(string)
					}
				}
				require.NotEmpty(t, findStep, "Determine latest AppImage step missing")
				require.NotEmpty(t, updateStep, "Update ebuild step missing")

				githubEnvFile := filepath.Join(tempDir, "github_env")
				err := os.WriteFile(githubEnvFile, []byte(""), 0644)
				require.NoError(t, err)

				// Step 1: Run find step
				findCmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nset -euo pipefail\n"+findStep)
				findCmd.Dir = tempDir
				findCmd.Env = append(os.Environ(),
					"RUNNER_TEMP="+tempDir,
					"GITHUB_ENV="+githubEnvFile,
				)
				findOut, err := findCmd.CombinedOutput()
				require.NoError(t, err, "Find AppImage failed: %s", string(findOut))

				// Parse exported env vars from GITHUB_ENV
				envBytes, err := os.ReadFile(githubEnvFile)
				require.NoError(t, err)
				envVars := make(map[string]string)
				for _, line := range strings.Split(string(envBytes), "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						envVars[parts[0]] = parts[1]
					}
				}

				// Step 2: Run update step
				updateStep = resolveGithubEnvForPackage(updateStep, "app-admin", "test-webapp-appimage")
				updateStep = strings.ReplaceAll(updateStep, "${{ env.workflow_filename }}", "test.yml")
				updateCmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=test-webapp-appimage\nset -euo pipefail\n"+updateStep)
				updateCmd.Dir = tempDir
				updateCmd.Env = append(os.Environ(),
					"RUNNER_TEMP="+tempDir,
					"appimage_url="+envVars["appimage_url"],
					"appimage_extension="+envVars["appimage_extension"],
					"version="+envVars["version"],
				)
				updateOut, err := updateCmd.CombinedOutput()
				require.NoError(t, err, "Update ebuild failed: %s", string(updateOut))
			},
		},
		{
			name: "Github Cmake Release",
			configString: `Type Github Cmake Release
GithubProjectUrl https://github.com/test/test
EbuildName test-cmake
Category app-admin
Description Test CMake
Homepage https://www.test.io/
License MIT
GentooAuthors true
SourceLicense Apache-2.0
`,
			category:   "app-admin",
			ebuildName: "test-cmake",
			version:    "1.0.0",
			runScript: func(t *testing.T, tempDir string, workflow map[string]any) {
				binDir := filepath.Join(tempDir, "bin")
				mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\nexit 0\n")
				mockCommand(t, binDir, "gh", "#!/bin/bash\necho v1.0.0\nexit 0\n")
				mockCommand(t, binDir, "curl", "#!/bin/bash\ncat <<'CM'\nfind_package(Qt6 REQUIRED COMPONENTS Core)\nCM\nexit 0\n")

				steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
				var script string
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						break
					}
				}
				require.NotEmpty(t, script)

				script = resolveGithubEnvForPackage(script, "app-admin", "test-cmake")
				script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")

				githubOutput := filepath.Join(tempDir, "github_output")
				_ = os.WriteFile(githubOutput, []byte(""), 0644)
				githubEnv := filepath.Join(tempDir, "github_env")
				_ = os.WriteFile(githubEnv, []byte(""), 0644)

				cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=test-cmake\nset -euo pipefail\n"+script)
				cmd.Dir = tempDir
				cmd.Env = append(os.Environ(),
					"RUNNER_TEMP="+tempDir,
					"GITHUB_OUTPUT="+githubOutput,
					"GITHUB_ENV="+githubEnv,
				)
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "Github CMake execution failed: %s", string(out))
			},
		},
	}

	for _, fam := range families {
		t.Run(fam.name, func(t *testing.T) {
			tempDir, cleanup := setupHermeticEnvironment(t)
			defer cleanup()

			configs, err := ParseInputConfigReader(strings.NewReader(fam.configString))
			require.NoError(t, err)
			require.Len(t, configs, 1)

			tmpls, err := ParseWorkflowTemplates()
			require.NoError(t, err)

			err = configs[0].GenerateGithubWorkflow("test.config", testDate, tmpls, tempDir, "1.0.0")
			require.NoError(t, err)

			yamlPath := filepath.Join(tempDir, fmt.Sprintf("%s-%s-update.yaml", fam.category, fam.ebuildName))
			yamlData, err := os.ReadFile(yamlPath)
			require.NoError(t, err)

			var workflow map[string]any
			err = yaml.Unmarshal(yamlData, &workflow)
			require.NoError(t, err)

			fam.runScript(t, tempDir, workflow)

			ebuildPath := filepath.Join(tempDir, fam.category, fam.ebuildName, fmt.Sprintf("%s-%s.ebuild", fam.ebuildName, fam.version))
			ebuildContent, err := os.ReadFile(ebuildPath)
			require.NoError(t, err, "generated ebuild file must exist: %s", ebuildPath)
			contentStr := string(ebuildContent)

			switch fam.name {
			case "Github Binary Release":
				require.Contains(t, contentStr, "# Copyright 2026 Gentoo Authors")
				require.Contains(t, contentStr, "# SPDX-License-Identifier: GPL-2.0-only")
				require.NotContains(t, contentStr, "Distributed under the terms of the GNU General Public License v2")
				require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"Proprietary\"")
			case "Web Binary":
				require.Contains(t, contentStr, "# Copyright 2026 Gentoo Authors")
				require.Contains(t, contentStr, "# Distributed under the terms of the GNU General Public License v2")
				require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"Apache-2.0\"")
			case "Github AppImage Release":
				require.Contains(t, contentStr, "# SPDX-License-Identifier: MIT")
				require.NotContains(t, contentStr, "Gentoo Authors")
				require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"MIT\"")
			case "Web AppImage":
				require.Contains(t, contentStr, "# SPDX-License-Identifier: BSD-2-Clause")
				require.NotContains(t, contentStr, "Gentoo Authors")
				require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"BSD-2-Clause\"")
			case "Github Cmake Release":
				require.Contains(t, contentStr, "# Copyright 2026 Gentoo Authors")
				require.Contains(t, contentStr, "# SPDX-License-Identifier: Apache-2.0")
				require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"MIT\"")
			}
		})
	}
}
