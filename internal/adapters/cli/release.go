package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func newReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release",
		Short: "Perform a semantic release",
		Long: `Analyze commits, calculate the next version, generate release notes,
create git tags, and publish a release. Use --dry-run to preview without mutations.`,
		RunE: runRelease,
	}
}

func runRelease(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	container, err := buildContainer()
	if err != nil {
		return err
	}

	cfg := container.Config()

	// Discover projects.
	projects, err := container.ProjectDetector().Detect(ctx, getWorkDir())
	if err != nil {
		return fmt.Errorf("detecting projects: %w", err)
	}

	// Filter to specific project if requested.
	if project != "" {
		projects = filterProject(projects, project)
		if len(projects) == 0 {
			return fmt.Errorf("project %q not found", project)
		}
	}

	// Analyze commits.
	commits, err := container.CommitAnalyzer().Analyze(ctx, "")
	if err != nil {
		return fmt.Errorf("analyzing commits: %w", err)
	}

	// Build release plan.
	branch, _ := container.GitRepository().CurrentBranch(ctx)
	policy := domain.FindBranchPolicy(cfg.Branches, branch)

	plan, err := container.ReleasePlanner().Plan(ctx, projects, commits, cfg.ReleaseMode, policy, cfg.DryRun)
	if err != nil {
		return fmt.Errorf("planning release: %w", err)
	}

	if !plan.HasReleasableProjects() {
		fmt.Fprintln(os.Stdout, "No releasable changes found.")
		return nil
	}

	// Execute release.
	result, err := container.ReleaseExecutor().Execute(ctx, plan)
	if err != nil {
		return fmt.Errorf("executing release: %w", err)
	}

	return printReleaseResult(result)
}

func printReleaseResult(result *domain.ReleaseResult) error {
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	for _, pr := range result.Projects {
		if pr.Skipped {
			fmt.Printf("[dry-run] %s: %s → %s (tag: %s)\n",
				projectName(pr), pr.Version.String(), pr.Version.String(), pr.TagName)
			continue
		}
		if pr.Error != nil {
			fmt.Fprintf(os.Stderr, "ERROR %s: %v\n", projectName(pr), pr.Error)
			continue
		}
		fmt.Printf("Released %s %s (tag: %s)\n", projectName(pr), pr.Version, pr.TagName)
		if pr.PublishURL != "" {
			fmt.Printf("  → %s\n", pr.PublishURL)
		}
	}
	return nil
}

func projectName(pr domain.ProjectReleaseResult) string {
	if pr.Project.Name == "" {
		return "repo"
	}
	return pr.Project.Name
}

func filterProject(projects []domain.Project, name string) []domain.Project {
	for _, p := range projects {
		if p.Name == name {
			return []domain.Project{p}
		}
	}
	return nil
}

func getWorkDir() string {
	dir, _ := os.Getwd()
	return dir
}
