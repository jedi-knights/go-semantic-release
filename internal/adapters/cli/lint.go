package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/lint"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func newLintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint",
		Short: "Lint commit messages against conventional commit rules",
		Long:  `Validate that recent commit messages follow the conventional commits specification and configured rules.`,
		RunE:  runLint,
	}
}

func runLint(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	container, err := buildContainer()
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
		return err
	}

	if len(commits) == 0 {
		fmt.Println("No commits to lint.")
		return nil
	}

	linter := lint.NewConventionalLinter(lintCfg)
	hasErrors := false
	totalViolations := 0

	for i := range commits {
		violations := linter.Lint(commits[i])
		if len(violations) == 0 {
			continue
		}

		hash := commits[i].Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		fmt.Printf("%s %s\n", hash, commits[i].Message)
		for _, v := range violations {
			icon := "⚠"
			if v.Severity == domain.LintError {
				icon = "✗"
				hasErrors = true
			}
			fmt.Printf("  %s %s: %s\n", icon, v.Rule, v.Message)
			totalViolations++
		}
		fmt.Println()
	}

	if totalViolations == 0 {
		fmt.Printf("All %d commit(s) pass lint checks.\n", len(commits))
		return nil
	}

	fmt.Printf("Found %d violation(s) in %d commit(s).\n", totalViolations, len(commits))
	if hasErrors {
		os.Exit(1)
	}
	return nil
}
