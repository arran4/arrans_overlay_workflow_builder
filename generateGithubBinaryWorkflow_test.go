package arrans_overlay_workflow_builder

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/arran4/g2"
)

func TestGenerateGithubBinaryTemplateData_GetMustHaveUseFlags_and_GetMustHaveUseFlags(t *testing.T) {
	tests := []struct {
		name                      string
		ggbtd                     *GenerateGithubBinaryTemplateData
		programName               string
		kw                        string
		wantGetMustHaveUseFlags   []string
		wantGetMustntHaveUseFlags []string
	}{
		{
			name: "Test 1",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/arran4/anytype-to-linkwarden
EbuildName anytype-to-linkwarden-bin
Description TODO
License unknown
ProgramName anytype-to-linkwarden
Document amd64=>anytype-to-linkwarden_${VERSION}_linux_386.tar.gz > readme.md > readme.md
Document amd64=>anytype-to-linkwarden_${VERSION}_linux_amd64.tar.gz > readme.md > readme.md
Document arm64=>anytype-to-linkwarden_${VERSION}_linux_arm64.tar.gz > readme.md > readme.md
Binary amd64=>anytype-to-linkwarden_${VERSION}_linux_amd64.tar.gz > anytype-to-linkwarden > anytype-to-linkwarden
Binary arm64=>anytype-to-linkwarden_${VERSION}_linux_arm64.tar.gz > anytype-to-linkwarden > anytype-to-linkwarden
`),
			programName:               "anytype-to-linkwarden",
			kw:                        "amd64",
			wantGetMustHaveUseFlags:   []string{"amd64"},
			wantGetMustntHaveUseFlags: []string{},
		},
		{
			name: "Discovered issue 1: Editorconfig",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/arran4/editorconfig-guesser
EbuildName editorconfig-guesser-bin
Category dev-util
Description Generates reasonable .editorconfig files for source files.
License MIT License
Workaround Programs as Alternatives => arm:ecguess
ProgramName ecguess
Document amd64=>cards_${VERSION}_linux_amd64.tar.gz > LICENSE > LICENSE
Document amd64=>cards_${VERSION}_linux_amd64.tar.gz > readme.md > readme.md
Document arm=>cards_${VERSION}_linux_armv6.tar.gz > LICENSE > LICENSE
Document arm=>cards_${VERSION}_linux_armv6.tar.gz > readme.md > readme.md
Document arm=>cards_${VERSION}_linux_armv7.tar.gz > LICENSE > LICENSE
Document arm=>cards_${VERSION}_linux_armv7.tar.gz > readme.md > readme.md
Document arm64=>cards_${VERSION}_linux_arm64.tar.gz > LICENSE > LICENSE
Document arm64=>cards_${VERSION}_linux_arm64.tar.gz > readme.md > readme.md
Binary amd64=>cards_${VERSION}_linux_amd64.tar.gz > ecguess > ecguess
Binary arm=>cards_${VERSION}_linux_armv7.tar.gz > ecguess > ecguess
Binary arm64=>cards_${VERSION}_linux_arm64.tar.gz > ecguess > ecguess
`),
			programName:               "editorconfig-guesser",
			kw:                        "arm",
			wantGetMustHaveUseFlags:   []string{},
			wantGetMustntHaveUseFlags: []string{},
		},
		{
			name: "Discovered issue 2: Hugo - amd64 - normal",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/gohugoio/hugo
EbuildName hugo-bin
Category www-apps
Description The world’s fastest framework for building websites.
Homepage https://gohugo.io
License Apache License 2.0
Workaround Programs as Alternatives => amd64:extended arm64:extended
Binary amd64=>hugo_${VERSION}_linux-amd64.tar.gz > hugo > hugo
Binary arm=>hugo_${VERSION}_linux-arm.tar.gz > hugo > hugo
Binary arm64=>hugo_${VERSION}_linux-arm64.tar.gz > hugo > hugo
ProgramName extended
Dependencies sys-libs/glibc sys-devel/gcc sys-libs/glibc sys-devel/gcc sys-libs/glibc sys-devel/gcc
Binary amd64=>hugo_extended_${VERSION}_Linux-64bit.tar.gz > hugo > hugo
Binary arm64=>hugo_extended_${VERSION}_linux-arm64.tar.gz > hugo > hugo
`),
			programName:               "hugo",
			kw:                        "amd64",
			wantGetMustHaveUseFlags:   []string{},
			wantGetMustntHaveUseFlags: []string{},
		},
		{
			name: "Discovered issue 3: Hugo - amd64 - extended",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/gohugoio/hugo
EbuildName hugo-bin
Category www-apps
Description The world’s fastest framework for building websites.
Homepage https://gohugo.io
License Apache License 2.0
Workaround Programs as Alternatives => amd64:extended arm64:extended
Binary amd64=>hugo_${VERSION}_linux-amd64.tar.gz > hugo > hugo
Binary arm=>hugo_${VERSION}_linux-arm.tar.gz > hugo > hugo
Binary arm64=>hugo_${VERSION}_linux-arm64.tar.gz > hugo > hugo
ProgramName extended
Dependencies sys-libs/glibc sys-devel/gcc sys-libs/glibc sys-devel/gcc sys-libs/glibc sys-devel/gcc
Binary amd64=>hugo_extended_${VERSION}_Linux-64bit.tar.gz > hugo > hugo
Binary arm64=>hugo_extended_${VERSION}_linux-arm64.tar.gz > hugo > hugo
`),
			programName:               "extended",
			kw:                        "amd64",
			wantGetMustHaveUseFlags:   []string{"amd64", "extended"},
			wantGetMustntHaveUseFlags: []string{},
		},
		{
			name: "Discovered issue 4: Hugo - arm64 - normal",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/gohugoio/hugo
EbuildName hugo-bin
Category www-apps
Description The world’s fastest framework for building websites.
Homepage https://gohugo.io
License Apache License 2.0
Workaround Programs as Alternatives => amd64:extended arm64:extended
Binary amd64=>hugo_${VERSION}_linux-amd64.tar.gz > hugo > hugo
Binary arm=>hugo_${VERSION}_linux-arm.tar.gz > hugo > hugo
Binary arm64=>hugo_${VERSION}_linux-arm64.tar.gz > hugo > hugo
ProgramName extended
Dependencies sys-libs/glibc sys-devel/gcc sys-libs/glibc sys-devel/gcc sys-libs/glibc sys-devel/gcc
Binary amd64=>hugo_extended_${VERSION}_Linux-64bit.tar.gz > hugo > hugo
Binary arm64=>hugo_extended_${VERSION}_linux-arm64.tar.gz > hugo > hugo
`),
			programName:               "hugo",
			kw:                        "arm64",
			wantGetMustHaveUseFlags:   []string{},
			wantGetMustntHaveUseFlags: []string{},
		},
		{
			name: "Discovered issue 4: Hugo - arm64 - extended",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/gohugoio/hugo
EbuildName hugo-bin
Category www-apps
Description The world’s fastest framework for building websites.
Homepage https://gohugo.io
License Apache License 2.0
Workaround Programs as Alternatives => amd64:extended arm64:extended
Binary amd64=>hugo_${VERSION}_linux-amd64.tar.gz > hugo > hugo
Binary arm=>hugo_${VERSION}_linux-arm.tar.gz > hugo > hugo
Binary arm64=>hugo_${VERSION}_linux-arm64.tar.gz > hugo > hugo
ProgramName extended
Dependencies sys-libs/glibc sys-devel/gcc sys-libs/glibc sys-devel/gcc sys-libs/glibc sys-devel/gcc
Binary amd64=>hugo_extended_${VERSION}_Linux-64bit.tar.gz > hugo > hugo
Binary arm64=>hugo_extended_${VERSION}_linux-arm64.tar.gz > hugo > hugo
`),
			programName:               "extended",
			kw:                        "arm64",
			wantGetMustHaveUseFlags:   []string{"arm64", "extended"},
			wantGetMustntHaveUseFlags: []string{},
		},
		{
			name: "Discovered issue 4: Chezmoi - arm64 - not android",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/twpayne/chezmoi
EbuildName chezmoi-bin
Category app-admin
Description Manage your dotfiles across multiple diverse machines, securely.
Homepage https://www.chezmoi.io/
License MIT License
Workaround Programs as Alternatives => amd64:glibc amd64:loong64 arm64:android ppc64:le
ProgramName android
Binary arm64=>chezmoi_${VERSION}_android_arm64.tar.gz > chezmoi > chezmoi
ProgramName chezmoi
Dependencies sys-libs/glibc
Binary amd64=>chezmoi_${VERSION}_linux-musl_amd64.tar.gz > chezmoi > chezmoi
Binary arm=>chezmoi_${VERSION}_linux_arm.tar.gz > chezmoi > chezmoi
Binary arm64=>chezmoi_${VERSION}_linux_arm64.tar.gz > chezmoi > chezmoi
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64.tar.gz > chezmoi > chezmoi
Binary riscv=>chezmoi_${VERSION}_linux_riscv64.tar.gz > chezmoi > chezmoi
Binary s390=>chezmoi_${VERSION}_linux_s390x.tar.gz > chezmoi > chezmoi
Binary x86=>chezmoi_${VERSION}_linux_i386.tar.gz > chezmoi > chezmoi
ProgramName glibc
Dependencies sys-libs/glibc
Binary amd64=>chezmoi_${VERSION}_linux-glibc_amd64.tar.gz > chezmoi > chezmoi
ProgramName le
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64le.tar.gz > chezmoi > chezmoi
ProgramName loong64
Binary amd64=>chezmoi_${VERSION}_linux_loong64.tar.gz > chezmoi > chezmoi
`),
			programName:               "chezmoi",
			kw:                        "arm64",
			wantGetMustHaveUseFlags:   []string{"arm64"},
			wantGetMustntHaveUseFlags: []string{"android"},
		},
		{
			name: "Discovered issue 4: Chezmoi - arm64 - android",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/twpayne/chezmoi
EbuildName chezmoi-bin
Category app-admin
Description Manage your dotfiles across multiple diverse machines, securely.
Homepage https://www.chezmoi.io/
License MIT License
Workaround Programs as Alternatives => amd64:glibc amd64:loong64 arm64:android ppc64:le
ProgramName android
Binary arm64=>chezmoi_${VERSION}_android_arm64.tar.gz > chezmoi > chezmoi
ProgramName chezmoi
Dependencies sys-libs/glibc
Binary amd64=>chezmoi_${VERSION}_linux-musl_amd64.tar.gz > chezmoi > chezmoi
Binary arm=>chezmoi_${VERSION}_linux_arm.tar.gz > chezmoi > chezmoi
Binary arm64=>chezmoi_${VERSION}_linux_arm64.tar.gz > chezmoi > chezmoi
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64.tar.gz > chezmoi > chezmoi
Binary riscv=>chezmoi_${VERSION}_linux_riscv64.tar.gz > chezmoi > chezmoi
Binary s390=>chezmoi_${VERSION}_linux_s390x.tar.gz > chezmoi > chezmoi
Binary x86=>chezmoi_${VERSION}_linux_i386.tar.gz > chezmoi > chezmoi
ProgramName glibc
Dependencies sys-libs/glibc
Binary amd64=>chezmoi_${VERSION}_linux-glibc_amd64.tar.gz > chezmoi > chezmoi
ProgramName le
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64le.tar.gz > chezmoi > chezmoi
ProgramName loong64
Binary amd64=>chezmoi_${VERSION}_linux_loong64.tar.gz > chezmoi > chezmoi
`),
			programName:               "android",
			kw:                        "arm64",
			wantGetMustHaveUseFlags:   []string{"arm64", "android"},
			wantGetMustntHaveUseFlags: []string{},
		},
		{
			name: "Discovered issue 4: Chezmoi - ppc64 - not le",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/twpayne/chezmoi
EbuildName chezmoi-bin
Category app-admin
Description Manage your dotfiles across multiple diverse machines, securely.
Homepage https://www.chezmoi.io/
License MIT License
Workaround Programs as Alternatives => amd64:glibc amd64:loong64 arm64:android ppc64:le
ProgramName android
Binary arm64=>chezmoi_${VERSION}_android_arm64.tar.gz > chezmoi > chezmoi
ProgramName chezmoi
Dependencies sys-libs/glibc
Binary amd64=>chezmoi_${VERSION}_linux-musl_amd64.tar.gz > chezmoi > chezmoi
Binary arm=>chezmoi_${VERSION}_linux_arm.tar.gz > chezmoi > chezmoi
Binary arm64=>chezmoi_${VERSION}_linux_arm64.tar.gz > chezmoi > chezmoi
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64.tar.gz > chezmoi > chezmoi
Binary riscv=>chezmoi_${VERSION}_linux_riscv64.tar.gz > chezmoi > chezmoi
Binary s390=>chezmoi_${VERSION}_linux_s390x.tar.gz > chezmoi > chezmoi
Binary x86=>chezmoi_${VERSION}_linux_i386.tar.gz > chezmoi > chezmoi
ProgramName glibc
Dependencies sys-libs/glibc
Binary amd64=>chezmoi_${VERSION}_linux-glibc_amd64.tar.gz > chezmoi > chezmoi
ProgramName le
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64le.tar.gz > chezmoi > chezmoi
ProgramName loong64
Binary amd64=>chezmoi_${VERSION}_linux_loong64.tar.gz > chezmoi > chezmoi
`),
			programName:               "chezmoi",
			kw:                        "ppc64",
			wantGetMustHaveUseFlags:   []string{"ppc64"},
			wantGetMustntHaveUseFlags: []string{"le"},
		},
		{
			name: "Discovered issue 4: Chezmoi - ppc64 - le",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/twpayne/chezmoi
EbuildName chezmoi-bin
Category app-admin
Description Manage your dotfiles across multiple diverse machines, securely.
Homepage https://www.chezmoi.io/
License MIT License
Workaround Programs as Alternatives => amd64:glibc amd64:loong64 arm64:android ppc64:le
ProgramName android
Binary arm64=>chezmoi_${VERSION}_android_arm64.tar.gz > chezmoi > chezmoi
ProgramName chezmoi
Dependencies sys-libs/glibc
Binary amd64=>chezmoi_${VERSION}_linux-musl_amd64.tar.gz > chezmoi > chezmoi
Binary arm=>chezmoi_${VERSION}_linux_arm.tar.gz > chezmoi > chezmoi
Binary arm64=>chezmoi_${VERSION}_linux_arm64.tar.gz > chezmoi > chezmoi
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64.tar.gz > chezmoi > chezmoi
Binary riscv=>chezmoi_${VERSION}_linux_riscv64.tar.gz > chezmoi > chezmoi
Binary s390=>chezmoi_${VERSION}_linux_s390x.tar.gz > chezmoi > chezmoi
Binary x86=>chezmoi_${VERSION}_linux_i386.tar.gz > chezmoi > chezmoi
ProgramName glibc
Dependencies sys-libs/glibc
Binary amd64=>chezmoi_${VERSION}_linux-glibc_amd64.tar.gz > chezmoi > chezmoi
ProgramName le
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64le.tar.gz > chezmoi > chezmoi
ProgramName loong64
Binary amd64=>chezmoi_${VERSION}_linux_loong64.tar.gz > chezmoi > chezmoi
`),
			programName:               "le",
			kw:                        "ppc64",
			wantGetMustHaveUseFlags:   []string{"ppc64", "le"},
			wantGetMustntHaveUseFlags: []string{},
		},
	}
	// TODO the problem with these tests is that `inferUseFlags` and later stages are too smart. -- When and if I get around to making this tree
	// structured it should resolve that.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ggbtd.GetMustHaveUseFlags(tt.programName, tt.kw); !reflect.DeepEqual(got, tt.wantGetMustHaveUseFlags) {
				t.Errorf("GetMustHaveUseFlags() = %v, want %v", got, tt.wantGetMustHaveUseFlags)
			}
			if got := tt.ggbtd.GetMustntHaveUseFlags(tt.programName, tt.kw); !reflect.DeepEqual(got, tt.wantGetMustntHaveUseFlags) {
				t.Errorf("GetMustntHaveUseFlags() = %v, want %v", got, tt.wantGetMustntHaveUseFlags)
			}
		})
	}
}

func NewGenerateGithubBinaryTemplateDataFromString(s string) *GenerateGithubBinaryTemplateData {
	ics, err := ParseInputConfigReader(bytes.NewReader([]byte(s)))
	if err != nil {
		log.Panicf("ParseInputConfigReader: %s", err)
	}
	if len(ics) == 0 {
		log.Panicf("no configs in ics")
	}
	ic := ics[0]
	base := &GenerateGithubWorkflowBase{
		Version:     "1",
		Now:         time.Now(),
		ConfigFile:  "test.config",
		InputConfig: ic,
	}
	switch ic.Type {
	case "Github Binary Release":
		return &GenerateGithubBinaryTemplateData{
			GenerateGithubWorkflowBase: base,
		}
	}
	log.Panicf("unkown type %s", ic.Type)
	return nil
}

func TestGenerateGithubBinaryTemplateData_IUseFlags_Determinism(t *testing.T) {
	t.Run("distinct raw alternative names normalizing to same flag", func(t *testing.T) {
		configStr := `Type Github Binary Release
GithubProjectUrl https://github.com/example/tool
EbuildName tool-bin
Category app-misc
Description Test tool
License MIT
Workaround Programs as Alternatives => amd64:foo_bar arm64:foo-bar
ProgramName tool
Binary amd64=>tool_${VERSION}_linux_amd64.tar.gz > tool > tool
Binary arm64=>tool_${VERSION}_linux_arm64.tar.gz > tool > tool
ProgramName foo_bar
Binary amd64=>tool_${VERSION}_foo_bar.tar.gz > tool > tool
ProgramName foo-bar
Binary arm64=>tool_${VERSION}_foo-bar.tar.gz > tool > tool
`
		for i := 0; i < 100; i++ {
			ggbtd := NewGenerateGithubBinaryTemplateDataFromString(configStr)
			flags := ggbtd.IUseFlags()

			// "foo-bar" (sorted before "foo_bar") normalizes to "foo_bar"
			expectedFlags := []g2.Flag{
				{Name: "foo_bar", Text: "Install foo-bar binary"},
			}
			if !reflect.DeepEqual(flags, expectedFlags) {
				t.Fatalf("iteration %d: IUseFlags() = %+v, want %+v", i, flags, expectedFlags)
			}

			g2Args := ggbtd.G2MetadataArgs()
			expectedArg := `--use-add "foo_bar:Install foo-bar binary"`
			if !strings.Contains(g2Args, expectedArg) {
				t.Fatalf("iteration %d: G2MetadataArgs() = %q, want containing %q", i, g2Args, expectedArg)
			}

			if strings.Count(g2Args, "--use-add") != 1 {
				t.Fatalf("iteration %d: G2MetadataArgs() contains duplicate --use-add: %q", i, g2Args)
			}

			meta, err := ggbtd.Metadata()
			if err != nil {
				t.Fatalf("iteration %d: Metadata() error = %v", i, err)
			}
			expectedXML := `<flag name="foo_bar">Install foo-bar binary</flag>`
			if !strings.Contains(meta, expectedXML) {
				t.Fatalf("iteration %d: Metadata() = %q, want containing %q", i, meta, expectedXML)
			}
			if strings.Count(meta, "<flag") != 1 {
				t.Fatalf("iteration %d: Metadata() contains duplicate flags: %q", i, meta)
			}
		}
	})

	t.Run("overlap of special flags man doc shell with explicit and extracted flags", func(t *testing.T) {
		configStr := `Type Github Binary Release
GithubProjectUrl https://github.com/example/tool
EbuildName tool-bin
Category app-misc
Description Test tool
License MIT
IUse doc man bash custom_flag
ProgramName tool
Binary amd64=>tool_${VERSION}_linux_amd64.tar.gz > tool > tool
Binary arm64[+extracted_flag]=>tool_${VERSION}_linux_arm64.tar.gz > tool > tool
Document amd64=>tool_${VERSION}_linux_amd64.tar.gz > README.md > README.md
ManualPage amd64=>tool_${VERSION}_linux_amd64.tar.gz > tool.1 > tool.1
ShellCompletionScript amd64:bash=>tool_${VERSION}_linux_amd64.tar.gz > completions/bash > tool
`
		for i := 0; i < 100; i++ {
			ggbtd := NewGenerateGithubBinaryTemplateDataFromString(configStr)
			flags := ggbtd.IUseFlags()

			expectedFlags := []g2.Flag{
				{Name: "bash", Text: "Install bash completion"},
				{Name: "custom_flag", Text: "Enable custom_flag"},
				{Name: "doc", Text: "Install documentation"},
				{Name: "extracted_flag", Text: "Enable extracted_flag"},
				{Name: "man", Text: "Install manual pages"},
			}
			if !reflect.DeepEqual(flags, expectedFlags) {
				t.Fatalf("iteration %d: IUseFlags() = %+v, want %+v", i, flags, expectedFlags)
			}

			g2Args := ggbtd.G2MetadataArgs()
			for _, ef := range expectedFlags {
				arg := fmt.Sprintf(`--use-add "%s:%s"`, ef.Name, ef.Text)
				if !strings.Contains(g2Args, arg) {
					t.Fatalf("iteration %d: G2MetadataArgs() missing %q: %q", i, arg, g2Args)
				}
			}
			if strings.Count(g2Args, "--use-add") != len(expectedFlags) {
				t.Fatalf("iteration %d: G2MetadataArgs() flag count mismatch: %q", i, g2Args)
			}

			meta, err := ggbtd.Metadata()
			if err != nil {
				t.Fatalf("iteration %d: Metadata() error = %v", i, err)
			}
			for _, ef := range expectedFlags {
				flagXML := fmt.Sprintf(`<flag name="%s">%s</flag>`, ef.Name, ef.Text)
				if !strings.Contains(meta, flagXML) {
					t.Fatalf("iteration %d: Metadata() missing %q: %q", i, flagXML, meta)
				}
			}
			if strings.Count(meta, "<flag") != len(expectedFlags) {
				t.Fatalf("iteration %d: Metadata() flag count mismatch: %q", i, meta)
			}
		}
	})

	t.Run("overlap of alternative program with special flag doc and man", func(t *testing.T) {
		configStr := `Type Github Binary Release
GithubProjectUrl https://github.com/example/tool
EbuildName tool-bin
Category app-misc
Description Test tool
License MIT
Workaround Programs as Alternatives => amd64:doc arm64:man
ProgramName tool
Binary amd64=>tool_${VERSION}_linux_amd64.tar.gz > tool > tool
Binary arm64=>tool_${VERSION}_linux_arm64.tar.gz > tool > tool
ProgramName doc
Binary amd64=>tool_${VERSION}_doc.tar.gz > tool > tool
ProgramName man
Binary arm64=>tool_${VERSION}_man.tar.gz > tool > tool
Document amd64=>tool_${VERSION}_linux_amd64.tar.gz > README.md > README.md
ManualPage arm64=>tool_${VERSION}_linux_arm64.tar.gz > tool.1 > tool.1
`
		for i := 0; i < 100; i++ {
			ggbtd := NewGenerateGithubBinaryTemplateDataFromString(configStr)
			flags := ggbtd.IUseFlags()

			// Alternatives take precedence over HasDocuments and HasManualPages
			expectedFlags := []g2.Flag{
				{Name: "doc", Text: "Install doc binary"},
				{Name: "man", Text: "Install man binary"},
			}
			if !reflect.DeepEqual(flags, expectedFlags) {
				t.Fatalf("iteration %d: IUseFlags() = %+v, want %+v", i, flags, expectedFlags)
			}

			g2Args := ggbtd.G2MetadataArgs()
			for _, ef := range expectedFlags {
				arg := fmt.Sprintf(`--use-add "%s:%s"`, ef.Name, ef.Text)
				if !strings.Contains(g2Args, arg) {
					t.Fatalf("iteration %d: G2MetadataArgs() missing %q: %q", i, arg, g2Args)
				}
			}
			if strings.Count(g2Args, "--use-add") != len(expectedFlags) {
				t.Fatalf("iteration %d: G2MetadataArgs() flag count mismatch: %q", i, g2Args)
			}
		}
	})
}
