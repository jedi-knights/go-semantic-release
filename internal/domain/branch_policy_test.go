package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestFindBranchPolicy(t *testing.T) {
	policies := domain.DefaultBranchPolicies()

	tests := []struct {
		name       string
		branch     string
		wantFound  bool
		wantPrerel bool
	}{
		{"main branch", "main", true, false},
		{"master branch", "master", true, false},
		{"beta branch", "beta", true, true},
		{"alpha branch", "alpha", true, true},
		{"next branch", "next", true, true},
		{"next-major branch", "next-major", true, true},
		{"unknown branch", "feature/foo", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := domain.FindBranchPolicy(policies, tt.branch)
			if (policy != nil) != tt.wantFound {
				t.Errorf("found = %v, wantFound = %v", policy != nil, tt.wantFound)
			}
			if policy != nil && policy.IsPrerelease() != tt.wantPrerel {
				t.Errorf("prerelease = %v, want %v", policy.IsPrerelease(), tt.wantPrerel)
			}
		})
	}
}

func TestFindBranchPolicy_AutoDetectMaintenance(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{"1.x maintenance", "1.x", true},
		{"1.0.x maintenance", "1.0.x", true},
		{"2.x maintenance", "2.x", true},
		{"not maintenance", "feature-x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := domain.FindBranchPolicy(nil, tt.branch)
			found := policy != nil && policy.IsMaintenance()
			if found != tt.want {
				t.Errorf("auto-detect maintenance = %v, want %v", found, tt.want)
			}
		})
	}
}

func TestMaintenanceRange(t *testing.T) {
	tests := []struct {
		name    string
		policy  domain.BranchPolicy
		wantMin domain.Version
		wantMax domain.Version
		wantErr bool
	}{
		{
			name:    "major range 1.x",
			policy:  domain.BranchPolicy{Range: "1.x"},
			wantMin: domain.NewVersion(1, 0, 0),
			wantMax: domain.NewVersion(2, 0, 0),
		},
		{
			name:    "minor range 1.2.x",
			policy:  domain.BranchPolicy{Range: "1.2.x"},
			wantMin: domain.NewVersion(1, 2, 0),
			wantMax: domain.NewVersion(1, 3, 0),
		},
		{
			name:    "inferred from name",
			policy:  domain.BranchPolicy{Name: "2.x"},
			wantMin: domain.NewVersion(2, 0, 0),
			wantMax: domain.NewVersion(3, 0, 0),
		},
		{
			name:    "invalid range",
			policy:  domain.BranchPolicy{Range: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minVer, maxVer, err := tt.policy.MaintenanceRange()
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !minVer.Equal(tt.wantMin) {
				t.Errorf("min = %v, want %v", minVer, tt.wantMin)
			}
			if !maxVer.Equal(tt.wantMax) {
				t.Errorf("max = %v, want %v", maxVer, tt.wantMax)
			}
		})
	}
}

func TestVersionInRange(t *testing.T) {
	tests := []struct {
		name    string
		version domain.Version
		min     domain.Version
		max     domain.Version
		want    bool
	}{
		{
			name:    "within range",
			version: domain.NewVersion(1, 2, 3),
			min:     domain.NewVersion(1, 0, 0),
			max:     domain.NewVersion(2, 0, 0),
			want:    true,
		},
		{
			name:    "at min boundary",
			version: domain.NewVersion(1, 0, 0),
			min:     domain.NewVersion(1, 0, 0),
			max:     domain.NewVersion(2, 0, 0),
			want:    true,
		},
		{
			name:    "at max boundary (exclusive)",
			version: domain.NewVersion(2, 0, 0),
			min:     domain.NewVersion(1, 0, 0),
			max:     domain.NewVersion(2, 0, 0),
			want:    false,
		},
		{
			name:    "below range",
			version: domain.NewVersion(0, 9, 0),
			min:     domain.NewVersion(1, 0, 0),
			max:     domain.NewVersion(2, 0, 0),
			want:    false,
		},
		{
			name:    "above range",
			version: domain.NewVersion(3, 0, 0),
			min:     domain.NewVersion(1, 0, 0),
			max:     domain.NewVersion(2, 0, 0),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.VersionInRange(tt.version, tt.min, tt.max)
			if got != tt.want {
				t.Errorf("VersionInRange(%v, %v, %v) = %v, want %v",
					tt.version, tt.min, tt.max, got, tt.want)
			}
		})
	}
}

func TestValidateMaintenanceVersion(t *testing.T) {
	tests := []struct {
		name    string
		version domain.Version
		policy  domain.BranchPolicy
		wantErr bool
	}{
		{
			name:    "valid patch in 1.0.x",
			version: domain.NewVersion(1, 0, 5),
			policy:  domain.BranchPolicy{Range: "1.0.x", Type: domain.BranchTypeMaintenance},
			wantErr: false,
		},
		{
			name:    "minor bump not allowed in 1.0.x",
			version: domain.NewVersion(1, 1, 0),
			policy:  domain.BranchPolicy{Range: "1.0.x", Type: domain.BranchTypeMaintenance},
			wantErr: true,
		},
		{
			name:    "valid minor in 1.x",
			version: domain.NewVersion(1, 5, 0),
			policy:  domain.BranchPolicy{Range: "1.x", Type: domain.BranchTypeMaintenance},
			wantErr: false,
		},
		{
			name:    "major bump not allowed in 1.x",
			version: domain.NewVersion(2, 0, 0),
			policy:  domain.BranchPolicy{Range: "1.x", Type: domain.BranchTypeMaintenance},
			wantErr: true,
		},
		{
			name:    "non-maintenance policy passes anything",
			version: domain.NewVersion(99, 0, 0),
			policy:  domain.BranchPolicy{Name: "main"},
			wantErr: false,
		},
		{
			// IsMaintenance() is true (Type == BranchTypeMaintenance) but both Range and Name are
			// invalid — MaintenanceRange() will fail, propagating the parse error.
			name:    "maintenance policy with unparseable range returns error",
			version: domain.NewVersion(1, 0, 0),
			policy: domain.BranchPolicy{
				Name:  "broken-range",
				Range: "not-a-range",
				Type:  domain.BranchTypeMaintenance,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateMaintenanceVersion(tt.version, tt.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestFindBranchPolicy_GlobMatch(t *testing.T) {
	// A policy with a glob pattern should match branches that satisfy the pattern.
	policies := []domain.BranchPolicy{
		{Name: "release/*", Type: domain.BranchTypeRelease},
	}

	tests := []struct {
		name      string
		branch    string
		wantFound bool
	}{
		{"matching glob", "release/1.0", true},
		{"non-matching branch", "hotfix/1.0", false},
		{"exact match in glob policy", "release/2.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := domain.FindBranchPolicy(policies, tt.branch)
			if (policy != nil) != tt.wantFound {
				t.Errorf("FindBranchPolicy(%q) found=%v, wantFound=%v", tt.branch, policy != nil, tt.wantFound)
			}
		})
	}
}

func TestParseMaintenanceRange_TooManyParts(t *testing.T) {
	// A range with 4 parts (e.g. "1.2.3.x") has len(parts)==4 and is not handled
	// by the 2-part or 3-part branches, so it falls through to the final error return.
	policy := domain.BranchPolicy{Range: "1.2.3.x"}
	_, _, err := policy.MaintenanceRange()
	if err == nil {
		t.Fatal("expected error for 4-part range like '1.2.3.x', got nil")
	}
}

func TestParseMaintenanceRange_NonNumericMinor(t *testing.T) {
	// A range like "1.abc.x" has a valid major but an invalid minor.
	policy := domain.BranchPolicy{Range: "1.abc.x"}
	_, _, err := policy.MaintenanceRange()
	if err == nil {
		t.Fatal("expected error for non-numeric minor in range, got nil")
	}
}
