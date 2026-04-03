package git

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// TemplateTagService formats and parses tags using Go templates.
type TemplateTagService struct {
	repoTemplate    string
	projectTemplate string
}

// NewTemplateTagService creates a tag service with configurable templates.
func NewTemplateTagService(repoTemplate, projectTemplate string) *TemplateTagService {
	if repoTemplate == "" {
		repoTemplate = "v{{.Version}}"
	}
	if projectTemplate == "" {
		projectTemplate = "{{.Project}}/v{{.Version}}"
	}
	return &TemplateTagService{
		repoTemplate:    repoTemplate,
		projectTemplate: projectTemplate,
	}
}

type tagData struct {
	Project string
	Version string
}

func (s *TemplateTagService) FormatTag(project string, version domain.Version) (string, error) {
	tmplStr := s.repoTemplate
	if project != "" {
		tmplStr = s.projectTemplate
	}

	tmpl, err := template.New("tag").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing tag template: %w", err)
	}

	var buf bytes.Buffer
	data := tagData{Project: project, Version: version.String()}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing tag template: %w", err)
	}
	return buf.String(), nil
}

func (s *TemplateTagService) ParseTag(tagName string) (string, domain.Version, error) {
	// Try project-scoped patterns first.
	// Pattern: project/vX.Y.Z
	if idx := strings.Index(tagName, "/v"); idx > 0 {
		project := tagName[:idx]
		ver, err := domain.ParseVersion(tagName[idx+1:])
		if err == nil {
			return project, ver, nil
		}
	}

	// Pattern: project@X.Y.Z
	if idx := strings.LastIndex(tagName, "@"); idx > 0 {
		project := tagName[:idx]
		ver, err := domain.ParseVersion(tagName[idx+1:])
		if err == nil {
			return project, ver, nil
		}
	}

	// Repo-level: vX.Y.Z or X.Y.Z
	ver, err := domain.ParseVersion(tagName)
	if err != nil {
		return "", domain.Version{}, fmt.Errorf("cannot parse tag %q: %w", tagName, err)
	}
	return "", ver, nil
}

func (s *TemplateTagService) FindLatestTag(tags []domain.Tag, project string) (*domain.Tag, error) {
	var latest *domain.Tag

	for i := range tags {
		proj, ver, err := s.ParseTag(tags[i].Name)
		if err != nil {
			continue
		}
		if proj != project {
			continue
		}

		tags[i].Version = ver
		tags[i].Project = proj

		if latest == nil || ver.GreaterThan(latest.Version) {
			t := tags[i]
			latest = &t
		}
	}
	return latest, nil
}
