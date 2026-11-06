package arrans_overlay_workflow_builder

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/xml"
	"errors"
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

// GenerateGithubWorkflowsFromInputConfigs renders and publishes GitHub Actions workflows
// for multiple input configurations.
//
// Publication safety contract (#180):
//  1. Validation & Pre-render Phase: All workflow templates are rendered, normalized,
//     and validated in-memory before any files are created or modified on disk.
//     If any input configuration or template execution fails, no files are written
//     and existing output files remain completely untouched.
//  2. Publication Phase: Outputs are written to outputDir. If an I/O error occurs
//     during publication, best-effort rollback restores previously modified files
//     to their original contents and removes newly created partial files.
//     Recovery includes the failing destination, restores contents and modes,
//     removes new files/directories, and reports every failed recovery operation.
//     This is best-effort rollback, not atomic publication: concurrent readers can
//     see intermediate files. Concurrent writers are unsupported; crashes, SIGKILL,
//     power loss and rollback I/O failures can leave partial state. No fsync or
//     crash-recovery journal is provided. This is unrelated to package-state #155.
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

	// Phase 1: Render and validate all candidate workflows in-memory before writing.
	type renderedOutput struct {
		path    string
		content []byte
	}
	outputs := make([]renderedOutput, 0, len(inputConfigs))
	targets := make(map[string]bool)
	for _, inputConfig := range inputConfigs {
		targetPath, content, err := inputConfig.RenderGithubWorkflow(file, now, templates, outputDir, version, ops...)
		if err != nil {
			return err
		}
		targetPath = filepath.Clean(targetPath)
		if targets[targetPath] {
			return fmt.Errorf("duplicate workflow output path %s", targetPath)
		}
		targets[targetPath] = true
		outputs = append(outputs, renderedOutput{path: targetPath, content: content})
	}

	if len(outputs) == 0 {
		return nil
	}

	// Inspect the complete previous state before any filesystem mutation. A
	// failed read/stat is not evidence of absence; symlinks are not output files.
	type previousFile struct {
		content []byte
		mode    fs.FileMode
	}
	originals := make(map[string]previousFile)
	for _, out := range outputs {
		info, err := fsys.Lstat(out.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspecting existing workflow %s: %w", out.path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("existing workflow %s is not a regular file", out.path)
		}
		data, err := fsys.ReadFile(out.path)
		if err != nil {
			return fmt.Errorf("backing up existing workflow %s: %w", out.path, err)
		}
		originals[out.path] = previousFile{content: bytes.Clone(data), mode: info.Mode()}
	}

	// Remember only directories that did not exist before publication, deepest
	// first. Never remove a pre-existing directory during recovery.
	var newDirs []string
	for dir := filepath.Clean(outputDir); ; dir = filepath.Dir(dir) {
		info, err := fsys.Lstat(dir)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("output ancestor %s is not a directory", dir)
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspecting output directory %s: %w", dir, err)
		}
		newDirs = append(newDirs, dir)
		if filepath.Dir(dir) == dir {
			return fmt.Errorf("no existing ancestor for output directory %s", outputDir)
		}
	}

	// Recovery includes the attempted write: WriteFile may truncate/modify its
	// destination before returning an error. Continue after each recovery error
	// and preserve all causes so callers can identify unrecovered paths.
	rollback := func(cause error, attempted int) error {
		failures := []error{cause}
		for i := attempted - 1; i >= 0; i-- {
			path := outputs[i].path
			if original, exists := originals[path]; exists {
				if err := fsys.WriteFile(path, original.content, original.mode); err != nil {
					failures = append(failures, fmt.Errorf("restoring workflow %s: %w", path, err))
				}
				if err := fsys.Chmod(path, original.mode); err != nil {
					failures = append(failures, fmt.Errorf("restoring permissions for %s: %w", path, err))
				}
			} else if err := fsys.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				failures = append(failures, fmt.Errorf("removing new workflow %s: %w", path, err))
			}
		}
		for _, dir := range newDirs {
			if err := fsys.Remove(dir); err != nil && !errors.Is(err, os.ErrNotExist) {
				failures = append(failures, fmt.Errorf("removing new output directory %s: %w", dir, err))
			}
		}
		return errors.Join(failures...)
	}

	if err := fsys.MkdirAll(outputDir, 0755); err != nil {
		return rollback(fmt.Errorf("creating output directory %s: %w", outputDir, err), 0)
	}
	for i, out := range outputs {
		mode := fs.FileMode(0644)
		if original, exists := originals[out.path]; exists {
			mode = original.mode
		}
		if err := fsys.WriteFile(out.path, out.content, mode); err != nil {
			return rollback(fmt.Errorf("writing %s: %w", out.path, err), i+1)
		}
	}
	for _, out := range outputs {
		fmt.Printf("Written: %s\n", out.path)
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

			"formatReleaseFilenameForUrl": func(originalVersion interface{}, filename interface{}) string {
				filenameStr := fmt.Sprintf("%v", filename)
				return os.Expand(filenameStr, func(s string) string {
					switch s {
					case "VERSION":
						if fmt.Sprintf("%v", originalVersion) == "true" {
							return "${originalVersion}"
						}
						return "${version}"
					case "PV":
						return "${version}"
					case "TAG":
						return "${tag}"
					case "GITHUB_OWNER":
						return "${{ env.github_owner }}"
					case "GITHUB_REPO":
						return "${{ env.github_repo }}"
					default:
						return "${" + s + "}"
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
			"shell_single_quote": ShellSingleQuote,
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
type OptGentooAuthors bool
type OptSourceLicense string
type OptGeneratedEbuildGentooAuthors bool
type OptGeneratedEbuildSourceLicense string

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
	GentooAuthors               bool
	SourceLicense               string
	HasExplicitSourceLicense    bool
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

// ResolveHeaderPolicy combines global options with per-entry overrides.
// The third result distinguishes an explicitly configured source license from
// the default MIT source header (which controls the Gentoo Authors GPL-2 header).
func (ic *InputConfig) ResolveHeaderPolicy(ops ...any) (gentooAuthors bool, sourceLicense string, hasExplicitSourceLicense bool) {
	sourceLicense = "MIT"
	for _, opt := range ops {
		switch o := opt.(type) {
		case OptGentooAuthors:
			gentooAuthors = bool(o)
		case OptGeneratedEbuildGentooAuthors:
			gentooAuthors = bool(o)
		case OptSourceLicense:
			sourceLicense = string(o)
			hasExplicitSourceLicense = true
		case OptGeneratedEbuildSourceLicense:
			sourceLicense = string(o)
			hasExplicitSourceLicense = true
		}
	}
	if ic != nil {
		if ic.GentooAuthors != nil {
			gentooAuthors = *ic.GentooAuthors
		}
		if ic.GeneratedEbuildGentooAuthors != nil {
			gentooAuthors = *ic.GeneratedEbuildGentooAuthors
		}
		if ic.SourceLicense != nil && *ic.SourceLicense != "" {
			sourceLicense = *ic.SourceLicense
			hasExplicitSourceLicense = true
		}
		if ic.GeneratedEbuildSourceLicense != nil && *ic.GeneratedEbuildSourceLicense != "" {
			sourceLicense = *ic.GeneratedEbuildSourceLicense
			hasExplicitSourceLicense = true
		}
	}
	return gentooAuthors, sourceLicense, hasExplicitSourceLicense
}

// RenderGithubWorkflow renders and normalizes the GitHub Actions workflow YAML for the InputConfig
// without writing it to disk. It returns the calculated target file path within outputDir,
// the rendered bytes, and any validation or template execution error.
func (ic *InputConfig) RenderGithubWorkflow(file string, now time.Time, templates *template.Template, outputDir, version string, ops ...any) (string, []byte, error) {
	if err := ic.Validate(); err != nil {
		return "", nil, fmt.Errorf("for %s validating config: %w", ic.EbuildName, err)
	}
	out := bytes.NewBuffer(nil)
	var data interface {
		WorkflowFileName() string
		TemplateFileName() string
	}
	gentooAuthors, sourceLicense, hasExplicitSourceLicense := ic.ResolveHeaderPolicy(ops...)

	base := &GenerateGithubWorkflowBase{
		Version:                  version,
		Now:                      now,
		ConfigFile:               file,
		InputConfig:              ic,
		GentooAuthors:            gentooAuthors,
		SourceLicense:            sourceLicense,
		HasExplicitSourceLicense: hasExplicitSourceLicense,
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
			return "", nil, fmt.Errorf("explicitly configured names must be validated as Gentoo repository names: cannot be empty")
		}
		matched, _ := regexp.MatchString(`^[A-Za-z0-9_][A-Za-z0-9_-]*$`, *base.GlobalOverlayRepoName)
		if !matched {
			return "", nil, fmt.Errorf("explicitly configured names must be validated as Gentoo repository names: invalid characters in name '%s'", *base.GlobalOverlayRepoName)
		}
		// Also must not look like a package name ending with a version.
		// A simple heuristic for "ends in a version" which Gentoo prohibits for repo names
		// matches -<number>(.<number>)*[a-z]?(_(alpha|beta|pre|rc|p)[0-9]*)*(-r[0-9]+)?$
		versionMatched, _ := regexp.MatchString(`-[0-9]+(\.[0-9]+)*[a-z]?(_(alpha|beta|pre|rc|p)[0-9]*)*(-r[0-9]+)?$`, *base.GlobalOverlayRepoName)
		if versionMatched {
			return "", nil, fmt.Errorf("explicitly configured names must be validated as Gentoo repository names: cannot end in a valid version string '%s'", *base.GlobalOverlayRepoName)
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
		return "", nil, fmt.Errorf("unknown type %s", ic.Type)
	}
	if err := templates.ExecuteTemplate(out, data.TemplateFileName(), data); err != nil {
		return "", nil, fmt.Errorf("for %s excuting template: %w", ic.EbuildName, err)
	}
	rendered := normalizeGeneratedWorkflow(out.Bytes())
	n := filepath.Join(outputDir, data.WorkflowFileName())
	return n, rendered, nil
}

func (ic *InputConfig) GenerateGithubWorkflow(file string, now time.Time, templates *template.Template, outputDir, version string, ops ...any) error {
	filePath, rendered, err := ic.RenderGithubWorkflow(file, now, templates, outputDir, version, ops...)
	if err != nil {
		return err
	}
	var fsys util.FileSystem = util.OSFS{}
	for _, opt := range ops {
		switch o := opt.(type) {
		case util.FileSystem:
			fsys = o
		}
	}

	if err := fsys.WriteFile(filePath, rendered, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", filePath, err)
	}
	fmt.Printf("Written: %s\n", filePath)
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

// GetHeaderLines provides header data as separate physical lines. Keep GetHeader
// for callers using the original string-valued policy API.
func (b *GenerateGithubWorkflowBase) GetHeaderLines() []string {
	return strings.Split(b.GetHeader(), "\n")
}

// ShellSingleQuote produces one literal Bash argument, preserving dollar signs,
// backticks, backslashes and embedded single quotes without expansion.
func ShellSingleQuote(content string) string {
	return "'" + strings.ReplaceAll(content, "'", "'\\''") + "'"
}

func (b *GenerateGithubWorkflowBase) GetHeader() string {
	if b.GentooAuthors {
		if !b.HasExplicitSourceLicense {
			return fmt.Sprintf("# Copyright %s Gentoo Authors\n# Distributed under the terms of the GNU General Public License v2", b.Now.Format("2006"))
		} else {
			return fmt.Sprintf("# Copyright %s Gentoo Authors\n# SPDX-License-Identifier: %s", b.Now.Format("2006"), b.SourceLicense)
		}
	} else {
		return fmt.Sprintf("# SPDX-License-Identifier: %s", b.SourceLicense)
	}
}
