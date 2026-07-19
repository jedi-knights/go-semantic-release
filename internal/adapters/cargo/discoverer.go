package cargo

import (
	"context"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.ProjectDiscoverer = (*RustDiscoverer)(nil)

// RustDiscoverer discovers a Rust project from a root Cargo.toml. It reports a
// single repo-root project (repo release mode): a Cargo workspace shares one
// [workspace.package] version, so there is one release per repository rather
// than one per crate. It returns no projects (nil, nil) when the repository is
// not a Cargo project, so it is harmless to run against non-Rust repos.
type RustDiscoverer struct {
	fs ports.FileSystem
}

// NewRustDiscoverer creates a discoverer for Rust/Cargo repositories.
func NewRustDiscoverer(fsys ports.FileSystem) *RustDiscoverer {
	return &RustDiscoverer{fs: fsys}
}

// Discover returns a single root project typed cargo-crate or cargo-workspace,
// or nil when rootPath is not a Cargo project. The context is unused because
// detection is a bounded, local filesystem read.
func (d *RustDiscoverer) Discover(_ context.Context, rootPath string) ([]domain.Project, error) {
	info, err := Detect(d.fs, rootPath)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	projType := domain.ProjectTypeCargoCrate
	if info.Kind == KindWorkspace {
		projType = domain.ProjectTypeCargoWorkspace
	}

	// Name is intentionally empty (like the repo-root fallback project): a
	// repo-mode release with an empty project name renders the unprefixed tag
	// format ("v1.2.3") rather than the project-scoped one ("name/v1.2.3"). The
	// Type still identifies the repo as Rust for `detect-projects`.
	return []domain.Project{{
		Name:      "",
		Path:      ".",
		Type:      projType,
		TagPrefix: "",
	}}, nil
}
