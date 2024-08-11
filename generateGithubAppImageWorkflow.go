package arrans_overlay_workflow_builder

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type GenerateGithubAppImageTemplateData struct {
	*InputConfig
	Now        time.Time
	ConfigFile string
}

func (ggaitd *GenerateGithubAppImageTemplateData) WorkflowName() string {
	return fmt.Sprintf("%s/%s update", ggaitd.Category, ggaitd.PackageName())
}

func (ggaitd *GenerateGithubAppImageTemplateData) KeywordList() []string {
	keywords := make([]string, 0)
	for programName := range ggaitd.Programs {
		for key := range ggaitd.Programs[programName].Binary {
			keywords = append(keywords, key)
		}
	}
	sort.Strings(keywords)
	return keywords
}

func (ggaitd *GenerateGithubAppImageTemplateData) Dependencies() []string {
	keywords := make([]string, 0)
	for programName := range ggaitd.Programs {
		keywords = append(keywords, ggaitd.Programs[programName].Dependencies...)
	}
	sort.Strings(keywords)
	return keywords
}

func (ggaitd *GenerateGithubAppImageTemplateData) Keywords() string {
	return strings.Join(ggaitd.KeywordList(), " ")
}

func (ggaitd *GenerateGithubAppImageTemplateData) TemplateFileName() string {
	return "github-appimage.tmpl"
}

func (ggaitd *GenerateGithubAppImageTemplateData) MaskedKeywords() string {
	list := ggaitd.KeywordList()
	for i := range list {
		list[i] = "~" + strings.TrimPrefix(list[i], "~")
	}
	return strings.Join(list, " ")
}

func (ggaitd *GenerateGithubAppImageTemplateData) WorkflowFileName() string {
	return fmt.Sprintf("%s-%s-update.yaml", ggaitd.Category, ggaitd.PackageName())
}

func (ggaitd *GenerateGithubAppImageTemplateData) PackageName() string {
	return strings.TrimSuffix(ggaitd.EbuildName, ".ebuild")
}

func (ggaitd *GenerateGithubAppImageTemplateData) HasDesktopFile() bool {
	for _, p := range ggaitd.Programs {
		if p.HasDesktopFile() {
			return true
		}
	}
	return false
}

func (ggaitd *GenerateGithubAppImageTemplateData) IsArchived(keyword string) bool {
	for _, p := range ggaitd.Programs {
		if p.IsArchived(keyword) {
			return true
		}
	}
	return false
}

func (ggaitd *GenerateGithubAppImageTemplateData) ExternalResources() map[string]*ExternalResource {
	result := make(map[string]*ExternalResource)
	for programName := range ggaitd.Programs {
		for kw, rfn := range ggaitd.Programs[programName].Binary {
			result[rfn[0]] = &ExternalResource{
				Keyword:         kw,
				ReleaseFilename: rfn[0],
				Archived:        len(rfn) > 2,
			}
		}
	}
	return result
}
