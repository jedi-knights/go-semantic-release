package domain

import (
	"fmt"
	"strings"
)

// Tag represents a git tag associated with a release version.
type Tag struct {
	Name        string
	Version     Version
	Project     string // empty for repo-level tags
	Hash        string
	IsAnnotated bool
}

// TagFormat defines how tags are constructed for a project.
type TagFormat struct {
	// Template is a Go template string. Available fields: .Project, .Version
	// Examples: "{{.Project}}/v{{.Version}}", "{{.Project}}@{{.Version}}", "v{{.Version}}"
	Template string
}

// DefaultRepoTagFormat returns the default tag format for repo-level releases.
func DefaultRepoTagFormat() TagFormat {
	return TagFormat{Template: "v{{.Version}}"}
}

// DefaultProjectTagFormat returns the default tag format for project-scoped releases.
func DefaultProjectTagFormat() TagFormat {
	return TagFormat{Template: "{{.Project}}/v{{.Version}}"}
}

// ParseProjectFromTag extracts the project name and version from a tag string
// using the given prefix. Returns empty project for repo-level tags.
func ParseProjectFromTag(tagName, prefix string) (project string, version Version, err error) {
	if prefix != "" {
		if !strings.HasPrefix(tagName, prefix) {
			return "", Version{}, fmt.Errorf("tag %q does not match prefix %q", tagName, prefix)
		}
		project = strings.TrimSuffix(prefix, "/")
		tagName = strings.TrimPrefix(tagName, prefix)
	}

	// Handle @ separator (e.g., "project@1.2.3")
	if idx := strings.LastIndex(tagName, "@"); idx >= 0 && project == "" {
		project = tagName[:idx]
		tagName = tagName[idx+1:]
	}

	version, err = ParseVersion(tagName)
	if err != nil {
		return "", Version{}, fmt.Errorf("parsing version from tag %q: %w", tagName, err)
	}
	return project, version, nil
}
