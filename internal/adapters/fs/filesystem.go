package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// OSFileSystem implements ports.FileSystem using the real filesystem.
type OSFileSystem struct{}

// NewOSFileSystem creates a real filesystem adapter.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

func (f *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *OSFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (f *OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *OSFileSystem) Walk(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (f *OSFileSystem) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
