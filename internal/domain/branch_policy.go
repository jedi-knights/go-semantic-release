package domain

// BranchPolicy defines version behavior for a specific branch or branch pattern.
type BranchPolicy struct {
	Name       string // branch name or pattern (e.g. "main", "beta", "release/*")
	Channel    string // prerelease channel (e.g. "beta", "alpha", "rc")
	Prerelease bool   // if true, versions on this branch get prerelease identifiers
	IsDefault  bool   // if true, this is the main release branch
}

// DefaultBranchPolicies returns the standard branch configuration.
func DefaultBranchPolicies() []BranchPolicy {
	return []BranchPolicy{
		{Name: "main", IsDefault: true},
		{Name: "master", IsDefault: true},
		{Name: "next", Prerelease: true, Channel: "next"},
		{Name: "beta", Prerelease: true, Channel: "beta"},
		{Name: "alpha", Prerelease: true, Channel: "alpha"},
	}
}

// FindBranchPolicy returns the matching policy for a branch name, or nil if none matches.
func FindBranchPolicy(policies []BranchPolicy, branch string) *BranchPolicy {
	for i := range policies {
		if policies[i].Name == branch {
			return &policies[i]
		}
	}
	return nil
}
