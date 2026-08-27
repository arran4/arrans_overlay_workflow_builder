package arrans_overlay_workflow_builder

// GenerateWebAppImageTemplateData reuses the Github AppImage workflow data but
// renders using a different template for non-GitHub downloads.
type GenerateWebAppImageTemplateData struct {
	*GenerateGithubAppImageTemplateData
}

// TemplateFileName returns the template used for arbitrary website downloads.
func (gwatd *GenerateWebAppImageTemplateData) TemplateFileName() string {
	return "web-appimage.tmpl"
}

func (gwatd *GenerateWebAppImageTemplateData) PipelineScriptBase64() string {
	return gwatd.GenerateGithubAppImageTemplateData.PipelineScriptBase64()
}
func (gwatd *GenerateWebAppImageTemplateData) GetDownloadPipeline() string {
	return gwatd.InputConfig.GetDownloadPipeline()
}
