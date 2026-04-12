package plugins_test

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestCommitAnalyzerPlugin_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := plugins.NewCommitAnalyzerPlugin(mocks.NewMockCommitParser(ctrl), nil)
	if p.Name() != "commit-analyzer" {
		t.Errorf("Name() = %q, want commit-analyzer", p.Name())
	}
}

func TestCommitAnalyzerPlugin_AnalyzeCommits_ReturnsHighestReleaseType(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := plugins.NewCommitAnalyzerPlugin(
		mocks.NewMockCommitParser(ctrl),
		domain.DefaultCommitTypeMapping(),
	)

	rc := &domain.ReleaseContext{
		Commits: []domain.Commit{
			{Type: "fix"},  // patch
			{Type: "feat"}, // minor → should win
			{Type: "chore"},
		},
	}

	rt, err := p.AnalyzeCommits(context.Background(), rc)
	if err != nil {
		t.Fatalf("AnalyzeCommits() error = %v", err)
	}
	if rt != domain.ReleaseMinor {
		t.Errorf("AnalyzeCommits() = %v, want ReleaseMinor", rt)
	}
}

func TestCommitAnalyzerPlugin_AnalyzeCommits_BreakingChangeTakesPrecedence(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := plugins.NewCommitAnalyzerPlugin(
		mocks.NewMockCommitParser(ctrl),
		domain.DefaultCommitTypeMapping(),
	)

	rc := &domain.ReleaseContext{
		Commits: []domain.Commit{
			{Type: "feat"},
			{Type: "fix", IsBreakingChange: true}, // major
		},
	}

	rt, err := p.AnalyzeCommits(context.Background(), rc)
	if err != nil {
		t.Fatalf("AnalyzeCommits() error = %v", err)
	}
	if rt != domain.ReleaseMajor {
		t.Errorf("AnalyzeCommits() = %v, want ReleaseMajor", rt)
	}
}

func TestCommitAnalyzerPlugin_AnalyzeCommits_EmptyCommitsReturnsNone(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := plugins.NewCommitAnalyzerPlugin(
		mocks.NewMockCommitParser(ctrl),
		domain.DefaultCommitTypeMapping(),
	)

	rc := &domain.ReleaseContext{Commits: nil}
	rt, err := p.AnalyzeCommits(context.Background(), rc)
	if err != nil {
		t.Fatalf("AnalyzeCommits() error = %v", err)
	}
	if rt != domain.ReleaseNone {
		t.Errorf("AnalyzeCommits() with no commits = %v, want ReleaseNone", rt)
	}
}
