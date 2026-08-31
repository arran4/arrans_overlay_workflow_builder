package arrans_overlay_workflow_builder

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/txtar"
)

func generateFromTxtarFixture(t *testing.T, filepath string, configName string) (*InputConfig, string) {
	b, err := testdataFS.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", filepath, err)
	}

	ar := txtar.Parse(b)
	var configData []byte
	for _, f := range ar.Files {
		if f.Name == configName {
			configData = f.Data
			break
		}
	}
	if configData == nil {
		t.Fatalf("No .config file found in txtar")
	}

	parsedConfigs, err := ParseInputConfigReader(bytes.NewReader(configData))
	if err != nil {
		t.Fatalf("ParseInputConfigReader() error = %v", err)
	}
	if len(parsedConfigs) == 0 {
		t.Fatalf("Expected at least one config to be parsed in %s", filepath)
	}
	ic := parsedConfigs[0]

	templates, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	out := bytes.NewBuffer(nil)
	base := &GenerateGithubWorkflowBase{
		Version:     "1.0.0",
		Now:         time.Date(2026, time.July, 12, 0, 0, 0, 0, time.UTC),
		ConfigFile:  "test.config",
		Schedule:    "24 2 * * *",
		InputConfig: ic,
	}

	var templateName string
	var data any
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

	return ic, string(normalizeGeneratedWorkflow(out.Bytes()))
}

func TestSemanticPipelineWebBinary(t *testing.T) {
	_, workflowStr := generateFromTxtarFixture(t, "testdata/txtar/web-binary/pipeline.txtar", "multi-resource.config")

	// 1. Assert helper materialization
	if !strings.Contains(workflowStr, " | base64 --decode > \"$RUNNER_TEMP/g2-pipeline.py\"") || !strings.Contains(workflowStr, "printf \"%s\" \"") {
		t.Errorf("Expected pipeline helper to be materialized, but it was not found.")
	}

	// 2. Assert version pipeline execution
	if !strings.Contains(workflowStr, "tags=\"$(python3 \"$RUNNER_TEMP/g2-pipeline.py\" \"$G2_PIPELINE\")\"") {
		t.Errorf("Expected version pipeline execution, but it was not found.")
	}

	// 3. Assert fallback TagsCommand isn't used (ensure we don't fall through to error)
	if strings.Contains(workflowStr, "TagsCommand or VersionPipeline is required") {
		t.Errorf("Should not fallback to tags command validation when VersionPipeline is specified.")
	}

	// 4. Assert download pipeline execution
	if !strings.Contains(workflowStr, "resolved_url=\"$(python3 \"$RUNNER_TEMP/g2-pipeline.py\" \"$G2_PIPELINE\")\"") {
		t.Errorf("Expected download pipeline execution, but it was not found.")
	}

	// 5. Assert cached URL is populated and used independently for resource 0 and 1
	if !strings.Contains(workflowStr, "resolved_download_urls[\"0\"]=\"$resolved_url\"") {
		t.Errorf("Expected resolved_download_urls[\"0\"] array to be populated, but it was not found.")
	}
	if !strings.Contains(workflowStr, "resolved_download_urls[\"1\"]=\"$resolved_url\"") {
		t.Errorf("Expected resolved_download_urls[\"1\"] array to be populated, but it was not found.")
	}

	// 6. Assert SRC_URI and Manifest use exact cached URL and filenames correctly match resources
	if !strings.Contains(workflowStr, "amd64? (  ${resolved_download_urls[\"0\"]} -> \\${P}-example-amd64-\\${PV}.tar.gz  )") {
		t.Errorf("Expected SRC_URI to use cached URL array directly for amd64, but it was not found.")
	}
	if !strings.Contains(workflowStr, "arm64? (  ${resolved_download_urls[\"1\"]} -> \\${P}-example-arm64-\\${PV}.tar.gz  )") {
		t.Errorf("Expected SRC_URI to use cached URL array directly for arm64, but it was not found.")
	}

	// Manifest URL should be exact URL without PV/ReleaseFilename appended to the source URL string.
	if !strings.Contains(workflowStr, "upsert-from-url \"${resolved_download_urls[\"0\"]}\" \"${{ env.epn }}-${version}-example-amd64-${version}.tar.gz\"") {
		t.Errorf("Expected Manifest to use exact cached URL array for amd64, but it was not found.")
	}
	if !strings.Contains(workflowStr, "upsert-from-url \"${resolved_download_urls[\"1\"]}\" \"${{ env.epn }}-${version}-example-arm64-${version}.tar.gz\"") {
		t.Errorf("Expected Manifest to use exact cached URL array for arm64, but it was not found.")
	}

	// 7. Assert no trailing \${PV} inside the evaluated pipeline blocks
	if strings.Contains(workflowStr, "${resolved_download_urls[\"0\"]}\\${PV}") {
		t.Errorf("Expected SRC_URI NOT to append PV to cached pipeline URL.")
	}

	// 8. Assert no unresolved pipeline placeholders remain in the replacement string
	if strings.Contains(workflowStr, "${G2_PIPELINE//\\${RELEASE_FILENAME}/example-${VERSION}.tar.gz}") {
		t.Errorf("Expected NO raw ${VERSION} string replacements left stranded in pipeline env blocks.")
	}
}

func TestSemanticPipelineWebAppImage(t *testing.T) {
	_, workflowStr := generateFromTxtarFixture(t, "testdata/txtar/web-appimage/pipeline.txtar", "input.config")

	// 1. Assert helper materialization occurs before execution
	helperIdx := strings.Index(workflowStr, "base64 --decode > \"$RUNNER_TEMP/g2-pipeline.py\"")
	execIdx := strings.Index(workflowStr, "python3 \"$RUNNER_TEMP/g2-pipeline.py\"")

	if helperIdx == -1 {
		t.Errorf("Expected pipeline helper to be materialized, but it was not found.")
	}
	if execIdx == -1 {
		t.Errorf("Expected download pipeline execution, but it was not found.")
	}
	if helperIdx != -1 && execIdx != -1 && helperIdx > execIdx {
		t.Errorf("Expected pipeline helper to be materialized BEFORE execution. HelperIdx: %d, ExecIdx: %d", helperIdx, execIdx)
	}
}

func TestSemanticPipelineFallbackWebBinary(t *testing.T) {
	_, workflowStr := generateFromTxtarFixture(t, "testdata/txtar/web-binary/pipeline.txtar", "fallback.config")

	if !strings.Contains(workflowStr, " | base64 --decode > \"$RUNNER_TEMP/g2-pipeline.py\"") || !strings.Contains(workflowStr, "printf \"%s\" \"") {
		t.Errorf("Expected pipeline helper to be materialized for fallback config, but it was not found.")
	}

	if !strings.Contains(workflowStr, "tags=$(curl -sL https://example.com/downloads/ | htmlq -a href a | grep -oP 'v\\d+\\.\\d+\\.\\d+')") {
		t.Errorf("Expected TagsCommand to be rendered for version discovery.")
	}

	if strings.Contains(workflowStr, "tags=\"$(python3 \"$RUNNER_TEMP/g2-pipeline.py\"") {
		t.Errorf("Expected VersionPipeline NOT to be synthesized and executed since TagsCommand and DownloadMatch exist.")
	}

	if !strings.Contains(workflowStr, "resolved_url=\"$(python3 \"$RUNNER_TEMP/g2-pipeline.py\" \"$G2_PIPELINE\")\"") {
		t.Errorf("Expected fallback download pipeline to be executed.")
	}
}

func executeWebAppImageTemplateForTest(t *testing.T, pipeline string, customVersion bool) error {
	config := &InputConfig{
		Type:             "Web AppImage",
		EbuildName:       "test",
		Category:         "app-misc",
		DownloadPageUrl:  "https://example.com",
		DownloadPipeline: pipeline,
	}
	if customVersion {
		config.CustomVersionSource = "curl | grep"
	}

	templates, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("ParseWorkflowTemplates() failed: %v", err)
	}
	out := bytes.NewBuffer(nil)
	base := &GenerateGithubWorkflowBase{InputConfig: config}
	data := &GenerateWebAppImageTemplateData{GenerateGithubAppImageTemplateData: &GenerateGithubAppImageTemplateData{GenerateGithubWorkflowBase: base}}
	return templates.ExecuteTemplate(out, "web-appimage.tmpl", data)
}

func TestSemanticPipelineWebAppImagePlaceholders(t *testing.T) {
	err1 := executeWebAppImageTemplateForTest(t, "get(https://example.com/downloads) | html_links | regex(${RELEASE_FILENAME})", false)
	if err1 == nil || !strings.Contains(err1.Error(), "RELEASE_FILENAME is not supported for Web AppImage pipeline generation") {
		t.Errorf("Expected failure for RELEASE_FILENAME, got: %v", err1)
	}

	err2 := executeWebAppImageTemplateForTest(t, "get(https://example.com/downloads) | html_links | regex(${TAG})", false)
	if err2 == nil || !strings.Contains(err2.Error(), "TAG is not supported for Web AppImage pipeline generation before download discovery") {
		t.Errorf("Expected failure for TAG, got: %v", err2)
	}

	err3 := executeWebAppImageTemplateForTest(t, "get(https://example.com/downloads/${VERSION}) | html_links", false)
	if err3 == nil || !strings.Contains(err3.Error(), "VERSION placeholder used in pipeline but no CustomVersionSource is configured") {
		t.Errorf("Expected failure for VERSION without source, got: %v", err3)
	}

	err4 := executeWebAppImageTemplateForTest(t, "get(https://example.com/downloads/${VERSION}) | html_links", true)
	if err4 != nil {
		t.Errorf("Expected success for VERSION with CustomVersionSource, got: %v", err4)
	}
}

func TestSemanticPipelineBashExecution(t *testing.T) {
	configStr := `Type Web Binary
Category app-misc
EbuildName example-multi
Description Example Multi Binary
Homepage https://example.com/
DownloadBaseUrl https://example.com/downloads/
VersionPipeline get(https://example.com/downloads/) | html_links | regex(v\d+\.\d+\.\d+)
DownloadPipeline get(https://example.com/downloads/${VERSION}) | html_links | regex(${RELEASE_FILENAME}) | first
ProgramName example
Binary amd64=>example-amd64-${VERSION}.tar.gz > example > example`

	config, err := ParseInputConfigReader(strings.NewReader(configStr))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	base := &GenerateGithubWorkflowBase{InputConfig: config[0]}
	b_tmpl := &GenerateWebBinaryTemplateData{
		GenerateGithubBinaryTemplateData: &GenerateGithubBinaryTemplateData{
			GenerateGithubWorkflowBase: base,
		},
	}
	b_tmpl.Programs = map[string]*Program{
		"example": {
			ProgramName: "example",
			Binary: map[string][]string{
				"amd64": {"example-amd64-${VERSION}.tar.gz", "example", "example"},
			},
		},
	}

	tmpl, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "web-binary.tmpl", b_tmpl)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	yamlOutput := buf.String()

	var snippet string
	lines := strings.Split(yamlOutput, "\n")
	for i, line := range lines {
		if strings.Contains(line, "resolved_filename=\"$(printf '%s'") {
			snippet = strings.Join(lines[i-1:i+6], "\n")
			break
		}
	}

	if snippet == "" {
		t.Fatalf("Could not find pipeline bash snippet in generated YAML")
	}

	bashScript := "originalVersion=\"1.2.3-beta1\"\n" +
		"version=\"1.2.3_beta1\"\n" +
		"tag=\"v1.2.3-beta1\"\n" +
		snippet + "\n" +
		"echo \"$G2_PIPELINE\"\n"

	cmd := exec.Command("bash", "-c", bashScript)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute bash snippet: %v\nOutput: %s\nScript:\n%s", err, out, bashScript)
	}

	result := strings.TrimSpace(string(out))
	expected := "get(https://example.com/downloads/1.2.3-beta1) | html_links | regex(example-amd64-1.2.3-beta1.tar.gz) | first"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWebAppImagePipelineBashExecution(t *testing.T) {
	configStr := `Type Web AppImage
Category app-misc
EbuildName example-appimage
Description Example AppImage
Homepage https://example.com/
DownloadPageUrl https://example.com/downloads/
CustomVersionSource printf '1.2.3-beta1'
DownloadPipeline get(https://example.com/downloads/${VERSION}) | html_links | regex(v\d+\.\d+\.\d+) | first
ProgramName example
Binary amd64=>test.AppImage > example`

	config, err := ParseInputConfigReader(strings.NewReader(configStr))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	base := &GenerateGithubWorkflowBase{InputConfig: config[0]}
	b_tmpl := &GenerateWebAppImageTemplateData{
		GenerateGithubAppImageTemplateData: &GenerateGithubAppImageTemplateData{
			GenerateGithubWorkflowBase: base,
		},
	}
	b_tmpl.Programs = map[string]*Program{
		"example": {
			ProgramName: "example",
			Binary: map[string][]string{
				"amd64": {"test.AppImage", "example"},
			},
		},
	}

	tmpl, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "web-appimage.tmpl", b_tmpl)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	yamlOutput := buf.String()

	if strings.Contains(yamlOutput, "G2_PIPELINE_TMPL") {
		t.Fatalf("Generated YAML contains G2_PIPELINE_TMPL")
	}

	var snippet string
	lines := strings.Split(yamlOutput, "\n")
	for i, line := range lines {
		if strings.Contains(line, "version=$(printf '1.2.3-beta1')") {
			snippet = strings.Join(lines[i:i+3], "\n")
			break
		}
	}

	if snippet == "" {
		t.Fatalf("Could not find pipeline bash snippet in generated YAML")
	}

	bashScript := "set -euo pipefail\n" +
		snippet + "\n" +
		"echo \"$G2_PIPELINE\"\n"

	cmd := exec.Command("bash", "-c", bashScript)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute bash snippet: %v\nOutput: %s\nScript:\n%s", err, out, bashScript)
	}

	result := strings.TrimSpace(string(out))
	expected := "get(https://example.com/downloads/1.2.3-beta1) | html_links | regex(v\\d+\\.\\d+\\.\\d+) | first"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWebBinaryReleaseFilenameFallback(t *testing.T) {
	configStr := `Type Web Binary
Category app-misc
EbuildName example-multi
Description Example Multi Binary
Homepage https://example.com/
DownloadBaseUrl https://example.com/downloads/
VersionPipeline get(https://example.com/downloads/) | html_links | regex(v\d+\.\d+\.\d+)
DownloadPipeline get(https://example.com/downloads/${VERSION}) | html_links | regex(${RELEASE_FILENAME}) | first
ProgramName example
Binary amd64=>example > example > example`

	config, err := ParseInputConfigReader(strings.NewReader(configStr))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	base := &GenerateGithubWorkflowBase{InputConfig: config[0]}
	b_tmpl := &GenerateWebBinaryTemplateData{
		GenerateGithubBinaryTemplateData: &GenerateGithubBinaryTemplateData{
			GenerateGithubWorkflowBase: base,
		},
	}
	b_tmpl.Programs = map[string]*Program{
		"example": {
			ProgramName: "example",
			Binary: map[string][]string{
				"amd64": {"", "example", "example"},
			},
		},
	}

	tmpl, err := ParseWorkflowTemplates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "web-binary.tmpl", b_tmpl)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	yamlOutput := buf.String()

	if !strings.Contains(yamlOutput, "resolved_filename=\"$(printf '%s' \"MA==\" | base64 --decode)\"") {
		t.Fatalf("Could not find fallback base64 encoded index in generated YAML. output:\n%s", yamlOutput)
	}
}
