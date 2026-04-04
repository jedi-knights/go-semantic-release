package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestDefaultEnabledLintConfig(t *testing.T) {
	cfg := domain.DefaultEnabledLintConfig()

	if !cfg.Enabled {
		t.Error("DefaultEnabledLintConfig().Enabled = false, want true")
	}
	if cfg.MaxSubjectLength != 72 {
		t.Errorf("DefaultEnabledLintConfig().MaxSubjectLength = %d, want 72", cfg.MaxSubjectLength)
	}
	if len(cfg.AllowedTypes) == 0 {
		t.Error("DefaultEnabledLintConfig().AllowedTypes is empty, want non-empty")
	}
}

func TestDefaultLintConfig_Disabled(t *testing.T) {
	cfg := domain.DefaultLintConfig()

	if cfg.Enabled {
		t.Error("DefaultLintConfig().Enabled = true, want false")
	}
	if cfg.MaxSubjectLength != 72 {
		t.Errorf("DefaultLintConfig().MaxSubjectLength = %d, want 72", cfg.MaxSubjectLength)
	}
}
