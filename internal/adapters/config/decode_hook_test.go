package config

import (
	"reflect"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestStringToGitHubAssetHook_StringInput(t *testing.T) {
	t.Parallel()
	hook := StringToGitHubAssetHookFunc()
	from := reflect.TypeOf("")
	to := reflect.TypeOf(domain.GitHubAsset{})

	result, err := hook(from, to, "dist/*.tar.gz")
	if err != nil {
		t.Fatalf("hook error: %v", err)
	}

	asset, ok := result.(domain.GitHubAsset)
	if !ok {
		t.Fatalf("expected domain.GitHubAsset, got %T", result)
	}
	if asset.Path != "dist/*.tar.gz" {
		t.Errorf("Path = %q, want %q", asset.Path, "dist/*.tar.gz")
	}
	if asset.Label != "" {
		t.Errorf("Label = %q, want empty", asset.Label)
	}
}

func TestStringToGitHubAssetHook_NonTargetType_Passthrough(t *testing.T) {
	t.Parallel()
	hook := StringToGitHubAssetHookFunc()
	from := reflect.TypeOf("")
	to := reflect.TypeOf("")

	result, err := hook(from, to, "some-value")
	if err != nil {
		t.Fatalf("hook error: %v", err)
	}
	if result != "some-value" {
		t.Errorf("expected passthrough, got %v", result)
	}
}

func TestStringToGitHubAssetHook_NonStringInput_Passthrough(t *testing.T) {
	t.Parallel()
	hook := StringToGitHubAssetHookFunc()
	from := reflect.TypeOf(map[string]any{})
	to := reflect.TypeOf(domain.GitHubAsset{})

	input := map[string]any{"path": "dist/*.tar.gz", "label": "Tarballs"}
	result, err := hook(from, to, input)
	if err != nil {
		t.Fatalf("hook error: %v", err)
	}
	// Non-string input must pass through unchanged so mapstructure handles map→struct decoding.
	if _, isAsset := result.(domain.GitHubAsset); isAsset {
		t.Error("hook must not convert non-string input to GitHubAsset")
	}
}
