package app

import (
	"context"
	"fmt"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// ProjectDetector orchestrates project discovery.
type ProjectDetector struct {
	discoverer ports.ProjectDiscoverer
	logger     ports.Logger
}

// NewProjectDetector creates a project detector.
func NewProjectDetector(discoverer ports.ProjectDiscoverer, logger ports.Logger) *ProjectDetector {
	return &ProjectDetector{discoverer: discoverer, logger: logger}
}

// Detect discovers projects in the repository.
func (d *ProjectDetector) Detect(ctx context.Context, rootPath string) ([]domain.Project, error) {
	projects, err := d.discoverer.Discover(ctx, rootPath)
	if err != nil {
		return nil, fmt.Errorf("discovering projects: %w", err)
	}

	if len(projects) == 0 {
		d.logger.Info("no projects discovered, using root project")
		return []domain.Project{{
			Name: "root",
			Path: ".",
			Type: domain.ProjectTypeRoot,
		}}, nil
	}

	d.logger.Info("discovered projects", "count", len(projects))
	for _, p := range projects {
		d.logger.Debug("project", "name", p.Name, "path", p.Path, "type", p.Type)
	}
	return projects, nil
}
