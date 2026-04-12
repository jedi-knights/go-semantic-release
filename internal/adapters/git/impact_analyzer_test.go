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
		analyzer := adaptergit.NewPathBasedImpactAnalyzer(false, nil, nil)
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

		sharedCommits := []domain.Commit{
			{Hash: "xxx", FilesChanged: []string{"pkg/shared/util.go"}},
		}

		analyzer := adaptergit.NewPathBasedImpactAnalyzer(true, nil, nil)
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

		analyzer := adaptergit.NewPathBasedImpactAnalyzer(false, nil, nil)
		result := analyzer.Analyze(projectsWithDeps, sharedCommits)

		if _, ok := result["api"]; ok {
			t.Error("api should not have commits when propagation is disabled")
		}
	})

	t.Run("include paths filter", func(t *testing.T) {
		analyzer := adaptergit.NewPathBasedImpactAnalyzer(false, []string{"services/api/**"}, nil)
		result := analyzer.Analyze(projects, commits)

		if len(result["api"]) != 2 {
			t.Errorf("api commits = %d, want 2", len(result["api"]))
		}
		if len(result["worker"]) != 0 {
			t.Errorf("worker commits = %d, want 0 (filtered out)", len(result["worker"]))
		}
		if len(result["shared"]) != 0 {
			t.Errorf("shared commits = %d, want 0 (filtered out)", len(result["shared"]))
		}
	})

	t.Run("exclude paths filter", func(t *testing.T) {
		analyzer := adaptergit.NewPathBasedImpactAnalyzer(false, nil, []string{"pkg/shared/**"})
		result := analyzer.Analyze(projects, commits)

		if len(result["api"]) != 2 {
			t.Errorf("api commits = %d, want 2", len(result["api"]))
		}
		if len(result["worker"]) != 1 {
			t.Errorf("worker commits = %d, want 1", len(result["worker"]))
		}
		if len(result["shared"]) != 0 {
			t.Errorf("shared commits = %d, want 0 (excluded)", len(result["shared"]))
		}
	})

	t.Run("include and exclude combined", func(t *testing.T) {
		analyzer := adaptergit.NewPathBasedImpactAnalyzer(false, []string{"services/**"}, []string{"services/worker/**"})
		result := analyzer.Analyze(projects, commits)

		if len(result["api"]) != 2 {
			t.Errorf("api commits = %d, want 2", len(result["api"]))
		}
		if len(result["worker"]) != 0 {
			t.Errorf("worker commits = %d, want 0 (excluded)", len(result["worker"]))
		}
	})

	t.Run("file exactly matches project path", func(t *testing.T) {
		// fileInProject has a branch: `file == projectPath` (exact match without
		// the trailing "/"). This covers it by passing the project root dir itself
		// as a changed file, which is unusual but valid (e.g. a renamed directory).
		exactProjects := []domain.Project{{Name: "api", Path: "services/api"}}
		exactCommits := []domain.Commit{
			{Hash: "x1", FilesChanged: []string{"services/api"}},
		}
		analyzer := adaptergit.NewPathBasedImpactAnalyzer(false, nil, nil)
		result := analyzer.Analyze(exactProjects, exactCommits)
		if len(result["api"]) != 1 {
			t.Errorf("api commits = %d, want 1 for exact path match", len(result["api"]))
		}
	})

	t.Run("glob pattern matching", func(t *testing.T) {
		fileCommits := []domain.Commit{
			{Hash: "e1", FilesChanged: []string{"services/api/handler.go"}},
			{Hash: "e2", FilesChanged: []string{"services/api/handler.ts"}},
			{Hash: "e3", FilesChanged: []string{"docs/readme.md"}},
		}
		analyzer := adaptergit.NewPathBasedImpactAnalyzer(false, []string{"*.go"}, nil)
		result := analyzer.Analyze(projects, fileCommits)

		if len(result["api"]) != 1 {
			t.Errorf("api commits = %d, want 1 (only .go files)", len(result["api"]))
		}
	})
}
