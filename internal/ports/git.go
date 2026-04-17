package ports

import (
	"context"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// GitRepository provides access to git operations.
type GitRepository interface {
	// CurrentBranch returns the name of the checked-out branch.
	CurrentBranch(ctx context.Context) (string, error)

	// ListTags returns all tags in the repository.
	ListTags(ctx context.Context) ([]domain.Tag, error)

	// CommitsSince returns commits since the given tag hash (exclusive).
	// If sinceHash is empty, returns all commits on the current branch.
	CommitsSince(ctx context.Context, sinceHash string) ([]domain.Commit, error)

	// FilesChangedInCommit returns the list of files changed by a commit.
	FilesChangedInCommit(ctx context.Context, hash string) ([]string, error)

	// CreateTag creates a git tag at the given commit hash.
	CreateTag(ctx context.Context, name string, hash string, message string) error

	// PushTag pushes a tag to the remote.
	PushTag(ctx context.Context, name string) error

	// HeadHash returns the hash of HEAD.
	HeadHash(ctx context.Context) (string, error)

	// RemoteURL returns the remote origin URL.
	RemoteURL(ctx context.Context) (string, error)

	// Stage adds the given file paths to the git index.
	Stage(ctx context.Context, files []string) error

	// Commit creates a commit with the given message using the staged index.
	Commit(ctx context.Context, message string) error

	// Push pushes the current branch to origin.
	Push(ctx context.Context) error
}
