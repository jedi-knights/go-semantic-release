package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the current and next version",
		RunE:  runVersion,
	}
}

func runVersion(cmd *cobra.Command, _ []string) error {
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

	for i := range plan.Projects {
		name := displayProjectName(plan.Projects[i].Project)
		if plan.Projects[i].ShouldRelease {
			fmt.Printf("%s: %s → %s\n", name, plan.Projects[i].CurrentVersion, plan.Projects[i].NextVersion)
		} else {
			fmt.Printf("%s: %s (no change)\n", name, plan.Projects[i].CurrentVersion)
		}
	}
	return nil
}
