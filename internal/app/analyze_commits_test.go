package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestCommitAnalyzer_Analyze_ParsesCommits(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockParser := mocks.NewMockCommitParser(ctrl)

	rawCommits := []domain.Commit{
		{Hash: "abc1234", Message: "feat: add login", Author: "Alice", AuthorEmail: "alice@example.com", Date: time.Now()},
		{Hash: "def5678", Message: "fix: correct panic", Author: "Bob", AuthorEmail: "bob@example.com", Date: time.Now()},
	}

	mockGit.EXPECT().CommitsSince(gomock.Any(), "").Return(rawCommits, nil)
	mockGit.EXPECT().FilesChangedInCommit(gomock.Any(), "abc1234").Return([]string{"main.go"}, nil)
	mockGit.EXPECT().FilesChangedInCommit(gomock.Any(), "def5678").Return([]string{"handler.go"}, nil)
	mockParser.EXPECT().Parse("feat: add login").Return(domain.Commit{Type: "feat", Description: "add login"}, nil)
	mockParser.EXPECT().Parse("fix: correct panic").Return(domain.Commit{Type: "fix", Description: "correct panic"}, nil)

	analyzer := app.NewCommitAnalyzer(mockGit, mockParser, noopLogger{})
	commits, err := analyzer.Analyze(context.Background(), "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("Analyze() returned %d commits, want 2", len(commits))
	}
	if commits[0].Hash != "abc1234" {
		t.Errorf("commits[0].Hash = %q, want abc1234", commits[0].Hash)
	}
	if len(commits[0].FilesChanged) != 1 {
		t.Errorf("commits[0].FilesChanged = %v, want 1 file", commits[0].FilesChanged)
	}
}

func TestCommitAnalyzer_Analyze_SkipsUnparseable(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockParser := mocks.NewMockCommitParser(ctrl)

	rawCommits := []domain.Commit{
		{Hash: "bad0001", Message: "not conventional"},
		{Hash: "good001", Message: "feat: valid"},
	}

	mockGit.EXPECT().CommitsSince(gomock.Any(), "").Return(rawCommits, nil)
	mockGit.EXPECT().FilesChangedInCommit(gomock.Any(), "good001").Return(nil, nil)
	mockParser.EXPECT().Parse("not conventional").Return(domain.Commit{}, errors.New("parse error"))
	mockParser.EXPECT().Parse("feat: valid").Return(domain.Commit{Type: "feat", Description: "valid"}, nil)

	analyzer := app.NewCommitAnalyzer(mockGit, mockParser, noopLogger{})
	commits, err := analyzer.Analyze(context.Background(), "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("Analyze() = %d commits, want 1 (unparseable skipped)", len(commits))
	}
}

func TestCommitAnalyzer_Analyze_GitError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockParser := mocks.NewMockCommitParser(ctrl)

	mockGit.EXPECT().CommitsSince(gomock.Any(), "").Return(nil, errors.New("git failure"))

	analyzer := app.NewCommitAnalyzer(mockGit, mockParser, noopLogger{})
	_, err := analyzer.Analyze(context.Background(), "")
	if err == nil {
		t.Error("Analyze() should return error when git fails")
	}
}

func TestCommitAnalyzer_Analyze_PreservesGitMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockParser := mocks.NewMockCommitParser(ctrl)

	ts := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	rawCommits := []domain.Commit{
		{Hash: "hash001", Message: "feat: new thing", Author: "Carol", AuthorEmail: "carol@example.com", Date: ts},
	}

	mockGit.EXPECT().CommitsSince(gomock.Any(), "HEAD~5").Return(rawCommits, nil)
	mockGit.EXPECT().FilesChangedInCommit(gomock.Any(), "hash001").Return([]string{"a.go", "b.go"}, nil)
	mockParser.EXPECT().Parse("feat: new thing").Return(domain.Commit{Type: "feat"}, nil)

	analyzer := app.NewCommitAnalyzer(mockGit, mockParser, noopLogger{})
	commits, err := analyzer.Analyze(context.Background(), "HEAD~5")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if commits[0].Author != "Carol" {
		t.Errorf("Author = %q, want Carol", commits[0].Author)
	}
	if commits[0].AuthorEmail != "carol@example.com" {
		t.Errorf("AuthorEmail = %q, want carol@example.com", commits[0].AuthorEmail)
	}
	if !commits[0].Date.Equal(ts) {
		t.Errorf("Date = %v, want %v", commits[0].Date, ts)
	}
}

func TestCommitAnalyzer_Analyze_BodyAppendedToMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockParser := mocks.NewMockCommitParser(ctrl)

	rawCommits := []domain.Commit{
		{Hash: "hbody", Message: "feat: thing", Body: "BREAKING CHANGE: removes old API"},
	}

	expectedMsg := "feat: thing\n\nBREAKING CHANGE: removes old API"
	mockGit.EXPECT().CommitsSince(gomock.Any(), "").Return(rawCommits, nil)
	mockGit.EXPECT().FilesChangedInCommit(gomock.Any(), "hbody").Return(nil, nil)
	mockParser.EXPECT().Parse(expectedMsg).Return(domain.Commit{Type: "feat", IsBreakingChange: true}, nil)

	analyzer := app.NewCommitAnalyzer(mockGit, mockParser, noopLogger{})
	commits, err := analyzer.Analyze(context.Background(), "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if !commits[0].IsBreakingChange {
		t.Error("breaking change commit should be preserved")
	}
}
