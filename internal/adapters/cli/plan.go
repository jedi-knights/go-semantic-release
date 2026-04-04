package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func newPlanCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show the release plan without executing",
		Long:  `Analyze commits and show what would happen during a release, including version bumps and affected projects.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlan(cmd, args, opts)
		},
	}
}

func runPlan(cmd *cobra.Command, _ []string, opts *rootOptions) error {
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

	// true = dry-run: plan only previews what would be released, never executes.
	plan, err := container.ReleasePlanner().Plan(ctx, projects, commits, cfg.ReleaseMode, policy, true)
	if err != nil {
		return fmt.Errorf("planning release: %w", err)
	}

	return printPlan(cmd.OutOrStdout(), plan, opts.jsonOut)
}

// printPlan renders the release plan to w. asJSON controls whether output is
// JSON-encoded; it is passed explicitly rather than read from the package-level
// jsonOut flag so callers can exercise both rendering paths in tests without
// mutating shared state.
func printPlan(w io.Writer, plan *domain.ReleasePlan, asJSON bool) error {
	if asJSON {
		return json.NewEncoder(w).Encode(plan)
	}

	fmt.Fprintf(w, "Branch: %s\n", plan.Branch)
	fmt.Fprintf(w, "Release mode: %s\n\n", modeString(plan))

	if !plan.HasReleasableProjects() {
		fmt.Fprintln(w, "No releasable changes found.")
		return nil
	}

	for i := range plan.Projects {
		status := "skip"
		if plan.Projects[i].ShouldRelease {
			status = "release"
		}
		fmt.Fprintf(w, "  %s [%s]\n", displayProjectName(plan.Projects[i].Project), status)
		fmt.Fprintf(w, "    Current: %s\n", plan.Projects[i].CurrentVersion)
		if plan.Projects[i].ShouldRelease {
			fmt.Fprintf(w, "    Next:    %s (%s)\n", plan.Projects[i].NextVersion, plan.Projects[i].ReleaseType)
		}
		fmt.Fprintf(w, "    Commits: %d\n", len(plan.Projects[i].Commits))
		fmt.Fprintf(w, "    Reason:  %s\n\n", plan.Projects[i].Reason)
	}
	return nil
}

func modeString(plan *domain.ReleasePlan) string {
	if plan.Policy != nil && plan.Policy.Prerelease {
		return fmt.Sprintf("prerelease (%s)", plan.Policy.Channel)
	}
	return "stable"
}

func displayProjectName(p domain.Project) string {
	if p.Name == "" || p.Name == "root" {
		return "(repository)"
	}
	return p.Name
}
