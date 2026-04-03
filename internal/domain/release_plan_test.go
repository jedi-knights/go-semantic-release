package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestReleasePlan_HasReleasableProjects(t *testing.T) {
	tests := []struct {
		name string
		plan domain.ReleasePlan
		want bool
	}{
		{
			name: "with releasable",
			plan: domain.ReleasePlan{
				Projects: []domain.ProjectReleasePlan{
					{ShouldRelease: false},
					{ShouldRelease: true},
				},
			},
			want: true,
		},
		{
			name: "none releasable",
			plan: domain.ReleasePlan{
				Projects: []domain.ProjectReleasePlan{
					{ShouldRelease: false},
				},
			},
			want: false,
		},
		{
			name: "empty",
			plan: domain.ReleasePlan{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.HasReleasableProjects(); got != tt.want {
				t.Errorf("HasReleasableProjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReleasePlan_ReleasableProjects(t *testing.T) {
	plan := domain.ReleasePlan{
		Projects: []domain.ProjectReleasePlan{
			{Project: domain.Project{Name: "skip"}, ShouldRelease: false},
			{Project: domain.Project{Name: "release1"}, ShouldRelease: true},
			{Project: domain.Project{Name: "release2"}, ShouldRelease: true},
		},
	}

	releasable := plan.ReleasableProjects()
	if len(releasable) != 2 {
		t.Fatalf("expected 2, got %d", len(releasable))
	}
	if releasable[0].Project.Name != "release1" {
		t.Errorf("first = %q, want %q", releasable[0].Project.Name, "release1")
	}
}
