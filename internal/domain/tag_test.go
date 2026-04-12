package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestDefaultRepoTagFormat(t *testing.T) {
	tf := domain.DefaultRepoTagFormat()
	if tf.Template == "" {
		t.Error("DefaultRepoTagFormat().Template must not be empty")
	}
}

func TestDefaultProjectTagFormat(t *testing.T) {
	tf := domain.DefaultProjectTagFormat()
	if tf.Template == "" {
		t.Error("DefaultProjectTagFormat().Template must not be empty")
	}
}

func TestParseProjectFromTag(t *testing.T) {
	tests := []struct {
		name        string
		tagName     string
		prefix      string
		wantProject string
		wantVersion domain.Version
		wantErr     bool
	}{
		{
			name:        "repo-level tag no prefix",
			tagName:     "v1.2.3",
			prefix:      "",
			wantProject: "",
			wantVersion: domain.NewVersion(1, 2, 3),
		},
		{
			name:        "project tag with slash prefix",
			tagName:     "my-service/v2.0.0",
			prefix:      "my-service/",
			wantProject: "my-service",
			wantVersion: domain.NewVersion(2, 0, 0),
		},
		{
			name:        "project tag with @ separator",
			tagName:     "my-service@1.0.1",
			prefix:      "",
			wantProject: "my-service",
			wantVersion: domain.NewVersion(1, 0, 1),
		},
		{
			name:    "prefix mismatch returns error",
			tagName: "other-service/v1.0.0",
			prefix:  "my-service/",
			wantErr: true,
		},
		{
			name:    "invalid version in tag",
			tagName: "v-not-a-version",
			prefix:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, version, err := domain.ParseProjectFromTag(tt.tagName, tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseProjectFromTag(%q, %q) error = %v, wantErr = %v", tt.tagName, tt.prefix, err, tt.wantErr)
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
