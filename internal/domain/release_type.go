package domain

// ReleaseType represents the kind of version bump required.
type ReleaseType int

const (
	// ReleaseNone indicates no release is needed.
	ReleaseNone ReleaseType = iota
	// ReleasePatch indicates a patch version bump.
	ReleasePatch
	// ReleaseMinor indicates a minor version bump.
	ReleaseMinor
	// ReleaseMajor indicates a major version bump.
	ReleaseMajor
)

// String returns the human-readable name.
func (rt ReleaseType) String() string {
	switch rt {
	case ReleasePatch:
		return "patch"
	case ReleaseMinor:
		return "minor"
	case ReleaseMajor:
		return "major"
	default:
		return "none"
	}
}

// Higher returns the higher of two release types.
func (rt ReleaseType) Higher(other ReleaseType) ReleaseType {
	if other > rt {
		return other
	}
	return rt
}

// IsReleasable returns true if this type requires a version bump.
func (rt ReleaseType) IsReleasable() bool {
	return rt > ReleaseNone
}
