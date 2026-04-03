package domain

// Config holds all configuration for the release process.
type Config struct {
	// Core settings.
	ReleaseMode      ReleaseMode `mapstructure:"release_mode"`
	TagFormat        string      `mapstructure:"tag_format"`
	ProjectTagFormat string      `mapstructure:"project_tag_format"`
	DryRun           bool        `mapstructure:"dry_run"`
	CI               bool        `mapstructure:"ci"`
	Debug            bool        `mapstructure:"debug"`
	RepositoryURL    string      `mapstructure:"repository_url"`

	// Branch and version policies.
	Branches    []BranchPolicy         `mapstructure:"branches"`
	CommitTypes map[string]ReleaseType `mapstructure:"commit_types"`

	// Project/monorepo settings.
	Projects              []ProjectConfig `mapstructure:"projects"`
	DiscoverModules       bool            `mapstructure:"discover_modules"`
	IncludePaths          []string        `mapstructure:"include_paths"`
	ExcludePaths          []string        `mapstructure:"exclude_paths"`
	DependencyPropagation bool            `mapstructure:"dependency_propagation"`

	// Changelog settings.
	ChangelogSections []ChangelogSectionConfig `mapstructure:"changelog_sections"`
	ChangelogTemplate string                   `mapstructure:"changelog_template"`

	// Prepare step settings.
	Prepare PrepareConfig `mapstructure:"prepare"`

	// Git identity for automated commits.
	GitAuthor    GitIdentity `mapstructure:"git_author"`
	GitCommitter GitIdentity `mapstructure:"git_committer"`

	// GitHub integration.
	GitHub GitHubConfig `mapstructure:"github"`

	// Extends allows inheriting from shared configurations.
	Extends []string `mapstructure:"extends"`
}

// PrepareConfig holds settings for the prepare lifecycle step.
type PrepareConfig struct {
	ChangelogFile   string   `mapstructure:"changelog_file"`
	VersionFile     string   `mapstructure:"version_file"`
	AdditionalFiles []string `mapstructure:"additional_files"`
}

// ProjectConfig defines a project in the configuration file.
type ProjectConfig struct {
	Name         string   `mapstructure:"name"`
	Path         string   `mapstructure:"path"`
	TagPrefix    string   `mapstructure:"tag_prefix"`
	Dependencies []string `mapstructure:"dependencies"`
}

// GitHubConfig holds GitHub-specific settings.
type GitHubConfig struct {
	Owner                  string   `mapstructure:"owner"`
	Repo                   string   `mapstructure:"repo"`
	Token                  string   `mapstructure:"token"`
	APIURL                 string   `mapstructure:"api_url"`
	CreateRelease          bool     `mapstructure:"create_release"`
	DraftRelease           bool     `mapstructure:"draft_release"`
	Assets                 []string `mapstructure:"assets"`
	SuccessComment         string   `mapstructure:"success_comment"`
	FailComment            string   `mapstructure:"fail_comment"`
	ReleasedLabels         []string `mapstructure:"released_labels"`
	FailLabels             []string `mapstructure:"fail_labels"`
	DiscussionCategoryName string   `mapstructure:"discussion_category_name"`
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() Config {
	return Config{
		ReleaseMode:           ReleaseModeRepo,
		TagFormat:             "v{{.Version}}",
		ProjectTagFormat:      "{{.Project}}/v{{.Version}}",
		DryRun:                false,
		CI:                    true,
		Branches:              DefaultBranchPolicies(),
		CommitTypes:           DefaultCommitTypeMapping(),
		ChangelogSections:     DefaultChangelogSections(),
		DiscoverModules:       false,
		DependencyPropagation: false,
		GitAuthor:             DefaultGitIdentity(),
		GitCommitter:          DefaultGitIdentity(),
		GitHub: GitHubConfig{
			CreateRelease: true,
		},
	}
}
