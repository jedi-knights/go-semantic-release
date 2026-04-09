package ports

import "github.com/jedi-knights/go-semantic-release/internal/domain"

// VersionCalculator computes the next version from current version and commits.
type VersionCalculator interface {
	// Calculate determines the next version from commits and branch policy.
	// prereleaseCounter is the number of existing prerelease tags for the same
	// base version and channel; it is used to produce the {channel}.{N} suffix
	// on prerelease branches and is ignored on stable branches.
	Calculate(current domain.Version, commits []domain.Commit, policy *domain.BranchPolicy, typeMapping map[string]domain.ReleaseType, prereleaseCounter int) (domain.Version, domain.ReleaseType, error)
}
