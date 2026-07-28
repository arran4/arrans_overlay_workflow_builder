package arrans_overlay_workflow_builder

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/arran4/arrans_overlay_workflow_builder/util"
)

func TestNormalizeGeneratedWorkflow(t *testing.T) {
	input := []byte("top:  \n  first  \n\t\n\n  second\t\n\nnext:\n  child\n")
	expected := "top:\n  first\n  second\n\nnext:\n  child\n"

	if actual := string(normalizeGeneratedWorkflow(input)); actual != expected {
		t.Fatalf("normalizeGeneratedWorkflow() = %q, want %q", actual, expected)
	}
}

func TestGenerateGithubWorkflowBaseCron(t *testing.T) {
	base := &GenerateGithubWorkflowBase{
		InputConfig: &InputConfig{GithubRepo: "changes-do-not-affect-the-test-schedule"},
		Schedule:    "24 2 * * *",
	}
	if actual := base.Cron(); actual != "24 2 * * *" {
		t.Fatalf("Cron() = %q, want %q", actual, "24 2 * * *")
	}
}

func TestGenerateGithubWorkflows_MockFS(t *testing.T) {
	mockFS := util.NewMockFS()

	inputConfigContent := `Type Github Binary Release
Category net-misc
EbuildName rustdesk
GithubProjectUrl https://github.com/rustdesk/rustdesk
MaintainerEmail gentoo@arran4.com
MaintainerName Arran Ubels
Workaround Semantic Version Prerelease Hack 1
Workaround Semantic Version Without V`

	err := mockFS.WriteFile("test.config", []byte(inputConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write to mockfs: %v", err)
	}

	fixedTime := time.Date(2026, time.July, 12, 0, 0, 0, 0, time.UTC)

	err = GenerateGithubWorkflows("test.config", "output_dir", "1.0.0", mockFS, fixedTime)
	if err != nil {
		t.Fatalf("GenerateGithubWorkflows failed: %v", err)
	}

	found := false
	var actualContent []byte
	for k, v := range mockFS.Files {
		if strings.HasPrefix(k, "output_dir/") {
			found = true
			actualContent = v
			break
		}
	}

	if !found {
		t.Fatalf("Expected output file in output_dir, but found none")
	}

	if !bytes.Contains(actualContent, []byte("echo \"${originalVersion}\"")) && !bytes.Contains(actualContent, []byte("sed 's/")) {
		t.Errorf("Expected workaround generated output, but got: %s", actualContent)
	}
}
