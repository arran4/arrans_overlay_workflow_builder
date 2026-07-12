package arrans_overlay_workflow_builder

import (
	"bytes"
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/txtar"
)

//go:embed testdata/txtar/*.txtar
var testdataFS embed.FS

func TestWorkflowTemplates(t *testing.T) {
	var cases []string
	err := fs.WalkDir(testdataFS, "testdata/txtar", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".txtar") {
			return nil
		}
		cases = append(cases, p)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk testdata: %v", err)
	}
	sort.Strings(cases)

	templates, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("ParseWorkflowTemplates() error = %v", err)
	}

	for _, tc := range cases {
		tc := tc
		t.Run(strings.TrimSuffix(path.Base(tc), ".txtar"), func(t *testing.T) {
			raw, err := testdataFS.ReadFile(tc)
			if err != nil {
				t.Fatalf("read testcase %s: %v", tc, err)
			}
			ar := txtar.Parse(raw)

			var inputConfigStr string
			var expectedYamlStr string

			for _, f := range ar.Files {
				switch f.Name {
				case "input.config":
					inputConfigStr = string(f.Data)

				case "expected.yaml":
					expectedYamlStr = string(f.Data)
				}
			}

			if inputConfigStr == "" {
				t.Fatalf("Missing input.config in %s", tc)
			}

			parsedConfigs, err := ParseInputConfigReader(strings.NewReader(inputConfigStr))
			if err != nil {
				t.Fatalf("ParseInputConfigReader() error = %v", err)
			}
			if len(parsedConfigs) == 0 {
				t.Fatalf("Expected at least one config to be parsed in %s", tc)
			}
			ic := parsedConfigs[0]

			out := bytes.NewBuffer(nil)

			base := &GenerateGithubWorkflowBase{
				Version:     "1.0.0",
				Now:         time.Now(),
				ConfigFile:  "test.config",
				InputConfig: ic,
			}

			var templateName string
			var data interface{}
			switch ic.Type {
			case "Github AppImage Release":
				data = &GenerateGithubAppImageTemplateData{GenerateGithubWorkflowBase: base}
				templateName = "github-appimage.tmpl"
			case "Github Binary Release":
				data = &GenerateGithubBinaryTemplateData{GenerateGithubWorkflowBase: base}
				templateName = "github-binary.tmpl"
			case "Web AppImage":
				data = &GenerateWebAppImageTemplateData{
					GenerateGithubAppImageTemplateData: &GenerateGithubAppImageTemplateData{
						GenerateGithubWorkflowBase: base,
					},
				}
				templateName = "web-appimage.tmpl"
			default:
				t.Fatalf("Unknown type: %s", ic.Type)
			}

			if err := templates.ExecuteTemplate(out, templateName, data); err != nil {
				t.Fatalf("ExecuteTemplate() error = %v", err)
			}

			result := out.String()

			if expectedYamlStr != "" && !strings.Contains(result, strings.TrimSpace(expectedYamlStr)) {
				t.Errorf("Expected string %q not found in output for %s. Result was:\n%s", strings.TrimSpace(expectedYamlStr), tc, result)
			}
			if expectedTagsCommand := "tags=$(cat tags.txt)"; strings.Contains(inputConfigStr, "cat tags.txt") && !strings.Contains(result, expectedTagsCommand) {
				t.Errorf("Expected custom TagsCommand logic %q not found in generated output.", expectedTagsCommand)
			}
		})
	}
}
