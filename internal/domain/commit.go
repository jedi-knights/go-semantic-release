package domain

import "time"

// Commit represents a parsed git commit with conventional commit metadata.
type Commit struct {
	Hash             string
	Message          string
	Author           string
	AuthorEmail      string
	Date             time.Time
	Type             string // e.g. "feat", "fix", "chore"
	Scope            string
	Description      string
	Body             string
	Footer           string
	IsBreakingChange bool
	BreakingNote     string
	FilesChanged     []string
}

// ReleaseType returns the release type implied by this commit.
func (c Commit) ReleaseType(typeMapping map[string]ReleaseType) ReleaseType {
	if c.IsBreakingChange {
		return ReleaseMajor
	}
	if rt, ok := typeMapping[c.Type]; ok {
		return rt
	}
	return ReleaseNone
}

// DefaultCommitTypeMapping returns the standard mapping from conventional commit types to release types.
func DefaultCommitTypeMapping() map[string]ReleaseType {
	return map[string]ReleaseType{
		"feat":     ReleaseMinor,
		"fix":      ReleasePatch,
		"perf":     ReleasePatch,
		"revert":   ReleasePatch,
		"refactor": ReleaseNone,
		"docs":     ReleaseNone,
		"style":    ReleaseNone,
		"test":     ReleaseNone,
		"build":    ReleaseNone,
		"ci":       ReleaseNone,
		"chore":    ReleaseNone,
	}
}
