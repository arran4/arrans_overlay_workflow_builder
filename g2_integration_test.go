package arrans_overlay_workflow_builder

import (
	"archive/tar"
	"compress/gzip"
	"debug/buildinfo"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Pinned released g2 version and verification
const (
	pinnedG2Version = "v0.0.107"
)

func verifyG2Available(t *testing.T) string {
	t.Helper()
	g2Path, err := exec.LookPath("g2")
	if err != nil {
		if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatalf("required real g2 %s executable missing: %v", pinnedG2Version, err)
		}
		t.Skipf("optional local real-g2 integration skipped: install released g2 %s (CI requires it)", pinnedG2Version)
	}
	info, err := buildinfo.ReadFile(g2Path)
	require.NoError(t, err, "read g2 executable build metadata")
	require.NoError(t, validateG2BuildInfo(info), "real-g2 dependency must match the pinned release")
	t.Logf("Running real g2 tests against pinned version %s (path: %s)", pinnedG2Version, g2Path)
	return g2Path
}

// g2 v0.0.107 has no version subcommand. The released Go executable carries
// module build metadata (including a +dirty suffix in the official release).
func validateG2BuildInfo(info *debug.BuildInfo) error {
	if info.Path != "github.com/arran4/g2/cmd/g2" || info.Main.Path != "github.com/arran4/g2" ||
		strings.TrimSuffix(info.Main.Version, "+dirty") != pinnedG2Version || info.Main.Replace != nil {
		return fmt.Errorf("expected released g2 %s; executable is %s (%s %s)", pinnedG2Version, info.Path, info.Main.Path, info.Main.Version)
	}
	for _, setting := range info.Settings {
		if setting.Key != "-ldflags" {
			continue
		}
		fields := strings.Fields(setting.Value)
		for i := 0; i+1 < len(fields); i++ {
			if fields[i] == "-X" && fields[i+1] == "main.version="+strings.TrimPrefix(pinnedG2Version, "v") {
				return nil
			}
		}
	}
	return fmt.Errorf("g2 %s is not a version-stamped release executable", pinnedG2Version)
}

func TestPinnedG2BuildInfo(t *testing.T) {
	for _, version := range []string{pinnedG2Version, pinnedG2Version + "+dirty", "(devel)", "v0.0.106", "v0.0.1070"} {
		t.Run(version, func(t *testing.T) {
			info := &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "-ldflags", Value: "-s -w -X main.version=0.0.107"}}, Path: "github.com/arran4/g2/cmd/g2", Main: debug.Module{Path: "github.com/arran4/g2", Version: version}}
			err := validateG2BuildInfo(info)
			if version == pinnedG2Version || version == pinnedG2Version+"+dirty" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
	require.Error(t, validateG2BuildInfo(&debug.BuildInfo{Path: "github.com/arran4/g2/cmd/g2", Main: debug.Module{Path: "github.com/arran4/g2", Version: pinnedG2Version}}))
	require.Error(t, validateG2BuildInfo(&debug.BuildInfo{Main: debug.Module{Version: pinnedG2Version}}))
}

func setupHermeticG2Overlay(t *testing.T) (string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "g2_overlay_test_*")
	require.NoError(t, err)

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	// Create profiles
	profilesDir := filepath.Join(tempDir, "profiles")
	require.NoError(t, os.MkdirAll(profilesDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "repo_name"), []byte("test-overlay\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "eapi"), []byte("8\n"), 0644))

	categories := []string{"app-admin", "app-misc", "app-text", "net-im", "www-misc"}
	require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "categories"), []byte(strings.Join(categories, "\n")+"\n"), 0644))

	// Pre-create category directories so g2 cache generate can read them
	for _, cat := range categories {
		require.NoError(t, os.MkdirAll(filepath.Join(tempDir, cat), 0755))
	}

	// Create metadata
	metadataDir := filepath.Join(tempDir, "metadata")
	require.NoError(t, os.MkdirAll(metadataDir, 0755))
	layoutConf := "masters =\ncache-formats = md5-dict\nthin-manifests = true\n"
	require.NoError(t, os.WriteFile(filepath.Join(metadataDir, "layout.conf"), []byte(layoutConf), 0644))

	// Create licenses
	licensesDir := filepath.Join(tempDir, "licenses")
	require.NoError(t, os.MkdirAll(licensesDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(licensesDir, "MIT"), []byte("MIT License\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(licensesDir, "AGPL-3"), []byte("AGPL-3 License\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(licensesDir, "Apache-2.0"), []byte("Apache-2.0 License\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(licensesDir, "Proprietary"), []byte("Proprietary License\n"), 0644))

	// Create dummy eclasses needed by cmake/xdg
	eclassDir := filepath.Join(tempDir, "eclass")
	require.NoError(t, os.MkdirAll(eclassDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(eclassDir, "cmake.eclass"), []byte("# cmake eclass stub\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(eclassDir, "xdg.eclass"), []byte("# xdg eclass stub\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(eclassDir, "xdg-utils.eclass"), []byte("# xdg-utils eclass stub\n"), 0644))

	return tempDir, cleanup
}

func createTarGzFixture(t *testing.T, filenames []string) []byte {
	t.Helper()
	var buf strings.Builder
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	for _, fname := range filenames {
		content := fmt.Sprintf("#!/bin/sh\necho 'running %s'\n", fname)
		hdr := &tar.Header{
			Name: fname,
			Mode: 0755,
			Size: int64(len(content)),
		}
		require.NoError(t, tarWriter.WriteHeader(hdr))
		_, err := tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzWriter.Close())
	return []byte(buf.String())
}

func resolveWorkflowEnv(script string, workflow map[string]any) string {
	if topEnv, ok := workflow["env"].(map[string]any); ok {
		for k, v := range topEnv {
			if s, ok := v.(string); ok {
				script = strings.ReplaceAll(script, fmt.Sprintf("${{ env.%s }}", k), s)
			}
		}
	}
	script = strings.ReplaceAll(script, "${{secrets.GITHUB_TOKEN}}", "test-token")
	script = strings.ReplaceAll(script, "${{ github.server_url }}", "https://github.com")
	script = strings.ReplaceAll(script, "${{ github.repository }}", "test/test")
	script = strings.ReplaceAll(script, "${{ github.sha }}", "123456")
	script = strings.ReplaceAll(script, "${{ github.ref }}", "refs/heads/main")
	return script
}

// TestRealG2_GithubBinary tests end-to-end ebuild generation and validation
// with real g2 for a multi-architecture Github Binary release.
func TestRealG2_GithubBinary(t *testing.T) {
	verifyG2Available(t)
	overlayDir, cleanup := setupHermeticG2Overlay(t)
	defer cleanup()

	// Start fixture server serving controlled release tarballs
	tarDataAmd64 := createTarGzFixture(t, []string{"mytool", "README.md", "LICENSE"})
	tarDataArm64 := createTarGzFixture(t, []string{"mytool", "README.md", "LICENSE"})
	tarData386 := createTarGzFixture(t, []string{"mytool", "README.md", "LICENSE"})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		switch {
		case strings.Contains(r.URL.Path, "amd64"):
			_, _ = w.Write(tarDataAmd64)
		case strings.Contains(r.URL.Path, "arm64"):
			_, _ = w.Write(tarDataArm64)
		default:
			_, _ = w.Write(tarData386)
		}
	}))
	defer server.Close()

	configString := `Type Github Binary Release
GithubProjectUrl https://github.com/myorg/mytool
EbuildName mytool-bin
Category app-admin
Description Fast developer utility tool
Homepage https://myorg.github.io/mytool
License MIT
Binary amd64=>mytool-${TAG}-linux-amd64.tar.gz > mytool > mytool
Binary arm64=>mytool_${VERSION}_linux_arm64.tar.gz > mytool > mytool
Binary x86=>mytool_${VERSION}_linux_386.tar.gz > mytool > mytool
Document amd64=>mytool-${TAG}-linux-amd64.tar.gz > README.md > README.md
Document amd64=>mytool-${TAG}-linux-amd64.tar.gz > LICENSE > LICENSE
Document x86=>mytool_${VERSION}_linux_386.tar.gz > README.md > README.md
Symlink mytool => mytool-alias
`

	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	testDate := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	err = configs[0].GenerateGithubWorkflow("test.config", testDate, tmpls, overlayDir, "1.0.0")
	require.NoError(t, err)

	yamlPath := filepath.Join(overlayDir, "app-admin-mytool-bin-update.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(yamlData, &workflow))

	steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
	var processScript string
	var stepEnv map[string]any
	for _, s := range steps {
		step := s.(map[string]any)
		if step["name"] == "Process each release" {
			processScript = step["run"].(string)
			if envVal, ok := step["env"].(map[string]any); ok {
				stepEnv = envVal
			}
			break
		}
	}
	require.NotEmpty(t, processScript)

	// Mock gh to return a controlled release tag v1.0.0
	binDir := filepath.Join(overlayDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	ghMock := filepath.Join(binDir, "gh")
	require.NoError(t, os.WriteFile(ghMock, []byte("#!/bin/sh\necho 'v1.0.0'\n"), 0755))

	githubOutput := filepath.Join(overlayDir, "github_output")
	_ = os.WriteFile(githubOutput, []byte(""), 0644)
	githubEnv := filepath.Join(overlayDir, "github_env")
	_ = os.WriteFile(githubEnv, []byte(""), 0644)

	resolvedScript := resolveWorkflowEnv(processScript, workflow)
	// Direct release asset downloads to our local test server
	resolvedScript = strings.ReplaceAll(resolvedScript, "https://github.com/myorg/mytool/releases/download/${tag}/", server.URL+"/")

	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=mytool-bin\nPV=1.0.0\nset -euo pipefail\n"+resolvedScript)
	cmd.Dir = overlayDir
	cmd.Env = append(os.Environ(),
		"RUNNER_TEMP="+overlayDir,
		"GITHUB_OUTPUT="+githubOutput,
		"GITHUB_ENV="+githubEnv,
	)
	for k, v := range stepEnv {
		if s, ok := v.(string); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
		}
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Process each release failed: %s", string(out))

	// Verify emitted ebuild exists
	pkgDir := filepath.Join(overlayDir, "app-admin", "mytool-bin")
	ebuildPath := filepath.Join(pkgDir, "mytool-bin-1.0.0.ebuild")
	ebuildBytes, err := os.ReadFile(ebuildPath)
	require.NoError(t, err, "generated ebuild must exist")
	ebuildContent := string(ebuildBytes)

	// 1. Check Bash syntax and literal variables
	require.Contains(t, ebuildContent, `KEYWORDS="~amd64 ~arm64 ~x86"`)
	require.Equal(t, 1, strings.Count(ebuildContent, "KEYWORDS="), "KEYWORDS must be assigned once")
	require.True(t, strings.HasPrefix(ebuildContent, "# SPDX-License-Identifier: MIT\n"), "default source-header policy")
	require.NotContains(t, ebuildContent, `KEYWORDS=""`)
	require.NotContains(t, ebuildContent, `KEYWORDS+=`)
	require.Contains(t, ebuildContent, `LICENSE+="MIT"`)
	require.Contains(t, ebuildContent, `SRC_URI=`)
	require.Contains(t, ebuildContent, "${P}")
	require.Contains(t, ebuildContent, "${DISTDIR}")
	require.Contains(t, ebuildContent, "${WORKDIR}")
	require.Contains(t, ebuildContent, `dosym "mytool-alias" "mytool"`)

	// 2. Run real g2 cache generate
	cacheCmd := exec.Command("g2", "cache", "generate", "-repo", overlayDir, "app-admin/mytool-bin")
	cacheOut, err := cacheCmd.CombinedOutput()
	require.NoError(t, err, "real g2 cache generate must succeed: %s", string(cacheOut))

	// 3. Inspect generated md5-cache record
	md5CachePath := filepath.Join(overlayDir, "metadata", "md5-cache", "app-admin", "mytool-bin-1.0.0")
	cacheBytes, err := os.ReadFile(md5CachePath)
	require.NoError(t, err, "md5-cache record must exist")
	cacheContent := string(cacheBytes)

	require.Contains(t, cacheContent, "EAPI=8")
	require.Contains(t, cacheContent, "KEYWORDS=~amd64 ~arm64 ~x86")
	require.Contains(t, cacheContent, "LICENSE=MIT")
	require.Contains(t, cacheContent, "SLOT=0")
	require.Contains(t, cacheContent, "_md5_=")

	// Ensure SRC_URI is on a single physical line in md5-cache
	for _, line := range strings.Split(cacheContent, "\n") {
		if strings.HasPrefix(line, "SRC_URI=") {
			require.NotContains(t, line, "\r")
			require.Contains(t, line, "amd64?")
			require.Contains(t, line, "arm64?")
			require.Contains(t, line, "x86?")
			require.Contains(t, line, "-> mytool-bin-1.0.0-mytool-v1.0.0-linux-amd64.tar.gz")
			require.Contains(t, line, "-> mytool-bin-1.0.0-mytool_1.0.0_linux_arm64.tar.gz")
			require.Contains(t, line, "-> mytool-bin-1.0.0-mytool_1.0.0_linux_386.tar.gz")
		}
	}

	verifyGeneratedG2Metadata(t, overlayDir, ebuildPath)

	// 4. Verify Manifest DIST entries match
	manifestPath := filepath.Join(pkgDir, "Manifest")
	manifestBytes, err := os.ReadFile(manifestPath)
	require.NoError(t, err, "Manifest must exist")
	manifestContent := string(manifestBytes)
	require.Contains(t, manifestContent, "DIST mytool-bin-1.0.0-mytool-v1.0.0-linux-amd64.tar.gz")
	require.Contains(t, manifestContent, "DIST mytool-bin-1.0.0-mytool_1.0.0_linux_arm64.tar.gz")
	require.Contains(t, manifestContent, "DIST mytool-bin-1.0.0-mytool_1.0.0_linux_386.tar.gz")

	// 5. Run real g2 lint with overlay policy
	lintCmd := exec.Command("g2", "lint", "-disable-rule", "PG0802", "-fail-severity", "error", overlayDir, "app-admin/mytool-bin")
	lintOut, err := lintCmd.CombinedOutput()
	lintStr := string(lintOut)
	require.NoError(t, err, "real g2 lint must pass cleanly without errors: %s", lintStr)
	require.NotContains(t, lintStr, "[Error]")
}

// TestRealG2_WebBinary tests ebuild generation and validation with real g2
// for a Web Binary release using TagsCommand.
func TestRealG2_WebBinary(t *testing.T) {
	verifyG2Available(t)
	overlayDir, cleanup := setupHermeticG2Overlay(t)
	defer cleanup()

	tarData := createTarGzFixture(t, []string{"webtool"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(tarData)
	}))
	defer server.Close()

	configString := fmt.Sprintf(`Type Web Binary
DownloadBaseUrl %s/
TagsCommand echo "2.1.0"
EbuildName webtool-bin
Category app-admin
Description Web Binary tool
Homepage https://example.org/webtool
License Apache-2.0
Binary amd64=>webtool-${VERSION}-linux-x86_64.tar.gz > webtool > webtool
`, server.URL)

	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	testDate := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	err = configs[0].GenerateGithubWorkflow("test.config", testDate, tmpls, overlayDir, "1.0.0")
	require.NoError(t, err)

	yamlPath := filepath.Join(overlayDir, "app-admin-webtool-bin-update.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(yamlData, &workflow))

	steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
	var processScript string
	var stepEnv map[string]any
	for _, s := range steps {
		step := s.(map[string]any)
		if step["name"] == "Process each release" {
			processScript = step["run"].(string)
			if envVal, ok := step["env"].(map[string]any); ok {
				stepEnv = envVal
			}
			break
		}
	}
	require.NotEmpty(t, processScript)

	resolvedScript := resolveWorkflowEnv(processScript, workflow)

	githubOutput := filepath.Join(overlayDir, "github_output")
	_ = os.WriteFile(githubOutput, []byte(""), 0644)
	githubEnv := filepath.Join(overlayDir, "github_env")
	_ = os.WriteFile(githubEnv, []byte(""), 0644)

	cmd := exec.Command("bash", "-c", "P=webtool-bin\nPV=2.1.0\nset -euo pipefail\n"+resolvedScript)
	cmd.Dir = overlayDir
	cmd.Env = append(os.Environ(),
		"RUNNER_TEMP="+overlayDir,
		"GITHUB_OUTPUT="+githubOutput,
		"GITHUB_ENV="+githubEnv,
	)
	for k, v := range stepEnv {
		if s, ok := v.(string); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
		}
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Web Binary process failed: %s", string(out))

	// Verify ebuild created
	pkgDir := filepath.Join(overlayDir, "app-admin", "webtool-bin")
	ebuildPath := filepath.Join(pkgDir, "webtool-bin-2.1.0.ebuild")
	ebuildBytes, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)
	ebuildContent := string(ebuildBytes)

	require.Contains(t, ebuildContent, `KEYWORDS="~amd64"`)
	require.Contains(t, ebuildContent, `LICENSE+="Apache-2.0"`)

	// Real g2 cache generate
	cacheCmd := exec.Command("g2", "cache", "generate", "-repo", overlayDir, "app-admin/webtool-bin")
	cacheOut, err := cacheCmd.CombinedOutput()
	require.NoError(t, err, "real g2 cache generate failed: %s", string(cacheOut))

	verifyGeneratedG2Metadata(t, overlayDir, ebuildPath)

	// Real g2 lint
	lintCmd := exec.Command("g2", "lint", "-disable-rule", "PG0802", "-fail-severity", "error", overlayDir, "app-admin/webtool-bin")
	lintOut, err := lintCmd.CombinedOutput()
	lintStr := string(lintOut)
	require.NoError(t, err, "real g2 lint must pass cleanly: %s", lintStr)
	require.NotContains(t, lintStr, "[Error]")
}

// TestRealG2_GithubAppImage tests ebuild generation and validation with real g2
// for a Github AppImage release.
func TestRealG2_GithubAppImage(t *testing.T) {
	verifyG2Available(t)
	overlayDir, cleanup := setupHermeticG2Overlay(t)
	defer cleanup()

	// Fixture server for AppImage download
	appImageData := []byte("#!/bin/sh\necho 'AppImage mock'\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(appImageData)
	}))
	defer server.Close()

	configString := `Type Github AppImage Release
GithubProjectUrl https://github.com/myorg/myapp
EbuildName myapp-appimage
Category app-misc
Description Desktop AppImage Application
Homepage https://myorg.github.io/myapp
License AGPL-3
Binary amd64=>myapp-${TAG}-x86_64.AppImage > myapp.AppImage
DesktopFile myapp.desktop
`

	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	testDate := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	err = configs[0].GenerateGithubWorkflow("test.config", testDate, tmpls, overlayDir, "1.0.0")
	require.NoError(t, err)

	yamlPath := filepath.Join(overlayDir, "app-misc-myapp-appimage-update.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(yamlData, &workflow))

	steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
	var processScript string
	var stepEnv map[string]any
	for _, s := range steps {
		step := s.(map[string]any)
		if step["name"] == "Process each release" {
			processScript = step["run"].(string)
			if envVal, ok := step["env"].(map[string]any); ok {
				stepEnv = envVal
			}
			break
		}
	}
	require.NotEmpty(t, processScript)

	binDir := filepath.Join(overlayDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	ghMock := filepath.Join(binDir, "gh")
	require.NoError(t, os.WriteFile(ghMock, []byte("#!/bin/sh\necho 'v1.5.0'\n"), 0755))

	resolvedScript := resolveWorkflowEnv(processScript, workflow)
	// Direct release asset downloads to our local test server
	resolvedScript = strings.ReplaceAll(resolvedScript, "https://github.com/myorg/myapp/releases/download/${tag}/", server.URL+"/")

	githubOutput := filepath.Join(overlayDir, "github_output")
	_ = os.WriteFile(githubOutput, []byte(""), 0644)
	githubEnv := filepath.Join(overlayDir, "github_env")
	_ = os.WriteFile(githubEnv, []byte(""), 0644)

	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=myapp-appimage\nPV=1.5.0\nset -euo pipefail\n"+resolvedScript)
	cmd.Dir = overlayDir
	cmd.Env = append(os.Environ(),
		"RUNNER_TEMP="+overlayDir,
		"GITHUB_OUTPUT="+githubOutput,
		"GITHUB_ENV="+githubEnv,
	)
	for k, v := range stepEnv {
		if s, ok := v.(string); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
		}
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "AppImage process failed: %s", string(out))

	// Verify ebuild
	pkgDir := filepath.Join(overlayDir, "app-misc", "myapp-appimage")
	ebuildPath := filepath.Join(pkgDir, "myapp-appimage-1.5.0.ebuild")
	ebuildBytes, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)
	ebuildContent := string(ebuildBytes)

	require.Contains(t, ebuildContent, `KEYWORDS="~amd64"`)
	require.Contains(t, ebuildContent, `LICENSE+="AGPL-3"`)

	// Real g2 cache generate
	cacheCmd := exec.Command("g2", "cache", "generate", "-repo", overlayDir, "app-misc/myapp-appimage")
	cacheOut, err := cacheCmd.CombinedOutput()
	require.NoError(t, err, "real g2 cache generate failed: %s", string(cacheOut))

	verifyGeneratedG2Metadata(t, overlayDir, ebuildPath)

	// Real g2 lint
	lintCmd := exec.Command("g2", "lint", "-disable-rule", "PG0802", "-fail-severity", "error", overlayDir, "app-misc/myapp-appimage")
	lintOut, err := lintCmd.CombinedOutput()
	lintStr := string(lintOut)
	require.NoError(t, err, "real g2 lint must pass: %s", lintStr)
	require.NotContains(t, lintStr, "[Error]")
}

// TestRealG2_GithubCmake tests ebuild generation and validation with real g2
// for a Github CMake release with dependencies.
func TestRealG2_GithubCmake(t *testing.T) {
	verifyG2Available(t)
	overlayDir, cleanup := setupHermeticG2Overlay(t)
	defer cleanup()

	tarData := createTarGzFixture(t, []string{"CMakeLists.txt"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(tarData)
	}))
	defer server.Close()

	configString := `Type Github Cmake Release
GithubProjectUrl https://github.com/myorg/cmaketool
EbuildName cmaketool
Category app-admin
Description CMake build tool
Homepage https://myorg.github.io/cmaketool
License Apache-2.0
`

	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	testDate := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	err = configs[0].GenerateGithubWorkflow("test.config", testDate, tmpls, overlayDir, "1.0.0")
	require.NoError(t, err)

	yamlPath := filepath.Join(overlayDir, "app-admin-cmaketool-update.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(yamlData, &workflow))

	steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
	var processScript string
	var stepEnv map[string]any
	for _, s := range steps {
		step := s.(map[string]any)
		if step["name"] == "Process each release" {
			processScript = step["run"].(string)
			if envVal, ok := step["env"].(map[string]any); ok {
				stepEnv = envVal
			}
			break
		}
	}
	require.NotEmpty(t, processScript)

	binDir := filepath.Join(overlayDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	ghMock := filepath.Join(binDir, "gh")
	require.NoError(t, os.WriteFile(ghMock, []byte("#!/bin/sh\necho 'v3.0.0'\n"), 0755))

	// Mock curl for CMakeLists inspection returning basic CMake
	curlMock := filepath.Join(binDir, "curl")
	require.NoError(t, os.WriteFile(curlMock, []byte("#!/bin/sh\necho 'find_package(Qt6 REQUIRED)'\n"), 0755))

	resolvedScript := resolveWorkflowEnv(processScript, workflow)
	// Point archive download in g2 manifest upsert to local fixture server
	resolvedScript = strings.ReplaceAll(resolvedScript, "https://github.com/myorg/cmaketool/archive/refs/tags/${tag}.tar.gz", server.URL+"/cmaketool-${tag}.tar.gz")

	githubOutput := filepath.Join(overlayDir, "github_output")
	_ = os.WriteFile(githubOutput, []byte(""), 0644)
	githubEnv := filepath.Join(overlayDir, "github_env")
	_ = os.WriteFile(githubEnv, []byte(""), 0644)

	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=cmaketool\nPV=3.0.0\nset -euo pipefail\n"+resolvedScript)
	cmd.Dir = overlayDir
	cmd.Env = append(os.Environ(),
		"RUNNER_TEMP="+overlayDir,
		"GITHUB_OUTPUT="+githubOutput,
		"GITHUB_ENV="+githubEnv,
	)
	for k, v := range stepEnv {
		if s, ok := v.(string); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
		}
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "CMake process failed: %s", string(out))

	// Verify ebuild
	pkgDir := filepath.Join(overlayDir, "app-admin", "cmaketool")
	ebuildPath := filepath.Join(pkgDir, "cmaketool-3.0.0.ebuild")
	ebuildBytes, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)
	ebuildContent := string(ebuildBytes)

	require.Contains(t, ebuildContent, `KEYWORDS="~amd64"`)
	require.Contains(t, ebuildContent, `LICENSE+="Apache-2.0"`)
	require.Contains(t, ebuildContent, `DEPEND=`)

	// Real g2 cache generate
	cacheCmd := exec.Command("g2", "cache", "generate", "-repo", overlayDir, "app-admin/cmaketool")
	cacheOut, err := cacheCmd.CombinedOutput()
	require.NoError(t, err, "real g2 cache generate failed: %s", string(cacheOut))

	verifyGeneratedG2Metadata(t, overlayDir, ebuildPath)

	// Real g2 lint
	lintCmd := exec.Command("g2", "lint", "-disable-rule", "PG0802", "-fail-severity", "error", overlayDir, "app-admin/cmaketool")
	lintOut, err := lintCmd.CombinedOutput()
	lintStr := string(lintOut)
	require.NoError(t, err, "real g2 lint must pass: %s", lintStr)
	require.NotContains(t, lintStr, "[Error]")
}

// Compare the real parser/cache output with shell-evaluated metadata, then check
// every DIST name, including architecture conditions and rename expressions.
func verifyGeneratedG2Metadata(t *testing.T, overlayDir, ebuildPath string) {
	t.Helper()
	out, err := exec.Command("bash", "-n", ebuildPath).CombinedOutput()
	require.NoError(t, err, "generated ebuild shell syntax: %s", out)
	ebuild, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)
	lines := strings.Split(string(ebuild), "\n")
	var keywordsAssignments int
	for _, line := range lines {
		if strings.HasPrefix(line, "KEYWORDS=") {
			keywordsAssignments++
		}
		require.False(t, strings.HasPrefix(line, "KEYWORDS+="), "KEYWORDS must not be appended to")
		if strings.HasPrefix(line, "SRC_URI=") || strings.HasPrefix(line, "SRC_URI+=") || strings.HasPrefix(line, "KEYWORDS=") {
			assertIssue172Widths(t, []string{line})
		}
	}
	require.Equal(t, 1, keywordsAssignments, "KEYWORDS must be defined exactly once")
	packageName := filepath.Base(filepath.Dir(ebuildPath))
	category := filepath.Base(filepath.Dir(filepath.Dir(ebuildPath)))
	p := strings.TrimSuffix(filepath.Base(ebuildPath), ".ebuild")
	pv := strings.TrimPrefix(p, packageName+"-")
	expected := evalIssue172Field(t, lines, "SRC_URI", fmt.Sprintf("P=%s\nPN=%s\nPV=%s", p, packageName, pv))
	cache, err := os.ReadFile(filepath.Join(overlayDir, "metadata", "md5-cache", category, p))
	require.NoError(t, err)
	var records []string
	for _, line := range strings.Split(string(cache), "\n") {
		if strings.HasPrefix(line, "SRC_URI=") {
			records = append(records, strings.TrimPrefix(line, "SRC_URI="))
		}
	}
	require.Len(t, records, 1, "exactly one physical SRC_URI cache record")
	require.NotContains(t, records[0], "\t")
	require.NotContains(t, records[0], "\r")
	require.Equal(t, strings.Fields(expected), strings.Fields(records[0]))
	tokens := strings.Fields(records[0])
	var expectedDist []string
	for i, token := range tokens {
		if !strings.HasPrefix(token, "http://") && !strings.HasPrefix(token, "https://") {
			continue
		}
		u, err := url.Parse(token)
		require.NoError(t, err)
		name := filepath.Base(u.Path)
		if i+2 < len(tokens) && tokens[i+1] == "->" {
			name = tokens[i+2]
		}
		if !slices.Contains(expectedDist, name) {
			expectedDist = append(expectedDist, name)
		}
	}
	require.NotEmpty(t, expectedDist)
	manifest, err := os.ReadFile(filepath.Join(filepath.Dir(ebuildPath), "Manifest"))
	require.NoError(t, err)
	var actualDist []string
	for _, line := range strings.Split(string(manifest), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[0] == "DIST" {
			actualDist = append(actualDist, fields[1])
		}
	}
	require.ElementsMatch(t, expectedDist, actualDist)
}

// TestIssue194_ReproduceAndVerifySharedUpdaterLintGating reproduces the operational
// failure reported in issue #194 / arrans_overlay PR #991 (run 42464010784)
// and asserts that the updated template ordering and severity threshold unblock publication.
func TestIssue194_ReproduceAndVerifySharedUpdaterLintGating(t *testing.T) {
	verifyG2Available(t)
	overlayDir, cleanup := setupHermeticG2Overlay(t)
	defer cleanup()

	// Ensure md5-cache directory exists so g2 lint evaluates cache presence
	require.NoError(t, os.MkdirAll(filepath.Join(overlayDir, "metadata", "md5-cache", "app-text"), 0755))

	tarAmd64 := createTarGzFixture(t, []string{"header", "README.md", "LICENSE"})
	tarArm64 := createTarGzFixture(t, []string{"header", "README.md", "LICENSE"})
	tar386 := createTarGzFixture(t, []string{"header", "README.md", "LICENSE"})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		path := r.URL.Path
		switch {
		case strings.Contains(path, "386"):
			_, _ = w.Write(tar386)
		case strings.Contains(path, "arm64"):
			_, _ = w.Write(tarArm64)
		default:
			_, _ = w.Write(tarAmd64)
		}
	}))
	defer server.Close()

	// Downstream app-text/header-bin configuration from current.config
	configString := `Type Github Binary Release
GithubProjectUrl https://github.com/arran4/header
Category app-text
EbuildName header-bin.ebuild
Description Go CLI and library for extracting, validating, editing and visualizing website metadata
Homepage https://github.com/arran4/header
License MIT
ProgramName header
Document amd64=>header_${VERSION}_linux_amd64.tar.gz > LICENSE > LICENSE
Document amd64=>header_${VERSION}_linux_amd64.tar.gz > README.md > README.md
Document arm64=>header_${VERSION}_linux_arm64.tar.gz > LICENSE > LICENSE
Document arm64=>header_${VERSION}_linux_arm64.tar.gz > README.md > README.md
Document x86=>header_${VERSION}_linux_386.tar.gz > LICENSE > LICENSE
Document x86=>header_${VERSION}_linux_386.tar.gz > README.md > README.md
Binary amd64=>header_${VERSION}_linux_amd64.tar.gz > header > header
Binary arm64=>header_${VERSION}_linux_arm64.tar.gz > header > header
Binary x86=>header_${VERSION}_linux_386.tar.gz > header > header
`

	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	testDate := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	err = configs[0].GenerateGithubWorkflow("test.config", testDate, tmpls, overlayDir, "1.0.0")
	require.NoError(t, err)

	yamlPath := filepath.Join(overlayDir, "app-text-header-bin-update.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(yamlData, &workflow))

	steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
	stepMap := make(map[string]map[string]any)
	for _, s := range steps {
		step := s.(map[string]any)
		if name, ok := step["name"].(string); ok {
			stepMap[name] = step
		}
	}

	// Verify step existence and attributes
	require.Contains(t, stepMap, "Generate md5-cache")
	require.Contains(t, stepMap, "Lint output")
	require.Contains(t, stepMap, "Commit and push changes")

	lintStep := stepMap["Lint output"]
	withMap := lintStep["with"].(map[string]any)
	actionStr := withMap["action"].(string)
	require.Contains(t, actionStr, "-fail-severity error")
	require.Contains(t, actionStr, ". ${{ env.ecn }}/${{ env.epn }}")

	// Mock gh to return tag v0.0.5
	binDir := filepath.Join(overlayDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	ghMock := filepath.Join(binDir, "gh")
	require.NoError(t, os.WriteFile(ghMock, []byte("#!/bin/sh\necho 'v0.0.5'\n"), 0755))

	githubOutput := filepath.Join(overlayDir, "github_output")
	_ = os.WriteFile(githubOutput, []byte(""), 0644)
	githubEnv := filepath.Join(overlayDir, "github_env")
	_ = os.WriteFile(githubEnv, []byte(""), 0644)

	processStep := stepMap["Process each release"]
	processScript := processStep["run"].(string)
	resolvedProcess := resolveWorkflowEnv(processScript, workflow)
	resolvedProcess = strings.ReplaceAll(resolvedProcess, "https://github.com/arran4/header/releases/download/${tag}/", server.URL+"/arran4/header/releases/download/${tag}/")

	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=header-bin\nPV=0.0.5\nset -euo pipefail\n"+resolvedProcess)
	cmd.Dir = overlayDir
	cmd.Env = append(os.Environ(),
		"RUNNER_TEMP="+overlayDir,
		"GITHUB_OUTPUT="+githubOutput,
		"GITHUB_ENV="+githubEnv,
	)
	if envMap, ok := processStep["env"].(map[string]any); ok {
		for k, v := range envMap {
			if s, ok := v.(string); ok {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
			}
		}
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Process each release failed: %s", string(out))

	pkgDir := filepath.Join(overlayDir, "app-text", "header-bin")
	ebuildPath := filepath.Join(pkgDir, "header-bin-0.0.5.ebuild")
	ebuildBytes, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)
	ebuildContent := string(ebuildBytes)

	// Verify intact, indivisible URLs exceeding 80 characters
	require.Contains(t, ebuildContent, `KEYWORDS="~amd64 ~arm64 ~x86"`)
	var hasLongLine bool
	for _, l := range strings.Split(ebuildContent, "\n") {
		if len(l) > 80 {
			hasLongLine = true
			break
		}
	}
	require.True(t, hasLongLine, "generated ebuild must have line >80 cols to test indivisible URLs")

	// Phase 1: Demonstrate old failure (without cache generate and default fail-severity warning)
	oldLintCmd := exec.Command("g2", "lint", "-disable-rule", "PG0802", overlayDir, "app-text/header-bin")
	oldLintOut, oldLintErr := oldLintCmd.CombinedOutput()
	require.Error(t, oldLintErr, "old lint configuration must fail on warnings: %s", string(oldLintOut))
	require.Contains(t, string(oldLintOut), "exceeding 80 positions")
	require.Contains(t, string(oldLintOut), "Missing md5-cache for ebuild header-bin-0.0.5")

	// Phase 2: Execute new Generate md5-cache step script
	cacheStep := stepMap["Generate md5-cache"]
	cacheScript := resolveWorkflowEnv(cacheStep["run"].(string), workflow)
	cacheCmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nset -euo pipefail\n"+cacheScript)
	cacheCmd.Dir = overlayDir
	cacheOut, err := cacheCmd.CombinedOutput()
	require.NoError(t, err, "Generate md5-cache step failed: %s", string(cacheOut))

	md5CachePath := filepath.Join(overlayDir, "metadata", "md5-cache", "app-text", "header-bin-0.0.5")
	_, err = os.Stat(md5CachePath)
	require.NoError(t, err, "md5-cache file must exist prior to lint")

	// Phase 3: Execute Lint output with -fail-severity error
	resolvedAction := resolveWorkflowEnv(actionStr, workflow)
	lintFields := strings.Fields(resolvedAction)
	require.Equal(t, "lint", lintFields[0])
	newLintCmd := exec.Command("g2", lintFields...)
	newLintCmd.Dir = overlayDir
	newLintOut, err := newLintCmd.CombinedOutput()
	newLintStr := string(newLintOut)
	require.NoError(t, err, "Lint output with -fail-severity error must succeed: %s", newLintStr)

	// Missing md5-cache must be completely eliminated
	require.NotContains(t, newLintStr, "Missing md5-cache")

	// Non-fatal warnings (e.g. line length > 80) must remain visible in output for diagnosis
	require.Contains(t, newLintStr, "exceeding 80 positions")
	require.NotContains(t, newLintStr, "[Error]")
}

// TestIssue194_NegativeGate_BlocksInvalidPublication tests that genuine errors
// in ebuilds, Manifests, or metadata still fail g2 lint under -fail-severity error,
// preventing publication of corrupted or invalid packages.
func TestIssue194_NegativeGate_BlocksInvalidPublication(t *testing.T) {
	verifyG2Available(t)

	tests := []struct {
		name        string
		setup       func(t *testing.T, pkgDir string)
		expectError string
	}{
		{
			name: "Missing Manifest file",
			setup: func(t *testing.T, pkgDir string) {
				writeValidMetadataXml(t, pkgDir)
				writeValidEbuild(t, pkgDir)
			},
			expectError: "[Error] Missing or unparsable Manifest file",
		},
		{
			name: "Distfile missing from Manifest",
			setup: func(t *testing.T, pkgDir string) {
				writeValidMetadataXml(t, pkgDir)
				require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "Manifest"),
					[]byte("DIST other.tar.gz 100 BLAKE2B 123 SHA512 456\n"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "testpkg-1.0.0.ebuild"), []byte(`EAPI=8
DESCRIPTION="Valid description with enough characters"
HOMEPAGE="https://example.com"
SRC_URI="https://example.com/missing.tar.gz"
LICENSE="MIT"
SLOT="0"
KEYWORDS="~amd64"
`), 0644))
			},
			expectError: "distfile missing from Manifest: [ missing.tar.gz ]",
		},
		{
			name: "Invalid or missing metadata.xml",
			setup: func(t *testing.T, pkgDir string) {
				require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "Manifest"), []byte(""), 0644))
				writeValidEbuild(t, pkgDir)
			},
			expectError: "[Error] Invalid or missing metadata.xml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			overlayDir, cleanup := setupHermeticG2Overlay(t)
			defer cleanup()

			pkgDir := filepath.Join(overlayDir, "app-text", "testpkg")
			require.NoError(t, os.MkdirAll(pkgDir, 0755))
			tc.setup(t, pkgDir)

			// Generate cache
			_ = exec.Command("g2", "cache", "generate", "-repo", overlayDir, "app-text/testpkg").Run()

			// Run lint with the updater policy: -disable-rule PG0802 -fail-severity error
			lintCmd := exec.Command("g2", "lint", "-disable-rule", "PG0802", "-fail-severity", "error", overlayDir, "app-text/testpkg")
			out, err := lintCmd.CombinedOutput()
			require.Error(t, err, "lint must fail for %s: %s", tc.name, string(out))
			require.Contains(t, string(out), tc.expectError)
		})
	}
}

func writeValidMetadataXml(t *testing.T, pkgDir string) {
	t.Helper()
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE pkgmetadata SYSTEM "https://www.gentoo.org/dtd/metadata.dtd">
<pkgmetadata>
	<maintainer type="person">
		<email>test@example.com</email>
	</maintainer>
</pkgmetadata>
`
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "metadata.xml"), []byte(xmlContent), 0644))
}

func writeValidEbuild(t *testing.T, pkgDir string) {
	t.Helper()
	ebuildContent := `# Copyright 2026 Gentoo Authors
# Distributed under the terms of the GNU General Public License v2
EAPI=8
DESCRIPTION="Valid test package description with length"
HOMEPAGE="https://example.com"
LICENSE="MIT"
SLOT="0"
KEYWORDS="~amd64"
`
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "testpkg-1.0.0.ebuild"), []byte(ebuildContent), 0644))
}

// TestIssue194_AppImage_UpdaterLintGating verifies that Github AppImage release
// updater workflows correctly generate md5-cache before linting and pass real g2 lint.
func TestIssue194_AppImage_UpdaterLintGating(t *testing.T) {
	verifyG2Available(t)
	overlayDir, cleanup := setupHermeticG2Overlay(t)
	defer cleanup()

	require.NoError(t, os.MkdirAll(filepath.Join(overlayDir, "metadata", "md5-cache", "app-misc"), 0755))

	appImageData := []byte("#!/bin/sh\necho 'AppImage mock'\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(appImageData)
	}))
	defer server.Close()

	configString := `Type Github AppImage Release
GithubProjectUrl https://github.com/myorg/myapp
EbuildName myapp-appimage
Category app-misc
Description Desktop AppImage Application
Homepage https://myorg.github.io/myapp
License AGPL-3
Binary amd64=>myapp-${TAG}-x86_64.AppImage > myapp.AppImage
DesktopFile myapp.desktop
`

	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	testDate := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	err = configs[0].GenerateGithubWorkflow("test.config", testDate, tmpls, overlayDir, "1.0.0")
	require.NoError(t, err)

	yamlPath := filepath.Join(overlayDir, "app-misc-myapp-appimage-update.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(yamlData, &workflow))

	steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
	stepMap := make(map[string]map[string]any)
	for _, s := range steps {
		step := s.(map[string]any)
		if name, ok := step["name"].(string); ok {
			stepMap[name] = step
		}
	}

	require.Contains(t, stepMap, "Generate md5-cache")
	require.Contains(t, stepMap, "Lint output")
	require.Contains(t, stepMap, "Commit and push changes")

	actionStr := stepMap["Lint output"]["with"].(map[string]any)["action"].(string)
	require.Contains(t, actionStr, "-fail-severity error")

	binDir := filepath.Join(overlayDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	ghMock := filepath.Join(binDir, "gh")
	require.NoError(t, os.WriteFile(ghMock, []byte("#!/bin/sh\necho 'v1.0.0'\n"), 0755))

	githubOutput := filepath.Join(overlayDir, "github_output")
	_ = os.WriteFile(githubOutput, []byte(""), 0644)
	githubEnv := filepath.Join(overlayDir, "github_env")
	_ = os.WriteFile(githubEnv, []byte(""), 0644)

	processStep := stepMap["Process each release"]
	processScript := processStep["run"].(string)
	resolvedProcess := resolveWorkflowEnv(processScript, workflow)
	resolvedProcess = strings.ReplaceAll(resolvedProcess, "https://github.com/myorg/myapp/releases/download/${tag}/", server.URL+"/")

	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=myapp-appimage\nPV=1.0.0\nset -euo pipefail\n"+resolvedProcess)
	cmd.Dir = overlayDir
	cmd.Env = append(os.Environ(),
		"RUNNER_TEMP="+overlayDir,
		"GITHUB_OUTPUT="+githubOutput,
		"GITHUB_ENV="+githubEnv,
	)
	if envMap, ok := processStep["env"].(map[string]any); ok {
		for k, v := range envMap {
			if s, ok := v.(string); ok {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, s))
			}
		}
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Process each release failed: %s", string(out))

	// Execute Generate md5-cache
	cacheStep := stepMap["Generate md5-cache"]
	cacheScript := resolveWorkflowEnv(cacheStep["run"].(string), workflow)
	cacheCmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nset -euo pipefail\n"+cacheScript)
	cacheCmd.Dir = overlayDir
	cacheOut, err := cacheCmd.CombinedOutput()
	require.NoError(t, err, "Generate md5-cache step failed: %s", string(cacheOut))

	md5CachePath := filepath.Join(overlayDir, "metadata", "md5-cache", "app-misc", "myapp-appimage-1.0.0")
	_, err = os.Stat(md5CachePath)
	require.NoError(t, err, "md5-cache file must exist prior to lint")

	// Execute Lint output with real g2
	resolvedAction := resolveWorkflowEnv(actionStr, workflow)
	lintFields := strings.Fields(resolvedAction)
	newLintCmd := exec.Command("g2", lintFields...)
	newLintCmd.Dir = overlayDir
	newLintOut, err := newLintCmd.CombinedOutput()
	require.NoError(t, err, "AppImage lint output must pass: %s", string(newLintOut))
}
