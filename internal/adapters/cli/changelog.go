package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func newChangelogCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "changelog",
		Short: "Generate changelog for the next release",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChangelog(cmd, args, opts)
		},
	}
}

func runChangelog(cmd *cobra.Command, _ []string, opts *rootOptions) error {
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

	out := cmd.OutOrStdout()
	gen := container.ChangelogGenerator()
	releasable := plan.ReleasableProjects()
	for i := range releasable {
		notes, err := gen.Generate(releasable[i].NextVersion, releasable[i].Project.Name, releasable[i].Commits, cfg.ChangelogSections)
		if err != nil {
			return fmt.Errorf("generating changelog for %s: %w", releasable[i].Project.Name, err)
		}
		fmt.Fprintln(out, notes)
		fmt.Fprintln(out)
	}

	if !plan.HasReleasableProjects() {
		fmt.Fprintln(out, "No releasable changes found.")
	}

	return nil
}
