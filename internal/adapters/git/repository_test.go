// White-box tests for the unexported parse helpers in the git adapter.
// Uses package git (not git_test) to access unexported functions directly.
package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// ---------------------------------------------------------------------------
// parseCommitEntry
// ---------------------------------------------------------------------------

func TestParseCommitEntry_Basic(t *testing.T) {
	entry := "abc1234|John Doe|john@example.com|2024-01-15T10:30:00Z|feat: add login|"

	got, err := parseCommitEntry(entry)
	if err != nil {
		t.Fatalf("parseCommitEntry: unexpected error: %v", err)
	}

	if got.Hash != "abc1234" {
		t.Errorf("Hash = %q, want %q", got.Hash, "abc1234")
	}
	if got.Author != "John Doe" {
		t.Errorf("Author = %q, want %q", got.Author, "John Doe")
	}
	if got.AuthorEmail != "john@example.com" {
		t.Errorf("AuthorEmail = %q, want %q", got.AuthorEmail, "john@example.com")
	}
	wantDate, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	if !got.Date.Equal(wantDate) {
		t.Errorf("Date = %v, want %v", got.Date, wantDate)
	}
	if got.Message != "feat: add login" {
		t.Errorf("Message = %q, want %q", got.Message, "feat: add login")
	}
	// Inline body field is empty and no second line — body should be empty.
	if got.Body != "" {
		t.Errorf("Body = %q, want empty", got.Body)
	}
}

func TestParseCommitEntry_WithBody(t *testing.T) {
	// Body delivered on a second line (as git log %b emits it).
	entry := "abc1234|Jane Smith|jane@example.com|2024-02-01T08:00:00Z|fix: correct panic|\nThis fixes the crash in the auth handler."

	got, err := parseCommitEntry(entry)
	if err != nil {
		t.Fatalf("parseCommitEntry: unexpected error: %v", err)
	}

	if got.Body == "" {
		t.Error("Body is empty, want non-empty body")
	}
}

func TestParseCommitEntry_TooFewFields(t *testing.T) {
	entry := "abc|author"

	_, err := parseCommitEntry(entry)
	if err == nil {
		t.Fatal("expected error for entry with too few fields, got nil")
	}
}

func TestParseCommitEntry_InvalidDate(t *testing.T) {
	// time.Parse returns the zero value on failure; parseCommitEntry ignores the error.
	entry := "abc1234|Author Name|author@example.com|not-a-date|chore: update deps|"

	got, err := parseCommitEntry(entry)
	if err != nil {
		t.Fatalf("parseCommitEntry: unexpected error: %v", err)
	}

	// Expect zero time — time.Parse fails silently and the result discards the error.
	if !got.Date.IsZero() {
		t.Errorf("Date = %v, want zero time for invalid date input", got.Date)
	}
}

// ---------------------------------------------------------------------------
// parseCommitLog
// ---------------------------------------------------------------------------

func TestParseCommitLog_MultipleEntries(t *testing.T) {
	entry1 := "aaa111|Alice|alice@example.com|2024-03-01T12:00:00Z|feat: feature one|"
	entry2 := "bbb222|Bob|bob@example.com|2024-03-02T12:00:00Z|fix: fix two|"
	output := entry1 + "\x00" + entry2

	commits, err := parseCommitLog(output)
	if err != nil {
		t.Fatalf("parseCommitLog: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	if commits[0].Hash != "aaa111" {
		t.Errorf("commits[0].Hash = %q, want %q", commits[0].Hash, "aaa111")
	}
	if commits[1].Hash != "bbb222" {
		t.Errorf("commits[1].Hash = %q, want %q", commits[1].Hash, "bbb222")
	}
}

func TestParseCommitLog_SkipsEmpty(t *testing.T) {
	// Trailing NUL produces an empty entry that should be skipped, not appended.
	entry := "ccc333|Carol|carol@example.com|2024-04-01T10:00:00Z|docs: update readme|"
	output := entry + "\x00"

	commits, err := parseCommitLog(output)
	if err != nil {
		t.Fatalf("parseCommitLog: %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("expected 1 commit (empty entry skipped), got %d", len(commits))
	}
}

func TestParseCommitLog_SkipsUnparseable(t *testing.T) {
	// First entry has too few fields (unparseable); second entry is valid.
	bad := "bad|entry"
	good := "ddd444|Dave|dave@example.com|2024-05-01T09:00:00Z|ci: configure pipeline|"
	output := bad + "\x00" + good

	commits, err := parseCommitLog(output)
	if err != nil {
		t.Fatalf("parseCommitLog: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit (bad entry skipped), got %d", len(commits))
	}
	if commits[0].Hash != "ddd444" {
		t.Errorf("commits[0].Hash = %q, want %q", commits[0].Hash, "ddd444")
	}
}

func TestParseCommitLog_Empty(t *testing.T) {
	commits, err := parseCommitLog("")
	if err != nil {
		t.Fatalf("parseCommitLog: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits for empty input, got %d", len(commits))
	}
}

// ---------------------------------------------------------------------------
// Integration helpers — build a real git repo in a temp directory.
// ---------------------------------------------------------------------------

// newTestGitRepo initialises a bare git repo in t.TempDir() and returns both
// the directory path and a *Repository pointing at it. Using --initial-branch
// ensures the branch name is deterministic across different git versions.
func newTestGitRepo(t *testing.T) (string, *Repository) {
	t.Helper()
	dir := t.TempDir()
	commands := [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range commands {
		out, err := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	return dir, NewRepository(dir)
}

// addTestCommit writes file content, stages it, commits with msg, and returns
// the resulting HEAD hash. It is the caller's responsibility to ensure at least
// one prior commit exists if the branch has not yet been created.
func addTestCommit(t *testing.T, dir, file, content, msg string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", file}, {"commit", "-m", msg}} {
		out, err := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	out, _ := exec.CommandContext(context.Background(), "git", "-C", dir, "rev-parse", "HEAD").Output()
	return strings.TrimSpace(string(out))
}

// ---------------------------------------------------------------------------
// Repository method tests
// ---------------------------------------------------------------------------

func TestRepository_CurrentBranch(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	// A branch only exists after the first commit.
	addTestCommit(t, dir, "README", "hello", "chore: initial commit")

	branch, err := repo.CurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("CurrentBranch: unexpected error: %v", err)
	}
	if branch == "" {
		t.Error("CurrentBranch returned empty string, want non-empty branch name")
	}
	// The helper hard-codes --initial-branch=main so the name is deterministic.
	if branch != "main" {
		t.Errorf("CurrentBranch = %q, want %q", branch, "main")
	}
}

func TestRepository_ListTags_Empty(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	addTestCommit(t, dir, "README", "hello", "chore: initial commit")

	tags, err := repo.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: unexpected error: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

func TestRepository_ListTags_WithTag(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	hash := addTestCommit(t, dir, "README", "hello", "chore: initial commit")

	// Create a lightweight tag directly via git so we can verify ListTags picks it up.
	out, err := exec.CommandContext(context.Background(), "git", "-C", dir, "tag", "v1.0.0", hash).CombinedOutput()
	if err != nil {
		t.Fatalf("git tag: %s: %v", out, err)
	}

	tags, err := repo.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: unexpected error: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].Name != "v1.0.0" {
		t.Errorf("tags[0].Name = %q, want %q", tags[0].Name, "v1.0.0")
	}
}

func TestRepository_CommitsSince_All(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	addTestCommit(t, dir, "a.txt", "a", "feat: first")
	addTestCommit(t, dir, "b.txt", "b", "feat: second")

	// Passing an empty sinceHash returns all commits in the repo.
	commits, err := repo.CommitsSince(context.Background(), "")
	if err != nil {
		t.Fatalf("CommitsSince: unexpected error: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("expected 2 commits, got %d", len(commits))
	}
}

func TestRepository_CommitsSince_Since(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	firstHash := addTestCommit(t, dir, "a.txt", "a", "feat: first")
	addTestCommit(t, dir, "b.txt", "b", "feat: second")

	// Passing the first commit's hash as sinceHash should return only the second commit.
	commits, err := repo.CommitsSince(context.Background(), firstHash)
	if err != nil {
		t.Fatalf("CommitsSince: unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit after firstHash, got %d", len(commits))
	}
	if commits[0].Message != "feat: second" {
		t.Errorf("commits[0].Message = %q, want %q", commits[0].Message, "feat: second")
	}
}

func TestRepository_HeadHash(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	addTestCommit(t, dir, "README", "hello", "chore: initial commit")

	hash, err := repo.HeadHash(context.Background())
	if err != nil {
		t.Fatalf("HeadHash: unexpected error: %v", err)
	}
	if hash == "" {
		t.Error("HeadHash returned empty string, want a non-empty SHA")
	}
}

func TestRepository_FilesChangedInCommit(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	// git diff-tree --no-commit-id needs a parent commit to produce output; add
	// a setup commit first, then the commit we actually want to inspect.
	addTestCommit(t, dir, "setup.txt", "setup", "chore: initial commit")
	hash := addTestCommit(t, dir, "foo.txt", "content", "feat: add foo")

	files, err := repo.FilesChangedInCommit(context.Background(), hash)
	if err != nil {
		t.Fatalf("FilesChangedInCommit: unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(files), files)
	}
	if files[0] != "foo.txt" {
		t.Errorf("files[0] = %q, want %q", files[0], "foo.txt")
	}
}

func TestRepository_CreateTag_Lightweight(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	hash := addTestCommit(t, dir, "README", "hello", "chore: initial commit")

	// An empty message signals a lightweight (non-annotated) tag.
	if err := repo.CreateTag(context.Background(), "v0.1.0", hash, ""); err != nil {
		t.Fatalf("CreateTag (lightweight): unexpected error: %v", err)
	}

	// Confirm git itself sees the tag.
	out, err := exec.CommandContext(context.Background(), "git", "-C", dir, "tag", "-l", "v0.1.0").Output()
	if err != nil {
		t.Fatalf("git tag -l: %v", err)
	}
	if strings.TrimSpace(string(out)) != "v0.1.0" {
		t.Errorf("tag v0.1.0 not found after CreateTag; git tag output: %q", string(out))
	}
}

func TestRepository_CreateTag_Annotated(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	hash := addTestCommit(t, dir, "README", "hello", "chore: initial commit")

	// A non-empty message produces an annotated tag.
	if err := repo.CreateTag(context.Background(), "v1.0.0", hash, "Release v1.0.0"); err != nil {
		t.Fatalf("CreateTag (annotated): unexpected error: %v", err)
	}

	out, err := exec.CommandContext(context.Background(), "git", "-C", dir, "tag", "-l", "v1.0.0").Output()
	if err != nil {
		t.Fatalf("git tag -l: %v", err)
	}
	if strings.TrimSpace(string(out)) != "v1.0.0" {
		t.Errorf("tag v1.0.0 not found after CreateTag; git tag output: %q", string(out))
	}
}

func TestRepository_CreateTag_Duplicate_SameHash(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	hash := addTestCommit(t, dir, "README", "hello", "chore: initial commit")

	if err := repo.CreateTag(context.Background(), "v1.0.0", hash, ""); err != nil {
		t.Fatalf("first CreateTag: unexpected error: %v", err)
	}

	// A second call with the same name and same hash must return ErrTagAlreadyExists
	// rather than a hard failure — the operation is idempotent by design.
	err := repo.CreateTag(context.Background(), "v1.0.0", hash, "")
	if err == nil {
		t.Fatal("expected ErrTagAlreadyExists on duplicate tag, got nil")
	}
	if err != domain.ErrTagAlreadyExists {
		t.Errorf("expected domain.ErrTagAlreadyExists, got: %v", err)
	}
}

func TestRepository_PushTag_NoRemote(t *testing.T) {
	dir, repo := newTestGitRepo(t)
	addTestCommit(t, dir, "README", "hello", "chore: initial commit")

	// Without a configured remote named "origin", PushTag must return an error.
	err := repo.PushTag(context.Background(), "v1.0.0")
	if err == nil {
		t.Fatal("expected error when pushing to a repo with no remote, got nil")
	}
}

func TestRepository_RemoteURL_NoRemote(t *testing.T) {
	_, repo := newTestGitRepo(t)

	// Without any remote configured, RemoteURL must return an error.
	_, err := repo.RemoteURL(context.Background())
	if err == nil {
		t.Fatal("expected error when fetching remote URL from a repo with no remote, got nil")
	}
}
