package arrans_overlay_workflow_builder

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGeneratedBashSyntax(t *testing.T) {
	tests := []struct {
		name         string
		configString string
		templateName string
	}{
		{
			name: "Github Binary Release",
			configString: `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
`,
			templateName: "github-binary.tmpl",
		},
		{
			name: "Github AppImage Release",
			configString: `Type Github AppImage Release
GithubProjectUrl https://github.com/test/test
EbuildName test-appimage
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
`,
			templateName: "github-appimage.tmpl",
		},
		{
			name: "Web Binary",
			configString: `Type Web Binary
DownloadBaseUrl https://example.com/download/
CustomVersionSource echo 1.0.0
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
`,
			templateName: "web-binary.tmpl",
		},
		{
			name: "Web AppImage",
			configString: `Type Web AppImage
DownloadPageUrl https://example.com/
DownloadMatch .*
EbuildName test-appimage
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
`,
			templateName: "web-appimage.tmpl",
		},
		{
			name: "Github Cmake Release",
			configString: `Type Github Cmake Release
GithubProjectUrl https://github.com/test/test
EbuildName test
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
`,
			templateName: "github-cmake.tmpl",
		},
	}

	templates, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputConfigs, err := ParseInputConfigReader(strings.NewReader(tt.configString))
			require.NoError(t, err)
			require.Len(t, inputConfigs, 1)

			ic := inputConfigs[0]
			ic.Programs = map[string]*Program{}

			base := &GenerateGithubWorkflowBase{
				Version:     "1.0",
				InputConfig: ic,
				ConfigFile:  "test.config",
				Now:         time.Now().UTC(),
			}

			var data interface{}
			switch tt.name {
			case "Github Binary Release":
				data = &GenerateGithubBinaryTemplateData{GenerateGithubWorkflowBase: base}
			case "Github AppImage Release":
				data = &GenerateGithubAppImageTemplateData{GenerateGithubWorkflowBase: base}
			case "Web Binary":
				data = &GenerateWebBinaryTemplateData{GenerateGithubBinaryTemplateData: &GenerateGithubBinaryTemplateData{GenerateGithubWorkflowBase: base}}
			case "Web AppImage":
				data = &GenerateWebAppImageTemplateData{GenerateGithubAppImageTemplateData: &GenerateGithubAppImageTemplateData{GenerateGithubWorkflowBase: base}}
			case "Github Cmake Release":
				data = &GenerateGithubCmakeTemplateData{GenerateGithubWorkflowBase: base}
			}

			var out bytes.Buffer
			err = templates.ExecuteTemplate(&out, tt.templateName, data)
			require.NoError(t, err)

			var workflow map[string]interface{}
			err = yaml.Unmarshal(out.Bytes(), &workflow)
			require.NoError(t, err)

			jobs, ok := workflow["jobs"].(map[string]interface{})
			require.True(t, ok)

			for jobName, jobInterface := range jobs {
				job, ok := jobInterface.(map[string]interface{})
				if !ok {
					continue
				}
				steps, ok := job["steps"].([]interface{})
				if !ok {
					continue
				}
				for _, stepInterface := range steps {
					step, ok := stepInterface.(map[string]interface{})
					if !ok {
						continue
					}
					runScript, ok := step["run"].(string)
					if ok && runScript != "" {
						cmd := exec.Command("bash", "-n")
						cmd.Stdin = strings.NewReader(runScript)
						out, err := cmd.CombinedOutput()
						require.NoError(t, err, "Syntax error in bash script for job '%s' step '%v':\n%s\nScript:\n%s", jobName, step["name"], string(out), runScript)
					}
				}
			}
		})
	}
}
