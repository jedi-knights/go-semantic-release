package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// ExternalPlugin wraps an external executable as a lifecycle plugin.
// Communication uses JSON over stdin/stdout.
type ExternalPlugin struct {
	name       string
	executable string
}

// NewExternalPlugin creates an external plugin adapter.
func NewExternalPlugin(name, executable string) *ExternalPlugin {
	return &ExternalPlugin{name: name, executable: executable}
}

func (p *ExternalPlugin) Name() string { return p.name }

// VerifyConditions calls the external plugin's verifyConditions step.
func (p *ExternalPlugin) VerifyConditions(ctx context.Context, rc *domain.ReleaseContext) error {
	_, err := p.invoke(ctx, string(domain.StepVerifyConditions), rc)
	return err
}

// AnalyzeCommits calls the external plugin's analyzeCommits step.
func (p *ExternalPlugin) AnalyzeCommits(ctx context.Context, rc *domain.ReleaseContext) (domain.ReleaseType, error) {
	resp, err := p.invoke(ctx, string(domain.StepAnalyzeCommits), rc)
	if err != nil {
		return domain.ReleaseNone, err
	}

	switch strings.ToLower(resp.ReleaseType) {
	case "major":
		return domain.ReleaseMajor, nil
	case "minor":
		return domain.ReleaseMinor, nil
	case "patch":
		return domain.ReleasePatch, nil
	default:
		return domain.ReleaseNone, nil
	}
}

// VerifyRelease calls the external plugin's verifyRelease step.
func (p *ExternalPlugin) VerifyRelease(ctx context.Context, rc *domain.ReleaseContext) error {
	_, err := p.invoke(ctx, string(domain.StepVerifyRelease), rc)
	return err
}

// GenerateNotes calls the external plugin's generateNotes step.
func (p *ExternalPlugin) GenerateNotes(ctx context.Context, rc *domain.ReleaseContext) (string, error) {
	resp, err := p.invoke(ctx, string(domain.StepGenerateNotes), rc)
	if err != nil {
		return "", err
	}
	return resp.Notes, nil
}

// Prepare calls the external plugin's prepare step.
func (p *ExternalPlugin) Prepare(ctx context.Context, rc *domain.ReleaseContext) error {
	_, err := p.invoke(ctx, string(domain.StepPrepare), rc)
	return err
}

// Publish calls the external plugin's publish step.
func (p *ExternalPlugin) Publish(ctx context.Context, rc *domain.ReleaseContext) (*domain.ProjectReleaseResult, error) {
	_, err := p.invoke(ctx, string(domain.StepPublish), rc)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// AddChannel calls the external plugin's addChannel step.
func (p *ExternalPlugin) AddChannel(ctx context.Context, rc *domain.ReleaseContext) error {
	_, err := p.invoke(ctx, string(domain.StepAddChannel), rc)
	return err
}

// Success calls the external plugin's success step.
func (p *ExternalPlugin) Success(ctx context.Context, rc *domain.ReleaseContext) error {
	_, err := p.invoke(ctx, string(domain.StepSuccess), rc)
	return err
}

// Fail calls the external plugin's fail step.
func (p *ExternalPlugin) Fail(ctx context.Context, rc *domain.ReleaseContext) error {
	_, err := p.invoke(ctx, string(domain.StepFail), rc)
	return err
}

func (p *ExternalPlugin) invoke(ctx context.Context, step string, rc *domain.ReleaseContext) (*ExternalPluginResponse, error) {
	request := ExternalPluginRequest{
		Step:    step,
		Context: toExternalContext(rc),
	}

	inputData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshaling plugin request: %w", err)
	}

	cmd := exec.CommandContext(ctx, p.executable, "--step", step)
	cmd.Stdin = bytes.NewReader(inputData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("external plugin %q step %s failed: %s", p.name, step, errMsg)
	}

	// If no output, the plugin doesn't implement this step — that's OK.
	if stdout.Len() == 0 {
		return &ExternalPluginResponse{}, nil
	}

	var resp ExternalPluginResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parsing plugin %q response: %w", p.name, err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("external plugin %q: %s", p.name, resp.Error)
	}

	return &resp, nil
}
