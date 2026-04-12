package domain_test

import (
	"errors"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestReleaseResult_HasErrors_NoProjects(t *testing.T) {
	rr := domain.ReleaseResult{}
	if rr.HasErrors() {
		t.Error("HasErrors() on empty result should be false")
	}
}

func TestReleaseResult_HasErrors_AllClean(t *testing.T) {
	rr := domain.ReleaseResult{
		Projects: []domain.ProjectReleaseResult{
			{Project: domain.Project{Name: "a"}},
			{Project: domain.Project{Name: "b"}},
		},
	}
	if rr.HasErrors() {
		t.Error("HasErrors() should be false when no project has an error")
	}
}

func TestReleaseResult_HasErrors_OneError(t *testing.T) {
	rr := domain.ReleaseResult{
		Projects: []domain.ProjectReleaseResult{
			{Project: domain.Project{Name: "a"}},
			{Project: domain.Project{Name: "b"}},
		},
	}
	rr.Projects[1].SetError(errors.New("boom"))

	if !rr.HasErrors() {
		t.Error("HasErrors() should be true when at least one project has an error")
	}
}

func TestProjectReleaseResult_SetError_SetsMessage(t *testing.T) {
	var pr domain.ProjectReleaseResult
	err := errors.New("tag already exists")
	pr.SetError(err)

	if pr.Error == nil {
		t.Fatal("Error field should not be nil after SetError")
	}
	if pr.ErrorMessage != err.Error() {
		t.Errorf("ErrorMessage = %q, want %q", pr.ErrorMessage, err.Error())
	}
}

func TestProjectReleaseResult_SetError_ClearsOnNil(t *testing.T) {
	var pr domain.ProjectReleaseResult
	pr.SetError(errors.New("initial error"))
	pr.SetError(nil)

	if pr.Error != nil {
		t.Error("Error field should be nil after SetError(nil)")
	}
	if pr.ErrorMessage != "" {
		t.Errorf("ErrorMessage should be empty after SetError(nil), got %q", pr.ErrorMessage)
	}
}
