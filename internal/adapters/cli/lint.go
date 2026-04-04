package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/lint"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func newLintCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "lint",
		Short: "Lint commit messages against conventional commit rules",
		Long:  `Validate that recent commit messages follow the conventional commits specification and configured rules.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLint(cmd, args, opts)
		},
	}
}

func runLint(cmd *cobra.Command, _ []string, opts *rootOptions) error {
	ctx := cmd.Context()

	container, _, err := buildContainerWithWorkDir(opts)
	if err != nil {
		return err
	}

	cfg := container.Config()
	lintCfg := cfg.Lint
	if len(lintCfg.AllowedTypes) == 0 {
		lintCfg = domain.DefaultLintConfig()
	}

	commits, err := container.CommitAnalyzer().Analyze(ctx, "")
	if err != nil {
		return fmt.Errorf("analyzing commits: %w", err)
	}

	if len(commits) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No commits to lint.")
		return nil
	}

	linter := lint.NewConventionalLinter(lintCfg)
	totalViolations := 0
	hasErrors := false

	for i := range commits {
		violations := linter.Lint(commits[i])
		if len(violations) == 0 {
			continue
		}

		hash := commits[i].Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		// Per-commit violation details go to stderr so they do not pollute
		// piped output. The clean-pass summary below goes to stdout.
		fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n", hash, commits[i].Message)
		for _, v := range violations {
			totalViolations++
			icon := "[WARN]"
			if v.Severity == domain.LintError {
				icon = "[ERROR]"
				hasErrors = true
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s: %s\n", icon, v.Rule, v.Message)
		}
		fmt.Fprintln(cmd.ErrOrStderr())
	}

	if totalViolations == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "All %d commit(s) pass lint checks.\n", len(commits))
		return nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Found %d violation(s) in %d commit(s).\n", totalViolations, len(commits))
	if hasErrors {
		// Violations already printed above; return ErrQuietExit so main exits
		// with code 1 without printing a redundant error message.
		return ErrQuietExit
	}
	return nil
}
