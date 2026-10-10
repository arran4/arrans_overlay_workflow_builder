package arrans_overlay_workflow_builder

import (
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIssue172MetadataWidth(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	configString := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description An extremely long description that will definitely exceed the eighty character limit that we have set for ebuilds to ensure they wrap correctly
Homepage https://www.test.io/
License MIT License
Binary amd64=>test-${TAG}-linux-amd64-very-long-asset-name > test > test
Binary arm64=>test-${TAG}-linux-arm64-very-long-asset-name > test > test
`
	inputConfigs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	ic := inputConfigs[0]

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)
	err = ic.GenerateGithubWorkflow("test.config", time.Now(), tmpls, tempDir, "1.0.0")
	require.NoError(t, err)

	yamlPath := filepath.Join(tempDir, "app-admin-test-bin-update.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	var workflow map[string]any
	err = yaml.Unmarshal(yamlData, &workflow)
	require.NoError(t, err)

	jobs := workflow["jobs"].(map[string]any)
	job := jobs["check-and-create-ebuild"].(map[string]any)
	steps := job["steps"].([]any)

	var script string
	for _, stepItem := range steps {
		step := stepItem.(map[string]any)
		if step["name"] == "Process each release" {
			script = step["run"].(string)
			break
		}
	}
	require.NotEmpty(t, script, "Process each release step not found")

	script = resolveGithubEnvForPackage(script, "app-admin", "test-bin")
	binDir := filepath.Join(tempDir, "bin")
	mockCommand(t, binDir, "g2", "#!/bin/bash\nif [[ \"$1\" == \"ebuild\" && \"$2\" == \"next-revision\" ]]; then echo \"${@: -1}\"; fi\n")
	mockCommand(t, binDir, "gh", "#!/bin/bash\necho v1.0.0\n")
	script = strings.ReplaceAll(script, "emit_metadata_field \"DESCRIPTION\" \"Test\"", "emit_metadata_field \"DESCRIPTION\" \"1234567890123456789012345678901234567890123456789012345678901234567890123\\${P}\"")
	script = strings.ReplaceAll(script, "DESCRIPTION+=\"Test\"", "DESCRIPTION+=\"1234567890123456789012345678901234567890123456789012345678901234567890123${P}\"")

	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nP=test-bin\nset -euo pipefail\n"+script)
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Bash failed: "+string(out))

	ebuildPath := filepath.Join(tempDir, "app-admin", "test-bin", "test-bin-1.0.0.ebuild")
	ebuildContent, err := os.ReadFile(ebuildPath)
	require.NoError(t, err)

	ebuildLines := strings.Split(string(ebuildContent), "\n")

	// Check physical widths
	for i, line := range ebuildLines {
		pos := 0
		for _, ch := range line {
			if ch == '	' {
				pos += 4
			} else {
				pos += 1
			}
		}
		if pos > 80 {
			t.Errorf("Ebuild line %d exceeds 80 characters (width %d): %s", i+1, pos, line)
		}
	}

	evalMetadata := func(varName string) string {
		var varLines []string
		for _, line := range ebuildLines {
			if strings.HasPrefix(strings.TrimSpace(line), varName+"=") || strings.HasPrefix(strings.TrimSpace(line), varName+"+=") {
				varLines = append(varLines, line)
			}
		}

		script := "P=test-bin\nPV=1.0.0\n" + strings.Join(varLines, "\n")
		script += "\nprintf '%s' \"$" + varName + "\""

		evalCmd := exec.Command("bash", "-c", script)
		output, evalErr := evalCmd.Output()
		require.NoError(t, evalErr, "Evaluating "+varName+" failed")

		evaluatedValue := string(output)
		require.NotContains(t, evaluatedValue, "\n")
		require.NotContains(t, evaluatedValue, "\t")
		return evaluatedValue
	}

	require.Equal(t, "1234567890123456789012345678901234567890123456789012345678901234567890123test-bin", evalMetadata("DESCRIPTION"), "DESCRIPTION variable expansion split or mangled")
	require.Equal(t, "https://www.test.io/", evalMetadata("HOMEPAGE"))
	require.Equal(t, "MIT License", evalMetadata("LICENSE"))
	require.Equal(t, "amd64", evalMetadata("KEYWORDS"))
	require.Equal(t, "", evalMetadata("IUSE"))
	require.Equal(t, "", evalMetadata("REQUIRED_USE"))
	require.Equal(t, "", evalMetadata("DEPEND"))
	require.Equal(t, "", evalMetadata("RDEPEND"))
	require.Equal(t, "", evalMetadata("BDEPEND"))
	require.Equal(t, " amd64? (   https://github.com/test/test/releases/download/v1.0.0/test-v1.0.0-linux-amd64-very-long-asset-name  -> test-bin-test-v1.0.0-linux-amd64-very-long-asset-name  )   arm64? (   https://github.com/test/test/releases/download/v1.0.0/test-v1.0.0-linux-arm64-very-long-asset-name  -> test-bin-test-v1.0.0-linux-arm64-very-long-asset-name  )  ", evalMetadata("SRC_URI"))
}

func TestIssue172GithubAppImageRDEPENDWidthAndSemantics(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	dependencies := []string{
		"dev-libs/glib",
		"dev-libs/libxml2",
		"dev-libs/openssl",
		"sys-libs/zlib",
		"media-libs/fontconfig",
		"media-libs/freetype",
		"x11-libs/libX11",
		"x11-libs/libXext",
		"x11-libs/libXrender",
		"x11-libs/libxcb",
	}
	configString := `Type Github AppImage Release
GithubProjectUrl https://github.com/test/test
EbuildName test-appimage
Category app-admin
Description Test AppImage
Homepage https://www.test.io/
License MIT
ProgramName testapp
Dependencies ` + strings.Join(dependencies, " ") + `
Binary amd64=>test-${TAG}.AppImage > test.AppImage
`
	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)
	err = configs[0].GenerateGithubWorkflow(
		"test.config",
		time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC),
		tmpls,
		tempDir,
		"1.0.0",
	)
	require.NoError(t, err)

	yamlData, err := os.ReadFile(filepath.Join(tempDir, "app-admin-test-appimage-update.yaml"))
	require.NoError(t, err)

	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(yamlData, &workflow))
	steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
	var script string
	for _, item := range steps {
		step := item.(map[string]any)
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

	cmd := exec.Command("bash", "-c", "set -euo pipefail\n"+script)
	cmd.Dir = tempDir
	_ = cmd.Run()

	ebuildData, err := os.ReadFile(filepath.Join(tempDir, "app-admin", "test-appimage", "test-appimage-1.0.0.ebuild"))
	require.NoError(t, err)
	lines := strings.Split(string(ebuildData), "\n")
	assertIssue172Widths(t, lines)

	got := evalIssue172Field(t, lines, "RDEPEND", "")

	gotParts := strings.Split(got, " ")
	expectedParts := append([]string{"sys-fs/fuse:0"}, dependencies...)
	require.ElementsMatch(t, expectedParts, gotParts)
}

func TestIssue172GithubCmakeDependencyWidthAndSemantics(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	binDir := filepath.Join(tempDir, "bin")
	mockCommand(t, binDir, "curl", `#!/bin/bash
cat <<'CM'
find_package(Qt6 REQUIRED COMPONENTS Core Gui Widgets Test Svg)
find_package(KF6 REQUIRED COMPONENTS Config CoreAddons I18n WidgetsAddons)
CM
`)

	configString := `Type Github Cmake Release
GithubProjectUrl https://github.com/test/test
EbuildName test-cmake
Category app-admin
Description Test CMake package
Homepage https://www.test.io/
License MIT
`
	configs, err := ParseInputConfigReader(strings.NewReader(configString))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)
	err = configs[0].GenerateGithubWorkflow(
		"test.config",
		time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC),
		tmpls,
		tempDir,
		"1.0.0",
	)
	require.NoError(t, err)

	yamlData, err := os.ReadFile(filepath.Join(tempDir, "app-admin-test-cmake-update.yaml"))
	require.NoError(t, err)

	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(yamlData, &workflow))
	steps := workflow["jobs"].(map[string]any)["check-and-create-ebuild"].(map[string]any)["steps"].([]any)
	var script string
	for _, item := range steps {
		step := item.(map[string]any)
		if step["name"] == "Process each release" {
			script = step["run"].(string)
			break
		}
	}
	require.NotEmpty(t, script)

	script = resolveGithubEnvForPackage(script, "app-admin", "test-cmake")
	script = strings.ReplaceAll(script, "${{ env.workflow_filename }}", "test.yml")
	cmd := exec.Command("bash", "-c", "PATH="+binDir+":$PATH\nset -euo pipefail\n"+script)
	cmd.Dir = tempDir
	_ = cmd.Run()

	ebuildData, err := os.ReadFile(filepath.Join(tempDir, "app-admin", "test-cmake", "test-cmake-1.0.0.ebuild"))
	require.NoError(t, err)
	lines := strings.Split(string(ebuildData), "\n")
	assertIssue172Widths(t, lines)

	var assignments []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "DEPEND=") || strings.HasPrefix(trimmed, "DEPEND+=") ||
			strings.HasPrefix(trimmed, "RDEPEND=") || strings.HasPrefix(trimmed, "RDEPEND+=") ||
			strings.HasPrefix(trimmed, "BDEPEND=") || strings.HasPrefix(trimmed, "BDEPEND+=") {
			assignments = append(assignments, line)
		}
	}

	evalScript := strings.Join(assignments, "\n") + `
printf '%s\0%s\0%s' "$DEPEND" "$RDEPEND" "$BDEPEND"
`
	evalCmd := exec.Command("bash", "-c", evalScript)
	evaluated, err := evalCmd.Output()
	require.NoError(t, err)
	parts := strings.Split(string(evaluated), "\x00")
	require.Len(t, parts, 3)

	wantDepend := " dev-qt/qtbase:6[,core,gui,widgets] dev-qt/qtsvg:6" +
		" kde-frameworks/kconfig:6 kde-frameworks/kcoreaddons:6" +
		" kde-frameworks/ki18n:6 kde-frameworks/kwidgetsaddons:6"
	wantBDepend := "virtual/pkgconfig kde-frameworks/extra-cmake-modules:0 dev-qt/qttools:6[linguist]"

	require.Equal(t, wantDepend, parts[0])
	require.Equal(t, wantDepend, parts[1], "RDEPEND must evaluate to DEPEND")
	require.Equal(t, wantBDepend, parts[2])
}

func assertIssue172Widths(t *testing.T, lines []string) {
	t.Helper()
	for i, line := range lines {
		width := 0
		for _, r := range line {
			if r == '\t' {
				width += 4
			} else {
				width++
			}
		}
		require.LessOrEqualf(t, width, 80, "ebuild line %d exceeds 80 positions: %q", i+1, line)
	}
}

func evalIssue172Field(t *testing.T, lines []string, name, preamble string) string {
	t.Helper()
	var assignments []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, name+"=") || strings.HasPrefix(trimmed, name+"+=") {
			assignments = append(assignments, line)
		}
	}
	require.NotEmptyf(t, assignments, "missing %s assignments", name)

	script := preamble + "\n" + strings.Join(assignments, "\n") + "\nprintf '%s' \"$" + name + "\""
	cmd := exec.Command("bash", "-c", script)
	output, err := cmd.Output()
	require.NoError(t, err)
	value := string(output)
	require.NotContains(t, value, "\n")
	require.NotContains(t, value, "\t")
	require.NotContains(t, value, `\n`)
	require.NotContains(t, value, `\t`)
	return value
}
