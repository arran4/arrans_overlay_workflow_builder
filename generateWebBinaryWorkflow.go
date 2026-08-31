package arrans_overlay_workflow_builder

// GenerateWebBinaryTemplateData reuses the Github Binary workflow data but
// renders using a different template for non-GitHub downloads.
type GenerateWebBinaryTemplateData struct {
	*GenerateGithubBinaryTemplateData
}

// TemplateFileName returns the template used for arbitrary website binary downloads.
func (gwbtd *GenerateWebBinaryTemplateData) TemplateFileName() string {
	return "web-binary.tmpl"
}

func (gwbtd *GenerateWebBinaryTemplateData) PipelineScriptBase64() string {
	return gwbtd.GenerateGithubBinaryTemplateData.PipelineScriptBase64()
}
func (gwbtd *GenerateWebBinaryTemplateData) GetVersionPipeline() string {
	return gwbtd.InputConfig.GetVersionPipeline()
}
