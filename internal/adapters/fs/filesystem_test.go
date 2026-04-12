package fs_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	fsadapter "github.com/jedi-knights/go-semantic-release/internal/adapters/fs"
)

func TestOSFileSystem_WriteAndReadFile(t *testing.T) {
	dir := t.TempDir()
	fsa := fsadapter.NewOSFileSystem()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello filesystem")

	if err := fsa.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := fsa.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("ReadFile() = %q, want %q", got, content)
	}
}

func TestOSFileSystem_ReadFile_Missing(t *testing.T) {
	fsa := fsadapter.NewOSFileSystem()
	_, err := fsa.ReadFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("ReadFile() on missing file should return error")
	}
}

func TestOSFileSystem_Exists_True(t *testing.T) {
	dir := t.TempDir()
	fsa := fsadapter.NewOSFileSystem()
	path := filepath.Join(dir, "exists.txt")

	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if !fsa.Exists(path) {
		t.Errorf("Exists(%q) = false, want true", path)
	}
}

func TestOSFileSystem_Exists_False(t *testing.T) {
	fsa := fsadapter.NewOSFileSystem()
	if fsa.Exists("/nonexistent/path/surely/not/here.txt") {
		t.Error("Exists() on missing path should return false")
	}
}

func TestOSFileSystem_Walk_VisitsFiles(t *testing.T) {
	dir := t.TempDir()
	fsa := fsadapter.NewOSFileSystem()

	// Create a small tree.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var visited []string
	err := fsa.Walk(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			visited = append(visited, filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(visited) != 2 {
		t.Errorf("Walk() visited %d files, want 2: %v", len(visited), visited)
	}
}

func TestOSFileSystem_Glob_MatchesPattern(t *testing.T) {
	dir := t.TempDir()
	fsa := fsadapter.NewOSFileSystem()

	for _, name := range []string{"a.go", "b.go", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	matches, err := fsa.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("Glob() returned %d matches, want 2: %v", len(matches), matches)
	}
}
