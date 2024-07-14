package arrans_overlay_workflow_builder

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-github/v62/github"
	"testing"
)

func TestDecodeFilename(t *testing.T) {
	tests := []struct {
		name           string
		groupedWordMap map[string][]*KeyedMeaning
		filename       string
		want           []*FileInfo
	}{
		{
			name:           "jan-linux-x86_64-0.5.1.AppImage",
			groupedWordMap: GroupAndSort(GenerateWordMeanings("jan", []string{"0.5.1"}, []string{"v0.5.1"})),
			filename:       "jan-linux-x86_64-0.5.1.AppImage",
			want: []*FileInfo{
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
		},
		{
			name:           "appimaged-838-aarch64.AppImage",
			groupedWordMap: GroupAndSort(GenerateWordMeanings("go-appimage", []string{"0"}, []string{"v0"})),
			filename:       "appimaged-838-aarch64.AppImage",
			want: []*FileInfo{
				{Unmatched: true, Captured: "appimaged-838"},
				{Separator: true, Captured: "-"},
				{Keyword: "~arm64", Captured: "aarch64"},
				{Separator: true, Captured: "."},
				{AppImage: true, SuffixOnly: true, OS: "linux", Captured: "AppImage"},
			},
		},
		{
			name:           "appimaged-838-aarch64.AppImage.zsync",
			groupedWordMap: GroupAndSort(GenerateWordMeanings("go-appimage", []string{"0"}, []string{"v0"})),
			filename:       "appimaged-838-aarch64.AppImage.zsync",
			want: []*FileInfo{
				{Unmatched: true, Captured: "appimaged-838"},
				{Separator: true, Captured: "-"},
				{Keyword: "~arm64", Captured: "aarch64"},
				{Separator: true, Captured: "."},
				{AppImage: true, SuffixOnly: true, OS: "linux", Captured: "AppImage"},
				{Separator: true, Captured: "."},
				{Unmatched: true, Captured: "zsync", SuffixOnly: true},
			},
		},
		{
			name:           "LocalSend-1.14.0-linux-x86-64.AppImage",
			groupedWordMap: GroupAndSort(GenerateWordMeanings("localsend", []string{"1.14.0"}, []string{"v1.14.0"})),
			filename:       "LocalSend-1.14.0-linux-x86-64.AppImage",
			want: []*FileInfo{
				{ProjectName: true, CaseInsensitive: true, Captured: "LocalSend"},
				{Separator: true, Captured: "-"},
				//{ProgramName: "LocalSend"},
				{Version: true, Captured: "1.14.0"},
				{Separator: true, Captured: "-"},
				{OS: "linux", Captured: "linux"},
				{Separator: true, Captured: "-"},
				{Keyword: "~amd64", Captured: "x86-64"},
				{Separator: true, Captured: "."},
				{AppImage: true, SuffixOnly: true, OS: "linux", Captured: "AppImage"},
			},
		},
		{
			name:           "StabilityMatrix-linux-x64.zip",
			groupedWordMap: GroupAndSort(GenerateWordMeanings("StabilityMatrix", []string{"2.11.4"}, []string{"v2.11.4"})),
			filename:       "StabilityMatrix-linux-x64.zip",
			want: []*FileInfo{
				{ProjectName: true, CaseInsensitive: true, Captured: "StabilityMatrix"},
				{Separator: true, Captured: "-"},
				//{ProgramName: "LocalSend"},
				{OS: "linux", Captured: "linux"},
				{Separator: true, Captured: "-"},
				{Keyword: "~amd64", Captured: "x64"},
				{Separator: true, Captured: "."},
				{Container: "zip", SuffixOnly: true, Captured: "zip"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeFilename(tt.groupedWordMap, tt.filename)
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(FileInfo{})); diff != "" {
				t.Errorf("DecodeFilename() = \n%s", diff)
			}
		})
	}
}

func TestCompileMeanings(t *testing.T) {
	tests := []struct {
		name         string
		input        []*FileInfo
		releaseAsset *github.ReleaseAsset
		filename     string
		want         *FileInfo
		ok           bool
	}{
		{
			name: "jan-linux-x86_64-0.5.1.AppImage",
			input: []*FileInfo{
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
			want: &FileInfo{
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
			input: []*FileInfo{
				{Unmatched: true, Captured: "appimaged-838"},
				{Separator: true, Captured: "-"},
				{Keyword: "~arm64", Captured: "aarch64"},
				{Separator: true, Captured: "."},
				{AppImage: true, SuffixOnly: true, OS: "linux", Captured: "AppImage"},
			},
			releaseAsset: nil,
			filename:     "appimaged-838-aarch64.AppImage",
			want: &FileInfo{
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
			input: []*FileInfo{
				{Unmatched: true, Captured: "appimaged-838"},
				{Keyword: "~arm64"},
				{AppImage: true, SuffixOnly: true, OS: "linux"},
				{Unmatched: true, Captured: "zsync", SuffixOnly: true},
			},
			releaseAsset: nil,
			filename:     "appimaged-838-aarch64.AppImage.zsync",
			want:         nil,
			ok:           false,
		},
		{
			name: "appimaged-838-aarch64-asdf.AppImage",
			input: []*FileInfo{
				{Unmatched: true, Captured: "appimaged-838"},
				{Keyword: "~arm64"},
				{Unmatched: true, Captured: "asdf"},
				{AppImage: true, SuffixOnly: true, OS: "linux"},
			},
			releaseAsset: nil,
			filename:     "appimaged-838-aarch64-asdf.AppImage",
			want:         nil,
			ok:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := CompileMeanings(tt.input, &FileInfo{
				ReleaseAsset: tt.releaseAsset,
				Filename:     tt.filename,
			})
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(FileInfo{})); diff != "" {
				t.Errorf("CompileMeanings() = \n%s", diff)
			}
			if gotOk != tt.ok {
				t.Errorf("CompileMeanings() gotOk = %v, want %v", gotOk, tt.ok)
			}
		})
	}
}
