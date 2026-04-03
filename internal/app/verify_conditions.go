package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// ConditionVerifier checks that all prerequisites for a release are met.
type ConditionVerifier struct {
	git    ports.GitRepository
	config domain.Config
	logger ports.Logger
}

// NewConditionVerifier creates a condition verifier.
func NewConditionVerifier(git ports.GitRepository, config domain.Config, logger ports.Logger) *ConditionVerifier {
	return &ConditionVerifier{git: git, config: config, logger: logger}
}

// VerificationResult captures the outcome of condition checks.
type VerificationResult struct {
	Passed   bool
	Failures []string
}

// Verify checks all release conditions.
func (v *ConditionVerifier) Verify(ctx context.Context) (*VerificationResult, error) {
	result := &VerificationResult{Passed: true}

	branch, err := v.git.CurrentBranch(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting current branch: %w", err)
	}

	v.checkBranch(branch, result)
	v.checkGitHub(result)

	return result, nil
}

func (v *ConditionVerifier) checkBranch(branch string, result *VerificationResult) {
	policy := domain.FindBranchPolicy(v.config.Branches, branch)
	if policy == nil {
		result.Passed = false
		result.Failures = append(result.Failures,
			fmt.Sprintf("branch %q is not configured for releases", branch))
	}
}

func (v *ConditionVerifier) checkGitHub(result *VerificationResult) {
	if !v.config.GitHub.CreateRelease {
		return
	}

	var missing []string
	if v.config.GitHub.Owner == "" {
		missing = append(missing, "github.owner")
	}
	if v.config.GitHub.Repo == "" {
		missing = append(missing, "github.repo")
	}
	if v.config.GitHub.Token == "" {
		missing = append(missing, "github.token (or SEMANTIC_RELEASE_GITHUB_TOKEN)")
	}

	if len(missing) > 0 {
		result.Passed = false
		result.Failures = append(result.Failures,
			fmt.Sprintf("missing GitHub config: %s", strings.Join(missing, ", ")))
	}
}
