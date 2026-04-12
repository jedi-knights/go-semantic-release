package di_test

import (
	"context"
	"os"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/di"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/platform"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

func TestNewContainer_ValidWorkDir(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{}
	_, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v, want nil", err)
	}
}

func TestNewContainer_InvalidWorkDir(t *testing.T) {
	cfg := domain.Config{}
	_, err := di.NewContainer(cfg, "/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("NewContainer() error = nil, want error for non-existent workDir")
	}
}

func TestContainer_Config(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{
		TagFormat: "v{{.Version}}",
		DryRun:    true,
	}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	got := c.Config()
	if got.TagFormat != "v{{.Version}}" {
		t.Errorf("Config().TagFormat = %q, want v{{.Version}}", got.TagFormat)
	}
	if !got.DryRun {
		t.Error("Config().DryRun = false, want true")
	}
}

func TestContainer_Logger(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.Logger() == nil {
		t.Error("Logger() = nil, want non-nil")
	}
}

func TestContainer_WithLogger(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	newLogger := platform.NewConsoleLogger(os.Stderr, platform.LogInfo)
	returned := c.WithLogger(newLogger)

	// WithLogger returns self for chaining.
	if returned != c {
		t.Error("WithLogger() should return the same container")
	}
	// The new logger should be returned.
	if c.Logger() == nil {
		t.Error("Logger() = nil after WithLogger, want non-nil")
	}
}

func TestContainer_GitRepository(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.GitRepository() == nil {
		t.Error("GitRepository() = nil, want non-nil")
	}
}

func TestContainer_CommitParser(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.CommitParser() == nil {
		t.Error("CommitParser() = nil, want non-nil")
	}
}

func TestContainer_TagService(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.TagService() == nil {
		t.Error("TagService() = nil, want non-nil")
	}
}

func TestContainer_VersionCalculator(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.VersionCalculator() == nil {
		t.Error("VersionCalculator() = nil, want non-nil")
	}
}

func TestContainer_ChangelogGenerator(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ChangelogGenerator() == nil {
		t.Error("ChangelogGenerator() = nil, want non-nil")
	}
}

func TestContainer_FileSystem(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.FileSystem() == nil {
		t.Error("FileSystem() = nil, want non-nil")
	}
}

func TestContainer_GitRepository_GoGit(t *testing.T) {
	// A non-git temp dir causes go-git backend to fail to open, so it falls
	// back to the CLI git adapter. The result must still be non-nil.
	dir := t.TempDir()
	cfg := domain.Config{GitBackend: "go-git"}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	repo := c.GitRepository()
	if repo == nil {
		t.Error("GitRepository() = nil after go-git fallback, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// ReleasePublisher
// ---------------------------------------------------------------------------

func TestContainer_ReleasePublisher_NoopWhenCreateReleaseFalse(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{}
	// cfg.GitHub.CreateRelease is false by default — expect noopPublisher.
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	p := c.ReleasePublisher()
	if p == nil {
		t.Fatal("ReleasePublisher() = nil, want noopPublisher")
	}
}

func TestContainer_ReleasePublisher_GitHubWhenCreateReleaseTrue(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{
		GitHub: domain.GitHubConfig{
			CreateRelease: true,
			Owner:         "owner",
			Repo:          "repo",
			Token:         "token",
		},
	}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	p := c.ReleasePublisher()
	if p == nil {
		t.Fatal("ReleasePublisher() = nil, want GitHub publisher")
	}
}

// ---------------------------------------------------------------------------
// noopPublisher.Publish — exercised through ReleasePublisher when
// GitHub.CreateRelease is false.
// ---------------------------------------------------------------------------

func TestContainer_NoopPublisher_Publish_BackgroundContext(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	p := c.ReleasePublisher()

	_, publishErr := p.Publish(context.Background(), ports.PublishParams{})
	if publishErr != nil {
		t.Errorf("noopPublisher.Publish(background) = %v, want nil", publishErr)
	}
}

func TestContainer_NoopPublisher_Publish_CancelledContext(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	p := c.ReleasePublisher()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so ctx.Err() != nil

	_, publishErr := p.Publish(ctx, ports.PublishParams{})
	if publishErr == nil {
		t.Fatal("noopPublisher.Publish(cancelled ctx) = nil, want ctx error")
	}
}

// ---------------------------------------------------------------------------
// ProjectImpactAnalyzer
// ---------------------------------------------------------------------------

func TestContainer_ProjectImpactAnalyzer(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ProjectImpactAnalyzer() == nil {
		t.Error("ProjectImpactAnalyzer() = nil, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// ProjectDiscoverer (exercises buildDiscoverer)
// ---------------------------------------------------------------------------

func TestContainer_ProjectDiscoverer(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ProjectDiscoverer() == nil {
		t.Error("ProjectDiscoverer() = nil, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// Plugins — various config combinations
// ---------------------------------------------------------------------------

func TestContainer_Plugins_DefaultConfig(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	// Default config: no lint, no prepare, no GitHub release.
	// Expect exactly the three base plugins: git, commit-analyzer, release-notes.
	ps, pluginsErr := c.Plugins()
	if pluginsErr != nil {
		t.Fatalf("Plugins() error = %v, want nil", pluginsErr)
	}
	if len(ps) != 3 {
		t.Errorf("Plugins() len = %d, want 3", len(ps))
	}
}

func TestContainer_Plugins_LintEnabled(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{
		Lint: domain.LintConfig{Enabled: true},
	}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	ps, pluginsErr := c.Plugins()
	if pluginsErr != nil {
		t.Fatalf("Plugins() error = %v, want nil", pluginsErr)
	}
	// Base 3 plugins + lint plugin = 4.
	if len(ps) != 4 {
		t.Errorf("Plugins() with lint len = %d, want 4", len(ps))
	}
}

func TestContainer_Plugins_WithChangelogFile(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{
		Prepare: domain.PrepareConfig{
			ChangelogFile: "CHANGELOG.md",
		},
	}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	ps, pluginsErr := c.Plugins()
	if pluginsErr != nil {
		t.Fatalf("Plugins() error = %v, want nil", pluginsErr)
	}
	// Base 3 plugins + prepare plugin = 4.
	if len(ps) != 4 {
		t.Errorf("Plugins() with changelog file len = %d, want 4", len(ps))
	}
}

// ---------------------------------------------------------------------------
// Pipeline
// ---------------------------------------------------------------------------

func TestContainer_Pipeline(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	pipeline, pipelineErr := c.Pipeline()
	if pipelineErr != nil {
		t.Fatalf("Pipeline() error = %v, want nil", pipelineErr)
	}
	if pipeline == nil {
		t.Error("Pipeline() = nil, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// CommitAnalyzer
// ---------------------------------------------------------------------------

func TestContainer_CommitAnalyzer(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.CommitAnalyzer() == nil {
		t.Error("CommitAnalyzer() = nil, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// ProjectDetector
// ---------------------------------------------------------------------------

func TestContainer_ProjectDetector(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ProjectDetector() == nil {
		t.Error("ProjectDetector() = nil, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// ReleasePlanner
// ---------------------------------------------------------------------------

func TestContainer_ReleasePlanner(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ReleasePlanner() == nil {
		t.Error("ReleasePlanner() = nil, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// ReleaseExecutor
// ---------------------------------------------------------------------------

func TestContainer_ReleaseExecutor(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ReleaseExecutor() == nil {
		t.Error("ReleaseExecutor() = nil, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// ConditionVerifier
// ---------------------------------------------------------------------------

func TestContainer_ConditionVerifier(t *testing.T) {
	dir := t.TempDir()
	c, err := di.NewContainer(domain.Config{}, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ConditionVerifier() == nil {
		t.Error("ConditionVerifier() = nil, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// buildDiscoverer — branch coverage for non-default config combinations
// ---------------------------------------------------------------------------

func TestContainer_Pipeline_ExternalPluginError(t *testing.T) {
	// Configuring a plugin name that is not a builtin alias and is not on PATH
	// causes LoadExternalPlugins to fail, which propagates through buildPlugins →
	// Plugins() → Pipeline() as an error. This covers the error paths in all three.
	dir := t.TempDir()
	cfg := domain.Config{
		Plugins: []string{"no-such-plugin-for-test-coverage-xyz123"},
	}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	_, pluginsErr := c.Plugins()
	if pluginsErr == nil {
		t.Fatal("Plugins() expected error for missing external plugin, got nil")
	}

	_, pipelineErr := c.Pipeline()
	if pipelineErr == nil {
		t.Fatal("Pipeline() expected error when Plugins() fails, got nil")
	}
}

func TestContainer_BuildDiscoverer_WithProjects(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{
		Projects: []domain.ProjectConfig{
			{Name: "api", Path: "."},
		},
	}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ProjectDiscoverer() == nil {
		t.Error("ProjectDiscoverer() = nil, want non-nil")
	}
}

func TestContainer_BuildDiscoverer_WithModuleDiscovery(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{DiscoverModules: true}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ProjectDiscoverer() == nil {
		t.Error("ProjectDiscoverer() = nil, want non-nil")
	}
}

func TestContainer_BuildDiscoverer_WithCmdDiscovery(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{DiscoverCmd: true}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	if c.ProjectDiscoverer() == nil {
		t.Error("ProjectDiscoverer() = nil, want non-nil")
	}
}

// ---------------------------------------------------------------------------
// buildPlugins — GitLab and Bitbucket plugin registration branches
// ---------------------------------------------------------------------------

func TestContainer_Plugins_GitLabEnabled(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{
		GitLab: domain.GitLabConfig{
			CreateRelease: true,
			Token:         "gl-token",
			APIURL:        "https://gitlab.example.com/api/v4",
		},
	}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	ps, pluginsErr := c.Plugins()
	if pluginsErr != nil {
		t.Fatalf("Plugins() error = %v, want nil", pluginsErr)
	}
	// Base 3 plugins + GitLab plugin = 4.
	if len(ps) != 4 {
		t.Errorf("Plugins() with GitLab len = %d, want 4", len(ps))
	}
}

func TestContainer_Plugins_BitbucketEnabled(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{
		Bitbucket: domain.BitbucketConfig{
			CreateRelease: true,
			Token:         "bb-token",
		},
	}
	c, err := di.NewContainer(cfg, dir)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	ps, pluginsErr := c.Plugins()
	if pluginsErr != nil {
		t.Fatalf("Plugins() error = %v, want nil", pluginsErr)
	}
	// Base 3 plugins + Bitbucket plugin = 4.
	if len(ps) != 4 {
		t.Errorf("Plugins() with Bitbucket len = %d, want 4", len(ps))
	}
}
