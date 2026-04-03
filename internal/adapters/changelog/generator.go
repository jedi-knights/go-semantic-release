package changelog

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

const defaultTemplate = `## {{if .Project}}[{{.Project}}] {{end}}{{.Version}} ({{.Date}})
{{range .Sections}}
### {{.Title}}

{{range .Commits}}- {{if .Scope}}**{{.Scope}}:** {{end}}{{.Description}} ({{.ShortHash}})
{{end}}{{end}}`

// TemplateGenerator implements ports.ChangelogGenerator using Go templates.
type TemplateGenerator struct {
	customTemplate string
}

// NewTemplateGenerator creates a changelog generator with an optional custom template.
func NewTemplateGenerator(customTemplate string) *TemplateGenerator {
	return &TemplateGenerator{customTemplate: customTemplate}
}

type templateData struct {
	Version  string
	Project  string
	Date     string
	Sections []sectionData
}

type sectionData struct {
	Title   string
	Commits []commitData
}

type commitData struct {
	Hash        string
	ShortHash   string
	Type        string
	Scope       string
	Description string
	Author      string
	Breaking    bool
}

func (g *TemplateGenerator) Generate(
	version domain.Version,
	project string,
	commits []domain.Commit,
	sections []domain.ChangelogSectionConfig,
) (string, error) {
	data := g.buildTemplateData(version, project, commits, sections)

	tmplStr := defaultTemplate
	if g.customTemplate != "" {
		tmplStr = g.customTemplate
	}

	tmpl, err := template.New("changelog").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing changelog template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing changelog template: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func (g *TemplateGenerator) buildTemplateData(
	version domain.Version,
	project string,
	commits []domain.Commit,
	sections []domain.ChangelogSectionConfig,
) templateData {
	// Group breaking changes separately.
	breakingCommits := filterBreakingCommits(commits)
	commitsByType := groupCommitsByType(commits)

	var secs []sectionData
	for _, sec := range sections {
		if sec.Hidden {
			continue
		}

		var sectionCommits []domain.Commit
		if sec.Type == "breaking" {
			sectionCommits = breakingCommits
		} else {
			sectionCommits = commitsByType[sec.Type]
		}

		if len(sectionCommits) == 0 {
			continue
		}

		secs = append(secs, sectionData{
			Title:   sec.Title,
			Commits: toCommitData(sectionCommits),
		})
	}

	return templateData{
		Version:  version.String(),
		Project:  project,
		Date:     time.Now().Format("2006-01-02"),
		Sections: secs,
	}
}

func filterBreakingCommits(commits []domain.Commit) []domain.Commit {
	var result []domain.Commit
	for _, c := range commits {
		if c.IsBreakingChange {
			result = append(result, c)
		}
	}
	return result
}

func groupCommitsByType(commits []domain.Commit) map[string][]domain.Commit {
	groups := make(map[string][]domain.Commit)
	for _, c := range commits {
		if c.Type != "" {
			groups[c.Type] = append(groups[c.Type], c)
		}
	}
	return groups
}

func toCommitData(commits []domain.Commit) []commitData {
	result := make([]commitData, 0, len(commits))
	for _, c := range commits {
		short := c.Hash
		if len(short) > 7 {
			short = short[:7]
		}
		result = append(result, commitData{
			Hash:        c.Hash,
			ShortHash:   short,
			Type:        c.Type,
			Scope:       c.Scope,
			Description: c.Description,
			Author:      c.Author,
			Breaking:    c.IsBreakingChange,
		})
	}
	return result
}
