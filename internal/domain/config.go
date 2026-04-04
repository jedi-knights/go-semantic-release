package domain

import "slices"

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

	// GitLab integration.
	GitLab GitLabConfig `mapstructure:"gitlab"`

	// Bitbucket integration.
	Bitbucket BitbucketConfig `mapstructure:"bitbucket"`

	// Commit linting.
	Lint LintConfig `mapstructure:"lint"`

	// Interactive mode. nil means unset (treated as false by IsInteractive).
	Interactive *bool `mapstructure:"interactive"`

	// Git backend: "cli" (default) or "go-git".
	GitBackend string `mapstructure:"git_backend"`

	// Plugin references for external plugin loading.
	Plugins []string `mapstructure:"plugins"`

	// Extends allows inheriting from shared configurations.
	Extends []string `mapstructure:"extends"`
}

// GitLabConfig holds GitLab-specific settings.
type GitLabConfig struct {
	ProjectID string `mapstructure:"project_id"`
	// Token is the GitLab personal access token. SENSITIVE: do not log this field.
	Token         string   `mapstructure:"token"`
	APIURL        string   `mapstructure:"api_url"`
	CreateRelease bool     `mapstructure:"create_release"`
	Assets        []string `mapstructure:"assets"`
	Milestones    []string `mapstructure:"milestones"`
}

// BitbucketConfig holds Bitbucket-specific settings.
type BitbucketConfig struct {
	Workspace string `mapstructure:"workspace"`
	RepoSlug  string `mapstructure:"repo_slug"`
	// Token is the Bitbucket access token. SENSITIVE: do not log this field.
	Token         string `mapstructure:"token"`
	APIURL        string `mapstructure:"api_url"`
	CreateRelease bool   `mapstructure:"create_release"`
}

// PrepareConfig holds settings for the prepare lifecycle step.
type PrepareConfig struct {
	ChangelogFile   string   `mapstructure:"changelog_file"`
	VersionFile     string   `mapstructure:"version_file"`
	AdditionalFiles []string `mapstructure:"additional_files"`
}

// ProjectConfig defines a project in the configuration file.
type ProjectConfig struct {
	Name          string   `mapstructure:"name"`
	Path          string   `mapstructure:"path"`
	TagPrefix     string   `mapstructure:"tag_prefix"`
	Dependencies  []string `mapstructure:"dependencies"`
	ChangelogFile string   `mapstructure:"changelog_file"` // per-project changelog filename, relative to the project's path
}

// GitHubConfig holds GitHub-specific settings.
type GitHubConfig struct {
	Owner string `mapstructure:"owner"`
	Repo  string `mapstructure:"repo"`
	// Token is the GitHub personal access token. SENSITIVE: do not log this field.
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

// AnyProjectDefinesChangelog reports whether any configured project has a per-project changelog_file set.
func (c Config) AnyProjectDefinesChangelog() bool {
	return slices.ContainsFunc(c.Projects, func(p ProjectConfig) bool {
		return p.ChangelogFile != ""
	})
}

// IsInteractive returns whether interactive mode is enabled.
// Defaults to false when the Interactive field has not been set.
func (c Config) IsInteractive() bool {
	return c.Interactive != nil && *c.Interactive
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
		Lint:                  DefaultLintConfig(),
		GitAuthor:             DefaultGitIdentity(),
		GitCommitter:          DefaultGitIdentity(),
		GitBackend:            "cli",
		GitHub: GitHubConfig{
			CreateRelease: true,
		},
	}
}
