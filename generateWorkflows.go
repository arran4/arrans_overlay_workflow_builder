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
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	//go:embed "templates/*.tmpl"
	templateFiles embed.FS
)

type ExternalResource struct {
	Keyword         string
	ReleaseFilename string
	Archived        bool
}

func GenerateGithubWorkflows(file string, outputDir string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	inputConfigs, err := ParseInputConfigReader(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("parsing %s: %w", file, err)
	}
	return GenerateGithubWorkflowsFromInputConfigs(file, inputConfigs, outputDir)
}

func GenerateGithubWorkflowsFromInputConfigs(file string, inputConfigs []*InputConfig, outputDir string) error {
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
	templates, err := ParseWorkflowTemplates()
	if err != nil {
		return err
	}
	now := time.Now()
	_ = os.MkdirAll(outputDir, 0755)
	for _, inputConfig := range inputConfigs {
		if err := inputConfig.GenerateGithubWorkflow(file, now, templates, outputDir); err != nil {
			return err
		}
	}
	return nil
}

func ParseWorkflowTemplates() (*template.Template, error) {
	subFs, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		return nil, fmt.Errorf("searching templates subdirectory: %w", err)
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
			"ebuildvardoublequotedSemanticVersionPrereleaseHack1": func(s string) string {
				return os.Expand(s, func(s string) string {
					switch s {
					case "VERSION":
						return "${originalVersion}"
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
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	return templates, nil
}

func (ic *InputConfig) GenerateGithubWorkflow(file string, now time.Time, templates *template.Template, outputDir string) error {
	if err := ic.Validate(); err != nil {
		return fmt.Errorf("for %s validating config: %w", ic.EbuildName, err)
	}
	out := bytes.NewBuffer(nil)
	var workflowName string
	var data interface {
		WorkflowFileName() string
		TemplateFileName() string
	}
	switch ic.Type {
	case "Github AppImage Release":
		data = &GenerateGithubAppImageTemplateData{
			Now:         now,
			ConfigFile:  file,
			InputConfig: ic,
		}
	case "Github Binary Release":
		data = &GenerateGithubBinaryTemplateData{
			Now:         now,
			ConfigFile:  file,
			InputConfig: ic,
		}
	default:
		return fmt.Errorf("unknown type %s", ic.Type)
	}
	if err := templates.ExecuteTemplate(out, data.TemplateFileName(), data); err != nil {
		return fmt.Errorf("for %s excuting template: %w", ic.EbuildName, err)
	}
	workflowName = data.WorkflowFileName()
	n := filepath.Join(outputDir, workflowName)
	if err := os.WriteFile(n, out.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", n, err)
	}
	fmt.Printf("Written: %s\n", n)
	return nil
}

func (ic *InputConfig) Cron() string {
	i := uint64(0)
	for _, r := range ic.GithubRepo {
		i += uint64(r)
	}
	minute := i % 60
	i /= 60
	hour := i % 24
	return fmt.Sprintf("%d %d * * *", minute, hour)
}
