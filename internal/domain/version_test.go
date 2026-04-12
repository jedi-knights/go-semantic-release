package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    domain.Version
		wantErr bool
	}{
		{
			name:  "basic version",
			input: "1.2.3",
			want:  domain.NewVersion(1, 2, 3),
		},
		{
			name:  "with v prefix",
			input: "v1.2.3",
			want:  domain.NewVersion(1, 2, 3),
		},
		{
			name:  "with prerelease",
			input: "1.2.3-beta.1",
			want:  domain.Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "beta.1"},
		},
		{
			name:  "with build metadata",
			input: "1.2.3+build.123",
			want:  domain.Version{Major: 1, Minor: 2, Patch: 3, Build: "build.123"},
		},
		{
			name:  "with prerelease and build",
			input: "v1.0.0-rc.1+sha.abc123",
			want:  domain.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "rc.1", Build: "sha.abc123"},
		},
		{
			name:  "zero version",
			input: "0.0.0",
			want:  domain.ZeroVersion(),
		},
		{
			name:    "invalid format",
			input:   "not-a-version",
			wantErr: true,
		},
		{
			name:    "only two parts",
			input:   "1.2",
			wantErr: true,
		},
		{
			name:    "non-numeric major",
			input:   "x.2.3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestVersion_Bump(t *testing.T) {
	tests := []struct {
		name     string
		version  domain.Version
		bump     domain.ReleaseType
		expected domain.Version
	}{
		{
			name:     "patch bump",
			version:  domain.NewVersion(1, 2, 3),
			bump:     domain.ReleasePatch,
			expected: domain.NewVersion(1, 2, 4),
		},
		{
			name:     "minor bump resets patch",
			version:  domain.NewVersion(1, 2, 3),
			bump:     domain.ReleaseMinor,
			expected: domain.NewVersion(1, 3, 0),
		},
		{
			name:     "major bump resets minor and patch",
			version:  domain.NewVersion(1, 2, 3),
			bump:     domain.ReleaseMajor,
			expected: domain.NewVersion(2, 0, 0),
		},
		{
			name:     "no bump returns same version",
			version:  domain.NewVersion(1, 2, 3),
			bump:     domain.ReleaseNone,
			expected: domain.NewVersion(1, 2, 3),
		},
		{
			name:     "bump from zero",
			version:  domain.ZeroVersion(),
			bump:     domain.ReleaseMinor,
			expected: domain.NewVersion(0, 1, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.Bump(tt.bump)
			if !got.Equal(tt.expected) {
				t.Errorf("Version.Bump(%v) = %v, want %v", tt.bump, got, tt.expected)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name    string
		version domain.Version
		want    string
	}{
		{"basic", domain.NewVersion(1, 2, 3), "1.2.3"},
		{"with prerelease", domain.Version{Major: 1, Prerelease: "beta"}, "1.0.0-beta"},
		{"with build", domain.Version{Major: 1, Build: "abc"}, "1.0.0+abc"},
		{"tag string", domain.NewVersion(1, 2, 3), "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersion_GreaterThan(t *testing.T) {
	tests := []struct {
		name string
		a, b domain.Version
		want bool
	}{
		{"major greater", domain.NewVersion(2, 0, 0), domain.NewVersion(1, 9, 9), true},
		{"minor greater", domain.NewVersion(1, 3, 0), domain.NewVersion(1, 2, 9), true},
		{"patch greater", domain.NewVersion(1, 2, 4), domain.NewVersion(1, 2, 3), true},
		{"equal", domain.NewVersion(1, 2, 3), domain.NewVersion(1, 2, 3), false},
		{"less", domain.NewVersion(1, 2, 3), domain.NewVersion(1, 2, 4), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.GreaterThan(tt.b); got != tt.want {
				t.Errorf("%v.GreaterThan(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestVersion_IsZero(t *testing.T) {
	if !domain.ZeroVersion().IsZero() {
		t.Error("ZeroVersion().IsZero() should be true")
	}
	if domain.NewVersion(1, 0, 0).IsZero() {
		t.Error("NewVersion(1,0,0).IsZero() should be false")
	}
}

func TestVersion_WithPrerelease(t *testing.T) {
	v := domain.NewVersion(1, 2, 3).WithPrerelease("beta.1")
	if v.Prerelease != "beta.1" {
		t.Errorf("WithPrerelease: got %q, want %q", v.Prerelease, "beta.1")
	}
	if v.String() != "1.2.3-beta.1" {
		t.Errorf("String() = %q, want %q", v.String(), "1.2.3-beta.1")
	}
}

func TestVersion_TagString(t *testing.T) {
	tests := []struct {
		name    string
		version domain.Version
		want    string
	}{
		{
			name:    "basic version",
			version: domain.NewVersion(1, 2, 3),
			want:    "v1.2.3",
		},
		{
			name:    "with prerelease",
			version: domain.Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "rc.1"},
			want:    "v1.0.0-rc.1",
		},
		{
			name:    "zero version",
			version: domain.ZeroVersion(),
			want:    "v0.0.0",
		},
		{
			name:    "with build metadata",
			version: domain.Version{Major: 2, Minor: 3, Patch: 4, Build: "sha.abc"},
			want:    "v2.3.4+sha.abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.TagString(); got != tt.want {
				t.Errorf("TagString() = %q, want %q", got, tt.want)
			}
		})
	}
}
