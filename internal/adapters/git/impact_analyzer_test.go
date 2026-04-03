package git_test

import (
	"testing"

	adaptergit "github.com/jedi-knights/go-semantic-release/internal/adapters/git"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestPathBasedImpactAnalyzer_Analyze(t *testing.T) {
	projects := []domain.Project{
		{Name: "api", Path: "services/api"},
		{Name: "worker", Path: "services/worker"},
		{Name: "shared", Path: "pkg/shared"},
	}

	commits := []domain.Commit{
		{Hash: "aaa", FilesChanged: []string{"services/api/handler.go", "services/api/routes.go"}},
		{Hash: "bbb", FilesChanged: []string{"services/worker/main.go"}},
		{Hash: "ccc", FilesChanged: []string{"pkg/shared/util.go"}},
		{Hash: "ddd", FilesChanged: []string{"services/api/config.go", "pkg/shared/types.go"}},
	}

	t.Run("basic mapping", func(t *testing.T) {
		analyzer := adaptergit.NewPathBasedImpactAnalyzer(false)
		result := analyzer.Analyze(projects, commits)

		if len(result["api"]) != 2 {
			t.Errorf("api commits = %d, want 2", len(result["api"]))
		}
		if len(result["worker"]) != 1 {
			t.Errorf("worker commits = %d, want 1", len(result["worker"]))
		}
		if len(result["shared"]) != 2 {
			t.Errorf("shared commits = %d, want 2", len(result["shared"]))
		}
	})

	t.Run("with dependency propagation", func(t *testing.T) {
		projectsWithDeps := []domain.Project{
			{Name: "api", Path: "services/api", Dependencies: []string{"shared"}},
			{Name: "worker", Path: "services/worker", Dependencies: []string{"shared"}},
			{Name: "shared", Path: "pkg/shared"},
		}

		// Only shared changes.
		sharedCommits := []domain.Commit{
			{Hash: "xxx", FilesChanged: []string{"pkg/shared/util.go"}},
		}

		analyzer := adaptergit.NewPathBasedImpactAnalyzer(true)
		result := analyzer.Analyze(projectsWithDeps, sharedCommits)

		if len(result["shared"]) != 1 {
			t.Errorf("shared commits = %d, want 1", len(result["shared"]))
		}
		if len(result["api"]) != 1 {
			t.Errorf("api should have propagated commit, got %d", len(result["api"]))
		}
		if len(result["worker"]) != 1 {
			t.Errorf("worker should have propagated commit, got %d", len(result["worker"]))
		}
	})

	t.Run("no propagation when disabled", func(t *testing.T) {
		projectsWithDeps := []domain.Project{
			{Name: "api", Path: "services/api", Dependencies: []string{"shared"}},
			{Name: "shared", Path: "pkg/shared"},
		}

		sharedCommits := []domain.Commit{
			{Hash: "xxx", FilesChanged: []string{"pkg/shared/util.go"}},
		}

		analyzer := adaptergit.NewPathBasedImpactAnalyzer(false)
		result := analyzer.Analyze(projectsWithDeps, sharedCommits)

		if _, ok := result["api"]; ok {
			t.Error("api should not have commits when propagation is disabled")
		}
	})
}
