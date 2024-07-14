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
	"strconv"
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

type ExternalResource struct {
	Keyword         string
	ReleaseFilename string
	Archived        bool
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

func (ggaitd *GenerateGithubAppImageTemplateData) ExternalResources() map[string]*ExternalResource {
	result := make(map[string]*ExternalResource)
	for programName, program := range ggaitd.Programs {
		for kw, rfn := range ggaitd.Programs[programName].ReleasesFilename {
			result[rfn] = &ExternalResource{
				Keyword:         kw,
				ReleaseFilename: rfn,
				Archived:        len(program.ArchiveFilename) > 0,
			}
		}
	}
	return result
}

func GenerateGithubWorkflows(file string) error {
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
			"quoteStr": strconv.Quote,
			"actionvardoublequoted": func(s string) string {
				return os.Expand(s, func(s string) string {
					switch s {
					case "VERSION":
						return "${version}"
					case "TAG":
						return "${tag}"
					case "GITHUB_OWNER":
						return "${{ env.github_owner }}"
					case "GITHUB_REPO":
						return "${{ env.github_repo }}"
					default:
						return fmt.Sprintf("${%s}", s)
					}
				})
			},
			"ebuildvardoublequoted": func(s string) string {
				return os.Expand(s, func(s string) string {
					switch s {
					case "VERSION":
						return "\\${PV}"
					case "TAG":
						return "${tag}"
					case "GITHUB_OWNER":
						return "${{ env.github_owner }}"
					case "GITHUB_REPO":
						return "${{ env.github_repo }}"
					case "KEYWORD":
						return "\\${ARCH}"
					default:
						return fmt.Sprintf("${%s}", s)
					}
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
		var workflowName string
		switch inputConfig.Type {
		case "Github AppImage":
			data := &GenerateGithubAppImageTemplateData{
				Now:         now,
				ConfigFile:  file,
				InputConfig: inputConfig,
			}
			if err := templates.ExecuteTemplate(out, "github-appimage.tmpl", data); err != nil {
				return fmt.Errorf("for %s excuting template: %w", inputConfig.EbuildName, err)
			}
			workflowName = data.WorkflowFileName()
		default:
			return fmt.Errorf("unknown type %s: %w", inputConfig.Type, err)
		}
		n := filepath.Join(outputDir, workflowName)
		if err := os.WriteFile(n, out.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", n, err)
		}
		fmt.Printf("Written: %s\n", n)
	}
	return nil
}
