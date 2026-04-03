package app_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestVersionCalculatorService_Calculate(t *testing.T) {
	calc := app.NewVersionCalculatorService()
	mapping := domain.DefaultCommitTypeMapping()

	tests := []struct {
		name        string
		current     domain.Version
		commits     []domain.Commit
		policy      *domain.BranchPolicy
		wantVersion domain.Version
		wantType    domain.ReleaseType
	}{
		{
			name:        "no commits",
			current:     domain.NewVersion(1, 0, 0),
			commits:     nil,
			wantVersion: domain.NewVersion(1, 0, 0),
			wantType:    domain.ReleaseNone,
		},
		{
			name:        "single feat commit",
			current:     domain.NewVersion(1, 0, 0),
			commits:     []domain.Commit{{Type: "feat"}},
			wantVersion: domain.NewVersion(1, 1, 0),
			wantType:    domain.ReleaseMinor,
		},
		{
			name:        "single fix commit",
			current:     domain.NewVersion(1, 2, 3),
			commits:     []domain.Commit{{Type: "fix"}},
			wantVersion: domain.NewVersion(1, 2, 4),
			wantType:    domain.ReleasePatch,
		},
		{
			name:    "breaking change wins",
			current: domain.NewVersion(1, 2, 3),
			commits: []domain.Commit{
				{Type: "fix"},
				{Type: "feat"},
				{Type: "feat", IsBreakingChange: true},
			},
			wantVersion: domain.NewVersion(2, 0, 0),
			wantType:    domain.ReleaseMajor,
		},
		{
			name:    "mixed commits highest wins",
			current: domain.NewVersion(1, 0, 0),
			commits: []domain.Commit{
				{Type: "fix"},
				{Type: "feat"},
				{Type: "chore"},
			},
			wantVersion: domain.NewVersion(1, 1, 0),
			wantType:    domain.ReleaseMinor,
		},
		{
			name:    "only non-releasable commits",
			current: domain.NewVersion(1, 0, 0),
			commits: []domain.Commit{
				{Type: "chore"},
				{Type: "docs"},
				{Type: "ci"},
			},
			wantVersion: domain.NewVersion(1, 0, 0),
			wantType:    domain.ReleaseNone,
		},
		{
			name:        "from zero version",
			current:     domain.ZeroVersion(),
			commits:     []domain.Commit{{Type: "feat"}},
			wantVersion: domain.NewVersion(0, 1, 0),
			wantType:    domain.ReleaseMinor,
		},
		{
			name:        "prerelease branch",
			current:     domain.NewVersion(1, 0, 0),
			commits:     []domain.Commit{{Type: "feat"}},
			policy:      &domain.BranchPolicy{Prerelease: true, Channel: "beta"},
			wantVersion: domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "beta.1.1.0"},
			wantType:    domain.ReleaseMinor,
		},
		{
			name:        "alpha prerelease",
			current:     domain.NewVersion(2, 0, 0),
			commits:     []domain.Commit{{Type: "fix"}},
			policy:      &domain.BranchPolicy{Prerelease: true, Channel: "alpha"},
			wantVersion: domain.Version{Major: 2, Minor: 0, Patch: 1, Prerelease: "alpha.2.0.1"},
			wantType:    domain.ReleasePatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, gotType, err := calc.Calculate(tt.current, tt.commits, tt.policy, mapping)
			if err != nil {
				t.Fatalf("Calculate() error = %v", err)
			}
			if gotType != tt.wantType {
				t.Errorf("type = %v, want %v", gotType, tt.wantType)
			}
			if !gotVersion.Equal(tt.wantVersion) {
				t.Errorf("version = %v, want %v", gotVersion, tt.wantVersion)
			}
		})
	}
}
