package ports

// Prompter provides interactive user prompts.
type Prompter interface {
	// Confirm asks the user a yes/no question and returns their answer.
	Confirm(message string) (bool, error)
}
