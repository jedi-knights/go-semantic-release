package github

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

// Compile-time interface compliance check.
var _ ports.ReleasePublisher = (*Publisher)(nil)

// Publisher implements ports.ReleasePublisher for GitHub Releases.
type Publisher struct {
	owner  string
	repo   string
	token  string
	client *http.Client
}

// NewPublisher creates a GitHub release publisher.
// If token is empty, it is resolved from GH_TOKEN, GITHUB_TOKEN, or
// SEMANTIC_RELEASE_GITHUB_TOKEN environment variables.
func NewPublisher(owner, repo, token string) *Publisher {
	if token == "" {
		for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN", "SEMANTIC_RELEASE_GITHUB_TOKEN"} {
			if v := os.Getenv(key); v != "" {
				token = v
				break
			}
		}
	}
	return &Publisher{
		owner:  owner,
		repo:   repo,
		token:  token,
		client: &http.Client{},
	}
}

type createReleaseRequest struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

type createReleaseResponse struct {
	HTMLURL string `json:"html_url"`
	ID      int    `json:"id"`
}

func (p *Publisher) Publish(ctx context.Context, params ports.PublishParams) (domain.ProjectReleaseResult, error) {
	name := params.TagName
	if params.Project != "" {
		name = fmt.Sprintf("%s %s", params.Project, params.Version.String())
	}

	reqBody := createReleaseRequest{
		TagName:    params.TagName,
		Name:       name,
		Body:       params.Changelog,
		Prerelease: params.Prerelease,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return domain.ProjectReleaseResult{}, fmt.Errorf("marshaling release request: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", p.owner, p.repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return domain.ProjectReleaseResult{}, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "token "+p.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.client.Do(req)
	if err != nil {
		return domain.ProjectReleaseResult{}, fmt.Errorf("publishing release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return domain.ProjectReleaseResult{}, fmt.Errorf("github release failed (%d): %s", resp.StatusCode, string(body))
	}

	var releaseResp createReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&releaseResp); err != nil {
		return domain.ProjectReleaseResult{}, fmt.Errorf("decoding release response: %w", err)
	}

	return domain.ProjectReleaseResult{
		TagName:    params.TagName,
		Published:  true,
		PublishURL: releaseResp.HTMLURL,
		Changelog:  params.Changelog,
	}, nil
}
