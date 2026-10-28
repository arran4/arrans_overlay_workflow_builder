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
				var stepEnv map[string]any
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						if envVal, ok := step["env"].(map[string]any); ok {
							stepEnv = envVal
						}
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
				for k, v := range stepEnv {
					if s, ok := v.(string); ok {
						cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
					}
				}
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
				var stepEnv map[string]any
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						if envVal, ok := step["env"].(map[string]any); ok {
							stepEnv = envVal
						}
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
				for k, v := range stepEnv {
					if s, ok := v.(string); ok {
						cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
					}
				}
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
				var stepEnv map[string]any
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						if envVal, ok := step["env"].(map[string]any); ok {
							stepEnv = envVal
						}
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
				for k, v := range stepEnv {
					if s, ok := v.(string); ok {
						cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
					}
				}
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
			runScript:  runWebAppImageSteps,
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
				var stepEnv map[string]any
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						if envVal, ok := step["env"].(map[string]any); ok {
							stepEnv = envVal
						}
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
				for k, v := range stepEnv {
					if s, ok := v.(string); ok {
						cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
					}
				}
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

			lines := strings.Split(contentStr, "\n")
			require.NotEmpty(t, lines)

			switch fam.name {
			case "Github Binary Release":
				require.Equal(t, "# Copyright 2026 Gentoo Authors", lines[0])
				require.Equal(t, "# SPDX-License-Identifier: GPL-2.0-only", lines[1])
				require.NotContains(t, contentStr, "Distributed under the terms of the GNU General Public License v2")
				require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"Proprietary\"")
			case "Web Binary":
				require.Equal(t, "# Copyright 2026 Gentoo Authors", lines[0])
				require.Equal(t, "# Distributed under the terms of the GNU General Public License v2", lines[1])
				require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"Apache-2.0\"")
			case "Github AppImage Release":
				require.Equal(t, "# SPDX-License-Identifier: MIT", lines[0])
				require.NotContains(t, contentStr, "Gentoo Authors")
				require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"MIT\"")
			case "Web AppImage":
				require.Equal(t, "# SPDX-License-Identifier: BSD-2-Clause", lines[0])
				require.NotContains(t, contentStr, "Gentoo Authors")
				require.Contains(t, contentStr, "LICENSE=\"\"\nLICENSE+=\"BSD-2-Clause\"")
			case "Github Cmake Release":
				require.Equal(t, "# Copyright 2026 Gentoo Authors", lines[0])
				require.Equal(t, "# SPDX-License-Identifier: Apache-2.0", lines[1])
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
	var updateStepEnv map[string]any
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
			if envMap, ok := step["env"].(map[string]any); ok {
				updateStepEnv = envMap
			}
		}
	}
	require.NotEmpty(t, findStep)
	require.NotEmpty(t, updateStep)

	const category = "app-admin"
	const packageName = "test-webapp-appimage"
	findStep = resolveGithubEnvForPackage(findStep, category, packageName)
	updateStep = resolveGithubEnvForPackage(updateStep, category, packageName)
	// Package-specific variables are present in the actual generated YAML env.
	// Resolve them as GitHub Actions would before invoking Bash.
	rawWorkflowEnv, exists := workflow["env"]
	require.True(t, exists, "generated workflow has no env")
	workflowEnv, ok := rawWorkflowEnv.(map[string]any)
	require.True(t, ok, "workflow env has unexpected type %T", rawWorkflowEnv)
	for key, rawValue := range workflowEnv {
		value, ok := rawValue.(string)
		require.True(t, ok, "workflow env %s has unexpected type %T", key, rawValue)
		findStep = strings.ReplaceAll(findStep, "${{ env."+key+" }}", value)
		updateStep = strings.ReplaceAll(updateStep, "${{ env."+key+" }}", value)
	}
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
	for k, v := range updateStepEnv {
		if s, ok := v.(string); ok {
			updateEnv = append(updateEnv, fmt.Sprintf("%s=%s", k, s))
		}
	}
	run("Update ebuild", updateStep, updateEnv)
}

func TestIssue169CLIPrecedenceSubprocess(t *testing.T) {
	type cliTestCase struct {
		name          string
		globalArgs    []string
		subArgs       []string
		configContent string
		wantFirstLine string
		wantLicense   string
	}

	baseConfig := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test Binary
Homepage https://www.test.io/
License Proprietary
Binary amd64=>test-${TAG}-linux-amd64 > test > test
`

	entryOverrideFalseConfig := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test Binary
Homepage https://www.test.io/
License Proprietary
GentooAuthors false
Binary amd64=>test-${TAG}-linux-amd64 > test > test
`

	entryOverrideTrueConfig := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test Binary
Homepage https://www.test.io/
License Proprietary
GentooAuthors true
Binary amd64=>test-${TAG}-linux-amd64 > test > test
`

	tests := []cliTestCase{
		{
			name:          "global_true_subcommand_omitted",
			globalArgs:    []string{"--gentoo-authors", "true"},
			subArgs:       []string{},
			configContent: baseConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_true_subcommand_false",
			globalArgs:    []string{"--gentoo-authors", "true"},
			subArgs:       []string{"-gentoo-authors", "false"},
			configContent: baseConfig,
			wantFirstLine: "# SPDX-License-Identifier: MIT",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_false_subcommand_true",
			globalArgs:    []string{"--gentoo-authors", "false"},
			subArgs:       []string{"-gentoo-authors", "true"},
			configContent: baseConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_source_license_subcommand_omitted",
			globalArgs:    []string{"--source-license", "Apache-2.0"},
			subArgs:       []string{},
			configContent: baseConfig,
			wantFirstLine: "# SPDX-License-Identifier: Apache-2.0",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_source_license_subcommand_override",
			globalArgs:    []string{"--source-license", "Apache-2.0"},
			subArgs:       []string{"-source-license", "BSD-2-Clause"},
			configContent: baseConfig,
			wantFirstLine: "# SPDX-License-Identifier: BSD-2-Clause",
			wantLicense:   "Proprietary",
		},
		{
			name:          "entry_override_disables_global_true",
			globalArgs:    []string{"--gentoo-authors", "true"},
			subArgs:       []string{},
			configContent: entryOverrideFalseConfig,
			wantFirstLine: "# SPDX-License-Identifier: MIT",
			wantLicense:   "Proprietary",
		},
		{
			name:          "entry_override_enables_when_global_false",
			globalArgs:    []string{"--gentoo-authors", "false"},
			subArgs:       []string{},
			configContent: entryOverrideTrueConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
		{
			name:          "alias_generated_ebuild_gentoo_authors",
			globalArgs:    []string{"--generated-ebuild-gentoo-authors=true"},
			subArgs:       []string{},
			configContent: baseConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
		{
			name:          "alias_generated_ebuild_source_license",
			globalArgs:    []string{"--generated-ebuild-source-license=GPL-3.0-or-later"},
			subArgs:       []string{},
			configContent: baseConfig,
			wantFirstLine: "# SPDX-License-Identifier: GPL-3.0-or-later",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_bare_gentoo_authors_preceding_generate",
			globalArgs:    []string{"--gentoo-authors"},
			subArgs:       []string{},
			configContent: baseConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_single_dash_bare_gentoo_authors",
			globalArgs:    []string{"-gentoo-authors"},
			subArgs:       []string{},
			configContent: baseConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_bare_generated_ebuild_gentoo_authors",
			globalArgs:    []string{"--generated-ebuild-gentoo-authors"},
			subArgs:       []string{},
			configContent: baseConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_single_dash_bare_generated_ebuild_gentoo_authors",
			globalArgs:    []string{"-generated-ebuild-gentoo-authors"},
			subArgs:       []string{},
			configContent: baseConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_false_equals",
			globalArgs:    []string{"--gentoo-authors=false"},
			subArgs:       []string{},
			configContent: baseConfig,
			wantFirstLine: "# SPDX-License-Identifier: MIT",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_bare_subcommand_override_false_space",
			globalArgs:    []string{"--gentoo-authors"},
			subArgs:       []string{"-gentoo-authors", "false"},
			configContent: baseConfig,
			wantFirstLine: "# SPDX-License-Identifier: MIT",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_bare_subcommand_override_false_equals",
			globalArgs:    []string{"--gentoo-authors"},
			subArgs:       []string{"-gentoo-authors=false"},
			configContent: baseConfig,
			wantFirstLine: "# SPDX-License-Identifier: MIT",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_false_subcommand_bare_true",
			globalArgs:    []string{"--gentoo-authors=false"},
			subArgs:       []string{"-gentoo-authors"},
			configContent: baseConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
		{
			name:          "global_false_subcommand_bare_generated_ebuild_gentoo_authors",
			globalArgs:    []string{"--gentoo-authors=false"},
			subArgs:       []string{"-generated-ebuild-gentoo-authors"},
			configContent: baseConfig,
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
			wantLicense:   "Proprietary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, cleanup := setupHermeticEnvironment(t)
			defer cleanup()

			configPath := filepath.Join(tempDir, "test.config")
			require.NoError(t, os.WriteFile(configPath, []byte(tc.configContent), 0644))

			outputDir := filepath.Join(tempDir, "workflows")
			require.NoError(t, os.MkdirAll(outputDir, 0755))

			var cmdArgs []string
			cmdArgs = append(cmdArgs, "run", "./cmd/generate/main.go")
			cmdArgs = append(cmdArgs, tc.globalArgs...)
			cmdArgs = append(cmdArgs, "generate", "workflows")
			cmdArgs = append(cmdArgs, tc.subArgs...)
			cmdArgs = append(cmdArgs,
				"-input-file", configPath,
				"-output-dir", outputDir,
				"-force-date", "2026-08-13 00:00:00 +0000 UTC",
			)

			cmd := exec.Command("go", cmdArgs...)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "CLI command failed: %s\nOutput:\n%s", strings.Join(cmdArgs, " "), string(out))

			yamlPath := filepath.Join(outputDir, "app-admin-test-bin-update.yaml")
			yamlData, err := os.ReadFile(yamlPath)
			require.NoError(t, err)

			var workflow map[string]any
			require.NoError(t, yaml.Unmarshal(yamlData, &workflow))

			steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
			var script string
			var stepEnv map[string]any
			for _, stepItem := range steps {
				step := stepItem.(map[string]any)
				if step["name"] == "Process each release" {
					script = step["run"].(string)
					if envVal, ok := step["env"].(map[string]any); ok {
						stepEnv = envVal
					}
					break
				}
			}
			require.NotEmpty(t, script)

			binDir := filepath.Join(tempDir, "bin")
			mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\nexit 0\n")
			mockCommand(t, binDir, "gh", "#!/bin/bash\necho v1.0.0\nexit 0\n")

			script = resolveGithubEnvForPackage(script, "app-admin", "test-bin")
			script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")

			githubOutput := filepath.Join(tempDir, "github_output")
			_ = os.WriteFile(githubOutput, []byte(""), 0644)
			githubEnv := filepath.Join(tempDir, "github_env")
			_ = os.WriteFile(githubEnv, []byte(""), 0644)

			execCmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=test-bin\nset -euo pipefail\n"+script)
			execCmd.Dir = tempDir
			execCmd.Env = append(os.Environ(),
				"RUNNER_TEMP="+tempDir,
				"GITHUB_OUTPUT="+githubOutput,
				"GITHUB_ENV="+githubEnv,
			)
			for k, v := range stepEnv {
				if s, ok := v.(string); ok {
					execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, s))
				}
			}
			execOut, err := execCmd.CombinedOutput()
			require.NoError(t, err, "execution failed: %s", string(execOut))

			ebuildPath := filepath.Join(tempDir, "app-admin", "test-bin", "test-bin-1.0.0.ebuild")
			ebuildContent, err := os.ReadFile(ebuildPath)
			require.NoError(t, err)

			lines := strings.Split(string(ebuildContent), "\n")
			require.NotEmpty(t, lines)
			require.Equal(t, tc.wantFirstLine, lines[0], "first physical line of generated ebuild must match expected header")

			contentStr := string(ebuildContent)
			expectedLicenseAssignment := fmt.Sprintf("LICENSE=\"\"\nLICENSE+=\"%s\"", tc.wantLicense)
			require.Contains(t, contentStr, expectedLicenseAssignment, "package LICENSE must remain independent of header")
		})
	}
}

func TestIssue169CLIErrorsAndConflictsSubprocess(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	configPath := filepath.Join(tempDir, "test.config")
	baseConfig := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test Binary
Homepage https://www.test.io/
License Proprietary
Binary amd64=>test-${TAG}-linux-amd64 > test > test
`
	require.NoError(t, os.WriteFile(configPath, []byte(baseConfig), 0644))

	outputDir := filepath.Join(tempDir, "workflows")
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	type errorTestCase struct {
		name             string
		cliArgs          []string
		wantErrSubstring string
	}

	tests := []errorTestCase{
		{
			name: "invalid_boolean_equals_global",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"--gentoo-authors=flase",
				"generate", "workflows",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: "invalid boolean value for flag --gentoo-authors: flase",
		},
		{
			name: "invalid_boolean_space_global",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"--gentoo-authors", "flase",
				"generate", "workflows",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: "Unknown command flase",
		},
		{
			name: "conflicting_global_gentoo_authors_aliases",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"--gentoo-authors=true",
				"--generated-ebuild-gentoo-authors=false",
				"generate", "workflows",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: "conflicting flags: --gentoo-authors=true and --generated-ebuild-gentoo-authors=false",
		},
		{
			name: "conflicting_global_same_flag_values",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"--gentoo-authors=true",
				"--gentoo-authors=false",
				"generate", "workflows",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: "conflicting flags: multiple --gentoo-authors with conflicting values",
		},
		{
			name: "conflicting_global_source_license_aliases",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"--source-license=Apache-2.0",
				"--generated-ebuild-source-license=MIT",
				"generate", "workflows",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: `conflicting flags: --source-license="Apache-2.0" and --generated-ebuild-source-license="MIT"`,
		},
		{
			name: "missing_required_global_source_license_before_command",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"--source-license",
				"generate", "workflows",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: "missing required value for flag --source-license",
		},
		{
			name: "missing_required_global_source_license_equals",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"--source-license=",
				"generate", "workflows",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: "missing required value for flag --source-license",
		},
		{
			name: "conflicting_subcommand_gentoo_authors_aliases",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"generate", "workflows",
				"-gentoo-authors=true",
				"-generated-ebuild-gentoo-authors=false",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: "conflicting flags: -gentoo-authors=true and -generated-ebuild-gentoo-authors=false",
		},
		{
			name: "conflicting_subcommand_source_license_aliases",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"generate", "workflows",
				"-source-license=Apache-2.0",
				"-generated-ebuild-source-license=MIT",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: `conflicting flags: -source-license="Apache-2.0" and -generated-ebuild-source-license="MIT"`,
		},
		{
			name: "invalid_boolean_subcommand_equals",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"generate", "workflows",
				"-gentoo-authors=flase",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: `invalid boolean value "flase" for -gentoo-authors`,
		},
		{
			name: "invalid_boolean_subcommand_space",
			cliArgs: []string{
				"run", "./cmd/generate/main.go",
				"generate", "workflows",
				"-gentoo-authors", "flase",
				"-input-file", configPath,
				"-output-dir", outputDir,
			},
			wantErrSubstring: "Unknown command flase",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("go", tc.cliArgs...)
			out, err := cmd.CombinedOutput()
			require.Error(t, err, "command expected to fail: %s\nOutput:\n%s", strings.Join(tc.cliArgs, " "), string(out))
			require.Contains(t, string(out), tc.wantErrSubstring, "stderr/stdout must contain expected error explanation")
		})
	}
}

func TestIssue169AllCombinationsPhysicalEbuild(t *testing.T) {
	testDate := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)

	familyTemplates := []struct {
		name       string
		baseConfig string
		category   string
		ebuildName string
		version    string
		runScript  func(t *testing.T, tempDir string, workflow map[string]any)
	}{
		{
			name: "Github Binary",
			baseConfig: `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test Binary
Homepage https://www.test.io/
License Proprietary
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
				var stepEnv map[string]any
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						if envVal, ok := step["env"].(map[string]any); ok {
							stepEnv = envVal
						}
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
				for k, v := range stepEnv {
					if s, ok := v.(string); ok {
						cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
					}
				}
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "execution failed: %s", string(out))
			},
		},
		{
			name: "Web Binary",
			baseConfig: `Type Web Binary
DownloadBaseUrl https://example.com/downloads/
TagsCommand echo "1.0.0"
EbuildName test-webbin-bin
Category app-admin
Description Test Web Binary
Homepage https://www.test.io/
License Apache-2.0
Binary amd64=>test-${TAG}-x86_64.tar.gz > test > test
`,
			category:   "app-admin",
			ebuildName: "test-webbin-bin",
			version:    "1.0.0",
			runScript: func(t *testing.T, tempDir string, workflow map[string]any) {
				binDir := filepath.Join(tempDir, "bin")
				mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\nexit 0\n")
				mockCommand(t, binDir, "curl", "#!/bin/bash\nexit 0\n")

				steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
				var script string
				var stepEnv map[string]any
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						if envVal, ok := step["env"].(map[string]any); ok {
							stepEnv = envVal
						}
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
				for k, v := range stepEnv {
					if s, ok := v.(string); ok {
						cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
					}
				}
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "execution failed: %s", string(out))
			},
		},
		{
			name: "Github AppImage",
			baseConfig: `Type Github AppImage Release
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
				var stepEnv map[string]any
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						if envVal, ok := step["env"].(map[string]any); ok {
							stepEnv = envVal
						}
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
				for k, v := range stepEnv {
					if s, ok := v.(string); ok {
						cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
					}
				}
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "execution failed: %s", string(out))
			},
		},
		{
			name: "Web AppImage",
			baseConfig: `Type Web AppImage
DownloadPageUrl https://example.com/downloads/
CustomDownloadUrl https://example.com/downloads/test-webapp-1.0.0.AppImage
CustomVersionSource echo "1.0.0"
EbuildName test-webapp-appimage
Category app-admin
Description Test Web AppImage
Homepage https://www.test.io/
License BSD-2-Clause
ProgramName testapp
Binary amd64=>test-webapp-1.0.0.AppImage > testapp.AppImage
`,
			category:   "app-admin",
			ebuildName: "test-webapp-appimage",
			version:    "1.0.0",
			runScript:  runWebAppImageSteps,
		},
		{
			name: "Github Cmake",
			baseConfig: `Type Github Cmake Release
GithubProjectUrl https://github.com/test/test
EbuildName test-cmake
Category app-admin
Description Test CMake
Homepage https://www.test.io/
License MIT
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
				var stepEnv map[string]any
				for _, stepItem := range steps {
					step := stepItem.(map[string]any)
					if step["name"] == "Process each release" {
						script = step["run"].(string)
						if envVal, ok := step["env"].(map[string]any); ok {
							stepEnv = envVal
						}
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
				for k, v := range stepEnv {
					if s, ok := v.(string); ok {
						cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
					}
				}
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "execution failed: %s", string(out))
			},
		},
	}

	combinations := []struct {
		comboName     string
		extraConfig   string
		options       []any
		wantFirstLine string
	}{
		{
			comboName:     "global_true_entry_unset",
			options:       []any{OptGentooAuthors(true)},
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
		},
		{
			comboName:     "global_false_entry_unset",
			options:       []any{OptGentooAuthors(false)},
			wantFirstLine: "# SPDX-License-Identifier: MIT",
		},
		{
			comboName:     "global_true_entry_false",
			extraConfig:   "GentooAuthors false\n",
			options:       []any{OptGentooAuthors(true)},
			wantFirstLine: "# SPDX-License-Identifier: MIT",
		},
		{
			comboName:     "global_false_entry_true",
			extraConfig:   "GentooAuthors true\n",
			options:       []any{OptGentooAuthors(false)},
			wantFirstLine: "# Copyright 2026 Gentoo Authors",
		},
		{
			comboName:     "global_source_license_entry_unset",
			options:       []any{OptSourceLicense("Apache-2.0")},
			wantFirstLine: "# SPDX-License-Identifier: Apache-2.0",
		},
		{
			comboName:     "global_source_license_entry_override",
			extraConfig:   "SourceLicense BSD-2-Clause\n",
			options:       []any{OptSourceLicense("Apache-2.0")},
			wantFirstLine: "# SPDX-License-Identifier: BSD-2-Clause",
		},
	}

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	for _, fam := range familyTemplates {
		for _, combo := range combinations {
			t.Run(fam.name+"/"+combo.comboName, func(t *testing.T) {
				tempDir, cleanup := setupHermeticEnvironment(t)
				defer cleanup()

				fullConfig := fam.baseConfig + combo.extraConfig
				configs, err := ParseInputConfigReader(strings.NewReader(fullConfig))
				require.NoError(t, err)
				require.Len(t, configs, 1)

				err = configs[0].GenerateGithubWorkflow("test.config", testDate, tmpls, tempDir, "1.0.0", combo.options...)
				require.NoError(t, err)

				yamlPath := filepath.Join(tempDir, fmt.Sprintf("%s-%s-update.yaml", fam.category, fam.ebuildName))
				yamlData, err := os.ReadFile(yamlPath)
				require.NoError(t, err)

				var workflow map[string]any
				require.NoError(t, yaml.Unmarshal(yamlData, &workflow))

				fam.runScript(t, tempDir, workflow)

				ebuildPath := filepath.Join(tempDir, fam.category, fam.ebuildName, fmt.Sprintf("%s-%s.ebuild", fam.ebuildName, fam.version))
				ebuildContent, err := os.ReadFile(ebuildPath)
				require.NoError(t, err)

				lines := strings.Split(string(ebuildContent), "\n")
				require.NotEmpty(t, lines)
				require.Equal(t, combo.wantFirstLine, lines[0], "first physical line mismatch for %s/%s", fam.name, combo.comboName)

				// Ensure package LICENSE= in ebuild is preserved
				contentStr := string(ebuildContent)
				require.Contains(t, contentStr, fmt.Sprintf("LICENSE=\"\"\nLICENSE+=\"%s\"", configs[0].License))
			})
		}
	}
}
