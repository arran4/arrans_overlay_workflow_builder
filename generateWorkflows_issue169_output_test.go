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
			runScript: runWebAppImageSteps,
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


// runWebAppImageSteps exercises Web AppImage's two run steps as separate
// GitHub Actions processes; the fixture is offline and needs no curl/Python.
func runWebAppImageSteps(t *testing.T, tempDir string, workflow map[string]any) {
	t.Helper()

	rawJobs, exists := workflow["jobs"]
	require.True(t, exists, "generated workflow has no jobs")
	jobs, ok := rawJobs.(map[string]any)
	require.True(t, ok, "jobs has unexpected type %T", rawJobs)
	rawJob, exists := jobs["fetch-latest"]
	require.True(t, exists, "Web AppImage must generate jobs.fetch-latest")
	job, ok := rawJob.(map[string]any)
	require.True(t, ok, "fetch-latest has unexpected type %T", rawJob)
	rawSteps, exists := job["steps"]
	require.True(t, exists, "fetch-latest has no steps")
	steps, ok := rawSteps.([]any)
	require.True(t, ok, "steps have unexpected type %T", rawSteps)

	var findStep, updateStep string
	for _, rawStep := range steps {
		step, ok := rawStep.(map[string]any)
		require.True(t, ok, "unexpected step %T", rawStep)
		name, _ := step["name"].(string)
		switch name {
		case "Determine latest AppImage":
			findStep, ok = step["run"].(string)
			require.True(t, ok, "%s has no run block", name)
		case "Update ebuild":
			updateStep, ok = step["run"].(string)
			require.True(t, ok, "%s has no run block", name)
		}
	}
	require.NotEmpty(t, findStep)
	require.NotEmpty(t, updateStep)

	const category = "app-admin"
	const packageName = "test-webapp-appimage"
	findStep = resolveGithubEnvForPackage(findStep, category, packageName)
	updateStep = resolveGithubEnvForPackage(updateStep, category, packageName)
	require.NotContains(t, findStep, "${{", "unresolved GitHub Actions expression")
	require.NotContains(t, updateStep, "${{", "unresolved GitHub Actions expression")

	binDir := filepath.Join(tempDir, "bin")
	// Fail if these tools are called: CustomDownloadUrl and CustomVersionSource
	// already supply a fixed version and URL, so no network is necessary.
	mockCommand(t, binDir, "curl", "#!/bin/sh\necho 'unexpected curl' >&2\nexit 99\n")
	mockCommand(t, binDir, "python3", "#!/bin/sh\necho 'unexpected python3' >&2\nexit 99\n")

	githubEnv := filepath.Join(tempDir, "github_env")
	githubOutput := filepath.Join(tempDir, "github_output")
	require.NoError(t, os.WriteFile(githubEnv, nil, 0600))
	require.NoError(t, os.WriteFile(githubOutput, nil, 0600))
	baseEnv := append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"RUNNER_TEMP="+tempDir,
		"GITHUB_ENV="+githubEnv,
		"GITHUB_OUTPUT="+githubOutput,
		"P="+packageName,
	)
	run := func(name, script string, env []string) {
		t.Helper()
		cmd := exec.Command("bash", "-c", "set -euo pipefail\n"+script)
		cmd.Dir = tempDir
		cmd.Env = env
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s failed:\n%s", name, string(output))
	}
	run("Determine latest AppImage", findStep, baseEnv)

	// GITHUB_ENV is transferred between processes by Actions. Treat it as
	// key=value data; never source/eval it as arbitrary shell code.
	envBytes, err := os.ReadFile(githubEnv)
	require.NoError(t, err)
	exported := map[string]string{}
	for _, line := range strings.Split(string(envBytes), "\n") {
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		require.True(t, found, "malformed GITHUB_ENV entry %q", line)
		require.NotEmpty(t, key)
		exported[key] = value
	}
	for _, key := range []string{"version", "appimage_url", "appimage_extension"} {
		require.NotEmpty(t, exported[key], "missing GITHUB_ENV key %s", key)
	}
	require.Equal(t, "1.0.0", exported["version"])
	require.Equal(t, "AppImage", exported["appimage_extension"])
	require.Equal(t, "https://example.com/downloads/test-webapp-1.0.0.AppImage", exported["appimage_url"])

	updateEnv := append(append([]string{}, baseEnv...),
		"version="+exported["version"],
		"appimage_url="+exported["appimage_url"],
		"appimage_extension="+exported["appimage_extension"],
	)
	run("Update ebuild", updateStep, updateEnv)
}
