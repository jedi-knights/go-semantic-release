package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

const maxExtendsDepth = 10

// ResolveExtends loads and merges all extended configurations into the base config.
// Supports local file paths and HTTP(S) URLs. Detects cycles.
func ResolveExtends(base domain.Config) (domain.Config, error) {
	return resolveExtendsRecursive(base, make(map[string]bool), 0)
}

func resolveExtendsRecursive(cfg domain.Config, seen map[string]bool, depth int) (domain.Config, error) {
	if depth > maxExtendsDepth {
		return cfg, fmt.Errorf("extends chain exceeds maximum depth of %d", maxExtendsDepth)
	}

	if len(cfg.Extends) == 0 {
		return cfg, nil
	}

	// Process extends in order; later entries override earlier ones.
	for _, ref := range cfg.Extends {
		if seen[ref] {
			return cfg, fmt.Errorf("circular extends detected: %q", ref)
		}
		seen[ref] = true

		parent, err := loadExtendsRef(ref)
		if err != nil {
			return cfg, fmt.Errorf("loading extends %q: %w", ref, err)
		}

		// Recursively resolve the parent's own extends.
		parent, err = resolveExtendsRecursive(parent, seen, depth+1)
		if err != nil {
			return cfg, err
		}

		// Merge: base values take precedence over parent.
		cfg = MergeConfigs(cfg, parent)
	}

	return cfg, nil
}

func loadExtendsRef(ref string) (domain.Config, error) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return loadExtendsFromURL(ref)
	}
	return loadExtendsFromFile(ref)
}

func loadExtendsFromFile(path string) (domain.Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return domain.Config{}, fmt.Errorf("resolving path: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(absPath)

	if err := v.ReadInConfig(); err != nil {
		return domain.Config{}, fmt.Errorf("reading config file: %w", err)
	}

	var cfg domain.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return domain.Config{}, fmt.Errorf("unmarshaling config: %w", err)
	}

	return cfg, nil
}

func loadExtendsFromURL(rawURL string) (domain.Config, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return domain.Config{}, fmt.Errorf("creating request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.Config{}, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.Config{}, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.Config{}, fmt.Errorf("reading response: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "semantic-release-extends-*.yaml")
	if err != nil {
		return domain.Config{}, fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return domain.Config{}, fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	return loadExtendsFromFile(tmpFile.Name())
}
