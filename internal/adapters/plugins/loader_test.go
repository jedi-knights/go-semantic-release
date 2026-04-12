package plugins_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
)

func writeExecutable(t *testing.T, dir, name, content string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("executable script tests not supported on Windows")
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+content+"\n"), 0o755); err != nil {
		t.Fatalf("writeExecutable: %v", err)
	}
	return path
}

func TestLoadExternalPlugins_Empty(t *testing.T) {
	result, err := plugins.LoadExternalPlugins([]string{})
	if err != nil {
		t.Fatalf("LoadExternalPlugins([]) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d plugins", len(result))
	}
}

func TestLoadExternalPlugins_EmptyStringSkipped(t *testing.T) {
	result, err := plugins.LoadExternalPlugins([]string{""})
	if err != nil {
		t.Fatalf("LoadExternalPlugins([\"\"] error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty-string ref, got %d plugins", len(result))
	}
}

func TestLoadExternalPlugins_BuiltinAliasSkipped(t *testing.T) {
	result, err := plugins.LoadExternalPlugins([]string{"@semantic-release/github"})
	if err != nil {
		t.Fatalf("LoadExternalPlugins() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected builtin alias to be skipped, got %d plugins", len(result))
	}
}

func TestLoadExternalPlugins_AllBuiltins(t *testing.T) {
	builtins := []string{
		"@semantic-release/commit-analyzer",
		"@semantic-release/release-notes-generator",
		"@semantic-release/changelog",
		"@semantic-release/git",
		"@semantic-release/github",
		"@semantic-release/gitlab",
	}
	result, err := plugins.LoadExternalPlugins(builtins)
	if err != nil {
		t.Fatalf("LoadExternalPlugins(all builtins) error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected all builtins to be skipped, got %d plugins", len(result))
	}
}

func TestLoadExternalPlugins_PathReference(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script tests not supported on Windows")
	}
	dir := t.TempDir()
	scriptPath := writeExecutable(t, dir, "my-plugin.sh", "exit 0")

	result, err := plugins.LoadExternalPlugins([]string{scriptPath})
	if err != nil {
		t.Fatalf("LoadExternalPlugins() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 plugin for path reference, got %d", len(result))
	}
}

func TestLoadExternalPlugins_NotFoundOnPath(t *testing.T) {
	// Use a name that is very unlikely to exist on any PATH.
	ref := "zzz-nonexistent-semantic-release-plugin-xyz-abc"

	_, err := plugins.LoadExternalPlugins([]string{ref})
	if err == nil {
		t.Fatal("expected error for missing executable, got nil")
	}
}

func TestLoadExternalPlugins_WithPrefixFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script tests not supported on Windows")
	}
	dir := t.TempDir()
	// Create the prefixed executable that resolveExecutable will fall back to.
	writeExecutable(t, dir, "semantic-release-myplug", "exit 0")

	// Prepend our temp dir to PATH so exec.LookPath finds the prefixed binary.
	original := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+original)

	result, err := plugins.LoadExternalPlugins([]string{"myplug"})
	if err != nil {
		t.Fatalf("LoadExternalPlugins() error = %v, expected prefix fallback to succeed", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 plugin via prefix fallback, got %d", len(result))
	}
}
