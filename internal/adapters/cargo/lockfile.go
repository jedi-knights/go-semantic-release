package cargo

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/tomledit"
)

// UpdateLockVersions rewrites the `version` of each named local crate inside a
// Cargo.lock file, leaving third-party dependencies untouched. crateVersions
// maps a crate name (as it appears in a `[[package]]` block's `name` field) to
// its new version string.
//
// Cargo.lock records each crate under an array-of-tables `[[package]]` block
// keyed by a sibling `name = "…"` field, so a block is selected by name rather
// than by a section header. The rewrite is surgical (line-by-line via
// tomledit.ReplaceKeyValue) to preserve formatting and comments. The top-level
// `version = N` lockfile-format line is never inside a `[[package]]` block and
// is an unquoted integer, so it is never matched.
//
// A crate listed in crateVersions but absent from the lock is silently skipped
// (best-effort): a missing lock entry should not abort a release. The function
// returns an error only on a scanner failure.
func UpdateLockVersions(content []byte, crateVersions map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var block []string
	flush := func() {
		processPackageBlock(block, crateVersions)
		for _, l := range block {
			buf.WriteString(l + "\n")
		}
		block = block[:0]
	}

	for scanner.Scan() {
		line := scanner.Text()
		// A table header (`[…]` or `[[…]]`) starts a new block; flush the
		// preceding one (the file preamble before the first header is its own
		// block and passes through untouched because its first line is not
		// "[[package]]").
		if isTableHeader(line) {
			flush()
		}
		block = append(block, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning Cargo.lock: %w", err)
	}
	flush()

	return buf.Bytes(), nil
}

// processPackageBlock rewrites the version line of a single `[[package]]` block
// in place when its crate name is in crateVersions. Blocks that are not
// `[[package]]` tables, or whose name is not a local crate, are left unchanged.
func processPackageBlock(block []string, crateVersions map[string]string) {
	if len(block) == 0 || strings.TrimSpace(block[0]) != "[[package]]" {
		return
	}
	// The name may appear before or after the version line, so resolve the
	// crate name for the whole block before rewriting.
	var newVersion string
	found := false
	for _, l := range block {
		if name, ok := tomledit.ReadKeyValue(l, "name"); ok {
			newVersion, found = crateVersions[name]
			break
		}
	}
	if !found {
		return
	}
	for i, l := range block {
		if updated, ok := tomledit.ReplaceKeyValue(l, "version", newVersion); ok {
			block[i] = updated
			break
		}
	}
}

// isTableHeader reports whether a trimmed line opens a TOML table or
// array-of-tables (`[section]` or `[[array]]`).
func isTableHeader(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "[")
}
