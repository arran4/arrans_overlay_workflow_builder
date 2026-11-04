package arrans_overlay_workflow_builder

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
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

// Pinned released g2 version and verification
const (
	pinnedG2Version = "v0.0.107"
)

func verifyG2Available(t *testing.T) string {
	t.Helper()
	g2Path, err := exec.LookPath("g2")
	if err != nil {
		t.Skipf("g2 executable not found in PATH; skipping real g2 integration tests")
	}
	return g2Path
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		if strings.Contains(r.URL.Path, "amd64") {
			_, _ = w.Write(tarDataAmd64)
		} else {
			_, _ = w.Write(tarDataArm64)
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
Binary arm64=>mytool-${TAG}-linux-arm64.tar.gz > mytool > mytool
Document amd64=>mytool-${TAG}-linux-amd64.tar.gz > README.md > README.md
Document amd64=>mytool-${TAG}-linux-amd64.tar.gz > LICENSE > LICENSE
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
	require.Contains(t, ebuildContent, `KEYWORDS="~amd64 ~arm64"`)
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
	require.Contains(t, cacheContent, "KEYWORDS=~amd64 ~arm64")
	require.Contains(t, cacheContent, "LICENSE=MIT")
	require.Contains(t, cacheContent, "SLOT=0")
	require.Contains(t, cacheContent, "_md5_=")

	// Ensure SRC_URI is on a single physical line in md5-cache
	for _, line := range strings.Split(cacheContent, "\n") {
		if strings.HasPrefix(line, "SRC_URI=") {
			require.NotContains(t, line, "\r")
			require.Contains(t, line, "amd64?")
			require.Contains(t, line, "arm64?")
			require.Contains(t, line, "-> mytool-bin-1.0.0-mytool-v1.0.0-linux-amd64.tar.gz")
			require.Contains(t, line, "-> mytool-bin-1.0.0-mytool-v1.0.0-linux-arm64.tar.gz")
		}
	}

	// 4. Verify Manifest DIST entries match
	manifestPath := filepath.Join(pkgDir, "Manifest")
	manifestBytes, err := os.ReadFile(manifestPath)
	require.NoError(t, err, "Manifest must exist")
	manifestContent := string(manifestBytes)
	require.Contains(t, manifestContent, "DIST mytool-bin-1.0.0-mytool-v1.0.0-linux-amd64.tar.gz")
	require.Contains(t, manifestContent, "DIST mytool-bin-1.0.0-mytool-v1.0.0-linux-arm64.tar.gz")

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

	// Real g2 lint
	lintCmd := exec.Command("g2", "lint", "-disable-rule", "PG0802", "-fail-severity", "error", overlayDir, "app-admin/cmaketool")
	lintOut, err := lintCmd.CombinedOutput()
	lintStr := string(lintOut)
	require.NoError(t, err, "real g2 lint must pass: %s", lintStr)
	require.NotContains(t, lintStr, "[Error]")
}
