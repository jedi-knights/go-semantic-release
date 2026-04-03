package config_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/config"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestMergeConfigs(t *testing.T) {
	t.Run("base values take precedence", func(t *testing.T) {
		base := domain.Config{
			TagFormat: "v{{.Version}}",
			GitHub:    domain.GitHubConfig{Owner: "base-owner"},
		}
		parent := domain.Config{
			TagFormat:   "{{.Version}}",
			ReleaseMode: domain.ReleaseModeIndependent,
			GitHub:      domain.GitHubConfig{Owner: "parent-owner", Repo: "parent-repo"},
		}

		result := config.MergeConfigs(base, parent)

		if result.TagFormat != "v{{.Version}}" {
			t.Errorf("TagFormat = %q, want base value", result.TagFormat)
		}
		if result.ReleaseMode != domain.ReleaseModeIndependent {
			t.Errorf("ReleaseMode = %q, want parent value (base was empty)", result.ReleaseMode)
		}
		if result.GitHub.Owner != "base-owner" {
			t.Errorf("GitHub.Owner = %q, want base value", result.GitHub.Owner)
		}
		if result.GitHub.Repo != "parent-repo" {
			t.Errorf("GitHub.Repo = %q, want parent value (base was empty)", result.GitHub.Repo)
		}
	})

	t.Run("slices inherit from parent when empty", func(t *testing.T) {
		base := domain.Config{}
		parent := domain.Config{
			Branches:     []domain.BranchPolicy{{Name: "main"}},
			IncludePaths: []string{"src/**"},
		}

		result := config.MergeConfigs(base, parent)

		if len(result.Branches) != 1 {
			t.Errorf("Branches length = %d, want 1", len(result.Branches))
		}
		if len(result.IncludePaths) != 1 || result.IncludePaths[0] != "src/**" {
			t.Errorf("IncludePaths = %v, want [src/**]", result.IncludePaths)
		}
	})

	t.Run("identity merging", func(t *testing.T) {
		base := domain.Config{
			GitAuthor: domain.GitIdentity{Name: "custom-bot"},
		}
		parent := domain.Config{
			GitAuthor: domain.GitIdentity{Name: "parent-bot", Email: "parent@example.com"},
		}

		result := config.MergeConfigs(base, parent)

		if result.GitAuthor.Name != "custom-bot" {
			t.Errorf("GitAuthor.Name = %q, want base value", result.GitAuthor.Name)
		}
		if result.GitAuthor.Email != "parent@example.com" {
			t.Errorf("GitAuthor.Email = %q, want parent value", result.GitAuthor.Email)
		}
	})
}
