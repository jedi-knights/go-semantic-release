package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestReleaseMode_String(t *testing.T) {
	tests := []struct {
		mode domain.ReleaseMode
		want string
	}{
		{domain.ReleaseModeRepo, "repo"},
		{domain.ReleaseModeIndependent, "independent"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectType_String(t *testing.T) {
	tests := []struct {
		pt   domain.ProjectType
		want string
	}{
		{domain.ProjectTypeGoWorkspace, "go-workspace"},
		{domain.ProjectTypeGoModule, "go-module"},
		{domain.ProjectTypeConfigured, "configured"},
		{domain.ProjectTypeRoot, "root"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.pt.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProject_IsRoot(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"dot path", ".", true},
		{"empty path", "", true},
		{"subdir", "cmd/app", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := domain.Project{Path: tt.path}
			if got := p.IsRoot(); got != tt.want {
				t.Errorf("IsRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}
