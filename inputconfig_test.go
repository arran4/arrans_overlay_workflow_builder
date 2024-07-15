package arrans_overlay_workflow_builder

import (
	"bytes"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"testing"
)

const testConfigData = `
# Example config
Type Github AppImage
GithubProjectUrl https://github.com/janhq/jan/
DesktopFile jan
InstalledFilename jan
Category app-misc
EbuildName jan-appimage
Description Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)
Workaround Test Workaround
Workaround Test Workaround with value => Values
Homepage https://jan.ai/
ReleasesFilename amd64=>jan-linux-x86_64-${VERSION}.AppImage
ReleasesFilename arm64=>jan-linux-arm64-${VERSION}.AppImage

Type Github AppImage
GithubProjectUrl https://github.com/anotherorg/anotherrepo/
InstalledFilename anotherapp
Icons hicolor-apps root
ReleasesFilename amd64=>anotherrepo-${VERSION}.AppImage

Type Github AppImage
GithubProjectUrl https://github.com/probonopd/go-appimage
EbuildName go-appimage-appimage
Description  Go implementation of AppImage tools
License MIT
InstalledFilename appimagetool.AppImage
ReleasesFilename amd64=>appimagetool-838-x86_64.AppImage 
ProgramName appimaged
InstalledFilename appimaged.AppImage
ReleasesFilename amd64=> appimaged-838-x86_64.AppImage 
ProgramName mkappimage
InstalledFilename mkappimage.AppImage
ReleasesFilename amd64=>mkappimage-838-x86_64.AppImage
`

func TestParseConfigFile(t *testing.T) {
	configs, err := ParseInputConfigReader(bytes.NewReader([]byte(testConfigData)))
	if err != nil {
		t.Fatalf("error parsing config file: %v", err)
	}
	entryCount := 3
	if len(configs) != entryCount {
		t.Fatalf("expected %d config entries, got %d", entryCount, len(configs))
	}

	expectedConfigs := []*InputConfig{
		{
			EntryNumber:      0,
			Type:             "Github AppImage",
			GithubProjectUrl: "https://github.com/janhq/jan/",
			Category:         "app-misc",
			EbuildName:       "jan-appimage.ebuild",
			Description:      "Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)",
			Homepage:         "https://jan.ai/",
			Workarounds: map[string]string{
				"Test Workaround":            "",
				"Test Workaround with value": "Values",
			},
			GithubOwner: "janhq",
			GithubRepo:  "jan",
			License:     "unknown",
			Programs: map[string]*Program{
				"": {
					ProgramName:       "",
					DesktopFile:       "jan.desktop",
					InstalledFilename: "jan",
					ArchiveFilename:   map[string]string{},
					ReleasesFilename: map[string]string{
						"amd64": "jan-linux-x86_64-${VERSION}.AppImage",
						"arm64": "jan-linux-arm64-${VERSION}.AppImage",
					},
				},
			},
		},
		{
			Type:             "Github AppImage",
			GithubProjectUrl: "https://github.com/anotherorg/anotherrepo/",
			Category:         "app-misc",
			EbuildName:       "anotherrepo-appimage.ebuild",
			GithubOwner:      "anotherorg",
			Workarounds:      map[string]string{},
			GithubRepo:       "anotherrepo",
			License:          "unknown",
			Programs: map[string]*Program{
				"": {
					ProgramName:       "",
					Icons:             []string{"hicolor-apps", "root"},
					InstalledFilename: "anotherapp",
					ArchiveFilename:   map[string]string{},
					ReleasesFilename: map[string]string{
						"amd64": "anotherrepo-${VERSION}.AppImage",
					},
				},
			},
		},
		{
			Type:             "Github AppImage",
			GithubProjectUrl: "https://github.com/probonopd/go-appimage",
			Category:         "app-misc",
			Description:      "Go implementation of AppImage tools",
			EbuildName:       "go-appimage-appimage.ebuild",
			Workarounds:      map[string]string{},
			GithubOwner:      "probonopd",
			GithubRepo:       "go-appimage",
			License:          "MIT",
			Programs: map[string]*Program{
				"": {
					ProgramName:       "",
					InstalledFilename: "appimagetool.AppImage",
					ArchiveFilename:   map[string]string{},
					ReleasesFilename: map[string]string{
						"amd64": "appimagetool-838-x86_64.AppImage",
					},
				},
				"appimaged": {
					ProgramName:       "appimaged",
					ArchiveFilename:   map[string]string{},
					InstalledFilename: "appimaged.AppImage",
					ReleasesFilename: map[string]string{
						"amd64": "appimaged-838-x86_64.AppImage",
					},
				},
				"mkappimage": {
					ProgramName:       "mkappimage",
					InstalledFilename: "mkappimage.AppImage",
					ArchiveFilename:   map[string]string{},
					ReleasesFilename: map[string]string{
						"amd64": "mkappimage-838-x86_64.AppImage",
					},
				},
			},
		},
	}

	// Assertion loop remains the same as before
	for i, expected := range expectedConfigs {
		t.Run(fmt.Sprintf("config[%d]", i), func(t *testing.T) {
			if diff := cmp.Diff(configs[i], expected); diff != "" {
				t.Errorf("unexpected config[%d]:\n%s", i, diff)
			}
		})
	}
}

func TestConfigString(t *testing.T) {
	config := &InputConfig{
		EntryNumber:      0,
		Type:             "Github AppImage",
		GithubProjectUrl: "https://github.com/janhq/jan/",
		Category:         "app-misc",
		EbuildName:       "jan-appimage.ebuild",
		Description:      "Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)",
		Homepage:         "https://jan.ai/",
		Workarounds: map[string]string{
			"Test Workaround":            "",
			"Test Workaround with value": "Values",
		},
		Programs: map[string]*Program{
			"": {
				ProgramName:       "",
				DesktopFile:       "jan.desktop",
				ArchiveFilename:   map[string]string{},
				InstalledFilename: "jan",
				ReleasesFilename: map[string]string{
					"amd64": "anotherrepo-${VERSION}.AppImage",
				},
				Icons: []string{"hicolor-apps", "root"},
			},
		},
	}

	expected := `Type Github AppImage
GithubProjectUrl https://github.com/janhq/jan/
Category app-misc
EbuildName jan-appimage.ebuild
Description Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)
Homepage https://jan.ai/
Workaround Test Workaround
Workaround Test Workaround with value => Values
DesktopFile jan.desktop
Icons hicolor-apps root
InstalledFilename jan
ReleasesFilename amd64=>anotherrepo-${VERSION}.AppImage
`

	result := config.String()
	if diff := cmp.Diff(result, expected); diff != "" {
		t.Errorf("InputConfig.String() = \n%s", diff)
	}
}
