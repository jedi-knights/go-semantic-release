package config

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.ConfigProvider = (*ViperProvider)(nil)

// Supported config file names, searched in order (matching semantic-release conventions).
var configNames = []string{
	".semantic-release",
	".releaserc",
	"release.config",
}

// ViperProvider implements ports.ConfigProvider using Viper.
type ViperProvider struct{}

// NewViperProvider creates a new Viper-based config provider.
func NewViperProvider() *ViperProvider {
	return &ViperProvider{}
}

func (p *ViperProvider) Load(path string) (domain.Config, error) {
	cfg := domain.DefaultConfig()

	v := viper.New()
	v.SetEnvPrefix("SEMANTIC_RELEASE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		// Search for multiple config file names/formats.
		v.AddConfigPath(".")
		found := false
		for _, name := range configNames {
			v.SetConfigName(name)
			if err := v.ReadInConfig(); err == nil {
				found = true
				break
			}
		}
		if !found {
			// No config file found — use defaults + env only.
			return cfg, nil
		}
	}

	if path != "" {
		if err := v.ReadInConfig(); err != nil {
			return cfg, fmt.Errorf("reading config: %w", err)
		}
	}

	// StringToGitHubAssetHookFunc must run first so string values are promoted to
	// GitHubAsset{Path: s} before mapstructure attempts its own map→struct decode.
	if err := v.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			StringToGitHubAssetHookFunc(),
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	)); err != nil {
		return cfg, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Resolve extended configurations.
	if len(cfg.Extends) > 0 {
		resolved, err := ResolveExtends(cfg)
		if err != nil {
			return cfg, fmt.Errorf("resolving extends: %w", err)
		}
		cfg = resolved
	}

	return cfg, nil
}

// WriteDefaultConfig writes a default config file to the given path.
func WriteDefaultConfig(path string) error {
	v := viper.New()
	v.SetConfigType("yaml")

	v.Set("release_mode", "repo")
	v.Set("tag_format", "v{{.Version}}")
	v.Set("project_tag_format", "{{.Project}}/v{{.Version}}")
	v.Set("dry_run", false)
	v.Set("ci", true)
	v.Set("discover_modules", false)
	v.Set("dependency_propagation", false)
	v.Set("github.create_release", true)

	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("writing default config to %s: %w", path, err)
	}
	return nil
}
