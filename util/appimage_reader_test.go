package util

import (
	"os"
	"testing"
)

func TestFindSquashFSOffset(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		expectedOffset int64
		expectError    bool
	}{
		{
			name:           "Magic hsqs at 0",
			data:           []byte{'h', 's', 'q', 's', 0, 0, 0, 0},
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "Magic sqsh at 0",
			data:           []byte{'s', 'q', 's', 'h', 0, 0, 0, 0},
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "Magic at 100",
			data:           append(make([]byte, 100), []byte{'h', 's', 'q', 's'}...),
			expectedOffset: 100,
			expectError:    false,
		},
		{
			name:           "Magic at 4096 (buffer boundary)",
			data:           append(make([]byte, 4096), []byte{'h', 's', 'q', 's'}...),
			expectedOffset: 4096,
			expectError:    false,
		},
		{
			name: "Magic split across buffer boundary (at 4094)",
			// Buffer size is 4096. Offset logic overlaps by 3 bytes.
			// Magic at 4094 means bytes at 4094, 4095, 4096, 4097.
			// This splits across the first 4096 read.
			data:           append(make([]byte, 4094), []byte{'h', 's', 'q', 's', 0, 0, 0, 0}...),
			expectedOffset: 4094,
			expectError:    false,
		},
		{
			name:           "No magic",
			data:           make([]byte, 5000),
			expectedOffset: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "appimage_test")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = os.Remove(tmpfile.Name())
			}()

			if _, err := tmpfile.Write(tt.data); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Re-open for reading to match usage in findSquashFSOffset if it took a path,
			// but it takes *os.File. So open it.
			f, err := os.Open(tmpfile.Name())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()

			offset, err := findSquashFSOffset(f)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if offset != tt.expectedOffset {
					t.Errorf("Expected offset %d, got %d", tt.expectedOffset, offset)
				}
			}
		})
	}
}
