package app

import (
	"fmt"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.VersionCalculator = (*VersionCalculatorService)(nil)

// VersionCalculatorService implements ports.VersionCalculator.
type VersionCalculatorService struct{}

// NewVersionCalculatorService creates a new version calculator.
func NewVersionCalculatorService() *VersionCalculatorService {
	return &VersionCalculatorService{}
}

func (s *VersionCalculatorService) Calculate(
	current domain.Version,
	commits []domain.Commit,
	policy *domain.BranchPolicy,
	typeMapping map[string]domain.ReleaseType,
	prereleaseCounter int,
) (domain.Version, domain.ReleaseType, error) {
	bump := aggregateBump(commits, typeMapping)
	if !bump.IsReleasable() {
		return current, domain.ReleaseNone, nil
	}

	// For maintenance branches, constrain the allowed bump type.
	if policy != nil && policy.IsMaintenance() {
		original := bump
		bump = constrainMaintenanceBump(bump, policy)
		if !bump.IsReleasable() {
			return current, domain.ReleaseNone,
				fmt.Errorf("commit requires %s bump but maintenance branch %q does not allow it",
					original, policy.Name)
		}
	}

	next := current.Bump(bump)

	// Validate maintenance range.
	if policy != nil && policy.IsMaintenance() {
		if err := domain.ValidateMaintenanceVersion(next, *policy); err != nil {
			return current, domain.ReleaseNone, err
		}
	}

	// Apply prerelease identifier.
	if policy != nil && policy.IsPrerelease() {
		pre := buildPrereleaseID(policy.Channel, prereleaseCounter)
		next = next.WithPrerelease(pre)
	}

	return next, bump, nil
}

// constrainMaintenanceBump limits the bump type based on the maintenance range.
// A "N.N.x" range only allows patch bumps; "N.x" allows patch and minor.
//
// NOTE: for "N.x" ranges this function permits minor bumps but does not verify
// that the resulting version stays within the major boundary. That upper-bound
// check is performed by ValidateMaintenanceVersion (called by Calculate after
// Bump). Callers must invoke both functions in sequence; calling this function
// alone is not sufficient to enforce the full maintenance constraint.
func constrainMaintenanceBump(bump domain.ReleaseType, policy *domain.BranchPolicy) domain.ReleaseType {
	_, maxVer, err := policy.MaintenanceRange()
	if err != nil {
		return domain.ReleaseNone
	}

	// Major bumps are never allowed on any maintenance branch.
	if bump == domain.ReleaseMajor {
		return domain.ReleaseNone
	}

	// "N.N.x" range (max differs only in minor): only patch is allowed.
	// "N.x" range (max differs in major): patch and minor are both allowed.
	if maxVer.Minor > 0 && maxVer.Patch == 0 {
		// Range like "1.2.x" → max is "1.3.0" → only patch allowed.
		if bump > domain.ReleasePatch {
			return domain.ReleaseNone
		}
	}

	return bump
}

func aggregateBump(commits []domain.Commit, typeMapping map[string]domain.ReleaseType) domain.ReleaseType {
	highest := domain.ReleaseNone
	for i := range commits {
		rt := commits[i].ReleaseType(typeMapping)
		highest = highest.Higher(rt)
	}
	return highest
}

func buildPrereleaseID(channel string, counter int) string {
	if channel == "" {
		channel = "pre"
	}
	if counter < 0 {
		counter = 0
	}
	return fmt.Sprintf("%s.%d", channel, counter)
}
