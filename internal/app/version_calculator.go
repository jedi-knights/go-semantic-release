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

	next := current.Bump(bump)

	if policy != nil && policy.Prerelease {
		pre := buildPrereleaseID(policy.Channel, next)
		next = next.WithPrerelease(pre)
	}

	return next, bump, nil
}

func aggregateBump(commits []domain.Commit, typeMapping map[string]domain.ReleaseType) domain.ReleaseType {
	highest := domain.ReleaseNone
	for _, c := range commits {
		rt := c.ReleaseType(typeMapping)
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
