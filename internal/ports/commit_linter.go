package ports

import "github.com/jedi-knights/go-semantic-release/internal/domain"

// CommitLinter validates commit messages against configured rules.
type CommitLinter interface {
	Lint(commit domain.Commit) []domain.LintViolation
}
