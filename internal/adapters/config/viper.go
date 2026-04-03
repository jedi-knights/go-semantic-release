package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// ViperProvider implements ports.ConfigProvider using Viper.
type ViperProvider struct{}

// NewViperProvider creates a new Viper-based config provider.
func NewViperProvider() *ViperProvider {
	return &ViperProvider{}
}

func (p *ViperProvider) Load(path string) (domain.Config, error) {
	cfg := domain.DefaultConfig()

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("GOSEMREL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName(".gosemrel")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && path != "" {
			return cfg, fmt.Errorf("reading config: %w", err)
		}
		// Config file not found is okay — use defaults + env.
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unmarshaling config: %w", err)
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
	v.Set("discover_modules", false)
	v.Set("dependency_propagation", false)
	v.Set("github.create_release", true)

	return v.WriteConfigAs(path)
}
