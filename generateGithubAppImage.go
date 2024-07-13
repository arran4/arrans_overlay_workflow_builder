package arrans_overlay_workflow_builder

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"
	"time"
)

var (
	//go:embed "templates/*.tmpl"
	templateFiles embed.FS
)

type GenerateGithubAppImageTemplateData struct {
	*InputConfig
	Now        time.Time
	ConfigFile string
}

func (ggaitd *GenerateGithubAppImageTemplateData) Cron() string {
	i := uint64(0)
	for _, r := range ggaitd.GithubRepo {
		i += uint64(r)
	}
	minute := i % 60
	i /= 60
	hour := i % 24
	return fmt.Sprintf("%d %d * * *", minute, hour)
}

func (ggaitd *GenerateGithubAppImageTemplateData) WorkflowName() string {
	return fmt.Sprintf("%s/%s update", ggaitd.Category, ggaitd.PackageName())
}

func (ggaitd *GenerateGithubAppImageTemplateData) KeywordList() []string {
	// TODO this is obsolete as it needs to be specific per program, migrate it to program
	keywords := make([]string, 0)
	for programName := range ggaitd.Programs {
		for key := range ggaitd.Programs[programName].ReleasesFilename {
			keywords = append(keywords, key)
		}
	}
	sort.Strings(keywords)
	return keywords
}

func (ggaitd *GenerateGithubAppImageTemplateData) Keywords() string {
	return strings.Join(ggaitd.KeywordList(), " ")
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
	case "P":
		return "\\${P}"
	case "GITHUB_OWNER":
		return "${{ env.github_owner }}"
	case "GITHUB_REPO":
		return "${{ env.github_repo }}"
	case "RELEASE_FILENAME":
		//return wf.EbuildVariableSubstitutor(wf.GenerateGithubAppImageTemplateData.ReleasesFilename[wf.Keyword])
		return "TODO" // TODO migrate to program.
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
	case "P":
		return "${{ env.epn }}-${version}"
	case "TAG":
		return "${tag}"
	case "GITHUB_OWNER":
		return "${{ env.github_owner }}"
	case "GITHUB_REPO":
		return "${{ env.github_repo }}"
	case "RELEASE_FILENAME":
		//return wf.GHAVariableSubstitutor(wf.GenerateGithubAppImageTemplateData.ReleasesFilename[wf.Keyword])
		return "TODO" // TODO migrate to program.
	case "KEYWORD":
		return wf.Keyword
	default:
		return fmt.Sprintf("${%s}", s)
	}
}

func (wf *WgetFile) SrcUri() string {
	return fmt.Sprintf("%s -> %s", os.Expand(wf.UrlTemplate, wf.EbuildVariableSubstitutor), os.Expand(wf.LocalFilenameTemplate, wf.EbuildVariableSubstitutor))
}

func (ggaitd *GenerateGithubAppImageTemplateData) HasDesktopFile() bool {
	for _, p := range ggaitd.Programs {
		if p.HasDesktopFile() {
			return true
		}
	}
	return false
}

func (ggaitd *GenerateGithubAppImageTemplateData) IsArchived() bool {
	for _, p := range ggaitd.Programs {
		if p.IsArchived() {
			return true
		}
	}
	return false
}

func (ggaitd *GenerateGithubAppImageTemplateData) ExternalResources() WgetFiles {
	result := make(WgetFiles, 0)
	for programName := range ggaitd.Programs {
		for kw, rfn := range ggaitd.Programs[programName].ReleasesFilename {
			result = append(result, &WgetFile{
				GenerateGithubAppImageTemplateData: ggaitd,
				UrlTemplate:                        "https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${TAG}/" + rfn,
				LocalFilenameTemplate: strings.Join(slices.DeleteFunc(slices.Clone([]string{"${P}", programName}), func(s string) bool {
					return s == ""
				}), "-") + ".${KEYWORD}",
				Keyword:   kw,
				Extension: filepath.Ext(rfn),
			})
		}
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
	const lineIndentation = "                "
	const openEco = `echo "`
	const closeEcho = `"`
	const openVariable = `SRC_URI=\"`
	const closeVariable = `\"`
	switch len(surls) {
	case 0:
		return "# Missing " + openEco + closeEcho // TODO consider
	case 1:
		return openEco + openVariable + surls[0] + closeVariable + closeEcho
	default:
		b := bytes.NewBufferString(openEco + openVariable)
		b.WriteString(closeEcho + "\n")
		b.WriteString(lineIndentation + openEco)
		for _, surl := range surls {
			// Assuming no quoting nonsense
			b.WriteString(surl)
			b.WriteString(closeEcho + "\n")
			b.WriteString(lineIndentation + openEco)
		}
		b.WriteString(closeVariable + closeEcho)
		return b.String()
	}
}

func GenerateGithubAppImage(file string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	inputConfigs, err := ParseInputConfigReader(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("parsing %s: %w", file, err)
	}
	missing := false
	for _, inputConfig := range inputConfigs {
		if inputConfig.Category == "" {
			log.Printf("%s needs a category", inputConfig.EbuildName)
			missing = true
		}
	}
	if missing {
		return fmt.Errorf("missing required fields")
	}
	subFs, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		return fmt.Errorf("searching templates subdirectory: %w", err)
	}
	templates, err := template.New("").
		Delims("[[", "]]").
		Funcs(map[string]any{
			"join": strings.Join,
			"filterEmpty": func(strs ...string) []string {
				return slices.DeleteFunc(slices.Clone(strs), func(s string) bool {
					return s == ""
				})
			},
		}).
		ParseFS(subFs, "*.tmpl")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}
	outputDir := "./output"
	now := time.Now()
	_ = os.MkdirAll(outputDir, 0755)
	for _, inputConfig := range inputConfigs {
		out := bytes.NewBuffer(nil)
		data := &GenerateGithubAppImageTemplateData{
			Now:         now,
			ConfigFile:  file,
			InputConfig: inputConfig,
		}
		if err := templates.ExecuteTemplate(out, "github-appimage.tmpl", data); err != nil {
			return fmt.Errorf("for %s excuting template: %w", inputConfig.EbuildName, err)
		}
		n := filepath.Join(outputDir, data.WorkflowFileName())
		if err := os.WriteFile(n, out.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", n, err)
		}
		fmt.Printf("Written: %s\n", n)
	}
	return nil
}
