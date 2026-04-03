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

// DefaultLintConfig returns sensible default lint configuration.
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

// LintSeverity indicates the severity of a lint violation.
type LintSeverity string

const (
	LintError   LintSeverity = "error"
	LintWarning LintSeverity = "warning"
)

// LintViolation represents a single lint rule violation.
type LintViolation struct {
	Rule     string
	Message  string
	Severity LintSeverity
}
