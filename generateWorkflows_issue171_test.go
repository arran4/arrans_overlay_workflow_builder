package arrans_overlay_workflow_builder

import (
	"bytes"
	"strings"
	"testing"
)

func TestIssue171DosymIndentation(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/derailed/k9s
EbuildName k9s-bin
Category app-misc
Description test
License MIT
ProgramName k9s
Symlink k9s=>/usr/bin/k9s
Binary amd64=>k9s_Linux_amd64.tar.gz > k9s > k9s`

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

	if !strings.Contains(output, "echo \"\tdosym /usr/bin/k9s k9s\"") {
		t.Errorf("Expected dosym to be indented with a tab, but got output: %s", output)
	}
}
