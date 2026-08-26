package arrans_overlay_workflow_builder

import (
	"bytes"
	"strings"
	"testing"
	"time"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
)

func TestSemanticGeneratedEbuildSanity(t *testing.T) {
	inputConfig := `Type Github AppImage Release
GithubProjectUrl https://github.com/rustdesk/rustdesk/
Category net-misc
EbuildName rustdesk-appimage
Description Open-source remote desktop application for self-hosting
Homepage https://rustdesk.com/
License AGPL-3
MaintainerEmail gentoo@arran4.com
MaintainerName Arran Ubels
ProgramName rustdesk
DesktopFile rustdesk.desktop
Icons hicolor-apps
Dependencies sys-libs/glibc sys-libs/zlib
Binary amd64=>rustdesk-${TAG}-x86_64.AppImage > rustdesk.AppImage
Binary arm64=>rustdesk-${TAG}-aarch64.AppImage > rustdesk.AppImage
`

	parsedConfigs, err := ParseInputConfigReader(strings.NewReader(inputConfig))
	if err != nil {
		t.Fatalf("ParseInputConfigReader() error = %v", err)
	}
	ic := parsedConfigs[0]

	templates, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("ParseWorkflowTemplates() error = %v", err)
	}

	base := &GenerateGithubWorkflowBase{
		Version:     "1.0.0",
		Now:         time.Date(2026, time.July, 12, 0, 0, 0, 0, time.UTC),
		ConfigFile:  "test.config",
		Schedule:    "24 2 * * *",
		InputConfig: ic,
	}

	data := &GenerateGithubAppImageTemplateData{
		GenerateGithubWorkflowBase: base,
	}

	out := bytes.NewBuffer(nil)
	if err := templates.ExecuteTemplate(out, "github-appimage.tmpl", data); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}

	workflow := string(normalizeGeneratedWorkflow(out.Bytes()))

	// POSITIVE ASSERTIONS
	if !strings.Contains(workflow, "echo '  if use amd64; then'") {
		t.Errorf("Missing expected valid Gentoo shell: echo '  if use amd64; then'")
	}
	if !strings.Contains(workflow, "echo '  if use arm64; then'") {
		t.Errorf("Missing expected valid Gentoo shell: echo '  if use arm64; then'")
	}

	expectedSed := "echo \"  sed -i 's:^Exec=.*:Exec=/opt/bin/${{ env.rustdesk_appimage_installed_name }}:' 'squashfs-root/${{ env.rustdesk_desktop_file }}'\""
	if !strings.Contains(workflow, expectedSed) {
		t.Errorf("Missing complete desktop-file rewrite. Expected: %s", expectedSed)
	}

	if !strings.Contains(workflow, "if [[ ! -v releaseTypes[${releaseType:=release}] ]]; then") {
		t.Errorf("Missing releaseType missing-entry condition")
	}

	if !strings.Contains(workflow, "releaseTypes[${releaseType:=release}]=\"$version\"") {
		t.Errorf("Missing releaseTypes assignment")
	}

	// NEGATIVE ASSERTIONS
	forbidden := []string{
		"`]]",
		"`]];",
		"echo '  if use amd64`]]",
		"echo '  if use arm64`]]",
		"echo \"  sed -i 's:^Exec=.*:Exec=/opt/bin/.*\"",
		"# shellcheck disable=SC2001\n                # shellcheck disable=SC2001",
		"# shellcheck disable=SC2001\n            # shellcheck disable=SC2001",
	}

	for _, bad := range forbidden {
		if strings.Contains(workflow, bad) {
			t.Errorf("Generated workflow contains corruption %q", bad)
		}
	}

	// ARCHITECTURE SANITY CHECK
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.Contains(line, "echo '  if use ") {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if !strings.HasSuffix(trimmed, "; then'") {
			t.Errorf("Malformed emitted USE conditional: %s", line)
		}
		if strings.Contains(line, "`") || strings.Contains(line, "[[") || strings.Contains(line, "]]") {
			t.Errorf("Template syntax leaked into emitted ebuild conditional: %s", line)
		}
	}
}

func TestLintOutputConditionalIsPresent(t *testing.T) {
	templates, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	ic := &InputConfig{
		Type:        "Github Binary Release",
		Category:    "app-misc",
		EbuildName:  "test-bin",
		Description: "Test application",
		Homepage:    "https://example.com",
		License:     "MIT",
		GithubRepo:  "user/repo",
		Programs: map[string]*Program{
			"test": {
				ProgramName: "test",
				Binary: map[string][]string{
					"amd64": {"test-amd64", "test"},
				},
			},
		},
	}

	fs := util.NewMockFS()
	err = ic.GenerateGithubWorkflow("-", time.Now(), templates, ".github/workflows", "v1.0.0", fs)
	if err != nil {
		t.Fatalf("Failed to generate workflow: %v", err)
	}

	if len(fs.Files) == 0 {
		t.Fatal("No files generated")
	}

	var content string
	for _, v := range fs.Files {
		content = string(v)
		break
	}

	if !strings.Contains(content, "- name: Lint output") {
		t.Fatal("Generated workflow is missing 'Lint output' step")
	}

	// Find the Lint output block
	lines := strings.Split(content, "\n")
	foundLint := false
	hasCondition := false

	for i, line := range lines {
		if strings.Contains(line, "- name: Lint output") {
			foundLint = true
			// Check the next few lines for the condition
			for j := 1; j <= 2 && i+j < len(lines); j++ {
				if strings.Contains(lines[i+j], "if: steps.process_releases.outputs.generated_tag") {
					hasCondition = true
					break
				}
			}
			break
		}
	}

	if !foundLint {
		t.Fatal("Generated workflow is missing 'Lint output' step after generation")
	}
	if !hasCondition {
		t.Fatal("Generated workflow is missing conditional check 'if: steps.process_releases.outputs.generated_tag' on 'Lint output' step")
	}
}
