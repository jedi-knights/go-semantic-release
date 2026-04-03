package domain

// ReleasePlan describes what will happen during a release execution.
type ReleasePlan struct {
	Projects []ProjectReleasePlan
	DryRun   bool
	Branch   string
	Policy   *BranchPolicy
}

// ProjectReleasePlan describes the release plan for a single project.
type ProjectReleasePlan struct {
	Project        Project
	CurrentVersion Version
	NextVersion    Version
	ReleaseType    ReleaseType
	Commits        []Commit
	ShouldRelease  bool
	Reason         string // human-readable explanation
}

// HasReleasableProjects returns true if at least one project needs a release.
func (rp ReleasePlan) HasReleasableProjects() bool {
	for i := range rp.Projects {
		if rp.Projects[i].ShouldRelease {
			return true
		}
	}
	return false
}

// ReleasableProjects returns only the projects that need a release.
func (rp ReleasePlan) ReleasableProjects() []ProjectReleasePlan {
	var result []ProjectReleasePlan
	for i := range rp.Projects {
		if rp.Projects[i].ShouldRelease {
			result = append(result, rp.Projects[i])
		}
	}
	return result
}
