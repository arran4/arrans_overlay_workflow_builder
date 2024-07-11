package arrans_overlay_workflow_builder

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v62/github"
	"testing"
)

/**
Identify all from list:
	appimaged-838-aarch64.AppImage
	appimaged-838-aarch64.AppImage.zsync
	appimaged-838-armhf.AppImage
	appimaged-838-armhf.AppImage.zsync
	appimaged-838-i686.AppImage
	appimaged-838-i686.AppImage.zsync
	appimaged-838-x86_64.AppImage
	appimaged-838-x86_64.AppImage.zsync
	appimagetool-838-aarch64.AppImage
	appimagetool-838-aarch64.AppImage.zsync
	appimagetool-838-armhf.AppImage
	appimagetool-838-armhf.AppImage.zsync
	appimagetool-838-i686.AppImage
	appimagetool-838-i686.AppImage.zsync
	appimagetool-838-x86_64.AppImage
	appimagetool-838-x86_64.AppImage.zsync
	mkappimage-838-aarch64.AppImage
	mkappimage-838-aarch64.AppImage.zsync
	mkappimage-838-armhf.AppImage
	mkappimage-838-armhf.AppImage.zsync
	mkappimage-838-i686.AppImage
	mkappimage-838-i686.AppImage.zsync
	mkappimage-838-x86_64.AppImage
	mkappimage-838-x86_64.AppImage.zsync
*/

func TestDecodeFilename(t *testing.T) {
	tests := []struct {
		name           string
		groupedWordMap map[string][]*KeyedMeaning
		filename       string
		want           []*Meaning
	}{
		{
			name:           "jan-linux-x86_64-0.5.1.AppImage",
			groupedWordMap: GroupAndSort(GenerateWordMeanings("jan", "0.5.1")),
			filename:       "jan-linux-x86_64-0.5.1.AppImage",
			want: []*Meaning{
				{ProjectName: true},
				{OS: "linux"},
				{Keyword: "~amd64"},
				{Version: true},
				{AppImage: true, SuffixOnly: true, OS: "linux"},
			},
		},
		{
			name:           "appimaged-838-aarch64.AppImage",
			groupedWordMap: GroupAndSort(GenerateWordMeanings("go-appimage", "0")),
			filename:       "appimaged-838-aarch64.AppImage",
			want: []*Meaning{
				{Unmatched: "appimaged-838"},
				{Keyword: "~arm64"},
				{AppImage: true, SuffixOnly: true, OS: "linux"},
			},
		},
		{
			name:           "appimaged-838-aarch64.AppImage.zsync",
			groupedWordMap: GroupAndSort(GenerateWordMeanings("go-appimage", "0")),
			filename:       "appimaged-838-aarch64.AppImage.zsync",
			want: []*Meaning{
				{Unmatched: "appimaged-838"},
				{Keyword: "~arm64"},
				{AppImage: true, SuffixOnly: true, OS: "linux"},
				{Unmatched: "zsync", SuffixOnly: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeFilename(tt.groupedWordMap, tt.filename)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("DecodeFilename() = \n%s", diff)
			}
		})
	}
}

func TestCompileMeanings(t *testing.T) {
	tests := []struct {
		name         string
		input        []*Meaning
		releaseAsset *github.ReleaseAsset
		filename     string
		want         *Meaning
		ok           bool
	}{
		{
			name: "jan-linux-x86_64-0.5.1.AppImage",
			input: []*Meaning{
				{ProjectName: true},
				{OS: "linux"},
				{Keyword: "~amd64"},
				{Version: true},
				{AppImage: true, SuffixOnly: true, OS: "linux"},
			},
			releaseAsset: nil,
			filename:     "jan-linux-x86_64-0.5.1.AppImage",
			want: &Meaning{
				Keyword:       "~amd64",
				OS:            "linux",
				Toolchain:     "",
				Container:     "",
				Containers:    nil,
				Filename:      "jan-linux-x86_64-0.5.1.AppImage",
				AppImage:      true,
				Version:       true,
				ProjectName:   true,
				SuffixOnly:    true,
				CaseSensitive: false,
				ReleaseAsset:  nil,
			},
			ok: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := CompileMeanings(tt.input, tt.releaseAsset, tt.filename)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("CompileMeanings() = \n%s", diff)
			}
			if gotOk != tt.ok {
				t.Errorf("CompileMeanings() gotOk = %v, want %v", gotOk, tt.ok)
			}
		})
	}
}
