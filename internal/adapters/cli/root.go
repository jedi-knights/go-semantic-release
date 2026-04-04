package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	adapterconfig "github.com/jedi-knights/go-semantic-release/internal/adapters/config"
	"github.com/jedi-knights/go-semantic-release/internal/di"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/platform"
)

// ErrQuietExit signals that the command has already printed its own error output
// and the caller (main) should exit with a non-zero code without printing anything
// further. This avoids duplicate error messages for commands like lint and verify
// that report structured output before failing.
var ErrQuietExit = errors.New("quiet exit")

// rootOptions holds all flag values bound to the root command. Using a struct
// rather than package-level vars eliminates the shared-mutable-state data race
// that occurs when Execute is called concurrently (e.g., in parallel tests).
type rootOptions struct {
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
	cfgFile       string
	project       string
	jsonOut       bool
	interactive   bool
	noInteractive bool
}

// NewRootCmd creates the root cobra command.
// The default action (no subcommand) runs the release, matching the original semantic-release behavior.
func NewRootCmd() *cobra.Command {
	opts := &rootOptions{}

	root := &cobra.Command{
		Use:   "semantic-release [options]",
		Short: "Run automated package publishing",
		Long: `semantic-release automates the whole package release workflow including:
determining the next version number, generating the release notes,
and publishing the package.

This is a native Go implementation compatible with the semantic-release CLI.`,
		// SilenceUsage prevents Cobra from printing the usage string on every RunE error.
		// SilenceErrors prevents Cobra from printing the error itself; main handles that
		// so it can filter ErrQuietExit without double-printing.
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRelease(cmd, args, opts)
		},
	}

	// Signal-aware context is wired in main via ExecuteContext so the stop()
	// function is always deferred regardless of whether the command succeeds or fails.

	// Flags matching the original semantic-release CLI (persistent so subcommands inherit them).
	pf := root.PersistentFlags()
	pf.StringArrayVarP(&opts.branches, "branches", "b", nil, "Git branches to release from (main/master are default; for other names set is_default in config)")
	pf.StringVarP(&opts.repositoryURL, "repository-url", "r", "", "Git repository URL")
	pf.StringVarP(&opts.tagFormat, "tag-format", "t", "", "Git tag format")
	pf.StringArrayVarP(&opts.plugins, "plugins", "p", nil, "Plugins")
	pf.StringArrayVarP(&opts.extends, "extends", "e", nil, "Shareable configurations")
	pf.BoolVarP(&opts.dryRun, "dry-run", "d", false, "Skip publishing")
	pf.BoolVar(&opts.ciFlag, "ci", false, "Toggle CI verifications")
	pf.BoolVar(&opts.noCIFlag, "no-ci", false, "Skip CI verifications")
	pf.BoolVar(&opts.debug, "debug", false, "Output debugging information")

	// Extension flags (Go-specific, also persistent).
	pf.StringVar(&opts.cfgFile, "config", "", "config file (default: .semantic-release.yaml)")
	pf.StringVar(&opts.project, "project", "", "target a specific project in a monorepo")
	pf.BoolVar(&opts.jsonOut, "json", false, "output in JSON format")
	pf.BoolVar(&opts.interactive, "interactive", false, "prompt for confirmation before release")
	pf.BoolVar(&opts.noInteractive, "no-interactive", false, "disable interactive prompts")

	// Subcommands are Go-specific extensions beyond the original semantic-release.
	root.AddCommand(
		newPlanCmd(opts),
		newVersionCmd(opts),
		newChangelogCmd(opts),
		newDetectProjectsCmd(opts),
		newVerifyCmd(opts),
		newConfigCmd(),
		newLintCmd(opts),
	)

	// Enforce mutually exclusive flag pairs. Cobra 1.5+ fires a clear error
	// message when both flags in a pair are set on the same invocation.
	// Note: these checks apply to the root command; for subcommands that
	// inherit the persistent flags, applyFlagAndEnvOverrides handles conflicts
	// via switch/case with first-wins semantics (ciFlag wins over noCIFlag when both are set).
	root.MarkFlagsMutuallyExclusive("ci", "no-ci")
	root.MarkFlagsMutuallyExclusive("interactive", "no-interactive")

	return root
}

// buildContainerWithWorkDir creates a DI container and also returns the resolved
// working directory so callers do not need a second os.Getwd() call.
func buildContainerWithWorkDir(opts *rootOptions) (*di.Container, string, error) {
	provider := adapterconfig.NewViperProvider()
	cfg, err := provider.Load(opts.cfgFile)
	if err != nil {
		return nil, "", fmt.Errorf("loading config: %w", err)
	}

	// Apply CLI flag and environment overrides.
	applyFlagAndEnvOverrides(&cfg, opts)

	workDir, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("getting working directory: %w", err)
	}

	container, err := di.NewContainer(cfg, workDir)
	if err != nil {
		return nil, "", fmt.Errorf("creating container: %w", err)
	}
	if cfg.Debug {
		container.WithLogger(platform.NewConsoleLogger(os.Stderr, platform.LogDebug))
	}

	return container, workDir, nil
}

// applyFlagAndEnvOverrides applies CLI flag values and environment-detected settings
// (e.g. CI auto-detection) on top of the loaded configuration.
func applyFlagAndEnvOverrides(cfg *domain.Config, opts *rootOptions) {
	if opts.dryRun {
		cfg.DryRun = true
	}
	if opts.debug {
		cfg.Debug = true
	}

	// --branches / -b overrides config branches.
	if len(opts.branches) > 0 {
		cfg.Branches = parseBranchFlags(opts.branches)
	}

	// --repository-url / -r overrides config.
	if opts.repositoryURL != "" {
		cfg.RepositoryURL = opts.repositoryURL
	}

	// --tag-format / -t overrides config.
	if opts.tagFormat != "" {
		cfg.TagFormat = opts.tagFormat
	}

	// --extends / -e overrides config.
	if len(opts.extends) > 0 {
		cfg.Extends = opts.extends
	}

	// --plugins / -p overrides config.
	if len(opts.plugins) > 0 {
		cfg.Plugins = opts.plugins
	}

	// CI detection: --ci forces CI mode, --no-ci disables it, otherwise
	// auto-detect via environment variables. platform.IsCI() is only called
	// when neither flag is set to avoid unnecessary env-var inspection.
	var isCI bool
	switch {
	case opts.ciFlag:
		isCI = true
	case opts.noCIFlag:
		isCI = false
	default:
		isCI = platform.IsCI()
	}
	cfg.CI = isCI

	// When not in CI and dry-run wasn't explicitly set, default to dry-run so
	// local runs never accidentally publish. Users who intend a real local release
	// must pass --no-ci explicitly.
	// Note: the user-facing warning for this auto dry-run is printed by runRelease
	// (not here) so it only appears for the release command, not plan/lint/verify/etc.
	if !isCI && !opts.dryRun {
		cfg.DryRun = true
	}
}

// parseBranchFlags converts CLI branch strings into BranchPolicy entries.
// Empty strings after trimming are silently dropped — this is intentional: a
// user passing --branches "" or trailing commas should not create a policy with
// a blank branch name.
func parseBranchFlags(branchNames []string) []domain.BranchPolicy {
	policies := make([]domain.BranchPolicy, 0, len(branchNames))
	for _, name := range branchNames {
		// Support comma-separated values.
		for _, n := range strings.Split(name, ",") {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			// IsDefault is set for the two canonical branch names used by most
			// Git hosting platforms. Any other name passed via --branches gets
			// IsDefault: false. Users who use a different default branch name
			// (e.g. "trunk") must set is_default in the config file instead.
			policies = append(policies, domain.BranchPolicy{
				Name:      n,
				IsDefault: n == "main" || n == "master",
			})
		}
	}
	return policies
}
