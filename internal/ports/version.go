package ports

import "github.com/jedi-knights/go-semantic-release/internal/domain"

// VersionCalculator computes the next version from current version and commits.
type VersionCalculator interface {
	Calculate(current domain.Version, commits []domain.Commit, policy *domain.BranchPolicy, typeMapping map[string]domain.ReleaseType) (domain.Version, domain.ReleaseType, error)
}
