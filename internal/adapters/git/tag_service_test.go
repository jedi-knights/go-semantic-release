package git_test

import (
	"testing"

	adaptergit "github.com/jedi-knights/go-semantic-release/internal/adapters/git"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestTemplateTagService_FormatTag(t *testing.T) {
	tests := []struct {
		name            string
		repoTemplate    string
		projectTemplate string
		project         string
		version         domain.Version
		want            string
	}{
		{
			name:    "repo-level default",
			project: "",
			version: domain.NewVersion(1, 2, 3),
			want:    "v1.2.3",
		},
		{
			name:    "project-level default",
			project: "api",
			version: domain.NewVersion(1, 2, 3),
			want:    "api/v1.2.3",
		},
		{
			name:            "custom project template with @",
			projectTemplate: "{{.Project}}@{{.Version}}",
			project:         "mylib",
			version:         domain.NewVersion(2, 0, 0),
			want:            "mylib@2.0.0",
		},
		{
			name:         "custom repo template",
			repoTemplate: "release-{{.Version}}",
			project:      "",
			version:      domain.NewVersion(3, 1, 0),
			want:         "release-3.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := adaptergit.NewTemplateTagService(tt.repoTemplate, tt.projectTemplate)
			got, err := svc.FormatTag(tt.project, tt.version)
			if err != nil {
				t.Fatalf("FormatTag() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("FormatTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateTagService_ParseTag(t *testing.T) {
	svc := adaptergit.NewTemplateTagService("", "")

	tests := []struct {
		name        string
		tag         string
		wantProject string
		wantVersion domain.Version
		wantErr     bool
	}{
		{
			name:        "repo tag with v",
			tag:         "v1.2.3",
			wantProject: "",
			wantVersion: domain.NewVersion(1, 2, 3),
		},
		{
			name:        "project tag with slash",
			tag:         "api/v1.2.3",
			wantProject: "api",
			wantVersion: domain.NewVersion(1, 2, 3),
		},
		{
			name:        "project tag with @",
			tag:         "mylib@2.0.0",
			wantProject: "mylib",
			wantVersion: domain.NewVersion(2, 0, 0),
		},
		{
			name:        "nested project tag",
			tag:         "services/api/v0.3.1",
			wantProject: "services/api",
			wantVersion: domain.NewVersion(0, 3, 1),
		},
		{
			name:    "invalid tag",
			tag:     "not-a-tag",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, version, err := svc.ParseTag(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseTag() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if project != tt.wantProject {
				t.Errorf("project = %q, want %q", project, tt.wantProject)
			}
			if !version.Equal(tt.wantVersion) {
				t.Errorf("version = %v, want %v", version, tt.wantVersion)
			}
		})
	}
}

func TestTemplateTagService_FindLatestTag(t *testing.T) {
	svc := adaptergit.NewTemplateTagService("", "")

	tags := []domain.Tag{
		{Name: "v1.0.0", Hash: "aaa"},
		{Name: "v1.1.0", Hash: "bbb"},
		{Name: "v2.0.0", Hash: "ccc"},
		{Name: "api/v1.0.0", Hash: "ddd"},
		{Name: "api/v1.1.0", Hash: "eee"},
	}

	t.Run("find latest repo tag", func(t *testing.T) {
		tag, err := svc.FindLatestTag(tags, "")
		if err != nil {
			t.Fatalf("FindLatestTag() error = %v", err)
		}
		if tag == nil {
			t.Fatal("expected tag, got nil")
		}
		if tag.Name != "v2.0.0" {
			t.Errorf("got tag %q, want %q", tag.Name, "v2.0.0")
		}
	})

	t.Run("find latest project tag", func(t *testing.T) {
		tag, err := svc.FindLatestTag(tags, "api")
		if err != nil {
			t.Fatalf("FindLatestTag() error = %v", err)
		}
		if tag == nil {
			t.Fatal("expected tag, got nil")
		}
		if tag.Name != "api/v1.1.0" {
			t.Errorf("got tag %q, want %q", tag.Name, "api/v1.1.0")
		}
	})

	t.Run("no matching project tags", func(t *testing.T) {
		tag, err := svc.FindLatestTag(tags, "nonexistent")
		if err != nil {
			t.Fatalf("FindLatestTag() error = %v", err)
		}
		if tag != nil {
			t.Errorf("expected nil, got %v", tag)
		}
	})
}
