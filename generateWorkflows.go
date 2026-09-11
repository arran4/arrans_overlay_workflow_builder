package arrans_overlay_workflow_builder

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/arran4/g2"
	"github.com/stoewer/go-strcase"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	//go:embed "templates/*.tmpl" "templates/_partials/*.tmpl" "templates/_partials/pipeline.py"
	templateFiles embed.FS
)

type ExternalResource struct {
	Keyword         string
	ReleaseFilename string
	Archived        bool
}

type NextGenerate func(file string, outputDir string, version string, ops ...any) error

func normalizeGeneratedWorkflow(output []byte) []byte {
	lines := bytes.Split(output, []byte("\n"))
	normalized := make([][]byte, 0, len(lines))
	for i := range lines {
		lines[i] = bytes.TrimRight(lines[i], " \t")
		if len(lines[i]) == 0 && len(normalized) > 0 && len(normalized[len(normalized)-1]) == 0 {
			continue
		}
		normalized = append(normalized, lines[i])
	}

	compacted := make([][]byte, 0, len(normalized))
	for i := range normalized {
		if len(normalized[i]) == 0 && i > 0 && i+1 < len(normalized) {
			previousIndent := len(normalized[i-1]) - len(bytes.TrimLeft(normalized[i-1], " "))
			nextIndent := len(normalized[i+1]) - len(bytes.TrimLeft(normalized[i+1], " "))
			if previousIndent > 0 && previousIndent == nextIndent {
				continue
			}
		}
		compacted = append(compacted, normalized[i])
	}
	return bytes.Join(compacted, []byte("\n"))
}

func GenerateGithubWorkflows(file, outputDir, version string, ops ...any) error {
	var fsys util.FileSystem = util.OSFS{}

	for _, opt := range ops {
		switch o := opt.(type) {
		case util.FileSystem:
			fsys = o
		}
	}

	b, err := fsys.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	inputConfigs, err := ParseInputConfigReader(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("parsing %s: %w", file, err)
	}
	return GenerateGithubWorkflowsFromInputConfigs(file, inputConfigs, outputDir, version, ops...)
}

func GenerateGithubWorkflowsFromInputConfigs(file string, inputConfigs []*InputConfig, outputDir, version string, ops ...any) error {
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
	var fsys util.FileSystem = util.OSFS{}
	for _, opt := range ops {
		switch o := opt.(type) {
		case util.FileSystem:
			fsys = o
		case time.Time:
			now = o
		}
	}

	_ = fsys.MkdirAll(outputDir, 0755)
	for _, inputConfig := range inputConfigs {
		if err := inputConfig.GenerateGithubWorkflow(file, now, templates, outputDir, version, ops...); err != nil {
			return err
		}
	}
	return nil
}

func IndentContent(indent, content string) string {
	if content == "" {
		return ""
	}
	var res strings.Builder
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		if l == "" && i == len(lines)-1 {
			continue
		}
		res.WriteString(indent)
		res.WriteString(l)
		res.WriteString("\n")
	}
	return res.String()
}

func ShellEchoEvalContent(content string) string {
	if content == "" {
		return ""
	}
	var res strings.Builder
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		if l == "" && i == len(lines)-1 {
			continue
		}
		escaped := strings.ReplaceAll(l, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "`", "\\`")
		fmt.Fprintf(&res, "echo \"%s\"\n", escaped)
	}
	return res.String()
}

func ShellEchoContent(content string) string {
	if content == "" {
		return ""
	}
	var res strings.Builder
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		if l == "" && i == len(lines)-1 {
			continue
		}
		escaped := strings.ReplaceAll(l, "'", "'\\''")
		fmt.Fprintf(&res, "echo '%s'\n", escaped)
	}
	return res.String()
}

func getEbuildIncludes(includes map[string][]string, section string) (string, error) {
	if includes == nil {
		return "", nil
	}
	paths, ok := includes[section]
	if !ok {
		return "", nil
	}
	var result strings.Builder
	for _, p := range paths {
		if strings.Contains(p, "\n") {
			result.WriteString(p)
			if !strings.HasSuffix(p, "\n") {
				result.WriteString("\n")
			}
		} else {
			content, err := os.ReadFile(strings.TrimSpace(p))
			if err == nil {
				result.WriteString(string(content))
				if len(content) > 0 && content[len(content)-1] != '\n' {
					result.WriteString("\n")
				}
			} else {
				result.WriteString(p)
				result.WriteString("\n")
			}
		}
	}
	return result.String(), nil
}

func ParseWorkflowTemplates() (*template.Template, error) {
	subFs, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		return nil, fmt.Errorf("searching templates subdirectory: %w", err)
	}
	templates, err := template.New("").
		Delims("[[", "]]").
		Funcs(map[string]any{
			"dict": func(values ...any) (map[string]any, error) {
				if len(values)%2 != 0 {
					return nil, fmt.Errorf("invalid dict call: odd number of arguments")
				}
				dict := make(map[string]any, len(values)/2)
				for i := 0; i < len(values); i += 2 {
					key, ok := values[i].(string)
					if !ok {
						return nil, fmt.Errorf("dict keys must be strings")
					}
					dict[key] = values[i+1]
				}
				return dict, nil
			},
			"stringsContains": strings.Contains,
			"base64": func(v any) string {
				return base64.StdEncoding.EncodeToString([]byte(fmt.Sprint(v)))
			},
			"fail": func(msg string) (string, error) {
				return "", fmt.Errorf("%s", msg)
			},
			"exitOnMatch": func(v any) bool {
				if d, ok := v.(map[string]any); ok {
					if val, exists := d["ExitOnMatch"]; exists {
						if b, ok := val.(bool); ok {
							return b
						}
					}
					return false
				}
				// Default to continue if we can't determine
				return false
			},
			"list": func(values ...any) []any {
				return values
			},
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
			"replace":           strings.ReplaceAll,
			"getEbuildIncludes": getEbuildIncludes,
			"repeat": func(count int, s string) string {
				return strings.Repeat(s, count)
			},
			"indent":             IndentContent,
			"shell_echo":         ShellEchoEvalContent,
			"shell_echo_literal": ShellEchoContent,
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
		ParseFS(subFs, "*.tmpl", "_partials/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	return templates, nil
}

type OptGenerateMd5Cache bool
type OptGenerateOverlay bool
type OptOverlayRepoName string
type OptUpsertOverlayRepoName bool

type GenerateGithubWorkflowBase struct {
	*InputConfig
	Version    string
	Now        time.Time
	ConfigFile string
	Schedule   string

	GlobalGenerateMd5Cache      bool
	GlobalGenerateOverlay       bool
	GlobalOverlayRepoName       *string
	GlobalUpsertOverlayRepoName bool
}

func (b *GenerateGithubWorkflowBase) GetOverlayRepoName() string {
	if b.GlobalOverlayRepoName == nil {
		return ""
	}
	return *b.GlobalOverlayRepoName
}

func (b *GenerateGithubWorkflowBase) HasOverlayRepoName() bool {
	return b.GlobalOverlayRepoName != nil
}

func (b *GenerateGithubWorkflowBase) Cron() string {
	if b.Schedule != "" {
		return b.Schedule
	}
	return b.InputConfig.Cron()
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

func (ic *InputConfig) GenerateGithubWorkflow(file string, now time.Time, templates *template.Template, outputDir, version string, ops ...any) error {
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
	for _, opt := range ops {
		switch o := opt.(type) {
		case OptGenerateMd5Cache:
			base.GlobalGenerateMd5Cache = bool(o)
		case OptGenerateOverlay:
			base.GlobalGenerateOverlay = bool(o)
		case OptOverlayRepoName:
			v := string(o)
			base.GlobalOverlayRepoName = &v
		case OptUpsertOverlayRepoName:
			base.GlobalUpsertOverlayRepoName = bool(o)
		}
	}
	if base.GlobalOverlayRepoName != nil {
		if *base.GlobalOverlayRepoName == "" {
			return fmt.Errorf("explicitly configured names must be validated as Gentoo repository names: cannot be empty")
		}
		matched, _ := regexp.MatchString(`^[A-Za-z0-9_][A-Za-z0-9_-]*$`, *base.GlobalOverlayRepoName)
		if !matched {
			return fmt.Errorf("explicitly configured names must be validated as Gentoo repository names: invalid characters in name '%s'", *base.GlobalOverlayRepoName)
		}
		// Also must not look like a package name ending with a version.
		// A simple heuristic for "ends in a version" which Gentoo prohibits for repo names
		// matches -<number>(.<number>)*[a-z]?(_(alpha|beta|pre|rc|p)[0-9]*)*(-r[0-9]+)?$
		versionMatched, _ := regexp.MatchString(`-[0-9]+(\.[0-9]+)*[a-z]?(_(alpha|beta|pre|rc|p)[0-9]*)*(-r[0-9]+)?$`, *base.GlobalOverlayRepoName)
		if versionMatched {
			return fmt.Errorf("explicitly configured names must be validated as Gentoo repository names: cannot end in a valid version string '%s'", *base.GlobalOverlayRepoName)
		}
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
	case "Github Cmake Release":
		data = &GenerateGithubCmakeTemplateData{
			GenerateGithubWorkflowBase: base,
		}
	case "Web Binary":
		data = &GenerateWebBinaryTemplateData{
			GenerateGithubBinaryTemplateData: &GenerateGithubBinaryTemplateData{
				GenerateGithubWorkflowBase: base,
			},
		}
	default:
		return fmt.Errorf("unknown type %s", ic.Type)
	}
	if err := templates.ExecuteTemplate(out, data.TemplateFileName(), data); err != nil {
		return fmt.Errorf("for %s excuting template: %w", ic.EbuildName, err)
	}
	rendered := normalizeGeneratedWorkflow(out.Bytes())
	workflowName = data.WorkflowFileName()
	n := filepath.Join(outputDir, workflowName)

	var fsys util.FileSystem = util.OSFS{}
	for _, opt := range ops {
		switch o := opt.(type) {
		case util.FileSystem:
			fsys = o
		}
	}

	if err := fsys.WriteFile(n, rendered, 0644); err != nil {
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

func (b *GenerateGithubWorkflowBase) WorkaroundVersionReplacement() string {
	return b.InputConfig.WorkaroundVersionReplacement()
}

func (b *GenerateGithubWorkflowBase) WorkaroundGentooVersionRegularExpression() string {
	return b.InputConfig.WorkaroundGentooVersionRegularExpression()
}

func (b *GenerateGithubWorkflowBase) FeatureGenerateMd5Cache() bool {
	return b.GlobalGenerateMd5Cache || b.InputConfig.FeatureGenerateMd5Cache()
}

func (b *GenerateGithubWorkflowBase) FeatureGenerateOverlay() bool {
	return b.GlobalGenerateOverlay || b.InputConfig.FeatureGenerateOverlay()
}

func (b *GenerateGithubWorkflowBase) FeatureUpsertOverlayRepoName() bool {
	return b.GlobalUpsertOverlayRepoName || b.InputConfig.FeatureUpsertOverlayRepoName()
}

func (b *GenerateGithubWorkflowBase) PipelineScriptBase64() string {
	bytes, err := templateFiles.ReadFile("templates/_partials/pipeline.py")
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

func (b *GenerateGithubWorkflowBase) GetVersionPipelineBase64() string {
	return base64.StdEncoding.EncodeToString([]byte(b.GetVersionPipeline()))
}

func (b *GenerateGithubWorkflowBase) GetDownloadPipelineBase64() string {
	return base64.StdEncoding.EncodeToString([]byte(b.GetDownloadPipeline()))
}
