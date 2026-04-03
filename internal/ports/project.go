package ports

import (
	"context"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// ProjectDiscoverer detects projects within a repository.
type ProjectDiscoverer interface {
	Discover(ctx context.Context, rootPath string) ([]domain.Project, error)
}

// ProjectImpactAnalyzer determines which projects are affected by a set of commits.
type ProjectImpactAnalyzer interface {
	Analyze(projects []domain.Project, commits []domain.Commit) map[string][]domain.Commit
}
