package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.GitRepository = (*Repository)(nil)

// Repository implements ports.GitRepository using the git CLI.
type Repository struct {
	workDir string
}

// NewRepository creates a new git CLI adapter.
func NewRepository(workDir string) *Repository {
	return &Repository{workDir: workDir}
}

func (r *Repository) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.workDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (r *Repository) CurrentBranch(ctx context.Context) (string, error) {
	return r.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
}

func (r *Repository) ListTags(ctx context.Context) ([]domain.Tag, error) {
	output, err := r.run(ctx, "tag", "--list", "--sort=-version:refname")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}

	lines := strings.Split(output, "\n")
	tags := make([]domain.Tag, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		hash, err := r.run(ctx, "rev-list", "-1", line)
		if err != nil {
			return nil, fmt.Errorf("resolving tag %s: %w", line, err)
		}
		tags = append(tags, domain.Tag{
			Name: line,
			Hash: hash,
		})
	}
	return tags, nil
}

func (r *Repository) CommitsSince(ctx context.Context, sinceHash string) ([]domain.Commit, error) {
	args := []string{"log", "--format=%H|%an|%ae|%aI|%s|%b%x00"}
	if sinceHash != "" {
		args = append(args, sinceHash+"..HEAD")
	}

	output, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}

	return parseCommitLog(output)
}

func parseCommitLog(output string) ([]domain.Commit, error) {
	entries := strings.Split(output, "\x00")
	commits := make([]domain.Commit, 0, len(entries))

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		commit, err := parseCommitEntry(entry)
		if err != nil {
			continue // skip unparseable entries
		}
		commits = append(commits, commit)
	}
	return commits, nil
}

func parseCommitEntry(entry string) (domain.Commit, error) {
	// First line: hash|author|email|date|subject
	// Remaining: body
	lines := strings.SplitN(entry, "\n", 2)
	firstLine := lines[0]

	parts := strings.SplitN(firstLine, "|", 6)
	if len(parts) < 5 {
		return domain.Commit{}, fmt.Errorf("unexpected commit format: %q", firstLine)
	}

	date, err := time.Parse(time.RFC3339, parts[3])
	if err != nil {
		return domain.Commit{}, fmt.Errorf("parsing commit date %q: %w", parts[3], err)
	}

	body := ""
	if len(parts) >= 6 {
		body = parts[5]
	}
	if len(lines) > 1 {
		body = body + "\n" + lines[1]
	}

	return domain.Commit{
		Hash:        parts[0],
		Author:      parts[1],
		AuthorEmail: parts[2],
		Date:        date,
		Message:     parts[4],
		Body:        strings.TrimSpace(body),
	}, nil
}

func (r *Repository) FilesChangedInCommit(ctx context.Context, hash string) ([]string, error) {
	output, err := r.run(ctx, "diff-tree", "--no-commit-id", "--name-only", "-r", hash)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}
	lines := strings.Split(output, "\n")
	result := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != "" {
			result = append(result, l)
		}
	}
	return result, nil
}

func (r *Repository) CreateTag(ctx context.Context, name, hash, message string) error {
	var err error
	if message != "" {
		_, err = r.run(ctx, "tag", "-a", name, hash, "-m", message)
	} else {
		_, err = r.run(ctx, "tag", name, hash)
	}
	if err == nil {
		return nil
	}
	// When the tag already exists, check whether it resolves to the same commit.
	// If it does, the operation is idempotent — return ErrTagAlreadyExists so
	// the caller can handle the re-run case without treating it as a hard failure.
	if strings.Contains(err.Error(), "already exists") {
		existing, resolveErr := r.run(ctx, "rev-parse", name+"^{commit}")
		if resolveErr == nil && existing == hash {
			return domain.ErrTagAlreadyExists
		}
	}
	return err
}

func (r *Repository) PushTag(ctx context.Context, name string) error {
	_, err := r.run(ctx, "push", "origin", name)
	return err
}

func (r *Repository) HeadHash(ctx context.Context) (string, error) {
	return r.run(ctx, "rev-parse", "HEAD")
}

func (r *Repository) RemoteURL(ctx context.Context) (string, error) {
	return r.run(ctx, "remote", "get-url", "origin")
}

func (r *Repository) Stage(ctx context.Context, files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, files...)
	_, err := r.run(ctx, args...)
	return err
}

func (r *Repository) Commit(ctx context.Context, message string) error {
	_, err := r.run(ctx, "commit", "-m", message)
	return err
}

func (r *Repository) Push(ctx context.Context) error {
	_, err := r.run(ctx, "push", "origin", "HEAD")
	return err
}
