package arrans_overlay_workflow_builder

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"io"
	"os"
	"path/filepath"
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

func TestSearchArchiveForFiles_Multilayer(t *testing.T) {
	tmp := t.TempDir()

	innerZipPath := filepath.Join(tmp, "inner.zip")
	zf, err := os.Create(innerZipPath)
	if err != nil {
		t.Fatalf("create inner zip: %v", err)
	}
	zw := zip.NewWriter(zf)
	w, err := zw.Create("prog")
	if err != nil {
		t.Fatalf("create file in zip: %v", err)
	}
	if _, err := w.Write([]byte("data")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := zf.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	outerPath := filepath.Join(tmp, "outer.tar.gz")
	of, err := os.Create(outerPath)
	if err != nil {
		t.Fatalf("create outer tar: %v", err)
	}
	gw := gzip.NewWriter(of)
	tw := tar.NewWriter(gw)
	innerFile, err := os.Open(innerZipPath)
	if err != nil {
		t.Fatalf("open inner zip: %v", err)
	}
	stat, _ := innerFile.Stat()
	hdr := &tar.Header{Name: "inner.zip", Mode: 0644, Size: stat.Size()}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := io.Copy(tw, innerFile); err != nil {
		t.Fatalf("copy inner: %v", err)
	}
	innerFile.Close()
	tw.Close()
	gw.Close()
	of.Close()

	brfi := &BinaryReleaseFileInfo{
		Filename:   "outer.tar.gz",
		Containers: []string{"tar", "gz"},
		tempFile:   outerPath,
	}

	layer, err := brfi.SearchArchiveForFiles()
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(layer.Layers) != 1 {
		t.Fatalf("expected 1 sub layer got %d", len(layer.Layers))
	}
	inner := layer.Layers[0]
	if inner.Archive.Filename != "inner.zip" {
		t.Errorf("inner archive name %s", inner.Archive.Filename)
	}
	if len(inner.Files) != 1 {
		t.Fatalf("expected 1 file inside, got %d", len(inner.Files))
	}
	if inner.Files[0].Filename != "prog" {
		t.Errorf("inner file name %s", inner.Files[0].Filename)
	}
}
