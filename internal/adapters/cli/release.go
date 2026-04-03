package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/prompt"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/platform"
)

// runRelease is the default action when semantic-release is invoked without a subcommand.
// This matches the original semantic-release behavior.
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

	// Interactive confirmation before release.
	if shouldPrompt(cfg) {
		if planErr := printPlan(plan); planErr != nil {
			return planErr
		}
		prompter := prompt.NewTerminalPrompter()
		confirmed, promptErr := prompter.Confirm("Proceed with release?")
		if promptErr != nil {
			return fmt.Errorf("reading confirmation: %w", promptErr)
		}
		if !confirmed {
			fmt.Println("Release cancelled.")
			return nil
		}
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

	for i := range result.Projects {
		if result.Projects[i].Skipped {
			fmt.Printf("[dry-run] %s: %s → %s (tag: %s)\n",
				projectName(result.Projects[i]), result.Projects[i].Version.String(), result.Projects[i].Version.String(), result.Projects[i].TagName)
			continue
		}
		if result.Projects[i].Error != nil {
			fmt.Fprintf(os.Stderr, "ERROR %s: %v\n", projectName(result.Projects[i]), result.Projects[i].Error)
			continue
		}
		fmt.Printf("Released %s %s (tag: %s)\n", projectName(result.Projects[i]), result.Projects[i].Version, result.Projects[i].TagName)
		if result.Projects[i].PublishURL != "" {
			fmt.Printf("  → %s\n", result.Projects[i].PublishURL)
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

// shouldPrompt determines whether to show an interactive confirmation.
func shouldPrompt(cfg domain.Config) bool {
	if noInteractive {
		return false
	}
	if interactive {
		return true
	}
	if cfg.Interactive != nil {
		return *cfg.Interactive
	}
	// Auto-detect: prompt when running locally in a terminal, not in CI.
	return !platform.IsCI() && prompt.IsTerminal()
}
