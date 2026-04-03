package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func newPlanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show the release plan without executing",
		Long:  `Analyze commits and show what would happen during a release, including version bumps and affected projects.`,
		RunE:  runPlan,
	}
}

func runPlan(cmd *cobra.Command, _ []string) error {
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

	return printPlan(plan)
}

func printPlan(plan *domain.ReleasePlan) error {
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(plan)
	}

	fmt.Printf("Branch: %s\n", plan.Branch)
	fmt.Printf("Release mode: %s\n\n", modeString(plan))

	if !plan.HasReleasableProjects() {
		fmt.Println("No releasable changes found.")
		return nil
	}

	for i := range plan.Projects {
		status := "skip"
		if plan.Projects[i].ShouldRelease {
			status = "release"
		}
		fmt.Printf("  %s [%s]\n", displayProjectName(plan.Projects[i].Project), status)
		fmt.Printf("    Current: %s\n", plan.Projects[i].CurrentVersion)
		if plan.Projects[i].ShouldRelease {
			fmt.Printf("    Next:    %s (%s)\n", plan.Projects[i].NextVersion, plan.Projects[i].ReleaseType)
		}
		fmt.Printf("    Commits: %d\n", len(plan.Projects[i].Commits))
		fmt.Printf("    Reason:  %s\n\n", plan.Projects[i].Reason)
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
