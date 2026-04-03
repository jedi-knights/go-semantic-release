package config

import "github.com/jedi-knights/go-semantic-release/internal/domain"

// MergeConfigs merges a parent config into a base config.
// Base values take precedence over parent (child overrides parent).
func MergeConfigs(base, parent domain.Config) domain.Config {
	if base.ReleaseMode == "" {
		base.ReleaseMode = parent.ReleaseMode
	}
	if base.TagFormat == "" {
		base.TagFormat = parent.TagFormat
	}
	if base.ProjectTagFormat == "" {
		base.ProjectTagFormat = parent.ProjectTagFormat
	}
	if base.RepositoryURL == "" {
		base.RepositoryURL = parent.RepositoryURL
	}
	if base.GitBackend == "" {
		base.GitBackend = parent.GitBackend
	}
	if base.ChangelogTemplate == "" {
		base.ChangelogTemplate = parent.ChangelogTemplate
	}

	if len(base.Branches) == 0 {
		base.Branches = parent.Branches
	}
	if len(base.CommitTypes) == 0 {
		base.CommitTypes = parent.CommitTypes
	}
	if len(base.Projects) == 0 {
		base.Projects = parent.Projects
	}
	if len(base.IncludePaths) == 0 {
		base.IncludePaths = parent.IncludePaths
	}
	if len(base.ExcludePaths) == 0 {
		base.ExcludePaths = parent.ExcludePaths
	}
	if len(base.ChangelogSections) == 0 {
		base.ChangelogSections = parent.ChangelogSections
	}
	if len(base.Plugins) == 0 {
		base.Plugins = parent.Plugins
	}

	base.Prepare = mergePrepare(base.Prepare, parent.Prepare)
	base.GitHub = mergeGitHub(base.GitHub, parent.GitHub)
	base.GitLab = mergeGitLab(base.GitLab, parent.GitLab)
	base.Bitbucket = mergeBitbucket(base.Bitbucket, parent.Bitbucket)
	base.Lint = mergeLint(base.Lint, parent.Lint)
	base.GitAuthor = mergeIdentity(base.GitAuthor, parent.GitAuthor)
	base.GitCommitter = mergeIdentity(base.GitCommitter, parent.GitCommitter)

	return base
}

func mergePrepare(base, parent domain.PrepareConfig) domain.PrepareConfig {
	if base.ChangelogFile == "" {
		base.ChangelogFile = parent.ChangelogFile
	}
	if base.VersionFile == "" {
		base.VersionFile = parent.VersionFile
	}
	if len(base.AdditionalFiles) == 0 {
		base.AdditionalFiles = parent.AdditionalFiles
	}
	return base
}

func mergeGitHub(base, parent domain.GitHubConfig) domain.GitHubConfig {
	if base.Owner == "" {
		base.Owner = parent.Owner
	}
	if base.Repo == "" {
		base.Repo = parent.Repo
	}
	if base.Token == "" {
		base.Token = parent.Token
	}
	if base.APIURL == "" {
		base.APIURL = parent.APIURL
	}
	if len(base.Assets) == 0 {
		base.Assets = parent.Assets
	}
	if base.SuccessComment == "" {
		base.SuccessComment = parent.SuccessComment
	}
	if base.FailComment == "" {
		base.FailComment = parent.FailComment
	}
	if len(base.ReleasedLabels) == 0 {
		base.ReleasedLabels = parent.ReleasedLabels
	}
	if len(base.FailLabels) == 0 {
		base.FailLabels = parent.FailLabels
	}
	return base
}

func mergeGitLab(base, parent domain.GitLabConfig) domain.GitLabConfig {
	if base.ProjectID == "" {
		base.ProjectID = parent.ProjectID
	}
	if base.Token == "" {
		base.Token = parent.Token
	}
	if base.APIURL == "" {
		base.APIURL = parent.APIURL
	}
	if len(base.Assets) == 0 {
		base.Assets = parent.Assets
	}
	return base
}

func mergeBitbucket(base, parent domain.BitbucketConfig) domain.BitbucketConfig {
	if base.Workspace == "" {
		base.Workspace = parent.Workspace
	}
	if base.RepoSlug == "" {
		base.RepoSlug = parent.RepoSlug
	}
	if base.Token == "" {
		base.Token = parent.Token
	}
	if base.APIURL == "" {
		base.APIURL = parent.APIURL
	}
	return base
}

func mergeLint(base, parent domain.LintConfig) domain.LintConfig {
	if !base.Enabled && parent.Enabled {
		base.Enabled = true
	}
	if base.MaxSubjectLength == 0 {
		base.MaxSubjectLength = parent.MaxSubjectLength
	}
	if len(base.AllowedTypes) == 0 {
		base.AllowedTypes = parent.AllowedTypes
	}
	if len(base.AllowedScopes) == 0 {
		base.AllowedScopes = parent.AllowedScopes
	}
	return base
}

func mergeIdentity(base, parent domain.GitIdentity) domain.GitIdentity {
	if base.Name == "" {
		base.Name = parent.Name
	}
	if base.Email == "" {
		base.Email = parent.Email
	}
	return base
}
