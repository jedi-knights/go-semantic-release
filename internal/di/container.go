package di

import (
	"github.com/jedi-knights/go-semantic-release/internal/adapters/changelog"
	adaptergit "github.com/jedi-knights/go-semantic-release/internal/adapters/git"
	adaptergithub "github.com/jedi-knights/go-semantic-release/internal/adapters/github"
	adapterfs "github.com/jedi-knights/go-semantic-release/internal/adapters/fs"
	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/platform"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Container is the dependency injection container that wires all components.
type Container struct {
	config domain.Config
	logger ports.Logger
	workDir string

	// Singletons (lazily initialized).
	gitRepo       ports.GitRepository
	commitParser  ports.CommitParser
	fileSystem    ports.FileSystem
	tagService    ports.TagService
	versionCalc   ports.VersionCalculator
	changelogGen  ports.ChangelogGenerator
	publisher     ports.ReleasePublisher
	impactAnalyzer ports.ProjectImpactAnalyzer
	discoverer    ports.ProjectDiscoverer
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
		c.gitRepo = adaptergit.NewRepository(c.workDir)
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
		c.impactAnalyzer = adaptergit.NewPathBasedImpactAnalyzer(c.config.DependencyPropagation)
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

	// Config-defined projects take priority.
	if len(c.config.Projects) > 0 {
		discoverers = append(discoverers, adaptergit.NewConfiguredDiscoverer(c.config.Projects))
	}

	// Go workspace discovery.
	discoverers = append(discoverers, adaptergit.NewWorkspaceDiscoverer(fs))

	// Module discovery if enabled.
	if c.config.DiscoverModules {
		discoverers = append(discoverers, adaptergit.NewModuleDiscoverer(fs))
	}

	return adaptergit.NewCompositeDiscoverer(discoverers...)
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
