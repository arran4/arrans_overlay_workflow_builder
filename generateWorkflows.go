package arrans_overlay_workflow_builder

import (
	"bytes"
	"embed"
	"encoding/xml"
	"fmt"
	"github.com/arran4/g2"
	"github.com/stoewer/go-strcase"
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

func GenerateGithubWorkflows(file, outputDir, version string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	inputConfigs, err := ParseInputConfigReader(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("parsing %s: %w", file, err)
	}
	return GenerateGithubWorkflowsFromInputConfigs(file, inputConfigs, outputDir, version)
}

func GenerateGithubWorkflowsFromInputConfigs(file string, inputConfigs []*InputConfig, outputDir, version string) error {
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
		if err := inputConfig.GenerateGithubWorkflow(file, now, templates, outputDir, version); err != nil {
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
			"UseFlagSafe": strcase.SnakeCase,
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

type GenerateGithubWorkflowBase struct {
	*InputConfig
	Version    string
	Now        time.Time
	ConfigFile string
}

func (b *GenerateGithubWorkflowBase) DefaultMetadata() (string, error) {
	pkgMd := &g2.PkgMetadata{
		XMLName: xml.Name{
			Local: "pkgmetadata",
		},
		Upstream: &g2.Upstream{
			RemoteID: []g2.RemoteID{},
		},
	}
	if b.MaintainerEmail != "" {
		pkgMd.Maintainers = append(pkgMd.Maintainers, g2.Maintainer{
			Email: b.MaintainerEmail,
			Name:  b.MaintainerName,
			Type:  "person",
		})
	}
	if b.GithubOwner != "" && b.GithubRepo != "" {
		pkgMd.Upstream.RemoteID = append(pkgMd.Upstream.RemoteID, g2.RemoteID{
			Type: "github",
			Text: fmt.Sprintf("%s/%s", b.GithubOwner, b.GithubRepo),
		})
	}
	if len(pkgMd.Upstream.RemoteID) == 0 {
		pkgMd.Upstream = nil
	}
	o, err := xml.MarshalIndent(pkgMd, "", "\t")
	if err != nil {
		return "", fmt.Errorf("marshalling metadata: %w", err)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE pkgmetadata SYSTEM "http://www.gentoo.org/dtd/metadata.dtd">
%s`, string(o)), nil
}

func (b *GenerateGithubWorkflowBase) G2MetadataArgs() string {
	if b == nil {
		return ""
	}
	args := ""
	if b.MaintainerEmail != "" {
		email := strings.ReplaceAll(strings.ReplaceAll(b.MaintainerEmail, "\\", "\\\\"), "\"", "\\\"")
		name := strings.ReplaceAll(strings.ReplaceAll(b.MaintainerName, "\\", "\\\\"), "\"", "\\\"")
		args += fmt.Sprintf("-m \"%s:%s:person\" ", email, name)
	} else {
		args += "-m \"gentoo@arran4.com:Arran Ubels:person\" "
	}
	if b.GithubOwner != "" && b.GithubRepo != "" {
		owner := strings.ReplaceAll(strings.ReplaceAll(b.GithubOwner, "\\", "\\\\"), "\"", "\\\"")
		repo := strings.ReplaceAll(strings.ReplaceAll(b.GithubRepo, "\\", "\\\\"), "\"", "\\\"")
		args += fmt.Sprintf("-u \"github:%s/%s\" ", owner, repo)
	}
	return strings.TrimSpace(args)
}

func (ic *InputConfig) GenerateGithubWorkflow(file string, now time.Time, templates *template.Template, outputDir, version string) error {
	if err := ic.Validate(); err != nil {
		return fmt.Errorf("for %s validating config: %w", ic.EbuildName, err)
	}
	out := bytes.NewBuffer(nil)
	var workflowName string
	var data interface {
		WorkflowFileName() string
		TemplateFileName() string
	}
	base := &GenerateGithubWorkflowBase{
		Version:     version,
		Now:         now,
		ConfigFile:  file,
		InputConfig: ic,
	}
	switch ic.Type {
	case "Github AppImage Release":
		data = &GenerateGithubAppImageTemplateData{
			GenerateGithubWorkflowBase: base,
		}
	case "Web AppImage":
		data = &GenerateWebAppImageTemplateData{
			GenerateGithubAppImageTemplateData: &GenerateGithubAppImageTemplateData{
				GenerateGithubWorkflowBase: base,
			},
		}
	case "Github Binary Release":
		data = &GenerateGithubBinaryTemplateData{
			GenerateGithubWorkflowBase: base,
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
