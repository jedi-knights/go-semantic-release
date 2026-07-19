// Package tomledit provides surgical, line-oriented edits to TOML content that
// preserve formatting, comments, and surrounding lines. It deliberately does not
// parse-and-reserialize: a full round-trip through a TOML encoder would reflow
// the file and drop comments. These helpers are shared by the version-file
// updater (Cargo.toml, pyproject.toml) and the Cargo.lock updater so both edit
// values identically.
package tomledit

import "strings"

// ReplaceKeyValue checks whether line assigns the given key a double-quoted
// string value and, if so, replaces the value with newValue while preserving
// leading indentation and any trailing content (e.g. an inline comment).
//
// The match is intentionally strict: the line must read exactly `key = "…"`
// (single spaces around `=`, double quotes). Non-canonical spacing or
// single-quoted strings are left untouched — cargo and most formatters emit the
// canonical form, so this avoids editing look-alike lines by accident.
func ReplaceKeyValue(line, key, newValue string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	prefix := key + ` = "`
	if !strings.HasPrefix(trimmed, prefix) {
		return line, false
	}
	rest := trimmed[len(prefix):]
	closeQuote := strings.Index(rest, `"`)
	if closeQuote < 0 {
		return line, false
	}
	trailer := rest[closeQuote+1:]
	return indent + prefix + newValue + `"` + trailer, true
}

// ReadKeyValue returns the double-quoted string value assigned to key on line,
// and true when line is a canonical `key = "…"` assignment. It is the read-only
// counterpart to ReplaceKeyValue and shares the same strict matching rules.
func ReadKeyValue(line, key string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")

	prefix := key + ` = "`
	if !strings.HasPrefix(trimmed, prefix) {
		return "", false
	}
	rest := trimmed[len(prefix):]
	closeQuote := strings.Index(rest, `"`)
	if closeQuote < 0 {
		return "", false
	}
	return rest[:closeQuote], true
}
