package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestReleaseType_String(t *testing.T) {
	tests := []struct {
		rt   domain.ReleaseType
		want string
	}{
		{domain.ReleaseNone, "none"},
		{domain.ReleasePatch, "patch"},
		{domain.ReleaseMinor, "minor"},
		{domain.ReleaseMajor, "major"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.rt.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReleaseType_Higher(t *testing.T) {
	tests := []struct {
		name string
		a, b domain.ReleaseType
		want domain.ReleaseType
	}{
		{"none vs patch", domain.ReleaseNone, domain.ReleasePatch, domain.ReleasePatch},
		{"patch vs minor", domain.ReleasePatch, domain.ReleaseMinor, domain.ReleaseMinor},
		{"major vs minor", domain.ReleaseMajor, domain.ReleaseMinor, domain.ReleaseMajor},
		{"same", domain.ReleasePatch, domain.ReleasePatch, domain.ReleasePatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Higher(tt.b); got != tt.want {
				t.Errorf("Higher() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReleaseType_IsReleasable(t *testing.T) {
	if domain.ReleaseNone.IsReleasable() {
		t.Error("ReleaseNone should not be releasable")
	}
	if !domain.ReleasePatch.IsReleasable() {
		t.Error("ReleasePatch should be releasable")
	}
}
