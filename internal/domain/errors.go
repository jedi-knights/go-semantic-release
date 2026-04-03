package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for the domain layer.
var (
	ErrNoReleasableChanges = errors.New("no releasable changes found")
	ErrInvalidVersion      = errors.New("invalid version")
	ErrInvalidCommit       = errors.New("invalid commit message")
	ErrProjectNotFound     = errors.New("project not found")
	ErrTagAlreadyExists    = errors.New("tag already exists")
	ErrBranchNotAllowed    = errors.New("branch is not configured for releases")
	ErrDryRun              = errors.New("dry run: no mutations performed")
)

// ProjectError wraps an error with project context.
type ProjectError struct {
	Project string
	Op      string
	Err     error
}

func (e *ProjectError) Error() string {
	return fmt.Sprintf("project %q: %s: %v", e.Project, e.Op, e.Err)
}

func (e *ProjectError) Unwrap() error {
	return e.Err
}

// NewProjectError creates a project-scoped error.
func NewProjectError(project, op string, err error) error {
	return &ProjectError{Project: project, Op: op, Err: err}
}

// ReleaseError wraps an error that occurred during the release pipeline.
type ReleaseError struct {
	Step string
	Err  error
}

func (e *ReleaseError) Error() string {
	return fmt.Sprintf("release step %q: %v", e.Step, e.Err)
}

func (e *ReleaseError) Unwrap() error {
	return e.Err
}

// NewReleaseError creates a release pipeline error.
func NewReleaseError(step string, err error) error {
	return &ReleaseError{Step: step, Err: err}
}
