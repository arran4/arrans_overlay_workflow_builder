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
