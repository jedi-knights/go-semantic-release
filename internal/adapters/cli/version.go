package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func newVersionCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the current and next version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd, args, opts)
		},
	}
}

func runVersion(cmd *cobra.Command, _ []string, opts *rootOptions) error {
	ctx := cmd.Context()

	container, workDir, err := buildContainerWithWorkDir(opts)
	if err != nil {
		return err
	}

	cfg := container.Config()

	projects, err := container.ProjectDetector().Detect(ctx, workDir)
	if err != nil {
		return fmt.Errorf("detecting projects: %w", err)
	}

	if opts.project != "" {
		projects = filterProject(projects, opts.project)
		if len(projects) == 0 {
			return fmt.Errorf("project %q not found", opts.project)
		}
	}

	commits, err := container.CommitAnalyzer().Analyze(ctx, "")
	if err != nil {
		return fmt.Errorf("analyzing commits: %w", err)
	}

	branch, err := container.GitRepository().CurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("resolving current branch: %w", err)
	}
	policy := domain.FindBranchPolicy(cfg.Branches, branch)

	plan, err := container.ReleasePlanner().Plan(ctx, projects, commits, cfg.ReleaseMode, policy, true)
	if err != nil {
		return fmt.Errorf("planning release: %w", err)
	}

	for i := range plan.Projects {
		name := displayProjectName(plan.Projects[i].Project)
		if plan.Projects[i].ShouldRelease {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s → %s\n", name, plan.Projects[i].CurrentVersion, plan.Projects[i].NextVersion)
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s (no change)\n", name, plan.Projects[i].CurrentVersion)
		}
	}
	return nil
}
