package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/config"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestResolveExtends_FileRef(t *testing.T) {
	// Create a temp parent config file.
	dir := t.TempDir()
	parentPath := filepath.Join(dir, "parent.yaml")
	err := os.WriteFile(parentPath, []byte(`
release_mode: independent
tag_format: "custom-v{{.Version}}"
github:
  owner: shared-owner
  repo: shared-repo
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	base := domain.Config{
		TagFormat: "v{{.Version}}", // should win over parent
		Extends:   []string{parentPath},
	}

	result, err := config.ResolveExtends(base)
	if err != nil {
		t.Fatalf("ResolveExtends() error = %v", err)
	}

	if result.TagFormat != "v{{.Version}}" {
		t.Errorf("TagFormat = %q, want base value", result.TagFormat)
	}
	if result.ReleaseMode != domain.ReleaseModeIndependent {
		t.Errorf("ReleaseMode = %q, want parent value", result.ReleaseMode)
	}
	if result.GitHub.Owner != "shared-owner" {
		t.Errorf("GitHub.Owner = %q, want parent value", result.GitHub.Owner)
	}
}

func TestResolveExtends_CycleDetection(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.yaml")
	bPath := filepath.Join(dir, "b.yaml")

	_ = os.WriteFile(aPath, []byte("extends:\n  - "+bPath+"\n"), 0o644)
	_ = os.WriteFile(bPath, []byte("extends:\n  - "+aPath+"\n"), 0o644)

	base := domain.Config{Extends: []string{aPath}}

	_, err := config.ResolveExtends(base)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestResolveExtends_ChainedExtends(t *testing.T) {
	dir := t.TempDir()

	grandparentPath := filepath.Join(dir, "grandparent.yaml")
	_ = os.WriteFile(grandparentPath, []byte(`
release_mode: independent
github:
  owner: grandparent-owner
`), 0o644)

	parentPath := filepath.Join(dir, "parent.yaml")
	_ = os.WriteFile(parentPath, []byte(`
tag_format: "parent-v{{.Version}}"
extends:
  - `+grandparentPath+`
`), 0o644)

	base := domain.Config{Extends: []string{parentPath}}

	result, err := config.ResolveExtends(base)
	if err != nil {
		t.Fatalf("ResolveExtends() error = %v", err)
	}

	if result.ReleaseMode != domain.ReleaseModeIndependent {
		t.Errorf("ReleaseMode = %q, want grandparent value", result.ReleaseMode)
	}
	if result.TagFormat != "parent-v{{.Version}}" {
		t.Errorf("TagFormat = %q, want parent value", result.TagFormat)
	}
	if result.GitHub.Owner != "grandparent-owner" {
		t.Errorf("GitHub.Owner = %q, want grandparent value", result.GitHub.Owner)
	}
}

func TestResolveExtends_NoExtends(t *testing.T) {
	base := domain.Config{TagFormat: "v{{.Version}}"}

	result, err := config.ResolveExtends(base)
	if err != nil {
		t.Fatalf("ResolveExtends() error = %v", err)
	}

	if result.TagFormat != "v{{.Version}}" {
		t.Errorf("TagFormat = %q, want original value", result.TagFormat)
	}
}
