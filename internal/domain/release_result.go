package domain

// ReleaseResult captures the outcome of a release execution.
type ReleaseResult struct {
	Projects []ProjectReleaseResult
	DryRun   bool
}

// ProjectReleaseResult captures the outcome for a single project release.
type ProjectReleaseResult struct {
	Project        Project
	CurrentVersion Version // version before this release
	Version        Version // next (released) version
	TagName        string
	TagCreated     bool
	Published      bool
	PublishURL     string // e.g. GitHub release URL
	Changelog      string // rendered changelog content
	// Error holds the per-project failure. It is excluded from JSON serialization
	// because encoding/json cannot marshal arbitrary error interfaces. Use
	// ErrorMessage for JSON output.
	Error        error  `json:"-"`
	ErrorMessage string `json:"error,omitempty"` // human-readable form of Error for JSON consumers
	Skipped      bool
	SkipReason   string
}

// HasErrors returns true if any project release encountered an error.
func (rr ReleaseResult) HasErrors() bool {
	for i := range rr.Projects {
		if rr.Projects[i].Error != nil {
			return true
		}
	}
	return false
}

// SetError sets both the Error and ErrorMessage fields atomically, keeping
// them consistent. Use this instead of assigning the fields individually to
// prevent them from diverging.
func (pr *ProjectReleaseResult) SetError(err error) {
	pr.Error = err
	if err != nil {
		pr.ErrorMessage = err.Error()
	} else {
		pr.ErrorMessage = ""
	}
}

