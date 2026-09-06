package arrans_overlay_workflow_builder

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os/exec"

	"bytes"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestShellEchoContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "empty",
			content: "",
			want:    "",
		},
		{
			name:    "single line",
			content: "hello world",
			want:    "echo 'hello world'\n",
		},
		{
			name:    "multiline",
			content: "hello\nworld",
			want:    "echo 'hello'\necho 'world'\n",
		},
		{
			name:    "single quotes escaped",
			content: "echo 'hello'",
			want:    "echo 'echo '\\''hello'\\'''\n",
		},
		{
			name:    "variables not escaped",
			content: "echo $VAR",
			want:    "echo 'echo $VAR'\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShellEchoContent(tt.content))
		})
	}
}

func TestShellEchoEvalContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "variables",
			content: "echo \"${VAR}\"",
			want:    "echo \"echo \\\"${VAR}\\\"\"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShellEchoEvalContent(tt.content))
		})
	}
}

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

func assertNoDuplicateShellcheckDirectives(t *testing.T, workflow string) {
	t.Helper()

	previousWasSC2001 := false
	for _, line := range strings.Split(workflow, "\n") {
		isSC2001 := strings.TrimSpace(line) == "# shellcheck disable=SC2001"

		if isSC2001 && previousWasSC2001 {
			t.Errorf("duplicate adjacent SC2001 directive")
		}

		previousWasSC2001 = isSC2001
	}
}

func TestBashOverlayNameNormalization(t *testing.T) {
	cases := []struct {
		githubRepo string
		expected   string
	}{
		{"owner/normal_overlay", "normal_overlay"},
		{"owner/my.overlay", "my_overlay"},
		{"owner/-invalid.start", "_invalid_start"},
		{"owner/valid-repo-name", "valid-repo-name"},
	}

	for _, c := range cases {
		t.Run(c.githubRepo, func(t *testing.T) {
			script := fmt.Sprintf(`export GITHUB_REPOSITORY="%s"; repo_name="${GITHUB_REPOSITORY#*/}"; repo_name="${repo_name//./_}"; repo_name="${repo_name/#-/_}"; echo "${repo_name}"`, c.githubRepo)
			cmd := exec.Command("bash", "-c", script)
			output, err := cmd.Output()
			require.NoError(t, err)
			require.Equal(t, c.expected+"\n", string(output))
		})
	}
}

func TestOverlayRepoNameValidation(t *testing.T) {
	inputConfig := &InputConfig{
		Type:       "Github Binary Release",
		Category:   "app-misc",
		EbuildName: "test-app",
	}

	tests := []struct {
		name      string
		repoName  string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid normal name",
			repoName:  "arrans-overlay",
			wantError: false,
		},
		{
			name:      "Valid name with underscore",
			repoName:  "my_repo",
			wantError: false,
		},
		{
			name:      "Valid name with numbers",
			repoName:  "repo123",
			wantError: false,
		},
		{
			name:      "Invalid starts with hyphen",
			repoName:  "-invalid",
			wantError: true,
			errorMsg:  "invalid characters",
		},
		{
			name:      "Invalid characters",
			repoName:  "my repo!",
			wantError: true,
			errorMsg:  "invalid characters",
		},
		{
			name:      "Invalid ends with version",
			repoName:  "foo-1",
			wantError: true,
			errorMsg:  "cannot end in a valid version string",
		},
		{
			name:      "Invalid ends with version minor",
			repoName:  "foo-1.2.3",
			wantError: true,
			errorMsg:  "invalid characters",
		},
		{
			name:      "Invalid ends with version revision",
			repoName:  "foo-1-r2",
			wantError: true,
			errorMsg:  "cannot end in a valid version string",
		},
		{
			name:      "Invalid ends with version alpha",
			repoName:  "foo-1_alpha1",
			wantError: true,
			errorMsg:  "cannot end in a valid version string",
		},
		{
			name:      "Invalid empty explicit name",
			repoName:  "",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := t.TempDir()
			err := GenerateGithubWorkflowsFromInputConfigs("test.config", []*InputConfig{inputConfig}, outputDir, "1.0.0", OptOverlayRepoName(tt.repoName))
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
