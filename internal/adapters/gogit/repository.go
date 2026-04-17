package gogit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.GitRepository = (*Repository)(nil)

// Repository implements ports.GitRepository using go-git (pure Go, no CLI dependency).
type Repository struct {
	repo    *git.Repository
	workDir string
}

// NewRepository opens an existing git repository at the given path.
func NewRepository(workDir string) (*Repository, error) {
	repo, err := git.PlainOpen(workDir)
	if err != nil {
		return nil, fmt.Errorf("opening git repository at %s: %w", workDir, err)
	}
	return &Repository{repo: repo, workDir: workDir}, nil
}

// CurrentBranch returns the name of the checked-out branch.
func (r *Repository) CurrentBranch(_ context.Context) (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}
	if !head.Name().IsBranch() {
		return head.Name().Short(), nil
	}
	return head.Name().Short(), nil
}

// ListTags returns all tags in the repository.
func (r *Repository) ListTags(_ context.Context) ([]domain.Tag, error) {
	tagRefs, err := r.repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	var tags []domain.Tag
	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		hash := ref.Hash().String()

		// For annotated tags, resolve to the commit hash.
		tagObj, tagErr := r.repo.TagObject(ref.Hash())
		isAnnotated := false
		if tagErr == nil {
			hash = tagObj.Target.String()
			isAnnotated = true
		}

		tags = append(tags, domain.Tag{
			Name:        ref.Name().Short(),
			Hash:        hash,
			IsAnnotated: isAnnotated,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by name descending (version sort).
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name > tags[j].Name
	})

	return tags, nil
}

// CommitsSince returns commits since the given hash (exclusive).
func (r *Repository) CommitsSince(_ context.Context, sinceHash string) ([]domain.Commit, error) {
	head, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD: %w", err)
	}

	logOpts := &git.LogOptions{
		From:  head.Hash(),
		Order: git.LogOrderCommitterTime,
	}

	iter, err := r.repo.Log(logOpts)
	if err != nil {
		return nil, fmt.Errorf("getting log: %w", err)
	}

	var commits []domain.Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if sinceHash != "" && c.Hash.String() == sinceHash {
			return fmt.Errorf("stop") // sentinel to break iteration
		}

		subject, body := splitMessage(c.Message)

		commits = append(commits, domain.Commit{
			Hash:        c.Hash.String(),
			Author:      c.Author.Name,
			AuthorEmail: c.Author.Email,
			Date:        c.Author.When,
			Message:     subject,
			Body:        body,
		})
		return nil
	})
	// Ignore sentinel error.
	if err != nil && err.Error() != "stop" {
		return nil, err
	}

	return commits, nil
}

// FilesChangedInCommit returns the list of files changed by a commit.
func (r *Repository) FilesChangedInCommit(_ context.Context, hash string) ([]string, error) {
	commitObj, err := r.repo.CommitObject(plumbing.NewHash(hash))
	if err != nil {
		return nil, fmt.Errorf("getting commit %s: %w", hash, err)
	}

	stats, err := commitObj.Stats()
	if err != nil {
		return nil, fmt.Errorf("getting stats for %s: %w", hash, err)
	}

	files := make([]string, 0, len(stats))
	for _, s := range stats {
		files = append(files, s.Name)
	}
	return files, nil
}

// CreateTag creates a git tag at the given commit hash.
func (r *Repository) CreateTag(_ context.Context, name, hash, message string) error {
	commitHash := plumbing.NewHash(hash)

	if message != "" {
		// Create annotated tag.
		_, err := r.repo.CreateTag(name, commitHash, &git.CreateTagOptions{
			Message: message,
			Tagger: &object.Signature{
				Name:  "semantic-release-bot",
				Email: "semantic-release-bot@users.noreply.github.com",
				When:  time.Now(),
			},
		})
		if err == nil {
			return nil
		}
		if errors.Is(err, git.ErrTagExists) {
			// Use the peel suffix "^{}" to dereference the annotated tag object
			// to the underlying commit hash; without it ResolveRevision returns
			// the tag object hash, which never matches the commit hash argument.
			resolved, resolveErr := r.repo.ResolveRevision(plumbing.Revision("refs/tags/" + name + "^{}"))
			if resolveErr == nil && resolved.String() == hash {
				return domain.ErrTagAlreadyExists
			}
		}
		return err
	}

	// Create lightweight tag.
	ref := plumbing.NewReferenceFromStrings("refs/tags/"+name, hash)
	return r.repo.Storer.SetReference(ref)
}

// PushTag pushes a tag to the remote.
func (r *Repository) PushTag(_ context.Context, name string) error {
	refSpec := config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", name, name))

	auth := resolveAuth()

	err := r.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{refSpec},
		Auth:       auth,
	})
	// NoErrAlreadyUpToDate is a non-nil sentinel that go-git returns when the
	// remote already has the ref at the same SHA. Treat it as success so that
	// re-runs (where the tag was already pushed in a prior attempt) do not fail.
	if err == nil || errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	return fmt.Errorf("pushing tag %s: %w", name, err)
}

// HeadHash returns the hash of HEAD.
func (r *Repository) HeadHash(_ context.Context) (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}
	return head.Hash().String(), nil
}

// RemoteURL returns the remote origin URL.
func (r *Repository) RemoteURL(_ context.Context) (string, error) {
	remote, err := r.repo.Remote("origin")
	if err != nil {
		return "", fmt.Errorf("getting remote: %w", err)
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", fmt.Errorf("no URLs configured for remote 'origin'")
	}
	return urls[0], nil
}

// Stage adds the given file paths to the worktree index.
func (r *Repository) Stage(_ context.Context, files []string) error {
	if len(files) == 0 {
		return nil
	}
	w, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}
	for _, file := range files {
		if _, err := w.Add(file); err != nil {
			return fmt.Errorf("staging %s: %w", file, err)
		}
	}
	return nil
}

// Commit creates a commit with the given message from the current index.
func (r *Repository) Commit(_ context.Context, message string) error {
	w, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}
	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "semantic-release-bot",
			Email: "semantic-release-bot@users.noreply.github.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	return nil
}

// Push pushes the current branch to origin.
func (r *Repository) Push(_ context.Context) error {
	auth := resolveAuth()
	err := r.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err == nil || errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	return fmt.Errorf("pushing branch: %w", err)
}

func splitMessage(msg string) (subject, body string) {
	parts := strings.SplitN(strings.TrimSpace(msg), "\n", 2)
	subject = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		body = strings.TrimSpace(parts[1])
	}
	return subject, body
}

func resolveAuth() transport.AuthMethod {
	// Try token-based auth for HTTPS remotes.
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN", "GL_TOKEN", "GITLAB_TOKEN", "BB_TOKEN", "BITBUCKET_TOKEN"} {
		if token := os.Getenv(key); token != "" {
			return &githttp.BasicAuth{
				Username: "git",
				Password: token,
			}
		}
	}
	return nil
}
