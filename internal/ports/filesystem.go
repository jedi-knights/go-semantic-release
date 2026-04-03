package ports

import "io/fs"

// FileSystem abstracts file system operations for testability.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	Exists(path string) bool
	Walk(root string, fn fs.WalkDirFunc) error
	Glob(pattern string) ([]string, error)
}
