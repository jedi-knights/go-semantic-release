package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestReleaseMode_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mode domain.ReleaseMode
		want string
	}{
		{"repo", domain.ReleaseModeRepo, "repo"},
		{"independent", domain.ReleaseModeIndependent, "independent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectType_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pt   domain.ProjectType
		want string
	}{
		{"go-workspace", domain.ProjectTypeGoWorkspace, "go-workspace"},
		{"go-module", domain.ProjectTypeGoModule, "go-module"},
		{"configured", domain.ProjectTypeConfigured, "configured"},
		{"root", domain.ProjectTypeRoot, "root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.pt.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProject_IsRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"dot path", ".", true},
		{"empty path", "", true},
		{"slash path", "/", true},
		{"subdir", "cmd/app", false},
		// IsRoot uses exact string matching without filepath.Clean normalization.
		// A path like "./subdir/.." would return false even though it resolves to ".".
		// Callers are responsible for cleaning paths before constructing a Project.
		{"uncleaned dot-dot path", "./subdir/..", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := domain.Project{Path: tt.path}
			if got := p.IsRoot(); got != tt.want {
				t.Errorf("IsRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}
