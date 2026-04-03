package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// PluginConfig holds configuration for the Bitbucket plugin.
type PluginConfig struct {
	Workspace string `mapstructure:"workspace"`
	RepoSlug  string `mapstructure:"repo_slug"`
	Token     string `mapstructure:"token"`
	APIURL    string `mapstructure:"api_url"`
}

// Plugin implements lifecycle interfaces for Bitbucket Cloud integration.
type Plugin struct {
	config PluginConfig
	client *http.Client
	logger ports.Logger
}

// NewPlugin creates a Bitbucket lifecycle plugin.
func NewPlugin(cfg PluginConfig, logger ports.Logger) *Plugin {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.bitbucket.org/2.0"
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
	for _, key := range []string{"BB_TOKEN", "BITBUCKET_TOKEN", "SEMANTIC_RELEASE_BITBUCKET_TOKEN"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

func (p *Plugin) Name() string { return "bitbucket" }

// VerifyConditions checks that Bitbucket credentials and config are valid.
func (p *Plugin) VerifyConditions(ctx context.Context, rc *domain.ReleaseContext) error {
	if p.config.Token == "" {
		return fmt.Errorf("Bitbucket token not found (set BB_TOKEN, BITBUCKET_TOKEN, or SEMANTIC_RELEASE_BITBUCKET_TOKEN)")
	}
	if p.config.Workspace == "" || p.config.RepoSlug == "" {
		return fmt.Errorf("Bitbucket workspace and repo_slug must be configured")
	}

	apiURL := fmt.Sprintf("%s/repositories/%s/%s", p.config.APIURL, p.config.Workspace, p.config.RepoSlug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("verifying Bitbucket access: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("Bitbucket token is invalid or lacks permissions (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bitbucket API returned HTTP %d for repo verification", resp.StatusCode)
	}

	return nil
}

// Publish creates a Bitbucket pipeline tag. Bitbucket Cloud doesn't have a native
// "Releases" concept like GitHub/GitLab, so we create an annotated tag via the API.
func (p *Plugin) Publish(ctx context.Context, rc *domain.ReleaseContext) (*domain.ProjectReleaseResult, error) {
	if rc.CurrentProject == nil {
		return nil, nil
	}

	tagName := rc.TagName

	// Create annotated tag via Bitbucket REST API.
	reqBody := bbCreateTagRequest{
		Name:    tagName,
		Target:  bbTarget{Hash: ""},
		Message: rc.Notes,
	}

	// Get head hash if available.
	if rc.CurrentProject.Project.Path != "" {
		reqBody.Target.Hash = "HEAD"
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling tag request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repositories/%s/%s/refs/tags", p.config.APIURL, p.config.Workspace, p.config.RepoSlug)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("publishing tag: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bitbucket create tag failed (%d): %s", resp.StatusCode, string(body))
	}

	repoURL := fmt.Sprintf("https://bitbucket.org/%s/%s/src/%s", p.config.Workspace, p.config.RepoSlug, tagName)
	p.logger.Info("created Bitbucket tag", "tag", tagName)

	return &domain.ProjectReleaseResult{
		Project:    rc.CurrentProject.Project,
		Version:    rc.CurrentProject.NextVersion,
		TagName:    tagName,
		Published:  true,
		PublishURL: repoURL,
		Changelog:  rc.Notes,
	}, nil
}

// AddChannel is a no-op for Bitbucket.
func (p *Plugin) AddChannel(_ context.Context, _ *domain.ReleaseContext) error {
	return nil
}

// Success logs the successful release.
func (p *Plugin) Success(_ context.Context, rc *domain.ReleaseContext) error {
	p.logger.Info("Bitbucket release successful", "tag", rc.TagName)
	return nil
}

// Fail logs the failed release.
func (p *Plugin) Fail(_ context.Context, rc *domain.ReleaseContext) error {
	if rc.Error != nil {
		p.logger.Error("Bitbucket release failed", "error", rc.Error)
	}
	return nil
}

// --- Helper types ---

type bbCreateTagRequest struct {
	Name    string   `json:"name"`
	Target  bbTarget `json:"target"`
	Message string   `json:"message,omitempty"`
}

type bbTarget struct {
	Hash string `json:"hash"`
}

func (p *Plugin) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.config.Token)
	req.Header.Set("Content-Type", "application/json")
}
