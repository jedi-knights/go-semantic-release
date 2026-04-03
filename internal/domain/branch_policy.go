package domain

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// BranchType categorizes how a branch participates in the release process.
type BranchType string

const (
	BranchTypeRelease     BranchType = "release"
	BranchTypeMaintenance BranchType = "maintenance"
	BranchTypePrerelease  BranchType = "prerelease"
)

// BranchPolicy defines version behavior for a specific branch or branch pattern.
type BranchPolicy struct {
	Name       string     `mapstructure:"name"`
	Channel    string     `mapstructure:"channel"`
	Prerelease bool       `mapstructure:"prerelease"`
	IsDefault  bool       `mapstructure:"is_default"`
	Range      string     `mapstructure:"range"` // maintenance range e.g. "1.x", "1.0.x"
	Type       BranchType `mapstructure:"branch_type"`
}

// IsMaintenance returns true if this is a maintenance branch with a version range.
func (bp BranchPolicy) IsMaintenance() bool {
	return bp.Range != "" || bp.Type == BranchTypeMaintenance
}

// IsPrerelease returns true if this is a prerelease branch.
func (bp BranchPolicy) IsPrerelease() bool {
	return bp.Prerelease || bp.Type == BranchTypePrerelease
}

// MaintenanceRange parses the Range field into min/max version constraints.
// Range format: "N.x" (major only) or "N.N.x" (major.minor).
func (bp BranchPolicy) MaintenanceRange() (minVer, maxVer Version, err error) {
	r := bp.Range
	if r == "" {
		// Try to infer from branch name (e.g. "1.x", "1.0.x").
		r = bp.Name
	}

	return parseMaintenanceRange(r)
}

func parseMaintenanceRange(r string) (minVer, maxVer Version, err error) {
	parts := strings.Split(r, ".")
	if len(parts) < 2 || parts[len(parts)-1] != "x" {
		return minVer, maxVer, fmt.Errorf("invalid maintenance range %q: expected N.x or N.N.x", r)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return minVer, maxVer, fmt.Errorf("invalid major in range %q: %w", r, err)
	}

	if len(parts) == 2 {
		// "N.x" — allows any minor/patch within this major.
		return NewVersion(major, 0, 0), NewVersion(major+1, 0, 0), nil
	}

	if len(parts) == 3 {
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return minVer, maxVer, fmt.Errorf("invalid minor in range %q: %w", r, err)
		}
		// "N.N.x" — allows any patch within this major.minor.
		return NewVersion(major, minor, 0), NewVersion(major, minor+1, 0), nil
	}

	return minVer, maxVer, fmt.Errorf("invalid maintenance range %q", r)
}

// VersionInRange checks if a version falls within the maintenance branch range.
// minVer <= version < maxVer.
func VersionInRange(version, minVer, maxVer Version) bool {
	if version.Equal(minVer) {
		return true
	}
	return (version.GreaterThan(minVer) || version.Equal(minVer)) && maxVer.GreaterThan(version)
}

// ValidateMaintenanceVersion checks that a proposed version is valid for the maintenance range.
func ValidateMaintenanceVersion(proposed Version, policy BranchPolicy) error {
	if !policy.IsMaintenance() {
		return nil
	}

	minVer, maxVer, err := policy.MaintenanceRange()
	if err != nil {
		return err
	}

	if !VersionInRange(proposed, minVer, maxVer) {
		return fmt.Errorf(
			"version %s is outside maintenance range [%s, %s) for branch %q",
			proposed, minVer, maxVer, policy.Name,
		)
	}
	return nil
}

// DefaultBranchPolicies returns the standard branch configuration matching semantic-release defaults.
func DefaultBranchPolicies() []BranchPolicy {
	return []BranchPolicy{
		{Name: "main", IsDefault: true, Type: BranchTypeRelease},
		{Name: "master", IsDefault: true, Type: BranchTypeRelease},
		{Name: "next", Prerelease: true, Channel: "next", Type: BranchTypePrerelease},
		{Name: "next-major", Prerelease: true, Channel: "next-major", Type: BranchTypePrerelease},
		{Name: "beta", Prerelease: true, Channel: "beta", Type: BranchTypePrerelease},
		{Name: "alpha", Prerelease: true, Channel: "alpha", Type: BranchTypePrerelease},
	}
}

// FindBranchPolicy returns the matching policy for a branch name, or nil if none matches.
// Supports glob patterns in policy names.
func FindBranchPolicy(policies []BranchPolicy, branch string) *BranchPolicy {
	for i := range policies {
		if policies[i].Name == branch {
			return &policies[i]
		}
		// Try glob match for patterns like "release/*".
		if matched, _ := filepath.Match(policies[i].Name, branch); matched {
			return &policies[i]
		}
	}

	// Auto-detect maintenance branches by name pattern (e.g. "1.x", "1.0.x").
	if isMaintenancePattern(branch) {
		return &BranchPolicy{
			Name:    branch,
			Range:   branch,
			Channel: "release-" + branch,
			Type:    BranchTypeMaintenance,
		}
	}

	return nil
}

func isMaintenancePattern(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 2 || parts[len(parts)-1] != "x" {
		return false
	}
	_, err := strconv.Atoi(parts[0])
	return err == nil
}
