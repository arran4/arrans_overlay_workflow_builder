package arrans_overlay_workflow_builder

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-github/v62/github"
	"io"
	"os"
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

func TestReadDependenciesWithAppendedData(t *testing.T) {
	// Step 1: Create a temp file
	tmpfile, err := os.CreateTemp("", "test_appimage_*.AppImage")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()
	defer func() {
		_ = tmpfile.Close()
	}()

	// Step 2: Copy the current executable (a valid ELF) to the temp file
	exePath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exeFile, err := os.Open(exePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = exeFile.Close()
	}()

	if _, err := io.Copy(tmpfile, exeFile); err != nil {
		t.Fatal(err)
	}

	// Step 3: Append "SquashFS" data
	// Just garbage data to simulate the appended filesystem
	appendedData := []byte("This represents appended SquashFS data")
	if _, err := tmpfile.Write(appendedData); err != nil {
		t.Fatal(err)
	}

	// Ensure writes are flushed
	_ = tmpfile.Sync()

	// Step 4: Call ReadDependencies on this file
	// We expect it to succeed in opening the ELF and reading libraries,
	// even with the appended data.
	program := &Program{
		Dependencies: []string{},
	}

	// Note: ReadDependencies returns (unknownSymbols, err)
	// We mainly care that err is nil, proving debug/elf handled the file.
	unknowns, err := ReadDependencies(tmpfile.Name(), program)
	if err != nil {
		t.Fatalf("ReadDependencies failed on ELF with appended data: %v", err)
	}

	// Optional: verify we got some result (though it depends on what the test binary imports)
	// Just passing without error proves debug/elf ignored the appended data.
	t.Logf("Successfully read dependencies from AppImage-like file. Unknowns: %d, Known: %d", len(unknowns), len(program.Dependencies))
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
