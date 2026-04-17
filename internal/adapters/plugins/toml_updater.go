package plugins

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// updateTOMLKey updates a single key under a TOML section in-place, preserving
// all formatting, comments, and surrounding content.
//
// keyPath is a dot-separated path such as "tool.poetry.version": the final
// segment is the key name; the preceding segments form the section header.
// A single-segment keyPath (no dots) targets a top-level key.
//
// Returns an error if the target section or key is not found.
func updateTOMLKey(content []byte, keyPath, newValue string) ([]byte, error) {
	lastDot := strings.LastIndex(keyPath, ".")
	if lastDot < 0 {
		return updateTOMLTopLevelKey(content, keyPath, newValue)
	}
	sectionPath := keyPath[:lastDot]
	keyName := keyPath[lastDot+1:]
	return updateTOMLSectionKey(content, sectionPath, keyName, newValue)
}

func updateTOMLSectionKey(content []byte, sectionPath, keyName, newValue string) ([]byte, error) {
	header := "[" + sectionPath + "]"

	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inSection := false
	replaced := false

	for scanner.Scan() {
		line := scanner.Text()

		if sec, ok := parseSectionHeader(line); ok {
			inSection = sec == header
		}

		if inSection && !replaced {
			if updated, ok := replaceKeyValue(line, keyName, newValue); ok {
				line = updated
				replaced = true
			}
		}

		buf.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning TOML content: %w", err)
	}
	if !replaced {
		return nil, fmt.Errorf("key %q not found under [%s] in TOML content", keyName, sectionPath)
	}
	return buf.Bytes(), nil
}

func updateTOMLTopLevelKey(content []byte, keyName, newValue string) ([]byte, error) {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inSection := false
	replaced := false

	for scanner.Scan() {
		line := scanner.Text()

		if _, ok := parseSectionHeader(line); ok {
			inSection = true
		}

		if !inSection && !replaced {
			if updated, ok := replaceKeyValue(line, keyName, newValue); ok {
				line = updated
				replaced = true
			}
		}

		buf.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning TOML content: %w", err)
	}
	if !replaced {
		return nil, fmt.Errorf("top-level key %q not found in TOML content", keyName)
	}
	return buf.Bytes(), nil
}

// parseSectionHeader returns the full section header (e.g. "[tool.poetry]") and
// true if line is a simple section header. Array-of-tables ([[...]]) are ignored.
func parseSectionHeader(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "[[") {
		return "", false
	}
	end := strings.Index(trimmed, "]")
	if end <= 0 {
		return "", false
	}
	return trimmed[:end+1], true
}

// replaceKeyValue checks whether line assigns the given key a quoted string
// value and replaces the value with newValue, preserving indentation and any
// trailing content (e.g. inline comments).
func replaceKeyValue(line, key, newValue string) (string, bool) {
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
