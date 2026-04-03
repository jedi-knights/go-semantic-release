package lint

import (
	"fmt"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.CommitLinter = (*ConventionalLinter)(nil)

// ConventionalLinter validates commits against conventional commit rules.
type ConventionalLinter struct {
	config domain.LintConfig
}

// NewConventionalLinter creates a new conventional commit linter.
func NewConventionalLinter(cfg domain.LintConfig) *ConventionalLinter {
	return &ConventionalLinter{config: cfg}
}

// Lint checks a single commit against all configured rules.
func (l *ConventionalLinter) Lint(commit domain.Commit) []domain.LintViolation {
	var violations []domain.LintViolation

	violations = append(violations, l.checkType(commit)...)
	violations = append(violations, l.checkScope(commit)...)
	violations = append(violations, l.checkDescription(commit)...)
	violations = append(violations, l.checkSubjectLength(commit)...)
	violations = append(violations, l.checkBody(commit)...)

	return violations
}

func (l *ConventionalLinter) checkType(commit domain.Commit) []domain.LintViolation {
	if commit.Type == "" {
		return []domain.LintViolation{{
			Rule:     "type-empty",
			Message:  "commit message must have a type (e.g., feat, fix)",
			Severity: domain.LintError,
		}}
	}

	if len(l.config.AllowedTypes) > 0 {
		allowed := false
		for _, t := range l.config.AllowedTypes {
			if commit.Type == t {
				allowed = true
				break
			}
		}
		if !allowed {
			return []domain.LintViolation{{
				Rule:     "type-enum",
				Message:  fmt.Sprintf("type %q is not allowed; allowed types: %s", commit.Type, strings.Join(l.config.AllowedTypes, ", ")),
				Severity: domain.LintError,
			}}
		}
	}

	return nil
}

func (l *ConventionalLinter) checkScope(commit domain.Commit) []domain.LintViolation {
	if l.config.RequireScope && commit.Scope == "" {
		return []domain.LintViolation{{
			Rule:     "scope-empty",
			Message:  "commit message must have a scope",
			Severity: domain.LintError,
		}}
	}

	if commit.Scope != "" && len(l.config.AllowedScopes) > 0 {
		allowed := false
		for _, s := range l.config.AllowedScopes {
			if commit.Scope == s {
				allowed = true
				break
			}
		}
		if !allowed {
			return []domain.LintViolation{{
				Rule:     "scope-enum",
				Message:  fmt.Sprintf("scope %q is not allowed; allowed scopes: %s", commit.Scope, strings.Join(l.config.AllowedScopes, ", ")),
				Severity: domain.LintError,
			}}
		}
	}

	return nil
}

func (l *ConventionalLinter) checkDescription(commit domain.Commit) []domain.LintViolation {
	if commit.Description == "" {
		return []domain.LintViolation{{
			Rule:     "description-empty",
			Message:  "commit message must have a description",
			Severity: domain.LintError,
		}}
	}

	if strings.HasSuffix(commit.Description, ".") {
		return []domain.LintViolation{{
			Rule:     "description-trailing-period",
			Message:  "description must not end with a period",
			Severity: domain.LintWarning,
		}}
	}

	return nil
}

func (l *ConventionalLinter) checkSubjectLength(commit domain.Commit) []domain.LintViolation {
	if l.config.MaxSubjectLength <= 0 {
		return nil
	}

	subject := commit.Message
	if idx := strings.IndexByte(subject, '\n'); idx >= 0 {
		subject = subject[:idx]
	}

	if len(subject) > l.config.MaxSubjectLength {
		return []domain.LintViolation{{
			Rule:     "subject-max-length",
			Message:  fmt.Sprintf("subject line is %d characters, maximum is %d", len(subject), l.config.MaxSubjectLength),
			Severity: domain.LintWarning,
		}}
	}

	return nil
}

func (l *ConventionalLinter) checkBody(commit domain.Commit) []domain.LintViolation {
	if l.config.RequireBody && commit.Body == "" {
		return []domain.LintViolation{{
			Rule:     "body-empty",
			Message:  "commit message must have a body",
			Severity: domain.LintWarning,
		}}
	}

	return nil
}
