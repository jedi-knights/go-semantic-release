package domain

// ReleaseMode defines whether the repository is released as a whole or per-project.
type ReleaseMode string

const (
	// ReleaseModeRepo performs a single release for the entire repository.
	ReleaseModeRepo ReleaseMode = "repo"
	// ReleaseModeIndependent versions and releases each project independently.
	ReleaseModeIndependent ReleaseMode = "independent"
)

// ProjectType describes how a project was discovered.
type ProjectType string

const (
	ProjectTypeGoWorkspace ProjectType = "go-workspace"
	ProjectTypeGoModule    ProjectType = "go-module"
	ProjectTypeConfigured  ProjectType = "configured"
	ProjectTypeRoot        ProjectType = "root"
)

// String returns the string representation of the release mode.
func (m ReleaseMode) String() string {
	return string(m)
}

// String returns the string representation of the project type.
func (pt ProjectType) String() string {
	return string(pt)
}

// Project represents a versioned unit within a repository.
type Project struct {
	Name         string
	Path         string // relative path from repo root
	Type         ProjectType
	ModulePath   string   // Go module path if applicable
	Dependencies []string // names of projects this depends on
	TagPrefix    string   // e.g. "myproject/" for tags like "myproject/v1.2.3"
}

// IsRoot returns true if this project represents the repository root.
func (p Project) IsRoot() bool {
	return p.Path == "." || p.Path == ""
}
