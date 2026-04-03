package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version (major.minor.patch) with optional prerelease and build metadata.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
}

// ZeroVersion returns a 0.0.0 version.
func ZeroVersion() Version {
	return Version{}
}

// NewVersion creates a version from major, minor, patch components.
func NewVersion(major, minor, patch int) Version {
	return Version{Major: major, Minor: minor, Patch: patch}
}

// ParseVersion parses a semantic version string. Accepts optional "v" prefix.
func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")

	// Split off build metadata first.
	build := ""
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		build = s[idx+1:]
		s = s[:idx]
	}

	// Split off prerelease.
	prerelease := ""
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		prerelease = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q: expected major.minor.patch", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version %q: %w", parts[2], err)
	}

	return Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
		Build:      build,
	}, nil
}

// String returns the version as "major.minor.patch[-prerelease][+build]".
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// TagString returns the version with a "v" prefix.
func (v Version) TagString() string {
	return "v" + v.String()
}

// IsZero returns true if this is the zero version (0.0.0 with no prerelease/build).
func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0 && v.Prerelease == "" && v.Build == ""
}

// Bump returns a new version incremented by the given release type.
func (v Version) Bump(rt ReleaseType) Version {
	switch rt {
	case ReleaseMajor:
		return NewVersion(v.Major+1, 0, 0)
	case ReleaseMinor:
		return NewVersion(v.Major, v.Minor+1, 0)
	case ReleasePatch:
		return NewVersion(v.Major, v.Minor, v.Patch+1)
	default:
		return v
	}
}

// WithPrerelease returns a copy with the given prerelease identifier.
func (v Version) WithPrerelease(pre string) Version {
	v.Prerelease = pre
	v.Build = ""
	return v
}

// GreaterThan returns true if v is greater than other.
func (v Version) GreaterThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	return v.Patch > other.Patch
}

// Equal returns true if versions are identical (excluding build metadata per semver spec).
func (v Version) Equal(other Version) bool {
	return v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch == other.Patch &&
		v.Prerelease == other.Prerelease
}
