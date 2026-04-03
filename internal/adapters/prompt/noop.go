package prompt

// NoopPrompter always confirms without prompting (for CI or non-interactive mode).
type NoopPrompter struct{}

// NewNoopPrompter creates a prompter that always returns true.
func NewNoopPrompter() *NoopPrompter {
	return &NoopPrompter{}
}

// Confirm always returns true without prompting.
func (p *NoopPrompter) Confirm(_ string) (bool, error) {
	return true, nil
}
