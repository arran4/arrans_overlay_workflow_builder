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

// Helper function to render a given template
func renderTestTemplate(t *testing.T, templateType string, useExplicitRepoName bool, repoName string, upsert bool) string {
	ic := &InputConfig{
		Type:             templateType,
		GithubProjectUrl: "https://github.com/owner/repo/",
		EbuildName:       "test-app",
		Category:         "app-misc",
		Description:      "Test app",
		Homepage:         "https://example.com",
		Features:         map[string]string{"Generate Overlay": ""},
	}

	if upsert {
		ic.Features["Upsert Overlay Repo Name"] = ""
	}

	switch templateType {
	case "Web AppImage", "Web Binary":
		ic.DownloadPageUrl = "https://example.com/download"
		ic.DownloadMatch = ".*"
		ic.DownloadBaseUrl = "https://example.com/download/${VERSION}"
		ic.VersionPipeline = "get(https://example.com) | regex(v.*)"
	}

	err := ic.Validate()
	assert.NoError(t, err)

	now := time.Now()
	base := &GenerateGithubWorkflowBase{
		Version:               "1.0.0",
		Now:                   now,
		ConfigFile:            "test.config",
		InputConfig:           ic,
		GlobalGenerateOverlay: true,
	}

	if useExplicitRepoName {
		base.GlobalOverlayRepoName = &repoName
	}

	var data interface{}

	switch ic.Type {
	case "Github AppImage Release":
		data = &GenerateGithubAppImageTemplateData{
			GenerateGithubWorkflowBase: base,
		}
	case "Web AppImage":
		data = &GenerateWebAppImageTemplateData{
			GenerateGithubAppImageTemplateData: &GenerateGithubAppImageTemplateData{
				GenerateGithubWorkflowBase: base,
			},
		}
	case "Github Binary Release":
		data = &GenerateGithubBinaryTemplateData{
			GenerateGithubWorkflowBase: base,
		}
	case "Github Cmake Release":
		data = &GenerateGithubCmakeTemplateData{
			GenerateGithubWorkflowBase: base,
		}
	case "Web Binary":
		data = &GenerateWebBinaryTemplateData{
			GenerateGithubBinaryTemplateData: &GenerateGithubBinaryTemplateData{
				GenerateGithubWorkflowBase: base,
			},
		}
	default:
		t.Fatalf("Unknown template type: %s", ic.Type)
	}

	templates, err := ParseWorkflowTemplates()
	assert.NoError(t, err)

	tmpl := templates.Lookup(data.(interface{ TemplateFileName() string }).TemplateFileName())
	assert.NotNil(t, tmpl, "Template not found")

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	assert.NoError(t, err)

	return buf.String()
}

func TestSemanticGenerateOverlayRender(t *testing.T) {
	templateFamilies := []string{
		"Github AppImage Release",
		"Github Binary Release",
		"Github Cmake Release",
		"Web AppImage",
		"Web Binary",
	}

	for _, family := range templateFamilies {
		t.Run(family, func(t *testing.T) {
			// 1. Explicit name + explicit upsert replaces fallback entirely
			outExplicit := renderTestTemplate(t, family, true, "arrans-overlay", true)
			assert.Contains(t, outExplicit, "echo \"arrans-overlay\" > profiles/repo_name")
			assert.NotContains(t, outExplicit, "repo_name=\"${GITHUB_REPOSITORY#*/}\"")

			// Guard condition must be present
			assert.Contains(t, outExplicit, "if [ -f profiles/repo_name ]; then")

			// Check provenance
			if family != "Github Cmake Release" {
				assert.Contains(t, outExplicit, "echo '# Generated via: ${{ github.server_url }}/${{ github.repository }}/blob/${{ github.sha }}/.github/workflows/${{ env.workflow_filename }}'")
			}
			assert.NotContains(t, outExplicit, "https://github.com/arran4/arrans_overlay/blob/main")

			// 2. Missing explicit name + explicit upsert defaults to GITHUB_REPOSITORY fallback and normalization
			outFallback := renderTestTemplate(t, family, false, "", true)
			assert.Contains(t, outFallback, "repo_name=\"${GITHUB_REPOSITORY#*/}\"")
			assert.Contains(t, outFallback, "repo_name=\"${repo_name//./_}\"")
			assert.Contains(t, outFallback, "repo_name=\"${repo_name/#-/_}\"")
			assert.Contains(t, outFallback, "echo \"${repo_name}\" > profiles/repo_name")

			// Guard condition must be present
			assert.Contains(t, outFallback, "if [ -f profiles/repo_name ]; then")

			// 3. Default + existing file => untouched
			outUntouched := renderTestTemplate(t, family, false, "", false)
			assert.Contains(t, outUntouched, "if [ -f profiles/repo_name ]; then")
			assert.Contains(t, outUntouched, "echo \"::warning title=Missing profiles/repo_name::profiles/repo_name is missing. Skipping creation because explicit upsert is not enabled.\"")

			// 4. Configured canonical identity without upsert => no mutation
			outNoUpsertConfigured := renderTestTemplate(t, family, true, "arrans-overlay", false)
			assert.Contains(t, outNoUpsertConfigured, "if [ -f profiles/repo_name ]; then")
			assert.Contains(t, outNoUpsertConfigured, "if [ \"$(cat profiles/repo_name)\" != \"arrans-overlay\" ]; then")
			assert.Contains(t, outNoUpsertConfigured, "echo \"::warning title=profiles/repo_name mismatch::Configured canonical overlay identity 'arrans-overlay' does not match existing profiles/repo_name '$(cat profiles/repo_name)' - leaving existing file untouched.\"")
			assert.Contains(t, outNoUpsertConfigured, "echo \"::warning title=Missing profiles/repo_name::profiles/repo_name is missing. Skipping creation because explicit upsert is not enabled.\"")

			// 5. Configured canonical identity differing from existing file + explicit upsert => warning and no overwrite
			outMismatchWithUpsert := renderTestTemplate(t, family, true, "arrans-overlay", true)
			assert.Contains(t, outMismatchWithUpsert, "if [ \"$(cat profiles/repo_name)\" != \"arrans-overlay\" ]; then")
			assert.Contains(t, outMismatchWithUpsert, "echo \"::warning title=profiles/repo_name mismatch::Configured canonical overlay identity 'arrans-overlay' does not match existing profiles/repo_name '$(cat profiles/repo_name)' - leaving existing file untouched.\"")

			// 6. Existing file untouched + fallback normalization + explicit upsert (simulating execution path conceptually via template)
			outFallbackWithUpsert := renderTestTemplate(t, family, false, "", true)
			assert.Contains(t, outFallbackWithUpsert, "if [ -f profiles/repo_name ]; then")
			assert.Contains(t, outFallbackWithUpsert, "repo_name=\"${GITHUB_REPOSITORY#*/}\"")
			assert.Contains(t, outFallbackWithUpsert, "repo_name=\"${repo_name//./_}\"")
			assert.Contains(t, outFallbackWithUpsert, "repo_name=\"${repo_name/#-/_}\"")
			assert.Contains(t, outFallbackWithUpsert, "echo \"${repo_name}\" > profiles/repo_name")
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
			name:      "Invalid empty explicit name",
			repoName:  "",
			wantError: true,
			errorMsg:  "cannot be empty",
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
