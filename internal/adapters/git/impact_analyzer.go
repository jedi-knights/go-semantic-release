package git

import (
	"path/filepath"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.ProjectImpactAnalyzer = (*PathBasedImpactAnalyzer)(nil)

// PathBasedImpactAnalyzer maps changed files to projects based on path prefixes.
type PathBasedImpactAnalyzer struct {
	propagateDeps bool
	includePaths  []string
	excludePaths  []string
}

// NewPathBasedImpactAnalyzer creates a new path-based impact analyzer.
func NewPathBasedImpactAnalyzer(propagateDeps bool, includePaths, excludePaths []string) *PathBasedImpactAnalyzer {
	return &PathBasedImpactAnalyzer{
		propagateDeps: propagateDeps,
		includePaths:  includePaths,
		excludePaths:  excludePaths,
	}
}

func (a *PathBasedImpactAnalyzer) Analyze(projects []domain.Project, commits []domain.Commit) map[string][]domain.Commit {
	result := make(map[string][]domain.Commit)

	for i := range commits {
		affected := a.findAffectedProjects(projects, commits[i].FilesChanged)
		for _, projName := range affected {
			result[projName] = append(result[projName], commits[i])
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

	filtered := a.filterFiles(files)
	for _, file := range filtered {
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

// filterFiles applies include/exclude glob patterns to the file list.
func (a *PathBasedImpactAnalyzer) filterFiles(files []string) []string {
	if len(a.includePaths) == 0 && len(a.excludePaths) == 0 {
		return files
	}

	result := make([]string, 0, len(files))
	for _, file := range files {
		if len(a.includePaths) > 0 && !matchesAny(file, a.includePaths) {
			continue
		}
		if matchesAny(file, a.excludePaths) {
			continue
		}
		result = append(result, file)
	}
	return result
}

// matchesAny returns true if the file matches any of the glob patterns.
func matchesAny(file string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, file); matched {
			return true
		}
		// Also try matching against just the filename for simple patterns.
		if matched, _ := filepath.Match(pattern, filepath.Base(file)); matched {
			return true
		}
		// Support prefix-based patterns like "services/api/**" by checking prefix.
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if strings.HasPrefix(file, prefix+"/") || file == prefix {
				return true
			}
		}
	}
	return false
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
