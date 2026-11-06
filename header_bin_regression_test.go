package arrans_overlay_workflow_builder

import (
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

// TestIssue187_HeaderBin_Regression exercises the exact downstream header-bin v0.0.3
// configuration and reproduces the validation pipeline against real g2 (v0.0.107).
//
// Reported defects investigated and verified:
//  1. Multiple KEYWORDS assignments:
//     Previously, KEYWORDS="" was followed by KEYWORDS+="..." which violated Gentoo QA
//     rules PG0101 and PG0106 ("Ebuild appends to KEYWORDS. It must be defined at most once").
//     Fixed: emitted as a single direct KEYWORDS="..." assignment.
//  2. Truncated SRC_URI values and DIST filename mismatch with Manifest:
//     Previously, metadata.tmpl's character chunking sliced URLs at column boundaries,
//     breaking URLs into fragments such as "header_0.0." and "3_linux_amd64.tar.gz".
//     This caused g2 static parsing to deduce distfile names like "header_0.0.",
//     failing with "distfile missing from Manifest: [ header_0.0. ]".
//     Fixed: URLs and rename targets are kept whole on their own lines.
//  3. Source header policy:
//     The ebuild generates "# SPDX-License-Identifier: MIT". g2 emits an informational
//     Notice ("[Notice] Ebuild 0.0.3 has an invalid header. It should start with a valid copyright line...").
//     This Notice has severity "Notice", does not cause g2 lint to fail (exit code 0),
//     and preserving the project's configurable copyright/source-licensing policy
//     is strictly maintained without misrepresenting upstream MIT licensing as GPLv2.
func TestIssue187_HeaderBin_Regression(t *testing.T) {
	verifyG2Available(t)
	overlayDir, cleanup := setupHermeticG2Overlay(t)
	defer cleanup()

	// Fixture server serving realistic header v0.0.3 tarballs for amd64, arm64, and x86 (386)
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

	// Exact production configuration from arrans_overlay PR #1002
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
	err = configs[0].GenerateGithubWorkflow("current.config", testDate, tmpls, overlayDir, "1.0.0")
	require.NoError(t, err)

	yamlPath := filepath.Join(overlayDir, "app-text-header-bin-update.yaml")
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

	// Mock gh to return release tag v0.0.3
	binDir := filepath.Join(overlayDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	ghMock := filepath.Join(binDir, "gh")
	require.NoError(t, os.WriteFile(ghMock, []byte("#!/bin/sh\necho 'v0.0.3'\n"), 0755))

	githubOutput := filepath.Join(overlayDir, "github_output")
	_ = os.WriteFile(githubOutput, []byte(""), 0644)
	githubEnv := filepath.Join(overlayDir, "github_env")
	_ = os.WriteFile(githubEnv, []byte(""), 0644)

	resolvedScript := resolveWorkflowEnv(processScript, workflow)
	// Redirect GitHub download URLs in g2 manifest upsert to the test server
	resolvedScript = strings.ReplaceAll(resolvedScript, "https://github.com/arran4/header/releases/download/${tag}/", server.URL+"/")

	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=header-bin\nPV=0.0.3\nset -euo pipefail\n"+resolvedScript)
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
	require.NoError(t, err, "Process each release step failed: %s", string(out))

	// Verify emitted ebuild
	pkgDir := filepath.Join(overlayDir, "app-text", "header-bin")
	ebuildPath := filepath.Join(pkgDir, "header-bin-0.0.3.ebuild")
	ebuildBytes, err := os.ReadFile(ebuildPath)
	require.NoError(t, err, "generated ebuild header-bin-0.0.3.ebuild must exist")
	ebuildContent := string(ebuildBytes)

	// Regression Assertion 1: KEYWORDS defined exactly once as a single line assignment
	require.Contains(t, ebuildContent, `KEYWORDS="~amd64 ~arm64 ~x86"`, "KEYWORDS must be defined in full as single assignment")
	require.NotContains(t, ebuildContent, `KEYWORDS=""`, "KEYWORDS must not be initialized empty")
	require.NotContains(t, ebuildContent, `KEYWORDS+=`, "KEYWORDS must not be appended to (PG0101/PG0106)")

	// Regression Assertion 2: Preserves configured SPDX header
	require.True(t, strings.HasPrefix(ebuildContent, "# SPDX-License-Identifier: MIT\n"), "must preserve SPDX header policy")

	// Regression Assertion 3: SRC_URI must not truncate URLs
	require.NotContains(t, ebuildContent, "header_0.0.\"", "SRC_URI must not truncate URLs at 80-col limit")
	require.Contains(t, ebuildContent, "header_0.0.3_linux_amd64.tar.gz", "SRC_URI must contain complete amd64 asset name")
	require.Contains(t, ebuildContent, "header_0.0.3_linux_arm64.tar.gz", "SRC_URI must contain complete arm64 asset name")
	require.Contains(t, ebuildContent, "header_0.0.3_linux_386.tar.gz", "SRC_URI must contain complete x86 asset name")

	// Regression Assertion 4: Manifest distfiles match SRC_URI targets exactly
	manifestPath := filepath.Join(pkgDir, "Manifest")
	manifestBytes, err := os.ReadFile(manifestPath)
	require.NoError(t, err, "Manifest must exist")
	manifestContent := string(manifestBytes)

	expectedAmd64Dist := "DIST header-bin-0.0.3-header_0.0.3_linux_amd64.tar.gz"
	expectedArm64Dist := "DIST header-bin-0.0.3-header_0.0.3_linux_arm64.tar.gz"
	expected386Dist := "DIST header-bin-0.0.3-header_0.0.3_linux_386.tar.gz"

	require.Contains(t, manifestContent, expectedAmd64Dist)
	require.Contains(t, manifestContent, expectedArm64Dist)
	require.Contains(t, manifestContent, expected386Dist)

	// Regression Assertion 5: Real g2 cache generation succeeds and verifies md5-cache
	cacheCmd := exec.Command("g2", "cache", "generate", "-repo", overlayDir, "app-text/header-bin")
	cacheOut, err := cacheCmd.CombinedOutput()
	require.NoError(t, err, "real g2 cache generate must succeed: %s", string(cacheOut))

	md5CachePath := filepath.Join(overlayDir, "metadata", "md5-cache", "app-text", "header-bin-0.0.3")
	cacheBytes, err := os.ReadFile(md5CachePath)
	require.NoError(t, err, "md5-cache entry must exist")
	cacheContent := string(cacheBytes)

	require.Contains(t, cacheContent, "EAPI=8")
	require.Contains(t, cacheContent, "KEYWORDS=~amd64 ~arm64 ~x86")
	require.Contains(t, cacheContent, "LICENSE=MIT")
	require.Contains(t, cacheContent, "SLOT=0")
	require.Contains(t, cacheContent, "_md5_=")

	// Verify SRC_URI in md5-cache record is a single normalized line
	for _, line := range strings.Split(cacheContent, "\n") {
		if strings.HasPrefix(line, "SRC_URI=") {
			require.NotContains(t, line, "\r")
			require.Contains(t, line, "amd64?")
			require.Contains(t, line, "arm64?")
			require.Contains(t, line, "x86?")
			require.Contains(t, line, "-> header-bin-0.0.3-header_0.0.3_linux_amd64.tar.gz")
			require.Contains(t, line, "-> header-bin-0.0.3-header_0.0.3_linux_arm64.tar.gz")
			require.Contains(t, line, "-> header-bin-0.0.3-header_0.0.3_linux_386.tar.gz")
		}
	}

	verifyGeneratedG2Metadata(t, overlayDir, ebuildPath)

	// Regression Assertion 6: Real g2 lint succeeds with zero errors under representative overlay policy
	lintCmd := exec.Command("g2", "lint", "-disable-rule", "PG0802", "-fail-severity", "error", overlayDir, "app-text/header-bin")
	lintOut, err := lintCmd.CombinedOutput()
	lintStr := string(lintOut)
	require.NoError(t, err, "real g2 lint must pass cleanly without errors: %s", lintStr)
	require.NotContains(t, lintStr, "[Error]")
	require.NotContains(t, lintStr, "Ebuild 0.0.3 appends to KEYWORDS")
	require.NotContains(t, lintStr, "KEYWORDS is defined multiple times")
	require.NotContains(t, lintStr, "distfile missing from Manifest")
	require.NotContains(t, lintStr, "Manifest entry for unused DIST file")
}
