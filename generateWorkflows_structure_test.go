package arrans_overlay_workflow_builder

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWorkflowYAMLStructure(t *testing.T) {
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
			now := time.Now()

			base := &GenerateGithubWorkflowBase{
				Version:     "1.0",
				InputConfig: ic,
				ConfigFile:  "test.config",
				Now:         now.UTC(),
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

			w := out.Bytes()

			var m map[string]interface{}
			err = yaml.Unmarshal(w, &m)
			require.NoError(t, err, "YAML parsing failed")

			_, hasEnv := m["env"]
			assert.True(t, hasEnv, "env mapping should exist at root")

			_, hasConcurrency := m["concurrency"]
			assert.True(t, hasConcurrency, "concurrency mapping should exist at root")

			_, hasJobs := m["jobs"]
			assert.True(t, hasJobs, "jobs mapping should exist at root")

			envMap, ok := m["env"].(map[string]interface{})
			if ok {
				_, envHasConcurrency := envMap["concurrency"]
				assert.False(t, envHasConcurrency, "concurrency should NOT be nested inside env")
			}
		})
	}
}
