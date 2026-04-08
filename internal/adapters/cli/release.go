package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/prompt"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// runRelease is the default action when semantic-release is invoked without a subcommand.
// This matches the original semantic-release behavior.
func runRelease(cmd *cobra.Command, _ []string, opts *rootOptions) error {
	ctx := cmd.Context()

	container, workDir, err := buildContainerWithWorkDir(opts)
	if err != nil {
		return err
	}

	cfg := container.Config()

	// Discover projects.
	projects, err := container.ProjectDetector().Detect(ctx, workDir)
	if err != nil {
		return fmt.Errorf("detecting projects: %w", err)
	}

	// Filter to specific project if requested.
	if opts.project != "" {
		projects = filterProject(projects, opts.project)
		if len(projects) == 0 {
			return fmt.Errorf("project %q not found", opts.project)
		}
	}

	// Analyze commits.
	commits, err := container.CommitAnalyzer().Analyze(ctx, "")
	if err != nil {
		return fmt.Errorf("analyzing commits: %w", err)
	}

	// Build release plan.
	branch, err := container.GitRepository().CurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("resolving current branch: %w", err)
	}
	policy := domain.FindBranchPolicy(cfg.Branches, branch)

	plan, err := container.ReleasePlanner().Plan(ctx, projects, commits, cfg.ReleaseMode, policy, cfg.DryRun)
	if err != nil {
		return fmt.Errorf("planning release: %w", err)
	}

	if !plan.HasReleasableProjects() {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No releasable changes found.")
		return nil
	}

	// Warn when dry-run was automatically engaged because we are not in CI.
	// This fires only for the release command — plan/lint/verify/etc. are unaffected.
	if cfg.DryRun && !opts.dryRun && !cfg.CI {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "note: not running in CI — defaulting to dry run (pass --no-ci to override)")
	}

	// Interactive confirmation before release.
	if shouldPrompt(cfg, opts) {
		if planErr := printPlan(cmd.OutOrStdout(), plan, opts.jsonOut); planErr != nil {
			return planErr
		}
		prompter := prompt.NewTerminalPrompter()
		confirmed, promptErr := prompter.Confirm("Proceed with release?")
		if promptErr != nil {
			return fmt.Errorf("reading confirmation: %w", promptErr)
		}
		if !confirmed {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Release cancelled.")
			return nil
		}
	}

	// Resolve plugins once before the loop. container.Plugins() uses sync.Once so
	// the build cost is incurred at most once, but the error check and call itself
	// belong outside the per-project loop for clarity.
	allPlugins, pluginsErr := container.Plugins()
	if pluginsErr != nil {
		return fmt.Errorf("loading plugins: %w", pluginsErr)
	}

	// Run prepare step (update CHANGELOG.md, VERSION, etc.) for each releasable project.
	//
	// Note: changelog notes are generated here for the prepare plugins and again
	// inside executeProject for the tag annotation and publish payload. Both calls
	// use the same generator with identical inputs so the output is deterministic.
	// The duplication avoids coupling the executor to the prepare layer; a future
	// refactor could pass notes through ReleaseContext to eliminate the second call.
	gen := container.ChangelogGenerator()
	releasable := plan.ReleasableProjects()
	for i := range releasable {
		notes, notesErr := gen.Generate(releasable[i].NextVersion, releasable[i].Project.Name, releasable[i].Commits, cfg.ChangelogSections)
		if notesErr != nil {
			return fmt.Errorf("generating notes: %w", notesErr)
		}

		rc := &domain.ReleaseContext{
			Config:         cfg,
			Branch:         branch,
			BranchPolicy:   policy,
			DryRun:         cfg.DryRun,
			CI:             cfg.CI,
			RepositoryRoot: workDir,
			CurrentProject: &releasable[i],
			Notes:          notes,
		}

		for _, plugin := range allPlugins {
			// Use the canonical ports.PreparePlugin interface so that any signature
			// change to the port is caught at compile time rather than silently
			// skipping the prepare step at runtime.
			if pp, ok := plugin.(ports.PreparePlugin); ok {
				if prepErr := pp.Prepare(ctx, rc); prepErr != nil {
					return fmt.Errorf("prepare step: %w", prepErr)
				}
			}
		}
	}

	// Execute release (create tags, push, publish).
	result, err := container.ReleaseExecutor().Execute(ctx, plan)
	if err != nil {
		return fmt.Errorf("executing release: %w", err)
	}

	return printReleaseResult(cmd.OutOrStdout(), cmd.ErrOrStderr(), result, opts.jsonOut)
}

// printReleaseResult renders the release result to w (stdout) and errW (stderr).
// asJSON is passed explicitly rather than read from the package-level jsonOut flag
// so callers can exercise both rendering paths in tests without mutating shared state.
func printReleaseResult(w, errW io.Writer, result *domain.ReleaseResult, asJSON bool) error {
	if asJSON {
		return json.NewEncoder(w).Encode(result)
	}

	for i := range result.Projects {
		if result.Projects[i].Skipped {
			// Use SkipReason from the result so the label is accurate regardless of
			// why the project was skipped (dry run, policy gate, etc.).
			_, _ = fmt.Fprintf(w, "[%s] %s: %s → %s (tag: %s)\n",
				result.Projects[i].SkipReason,
				projectName(result.Projects[i]),
				result.Projects[i].CurrentVersion.String(),
				result.Projects[i].Version.String(),
				result.Projects[i].TagName)
			continue
		}
		if result.Projects[i].Error != nil {
			_, _ = fmt.Fprintf(errW, "ERROR %s: %v\n", projectName(result.Projects[i]), result.Projects[i].Error)
			continue
		}
		_, _ = fmt.Fprintf(w, "Released %s %s (tag: %s)\n", projectName(result.Projects[i]), result.Projects[i].Version, result.Projects[i].TagName)
		if result.Projects[i].PublishURL != "" {
			_, _ = fmt.Fprintf(w, "  → %s\n", result.Projects[i].PublishURL)
		}
	}

	// Per-project errors were already printed to stderr above. Return ErrQuietExit
	// so main exits with code 1 without duplicating the error messages.
	if result.HasErrors() {
		return ErrQuietExit
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

// shouldPrompt determines whether to show an interactive confirmation.
// It uses cfg.CI (already resolved by applyFlagAndEnvOverrides) rather than
// re-inspecting environment variables, keeping CI detection in one place.
func shouldPrompt(cfg domain.Config, opts *rootOptions) bool {
	if opts.noInteractive {
		return false
	}
	if opts.interactive {
		return true
	}
	if cfg.Interactive != nil {
		return *cfg.Interactive
	}
	// Auto-detect: prompt when running locally in a terminal, not in CI.
	return !cfg.CI && prompt.IsTerminal()
}
