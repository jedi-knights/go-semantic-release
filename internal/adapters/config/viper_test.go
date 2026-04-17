package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/config"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestViperProvider_Load_NoConfigFile_ReturnsDefaults(t *testing.T) {
	// Cannot call t.Parallel(): os.Chdir modifies process-global working directory.
	dir := t.TempDir()
	// Change into an empty directory so no config file is found.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if chErr := os.Chdir(dir); chErr != nil {
		t.Fatalf("chdir: %v", chErr)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	p := config.NewViperProvider()
	cfg, err := p.Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	defaults := domain.DefaultConfig()
	if cfg.TagFormat != defaults.TagFormat {
		t.Errorf("TagFormat = %q, want %q", cfg.TagFormat, defaults.TagFormat)
	}
	if cfg.ReleaseMode != defaults.ReleaseMode {
		t.Errorf("ReleaseMode = %v, want %v", cfg.ReleaseMode, defaults.ReleaseMode)
	}
}

func TestViperProvider_Load_ExplicitPath_OverridesDefaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".semantic-release.yaml")
	yaml := []byte("dry_run: true\ntag_format: \"{{.Version}}\"\n")
	if err := os.WriteFile(cfgPath, yaml, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	p := config.NewViperProvider()
	cfg, err := p.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.DryRun {
		t.Error("DryRun should be true from config file")
	}
	if cfg.TagFormat != "{{.Version}}" {
		t.Errorf("TagFormat = %q, want %q", cfg.TagFormat, "{{.Version}}")
	}
}

func TestViperProvider_Load_MissingExplicitPath_ReturnsError(t *testing.T) {
	t.Parallel()
	p := config.NewViperProvider()
	_, err := p.Load("/nonexistent/path/.releaserc.yaml")
	if err == nil {
		t.Error("Load() with missing explicit path should return error")
	}
}

func TestViperProvider_Load_MalformedYAML_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".semantic-release.yaml")
	if err := os.WriteFile(cfgPath, []byte(":\tinvalid: yaml: ["), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	p := config.NewViperProvider()
	_, err := p.Load(cfgPath)
	if err == nil {
		t.Error("Load() with malformed YAML should return error")
	}
}

func TestViperProvider_Load_GitHubAssets_StringForm(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".semantic-release.yaml")
	yaml := []byte("github:\n  assets:\n    - dist/*.tar.gz\n    - checksums.txt\n")
	if err := os.WriteFile(cfgPath, yaml, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	p := config.NewViperProvider()
	cfg, err := p.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.GitHub.Assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(cfg.GitHub.Assets))
	}
	if cfg.GitHub.Assets[0].Path != "dist/*.tar.gz" {
		t.Errorf("Assets[0].Path = %q, want %q", cfg.GitHub.Assets[0].Path, "dist/*.tar.gz")
	}
	if cfg.GitHub.Assets[0].Label != "" {
		t.Errorf("Assets[0].Label = %q, want empty for plain string form", cfg.GitHub.Assets[0].Label)
	}
	if cfg.GitHub.Assets[1].Path != "checksums.txt" {
		t.Errorf("Assets[1].Path = %q, want %q", cfg.GitHub.Assets[1].Path, "checksums.txt")
	}
}

func TestViperProvider_Load_GitHubAssets_StructForm(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".semantic-release.yaml")
	yaml := []byte("github:\n  assets:\n    - path: dist/*.tar.gz\n      label: Source Tarballs\n    - path: checksums.txt\n      label: Checksums\n")
	if err := os.WriteFile(cfgPath, yaml, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	p := config.NewViperProvider()
	cfg, err := p.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.GitHub.Assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(cfg.GitHub.Assets))
	}
	if cfg.GitHub.Assets[0].Path != "dist/*.tar.gz" {
		t.Errorf("Assets[0].Path = %q, want %q", cfg.GitHub.Assets[0].Path, "dist/*.tar.gz")
	}
	if cfg.GitHub.Assets[0].Label != "Source Tarballs" {
		t.Errorf("Assets[0].Label = %q, want %q", cfg.GitHub.Assets[0].Label, "Source Tarballs")
	}
	if cfg.GitHub.Assets[1].Label != "Checksums" {
		t.Errorf("Assets[1].Label = %q, want %q", cfg.GitHub.Assets[1].Label, "Checksums")
	}
}

func TestWriteDefaultConfig_CreatesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")

	if err := config.WriteDefaultConfig(path); err != nil {
		t.Fatalf("WriteDefaultConfig() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() after WriteDefaultConfig() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("WriteDefaultConfig() wrote empty file")
	}
}

func TestWriteDefaultConfig_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")

	if err := config.WriteDefaultConfig(path); err != nil {
		t.Fatalf("WriteDefaultConfig() error = %v", err)
	}

	p := config.NewViperProvider()
	cfg, err := p.Load(path)
	if err != nil {
		t.Fatalf("Load() after WriteDefaultConfig() error = %v", err)
	}

	defaults := domain.DefaultConfig()
	if cfg.TagFormat != defaults.TagFormat {
		t.Errorf("TagFormat = %q, want %q", cfg.TagFormat, defaults.TagFormat)
	}
	if cfg.ReleaseMode != defaults.ReleaseMode {
		t.Errorf("ReleaseMode = %v, want %v", cfg.ReleaseMode, defaults.ReleaseMode)
	}
	if cfg.GitHub.CreateRelease != defaults.GitHub.CreateRelease {
		t.Errorf("GitHub.CreateRelease = %v, want %v", cfg.GitHub.CreateRelease, defaults.GitHub.CreateRelease)
	}
}
