package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	adapterconfig "github.com/jedi-knights/go-semantic-release/internal/adapters/config"
	"github.com/jedi-knights/go-semantic-release/internal/di"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/platform"
)

// CLI flags matching the original semantic-release exactly.
var (
	// Original semantic-release flags.
	branches      []string
	repositoryURL string
	tagFormat     string
	plugins       []string
	extends       []string
	dryRun        bool
	ciFlag        bool
	noCIFlag      bool
	debug         bool

	// Extension flags (Go-specific).
	cfgFile string
	project string
	jsonOut bool
)

// NewRootCmd creates the root cobra command.
// The default action (no subcommand) runs the release, matching the original semantic-release behavior.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "semantic-release [options]",
		Short: "Run automated package publishing",
		Long: `semantic-release automates the whole package release workflow including:
determining the next version number, generating the release notes,
and publishing the package.

This is a native Go implementation compatible with the semantic-release CLI.`,
		SilenceUsage: true,
		RunE:         runRelease,
	}

	// Flags matching the original semantic-release CLI (persistent so subcommands inherit them).
	pf := root.PersistentFlags()
	pf.StringArrayVarP(&branches, "branches", "b", nil, "Git branches to release from")
	pf.StringVarP(&repositoryURL, "repository-url", "r", "", "Git repository URL")
	pf.StringVarP(&tagFormat, "tag-format", "t", "", "Git tag format")
	pf.StringArrayVarP(&plugins, "plugins", "p", nil, "Plugins")
	pf.StringArrayVarP(&extends, "extends", "e", nil, "Shareable configurations")
	pf.BoolVarP(&dryRun, "dry-run", "d", false, "Skip publishing")
	pf.BoolVar(&ciFlag, "ci", false, "Toggle CI verifications")
	pf.BoolVar(&noCIFlag, "no-ci", false, "Skip CI verifications")
	pf.BoolVar(&debug, "debug", false, "Output debugging information")

	// Extension flags (Go-specific, also persistent).
	pf.StringVar(&cfgFile, "config", "", "config file (default: .semantic-release.yaml)")
	pf.StringVar(&project, "project", "", "target a specific project in a monorepo")
	pf.BoolVar(&jsonOut, "json", false, "output in JSON format")

	// Subcommands are Go-specific extensions beyond the original semantic-release.
	root.AddCommand(
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

	// Apply CLI flag overrides matching original semantic-release behavior.
	applyFlagOverrides(&cfg)

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

func applyFlagOverrides(cfg *domain.Config) {
	if dryRun || viper.GetBool("dry_run") {
		cfg.DryRun = true
	}
	if debug {
		cfg.Debug = true
	}

	// --branches / -b overrides config branches.
	if len(branches) > 0 {
		cfg.Branches = parseBranchFlags(branches)
	}

	// --repository-url / -r overrides config.
	if repositoryURL != "" {
		cfg.RepositoryURL = repositoryURL
	}

	// --tag-format / -t overrides config.
	if tagFormat != "" {
		cfg.TagFormat = tagFormat
	}

	// --extends / -e overrides config.
	if len(extends) > 0 {
		cfg.Extends = extends
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
}

// parseBranchFlags converts CLI branch strings into BranchPolicy entries.
func parseBranchFlags(branchNames []string) []domain.BranchPolicy {
	var policies []domain.BranchPolicy
	for _, name := range branchNames {
		// Support comma-separated values.
		for _, n := range strings.Split(name, ",") {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			policies = append(policies, domain.BranchPolicy{
				Name:      n,
				IsDefault: n == "main" || n == "master",
			})
		}
	}
	return policies
}

func getConfig() (domain.Config, error) {
	provider := adapterconfig.NewViperProvider()
	return provider.Load(cfgFile)
}
