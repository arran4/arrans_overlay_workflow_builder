package arrans_overlay_workflow_builder

import (
	"bytes"
	"strings"
	"testing"
)

func TestBinaryWorkflowNoOpSrcUnpack(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/example/foo
EbuildName foo-bin
Category app-misc
Description test
License MIT
ProgramName foo
Binary amd64=>foo-${VERSION} > foo`
	data := NewGenerateGithubBinaryTemplateDataFromString(cfg)
	templates, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("ParseWorkflowTemplates: %v", err)
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, data.TemplateFileName(), data); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "src_unpack() { :; }") {
		t.Fatalf("expected no-op src_unpack, got:\n%s", out)
	}
}

func TestBinaryWorkflowHasSrcUnpackCommands(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/example/foo
EbuildName foo-bin
Category app-misc
Description test
License MIT
ProgramName foo
Binary amd64=>foo-${VERSION}.tar.gz > foo > foo`
	data := NewGenerateGithubBinaryTemplateDataFromString(cfg)
	templates, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("ParseWorkflowTemplates: %v", err)
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, data.TemplateFileName(), data); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "src_unpack() { :; }") {
		t.Fatalf("unexpected no-op src_unpack: %s", out)
	}
	if !strings.Contains(out, "unpack \\\"\\${DISTDIR}/") {
		t.Fatalf("expected unpack command in src_unpack, got:\n%s", out)
	}
}
