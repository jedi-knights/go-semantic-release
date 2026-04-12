package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestDefaultChangelogSections_Count(t *testing.T) {
	sections := domain.DefaultChangelogSections()
	if len(sections) == 0 {
		t.Fatal("DefaultChangelogSections() returned empty slice")
	}
}

func TestDefaultChangelogSections_VisibleSections(t *testing.T) {
	sections := domain.DefaultChangelogSections()

	wantVisible := []string{"breaking", "feat", "fix", "perf", "revert"}
	visible := make([]string, 0)
	for _, s := range sections {
		if !s.Hidden {
			visible = append(visible, s.Type)
		}
	}

	if len(visible) != len(wantVisible) {
		t.Errorf("visible section count = %d, want %d; got %v", len(visible), len(wantVisible), visible)
		return
	}

	for i, want := range wantVisible {
		if visible[i] != want {
			t.Errorf("visible[%d] = %q, want %q", i, visible[i], want)
		}
	}
}

func TestDefaultChangelogSections_HiddenSections(t *testing.T) {
	sections := domain.DefaultChangelogSections()

	wantHidden := []string{"refactor", "docs", "style", "test", "build", "ci", "chore"}
	hidden := make([]string, 0)
	for _, s := range sections {
		if s.Hidden {
			hidden = append(hidden, s.Type)
		}
	}

	if len(hidden) != len(wantHidden) {
		t.Errorf("hidden section count = %d, want %d; got %v", len(hidden), len(wantHidden), hidden)
		return
	}

	for i, want := range wantHidden {
		if hidden[i] != want {
			t.Errorf("hidden[%d] = %q, want %q", i, hidden[i], want)
		}
	}
}

func TestDefaultChangelogSections_TitlesNonEmpty(t *testing.T) {
	for _, s := range domain.DefaultChangelogSections() {
		if s.Title == "" {
			t.Errorf("section type=%q has empty Title", s.Type)
		}
		if s.Type == "" {
			t.Error("section has empty Type")
		}
	}
}
