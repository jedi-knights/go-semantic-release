package plugins

import (
	"context"
	"fmt"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// LintPlugin implements VerifyReleasePlugin by linting commit messages.
type LintPlugin struct {
	linter ports.CommitLinter
	logger ports.Logger
}

// NewLintPlugin creates a commit linting plugin.
func NewLintPlugin(linter ports.CommitLinter, logger ports.Logger) *LintPlugin {
	return &LintPlugin{linter: linter, logger: logger}
}

func (p *LintPlugin) Name() string { return "commit-lint" }

// VerifyRelease lints all commits and returns an error if any have error-severity violations.
func (p *LintPlugin) VerifyRelease(_ context.Context, rc *domain.ReleaseContext) error {
	var allErrors []string

	for i := range rc.Commits {
		violations := p.linter.Lint(rc.Commits[i])
		for _, v := range violations {
			msg := fmt.Sprintf("%s (%s): %s [%s]", rc.Commits[i].Hash[:minInt(7, len(rc.Commits[i].Hash))], v.Severity, v.Message, v.Rule)
			if v.Severity == domain.LintError {
				allErrors = append(allErrors, msg)
			} else {
				p.logger.Warn("lint warning", "commit", rc.Commits[i].Hash, "rule", v.Rule, "message", v.Message)
			}
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("commit lint errors:\n  %s", strings.Join(allErrors, "\n  "))
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
