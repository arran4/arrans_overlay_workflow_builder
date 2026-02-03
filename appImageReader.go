package arrans_overlay_workflow_builder

import (
	"bytes"
	"fmt"
	"github.com/CalebQ42/squashfs"
	"io"
	"log"
	"os"
)

type AppImage struct {
	file   *os.File
	reader *squashfs.Reader
}

func NewAppImage(path string) (*AppImage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	offset, err := findSquashfsOffset(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("finding squashfs offset: %w", err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("getting file stat: %w", err)
	}

	size := stat.Size() - offset
	sr := io.NewSectionReader(f, offset, size)

	r, err := squashfs.NewReader(sr)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("initializing squashfs reader: %w", err)
	}

	return &AppImage{
		file:   f,
		reader: &r,
	}, nil
}

func (ai *AppImage) Close() error {
	return ai.file.Close()
}

func (ai *AppImage) ListFiles(path string) []string {
	entries, err := ai.reader.ReadDir(path)
	if err != nil {
		log.Printf("Error listing files in %s: %s", path, err)
		return []string{}
	}

	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	return files
}

func findSquashfsOffset(r io.ReaderAt) (int64, error) {
	buf := make([]byte, 4096)
	var offset int64 = 0

	// Scan up to 20MB
	maxScan := int64(20 * 1024 * 1024)

	for offset < maxScan {
		n, err := r.ReadAt(buf, offset)
		if n > 0 {
			// Search in buffer
			for i := 0; i <= n-4; i++ {
				// hsqs (Little Endian)
				if bytes.Equal(buf[i:i+4], []byte{'h', 's', 'q', 's'}) {
					return offset + int64(i), nil
				}
				// sqsh (Big Endian)
				if bytes.Equal(buf[i:i+4], []byte{'s', 'q', 's', 'h'}) {
					return offset + int64(i), nil
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}

		offset += int64(n) - 3 // Overlap
		if n < 4 {
			break
		}
	}

	return 0, fmt.Errorf("squashfs magic not found")
}
