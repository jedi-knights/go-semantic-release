package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestCommit_ReleaseType(t *testing.T) {
	mapping := domain.DefaultCommitTypeMapping()

	tests := []struct {
		name   string
		commit domain.Commit
		want   domain.ReleaseType
	}{
		{
			name:   "feat is minor",
			commit: domain.Commit{Type: "feat"},
			want:   domain.ReleaseMinor,
		},
		{
			name:   "fix is patch",
			commit: domain.Commit{Type: "fix"},
			want:   domain.ReleasePatch,
		},
		{
			name:   "breaking change is always major",
			commit: domain.Commit{Type: "fix", IsBreakingChange: true},
			want:   domain.ReleaseMajor,
		},
		{
			name:   "chore is none",
			commit: domain.Commit{Type: "chore"},
			want:   domain.ReleaseNone,
		},
		{
			name:   "docs is none",
			commit: domain.Commit{Type: "docs"},
			want:   domain.ReleaseNone,
		},
		{
			name:   "perf is patch",
			commit: domain.Commit{Type: "perf"},
			want:   domain.ReleasePatch,
		},
		{
			name:   "unknown type is none",
			commit: domain.Commit{Type: "unknown"},
			want:   domain.ReleaseNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.commit.ReleaseType(mapping); got != tt.want {
				t.Errorf("ReleaseType() = %v, want %v", got, tt.want)
			}
		})
	}
}
