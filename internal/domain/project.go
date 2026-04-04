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
	// ProjectTypeGoWorkspace indicates the project was discovered via a go.work file.
	ProjectTypeGoWorkspace ProjectType = "go-workspace"
	// ProjectTypeGoModule indicates the project was discovered by finding a go.mod file.
	ProjectTypeGoModule ProjectType = "go-module"
	// ProjectTypeConfigured indicates the project was defined explicitly in the release config.
	ProjectTypeConfigured ProjectType = "configured"
	// ProjectTypeRoot indicates the project is at the repository root (no path prefix on tags).
	ProjectTypeRoot ProjectType = "root"
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
	Name string
	// Path is the relative path from the repository root (e.g. "services/auth-server").
	// It should always be a relative path; IsRoot handles ".", "", and "/" defensively.
	Path          string
	Type          ProjectType
	ModulePath    string   // Go module path if applicable
	Dependencies  []string // names of projects this depends on
	TagPrefix     string   // e.g. "myproject/" for tags like "myproject/v1.2.3"
	ChangelogFile string   // per-project changelog filename, relative to the project's path; empty means use global
}

// IsRoot returns true if this project represents the repository root.
// The "/" case is defensive — Path should always be a relative path ("." or ""),
// but we guard against an absolute-path misuse rather than silently misbehaving.
func (p Project) IsRoot() bool {
	return p.Path == "." || p.Path == "" || p.Path == "/"
}
