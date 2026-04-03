package domain

// Config holds all configuration for the release process.
type Config struct {
	ReleaseMode        ReleaseMode              `mapstructure:"release_mode"`
	TagFormat          string                   `mapstructure:"tag_format"`
	ProjectTagFormat   string                   `mapstructure:"project_tag_format"`
	DryRun             bool                     `mapstructure:"dry_run"`
	Projects           []ProjectConfig          `mapstructure:"projects"`
	Branches           []BranchPolicy           `mapstructure:"branches"`
	CommitTypes        map[string]ReleaseType   `mapstructure:"commit_types"`
	ChangelogSections  []ChangelogSectionConfig `mapstructure:"changelog_sections"`
	ChangelogTemplate  string                   `mapstructure:"changelog_template"`
	GitHub             GitHubConfig             `mapstructure:"github"`
	DiscoverModules    bool                     `mapstructure:"discover_modules"`
	IncludePaths       []string                 `mapstructure:"include_paths"`
	ExcludePaths       []string                 `mapstructure:"exclude_paths"`
	DependencyPropagation bool                  `mapstructure:"dependency_propagation"`
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
	Owner      string `mapstructure:"owner"`
	Repo       string `mapstructure:"repo"`
	Token      string `mapstructure:"token"`
	CreateRelease bool `mapstructure:"create_release"`
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() Config {
	return Config{
		ReleaseMode:      ReleaseModeRepo,
		TagFormat:        "v{{.Version}}",
		ProjectTagFormat: "{{.Project}}/v{{.Version}}",
		DryRun:           false,
		Branches:         DefaultBranchPolicies(),
		CommitTypes:      DefaultCommitTypeMapping(),
		ChangelogSections: DefaultChangelogSections(),
		DiscoverModules:  false,
		DependencyPropagation: false,
		GitHub: GitHubConfig{
			CreateRelease: true,
		},
	}
}
