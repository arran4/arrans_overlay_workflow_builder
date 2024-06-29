package arrans_overlay_workflow_builder

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
		if err := templates.ExecuteTemplate(out, "github-appimage.tmpl", &GenerateGithubAppImageTemplateData{
			InputConfig: inputConfig,
		}); err != nil {
			return fmt.Errorf("excuting template: %w", err)
		}
		n := filepath.Join(outputDir, fmt.Sprintf("%s-%s-update.yaml", inputConfig.Category, strings.TrimSuffix(inputConfig.EbuildName, ".ebuild")))
		if err := os.WriteFile(n, out.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", n, err)
		}
		fmt.Printf("Written: %s\n", n)
	}
	return nil
}
