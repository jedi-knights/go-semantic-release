package domain

// Changelog represents generated release notes for a version.
type Changelog struct {
	Title    string
	Version  Version
	Project  string // empty for repo-level
	Sections []ChangelogSection
}

// ChangelogSection groups commits by type.
type ChangelogSection struct {
	Title   string // e.g. "Features", "Bug Fixes", "Breaking Changes"
	Type    string // commit type key
	Commits []Commit
}

// DefaultChangelogSections defines the standard section ordering and titles.
func DefaultChangelogSections() []ChangelogSectionConfig {
	return []ChangelogSectionConfig{
		{Type: "breaking", Title: "Breaking Changes", Hidden: false},
		{Type: "feat", Title: "Features", Hidden: false},
		{Type: "fix", Title: "Bug Fixes", Hidden: false},
		{Type: "perf", Title: "Performance Improvements", Hidden: false},
		{Type: "revert", Title: "Reverts", Hidden: false},
		{Type: "refactor", Title: "Code Refactoring", Hidden: true},
		{Type: "docs", Title: "Documentation", Hidden: true},
		{Type: "style", Title: "Styles", Hidden: true},
		{Type: "test", Title: "Tests", Hidden: true},
		{Type: "build", Title: "Build System", Hidden: true},
		{Type: "ci", Title: "Continuous Integration", Hidden: true},
		{Type: "chore", Title: "Chores", Hidden: true},
	}
}

// ChangelogSectionConfig controls which sections appear in changelogs.
type ChangelogSectionConfig struct {
	Type   string
	Title  string
	Hidden bool // if true, section is omitted from output unless explicitly requested
}
