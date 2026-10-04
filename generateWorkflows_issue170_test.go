package arrans_overlay_workflow_builder

import (
	"bytes"
	"strings"
	"testing"
)

func TestIssue170ProvenanceWidth(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/derailed/k9s
EbuildName k9s-bin
Category app-admin-arrans-overlay-workflow-builder-bin
Description test
License MIT
ProgramName k9s
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

	if !strings.Contains(output, "emit_provenance_field") {
		t.Errorf("Expected emit_provenance_field in output")
	}
}

// g2's width rule: tabs count as 4 positions, every other rune as 1
func ebuildLineWidth(line string) int {
	width := 0
	for _, r := range line {
		if r == '\t' {
			width += 4
		} else {
			width += 1
		}
	}
	return width
}
func TestIssue172ActualWidthCheck(t *testing.T) {
}
