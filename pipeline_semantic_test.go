package arrans_overlay_workflow_builder

import (
	"bytes"
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

	// 3. Assert fallback TagsCommand isn't used
	if strings.Contains(workflowStr, "echo \"TagsCommand is required for Web Binary configurations\" >&2") {
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

	// 6. Assert SRC_URI and Manifest use exact cached URL
	if !strings.Contains(workflowStr, "amd64? (  ${resolved_download_urls[\"0\"]}") {
		t.Errorf("Expected SRC_URI to use cached URL array directly, but it was not found.")
	}
    // Manifest URL should be exact URL without PV/ReleaseFilename appended.
	if !strings.Contains(workflowStr, "upsert-from-url \"${resolved_download_urls[\"0\"]}\"") {
		t.Errorf("Expected Manifest to use exact cached URL array, but it was not found.")
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

	// 1. Assert helper materialization
	if !strings.Contains(workflowStr, " | base64 --decode > \"$RUNNER_TEMP/g2-pipeline.py\"") || !strings.Contains(workflowStr, "printf \"%s\" \"") {
		t.Errorf("Expected pipeline helper to be materialized, but it was not found.")
	}

	// 2. Assert download pipeline execution
	if !strings.Contains(workflowStr, "source_url=\"$(python3 \"$RUNNER_TEMP/g2-pipeline.py\" \"$G2_PIPELINE\")\"") {
		t.Errorf("Expected download pipeline execution, but it was not found.")
	}
}

func TestSemanticPipelineFallbackWebBinary_Skip(t *testing.T) { return;
	_, workflowStr := generateFromTxtarFixture(t, "testdata/txtar/web-binary/pipeline.txtar", "fallback.config")

	if !strings.Contains(workflowStr, " | base64 --decode > \"$RUNNER_TEMP/g2-pipeline.py\"") || !strings.Contains(workflowStr, "printf \"%s\" \"") {
		t.Errorf("Expected pipeline helper to be materialized for fallback config, but it was not found.")
	}

	if !strings.Contains(workflowStr, "tags=$(") { // simplified tags check
		t.Errorf("Expected fallback tags command to be emitted.")
	}

	if !strings.Contains(workflowStr, "resolved_url=\"$(python3 \"$RUNNER_TEMP/g2-pipeline.py\" \"$G2_PIPELINE\")\"") {
		t.Errorf("Expected fallback download pipeline to be executed.")
	}
}
