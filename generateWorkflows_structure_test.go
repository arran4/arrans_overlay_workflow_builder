package arrans_overlay_workflow_builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGeneratedGithubBinaryWorkflowStructure(t *testing.T) {
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

	content, err := os.ReadFile(filepath.Join(tempDir, "app-admin-test-bin-update.yaml"))
	require.NoError(t, err)

	// A generated file existing on disk says nothing about whether Actions can
	// execute it. Parse the actual YAML and assert the build pipeline's shape.
	var workflow struct {
		Name        string `yaml:"name"`
		Permissions struct {
			Contents string `yaml:"contents"`
		} `yaml:"permissions"`
		On  map[string]any `yaml:"on"`
		Env struct {
			Category string `yaml:"ecn"`
			Package  string `yaml:"epn"`
		} `yaml:"env"`
		Jobs map[string]struct {
			RunsOn string `yaml:"runs-on"`
			Steps  []struct {
				Name string            `yaml:"name"`
				ID   string            `yaml:"id"`
				Uses string            `yaml:"uses"`
				Run  string            `yaml:"run"`
				If   string            `yaml:"if"`
				With map[string]string `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(content, &workflow))
	require.NotEmpty(t, workflow.Name)
	require.Equal(t, "write", workflow.Permissions.Contents)
	require.Contains(t, workflow.On, "workflow_dispatch")
	require.Contains(t, workflow.On, "schedule")
	require.Equal(t, "app-admin", workflow.Env.Category)
	require.Equal(t, "test-bin", workflow.Env.Package)

	job, exists := workflow.Jobs["check-and-create-ebuild"]
	require.True(t, exists, "missing primary generation job")
	require.Equal(t, "ubuntu-latest", job.RunsOn)
	require.NotEmpty(t, job.Steps)

	positions := make(map[string]int, len(job.Steps))
	for i, step := range job.Steps {
		require.NotEmpty(t, step.Name, "unnamed step at index %d", i)
		_, seen := positions[step.Name]
		require.False(t, seen, "duplicate workflow step %q", step.Name)
		positions[step.Name] = i
		require.NotEqual(t, step.Run != "", step.Uses != "",
			"step %q must have exactly one executable run or uses", step.Name)
	}
	for _, name := range []string{
		"Checkout repository", "Install g2", "Process each release",
		"Deduplicate ebuilds", "Generate md5-cache", "Lint output", "Commit and push changes",
	} {
		_, present := positions[name]
		require.True(t, present, "required workflow step %q absent", name)
	}
	require.Less(t, positions["Checkout repository"], positions["Process each release"])
	require.Less(t, positions["Install g2"], positions["Process each release"])
	require.Less(t, positions["Process each release"], positions["Deduplicate ebuilds"])
	require.Less(t, positions["Deduplicate ebuilds"], positions["Generate md5-cache"])
	require.Less(t, positions["Generate md5-cache"], positions["Lint output"])
	require.Less(t, positions["Lint output"], positions["Commit and push changes"])
	require.Equal(t, "process_releases", job.Steps[positions["Process each release"]].ID)
	require.NotEmpty(t, job.Steps[positions["Process each release"]].Run)
	require.Contains(t, job.Steps[positions["Generate md5-cache"]].If, "process_releases.outputs.generated_tag")
	require.Contains(t, job.Steps[positions["Lint output"]].If, "process_releases.outputs.generated_tag")
	require.Contains(t, job.Steps[positions["Commit and push changes"]].If, "process_releases.outputs.generated_tag")

	lintAction := job.Steps[positions["Lint output"]].With["action"]
	require.Equal(t, "lint -fail-severity error . ${{ env.ecn }}/${{ env.epn }}", lintAction)
	require.NotContains(t, lintAction, "PG0802")
	commitRun := job.Steps[positions["Commit and push changes"]].Run
	require.Contains(t, commitRun, "metadata/md5-cache")
	require.Contains(t, commitRun, "metadata/cache")
	require.Contains(t, commitRun, "git ls-files")
	require.NotContains(t, commitRun, "fi || true")
}

func TestGenerateMd5CacheConfigurations(t *testing.T) {
	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	type templateCase struct {
		name         string
		configFormat string
		outFile      string
	}

	cases := []templateCase{
		{
			name: "Github Binary",
			configFormat: `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
Category app-admin
EbuildName test-bin
Description Test
Homepage https://example.test/
License BSD-2-Clause
Binary amd64=>test-${VERSION}.tar.gz > test > test
%s`,
			outFile: "app-admin-test-bin-update.yaml",
		},
		{
			name: "Github AppImage",
			configFormat: `Type Github AppImage Release
GithubProjectUrl https://github.com/test/test
Category app-misc
EbuildName test-appimage
Description Test
Homepage https://example.test/
License BSD-2-Clause
Binary amd64=>test-${TAG}.AppImage > test.AppImage
%s`,
			outFile: "app-misc-test-appimage-update.yaml",
		},
		{
			name: "Github CMake",
			configFormat: `Type Github Cmake Release
GithubProjectUrl https://github.com/test/test
Category app-admin
EbuildName test-cmake
Description Test
Homepage https://example.test/
License BSD-2-Clause
%s`,
			outFile: "app-admin-test-cmake-update.yaml",
		},
		{
			name: "Web Binary",
			configFormat: `Type Web Binary
Category app-admin
EbuildName test-web-bin
Description Test
Homepage https://example.test/
License BSD-2-Clause
DownloadBaseUrl https://example.test/downloads/
TagsCommand curl -s https://example.test/releases
Binary amd64=>test-${VERSION}.tar.gz > test > test
%s`,
			outFile: "app-admin-test-web-bin-update.yaml",
		},
		{
			name: "Web AppImage",
			configFormat: `Type Web AppImage
DownloadPageUrl https://example.test/downloads/
Category app-misc
EbuildName test-web-appimage
Description Test
Homepage https://example.test/
License BSD-2-Clause
Binary amd64=>test-${VERSION}-x86_64.AppImage > test.AppImage
%s`,
			outFile: "app-misc-test-web-appimage-update.yaml",
		},
	}

	type parsedWorkflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string            `yaml:"name"`
				With map[string]string `yaml:"with"`
				Run  string            `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name+"/Disabled", func(t *testing.T) {
			tempDir, cleanup := setupHermeticEnvironment(t)
			defer cleanup()

			configContent := fmt.Sprintf(tc.configFormat, "")
			configs, err := ParseInputConfigReader(strings.NewReader(configContent))
			require.NoError(t, err)
			require.Len(t, configs, 1)

			err = configs[0].GenerateGithubWorkflow("test.config", time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), tmpls, tempDir, "1.0.0")
			require.NoError(t, err)

			data, err := os.ReadFile(filepath.Join(tempDir, tc.outFile))
			require.NoError(t, err)

			var wf parsedWorkflow
			require.NoError(t, yaml.Unmarshal(data, &wf))
			var job struct {
				Steps []struct {
					Name string            `yaml:"name"`
					With map[string]string `yaml:"with"`
					Run  string            `yaml:"run"`
				} `yaml:"steps"`
			}
			if j, ok := wf.Jobs["check-and-create-ebuild"]; ok {
				job = j
			} else if j, ok := wf.Jobs["fetch-latest"]; ok {
				job = j
			} else {
				t.Fatalf("neither check-and-create-ebuild nor fetch-latest found in workflow")
			}

			var lintAction, commitRun, cacheRun string
			for _, step := range job.Steps {
				if step.Name == "Generate md5-cache" {
					cacheRun = step.Run
				}
				if step.Name == "Lint output" {
					lintAction = step.With["action"]
				}
				if step.Name == "Commit and push changes" {
					commitRun = step.Run
				}
			}

			require.Equal(t, "lint -fail-severity error . ${{ env.ecn }}/${{ env.epn }}", lintAction)
			require.NotContains(t, lintAction, "PG0802", "PG0802 must never be conditionally suppressed")
			require.Contains(t, commitRun, "metadata/md5-cache", "Commit and push changes must always stage package-scoped md5-cache changes")
			require.Contains(t, commitRun, "metadata/cache", "Commit and push changes must always stage package-scoped metadata/cache changes")
			require.Contains(t, commitRun, "git ls-files", "Commit and push changes must use index-aware pathspec handling for deleted cache files")
			require.NotContains(t, commitRun, "fi || true", "Commit and push changes must never mask push/rebase failures")
			require.Contains(t, cacheRun, "masters_line=$(grep -E '^[ \\t]*masters[ \\t]*=' metadata/layout.conf",
				"Generate md5-cache step must inspect declared masters in layout.conf")
			require.Contains(t, cacheRun, "unavailable masters permitted by CI policy",
				"Generate md5-cache step must handle unavailable masters permitted by CI policy")
			require.Contains(t, cacheRun, "canonical cache metadata requires Portage evaluation",
				"Generate md5-cache step must handle Portage evaluation requirement")
		})

		t.Run(tc.name+"/EnabledViaConfigFeature", func(t *testing.T) {
			tempDir, cleanup := setupHermeticEnvironment(t)
			defer cleanup()

			configContent := fmt.Sprintf(tc.configFormat, "Feature Generate Md5 Cache\n")
			configs, err := ParseInputConfigReader(strings.NewReader(configContent))
			require.NoError(t, err)
			require.Len(t, configs, 1)

			err = configs[0].GenerateGithubWorkflow("test.config", time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), tmpls, tempDir, "1.0.0")
			require.NoError(t, err)

			data, err := os.ReadFile(filepath.Join(tempDir, tc.outFile))
			require.NoError(t, err)

			var wf parsedWorkflow
			require.NoError(t, yaml.Unmarshal(data, &wf))
			var job struct {
				Steps []struct {
					Name string            `yaml:"name"`
					With map[string]string `yaml:"with"`
					Run  string            `yaml:"run"`
				} `yaml:"steps"`
			}
			if j, ok := wf.Jobs["check-and-create-ebuild"]; ok {
				job = j
			} else if j, ok := wf.Jobs["fetch-latest"]; ok {
				job = j
			} else {
				t.Fatalf("neither check-and-create-ebuild nor fetch-latest found in workflow")
			}

			var lintAction, commitRun, cacheRun string
			for _, step := range job.Steps {
				if step.Name == "Generate md5-cache" {
					cacheRun = step.Run
				}
				if step.Name == "Lint output" {
					lintAction = step.With["action"]
				}
				if step.Name == "Commit and push changes" {
					commitRun = step.Run
				}
			}

			require.Equal(t, "lint -fail-severity error . ${{ env.ecn }}/${{ env.epn }}", lintAction)
			require.NotContains(t, lintAction, "PG0802")
			require.Contains(t, commitRun, ":(glob)${cache_root}/${ecn}/${epn}-[0-9]*",
				"enabled GenerateMd5Cache via config must stage package-scoped cache with index-aware pathspec")
			require.NotContains(t, commitRun, "fi || true")
			require.Contains(t, cacheRun, "masters_line=$(grep -E '^[ \\t]*masters[ \\t]*=' metadata/layout.conf")
			require.Contains(t, cacheRun, "unavailable masters permitted by CI policy")
			require.Contains(t, cacheRun, "canonical cache metadata requires Portage evaluation")
		})

		t.Run(tc.name+"/EnabledViaGlobalOption", func(t *testing.T) {
			tempDir, cleanup := setupHermeticEnvironment(t)
			defer cleanup()

			configContent := fmt.Sprintf(tc.configFormat, "")
			configs, err := ParseInputConfigReader(strings.NewReader(configContent))
			require.NoError(t, err)
			require.Len(t, configs, 1)

			err = configs[0].GenerateGithubWorkflow("test.config", time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), tmpls, tempDir, "1.0.0", OptGenerateMd5Cache(true))
			require.NoError(t, err)

			data, err := os.ReadFile(filepath.Join(tempDir, tc.outFile))
			require.NoError(t, err)

			var wf parsedWorkflow
			require.NoError(t, yaml.Unmarshal(data, &wf))
			var job struct {
				Steps []struct {
					Name string            `yaml:"name"`
					With map[string]string `yaml:"with"`
					Run  string            `yaml:"run"`
				} `yaml:"steps"`
			}
			if j, ok := wf.Jobs["check-and-create-ebuild"]; ok {
				job = j
			} else if j, ok := wf.Jobs["fetch-latest"]; ok {
				job = j
			} else {
				t.Fatalf("neither check-and-create-ebuild nor fetch-latest found in workflow")
			}

			var lintAction, commitRun, cacheRun string
			for _, step := range job.Steps {
				if step.Name == "Generate md5-cache" {
					cacheRun = step.Run
				}
				if step.Name == "Lint output" {
					lintAction = step.With["action"]
				}
				if step.Name == "Commit and push changes" {
					commitRun = step.Run
				}
			}

			require.Equal(t, "lint -fail-severity error . ${{ env.ecn }}/${{ env.epn }}", lintAction)
			require.NotContains(t, lintAction, "PG0802")
			require.Contains(t, commitRun, ":(glob)${cache_root}/${ecn}/${epn}-[0-9]*",
				"enabled GenerateMd5Cache via option must stage package-scoped cache with index-aware pathspec")
			require.NotContains(t, commitRun, "fi || true")
			require.Contains(t, cacheRun, "masters_line=$(grep -E '^[ \\t]*masters[ \\t]*=' metadata/layout.conf")
			require.Contains(t, cacheRun, "unavailable masters permitted by CI policy")
			require.Contains(t, cacheRun, "canonical cache metadata requires Portage evaluation")
		})
	}
}

func TestDeduplicateEbuildsStepStructure(t *testing.T) {
	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	type templateCase struct {
		name    string
		config  string
		outFile string
	}

	cases := []templateCase{
		{
			name: "Github Binary",
			config: `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
Category app-admin
EbuildName test-bin
Description Test
Homepage https://example.test/
License BSD-2-Clause
Binary amd64=>test-${VERSION}.tar.gz > test > test
`,
			outFile: "app-admin-test-bin-update.yaml",
		},
		{
			name: "Github AppImage",
			config: `Type Github AppImage Release
GithubProjectUrl https://github.com/test/test
Category app-misc
EbuildName test-appimage
Description Test
Homepage https://example.test/
License BSD-2-Clause
Binary amd64=>test-${TAG}.AppImage > test.AppImage
`,
			outFile: "app-misc-test-appimage-update.yaml",
		},
		{
			name: "Github CMake",
			config: `Type Github Cmake Release
GithubProjectUrl https://github.com/test/test
Category app-admin
EbuildName test-cmake
Description Test
Homepage https://example.test/
License BSD-2-Clause
`,
			outFile: "app-admin-test-cmake-update.yaml",
		},
		{
			name: "Web Binary",
			config: `Type Web Binary
Category app-admin
EbuildName test-web-bin
Description Test
Homepage https://example.test/
License BSD-2-Clause
DownloadBaseUrl https://example.test/downloads/
TagsCommand curl -s https://example.test/releases
Binary amd64=>test-${VERSION}.tar.gz > test > test
`,
			outFile: "app-admin-test-web-bin-update.yaml",
		},
		{
			name: "Web AppImage",
			config: `Type Web AppImage
DownloadPageUrl https://example.test/downloads/
Category app-misc
EbuildName test-web-appimage
Description Test
Homepage https://example.test/
License BSD-2-Clause
Binary amd64=>test-${VERSION}-x86_64.AppImage > test.AppImage
`,
			outFile: "app-misc-test-web-appimage-update.yaml",
		},
	}

	type parsedWorkflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string `yaml:"name"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tempDir, cleanup := setupHermeticEnvironment(t)
			defer cleanup()

			configs, err := ParseInputConfigReader(strings.NewReader(tc.config))
			require.NoError(t, err)
			require.Len(t, configs, 1)

			err = configs[0].GenerateGithubWorkflow("test.config", time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), tmpls, tempDir, "1.0.0")
			require.NoError(t, err)

			data, err := os.ReadFile(filepath.Join(tempDir, tc.outFile))
			require.NoError(t, err)

			var wf parsedWorkflow
			require.NoError(t, yaml.Unmarshal(data, &wf))
			var job struct {
				Steps []struct {
					Name string `yaml:"name"`
					Run  string `yaml:"run"`
				} `yaml:"steps"`
			}
			if j, ok := wf.Jobs["check-and-create-ebuild"]; ok {
				job = j
			} else if j, ok := wf.Jobs["fetch-latest"]; ok {
				job = j
			} else {
				t.Fatalf("neither check-and-create-ebuild nor fetch-latest found in workflow")
			}

			var dedupRun string
			for _, step := range job.Steps {
				if step.Name == "Deduplicate ebuilds" {
					dedupRun = step.Run
				}
			}

			require.NotEmpty(t, dedupRun, "Deduplicate ebuilds step must be present and non-empty")
			require.Contains(t, dedupRun, "backup_dir=$(mktemp -d)", "must snapshot ebuild_dir to backup_dir")
			require.Contains(t, dedupRun, "metadata/md5-cache", "must include metadata/md5-cache in rollback snapshot")
			require.Contains(t, dedupRun, "g2 ebuild deduplicate \"${ebuild_dir}\"", "must run g2 ebuild deduplicate")
			require.Contains(t, dedupRun, "skipped cleanup: directory contains uncertain ebuilds", "must detect uncertain ebuilds cleanup skip")
			require.Contains(t, dedupRun, "g2 manifest verify \"${ebuild_dir}\"", "must verify manifest after deduplication")
			require.NotContains(t, dedupRun, "g2 manifest verify \"${ebuild_dir}\" 2>\"${manifest_verify_err}\" || true", "must not swallow manifest verify failures")
			require.Contains(t, dedupRun, "EXTRA in manifest:", "must detect extra manifest entries")
			require.Contains(t, dedupRun, "retaining older ebuilds to maintain Manifest consistency", "must log notice when deduplication is deferred")
			require.Contains(t, dedupRun, "restore_dedup_snapshot", "must restore full snapshot on fallback or error")
		})
	}
}

