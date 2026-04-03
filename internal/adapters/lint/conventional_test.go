package lint_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/lint"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestConventionalLinter_Lint(t *testing.T) {
	defaultCfg := domain.DefaultLintConfig()

	tests := []struct {
		name       string
		config     domain.LintConfig
		commit     domain.Commit
		wantCount  int
		wantRules  []string
		wantErrors int
	}{
		{
			name:   "valid feat commit",
			config: defaultCfg,
			commit: domain.Commit{
				Type:        "feat",
				Description: "add user login",
				Message:     "feat: add user login",
			},
			wantCount: 0,
		},
		{
			name:   "missing type",
			config: defaultCfg,
			commit: domain.Commit{
				Description: "update readme",
				Message:     "update readme",
			},
			wantCount:  1,
			wantRules:  []string{"type-empty"},
			wantErrors: 1,
		},
		{
			name:   "disallowed type",
			config: defaultCfg,
			commit: domain.Commit{
				Type:        "wip",
				Description: "work in progress",
				Message:     "wip: work in progress",
			},
			wantCount:  1,
			wantRules:  []string{"type-enum"},
			wantErrors: 1,
		},
		{
			name:   "missing description",
			config: defaultCfg,
			commit: domain.Commit{
				Type:    "feat",
				Message: "feat:",
			},
			wantCount:  1,
			wantRules:  []string{"description-empty"},
			wantErrors: 1,
		},
		{
			name:   "description trailing period",
			config: defaultCfg,
			commit: domain.Commit{
				Type:        "fix",
				Description: "fix the bug.",
				Message:     "fix: fix the bug.",
			},
			wantCount: 1,
			wantRules: []string{"description-trailing-period"},
		},
		{
			name: "subject too long",
			config: domain.LintConfig{
				MaxSubjectLength: 20,
				AllowedTypes:     defaultCfg.AllowedTypes,
			},
			commit: domain.Commit{
				Type:        "feat",
				Description: "this is a very long description that exceeds the limit",
				Message:     "feat: this is a very long description that exceeds the limit",
			},
			wantCount: 1,
			wantRules: []string{"subject-max-length"},
		},
		{
			name: "scope required but missing",
			config: domain.LintConfig{
				RequireScope: true,
				AllowedTypes: defaultCfg.AllowedTypes,
			},
			commit: domain.Commit{
				Type:        "feat",
				Description: "add login",
				Message:     "feat: add login",
			},
			wantCount:  1,
			wantRules:  []string{"scope-empty"},
			wantErrors: 1,
		},
		{
			name: "scope not in allowed list",
			config: domain.LintConfig{
				AllowedTypes:  defaultCfg.AllowedTypes,
				AllowedScopes: []string{"api", "core"},
			},
			commit: domain.Commit{
				Type:        "feat",
				Scope:       "unknown",
				Description: "add login",
				Message:     "feat(unknown): add login",
			},
			wantCount:  1,
			wantRules:  []string{"scope-enum"},
			wantErrors: 1,
		},
		{
			name: "body required but missing",
			config: domain.LintConfig{
				AllowedTypes: defaultCfg.AllowedTypes,
				RequireBody:  true,
			},
			commit: domain.Commit{
				Type:        "feat",
				Description: "add login",
				Message:     "feat: add login",
			},
			wantCount: 1,
			wantRules: []string{"body-empty"},
		},
		{
			name: "valid commit with scope and body",
			config: domain.LintConfig{
				AllowedTypes:  defaultCfg.AllowedTypes,
				AllowedScopes: []string{"api"},
				RequireScope:  true,
				RequireBody:   true,
			},
			commit: domain.Commit{
				Type:        "feat",
				Scope:       "api",
				Description: "add login",
				Body:        "Added OAuth2 login flow",
				Message:     "feat(api): add login",
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linter := lint.NewConventionalLinter(tt.config)
			violations := linter.Lint(tt.commit)

			if len(violations) != tt.wantCount {
				t.Errorf("got %d violations, want %d", len(violations), tt.wantCount)
				for _, v := range violations {
					t.Logf("  %s: %s (%s)", v.Rule, v.Message, v.Severity)
				}
			}

			for i, rule := range tt.wantRules {
				if i < len(violations) && violations[i].Rule != rule {
					t.Errorf("violation[%d].Rule = %q, want %q", i, violations[i].Rule, rule)
				}
			}

			if tt.wantErrors > 0 {
				errorCount := 0
				for _, v := range violations {
					if v.Severity == domain.LintError {
						errorCount++
					}
				}
				if errorCount != tt.wantErrors {
					t.Errorf("got %d errors, want %d", errorCount, tt.wantErrors)
				}
			}
		})
	}
}
