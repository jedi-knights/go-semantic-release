package di

import (
	"github.com/jedi-knights/go-semantic-release/internal/adapters/bitbucket"
	"github.com/jedi-knights/go-semantic-release/internal/adapters/changelog"
	adapterfs "github.com/jedi-knights/go-semantic-release/internal/adapters/fs"
	adaptergit "github.com/jedi-knights/go-semantic-release/internal/adapters/git"
	adaptergithub "github.com/jedi-knights/go-semantic-release/internal/adapters/github"
	"github.com/jedi-knights/go-semantic-release/internal/adapters/gitlab"
	adaptergogit "github.com/jedi-knights/go-semantic-release/internal/adapters/gogit"
	adapterlint "github.com/jedi-knights/go-semantic-release/internal/adapters/lint"
	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/platform"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Container is the dependency injection container that wires all components.
type Container struct {
	config  domain.Config
	logger  ports.Logger
	workDir string

	// Singletons (lazily initialized).
	gitRepo        ports.GitRepository
	commitParser   ports.CommitParser
	fileSystem     ports.FileSystem
	tagService     ports.TagService
	versionCalc    ports.VersionCalculator
	changelogGen   ports.ChangelogGenerator
	publisher      ports.ReleasePublisher
	impactAnalyzer ports.ProjectImpactAnalyzer
	discoverer     ports.ProjectDiscoverer
}

// NewContainer creates a DI container with the given config.
func NewContainer(config domain.Config, workDir string) *Container {
	return &Container{
		config:  config,
		workDir: workDir,
		logger:  platform.DefaultLogger(),
	}
}

// WithLogger overrides the logger.
func (c *Container) WithLogger(logger ports.Logger) *Container {
	c.logger = logger
	return c
}

func (c *Container) Logger() ports.Logger {
	return c.logger
}

func (c *Container) Config() domain.Config {
	return c.config
}

func (c *Container) GitRepository() ports.GitRepository {
	if c.gitRepo == nil {
		if c.config.GitBackend == "go-git" {
			repo, err := adaptergogit.NewRepository(c.workDir)
			if err != nil {
				c.logger.Warn("failed to open go-git repository, falling back to CLI", "error", err)
				c.gitRepo = adaptergit.NewRepository(c.workDir)
			} else {
				c.gitRepo = repo
			}
		} else {
			c.gitRepo = adaptergit.NewRepository(c.workDir)
		}
	}
	return c.gitRepo
}

func (c *Container) CommitParser() ports.CommitParser {
	if c.commitParser == nil {
		c.commitParser = adaptergit.NewConventionalCommitParser()
	}
	return c.commitParser
}

func (c *Container) FileSystem() ports.FileSystem {
	if c.fileSystem == nil {
		c.fileSystem = adapterfs.NewOSFileSystem()
	}
	return c.fileSystem
}

func (c *Container) TagService() ports.TagService {
	if c.tagService == nil {
		c.tagService = adaptergit.NewTemplateTagService(c.config.TagFormat, c.config.ProjectTagFormat)
	}
	return c.tagService
}

func (c *Container) VersionCalculator() ports.VersionCalculator {
	if c.versionCalc == nil {
		c.versionCalc = app.NewVersionCalculatorService()
	}
	return c.versionCalc
}

func (c *Container) ChangelogGenerator() ports.ChangelogGenerator {
	if c.changelogGen == nil {
		c.changelogGen = changelog.NewTemplateGenerator(c.config.ChangelogTemplate)
	}
	return c.changelogGen
}

func (c *Container) ReleasePublisher() ports.ReleasePublisher {
	if c.publisher == nil && c.config.GitHub.CreateRelease {
		c.publisher = adaptergithub.NewPublisher(
			c.config.GitHub.Owner,
			c.config.GitHub.Repo,
			c.config.GitHub.Token,
		)
	}
	return c.publisher
}

func (c *Container) ProjectImpactAnalyzer() ports.ProjectImpactAnalyzer {
	if c.impactAnalyzer == nil {
		c.impactAnalyzer = adaptergit.NewPathBasedImpactAnalyzer(c.config.DependencyPropagation, c.config.IncludePaths, c.config.ExcludePaths)
	}
	return c.impactAnalyzer
}

func (c *Container) ProjectDiscoverer() ports.ProjectDiscoverer {
	if c.discoverer == nil {
		c.discoverer = c.buildDiscoverer()
	}
	return c.discoverer
}

func (c *Container) buildDiscoverer() ports.ProjectDiscoverer {
	fs := c.FileSystem()
	var discoverers []ports.ProjectDiscoverer

	if len(c.config.Projects) > 0 {
		discoverers = append(discoverers, adaptergit.NewConfiguredDiscoverer(c.config.Projects))
	}
	discoverers = append(discoverers, adaptergit.NewWorkspaceDiscoverer(fs))
	if c.config.DiscoverModules {
		discoverers = append(discoverers, adaptergit.NewModuleDiscoverer(fs))
	}

	return adaptergit.NewCompositeDiscoverer(discoverers...)
}

// Plugins builds the ordered list of lifecycle plugins based on config.
func (c *Container) Plugins() []ports.Plugin {
	ps := []ports.Plugin{
		// Git plugin: verifyConditions + publish (tag + push).
		plugins.NewGitPlugin(
			c.GitRepository(),
			c.TagService(),
			c.FileSystem(),
			c.logger,
			c.config.GitAuthor,
			nil,
		),
		// Commit analyzer plugin: analyzeCommits.
		plugins.NewCommitAnalyzerPlugin(
			c.CommitParser(),
			c.config.CommitTypes,
		),
		// Release notes plugin: generateNotes.
		plugins.NewReleaseNotesPlugin(
			c.ChangelogGenerator(),
			c.config.ChangelogSections,
		),
	}

	// Prepare plugin: update CHANGELOG.md, VERSION files.
	if c.config.Prepare.ChangelogFile != "" || c.config.Prepare.VersionFile != "" {
		ps = append(ps, plugins.NewPreparePlugin(
			c.FileSystem(),
			c.logger,
			plugins.PrepareConfig{
				ChangelogFile:   c.config.Prepare.ChangelogFile,
				VersionFile:     c.config.Prepare.VersionFile,
				AdditionalFiles: c.config.Prepare.AdditionalFiles,
			},
		))
	}

	// Lint plugin: verifyRelease (commit message linting).
	if c.config.Lint.Enabled {
		lintCfg := c.config.Lint
		if len(lintCfg.AllowedTypes) == 0 {
			lintCfg = domain.DefaultLintConfig()
			lintCfg.Enabled = true
		}
		ps = append(ps, plugins.NewLintPlugin(
			adapterlint.NewConventionalLinter(lintCfg),
			c.logger,
		))
	}

	// GitHub plugin: verifyConditions + publish + addChannel + success + fail.
	if c.config.GitHub.CreateRelease {
		ps = append(ps, adaptergithub.NewPlugin(
			adaptergithub.PluginConfig{
				Owner:                  c.config.GitHub.Owner,
				Repo:                   c.config.GitHub.Repo,
				Token:                  c.config.GitHub.Token,
				APIURL:                 c.config.GitHub.APIURL,
				Assets:                 c.config.GitHub.Assets,
				DraftRelease:           c.config.GitHub.DraftRelease,
				DiscussionCategoryName: c.config.GitHub.DiscussionCategoryName,
				SuccessComment:         c.config.GitHub.SuccessComment,
				FailComment:            c.config.GitHub.FailComment,
				ReleasedLabels:         c.config.GitHub.ReleasedLabels,
				FailLabels:             c.config.GitHub.FailLabels,
			},
			c.logger,
		))
	}

	// GitLab plugin: verifyConditions + publish + addChannel + success + fail.
	if c.config.GitLab.CreateRelease {
		ps = append(ps, gitlab.NewPlugin(
			gitlab.PluginConfig{
				ProjectID:  c.config.GitLab.ProjectID,
				Token:      c.config.GitLab.Token,
				APIURL:     c.config.GitLab.APIURL,
				Assets:     c.config.GitLab.Assets,
				Milestones: c.config.GitLab.Milestones,
			},
			c.logger,
		))
	}

	// Bitbucket plugin: verifyConditions + publish + addChannel + success + fail.
	if c.config.Bitbucket.CreateRelease {
		ps = append(ps, bitbucket.NewPlugin(
			bitbucket.PluginConfig{
				Workspace: c.config.Bitbucket.Workspace,
				RepoSlug:  c.config.Bitbucket.RepoSlug,
				Token:     c.config.Bitbucket.Token,
				APIURL:    c.config.Bitbucket.APIURL,
			},
			c.logger,
		))
	}

	// External plugins from config/flags.
	if len(c.config.Plugins) > 0 {
		external, err := plugins.LoadExternalPlugins(c.config.Plugins)
		if err != nil {
			c.logger.Warn("failed to load external plugins", "error", err)
		} else {
			ps = append(ps, external...)
		}
	}

	return ps
}

// Pipeline creates a lifecycle pipeline with all configured plugins.
func (c *Container) Pipeline() *app.Pipeline {
	return app.NewPipeline(c.Plugins(), c.logger)
}

// CommitAnalyzer creates a CommitAnalyzer use case.
func (c *Container) CommitAnalyzer() *app.CommitAnalyzer {
	return app.NewCommitAnalyzer(c.GitRepository(), c.CommitParser(), c.logger)
}

// ProjectDetector creates a ProjectDetector use case.
func (c *Container) ProjectDetector() *app.ProjectDetector {
	return app.NewProjectDetector(c.ProjectDiscoverer(), c.logger)
}

// ReleasePlanner creates a ReleasePlanner use case.
func (c *Container) ReleasePlanner() *app.ReleasePlanner {
	return app.NewReleasePlanner(
		c.GitRepository(),
		c.TagService(),
		c.VersionCalculator(),
		c.ProjectImpactAnalyzer(),
		c.logger,
		c.config.CommitTypes,
	)
}

// ReleaseExecutor creates a ReleaseExecutor use case.
func (c *Container) ReleaseExecutor() *app.ReleaseExecutor {
	return app.NewReleaseExecutor(
		c.GitRepository(),
		c.TagService(),
		c.ChangelogGenerator(),
		c.ReleasePublisher(),
		c.logger,
		c.config.ChangelogSections,
	)
}

// ConditionVerifier creates a ConditionVerifier use case.
func (c *Container) ConditionVerifier() *app.ConditionVerifier {
	return app.NewConditionVerifier(c.GitRepository(), c.config, c.logger)
}
