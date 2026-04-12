package gogit_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/gogit"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// newTestRepo initialises a real git repository in a temp directory so that
// NewRepository can open it via PlainOpen.
func newTestRepo(t *testing.T) (testRepo *git.Repository, testDir string) {
	t.Helper()

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}

	cfg, err := repo.Config()
	if err != nil {
		t.Fatalf("repo.Config: %v", err)
	}
	cfg.User.Name = "test"
	cfg.User.Email = "test@test.com"
	if err := repo.Storer.SetConfig(cfg); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}

	return repo, dir
}

// addCommit writes a file with non-empty content and creates a commit.
// Content is set to the filename so each file is uniquely non-empty,
// which matters for go-git Stats() to report the file in changed-file lists.
func addCommit(t *testing.T, repo *git.Repository, dir, filename, msg string) plumbing.Hash {
	t.Helper()

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}

	// Write non-empty content — go-git Stats() omits empty files from results.
	if writeErr := os.WriteFile(filepath.Join(dir, filename), []byte("content: "+filename+"\n"), 0o644); writeErr != nil {
		t.Fatalf("WriteFile %s: %v", filename, writeErr)
	}

	if _, addErr := wt.Add("."); addErr != nil {
		t.Fatalf("wt.Add: %v", addErr)
	}

	hash, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("wt.Commit(%q): %v", msg, err)
	}
	return hash
}

// runGit executes a git sub-command in dir and returns its combined output.
// It is used for operations that are awkward to drive through go-git's API
// (e.g. adding a remote, detaching HEAD).
func runGit(t *testing.T, dir string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.Bytes(), err
}

// ---------------------------------------------------------------------------
// NewRepository
// ---------------------------------------------------------------------------

func TestGoGit_NewRepository_InvalidPath(t *testing.T) {
	_, err := gogit.NewRepository("/this/path/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}
}

// ---------------------------------------------------------------------------
// HeadHash
// ---------------------------------------------------------------------------

func TestGoGit_HeadHash(t *testing.T) {
	repo, dir := newTestRepo(t)
	hash := addCommit(t, repo, dir, "a.txt", "initial commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	got, err := r.HeadHash(context.Background())
	if err != nil {
		t.Fatalf("HeadHash: %v", err)
	}
	if got == "" {
		t.Fatal("HeadHash returned empty string")
	}
	if got != hash.String() {
		t.Errorf("HeadHash = %q, want %q", got, hash.String())
	}
}

// ---------------------------------------------------------------------------
// CurrentBranch
// ---------------------------------------------------------------------------

func TestGoGit_CurrentBranch(t *testing.T) {
	repo, dir := newTestRepo(t)
	// HEAD does not exist until at least one commit is made.
	addCommit(t, repo, dir, "b.txt", "initial commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	branch, err := r.CurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	// git.PlainInit creates "master" by default; some environments may differ.
	if branch == "" {
		t.Error("CurrentBranch returned empty string")
	}
}

// ---------------------------------------------------------------------------
// ListTags
// ---------------------------------------------------------------------------

func TestGoGit_ListTags_Empty(t *testing.T) {
	repo, dir := newTestRepo(t)
	addCommit(t, repo, dir, "c.txt", "initial commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	tags, err := r.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected no tags, got %d", len(tags))
	}
}

func TestGoGit_ListTags_WithTags(t *testing.T) {
	repo, dir := newTestRepo(t)
	commitHash := addCommit(t, repo, dir, "d.txt", "initial commit")

	// Create a lightweight tag directly via go-git so we control the name.
	ref := plumbing.NewReferenceFromStrings("refs/tags/v0.1.0", commitHash.String())
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatalf("SetReference: %v", err)
	}

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	tags, err := r.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].Name != "v0.1.0" {
		t.Errorf("tag name = %q, want %q", tags[0].Name, "v0.1.0")
	}
}

// ---------------------------------------------------------------------------
// CommitsSince
// ---------------------------------------------------------------------------

func TestGoGit_CommitsSince_All(t *testing.T) {
	repo, dir := newTestRepo(t)
	addCommit(t, repo, dir, "e1.txt", "first commit")
	addCommit(t, repo, dir, "e2.txt", "second commit")
	addCommit(t, repo, dir, "e3.txt", "third commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	commits, err := r.CommitsSince(context.Background(), "")
	if err != nil {
		t.Fatalf("CommitsSince: %v", err)
	}
	if len(commits) != 3 {
		t.Errorf("expected 3 commits, got %d", len(commits))
	}
}

func TestGoGit_CommitsSince_Since(t *testing.T) {
	repo, dir := newTestRepo(t)
	h1 := addCommit(t, repo, dir, "f1.txt", "first commit")
	addCommit(t, repo, dir, "f2.txt", "second commit")
	addCommit(t, repo, dir, "f3.txt", "third commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	// Commits since h1 (exclusive) should be second and third only.
	commits, err := r.CommitsSince(context.Background(), h1.String())
	if err != nil {
		t.Fatalf("CommitsSince: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("expected 2 commits since first, got %d", len(commits))
	}
}

// ---------------------------------------------------------------------------
// FilesChangedInCommit
// ---------------------------------------------------------------------------

func TestGoGit_FilesChangedInCommit(t *testing.T) {
	repo, dir := newTestRepo(t)
	// First commit needed so Stats() has a parent to diff against.
	addCommit(t, repo, dir, "g_base.txt", "base commit")
	h2 := addCommit(t, repo, dir, "g_target.txt", "target commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	files, err := r.FilesChangedInCommit(context.Background(), h2.String())
	if err != nil {
		t.Fatalf("FilesChangedInCommit: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one changed file, got none")
	}
	found := false
	for _, f := range files {
		if f == "g_target.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("g_target.txt not in changed files: %v", files)
	}
}

// ---------------------------------------------------------------------------
// CreateTag
// ---------------------------------------------------------------------------

func TestGoGit_CreateTag_Lightweight(t *testing.T) {
	repo, dir := newTestRepo(t)
	h := addCommit(t, repo, dir, "h.txt", "initial commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	if tagErr := r.CreateTag(context.Background(), "v1.0.0", h.String(), ""); tagErr != nil {
		t.Fatalf("CreateTag (lightweight): %v", tagErr)
	}

	tags, err := r.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "v1.0.0" {
		t.Errorf("expected tag v1.0.0; got %+v", tags)
	}
	if tags[0].IsAnnotated {
		t.Error("expected lightweight tag (IsAnnotated=false)")
	}
}

func TestGoGit_CreateTag_Annotated(t *testing.T) {
	repo, dir := newTestRepo(t)
	h := addCommit(t, repo, dir, "i.txt", "initial commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	if tagErr := r.CreateTag(context.Background(), "v1.0.0", h.String(), "Release v1.0.0"); tagErr != nil {
		t.Fatalf("CreateTag (annotated): %v", tagErr)
	}

	tags, err := r.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "v1.0.0" {
		t.Errorf("expected tag v1.0.0; got %+v", tags)
	}
	if !tags[0].IsAnnotated {
		t.Error("expected annotated tag (IsAnnotated=true)")
	}
}

func TestGoGit_CreateTag_AlreadyExists_SameCommit(t *testing.T) {
	repo, dir := newTestRepo(t)
	h := addCommit(t, repo, dir, "j.txt", "initial commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	// First creation must succeed.
	if tagErr := r.CreateTag(context.Background(), "v1.0.0", h.String(), "Release v1.0.0"); tagErr != nil {
		t.Fatalf("first CreateTag: %v", tagErr)
	}

	// Second creation at the same commit must return ErrTagAlreadyExists.
	err = r.CreateTag(context.Background(), "v1.0.0", h.String(), "Release v1.0.0")
	if !errors.Is(err, domain.ErrTagAlreadyExists) {
		t.Errorf("expected ErrTagAlreadyExists, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// RemoteURL
// ---------------------------------------------------------------------------

func TestGoGit_RemoteURL_NoRemote(t *testing.T) {
	repo, dir := newTestRepo(t)
	addCommit(t, repo, dir, "k.txt", "initial commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	_, err = r.RemoteURL(context.Background())
	if err == nil {
		t.Fatal("expected error when no remote configured, got nil")
	}
}

func TestRepository_RemoteURL_WithRemote(t *testing.T) {
	repo, dir := newTestRepo(t)
	addCommit(t, repo, dir, "l.txt", "initial commit")

	// Add a remote using the system git binary so we don't need to wire
	// go-git's remote creation API directly in the test.
	if out, err := runGit(t, dir, "remote", "add", "origin", "https://example.com/repo.git"); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, out)
	}

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	url, err := r.RemoteURL(context.Background())
	if err != nil {
		t.Fatalf("RemoteURL: %v", err)
	}
	if url != "https://example.com/repo.git" {
		t.Errorf("RemoteURL = %q, want %q", url, "https://example.com/repo.git")
	}

	_ = repo // used only to satisfy the test repo setup pattern
}

func TestRepository_PushTag_NoRemote(t *testing.T) {
	repo, dir := newTestRepo(t)
	h := addCommit(t, repo, dir, "m.txt", "initial commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	// Create a tag so PushTag has something to push, then attempt to push it
	// to the non-existent "origin" remote — this must return an error.
	if tagErr := r.CreateTag(context.Background(), "v1.0.0", h.String(), ""); tagErr != nil {
		t.Fatalf("CreateTag: %v", tagErr)
	}

	err = r.PushTag(context.Background(), "v1.0.0")
	if err == nil {
		t.Fatal("expected error when pushing tag with no remote, got nil")
	}
}

func TestRepository_CurrentBranch_Detached(t *testing.T) {
	repo, dir := newTestRepo(t)
	h := addCommit(t, repo, dir, "n.txt", "initial commit")
	// Add a second commit so HEAD can be detached onto the first one.
	addCommit(t, repo, dir, "n2.txt", "second commit")

	// Detach HEAD at the first commit using the system git binary.
	if out, err := runGit(t, dir, "checkout", "--detach", h.String()); err != nil {
		t.Fatalf("git checkout --detach: %v\n%s", err, out)
	}

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	branch, err := r.CurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("CurrentBranch in detached HEAD: %v", err)
	}
	// In detached HEAD state go-git returns "HEAD" (the symbolic ref short name),
	// not the commit hash. We verify the returned string is non-empty.
	if branch == "" {
		t.Fatal("CurrentBranch returned empty string in detached HEAD state")
	}

	_ = repo // repo used only via addCommit helper
}

func TestRepository_PushTag_WithToken(t *testing.T) {
	// Set a token so that resolveAuth returns &githttp.BasicAuth instead of nil,
	// covering the early-return branch. The push itself fails (no remote) but
	// resolveAuth has been exercised.
	t.Setenv("GH_TOKEN", "test-token")

	repo, dir := newTestRepo(t)
	h := addCommit(t, repo, dir, "o.txt", "initial commit")

	r, err := gogit.NewRepository(dir)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	if tagErr := r.CreateTag(context.Background(), "v1.0.0", h.String(), ""); tagErr != nil {
		t.Fatalf("CreateTag: %v", tagErr)
	}

	// Expect an error since there is no remote — resolveAuth was still called.
	if err := r.PushTag(context.Background(), "v1.0.0"); err == nil {
		t.Fatal("expected error when pushing tag with no remote, got nil")
	}

	_ = repo
}
