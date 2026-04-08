package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.Prompter = (*TerminalPrompter)(nil)

// TerminalPrompter prompts the user via stdin/stdout.
type TerminalPrompter struct {
	reader io.Reader
	writer io.Writer
}

// NewTerminalPrompter creates a prompter that reads from stdin and writes to stdout.
func NewTerminalPrompter() *TerminalPrompter {
	return &TerminalPrompter{reader: os.Stdin, writer: os.Stdout}
}

// NewTerminalPrompterWithIO creates a prompter with custom reader/writer (for testing).
func NewTerminalPrompterWithIO(r io.Reader, w io.Writer) *TerminalPrompter {
	return &TerminalPrompter{reader: r, writer: w}
}

// Confirm asks the user a yes/no question and returns their answer.
func (p *TerminalPrompter) Confirm(message string) (bool, error) {
	if _, err := fmt.Fprintf(p.writer, "%s [y/N] ", message); err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(p.reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, nil // EOF
	}

	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

// IsTerminal returns true if stdin appears to be a terminal.
func IsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
