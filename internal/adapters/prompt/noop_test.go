package prompt_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/prompt"
)

func TestNoopPrompter_Confirm_AlwaysTrue(t *testing.T) {
	p := prompt.NewNoopPrompter()

	for _, question := range []string{"Are you sure?", "", "Delete everything?"} {
		got, err := p.Confirm(question)
		if err != nil {
			t.Errorf("Confirm(%q) unexpected error: %v", question, err)
		}
		if !got {
			t.Errorf("Confirm(%q) = false, want true", question)
		}
	}
}
