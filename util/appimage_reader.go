package util

import (
	"bytes"
	"fmt"
	"github.com/CalebQ42/squashfs"
	"io"
	"io/fs"
	"os"
	"strings"
)

type AppImage struct {
	r *squashfs.Reader
	f *os.File
}

func NewAppImage(path string) (*AppImage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	offset, err := findSquashFSOffset(f)
	if err != nil {
		f.Close()
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	sr := io.NewSectionReader(f, offset, stat.Size()-offset)

	r, err := squashfs.NewReader(sr)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to create squashfs reader: %w", err)
	}

	return &AppImage{r: &r, f: f}, nil
}

func (ai *AppImage) Close() error {
	return ai.f.Close()
}

func (ai *AppImage) ListFiles(targetDir string) []string {
	var files []string
	targetDir = strings.TrimPrefix(targetDir, "./")
	if targetDir == "." {
		targetDir = ""
	}
	// Ensure targetDir ends with slash if not empty to match prefix correctly
	if targetDir != "" && !strings.HasSuffix(targetDir, "/") {
		targetDir += "/"
	}

	err := fs.WalkDir(ai.r, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// If targetDir is empty, we match everything (except maybe we want to mimic go-appimage?)
		// But assuming we want everything.

		if targetDir == "" {
			files = append(files, path)
			return nil
		}

		if strings.HasPrefix(path, targetDir) {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		// Log error? For now just return what we found
	}

	return files
}

func findSquashFSOffset(f *os.File) (int64, error) {
	hsqs := []byte{'h', 's', 'q', 's'}
	sqsh := []byte{'s', 'q', 's', 'h'}

	buf := make([]byte, 4096)
	var offset int64 = 0

	for {
		n, err := f.ReadAt(buf, offset)
		if n < 4 {
			if err != nil {
				if err == io.EOF {
					break
				}
				return 0, err
			}
			break
		}

		for i := 0; i <= n-4; i++ {
			if bytes.Equal(buf[i:i+4], hsqs) || bytes.Equal(buf[i:i+4], sqsh) {
				return offset + int64(i), nil
			}
		}

		if err == io.EOF {
			break
		}

		// Overlap for next read
		offset += int64(n) - 3
	}

	return 0, fmt.Errorf("squashfs magic not found")
}
