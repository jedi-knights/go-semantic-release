package plugins

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// builtinAliases maps semantic-release plugin names to built-in equivalents.
var builtinAliases = map[string]bool{
	"@semantic-release/commit-analyzer":         true,
	"@semantic-release/release-notes-generator": true,
	"@semantic-release/changelog":               true,
	"@semantic-release/git":                     true,
	"@semantic-release/github":                  true,
	"@semantic-release/gitlab":                  true,
}

// LoadExternalPlugins resolves plugin references to Plugin instances.
// Built-in aliases are skipped (they're already wired in the DI container).
// External plugins are resolved as executables on $PATH or by absolute/relative path.
func LoadExternalPlugins(refs []string) ([]ports.Plugin, error) {
	result := make([]ports.Plugin, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}

		// Skip built-in aliases.
		if builtinAliases[ref] {
			continue
		}

		executable, err := resolveExecutable(ref)
		if err != nil {
			return nil, fmt.Errorf("resolving plugin %q: %w", ref, err)
		}

		name := filepath.Base(ref)
		result = append(result, NewExternalPlugin(name, executable))
	}

	return result, nil
}

func resolveExecutable(ref string) (string, error) {
	// If it's an absolute or relative path, use it directly.
	if strings.Contains(ref, "/") || strings.Contains(ref, string(filepath.Separator)) {
		absPath, err := filepath.Abs(ref)
		if err != nil {
			return "", err
		}
		return absPath, nil
	}

	// Try as a command on $PATH.
	path, err := exec.LookPath(ref)
	if err != nil {
		// Also try with "semantic-release-" prefix convention.
		prefixed := "semantic-release-" + ref
		path, err = exec.LookPath(prefixed)
		if err != nil {
			return "", fmt.Errorf("executable not found: tried %q and %q", ref, prefixed)
		}
	}

	return path, nil
}
