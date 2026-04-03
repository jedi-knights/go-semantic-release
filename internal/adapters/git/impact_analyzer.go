package git

import (
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// PathBasedImpactAnalyzer maps changed files to projects based on path prefixes.
type PathBasedImpactAnalyzer struct {
	propagateDeps bool
}

// NewPathBasedImpactAnalyzer creates a new path-based impact analyzer.
func NewPathBasedImpactAnalyzer(propagateDeps bool) *PathBasedImpactAnalyzer {
	return &PathBasedImpactAnalyzer{propagateDeps: propagateDeps}
}

func (a *PathBasedImpactAnalyzer) Analyze(projects []domain.Project, commits []domain.Commit) map[string][]domain.Commit {
	result := make(map[string][]domain.Commit)

	for _, commit := range commits {
		affected := a.findAffectedProjects(projects, commit.FilesChanged)
		for _, projName := range affected {
			result[projName] = append(result[projName], commit)
		}
	}

	if a.propagateDeps {
		a.propagateDependencies(projects, result)
	}

	return result
}

func (a *PathBasedImpactAnalyzer) findAffectedProjects(projects []domain.Project, files []string) []string {
	seen := make(map[string]bool)
	var affected []string

	for _, file := range files {
		for _, proj := range projects {
			if seen[proj.Name] {
				continue
			}
			if proj.IsRoot() || fileInProject(file, proj.Path) {
				seen[proj.Name] = true
				affected = append(affected, proj.Name)
			}
		}
	}
	return affected
}

func fileInProject(file, projectPath string) bool {
	if projectPath == "" || projectPath == "." {
		return true
	}
	prefix := projectPath + "/"
	return strings.HasPrefix(file, prefix) || file == projectPath
}

func (a *PathBasedImpactAnalyzer) propagateDependencies(projects []domain.Project, result map[string][]domain.Commit) {
	projectMap := make(map[string]domain.Project, len(projects))
	for _, p := range projects {
		projectMap[p.Name] = p
	}

	// Simple single-pass propagation: if a dependency has commits, mark dependents.
	for _, proj := range projects {
		for _, dep := range proj.Dependencies {
			if commits, ok := result[dep]; ok && len(commits) > 0 {
				if _, alreadyAffected := result[proj.Name]; !alreadyAffected {
					// Propagate with a synthetic marker — use the dependency's commits.
					result[proj.Name] = commits
				}
			}
		}
	}
}
