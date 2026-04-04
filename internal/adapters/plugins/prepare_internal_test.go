package plugins

import (
	"strings"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestChangelogPath(t *testing.T) {
	t.Parallel()
	withGlobal := &PreparePlugin{changelogFile: "CHANGELOG.md"}
	noGlobal := &PreparePlugin{}

	tests := []struct {
		name   string
		plugin *PreparePlugin
		rc     *domain.ReleaseContext
		want   string
	}{
		{
			name:   "per-project changelog resolves under project path",
			plugin: withGlobal,
			rc: &domain.ReleaseContext{
				RepositoryRoot: "/repo",
				CurrentProject: &domain.ProjectReleasePlan{
					Project: domain.Project{
						Path:          "services/auth-server",
						ChangelogFile: "CHANGELOG.md",
					},
				},
			},
			want: "/repo/services/auth-server/CHANGELOG.md",
		},
		{
			name:   "no per-project override falls back to global",
			plugin: withGlobal,
			rc: &domain.ReleaseContext{
				RepositoryRoot: "/repo",
				CurrentProject: &domain.ProjectReleasePlan{
					Project: domain.Project{Path: "services/api"},
				},
			},
			want: "/repo/CHANGELOG.md",
		},
		{
			name:   "both empty returns empty string",
			plugin: noGlobal,
			rc: &domain.ReleaseContext{
				RepositoryRoot: "/repo",
				CurrentProject: &domain.ProjectReleasePlan{
					Project: domain.Project{Path: "services/api"},
				},
			},
			want: "",
		},
		{
			name:   "nil CurrentProject falls back to global",
			plugin: withGlobal,
			rc: &domain.ReleaseContext{
				RepositoryRoot: "/repo",
				CurrentProject: nil,
			},
			want: "/repo/CHANGELOG.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.plugin.changelogPath(tt.rc)
			if got != tt.want {
				t.Errorf("changelogPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrependChangelog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		existing      string
		newEntry      string
		wantPrefix    string // expected prefix of result
		wantSubstring string // substring that must appear
	}{
		{
			name:       "empty existing creates header then entry",
			existing:   "",
			newEntry:   "## 1.0.0\n\n- first",
			wantPrefix: "# Changelog\n\n## 1.0.0",
		},
		{
			name:          "existing with title inserts after title",
			existing:      "# Changelog\n\n## 0.1.0\n\n- old\n",
			newEntry:      "## 1.0.0\n\n- new",
			wantPrefix:    "# Changelog\n\n## 1.0.0",
			wantSubstring: "## 0.1.0",
		},
		{
			name:          "existing without title prepends before content",
			existing:      "## 0.1.0\n\n- old\n",
			newEntry:      "## 1.0.0\n\n- new",
			wantPrefix:    "## 1.0.0",
			wantSubstring: "## 0.1.0",
		},
		{
			// A file with only the title line and no body. TrimLeft strips the trailing
			// newline from SplitN so the result ends with exactly one trailing newline.
			name:       "title-only existing inserts entry after title without extra blank lines",
			existing:   "# Changelog\n",
			newEntry:   "## 1.0.0\n\n- initial",
			wantPrefix: "# Changelog\n\n## 1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := prependChangelog(tt.existing, tt.newEntry)
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("prependChangelog() prefix = %q, want prefix %q", got, tt.wantPrefix)
			}
			if tt.wantSubstring != "" && !strings.Contains(got, tt.wantSubstring) {
				t.Errorf("prependChangelog() result missing %q:\n%s", tt.wantSubstring, got)
			}
		})
	}
}
