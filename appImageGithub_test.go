package arrans_overlay_workflow_builder

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-github/v62/github"
	"os"
	"testing"
)

func TestFindSquashfsOffset(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    int64
		wantErr bool
	}{
		{
			name:    "found at start",
			content: []byte{0x68, 0x73, 0x71, 0x73, 0x00},
			want:    0,
			wantErr: false,
		},
		{
			name:    "found at offset",
			content: append(make([]byte, 100), []byte{0x68, 0x73, 0x71, 0x73}...),
			want:    100,
			wantErr: false,
		},
		{
			name:    "not found",
			content: []byte{0x00, 0x01, 0x02, 0x03},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "squashfs-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name()) // clean up

			if _, err := tmpfile.Write(tt.content); err != nil {
				tmpfile.Close()
				t.Fatal(err)
			}
			if _, err := tmpfile.Seek(0, 0); err != nil {
				tmpfile.Close()
				t.Fatal(err)
			}

			got, err := findSquashfsOffset(tmpfile)
			tmpfile.Close()

			if (err != nil) != tt.wantErr {
				t.Errorf("findSquashfsOffset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("findSquashfsOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

func TestSelectPrimaryAppImage(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		input     []*AppImageFileInfo
		wantFirst string
	}{
		{
			name: "prefer repo name match",
			repo: "example",
			input: []*AppImageFileInfo{
				{ProgramName: "other", OriginalFilename: "other.AppImage"},
				{ProgramName: "example", OriginalFilename: "example.AppImage"},
				{ProgramName: "third", OriginalFilename: "third.AppImage"},
			},
			wantFirst: "example.AppImage",
		},
		{
			name: "prefer blank when no match",
			repo: "example",
			input: []*AppImageFileInfo{
				{ProgramName: "foo", OriginalFilename: "foo.AppImage"},
				{ProgramName: "", OriginalFilename: "example.AppImage"},
				{ProgramName: "bar", OriginalFilename: "bar.AppImage"},
			},
			wantFirst: "example.AppImage",
		},
		{
			name: "fallback to closest name",
			repo: "example",
			input: []*AppImageFileInfo{
				{ProgramName: "sample", OriginalFilename: "sample.AppImage"},
				{ProgramName: "exampel", OriginalFilename: "exampel.AppImage"},
				{ProgramName: "another", OriginalFilename: "another.AppImage"},
			},
			wantFirst: "exampel.AppImage",
		},
		{
			name: "match ignoring case",
			repo: "Example",
			input: []*AppImageFileInfo{
				{ProgramName: "EXAMPLE", OriginalFilename: "EXAMPLE.AppImage"},
				{ProgramName: "other", OriginalFilename: "other.AppImage"},
			},
			wantFirst: "EXAMPLE.AppImage",
		},
		{
			name: "match camel and snake case",
			repo: "exampleApp",
			input: []*AppImageFileInfo{
				{ProgramName: "example_app", OriginalFilename: "example_app.AppImage"},
				{ProgramName: "sample", OriginalFilename: "sample.AppImage"},
			},
			wantFirst: "example_app.AppImage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectPrimaryAppImage(tt.input, tt.repo)
			if len(got) != 1 {
				t.Fatalf("len(got) = %d, want 1", len(got))
			}
			if got[0].OriginalFilename != tt.wantFirst {
				t.Fatalf("got %s, want %s", got[0].OriginalFilename, tt.wantFirst)
			}
		})
	}
}
