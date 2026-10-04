package arrans_overlay_workflow_builder

import (
	"bytes"
	"strings"
	"testing"
)

func TestIssue172MetadataWidth(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/derailed/k9s
EbuildName k9s-bin
Category app-admin-arrans-overlay-workflow-builder-bin
Description This is a very long description that should exceed the eighty character line limit for ebuild generation and will require wrapping
Homepage https://github.com/derailed/k9s
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

	if !strings.Contains(output, "emit_metadata_field \"DESCRIPTION\" \"${{ env.description }}\"") {
		t.Errorf("Expected emit_metadata_field for DESCRIPTION")
	}
}
