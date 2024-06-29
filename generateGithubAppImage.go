package arrans_overlay_workflow_builder

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

var (
	//go:embed "templates/*.tmpl"
	templateFiles embed.FS
)

type GenerateGithubAppImageTemplateData struct {
	*InputConfig
}

func (ggaitd *GenerateGithubAppImageTemplateData) Cron() string {
	minute := rand.Intn(60)
	hour := rand.Intn(24)
	return fmt.Sprintf("%d %d * * *", minute, hour)
}

func (ggaitd *GenerateGithubAppImageTemplateData) WorkflowName() string {
	return fmt.Sprintf("%s/%s update", ggaitd.Category, ggaitd.PackageName())
}

func (ggaitd *GenerateGithubAppImageTemplateData) KeywordList() []string {
	keywords := make([]string, 0, len(ggaitd.ReleasesFilename))
	for key := range ggaitd.ReleasesFilename {
		keywords = append(keywords, key)
	}
	sort.Strings(keywords)
	return keywords
}

func (ggaitd *GenerateGithubAppImageTemplateData) Keywords() string {
	return strings.Join(ggaitd.KeywordList(), " ")
}

func (ggaitd *GenerateGithubAppImageTemplateData) WorkflowFileName() string {
	return fmt.Sprintf("%s-%s-update.yaml", ggaitd.Category, ggaitd.PackageName())
}

func (ggaitd *GenerateGithubAppImageTemplateData) PackageName() string {
	return strings.TrimSuffix(ggaitd.EbuildName, ".ebuild")
}

type WgetFile struct {
	UrlTemplate                        string
	LocalFilenameTemplate              string
	Keyword                            string
	GenerateGithubAppImageTemplateData *GenerateGithubAppImageTemplateData
	// TODO use Extension to guide the template on how to unzip - requires additional config values for app image location
	Extension string
}

func (wf *WgetFile) WgetLocalFilename() string {
	return os.Expand(wf.LocalFilenameTemplate, wf.GHAVariableSubstitutor)
}

func (wf *WgetFile) UrlWget() string {
	return os.Expand(wf.UrlTemplate, wf.GHAVariableSubstitutor)
}

func (wf *WgetFile) EbuildVariableSubstitutor(s string) string {
	switch s {
	case "VERSION":
		return "\\${PV}"
	case "TAG":
		return "v\\${PV}"
	case "GITHUB_OWNER":
		return "${{ env.github_owner }}"
	case "GITHUB_REPO":
		return "${{ env.github_repo }}"
	case "RELEASE_FILENAME":
		return wf.EbuildVariableSubstitutor(wf.GenerateGithubAppImageTemplateData.ReleasesFilename[wf.Keyword])
	case "KEYWORD":
		return wf.Keyword
	default:
		return fmt.Sprintf("${%s}", s)
	}
}

func (wf *WgetFile) GHAVariableSubstitutor(s string) string {
	switch s {
	case "VERSION":
		return "${version}"
	case "TAG":
		return "${tag}"
	case "GITHUB_OWNER":
		return "${{ env.github_owner }}"
	case "GITHUB_REPO":
		return "${{ env.github_repo }}"
	case "RELEASE_FILENAME":
		return wf.GHAVariableSubstitutor(wf.GenerateGithubAppImageTemplateData.ReleasesFilename[wf.Keyword])
	case "KEYWORD":
		return wf.Keyword
	default:
		return fmt.Sprintf("${%s}", s)
	}
}

func (wf *WgetFile) SrcUri() string {
	return fmt.Sprintf("%s -> %s", os.Expand(wf.UrlTemplate, wf.EbuildVariableSubstitutor), os.Expand(wf.LocalFilenameTemplate, wf.EbuildVariableSubstitutor))
}

func (ggaitd *GenerateGithubAppImageTemplateData) ExternalResources() WgetFiles {
	result := make(WgetFiles, 0, len(ggaitd.ReleasesFilename))
	for kw, rfn := range ggaitd.ReleasesFilename {
		result = append(result, &WgetFile{
			GenerateGithubAppImageTemplateData: ggaitd,
			UrlTemplate:                        "https://github.com//${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${TAG}/" + rfn,
			LocalFilenameTemplate:              "${{ env.epn }}-${VERSION}.${KEYWORD}",
			Keyword:                            kw,
			Extension:                          filepath.Ext(rfn),
		})
	}
	return result
}

type WgetFiles []*WgetFile

func (wfs WgetFiles) SrcUris() []string {
	result := make([]string, 0, len(wfs))
	for _, wf := range wfs {
		if len(wfs) == 1 {
			result = append(result, wf.SrcUri())
		} else {
			result = append(result, fmt.Sprintf(" %s? ( %s )", wf.Keyword, wf.SrcUri()))
		}
	}
	return result
}

func (ggaitd *GenerateGithubAppImageTemplateData) SrcUriEchos() string {
	surls := ggaitd.ExternalResources().SrcUris()
	const linePrefix = "                echo \""
	const lineSuffix = "\"\n"
	b := bytes.NewBufferString(linePrefix + `SRC_URI="`)
	switch len(surls) {
	case 0:
		return "" // TODO consider
	case 1:
		return fmt.Sprintf(surls[0])
	default:
		b.WriteString(lineSuffix)
		b.WriteString(linePrefix)
		for _, surl := range surls {
			// Assuming no quoting nonsense
			b.WriteString(surl)
			b.WriteString(lineSuffix)
			b.WriteString(linePrefix)
		}
		b.WriteString(lineSuffix)
		b.WriteString(linePrefix)
	}
	b.WriteString("\"'")
	return b.String()
}

func GenerateGithubAppImage(file string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	inputConfigs, err := ParseInputConfigFile(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("parsing %s: %w", file, err)
	}
	subFs, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		return fmt.Errorf("searching templates subdirectory: %w", err)
	}
	templates, err := template.New("").Delims("[[", "]]").ParseFS(subFs, "*.tmpl")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}
	outputDir := "./output"
	_ = os.MkdirAll(outputDir, 0755)
	for _, inputConfig := range inputConfigs {
		out := bytes.NewBuffer(nil)
		data := &GenerateGithubAppImageTemplateData{
			InputConfig: inputConfig,
		}
		if err := templates.ExecuteTemplate(out, "github-appimage.tmpl", data); err != nil {
			return fmt.Errorf("excuting template: %w", err)
		}
		n := filepath.Join(outputDir, data.WorkflowFileName())
		if err := os.WriteFile(n, out.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", n, err)
		}
		fmt.Printf("Written: %s\n", n)
	}
	return nil
}
