package ports

import "github.com/jedi-knights/go-semantic-release/internal/domain"

// CommitParser parses raw commit messages into structured Commit objects.
type CommitParser interface {
	Parse(message string) (domain.Commit, error)
}
