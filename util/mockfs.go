package util

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type MockFS struct {
	Files map[string][]byte
	Modes map[string]fs.FileMode
	Dirs  map[string]bool
}

func NewMockFS() *MockFS {
	return &MockFS{
		Files: make(map[string][]byte),
		Modes: make(map[string]fs.FileMode),
		Dirs:  map[string]bool{".": true, "/": true},
	}
}

func (m *MockFS) ReadFile(name string) ([]byte, error) {
	if b, ok := m.Files[name]; ok {
		return b, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	if m.Modes == nil {
		m.Modes = make(map[string]fs.FileMode)
	}
	if _, exists := m.Files[name]; !exists {
		m.Modes[name] = perm.Perm()
	}
	m.Files[name] = append([]byte{}, data...)
	return nil
}

func (m *MockFS) MkdirAll(path string, perm fs.FileMode) error {
	if m.Dirs == nil {
		m.Dirs = make(map[string]bool)
	}
	for dir := filepath.Clean(path); ; dir = filepath.Dir(dir) {
		m.Dirs[dir] = true
		if filepath.Dir(dir) == dir {
			break
		}
	}
	return nil
}

func (m *MockFS) Remove(name string) error {
	if _, ok := m.Files[name]; ok {
		delete(m.Files, name)
		delete(m.Modes, name)
		return nil
	}
	if m.Dirs[name] {
		delete(m.Dirs, name)
		return nil
	}
	return os.ErrNotExist
}

func (m *MockFS) Lstat(name string) (os.FileInfo, error) {
	if b, ok := m.Files[name]; ok {
		return mockFileInfo{name: filepath.Base(name), size: int64(len(b)), mode: m.Modes[name]}, nil
	}
	if m.Dirs[filepath.Clean(name)] || name == "." || name == "/" {
		return mockFileInfo{name: filepath.Base(name), mode: fs.ModeDir | 0755}, nil
	}
	return nil, os.ErrNotExist
}
func (m *MockFS) Chmod(name string, mode os.FileMode) error {
	if _, ok := m.Files[name]; !ok {
		return os.ErrNotExist
	}
	if m.Modes == nil {
		m.Modes = make(map[string]fs.FileMode)
	}
	m.Modes[name] = mode
	return nil
}

type mockFileInfo struct {
	name string
	size int64
	mode fs.FileMode
}

func (i mockFileInfo) Name() string       { return i.name }
func (i mockFileInfo) Size() int64        { return i.size }
func (i mockFileInfo) Mode() fs.FileMode  { return i.mode }
func (i mockFileInfo) ModTime() time.Time { return time.Time{} }
func (i mockFileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i mockFileInfo) Sys() any           { return nil }
