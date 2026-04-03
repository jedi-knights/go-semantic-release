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
