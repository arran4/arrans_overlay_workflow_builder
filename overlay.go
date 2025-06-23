package arrans_binary_overlay_builder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	g2 "github.com/arran4/g2"
)

// GenerateEbuild renders the ebuild template for this config and writes it to outputDir.
func (ic *InputConfig) GenerateEbuild(templates *template.Template, outputDir, version string) error {
	if err := ic.Validate(); err != nil {
		return fmt.Errorf("for %s validating config: %w", ic.EbuildName, err)
	}
	out := bytes.NewBuffer(nil)
	tag := ic.TagFromVersion(version)
	base := &GenerateGithubWorkflowBase{
		Version:     version,
		Tag:         tag,
		Now:         time.Now(),
		ConfigFile:  "-",
		InputConfig: ic,
	}
	var data interface{ TemplateFileName() string }
	switch ic.Type {
	case "Github AppImage Release":
		data = &GenerateGithubAppImageTemplateData{GenerateGithubWorkflowBase: base}
	case "Github Binary Release":
		data = &GenerateGithubBinaryTemplateData{GenerateGithubWorkflowBase: base}
	default:
		return fmt.Errorf("unknown type %s", ic.Type)
	}
	if err := templates.ExecuteTemplate(out, data.TemplateFileName(), data); err != nil {
		return fmt.Errorf("for %s executing template: %w", ic.EbuildName, err)
	}
	dir := filepath.Join(outputDir, ic.Category, strings.TrimSuffix(ic.EbuildName, ".ebuild"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	name := filepath.Join(dir, fmt.Sprintf("%s-%s.ebuild", strings.TrimSuffix(ic.EbuildName, ".ebuild"), version))
	if err := os.WriteFile(name, out.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}
	manifest := filepath.Join(dir, "Manifest")
	switch d := data.(type) {
	case *GenerateGithubBinaryTemplateData:
		for _, er := range d.ExternalResources() {
			url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", ic.GithubOwner, ic.GithubRepo, tag, er.ReleaseFilename())
			size, b2, sha, err := g2.DownloadAndChecksum(url)
			if err != nil {
				return fmt.Errorf("download %s: %w", url, err)
			}
			line := fmt.Sprintf("DIST %s %d BLAKE2B %s SHA512 %s", er.ReleaseFilename(), size, b2, sha)
			if err := g2.UpsertManifest(manifest, er.ReleaseFilename(), line); err != nil {
				return fmt.Errorf("manifest: %w", err)
			}
		}
	case *GenerateGithubAppImageTemplateData:
		for _, er := range d.ExternalResources() {
			url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", ic.GithubOwner, ic.GithubRepo, tag, er.ReleaseFilename)
			size, b2, sha, err := g2.DownloadAndChecksum(url)
			if err != nil {
				return fmt.Errorf("download %s: %w", url, err)
			}
			line := fmt.Sprintf("DIST %s %d BLAKE2B %s SHA512 %s", er.ReleaseFilename, size, b2, sha)
			if err := g2.UpsertManifest(manifest, er.ReleaseFilename, line); err != nil {
				return fmt.Errorf("manifest: %w", err)
			}
		}
	}
	fmt.Printf("Written: %s\n", name)
	return nil
}
