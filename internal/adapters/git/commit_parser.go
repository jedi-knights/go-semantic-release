package git

import (
	"regexp"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.CommitParser = (*ConventionalCommitParser)(nil)

var conventionalCommitRe = regexp.MustCompile(
	`^(?P<type>\w+)` +
		`(?:\((?P<scope>[^)]*)\))?` +
		`(?P<breaking>!)?` +
		`:\s*(?P<description>.+)$`,
)

// ConventionalCommitParser implements ports.CommitParser for Conventional Commits.
type ConventionalCommitParser struct{}

// NewConventionalCommitParser creates a new parser.
func NewConventionalCommitParser() *ConventionalCommitParser {
	return &ConventionalCommitParser{}
}

func (p *ConventionalCommitParser) Parse(message string) (domain.Commit, error) {
	lines := strings.SplitN(message, "\n", 2)
	subject := strings.TrimSpace(lines[0])

	matches := conventionalCommitRe.FindStringSubmatch(subject)
	if matches == nil {
		return domain.Commit{
			Message:     subject,
			Description: subject,
		}, nil
	}

	commit := domain.Commit{
		Message:     subject,
		Type:        matches[1],
		Scope:       matches[2],
		Description: matches[4],
	}

	// Check for ! marker.
	if matches[3] == "!" {
		commit.IsBreakingChange = true
	}

	// Parse body and footer.
	if len(lines) > 1 {
		body := strings.TrimSpace(lines[1])
		commit.Body, commit.Footer = splitBodyFooter(body)
		detectBreakingChange(&commit)
	}

	return commit, nil
}

func splitBodyFooter(text string) (body, footer string) {
	// Footer is separated from body by a blank line and starts with a token.
	parts := strings.Split(text, "\n\n")
	if len(parts) <= 1 {
		return text, ""
	}

	lastPart := parts[len(parts)-1]
	if isFooter(lastPart) {
		return strings.Join(parts[:len(parts)-1], "\n\n"), lastPart
	}
	return text, ""
}

var footerTokenRe = regexp.MustCompile(`^[\w-]+(?:: | #)`)

func isFooter(text string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return false
	}
	return footerTokenRe.MatchString(lines[0]) ||
		strings.HasPrefix(lines[0], "BREAKING CHANGE:") ||
		strings.HasPrefix(lines[0], "BREAKING-CHANGE:")
}

func detectBreakingChange(commit *domain.Commit) {
	for _, prefix := range []string{"BREAKING CHANGE:", "BREAKING-CHANGE:"} {
		if note := findBreakingNote(commit.Footer, prefix); note != "" {
			commit.IsBreakingChange = true
			commit.BreakingNote = note
			return
		}
		if note := findBreakingNote(commit.Body, prefix); note != "" {
			commit.IsBreakingChange = true
			commit.BreakingNote = note
			return
		}
	}
}

func findBreakingNote(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}
