package di

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

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
// It is safe for concurrent use: lazy singletons are protected by mu; the logger
// is stored atomically so WithLogger has no ordering constraints.
// fileSystem is the exception: it is initialized via fileSystemOnce (sync.Once)
// rather than mu so that ProjectDiscoverer can call FileSystem() while already
// holding mu without deadlocking.
type Container struct {
	mu        sync.Mutex
	config    domain.Config // immutable after construction; read without mu is safe
	loggerPtr atomic.Pointer[ports.Logger]
	workDir   string

	// fileSystem is initialized exactly once via fileSystemOnce; all other
	// singletons below are lazily initialized under mu.
	fileSystemOnce sync.Once
	fileSystem     ports.FileSystem

	// Singletons (lazily initialized; access only while holding mu).
	gitRepo        ports.GitRepository
	commitParser   ports.CommitParser
	tagService     ports.TagService
	versionCalc    ports.VersionCalculator
	changelogGen   ports.ChangelogGenerator
	publisher      ports.ReleasePublisher
	impactAnalyzer ports.ProjectImpactAnalyzer
	discoverer     ports.ProjectDiscoverer

	// Plugin list is built exactly once via pluginsOnce.
	pluginsOnce sync.Once
	pluginList  []ports.Plugin
	pluginsErr  error
}

// NewContainer creates a DI container with the given config.
// workDir must be an absolute path to the repository root; construction fails if the path does not exist.
func NewContainer(config domain.Config, workDir string) (*Container, error) {
	if _, err := os.Stat(workDir); err != nil {
		return nil, fmt.Errorf("invalid workDir %q: %w", workDir, err)
	}
	c := &Container{config: config, workDir: workDir}
	var l ports.Logger = platform.DefaultLogger()
	c.loggerPtr.Store(&l)
	return c, nil
}

// WithLogger overrides the logger. Safe to call concurrently with any other method.
func (c *Container) WithLogger(logger ports.Logger) *Container {
	c.loggerPtr.Store(&logger)
	return c
}

// Logger returns the current logger. Safe to call concurrently.
func (c *Container) Logger() ports.Logger {
	return *c.loggerPtr.Load()
}

// Config returns the container's configuration. config is immutable after
// construction so this is safe to call without holding mu.
func (c *Container) Config() domain.Config {
	return c.config
}

// GitRepository returns the configured git repository implementation (CLI or go-git).
func (c *Container) GitRepository() ports.GitRepository {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gitRepo == nil {
		if c.config.GitBackend == "go-git" {
			repo, err := adaptergogit.NewRepository(c.workDir)
			if err != nil {
				c.Logger().Warn("failed to open go-git repository, falling back to CLI", "error", err)
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

// CommitParser returns the conventional commit parser.
func (c *Container) CommitParser() ports.CommitParser {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.commitParser == nil {
		c.commitParser = adaptergit.NewConventionalCommitParser()
	}
	return c.commitParser
}

// FileSystem returns the OS filesystem adapter.
// Initialization is guarded by fileSystemOnce rather than mu so that
// ProjectDiscoverer (and buildPlugins) can call FileSystem() while already
// holding mu without deadlocking.
func (c *Container) FileSystem() ports.FileSystem {
	c.fileSystemOnce.Do(func() {
		c.fileSystem = adapterfs.NewOSFileSystem()
	})
	return c.fileSystem
}

// TagService returns the tag formatter configured for this repository's tag format.
func (c *Container) TagService() ports.TagService {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tagService == nil {
		c.tagService = adaptergit.NewTemplateTagService(c.config.TagFormat, c.config.ProjectTagFormat)
	}
	return c.tagService
}

// VersionCalculator returns the semver bump calculator.
func (c *Container) VersionCalculator() ports.VersionCalculator {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.versionCalc == nil {
		c.versionCalc = app.NewVersionCalculatorService()
	}
	return c.versionCalc
}

// ChangelogGenerator returns the template-based changelog generator.
func (c *Container) ChangelogGenerator() ports.ChangelogGenerator {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.changelogGen == nil {
		c.changelogGen = changelog.NewTemplateGenerator(c.config.ChangelogTemplate)
	}
	return c.changelogGen
}

// ReleasePublisher returns the configured release publisher.
// When github.create_release is false, a no-op publisher is returned so callers
// that receive the publisher from the container never need to nil-check the result.
// MustNewReleaseExecutor panics on nil publisher; callers that bypass the container
// must pass noopPublisher{} explicitly when publishing is disabled.
func (c *Container) ReleasePublisher() ports.ReleasePublisher {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.publisher == nil {
		if c.config.GitHub.CreateRelease {
			c.publisher = adaptergithub.NewPublisher(
				c.config.GitHub.Owner,
				c.config.GitHub.Repo,
				c.config.GitHub.Token,
			)
		} else {
			c.publisher = noopPublisher{}
		}
	}
	return c.publisher
}

// ProjectImpactAnalyzer returns the path-based project impact analyzer.
func (c *Container) ProjectImpactAnalyzer() ports.ProjectImpactAnalyzer {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.impactAnalyzer == nil {
		c.impactAnalyzer = adaptergit.NewPathBasedImpactAnalyzer(c.config.DependencyPropagation, c.config.IncludePaths, c.config.ExcludePaths)
	}
	return c.impactAnalyzer
}

// ProjectDiscoverer returns the composite project discoverer for this repository.
func (c *Container) ProjectDiscoverer() ports.ProjectDiscoverer {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.discoverer == nil {
		c.discoverer = c.buildDiscoverer()
	}
	return c.discoverer
}

// buildDiscoverer must be called with mu held. It calls c.FileSystem() safely
// because FileSystem uses its own fileSystemOnce and does not acquire mu.
func (c *Container) buildDiscoverer() ports.ProjectDiscoverer {
	fs := c.FileSystem()
	var discoverers []ports.ProjectDiscoverer

	if len(c.config.Projects) > 0 {
		discoverers = append(discoverers, adaptergit.NewConfiguredDiscoverer(c.config.Projects))
	}
	// WorkspaceDiscoverer is always appended so that repos without explicit project config
	// are still discovered via go.work. Note: CompositeDiscoverer uses first-wins semantics,
	// so if ConfiguredDiscoverer returns results, WorkspaceDiscoverer is NOT called and
	// additional go.work modules are silently skipped. A future MergingDiscoverer would be
	// needed to combine both sources.
	discoverers = append(discoverers, adaptergit.NewWorkspaceDiscoverer(fs))
	if c.config.DiscoverModules {
		discoverers = append(discoverers, adaptergit.NewModuleDiscoverer(fs))
	}

	return adaptergit.NewCompositeDiscoverer(discoverers...)
}

// Plugins builds and caches the ordered list of lifecycle plugins based on config.
// The list is constructed exactly once (via sync.Once) regardless of concurrent callers.
// Returns an error if any explicitly configured external plugin fails to load.
//
// Once a load error occurs the sync.Once body is complete and will never re-run,
// so subsequent calls will always return (nil, err) for the same permanent error.
// Callers that need to react to transient failures must create a new Container.
//
// Concurrency note: pluginsOnce.Do intentionally does NOT hold c.mu. Each
// accessor called inside Do (GitRepository, CommitParser, etc.) acquires c.mu
// independently for its own singleton initialisation. Holding c.mu across the
// entire Do body would cause a deadlock because those accessors also try to
// acquire c.mu. sync.Once provides its own mutual-exclusion guarantee for the
// Do body itself, so no additional locking is required here.
func (c *Container) Plugins() ([]ports.Plugin, error) {
	c.pluginsOnce.Do(func() {
		c.pluginList, c.pluginsErr = c.buildPlugins()
	})
	return c.pluginList, c.pluginsErr
}

// buildPlugins constructs the ordered plugin list. It must only be called from
// within pluginsOnce.Do; sync.Once provides the mutual-exclusion guarantee.
//
// Side effect: calling buildPlugins initializes most container singletons
// (GitRepository, TagService, FileSystem, CommitParser, ChangelogGenerator, …)
// as a side effect of constructing their respective plugins. A first call to
// Plugins() is therefore equivalent to eagerly initialising the majority of
// the container's infrastructure.
func (c *Container) buildPlugins() ([]ports.Plugin, error) {
	logger := c.Logger()

	ps := []ports.Plugin{
		// Git plugin: verifyConditions + publish (tag + push).
		plugins.NewGitPlugin(
			c.GitRepository(),
			c.TagService(),
			c.FileSystem(),
			logger,
			c.config.GitAuthor,
			c.config.Prepare.AdditionalFiles,
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
	// Register if the global prepare config is set OR any project defines a per-project changelog_file.
	if c.config.Prepare.ChangelogFile != "" || c.config.Prepare.VersionFile != "" || c.config.AnyProjectDefinesChangelog() {
		ps = append(ps, plugins.NewPreparePlugin(
			c.FileSystem(),
			logger,
			c.config.Prepare,
		))
	}

	// Lint plugin: verifyRelease (commit message linting).
	if c.config.Lint.Enabled {
		lintCfg := c.config.Lint
		if len(lintCfg.AllowedTypes) == 0 {
			// No allowed types configured — fall back to the full default set.
			lintCfg = domain.DefaultEnabledLintConfig()
		}
		ps = append(ps, plugins.NewLintPlugin(
			adapterlint.NewConventionalLinter(lintCfg),
			logger,
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
			logger,
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
			logger,
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
			logger,
		))
	}

	// External plugins from config/flags.
	if len(c.config.Plugins) > 0 {
		external, err := plugins.LoadExternalPlugins(c.config.Plugins)
		if err != nil {
			// Log at Error and surface to the caller so the pipeline can be gated.
			// Return (nil, err) so callers cannot accidentally use a partial list.
			logger.Error("failed to load external plugins", "error", err)
			return nil, err
		}
		ps = append(ps, external...)
	}

	return ps, nil
}

// noopPublisher is a null-object implementation of ports.ReleasePublisher.
// It is used when github.create_release is false so that ReleasePublisher()
// always returns a non-nil value and callers never need to nil-check.
type noopPublisher struct{}

func (noopPublisher) Publish(ctx context.Context, _ ports.PublishParams) (domain.ProjectReleaseResult, error) {
	if ctx.Err() != nil {
		return domain.ProjectReleaseResult{}, ctx.Err()
	}
	return domain.ProjectReleaseResult{}, nil
}

// Pipeline creates a lifecycle pipeline with all configured plugins.
// Returns an error if any explicitly configured external plugin failed to load.
func (c *Container) Pipeline() (*app.Pipeline, error) {
	ps, err := c.Plugins()
	if err != nil {
		return nil, err
	}
	return app.NewPipeline(ps, c.Logger()), nil
}

// CommitAnalyzer creates a CommitAnalyzer use case.
// A new instance is returned on each call — this is intentional. Use cases are
// lightweight value objects; the underlying infrastructure (GitRepository, CommitParser)
// is shared via their own singletons.
func (c *Container) CommitAnalyzer() *app.CommitAnalyzer {
	return app.NewCommitAnalyzer(c.GitRepository(), c.CommitParser(), c.Logger())
}

// ProjectDetector creates a ProjectDetector use case.
// A new instance is returned on each call (intentional — see CommitAnalyzer).
func (c *Container) ProjectDetector() *app.ProjectDetector {
	return app.NewProjectDetector(c.ProjectDiscoverer(), c.Logger())
}

// ReleasePlanner creates a ReleasePlanner use case.
// A new instance is returned on each call (intentional — see CommitAnalyzer).
func (c *Container) ReleasePlanner() *app.ReleasePlanner {
	return app.NewReleasePlanner(
		c.GitRepository(),
		c.TagService(),
		c.VersionCalculator(),
		c.ProjectImpactAnalyzer(),
		c.Logger(),
		c.config.CommitTypes,
	)
}

// ReleaseExecutor creates a ReleaseExecutor use case.
// A new instance is returned on each call (intentional — see CommitAnalyzer).
func (c *Container) ReleaseExecutor() *app.ReleaseExecutor {
	return app.MustNewReleaseExecutor(
		c.GitRepository(),
		c.TagService(),
		c.ChangelogGenerator(),
		c.ReleasePublisher(),
		c.Logger(),
		c.config.ChangelogSections,
	)
}

// ConditionVerifier creates a ConditionVerifier use case.
// A new instance is returned on each call (intentional — see CommitAnalyzer).
func (c *Container) ConditionVerifier() *app.ConditionVerifier {
	return app.NewConditionVerifier(c.GitRepository(), c.config, c.Logger())
}
