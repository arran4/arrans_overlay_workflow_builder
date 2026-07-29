package util

import "os"

type FileSystem interface {
	ReadFile(name string) ([]byte, error)
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(name string, data []byte, perm os.FileMode) error
}

type OSFS struct{}

func (OSFS) ReadFile(name string) ([]byte, error)         { return os.ReadFile(name) }
func (OSFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (OSFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
