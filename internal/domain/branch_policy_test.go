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
		{"unknown branch", "feature/foo", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := domain.FindBranchPolicy(policies, tt.branch)
			if (policy != nil) != tt.wantFound {
				t.Errorf("found = %v, wantFound = %v", policy != nil, tt.wantFound)
			}
			if policy != nil && policy.Prerelease != tt.wantPrerel {
				t.Errorf("prerelease = %v, want %v", policy.Prerelease, tt.wantPrerel)
			}
		})
	}
}
