package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

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

// PluginConfig holds configuration for the GitLab plugin.
type PluginConfig struct {
	ProjectID  string   `mapstructure:"project_id"`
	Token      string   `mapstructure:"token"`
	APIURL     string   `mapstructure:"api_url"`
	Assets     []string `mapstructure:"assets"`
	Milestones []string `mapstructure:"milestones"`
}

// Plugin implements lifecycle interfaces for GitLab integration.
type Plugin struct {
	config PluginConfig
	client *http.Client
	logger ports.Logger
}

// NewPlugin creates a GitLab lifecycle plugin.
func NewPlugin(cfg PluginConfig, logger ports.Logger) *Plugin {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://gitlab.com/api/v4"
	}
	if cfg.Token == "" {
		cfg.Token = resolveToken()
	}
	return &Plugin{
		config: cfg,
		client: &http.Client{},
		logger: logger,
	}
}

func resolveToken() string {
	for _, key := range []string{"GL_TOKEN", "GITLAB_TOKEN", "SEMANTIC_RELEASE_GITLAB_TOKEN"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

func (p *Plugin) Name() string { return "gitlab" }

// VerifyConditions checks that GitLab credentials and config are valid.
func (p *Plugin) VerifyConditions(ctx context.Context, rc *domain.ReleaseContext) error {
	if p.config.Token == "" {
		return fmt.Errorf("GitLab token not found (set GL_TOKEN, GITLAB_TOKEN, or SEMANTIC_RELEASE_GITLAB_TOKEN)")
	}
	if p.config.ProjectID == "" {
		return fmt.Errorf("GitLab project_id must be configured")
	}

	apiURL := fmt.Sprintf("%s/projects/%s", p.config.APIURL, url.PathEscape(p.config.ProjectID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("verifying GitLab access: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("GitLab token is invalid or lacks permissions (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitLab API returned HTTP %d for project verification", resp.StatusCode)
	}

	return nil
}

// Publish creates a GitLab release.
func (p *Plugin) Publish(ctx context.Context, rc *domain.ReleaseContext) (*domain.ProjectReleaseResult, error) {
	if rc.CurrentProject == nil {
		return nil, nil
	}

	tagName := rc.TagName
	releaseName := tagName
	if rc.CurrentProject.Project.Name != "" {
		releaseName = fmt.Sprintf("%s %s", rc.CurrentProject.Project.Name, rc.CurrentProject.NextVersion.String())
	}

	reqBody := glCreateReleaseRequest{
		TagName:     tagName,
		Name:        releaseName,
		Description: rc.Notes,
		Milestones:  p.config.Milestones,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling release request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/projects/%s/releases", p.config.APIURL, url.PathEscape(p.config.ProjectID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("publishing release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab create release failed (%d): %s", resp.StatusCode, string(body))
	}

	var release glRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding release response: %w", err)
	}

	p.logger.Info("created GitLab release", "tag", tagName, "url", release.Links.Self)

	return &domain.ProjectReleaseResult{
		Project:    rc.CurrentProject.Project,
		Version:    rc.CurrentProject.NextVersion,
		TagName:    tagName,
		Published:  true,
		PublishURL: release.Links.Self,
		Changelog:  rc.Notes,
	}, nil
}

// AddChannel is a no-op for GitLab (releases don't have a channel/prerelease distinction).
func (p *Plugin) AddChannel(_ context.Context, _ *domain.ReleaseContext) error {
	return nil
}

// Success logs the successful release.
func (p *Plugin) Success(_ context.Context, rc *domain.ReleaseContext) error {
	p.logger.Info("GitLab release successful", "tag", rc.TagName)
	return nil
}

// Fail logs the failed release.
func (p *Plugin) Fail(_ context.Context, rc *domain.ReleaseContext) error {
	if rc.Error != nil {
		p.logger.Error("GitLab release failed", "error", rc.Error)
	}
	return nil
}

// --- Helper types ---

type glCreateReleaseRequest struct {
	TagName     string   `json:"tag_name"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Milestones  []string `json:"milestones,omitempty"`
}

type glRelease struct {
	TagName     string `json:"tag_name"`
	Description string `json:"description"`
	Links       struct {
		Self string `json:"self"`
	} `json:"_links"`
}

func (p *Plugin) setHeaders(req *http.Request) {
	req.Header.Set("PRIVATE-TOKEN", p.config.Token)
	req.Header.Set("Content-Type", "application/json")
}
