package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	adapterconfig "github.com/jedi-knights/go-semantic-release/internal/adapters/config"
	"github.com/jedi-knights/go-semantic-release/internal/di"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

var (
	cfgFile string
	dryRun  bool
	project string
	jsonOut bool
)

// NewRootCmd creates the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gosemrel",
		Short: "Semantic release utility for Go projects",
		Long: `gosemrel is a native Go semantic release utility that analyzes
conventional commits to determine the next version, generate changelogs,
create tags, and publish releases. It supports monorepos with independent
project versioning.`,
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: .gosemrel.yaml)")
	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "perform a dry run without mutations")
	root.PersistentFlags().StringVar(&project, "project", "", "target a specific project in a monorepo")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")

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

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	return di.NewContainer(cfg, workDir), nil
}

func getConfig() (domain.Config, error) {
	provider := adapterconfig.NewViperProvider()
	return provider.Load(cfgFile)
}
