package arrans_overlay_workflow_builder

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"testing"
)

func TestBinaryReleaseFileInfo_CompileMeanings(t *testing.T) {
	tests := []struct {
		name  string
		input []*FilenamePartMeaning
		base  *BinaryReleaseFileInfo
		want  *BinaryReleaseFileInfo
		want1 bool
	}{
		{
			name: "hugo_extended",
			input: []*FilenamePartMeaning{
				{Captured: "hugo", ProjectName: true, CaseInsensitive: true},
			},
			base: &BinaryReleaseFileInfo{
				Filename:        "hugo",
				ExecutableBit:   true,
				ArchivePathname: "hugo",
				Container: &BinaryReleaseFileInfo{
					Containers:       []string{"tar", "gz"},
					Filename:         "hugo_extended_${TAG}_Linux-64bit.tar.gz",
					InstalledName:    "extended",
					OriginalFilename: "hugo_extended_0.131.0_Linux-64bit.tar.gz",
					ProgramName:      "extended",
					Keyword:          "~amd64",
					Tag:              true,
					ProjectName:      true,
					SuffixOnly:       true,
					OS:               "linux",
				},
			},
			want: &BinaryReleaseFileInfo{
				Container: &BinaryReleaseFileInfo{
					Containers:       []string{"tar", "gz"},
					Filename:         "hugo_extended_${TAG}_Linux-64bit.tar.gz",
					InstalledName:    "extended",
					OriginalFilename: "hugo_extended_0.131.0_Linux-64bit.tar.gz",
					ProgramName:      "extended",
					Keyword:          "~amd64",
					Tag:              true,
					ProjectName:      true,
					SuffixOnly:       true,
					OS:               "linux",
				},
				Keyword:          "~amd64",
				OS:               "linux",
				ProgramName:      "extended",
				OriginalFilename: "hugo",
				ArchivePathname:  "hugo",
				InstalledName:    "hugo",
				ExecutableBit:    true,
				Filename:         "hugo",
				ProjectName:      true,
				SuffixOnly:       true,
				Unmatched:        []string{},
				Binary:           true,
			},
			want1: true,
		},
		{
			name: "pagefind_extended",
			input: []*FilenamePartMeaning{
				{Captured: "pagefind", ProjectName: true, CaseInsensitive: true},
				{Separator: true, Captured: "_"},
				{Captured: "extended", Unmatched: true},
			},
			base: &BinaryReleaseFileInfo{
				Filename:        "pagefind_extended",
				ArchivePathname: "pagefind_extended",
				ExecutableBit:   true,
				Container: &BinaryReleaseFileInfo{
					Keyword:          "~arm64",
					OS:               "linux",
					Toolchain:        "musl",
					ProgramName:      "extended",
					OriginalFilename: "pagefind_extended-v1.1.0-aarch64-unknown-linux-musl.tar.gz",
					InstalledName:    "extended",
					Containers:       []string{"tar", "gz"},
					Filename:         "pagefind_extended-${TAG}-aarch64-unknown-linux-musl.tar.gz",
					Tag:              true,
					ProjectName:      true,
					SuffixOnly:       true,
				},
			},
			want: &BinaryReleaseFileInfo{
				Keyword:          "~arm64",
				OS:               "linux",
				Toolchain:        "musl",
				ProgramName:      "extended",
				OriginalFilename: "pagefind_extended",
				ArchivePathname:  "pagefind_extended",
				InstalledName:    "pagefind_extended",
				ExecutableBit:    true,
				Installer:        false,
				AppImage:         false,
				Container: &BinaryReleaseFileInfo{
					Keyword:          "~arm64",
					OS:               "linux",
					Toolchain:        "musl",
					ProgramName:      "extended",
					OriginalFilename: "pagefind_extended-v1.1.0-aarch64-unknown-linux-musl.tar.gz",
					InstalledName:    "extended",
					Containers:       []string{"tar", "gz"},
					Filename:         "pagefind_extended-${TAG}-aarch64-unknown-linux-musl.tar.gz",
					Tag:              true,
					ProjectName:      true,
					SuffixOnly:       true,
				},
				Filename:    "pagefind_extended",
				Binary:      true,
				SuffixOnly:  true,
				ProjectName: true,
				Unmatched:   []string{},
			},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBrfi, gotOk := tt.base.CompileMeanings(tt.input, nil)
			if diff := cmp.Diff(gotBrfi, tt.want, cmpopts.IgnoreUnexported(BinaryReleaseFileInfo{})); diff != "" {
				t.Errorf("CompileMeanings() gotBrfi =\n%v", diff)
			}
			if gotOk != tt.want1 {
				t.Errorf("CompileMeanings() gotOk = %v, want %v", gotOk, tt.want1)
			}
		})
	}
}
