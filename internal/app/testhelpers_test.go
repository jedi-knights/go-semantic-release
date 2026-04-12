package app_test

import "github.com/jedi-knights/go-semantic-release/internal/ports"

// noopLogger satisfies ports.Logger and discards all output.
// Used in tests that verify behaviour other than log output.
type noopLogger struct{}

var _ ports.Logger = noopLogger{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}
