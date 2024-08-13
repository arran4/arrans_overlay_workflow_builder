package arrans_overlay_workflow_builder

import (
	"bytes"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"testing"
)

const testConfigData = `
# Example config
Type Github AppImage Release
GithubProjectUrl https://github.com/janhq/jan/
DesktopFile jan
Category app-misc
EbuildName jan-appimage
Description Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)
Workaround Test Workaround
Workaround Test Workaround with value => Values
Homepage https://jan.ai/
Dependencies dev-libs/libappindicator
Binary amd64=>jan-linux-x86_64-${VERSION}.AppImage > jan
Binary arm64=>jan-linux-arm64-${VERSION}.AppImage > jan

Type Github AppImage Release
GithubProjectUrl https://github.com/anotherorg/anotherrepo/
Icons hicolor-apps root
Binary amd64=>anotherrepo-${VERSION}.AppImage > anotherapp

Type Github AppImage Release
GithubProjectUrl https://github.com/probonopd/go-appimage
EbuildName go-appimage-appimage
Description  Go implementation of AppImage tools
License MIT
Binary amd64=>appimagetool-838-x86_64.AppImage > appimagetool.AppImage 
ProgramName appimaged
Binary amd64=> appimaged-838-x86_64.AppImage > appimaged.AppImage 
ProgramName mkappimage
Binary amd64=>mkappimage-838-x86_64.AppImage > mkappimage.AppImage

Type Github Binary Release
GithubProjectUrl https://github.com/goreleaser/goreleaser
Category dev-go
EbuildName goreleaser-bin
Description Deliver Go binaries as fast and easily as possible
Homepage https://goreleaser.com
License MIT License
ShellCompletionScript amd64:bash=>goreleaser_Linux_x86_64.tar.gz > completion/goreleaser.bash > goreleaser.bash
ShellCompletionScript amd64:fish=>goreleaser_Linux_x86_64.tar.gz > completion/goreleaser.fish > goreleaser.fish
ShellCompletionScript arm:bash=>goreleaser_Linux_armv7.tar.gz > completion/goreleaser.bash > goreleaser.bash
ShellCompletionScript arm:fish=>goreleaser_Linux_armv7.tar.gz > completion/goreleaser.fish > goreleaser.fish
Binary amd64=>goreleaser_Linux_x86_64.tar.gz > goreleaser > goreleaser
Binary arm=>goreleaser_Linux_armv7.tar.gz > goreleaser > goreleaser
Binary arm64=>goreleaser_Linux_arm64.tar.gz > goreleaser > goreleaser
Binary ppc64=>goreleaser_Linux_ppc64.tar.gz > goreleaser > goreleaser
Binary x86=>goreleaser_Linux_i386.tar.gz > goreleaser > goreleaser

`

func TestParseConfigFile(t *testing.T) {
	configs, err := ParseInputConfigReader(bytes.NewReader([]byte(testConfigData)))
	if err != nil {
		t.Fatalf("error parsing config file: %v", err)
	}
	entryCount := 4
	if len(configs) != entryCount {
		t.Fatalf("expected %d config entries, got %d", entryCount, len(configs))
	}

	expectedConfigs := []*InputConfig{
		{
			EntryNumber:      0,
			Type:             "Github AppImage Release",
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
					ProgramName:  "",
					DesktopFile:  "jan.desktop",
					Icons:        []string{},
					Dependencies: []string{"dev-libs/libappindicator"},
					Binary: map[string][]string{
						"amd64": {"jan-linux-x86_64-${VERSION}.AppImage", "jan"},
						"arm64": {"jan-linux-arm64-${VERSION}.AppImage", "jan"},
					},
				},
			},
		},
		{
			Type:             "Github AppImage Release",
			GithubProjectUrl: "https://github.com/anotherorg/anotherrepo/",
			Category:         "app-misc",
			EbuildName:       "anotherrepo-appimage.ebuild",
			GithubOwner:      "anotherorg",
			Workarounds:      map[string]string{},
			GithubRepo:       "anotherrepo",
			License:          "unknown",
			Programs: map[string]*Program{
				"": {
					ProgramName:  "",
					Icons:        []string{"hicolor-apps", "root"},
					Dependencies: []string{},
					Binary: map[string][]string{
						"amd64": {"anotherrepo-${VERSION}.AppImage", "anotherapp"},
					},
				},
			},
		},
		{
			Type:             "Github AppImage Release",
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
					ProgramName:  "",
					Icons:        []string{},
					Dependencies: []string{},
					Binary: map[string][]string{
						"amd64": {"appimagetool-838-x86_64.AppImage", "appimagetool.AppImage"},
					},
				},
				"appimaged": {
					ProgramName:  "appimaged",
					Icons:        []string{},
					Dependencies: []string{},
					Binary: map[string][]string{
						"amd64": {"appimaged-838-x86_64.AppImage", "appimaged.AppImage"},
					},
				},
				"mkappimage": {
					ProgramName:  "mkappimage",
					Icons:        []string{},
					Dependencies: []string{},
					Binary: map[string][]string{
						"amd64": {"mkappimage-838-x86_64.AppImage", "mkappimage.AppImage"},
					},
				},
			},
		},
		&InputConfig{
			EntryNumber:      0,
			Type:             "Github Binary Release",
			GithubProjectUrl: "https://github.com/goreleaser/goreleaser",
			Category:         "dev-go",
			EbuildName:       "goreleaser-bin.ebuild",
			Description:      "Deliver Go binaries as fast and easily as possible",
			Homepage:         "https://goreleaser.com",
			GithubRepo:       "goreleaser",
			GithubOwner:      "goreleaser",
			License:          "MIT License",
			Workarounds:      map[string]string{},
			Programs: map[string]*Program{
				"": {
					Binary: map[string][]string{
						"amd64": {"goreleaser_Linux_x86_64.tar.gz", "goreleaser", "goreleaser"},
						"arm":   {"goreleaser_Linux_armv7.tar.gz", "goreleaser", "goreleaser"},
						"arm64": {"goreleaser_Linux_arm64.tar.gz", "goreleaser", "goreleaser"},
						"ppc64": {"goreleaser_Linux_ppc64.tar.gz", "goreleaser", "goreleaser"},
						"x86":   {"goreleaser_Linux_i386.tar.gz", "goreleaser", "goreleaser"},
					},
					ShellCompletionScripts: map[string]map[string][]string{
						"amd64": {
							"bash": {"goreleaser_Linux_x86_64.tar.gz", "completion/goreleaser.bash", "goreleaser.bash"},
							"fish": {"goreleaser_Linux_x86_64.tar.gz", "completion/goreleaser.fish", "goreleaser.fish"},
						},
						"arm": {
							"bash": {"goreleaser_Linux_armv7.tar.gz", "completion/goreleaser.bash", "goreleaser.bash"},
							"fish": {"goreleaser_Linux_armv7.tar.gz", "completion/goreleaser.fish", "goreleaser.fish"},
						},
					},
					Documents:    map[string][]string{},
					ManualPage:   map[string][]string{},
					Dependencies: []string{},
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
	for _, test := range []struct {
		name   string
		config *InputConfig
		want   string
	}{
		{
			name: "jan-appimage",
			config: &InputConfig{
				EntryNumber:      0,
				Type:             "Github AppImage Release",
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
						ProgramName:  "",
						DesktopFile:  "jan.desktop",
						Dependencies: []string{"dev-libs/libappindicator"},
						Binary: map[string][]string{
							"amd64": {"anotherrepo-${VERSION}.AppImage", "jan"},
						},
						Icons: []string{"hicolor-apps", "root"},
					},
				},
			},
			want: `Type Github AppImage Release
GithubProjectUrl https://github.com/janhq/jan/
Category app-misc
EbuildName jan-appimage.ebuild
Description Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)
Homepage https://jan.ai/
Workaround Test Workaround
Workaround Test Workaround with value => Values
DesktopFile jan.desktop
Icons hicolor-apps root
Dependencies dev-libs/libappindicator
Binary amd64=>anotherrepo-${VERSION}.AppImage > jan
`,
		},
		{
			name: "goreleaser-bin",
			config: &InputConfig{
				EntryNumber:      0,
				Type:             "Github Binary Release",
				GithubProjectUrl: "https://github.com/goreleaser/goreleaser",
				Category:         "dev-go",
				EbuildName:       "goreleaser-bin",
				Description:      "Deliver Go binaries as fast and easily as possible",
				Homepage:         "https://goreleaser.com",
				License:          "MIT License",
				Programs: map[string]*Program{
					"": {
						Binary: map[string][]string{
							"amd64": {"goreleaser_Linux_x86_64.tar.gz", "goreleaser", "goreleaser"},
							"arm":   {"goreleaser_Linux_armv7.tar.gz", "goreleaser", "goreleaser"},
							"arm64": {"goreleaser_Linux_arm64.tar.gz", "goreleaser", "goreleaser"},
							"ppc64": {"goreleaser_Linux_ppc64.tar.gz", "goreleaser", "goreleaser"},
							"x86":   {"goreleaser_Linux_i386.tar.gz", "goreleaser", "goreleaser"},
						},
						ShellCompletionScripts: map[string]map[string][]string{
							"amd64": {
								"bash": {"goreleaser_Linux_x86_64.tar.gz", "completion/goreleaser.bash", "goreleaser.bash"},
								"fish": {"goreleaser_Linux_x86_64.tar.gz", "completion/goreleaser.fish", "goreleaser.fish"},
							},
							"arm": {
								"bash": {"goreleaser_Linux_armv7.tar.gz", "completion/goreleaser.bash", "goreleaser.bash"},
								"fish": {"goreleaser_Linux_armv7.tar.gz", "completion/goreleaser.fish", "goreleaser.fish"},
							},
						},
					},
				},
			},
			want: `Type Github Binary Release
GithubProjectUrl https://github.com/goreleaser/goreleaser
Category dev-go
EbuildName goreleaser-bin
Description Deliver Go binaries as fast and easily as possible
Homepage https://goreleaser.com
License MIT License
ShellCompletionScript amd64:bash=>goreleaser_Linux_x86_64.tar.gz > completion/goreleaser.bash > goreleaser.bash
ShellCompletionScript amd64:fish=>goreleaser_Linux_x86_64.tar.gz > completion/goreleaser.fish > goreleaser.fish
ShellCompletionScript arm:bash=>goreleaser_Linux_armv7.tar.gz > completion/goreleaser.bash > goreleaser.bash
ShellCompletionScript arm:fish=>goreleaser_Linux_armv7.tar.gz > completion/goreleaser.fish > goreleaser.fish
Binary amd64=>goreleaser_Linux_x86_64.tar.gz > goreleaser > goreleaser
Binary arm=>goreleaser_Linux_armv7.tar.gz > goreleaser > goreleaser
Binary arm64=>goreleaser_Linux_arm64.tar.gz > goreleaser > goreleaser
Binary ppc64=>goreleaser_Linux_ppc64.tar.gz > goreleaser > goreleaser
Binary x86=>goreleaser_Linux_i386.tar.gz > goreleaser > goreleaser
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := test.config.String()
			if diff := cmp.Diff(result, test.want); diff != "" {
				t.Errorf("InputConfig.String() = \n%s", diff)
			}
		})
	}
}
