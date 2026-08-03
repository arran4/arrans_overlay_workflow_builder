package arrans_overlay_workflow_builder

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/stoewer/go-strcase"
)

const defaultAppImageExtension = "AppImage"

// GenerateWebAppImageConfigEntry builds a configuration entry for a web-hosted
// AppImage by inspecting the download page and deriving sensible defaults.
func GenerateWebAppImageConfigEntry(pageURL, matchExpr, linkExt string) (*InputConfig, []util.AppImageLink, error) {
	ext := strings.TrimPrefix(linkExt, ".")
	if ext == "" {
		ext = defaultAppImageExtension
	}
	links, err := util.FindAppImageLinksWithExtension(pageURL, matchExpr, ext)
	if err != nil {
		return nil, nil, err
	}
	if len(links) == 0 {
		return nil, nil, fmt.Errorf("no AppImage links discovered at %s", pageURL)
	}

	latest := links[len(links)-1]
	baseName, prefix, version := splitAppImageName(latest.Filename)
	programName := sanitizeProgramName(baseName)
	packageName := programName
	if packageName == "" {
		packageName = "appimage"
	}
	if programName == "" {
		programName = packageName
	}

	ic := &InputConfig{
		Type:            "Web AppImage",
		DownloadPageUrl: pageURL,
		Category:        DefaultCategory,
		EbuildName:      fmt.Sprintf("%s-appimage.ebuild", packageName),
		Description:     fmt.Sprintf("%s AppImage", humanizeName(programName)),
		Homepage:        pageURL,
		GithubRepo:      packageName,
		Features:        map[string]string{},
		Workarounds:     map[string]string{},
		Programs:        map[string]*Program{},
		License:         DefaultLicense,
	}

	if matchExpr != "" {
		ic.DownloadMatch = matchExpr
	} else {
		ic.DownloadMatch = defaultDownloadMatch(prefix, programName, ext)
	}

	binaryPattern := fmt.Sprintf("%s.AppImage", strings.TrimSuffix(latest.Filename, filepath.Ext(latest.Filename)))
	if version != "" && prefix != "" {
		binaryPattern = fmt.Sprintf("%s${VERSION}.AppImage", prefix)
	}

	program := &Program{
		ProgramName:  programName,
		Binary:       map[string][]string{"~amd64": {binaryPattern, fmt.Sprintf("%s.AppImage", packageName)}},
		Dependencies: []string{},
	}
	ic.Programs[program.ProgramName] = program

	log.Printf("discovered %d AppImage links, latest %s", len(links), latest.Filename)
	return ic, links, nil
}

// ConfigViewWebAppImage prints the derived configuration for inspection.
func ConfigViewWebAppImage(pageURL, matchExpr, linkExt string) error {
	ic, _, err := GenerateWebAppImageConfigEntry(pageURL, matchExpr, linkExt)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", ic.String())
	return nil
}

// ConfigAddWebAppImage appends a generated configuration entry to the provided file.
func ConfigAddWebAppImage(toConfig, pageURL, matchExpr, linkExt string) error {
	ic, _, err := GenerateWebAppImageConfigEntry(pageURL, matchExpr, linkExt)
	if err != nil {
		return err
	}

	log.Printf("Reading config")
	config, err := ReadConfigurationFile(toConfig)
	if err != nil {
		return fmt.Errorf("reading configuration file: %s: %w", toConfig, err)
	}

	for _, entry := range config {
		if entry.EntryNumber >= ic.EntryNumber {
			ic.EntryNumber = entry.EntryNumber + 1
		}
	}

	log.Printf("Appending to config as entry id: %d", ic.EntryNumber)
	if err := AppendToConfigurationFile(toConfig, ic); err != nil {
		return fmt.Errorf("appending to configuration file: %s: %w", toConfig, err)
	}
	return nil
}

// CmdOneshotWebAppImage writes the derived configuration to stdout and renders a workflow.
func CmdOneshotWebAppImage(pageURL, matchExpr, linkExt, outputDir, version string) error {
	ic, _, err := GenerateWebAppImageConfigEntry(pageURL, matchExpr, linkExt)
	if err != nil {
		return err
	}

	log.Printf("Showing potential addition to config as entry id: %d", ic.EntryNumber)
	fmt.Printf("%s\n", ic.String())

	missing := false
	if ic.Category == "" {
		log.Printf("%s needs a category", ic.EbuildName)
		missing = true
	}
	if missing {
		return fmt.Errorf("missing required fields")
	}

	templates, err := ParseWorkflowTemplates()
	if err != nil {
		return err
	}
	_ = os.MkdirAll(outputDir, 0755)
	if err := ic.GenerateGithubWorkflow("-", time.Now(), templates, outputDir, version); err != nil {
		return err
	}
	return nil
}

func sanitizeProgramName(name string) string {
	if name == "" {
		return ""
	}
	return strcase.KebabCase(name)
}

func humanizeName(name string) string {
	if name == "" {
		return "App"
	}
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		if len(part) == 1 {
			parts[i] = strings.ToUpper(part)
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func splitAppImageName(filename string) (baseName, prefix, version string) {
	trimmed := strings.TrimSuffix(filename, filepath.Ext(filename))
	trimmed = strings.Trim(trimmed, "-_ ")
	versionPattern := regexp.MustCompile(`(?i)v?\d+(?:\.\d+)*[0-9A-Za-z-]*`)
	matches := versionPattern.FindAllStringIndex(trimmed, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		start, end := matches[i][0], matches[i][1]
		candidate := trimmed[start:end]
		if !isLikelyVersion(candidate) {
			continue
		}
		version = candidate
		prefix = trimmed[:start]
		baseName = strings.Trim(prefix, "-_ ")
		if baseName == "" {
			baseName = trimmed
		}
		if prefix != "" {
			return baseName, prefix, version
		}
	}
	return trimmed, trimmed, ""
}

func isLikelyVersion(candidate string) bool {
	lower := strings.ToLower(candidate)
	if strings.Contains(lower, "beta") || strings.Contains(lower, "rc") {
		return true
	}
	if strings.Count(candidate, ".") > 0 {
		return true
	}
	digits := 0
	for _, r := range candidate {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return digits >= 3
}

func defaultDownloadMatch(prefix, programName, extension string) string {
	target := prefix
	if target == "" {
		target = programName
	}
	target = strings.TrimSpace(target)
	ext := strings.TrimPrefix(extension, ".")
	if ext == "" {
		ext = defaultAppImageExtension
	}
	pattern := fmt.Sprintf(`\.%s$`, regexp.QuoteMeta(ext))
	if target == "" {
		return pattern
	}
	return fmt.Sprintf("%s.*%s", regexp.QuoteMeta(target), pattern)
}
