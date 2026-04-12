package config_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestResolveExtends_MissingFile(t *testing.T) {
	base := domain.Config{Extends: []string{"/nonexistent/path/config.yaml"}}

	_, err := config.ResolveExtends(base)
	if err == nil {
		t.Fatal("ResolveExtends() error = nil, want error for missing file")
	}
}

func TestResolveExtends_MaxDepth(t *testing.T) {
	// Create a chain of 12 config files, each extending the next.
	// This exceeds maxExtendsDepth (10) and should return an error.
	dir := t.TempDir()
	const chainLen = 12

	// Create files from last to first so each can reference the next.
	paths := make([]string, chainLen)
	for i := 0; i < chainLen; i++ {
		paths[i] = filepath.Join(dir, fmt.Sprintf("config%02d.yaml", i))
	}

	// Last file has no extends.
	_ = os.WriteFile(paths[chainLen-1], []byte("tag_format: \"leaf\"\n"), 0o644)

	// Each preceding file extends the next.
	for i := chainLen - 2; i >= 0; i-- {
		content := fmt.Sprintf("extends:\n  - %s\n", paths[i+1])
		_ = os.WriteFile(paths[i], []byte(content), 0o644)
	}

	base := domain.Config{Extends: []string{paths[0]}}
	_, err := config.ResolveExtends(base)
	if err == nil {
		t.Fatal("ResolveExtends() error = nil, want error for chain exceeding max depth")
	}
	if !strings.Contains(err.Error(), "depth") && !strings.Contains(err.Error(), "maximum") {
		t.Errorf("error = %q, want to mention depth limit", err.Error())
	}
}

func TestResolveExtends_URL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`tag_format: "url-v{{.Version}}"`))
	}))
	defer srv.Close()

	// Base config has no TagFormat so the parent value should be merged in.
	base := domain.Config{Extends: []string{srv.URL + "/config.yaml"}}

	result, err := config.ResolveExtends(base)
	if err != nil {
		t.Fatalf("ResolveExtends() error = %v", err)
	}
	if result.TagFormat != "url-v{{.Version}}" {
		t.Errorf("TagFormat = %q, want url-v{{.Version}}", result.TagFormat)
	}
}

func TestResolveExtends_URL_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	base := domain.Config{Extends: []string{srv.URL + "/config.yaml"}}

	_, err := config.ResolveExtends(base)
	if err == nil {
		t.Fatal("ResolveExtends() error = nil, want error for HTTP 404")
	}
}
