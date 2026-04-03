package git_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/git"
)

func TestConventionalCommitParser_Parse(t *testing.T) {
	parser := git.NewConventionalCommitParser()

	tests := []struct {
		name            string
		message         string
		wantType        string
		wantScope       string
		wantDescription string
		wantBreaking    bool
		wantBreakingNote string
	}{
		{
			name:            "simple feat",
			message:         "feat: add user login",
			wantType:        "feat",
			wantDescription: "add user login",
		},
		{
			name:            "feat with scope",
			message:         "feat(auth): add OAuth support",
			wantType:        "feat",
			wantScope:       "auth",
			wantDescription: "add OAuth support",
		},
		{
			name:            "fix",
			message:         "fix: correct null pointer",
			wantType:        "fix",
			wantDescription: "correct null pointer",
		},
		{
			name:            "breaking change with bang",
			message:         "feat!: redesign API",
			wantType:        "feat",
			wantDescription: "redesign API",
			wantBreaking:    true,
		},
		{
			name:            "breaking change with scope and bang",
			message:         "refactor(core)!: remove deprecated methods",
			wantType:        "refactor",
			wantScope:       "core",
			wantDescription: "remove deprecated methods",
			wantBreaking:    true,
		},
		{
			name: "breaking change in footer",
			message: `feat: update authentication

BREAKING CHANGE: the login endpoint now requires an API key`,
			wantType:         "feat",
			wantDescription:  "update authentication",
			wantBreaking:     true,
			wantBreakingNote: "the login endpoint now requires an API key",
		},
		{
			name: "breaking change with hyphen in footer",
			message: `fix: change return type

BREAKING-CHANGE: return type changed from string to int`,
			wantType:         "fix",
			wantDescription:  "change return type",
			wantBreaking:     true,
			wantBreakingNote: "return type changed from string to int",
		},
		{
			name:            "non-conventional commit",
			message:         "updated the readme file",
			wantType:        "",
			wantDescription: "updated the readme file",
			wantBreaking:    false,
		},
		{
			name:            "chore commit",
			message:         "chore: update dependencies",
			wantType:        "chore",
			wantDescription: "update dependencies",
		},
		{
			name: "commit with body and footer",
			message: `feat(api): add pagination support

This adds cursor-based pagination to all list endpoints.

Reviewed-by: Jane Doe`,
			wantType:        "feat",
			wantScope:       "api",
			wantDescription: "add pagination support",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commit, err := parser.Parse(tt.message)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if commit.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", commit.Type, tt.wantType)
			}
			if commit.Scope != tt.wantScope {
				t.Errorf("Scope = %q, want %q", commit.Scope, tt.wantScope)
			}
			if commit.Description != tt.wantDescription {
				t.Errorf("Description = %q, want %q", commit.Description, tt.wantDescription)
			}
			if commit.IsBreakingChange != tt.wantBreaking {
				t.Errorf("IsBreakingChange = %v, want %v", commit.IsBreakingChange, tt.wantBreaking)
			}
			if tt.wantBreakingNote != "" && commit.BreakingNote != tt.wantBreakingNote {
				t.Errorf("BreakingNote = %q, want %q", commit.BreakingNote, tt.wantBreakingNote)
			}
		})
	}
}
