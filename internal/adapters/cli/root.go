package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	adapterconfig "github.com/jedi-knights/go-semantic-release/internal/adapters/config"
	"github.com/jedi-knights/go-semantic-release/internal/di"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/platform"
)

var (
	cfgFile string
	dryRun  bool
	project string
	jsonOut bool
	ciFlag  bool
	noCIFlag bool
	debug   bool
)

// NewRootCmd creates the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "semantic-release",
		Short: "Semantic release utility for Go projects",
		Long: `semantic-release is a native Go semantic release utility that analyzes
conventional commits to determine the next version, generate changelogs,
create tags, and publish releases. It supports monorepos with independent
project versioning.`,
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: .semantic-release.yaml)")
	root.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "perform a dry run without mutations")
	root.PersistentFlags().StringVar(&project, "project", "", "target a specific project in a monorepo")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")
	root.PersistentFlags().BoolVar(&ciFlag, "ci", false, "force CI mode")
	root.PersistentFlags().BoolVar(&noCIFlag, "no-ci", false, "skip CI environment verification")
	root.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	root.AddCommand(
		newReleaseCmd(),
		newPlanCmd(),
		newVersionCmd(),
		newChangelogCmd(),
		newDetectProjectsCmd(),
		newVerifyCmd(),
		newConfigCmd(),
	)

	return root
}

func buildContainer() (*di.Container, error) {
	provider := adapterconfig.NewViperProvider()
	cfg, err := provider.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Apply CLI flag overrides.
	if dryRun || viper.GetBool("dry_run") {
		cfg.DryRun = true
	}
	if debug {
		cfg.Debug = true
	}

	// CI detection: --ci forces CI mode, --no-ci disables it,
	// otherwise auto-detect and default dry-run when not in CI.
	isCI := platform.IsCI()
	if ciFlag {
		isCI = true
	}
	if noCIFlag {
		isCI = false
	}
	cfg.CI = isCI

	// When not in CI and dry-run wasn't explicitly set, default to dry-run.
	if !isCI && !dryRun {
		cfg.DryRun = true
	}

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	container := di.NewContainer(cfg, workDir)
	if cfg.Debug {
		container.WithLogger(platform.NewConsoleLogger(os.Stderr, platform.LogDebug))
	}

	return container, nil
}

func getConfig() (domain.Config, error) {
	provider := adapterconfig.NewViperProvider()
	return provider.Load(cfgFile)
}
