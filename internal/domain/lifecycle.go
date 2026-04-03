package domain

// Step represents a named step in the release lifecycle.
type Step string

const (
	StepVerifyConditions Step = "verifyConditions"
	StepAnalyzeCommits   Step = "analyzeCommits"
	StepVerifyRelease    Step = "verifyRelease"
	StepGenerateNotes    Step = "generateNotes"
	StepPrepare          Step = "prepare"
	StepPublish          Step = "publish"
	StepAddChannel       Step = "addChannel"
	StepSuccess          Step = "success"
	StepFail             Step = "fail"
)

// String returns the string representation of the lifecycle step.
func (s Step) String() string {
	return string(s)
}

// StepOrder defines the canonical execution order for lifecycle steps.
var StepOrder = []Step{
	StepVerifyConditions,
	StepAnalyzeCommits,
	StepVerifyRelease,
	StepGenerateNotes,
	StepPrepare,
	StepPublish,
	StepAddChannel,
	StepSuccess,
	StepFail,
}

// ReleaseContext holds all state passed through the lifecycle pipeline.
type ReleaseContext struct {
	Config         Config
	Branch         string
	BranchPolicy   *BranchPolicy
	Projects       []Project
	Commits        []Commit
	Plan           *ReleasePlan
	Result         *ReleaseResult
	DryRun         bool
	CI             bool
	RepositoryURL  string
	RepositoryRoot string

	// Per-project state populated during pipeline execution.
	CurrentProject *ProjectReleasePlan
	Notes          string // generated release notes for current project
	TagName        string
	Error          error // set when fail step is invoked
}

// GitIdentity configures the git author/committer for automated commits.
type GitIdentity struct {
	Name  string `mapstructure:"name"`
	Email string `mapstructure:"email"`
}

// DefaultGitIdentity returns the default bot identity.
func DefaultGitIdentity() GitIdentity {
	return GitIdentity{
		Name:  "semantic-release-bot",
		Email: "semantic-release-bot@users.noreply.github.com",
	}
}
