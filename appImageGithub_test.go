package arrans_overlay_workflow_builder

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-github/v62/github"
	"testing"
)

func TestCompileMeanings(t *testing.T) {
	tests := []struct {
		name         string
		input        []*FilenamePartMeaning
		releaseAsset *github.ReleaseAsset
		filename     string
		want         *AppImageFileInfo
		ok           bool
	}{
		{
			name: "jan-linux-x86_64-0.5.1.AppImage",
			input: []*FilenamePartMeaning{
				{ProjectName: true, CaseInsensitive: true, Captured: "jan"},
				{Separator: true, Captured: "-"},
				{OS: "linux", Captured: "linux"},
				{Separator: true, Captured: "-"},
				{Keyword: "~amd64", Captured: "x86_64"},
				{Separator: true, Captured: "-"},
				{Version: true, Captured: "0.5.1"},
				{Separator: true, Captured: "."},
				{AppImage: true, SuffixOnly: true, OS: "linux", Captured: "AppImage"},
			},
			releaseAsset: nil,
			filename:     "jan-linux-x86_64-0.5.1.AppImage",
			want: &AppImageFileInfo{
				Keyword:          "~amd64",
				OS:               "linux",
				Toolchain:        "",
				Container:        "",
				Containers:       nil,
				Filename:         "jan-linux-x86_64-${VERSION}.AppImage",
				OriginalFilename: "jan-linux-x86_64-0.5.1.AppImage",
				AppImage:         true,
				Version:          true,
				ProjectName:      true,
				SuffixOnly:       true,
				CaseInsensitive:  false,
				ReleaseAsset:     nil,
			},
			ok: true,
		},
		{
			name: "appimaged-838-aarch64.AppImage",
			input: []*FilenamePartMeaning{
				{Unmatched: true, Captured: "appimaged-838"},
				{Separator: true, Captured: "-"},
				{Keyword: "~arm64", Captured: "aarch64"},
				{Separator: true, Captured: "."},
				{AppImage: true, SuffixOnly: true, OS: "linux", Captured: "AppImage"},
			},
			releaseAsset: nil,
			filename:     "appimaged-838-aarch64.AppImage",
			want: &AppImageFileInfo{
				Keyword:          "~arm64",
				OS:               "linux",
				Toolchain:        "",
				Container:        "",
				ProgramName:      "appimaged-838",
				Containers:       nil,
				Filename:         "appimaged-838-aarch64.AppImage",
				OriginalFilename: "appimaged-838-aarch64.AppImage",
				AppImage:         true,
				SuffixOnly:       true,
				CaseInsensitive:  false,
				ReleaseAsset:     nil,
			},
			ok: true,
		},
		{
			name: "appimaged-838-aarch64.AppImage.zsync",
			input: []*FilenamePartMeaning{
				{Unmatched: true, Captured: "appimaged-838"},
				{Separator: true, Captured: "-"},
				{Keyword: "~arm64", Captured: "aarch64"},
				{Separator: true, Captured: "."},
				{AppImage: true, SuffixOnly: true, OS: "linux", Captured: "AppImage"},
				{Separator: true, Captured: "."},
				{Unmatched: true, Captured: "zsync", SuffixOnly: true},
			},
			releaseAsset: nil,
			filename:     "appimaged-838-aarch64.AppImage.zsync",
			want: &AppImageFileInfo{
				Keyword:          "~arm64",
				OS:               "linux",
				Toolchain:        "",
				Container:        "",
				ProgramName:      "appimaged-838",
				Containers:       nil,
				Filename:         "appimaged-838-aarch64.AppImage.zsync",
				OriginalFilename: "appimaged-838-aarch64.AppImage.zsync",
				AppImage:         true,
				SuffixOnly:       true,
				CaseInsensitive:  false,
				ReleaseAsset:     nil,
				Unmatched:        []string{"zsync"},
			},
			ok: true,
		},
		{
			name: "appimaged-838-aarch64-asdf.AppImage",
			input: []*FilenamePartMeaning{
				{Unmatched: true, Captured: "appimaged-838"},
				{Separator: true, Captured: "-"},
				{Keyword: "~arm64", Captured: "aarch64"},
				{Separator: true, Captured: "-"},
				{Unmatched: true, Captured: "asdf"},
				{Separator: true, Captured: "."},
				{AppImage: true, SuffixOnly: true, OS: "linux", Captured: "AppImage"},
			},
			releaseAsset: nil,
			filename:     "appimaged-838-aarch64-asdf.AppImage",
			want: &AppImageFileInfo{
				Keyword:          "~arm64",
				OS:               "linux",
				Toolchain:        "",
				Container:        "",
				ProgramName:      "appimaged-838",
				Containers:       nil,
				Filename:         "appimaged-838-aarch64-asdf.AppImage",
				OriginalFilename: "appimaged-838-aarch64-asdf.AppImage",
				AppImage:         true,
				SuffixOnly:       true,
				CaseInsensitive:  false,
				ReleaseAsset:     nil,
				Unmatched:        []string{"asdf"},
			},
			ok: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &AppImageFileInfo{
				ReleaseAsset: tt.releaseAsset,
				Filename:     tt.filename,
			}
			got, gotOk := base.CompileMeanings(tt.input)
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(AppImageFileInfo{})); diff != "" {
				t.Errorf("CompileMeanings() = \n%s", diff)
			}
			if gotOk != tt.ok {
				t.Errorf("CompileMeanings() gotOk = %v, want %v", gotOk, tt.ok)
			}
		})
	}
}
