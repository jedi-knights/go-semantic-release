package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func newChangelogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "changelog",
		Short: "Generate changelog for the next release",
		RunE:  runChangelog,
	}
}

func runChangelog(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	container, err := buildContainer()
	if err != nil {
		return err
	}

	cfg := container.Config()

	projects, err := container.ProjectDetector().Detect(ctx, getWorkDir())
	if err != nil {
		return err
	}

	if project != "" {
		projects = filterProject(projects, project)
	}

	commits, err := container.CommitAnalyzer().Analyze(ctx, "")
	if err != nil {
		return err
	}

	branch, _ := container.GitRepository().CurrentBranch(ctx)
	policy := domain.FindBranchPolicy(cfg.Branches, branch)

	plan, err := container.ReleasePlanner().Plan(ctx, projects, commits, cfg.ReleaseMode, policy, true)
	if err != nil {
		return err
	}

	gen := container.ChangelogGenerator()
	for _, pp := range plan.ReleasableProjects() {
		notes, err := gen.Generate(pp.NextVersion, pp.Project.Name, pp.Commits, cfg.ChangelogSections)
		if err != nil {
			return fmt.Errorf("generating changelog for %s: %w", pp.Project.Name, err)
		}
		fmt.Println(notes)
		fmt.Println()
	}

	if !plan.HasReleasableProjects() {
		fmt.Println("No releasable changes found.")
	}

	return nil
}
