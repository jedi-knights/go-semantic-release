package app

import (
	"fmt"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

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
) (domain.Version, domain.ReleaseType, error) {
	bump := aggregateBump(commits, typeMapping)
	if !bump.IsReleasable() {
		return current, domain.ReleaseNone, nil
	}

	// For maintenance branches, constrain the allowed bump type.
	if policy != nil && policy.IsMaintenance() {
		bump = constrainMaintenanceBump(bump, policy)
		if !bump.IsReleasable() {
			return current, domain.ReleaseNone,
				fmt.Errorf("commit requires %s bump but maintenance branch %q does not allow it",
					aggregateBump(commits, typeMapping), policy.Name)
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
		pre := buildPrereleaseID(policy.Channel, next)
		next = next.WithPrerelease(pre)
	}

	return next, bump, nil
}

// constrainMaintenanceBump limits the bump type based on the maintenance range.
// A "N.N.x" range only allows patch bumps; "N.x" allows patch and minor.
func constrainMaintenanceBump(bump domain.ReleaseType, policy *domain.BranchPolicy) domain.ReleaseType {
	_, maxVer, err := policy.MaintenanceRange()
	if err != nil {
		return bump
	}

	// If max differs only in minor (N.N+1.0), only patch is allowed.
	// If max differs in major (N+1.0.0), patch and minor are allowed.
	if maxVer.Minor > 0 && maxVer.Patch == 0 {
		// Range like "1.2.x" → max is "1.3.0" → only patch allowed.
		if bump > domain.ReleasePatch {
			return domain.ReleaseNone
		}
	}

	// Major bumps are never allowed on maintenance branches.
	if bump == domain.ReleaseMajor {
		return domain.ReleaseNone
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

func buildPrereleaseID(channel string, version domain.Version) string {
	if channel == "" {
		channel = "pre"
	}
	return fmt.Sprintf("%s.%d.%d.%d", channel, version.Major, version.Minor, version.Patch)
}
