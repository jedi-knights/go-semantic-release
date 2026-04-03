package plugins

import "github.com/jedi-knights/go-semantic-release/internal/domain"

// ExternalPluginRequest is sent to an external plugin executable via stdin.
type ExternalPluginRequest struct {
	Step    string                `json:"step"`
	Context ExternalPluginContext `json:"context"`
}

// ExternalPluginContext contains the release context data sent to external plugins.
type ExternalPluginContext struct {
	Branch        string                 `json:"branch"`
	DryRun        bool                   `json:"dry_run"`
	CI            bool                   `json:"ci"`
	RepositoryURL string                 `json:"repository_url"`
	TagName       string                 `json:"tag_name"`
	Notes         string                 `json:"notes"`
	Commits       []ExternalPluginCommit `json:"commits"`
	Project       *ExternalPluginProject `json:"project,omitempty"`
	NextVersion   string                 `json:"next_version,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

// ExternalPluginCommit is the commit representation sent to external plugins.
type ExternalPluginCommit struct {
	Hash        string `json:"hash"`
	Message     string `json:"message"`
	Type        string `json:"type"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	Breaking    bool   `json:"breaking"`
}

// ExternalPluginProject is the project representation sent to external plugins.
type ExternalPluginProject struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

// ExternalPluginResponse is received from an external plugin executable via stdout.
type ExternalPluginResponse struct {
	ReleaseType string `json:"release_type,omitempty"` // for analyzeCommits
	Notes       string `json:"notes,omitempty"`        // for generateNotes
	Error       string `json:"error,omitempty"`
}

// toExternalContext converts a ReleaseContext to the external protocol format.
func toExternalContext(rc *domain.ReleaseContext) ExternalPluginContext {
	ctx := ExternalPluginContext{
		Branch:        rc.Branch,
		DryRun:        rc.DryRun,
		CI:            rc.CI,
		RepositoryURL: rc.RepositoryURL,
		TagName:       rc.TagName,
		Notes:         rc.Notes,
	}

	if rc.Error != nil {
		ctx.Error = rc.Error.Error()
	}

	for i := range rc.Commits {
		ctx.Commits = append(ctx.Commits, ExternalPluginCommit{
			Hash:        rc.Commits[i].Hash,
			Message:     rc.Commits[i].Message,
			Type:        rc.Commits[i].Type,
			Scope:       rc.Commits[i].Scope,
			Description: rc.Commits[i].Description,
			Breaking:    rc.Commits[i].IsBreakingChange,
		})
	}

	if rc.CurrentProject != nil {
		ctx.Project = &ExternalPluginProject{
			Name:    rc.CurrentProject.Project.Name,
			Path:    rc.CurrentProject.Project.Path,
			Version: rc.CurrentProject.NextVersion.String(),
		}
		ctx.NextVersion = rc.CurrentProject.NextVersion.String()
	}

	return ctx
}
