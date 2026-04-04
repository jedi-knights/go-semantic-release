package domain

// LintConfig holds configuration for commit message linting.
type LintConfig struct {
	Enabled            bool     `mapstructure:"enabled"`
	MaxSubjectLength   int      `mapstructure:"max_subject_length"`
	RequireScope       bool     `mapstructure:"require_scope"`
	AllowedTypes       []string `mapstructure:"allowed_types"`
	AllowedScopes      []string `mapstructure:"allowed_scopes"`
	RequireBody        bool     `mapstructure:"require_body"`
	RequireDescription bool     `mapstructure:"require_description"`
}

// DefaultLintConfig returns sensible default lint configuration with linting disabled.
func DefaultLintConfig() LintConfig {
	return LintConfig{
		Enabled:          false,
		MaxSubjectLength: 72,
		AllowedTypes: []string{
			"feat", "fix", "docs", "style", "refactor",
			"perf", "test", "build", "ci", "chore", "revert",
		},
	}
}

// DefaultEnabledLintConfig returns the default lint configuration with linting enabled.
// Use this instead of DefaultLintConfig() when you need a ready-to-use linting setup.
func DefaultEnabledLintConfig() LintConfig {
	cfg := DefaultLintConfig()
	cfg.Enabled = true
	return cfg
}

// LintSeverity indicates the severity of a lint violation.
type LintSeverity string

const (
	// LintError indicates a violation that must be corrected before release.
	LintError LintSeverity = "error"
	// LintWarning indicates a violation that should be reviewed but does not block release.
	LintWarning LintSeverity = "warning"
)

// LintViolation represents a single lint rule violation.
type LintViolation struct {
	Rule     string
	Message  string
	Severity LintSeverity
}
