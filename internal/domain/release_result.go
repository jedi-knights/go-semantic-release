package domain

// ReleaseResult captures the outcome of a release execution.
type ReleaseResult struct {
	Projects []ProjectReleaseResult
	DryRun   bool
}

// ProjectReleaseResult captures the outcome for a single project release.
type ProjectReleaseResult struct {
	Project     Project
	Version     Version
	TagName     string
	TagCreated  bool
	Published   bool
	PublishURL  string // e.g. GitHub release URL
	Changelog   string // rendered changelog content
	Error       error
	Skipped     bool
	SkipReason  string
}

// HasErrors returns true if any project release encountered an error.
func (rr ReleaseResult) HasErrors() bool {
	for _, p := range rr.Projects {
		if p.Error != nil {
			return true
		}
	}
	return false
}

// Errors returns all project results that have errors.
func (rr ReleaseResult) Errors() []ProjectReleaseResult {
	var errs []ProjectReleaseResult
	for _, p := range rr.Projects {
		if p.Error != nil {
			errs = append(errs, p)
		}
	}
	return errs
}
