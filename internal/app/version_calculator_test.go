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
		counter     int
		wantVersion domain.Version
		wantType    domain.ReleaseType
		wantErr     bool
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
			name:        "prerelease branch first rc",
			current:     domain.NewVersion(1, 0, 0),
			commits:     []domain.Commit{{Type: "feat"}},
			policy:      &domain.BranchPolicy{Prerelease: true, Channel: "beta"},
			counter:     0,
			wantVersion: domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "beta.0"},
			wantType:    domain.ReleaseMinor,
		},
		{
			name:        "prerelease counter increments",
			current:     domain.NewVersion(1, 0, 0),
			commits:     []domain.Commit{{Type: "feat"}},
			policy:      &domain.BranchPolicy{Prerelease: true, Channel: "rc"},
			counter:     3,
			wantVersion: domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "rc.3"},
			wantType:    domain.ReleaseMinor,
		},
		{
			name:        "alpha prerelease first rc",
			current:     domain.NewVersion(2, 0, 0),
			commits:     []domain.Commit{{Type: "fix"}},
			policy:      &domain.BranchPolicy{Prerelease: true, Channel: "alpha"},
			counter:     0,
			wantVersion: domain.Version{Major: 2, Minor: 0, Patch: 1, Prerelease: "alpha.0"},
			wantType:    domain.ReleasePatch,
		},
		{
			name:        "empty channel defaults to pre",
			current:     domain.NewVersion(1, 0, 0),
			commits:     []domain.Commit{{Type: "fix"}},
			policy:      &domain.BranchPolicy{Prerelease: true, Channel: ""},
			counter:     2,
			wantVersion: domain.Version{Major: 1, Minor: 0, Patch: 1, Prerelease: "pre.2"},
			wantType:    domain.ReleasePatch,
		},
		// Maintenance branch cases.
		{
			name:    "maintenance 1.0.x blocks minor bump",
			current: domain.NewVersion(1, 0, 3),
			commits: []domain.Commit{{Type: "feat"}},
			policy: &domain.BranchPolicy{
				Name:  "1.0.x",
				Range: "1.0.x",
				Type:  domain.BranchTypeMaintenance,
			},
			wantVersion: domain.NewVersion(1, 0, 3),
			wantType:    domain.ReleaseNone,
			wantErr:     true,
		},
		{
			name:    "maintenance 1.x allows minor bump",
			current: domain.NewVersion(1, 0, 3),
			commits: []domain.Commit{{Type: "feat"}},
			policy: &domain.BranchPolicy{
				Name:  "1.x",
				Range: "1.x",
				Type:  domain.BranchTypeMaintenance,
			},
			wantVersion: domain.NewVersion(1, 1, 0),
			wantType:    domain.ReleaseMinor,
		},
		{
			name:    "maintenance 1.x blocks major bump",
			current: domain.NewVersion(1, 2, 0),
			commits: []domain.Commit{{Type: "feat", IsBreakingChange: true}},
			policy: &domain.BranchPolicy{
				Name:  "1.x",
				Range: "1.x",
				Type:  domain.BranchTypeMaintenance,
			},
			wantVersion: domain.NewVersion(1, 2, 0),
			wantType:    domain.ReleaseNone,
			wantErr:     true,
		},
		{
			// "N.N.x" ranges also block major bumps via the universal guard,
			// not just via the minor-bump check that fires first.
			name:    "maintenance 1.0.x blocks major bump",
			current: domain.NewVersion(1, 0, 3),
			commits: []domain.Commit{{Type: "feat", IsBreakingChange: true}},
			policy: &domain.BranchPolicy{
				Name:  "1.0.x",
				Range: "1.0.x",
				Type:  domain.BranchTypeMaintenance,
			},
			wantVersion: domain.NewVersion(1, 0, 3),
			wantType:    domain.ReleaseNone,
			wantErr:     true,
		},
		{
			// A policy that is IsMaintenance() but has an unparseable range must
			// fail-closed: Calculate returns an error rather than allowing the
			// bump through unconstrained.
			name:    "maintenance with invalid range fails closed",
			current: domain.NewVersion(1, 0, 0),
			commits: []domain.Commit{{Type: "feat"}},
			policy: &domain.BranchPolicy{
				Name:  "invalid",
				Range: "invalid",
				Type:  domain.BranchTypeMaintenance,
			},
			wantVersion: domain.NewVersion(1, 0, 0),
			wantType:    domain.ReleaseNone,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, gotType, err := calc.Calculate(tt.current, tt.commits, tt.policy, mapping, tt.counter)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Calculate() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Calculate() unexpected error = %v", err)
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
