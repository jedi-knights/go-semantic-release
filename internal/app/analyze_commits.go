package app

import (
	"context"
	"fmt"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// CommitAnalyzer analyzes commits since the last release.
type CommitAnalyzer struct {
	git    ports.GitRepository
	parser ports.CommitParser
	logger ports.Logger
}

// NewCommitAnalyzer creates a commit analyzer.
func NewCommitAnalyzer(git ports.GitRepository, parser ports.CommitParser, logger ports.Logger) *CommitAnalyzer {
	return &CommitAnalyzer{git: git, parser: parser, logger: logger}
}

// Analyze retrieves and parses commits since the given tag hash.
func (a *CommitAnalyzer) Analyze(ctx context.Context, sinceHash string) ([]domain.Commit, error) {
	rawCommits, err := a.git.CommitsSince(ctx, sinceHash)
	if err != nil {
		return nil, fmt.Errorf("fetching commits: %w", err)
	}

	a.logger.Debug("found raw commits", "count", len(rawCommits))

	parsed := make([]domain.Commit, 0, len(rawCommits))
	for _, raw := range rawCommits {
		fullMessage := raw.Message
		if raw.Body != "" {
			fullMessage = raw.Message + "\n\n" + raw.Body
		}

		commit, err := a.parser.Parse(fullMessage)
		if err != nil {
			a.logger.Warn("skipping unparseable commit", "hash", raw.Hash, "error", err)
			continue
		}

		// Preserve git metadata from raw commit.
		commit.Hash = raw.Hash
		commit.Author = raw.Author
		commit.AuthorEmail = raw.AuthorEmail
		commit.Date = raw.Date

		// Populate changed files.
		files, err := a.git.FilesChangedInCommit(ctx, raw.Hash)
		if err != nil {
			a.logger.Warn("failed to get changed files", "hash", raw.Hash, "error", err)
		}
		commit.FilesChanged = files

		parsed = append(parsed, commit)
	}

	a.logger.Info("analyzed commits", "total", len(rawCommits), "parsed", len(parsed))
	return parsed, nil
}
