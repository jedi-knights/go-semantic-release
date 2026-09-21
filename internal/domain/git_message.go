package domain

import (
	"bytes"
	"fmt"
	"text/template"
)

// RenderGitCommitMessage renders the release commit message template with
// release data. Supports {{.Version}}, {{.Tag}}, and {{.Notes}} placeholders.
// Falls back to "chore(release): {tagName}" on empty template or render error.
func RenderGitCommitMessage(tmpl, tagName string, version Version, notes string) string {
	if tmpl == "" {
		return fmt.Sprintf("chore(release): %s", tagName)
	}
	data := struct {
		Version string
		Tag     string
		Notes   string
	}{
		Version: version.String(),
		Tag:     tagName,
		Notes:   notes,
	}
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return fmt.Sprintf("chore(release): %s", tagName)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Sprintf("chore(release): %s", tagName)
	}
	return buf.String()
}
