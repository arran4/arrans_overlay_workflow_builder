package arrans_overlay_workflow_builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateWebAppImageWorkflow(t *testing.T) {
	ic := &InputConfig{
		Type:            "Web AppImage",
		DownloadPageUrl: "https://www.example.com/download",
		Category:        "net-im",
		EbuildName:      "example-appimage.ebuild",
		Description:     "Example",
		Homepage:        "https://www.example.com",
		GithubRepo:      "example",
		Programs: map[string]*Program{
			"example": {
				ProgramName: "example",
				Binary: map[string][]string{
					"~amd64": {"example-${VERSION}.AppImage", "example.AppImage"},
				},
			},
		},
	}
	templates, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("ParseWorkflowTemplates: %v", err)
	}
	outDir := t.TempDir()
	if err := ic.GenerateGithubWorkflow("test.config", time.Now(), templates, outDir, "1"); err != nil {
		t.Fatalf("GenerateGithubWorkflow: %v", err)
	}
	wf := filepath.Join(outDir, "net-im-example-appimage-update.yaml")
	b, err := os.ReadFile(wf)
	if err != nil {
		t.Fatalf("reading workflow: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "htmlq -a href 'a'") {
		t.Fatalf("workflow missing html scrape: %s", content)
	}
	if !strings.Contains(content, "curl -sIL -o /dev/null -w '%{url_effective}'") {
		t.Fatalf("workflow missing redirect resolution: %s", content)
	}
}
