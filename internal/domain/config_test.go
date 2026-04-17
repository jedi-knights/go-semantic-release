package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestConfig_AnyProjectDefinesChangelog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		projects []domain.ProjectConfig
		want     bool
	}{
		{
			name:     "no projects",
			projects: nil,
			want:     false,
		},
		{
			name: "projects without changelog",
			projects: []domain.ProjectConfig{
				{Name: "a", Path: "./a"},
				{Name: "b", Path: "./b"},
			},
			want: false,
		},
		{
			name: "one project with changelog",
			projects: []domain.ProjectConfig{
				{Name: "a", Path: "./a", ChangelogFile: "CHANGELOG.md"},
				{Name: "b", Path: "./b"},
			},
			want: true,
		},
		{
			name: "all projects with changelog",
			projects: []domain.ProjectConfig{
				{Name: "a", ChangelogFile: "CHANGELOG.md"},
				{Name: "b", ChangelogFile: "CHANGES.md"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := domain.Config{Projects: tt.projects}
			if got := cfg.AnyProjectDefinesChangelog(); got != tt.want {
				t.Errorf("AnyProjectDefinesChangelog() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_IsInteractive(t *testing.T) {
	t.Parallel()
	trueVal := true
	falseVal := false

	tests := []struct {
		name        string
		interactive *bool
		want        bool
	}{
		{name: "nil (unset)", interactive: nil, want: false},
		{name: "explicitly false", interactive: &falseVal, want: false},
		{name: "explicitly true", interactive: &trueVal, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := domain.Config{Interactive: tt.interactive}
			if got := cfg.IsInteractive(); got != tt.want {
				t.Errorf("IsInteractive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVersionFileEntry_PlainPath(t *testing.T) {
	t.Parallel()
	e := domain.ParseVersionFileEntry("VERSION")
	if e.Path != "VERSION" {
		t.Errorf("Path = %q, want %q", e.Path, "VERSION")
	}
	if e.KeyPath != "" {
		t.Errorf("KeyPath = %q, want empty", e.KeyPath)
	}
}

func TestParseVersionFileEntry_TOMLKeyPath(t *testing.T) {
	t.Parallel()
	e := domain.ParseVersionFileEntry("pyproject.toml:tool.poetry.version")
	if e.Path != "pyproject.toml" {
		t.Errorf("Path = %q, want %q", e.Path, "pyproject.toml")
	}
	if e.KeyPath != "tool.poetry.version" {
		t.Errorf("KeyPath = %q, want %q", e.KeyPath, "tool.poetry.version")
	}
}

func TestParseVersionFileEntry_EmptyKeyPath(t *testing.T) {
	t.Parallel()
	// A trailing colon means the file has an explicitly empty key path.
	e := domain.ParseVersionFileEntry("some.toml:")
	if e.Path != "some.toml" {
		t.Errorf("Path = %q, want %q", e.Path, "some.toml")
	}
	if e.KeyPath != "" {
		t.Errorf("KeyPath = %q, want empty", e.KeyPath)
	}
}

func TestDefaultConfig_SensibleDefaults(t *testing.T) {
	t.Parallel()
	cfg := domain.DefaultConfig()

	if cfg.ReleaseMode != domain.ReleaseModeRepo {
		t.Errorf("ReleaseMode = %v, want ReleaseModeRepo", cfg.ReleaseMode)
	}
	if cfg.TagFormat == "" {
		t.Error("TagFormat must not be empty")
	}
	if cfg.ProjectTagFormat == "" {
		t.Error("ProjectTagFormat must not be empty")
	}
	if cfg.DryRun {
		t.Error("DryRun should default to false")
	}
	if !cfg.CI {
		t.Error("CI should default to true")
	}
	if len(cfg.Branches) == 0 {
		t.Error("Branches must not be empty")
	}
	if len(cfg.CommitTypes) == 0 {
		t.Error("CommitTypes must not be empty")
	}
	if len(cfg.ChangelogSections) == 0 {
		t.Error("ChangelogSections must not be empty")
	}
	if cfg.GitBackend != "cli" {
		t.Errorf("GitBackend = %q, want %q", cfg.GitBackend, "cli")
	}
	if !cfg.GitHub.CreateRelease {
		t.Error("GitHub.CreateRelease should default to true")
	}
}
