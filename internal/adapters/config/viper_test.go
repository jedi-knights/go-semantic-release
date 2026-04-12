package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/config"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestViperProvider_Load_NoConfigFile_ReturnsDefaults(t *testing.T) {
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
	p := config.NewViperProvider()
	_, err := p.Load("/nonexistent/path/.releaserc.yaml")
	if err == nil {
		t.Error("Load() with missing explicit path should return error")
	}
}

func TestWriteDefaultConfig_CreatesFile(t *testing.T) {
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
