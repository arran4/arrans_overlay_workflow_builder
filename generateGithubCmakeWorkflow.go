package arrans_overlay_workflow_builder

import (
	"fmt"
	"strings"
)

type GenerateGithubCmakeTemplateData struct {
	*GenerateGithubWorkflowBase
}

func (ggctd *GenerateGithubCmakeTemplateData) TemplateFileName() string {
	return "github-cmake.tmpl"
}

func (ggctd *GenerateGithubCmakeTemplateData) WorkflowName() string {
	return fmt.Sprintf("%s/%s update", ggctd.Category, ggctd.PackageName())
}

func (ggctd *GenerateGithubCmakeTemplateData) WorkflowFileName() string {
	return fmt.Sprintf("%s-%s-update.yaml", ggctd.Category, ggctd.PackageName())
}

func (ggctd *GenerateGithubCmakeTemplateData) PackageName() string {
	return strings.TrimSuffix(ggctd.EbuildName, ".ebuild")
}

func (ggctd *GenerateGithubCmakeTemplateData) Metadata() (string, error) {
	return ggctd.DefaultMetadata()
}

func (ggctd *GenerateGithubCmakeTemplateData) WorkaroundVersionReplacement() string {
	return ggctd.InputConfig.WorkaroundVersionReplacement()
}

func (ggctd *GenerateGithubCmakeTemplateData) WorkaroundGentooVersionRegularExpression() string {
	return ggctd.InputConfig.WorkaroundGentooVersionRegularExpression()
}
