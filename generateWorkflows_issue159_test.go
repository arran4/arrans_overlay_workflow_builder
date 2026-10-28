package arrans_overlay_workflow_builder

import (
	"bytes"
	"strings"
	"testing"
)

func TestIssue159K9sFilename(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/derailed/k9s
EbuildName k9s-bin
Category app-misc
Description test
License MIT
ProgramName k9s
Binary amd64=>k9s_Linux_amd64.tar.gz > k9s > k9s`

	data := NewGenerateGithubBinaryTemplateDataFromString(cfg)

	resources := data.ExternalResources()
	if len(resources) == 0 {
		t.Fatalf("Expected resources to be parsed")
	}

	// assert ExternalResources()[0].ReleaseFilename() is k9s_Linux_amd64.tar.gz
	found := false
	for _, res := range resources {
		if res.ReleaseFilename() == "k9s_Linux_amd64.tar.gz" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Expected to find k9s_Linux_amd64.tar.gz in resources, got %v", resources[0].ReleaseFilename())
	}

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

	// Find the SRC_URI line for the URL
	if !strings.Contains(output, "https://github.com/${{ env.github_owner }}/${{ env.github_repo }}/releases/download/${tag}/k9s_Linux_amd64.tar.gz") {
		t.Errorf("Expected URL to contain k9s_Linux_amd64.tar.gz, but it was not found. Output: %s", output)
	}

	if strings.Contains(output, "https://github.com/${{ env.github_owner }}/${{ env.github_repo }}/releases/download/${tag}/\\${PV}") {
		t.Errorf("URL erroneously contains \\${PV}")
	}

	if strings.Contains(output, "https://github.com/${{ env.github_owner }}/${{ env.github_repo }}/releases/download/${tag}/${version}") {
		t.Errorf("URL erroneously contains ${version}")
	}
}

func TestIssue159JoltFilename(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/jolt/jolt
EbuildName jolt-bin
Category app-misc
Description test
License MIT
ProgramName jolt
Binary amd64=>jolt-${VERSION}-linux-amd64.tar.gz > jolt > jolt`

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

	if !strings.Contains(output, "https://github.com/${{ env.github_owner }}/${{ env.github_repo }}/releases/download/${tag}/jolt-${version}-linux-amd64.tar.gz") {
		t.Errorf("Expected URL to contain jolt-${version}-linux-amd64.tar.gz, but it was not found. Output: %s", output)
	}

	// Check DIST rename
	if !strings.Contains(output, "-> \\${P}-jolt-\\${PV}-linux-amd64.tar.gz") {
		t.Errorf("DIST rename erroneously formatted. Expected \\${P}-jolt-\\${PV}-linux-amd64.tar.gz")
	}
}

func TestIssue159JoltFilenameWithPrereleaseHack(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/jolt/jolt
EbuildName jolt-bin
Category app-misc
Description test
License MIT
Workaround Semantic Version Prerelease Hack 1
ProgramName jolt
Binary amd64=>jolt-${VERSION}-linux-amd64.tar.gz > jolt > jolt`

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

	if !strings.Contains(output, "https://github.com/${{ env.github_owner }}/${{ env.github_repo }}/releases/download/${tag}/jolt-${originalVersion}-linux-amd64.tar.gz") {
		t.Errorf("Expected URL to contain jolt-${originalVersion}-linux-amd64.tar.gz, but it was not found. Output: %s", output)
	}

	// Check DIST rename
	if !strings.Contains(output, "-> \\${P}-jolt-\\${PV}-linux-amd64.tar.gz") {
		t.Errorf("DIST rename erroneously formatted. Expected \\${P}-jolt-\\${PV}-linux-amd64.tar.gz")
	}
}
