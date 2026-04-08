package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance checks.
var (
	_ ports.Plugin                 = (*Plugin)(nil)
	_ ports.VerifyConditionsPlugin = (*Plugin)(nil)
	_ ports.PublishPlugin          = (*Plugin)(nil)
	_ ports.AddChannelPlugin       = (*Plugin)(nil)
	_ ports.SuccessPlugin          = (*Plugin)(nil)
	_ ports.FailPlugin             = (*Plugin)(nil)
)

// PluginConfig holds configuration for the GitHub plugin.
type PluginConfig struct {
	Owner                  string   `mapstructure:"owner"`
	Repo                   string   `mapstructure:"repo"`
	Token                  string   `mapstructure:"token"`
	APIURL                 string   `mapstructure:"api_url"`
	Assets                 []string `mapstructure:"assets"`
	DraftRelease           bool     `mapstructure:"draft_release"`
	DiscussionCategoryName string   `mapstructure:"discussion_category_name"`
	SuccessComment         string   `mapstructure:"success_comment"`
	FailComment            string   `mapstructure:"fail_comment"`
	ReleasedLabels         []string `mapstructure:"released_labels"`
	FailLabels             []string `mapstructure:"fail_labels"`
}

// Plugin implements multiple lifecycle interfaces for GitHub integration.
type Plugin struct {
	config PluginConfig
	client *http.Client
	logger ports.Logger
}

// NewPlugin creates a GitHub lifecycle plugin.
func NewPlugin(cfg PluginConfig, logger ports.Logger) *Plugin {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.github.com"
	}
	if cfg.Token == "" {
		cfg.Token = resolveToken()
	}
	if cfg.SuccessComment == "" {
		cfg.SuccessComment = "🎉 This issue has been resolved in version {{.Version}} 🎉\n\nThe release is available on [GitHub release]({{.ReleaseURL}})"
	}
	if cfg.FailComment == "" {
		cfg.FailComment = "The release from branch `{{.Branch}}` has failed.\n\nError: {{.Error}}"
	}
	if len(cfg.ReleasedLabels) == 0 {
		cfg.ReleasedLabels = []string{"released"}
	}
	if len(cfg.FailLabels) == 0 {
		cfg.FailLabels = []string{"semantic-release"}
	}
	return &Plugin{
		config: cfg,
		client: &http.Client{},
		logger: logger,
	}
}

func resolveToken() string {
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN", "SEMANTIC_RELEASE_GITHUB_TOKEN"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

func (p *Plugin) Name() string { return "github" }

// VerifyConditions checks that GitHub credentials and config are valid.
func (p *Plugin) VerifyConditions(ctx context.Context, rc *domain.ReleaseContext) error {
	if p.config.Token == "" {
		return fmt.Errorf("GitHub token not found (set GH_TOKEN, GITHUB_TOKEN, or SEMANTIC_RELEASE_GITHUB_TOKEN)")
	}

	owner, repo := p.config.Owner, p.config.Repo
	if owner == "" || repo == "" {
		return fmt.Errorf("GitHub owner and repo must be configured")
	}

	// Verify token is valid with a lightweight API call.
	url := fmt.Sprintf("%s/repos/%s/%s", p.config.APIURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("verifying GitHub access: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("GitHub token is invalid or lacks permissions (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned HTTP %d for repo verification", resp.StatusCode)
	}

	return nil
}

// Publish creates a GitHub release, optionally uploads assets.
func (p *Plugin) Publish(ctx context.Context, rc *domain.ReleaseContext) (*domain.ProjectReleaseResult, error) {
	if rc.CurrentProject == nil {
		return nil, nil
	}

	tagName := rc.TagName
	releaseName := tagName
	if rc.CurrentProject.Project.Name != "" {
		releaseName = fmt.Sprintf("%s %s", rc.CurrentProject.Project.Name, rc.CurrentProject.NextVersion.String())
	}

	isPrerelease := rc.BranchPolicy != nil && rc.BranchPolicy.Prerelease

	reqBody := ghCreateReleaseRequest{
		TagName:                tagName,
		Name:                   releaseName,
		Body:                   rc.Notes,
		Prerelease:             isPrerelease,
		Draft:                  p.config.DraftRelease,
		DiscussionCategoryName: p.config.DiscussionCategoryName,
	}

	releaseResp, err := p.createGHRelease(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	// Upload assets.
	for _, pattern := range p.config.Assets {
		if err := p.uploadAssetGlob(ctx, releaseResp.ID, pattern); err != nil {
			p.logger.Warn("failed to upload asset", "pattern", pattern, "error", err)
		}
	}

	return &domain.ProjectReleaseResult{
		Project:    rc.CurrentProject.Project,
		Version:    rc.CurrentProject.NextVersion,
		TagName:    tagName,
		Published:  true,
		PublishURL: releaseResp.HTMLURL,
		Changelog:  rc.Notes,
	}, nil
}

// AddChannel updates a release's prerelease status based on the channel.
func (p *Plugin) AddChannel(ctx context.Context, rc *domain.ReleaseContext) error {
	if rc.TagName == "" {
		return nil
	}

	// Find existing release by tag.
	release, err := p.getReleaseByTag(ctx, rc.TagName)
	if err != nil {
		return fmt.Errorf("finding release for tag %s: %w", rc.TagName, err)
	}
	if release == nil {
		return nil
	}

	isPrerelease := rc.BranchPolicy != nil && rc.BranchPolicy.Prerelease

	// Update the prerelease field.
	updateBody := map[string]any{
		"prerelease": isPrerelease,
	}
	jsonData, _ := json.Marshal(updateBody)

	url := fmt.Sprintf("%s/repos/%s/%s/releases/%d", p.config.APIURL, p.config.Owner, p.config.Repo, release.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("updating release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("updating release failed (%d): %s", resp.StatusCode, string(body))
	}

	p.logger.Info("updated release channel", "tag", rc.TagName, "prerelease", isPrerelease)
	return nil
}

// Success comments on merged PRs and resolved issues.
func (p *Plugin) Success(ctx context.Context, rc *domain.ReleaseContext) error {
	if rc.CurrentProject == nil || rc.Result == nil {
		return nil
	}

	// Find the publish URL for this project.
	releaseURL := ""
	for i := range rc.Result.Projects {
		if rc.Result.Projects[i].Project.Name == rc.CurrentProject.Project.Name {
			releaseURL = rc.Result.Projects[i].PublishURL
			break
		}
	}

	comment := strings.NewReplacer(
		"{{.Version}}", rc.CurrentProject.NextVersion.String(),
		"{{.ReleaseURL}}", releaseURL,
		"{{.Branch}}", rc.Branch,
		"{{.TagName}}", rc.TagName,
	).Replace(p.config.SuccessComment)

	// Comment on commits' associated PRs.
	for i := range rc.CurrentProject.Commits {
		prs, err := p.getPRsForCommit(ctx, rc.CurrentProject.Commits[i].Hash)
		if err != nil {
			p.logger.Debug("failed to get PRs for commit", "hash", rc.CurrentProject.Commits[i].Hash, "error", err)
			continue
		}
		for _, pr := range prs {
			if err := p.commentOnIssue(ctx, pr.Number, comment); err != nil {
				p.logger.Debug("failed to comment on PR", "number", pr.Number, "error", err)
			}
			p.addLabelsToIssue(ctx, pr.Number, p.config.ReleasedLabels)
		}
	}

	return nil
}

// Fail opens or updates a GitHub issue documenting the failure.
func (p *Plugin) Fail(ctx context.Context, rc *domain.ReleaseContext) error {
	if rc.Error == nil {
		return nil
	}

	body := strings.NewReplacer(
		"{{.Branch}}", rc.Branch,
		"{{.Error}}", rc.Error.Error(),
	).Replace(p.config.FailComment)

	title := "The automated release is failing"

	// Check for existing failure issue.
	existing, err := p.findFailureIssue(ctx, title)
	if err != nil {
		p.logger.Debug("failed to search for existing failure issue", "error", err)
	}

	if existing != nil {
		return p.commentOnIssue(ctx, existing.Number, body)
	}

	return p.createIssue(ctx, title, body, p.config.FailLabels)
}

// --- Helper methods ---

type ghCreateReleaseRequest struct {
	TagName                string `json:"tag_name"`
	Name                   string `json:"name"`
	Body                   string `json:"body"`
	Prerelease             bool   `json:"prerelease"`
	Draft                  bool   `json:"draft"`
	DiscussionCategoryName string `json:"discussion_category_name,omitempty"`
}

type ghRelease struct {
	ID        int    `json:"id"`
	HTMLURL   string `json:"html_url"`
	TagName   string `json:"tag_name"`
	UploadURL string `json:"upload_url"`
}

type ghPR struct {
	Number int `json:"number"`
}

type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

func (p *Plugin) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "token "+p.config.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
}

func (p *Plugin) createGHRelease(ctx context.Context, reqBody ghCreateReleaseRequest) (*ghRelease, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling release request: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases", p.config.APIURL, p.config.Owner, p.config.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("publishing release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github create release failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding release response: %w", err)
	}
	return &release, nil
}

func (p *Plugin) getReleaseByTag(ctx context.Context, tag string) (*ghRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", p.config.APIURL, p.config.Owner, p.config.Repo, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func (p *Plugin) uploadAssetGlob(ctx context.Context, releaseID int, pattern string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("globbing %s: %w", pattern, err)
	}

	for _, path := range matches {
		if err := p.uploadAsset(ctx, releaseID, path); err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugin) uploadAsset(ctx context.Context, releaseID int, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", filePath, err)
	}

	name := filepath.Base(filePath)
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	url := fmt.Sprintf("https://uploads.github.com/repos/%s/%s/releases/%d/assets?name=%s",
		p.config.Owner, p.config.Repo, releaseID, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, file)
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}

	req.Header.Set("Authorization", "token "+p.config.Token)
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = stat.Size()

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("uploading asset %s: %w", name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload asset failed (%d): %s", resp.StatusCode, string(body))
	}

	p.logger.Info("uploaded asset", "file", name, "release", releaseID)
	return nil
}

func (p *Plugin) getPRsForCommit(ctx context.Context, sha string) ([]ghPR, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/commits/%s/pulls", p.config.APIURL, p.config.Owner, p.config.Repo, sha)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var prs []ghPR
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, err
	}
	return prs, nil
}

func (p *Plugin) commentOnIssue(ctx context.Context, number int, body string) error {
	payload := map[string]string{"body": body}
	jsonData, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", p.config.APIURL, p.config.Owner, p.config.Repo, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("comment failed (%d)", resp.StatusCode)
	}
	return nil
}

func (p *Plugin) addLabelsToIssue(ctx context.Context, number int, labels []string) {
	if len(labels) == 0 {
		return
	}
	payload := map[string][]string{"labels": labels}
	jsonData, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", p.config.APIURL, p.config.Owner, p.config.Repo, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func (p *Plugin) findFailureIssue(ctx context.Context, title string) (*ghIssue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues?state=open&labels=%s&creator=app",
		p.config.APIURL, p.config.Owner, p.config.Repo, strings.Join(p.config.FailLabels, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var issues []ghIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}

	for _, issue := range issues {
		if issue.Title == title {
			return &issue, nil
		}
	}
	return nil, nil
}

func (p *Plugin) createIssue(ctx context.Context, title, body string, labels []string) error {
	payload := map[string]any{
		"title":  title,
		"body":   body,
		"labels": labels,
	}
	jsonData, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/repos/%s/%s/issues", p.config.APIURL, p.config.Owner, p.config.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create issue failed (%d): %s", resp.StatusCode, string(respBody))
	}

	p.logger.Info("created failure issue", "title", title)
	return nil
}
