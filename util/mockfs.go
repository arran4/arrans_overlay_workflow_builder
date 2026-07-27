package util

import (
	"io/fs"
	"os"
)

type MockFS struct {
	Files map[string][]byte
}

func NewMockFS() *MockFS {
	return &MockFS{
		Files: make(map[string][]byte),
	}
}

func (m *MockFS) ReadFile(name string) ([]byte, error) {
	if b, ok := m.Files[name]; ok {
		return b, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	m.Files[name] = append([]byte{}, data...)
	return nil
}

func (m *MockFS) MkdirAll(path string, perm fs.FileMode) error {
	return nil
}
