package config

import (
	"reflect"

	"github.com/go-viper/mapstructure/v2"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// StringToGitHubAssetHookFunc returns a mapstructure DecodeHookFuncType that
// converts a plain string into a domain.GitHubAsset with only the Path set.
// This enables backward-compatible YAML where assets can be listed as bare
// glob strings alongside the new structured {path, label} form.
func StringToGitHubAssetHookFunc() mapstructure.DecodeHookFuncType {
	assetType := reflect.TypeOf(domain.GitHubAsset{})
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if t != assetType {
			return data, nil
		}
		s, ok := data.(string)
		if !ok {
			return data, nil
		}
		return domain.GitHubAsset{Path: s}, nil
	}
}
