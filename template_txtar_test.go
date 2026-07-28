package arrans_overlay_workflow_builder

import (
	"bytes"
	"embed"
	"flag"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/txtar"
)

var updateTxtar = flag.Bool("update-txtar", false, "update txtar expected.yaml files")

//go:embed testdata/txtar/**/*.txtar
var testdataFS embed.FS

func TestWorkflowTemplates(t *testing.T) {
	var cases []string
	err := fs.WalkDir(testdataFS, "testdata/txtar", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
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
			if strings.TrimSpace(string(ar.Comment)) == "" {
				t.Fatalf("Missing txtar description in %s", tc)
			}

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
				Now:         time.Date(2026, time.July, 12, 0, 0, 0, 0, time.UTC),
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
			case "Github Cmake Release":
				data = &GenerateGithubCmakeTemplateData{GenerateGithubWorkflowBase: base}
				templateName = "github-cmake.tmpl"
			case "Web AppImage":
				data = &GenerateWebAppImageTemplateData{
					GenerateGithubAppImageTemplateData: &GenerateGithubAppImageTemplateData{
						GenerateGithubWorkflowBase: base,
					},
				}
				templateName = "web-appimage.tmpl"
			case "Web Binary":
				data = &GenerateWebBinaryTemplateData{
					GenerateGithubBinaryTemplateData: &GenerateGithubBinaryTemplateData{
						GenerateGithubWorkflowBase: base,
					},
				}
				templateName = "web-binary.tmpl"
			default:
				t.Fatalf("Unknown type: %s", ic.Type)
			}

			if err := templates.ExecuteTemplate(out, templateName, data); err != nil {
				t.Fatalf("ExecuteTemplate() error = %v", err)
			}

			result := string(normalizeGeneratedWorkflow(out.Bytes()))

			if *updateTxtar {
				for i := range ar.Files {
					if ar.Files[i].Name == "expected.yaml" {
						ar.Files[i].Data = []byte(result)
						if err := os.WriteFile(tc, txtar.Format(ar), 0644); err != nil {
							t.Fatalf("update testcase %s: %v", tc, err)
						}
						return
					}
				}
				t.Fatalf("Missing expected.yaml in %s", tc)
			}

			if expectedYamlStr == "" {
				t.Fatalf("Missing expected.yaml output in %s", tc)
			}
			expected := strings.ReplaceAll(expectedYamlStr, "\r\n", "\n")
			actual := strings.ReplaceAll(result, "\r\n", "\n")
			if actual != expected {
				t.Errorf("generated workflow does not match full expected output for %s\nexpected:\n%s\nactual:\n%s", tc, expected, actual)
			}
		})
	}
}
