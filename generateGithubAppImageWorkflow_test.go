package arrans_overlay_workflow_builder

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"slices"
	"sort"
	"testing"
	"time"
)

func TestInstalledFilename(t *testing.T) {
	type Test struct {
		Name              string
		ExpectedFilenames []string
		Input             string
	}
	for _, test := range []Test{
		{
			Name:              "RustDesk test 1",
			ExpectedFilenames: []string{"rustdesk.AppImage"},
			Input:             "Type Github AppImage Release\nGithubProjectUrl https://github.com/rustdesk/rustdesk/\nCategory net-misc\nEbuildName rustdesk-appimage\nDescription An open-source remote desktop application designed for self-hosting, as an alternative to TeamViewer.\nHomepage https://rustdesk.com\nLicense GNU Affero General Public License v3.0\nWorkaround Semantic Version Prerelease Hack 1\nWorkaround Semantic Version Without V\nProgramName rustdesk\nDesktopFile rustdesk.desktop\nIcons hicolor-apps\nDependencies sys-libs/glibc sys-libs/zlib sys-libs/zlib sys-libs/glibc\nBinary amd64=>rustdesk-${TAG}-x86_64.AppImage > rustdesk.AppImage\nBinary arm64=>rustdesk-${TAG}-aarch64.AppImage > rustdesk.AppImage\n",
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			ics, err := ParseInputConfigReader(bytes.NewReader([]byte(test.Input)))
			if err != nil {
				t.Fatal(err)
			}
			if len(ics) != 1 {
				t.Fatalf("len(ics) = %d, want 1", len(ics))
			}
			data := NewTestGithubWorkflow(t, ics[0])
			filenames := slices.Collect(func(yield func(s string) bool) {
				for _, v := range data.GetPrograms() {
					if !yield(v.InstalledFilename()) {
						return
					}
				}
			})
			sort.Strings(filenames)
			if diff := cmp.Diff(test.ExpectedFilenames, filenames); diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func NewTestGithubWorkflow(t *testing.T, ic *InputConfig) interface {
	WorkflowFileName() string
	TemplateFileName() string
	GetPrograms() map[string]*Program
} {
	base := &GenerateGithubWorkflowBase{
		Version:     "1",
		Now:         time.Now(),
		ConfigFile:  "test.config",
		InputConfig: ic,
	}
	var data interface {
		WorkflowFileName() string
		TemplateFileName() string
		GetPrograms() map[string]*Program
	}
	switch ic.Type {
	case "Github AppImage Release":
		data = &GenerateGithubAppImageTemplateData{
			GenerateGithubWorkflowBase: base,
		}
	case "Github Binary Release":
		data = &GenerateGithubBinaryTemplateData{
			GenerateGithubWorkflowBase: base,
		}
	default:
		t.Fatalf("unkown type %s", ic.Type)
	}
	return data
}

func TestExternalResourcesToArchivedResourceNameConsistency(t *testing.T) {
	type Test struct {
		Name  string
		Input string
	}
	for _, test := range []Test{
		{
			Name:  "RustDesk test 1",
			Input: "Type Github AppImage Release\nGithubProjectUrl https://github.com/rustdesk/rustdesk/\nCategory net-misc\nEbuildName rustdesk-appimage\nDescription An open-source remote desktop application designed for self-hosting, as an alternative to TeamViewer.\nHomepage https://rustdesk.com\nLicense GNU Affero General Public License v3.0\nWorkaround Semantic Version Prerelease Hack 1\nWorkaround Semantic Version Without V\nProgramName rustdesk\nDesktopFile rustdesk.desktop\nIcons hicolor-apps\nDependencies sys-libs/glibc sys-libs/zlib sys-libs/zlib sys-libs/glibc\nBinary amd64=>rustdesk-${TAG}-x86_64.AppImage > rustdesk.AppImage\nBinary arm64=>rustdesk-${TAG}-aarch64.AppImage > rustdesk.AppImage\n",
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			ics, err := ParseInputConfigReader(bytes.NewReader([]byte(test.Input)))
			if err != nil {
				t.Fatal(err)
			}
			if len(ics) != 1 {
				t.Fatalf("len(ics) = %d, want 1", len(ics))
			}
			data := NewTestGithubWorkflow(t, ics[0])
			type ERSimple interface {
				ExternalResources() map[string]*ExternalResource
			}
			type ERComplex interface {
				ExternalResources() []*ExternalResourceKeywordExtended
			}
			var releaseFilename []string
			var archivedFilename []string
			switch data := data.(type) {
			case ERSimple:
				releaseFilename = slices.Collect(func(yield func(s string) bool) {
					for _, v := range data.ExternalResources() {
						if !yield(v.ReleaseFilename) {
							return
						}
					}
				})
			case ERComplex:
				releaseFilename = slices.Collect(func(yield func(s string) bool) {
					for _, v := range data.ExternalResources() {
						if !yield(v.ReleaseFilename()) {
							return
						}
					}
				})
			default:
				t.Fatalf("unkown type %s", ics[0].Type)
			}
			archivedFilename = slices.Collect(func(yield func(s string) bool) {
				for _, prog := range data.GetPrograms() {
					for _, bin := range prog.Binary {
						name := bin[0]
						if !yield(name) {
							return
						}
					}
				}
			})
			sort.Strings(releaseFilename)
			sort.Strings(archivedFilename)
			if diff := cmp.Diff(releaseFilename, archivedFilename); diff != "" {
				t.Fatalf("mismatch (-releaseFn +archiveFn):\n%s", diff)
			}
		})
	}
}
