package prompt_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/prompt"
)

func TestTerminalPrompter_Confirm(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"yes", "y\n", true},
		{"YES", "YES\n", true},
		{"no", "n\n", false},
		{"empty", "\n", false},
		{"full yes", "yes\n", true},
		{"anything else", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			w := &bytes.Buffer{}
			p := prompt.NewTerminalPrompterWithIO(r, w)

			got, err := p.Confirm("Proceed?")
			if err != nil {
				t.Fatalf("Confirm() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Confirm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoopPrompter_AlwaysConfirms(t *testing.T) {
	p := prompt.NewNoopPrompter()
	got, err := p.Confirm("anything")
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if !got {
		t.Error("NoopPrompter should always return true")
	}
}

func TestNewTerminalPrompter(t *testing.T) {
	p := prompt.NewTerminalPrompter()
	if p == nil {
		t.Fatal("NewTerminalPrompter() returned nil")
	}
}

func TestIsTerminal(t *testing.T) {
	// In a test environment stdin is not a terminal; IsTerminal must not panic.
	// The result (true or false) is environment-dependent; we only assert it runs.
	_ = prompt.IsTerminal()
}

func TestTerminalPrompter_Confirm_EOF(t *testing.T) {
	// An empty reader simulates EOF on the first Scan — Confirm should return (false, nil).
	r := strings.NewReader("")
	w := &bytes.Buffer{}
	p := prompt.NewTerminalPrompterWithIO(r, w)

	got, err := p.Confirm("Continue?")
	if err != nil {
		t.Fatalf("Confirm() on EOF returned error: %v", err)
	}
	if got {
		t.Error("Confirm() on EOF should return false")
	}
}
