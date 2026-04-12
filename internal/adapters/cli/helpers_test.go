package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// ---------------------------------------------------------------------------
// parseBranchFlags
// ---------------------------------------------------------------------------

func TestParseBranchFlags_Simple(t *testing.T) {
	got := parseBranchFlags([]string{"main"})
	if len(got) != 1 {
		t.Fatalf("got %d policies, want 1", len(got))
	}
	if got[0].Name != "main" || !got[0].IsDefault {
		t.Errorf("got %+v, want {Name:main IsDefault:true}", got[0])
	}
}

func TestParseBranchFlags_Master(t *testing.T) {
	got := parseBranchFlags([]string{"master"})
	if len(got) != 1 {
		t.Fatalf("got %d policies, want 1", len(got))
	}
	if got[0].Name != "master" || !got[0].IsDefault {
		t.Errorf("got %+v, want {Name:master IsDefault:true}", got[0])
	}
}

func TestParseBranchFlags_NonDefault(t *testing.T) {
	got := parseBranchFlags([]string{"feature"})
	if len(got) != 1 {
		t.Fatalf("got %d policies, want 1", len(got))
	}
	if got[0].Name != "feature" || got[0].IsDefault {
		t.Errorf("got %+v, want {Name:feature IsDefault:false}", got[0])
	}
}

func TestParseBranchFlags_CommaSeparated(t *testing.T) {
	got := parseBranchFlags([]string{"main,feature"})
	if len(got) != 2 {
		t.Fatalf("got %d policies, want 2", len(got))
	}
	if got[0].Name != "main" || !got[0].IsDefault {
		t.Errorf("first policy = %+v, want {Name:main IsDefault:true}", got[0])
	}
	if got[1].Name != "feature" || got[1].IsDefault {
		t.Errorf("second policy = %+v, want {Name:feature IsDefault:false}", got[1])
	}
}

func TestParseBranchFlags_EmptyStrings(t *testing.T) {
	got := parseBranchFlags([]string{"", " "})
	if len(got) != 0 {
		t.Errorf("got %d policies, want 0", len(got))
	}
}

func TestParseBranchFlags_Multiple(t *testing.T) {
	got := parseBranchFlags([]string{"main", "develop"})
	if len(got) != 2 {
		t.Fatalf("got %d policies, want 2", len(got))
	}
	if got[0].Name != "main" || !got[0].IsDefault {
		t.Errorf("first policy = %+v, want {Name:main IsDefault:true}", got[0])
	}
	if got[1].Name != "develop" || got[1].IsDefault {
		t.Errorf("second policy = %+v, want {Name:develop IsDefault:false}", got[1])
	}
}

// ---------------------------------------------------------------------------
// applyFlagAndEnvOverrides
// ---------------------------------------------------------------------------

func TestApplyFlagAndEnvOverrides_DryRun(t *testing.T) {
	cfg := domain.Config{CI: true}
	opts := &rootOptions{dryRun: true, ciFlag: true}
	applyFlagAndEnvOverrides(&cfg, opts)
	if !cfg.DryRun {
		t.Error("expected DryRun = true")
	}
}

func TestApplyFlagAndEnvOverrides_Debug(t *testing.T) {
	cfg := domain.Config{CI: true}
	opts := &rootOptions{debug: true, ciFlag: true}
	applyFlagAndEnvOverrides(&cfg, opts)
	if !cfg.Debug {
		t.Error("expected Debug = true")
	}
}

func TestApplyFlagAndEnvOverrides_CIFlag(t *testing.T) {
	cfg := domain.Config{}
	opts := &rootOptions{ciFlag: true}
	applyFlagAndEnvOverrides(&cfg, opts)
	if !cfg.CI {
		t.Error("expected CI = true")
	}
	// isCI=true and dryRun=false: the auto dry-run guard should NOT fire.
	if cfg.DryRun {
		t.Error("expected DryRun = false (CI mode suppresses auto dry-run)")
	}
}

func TestApplyFlagAndEnvOverrides_NoCIFlag(t *testing.T) {
	cfg := domain.Config{}
	opts := &rootOptions{noCIFlag: true}
	applyFlagAndEnvOverrides(&cfg, opts)
	if cfg.CI {
		t.Error("expected CI = false")
	}
	// Not CI, not explicitly dry-run → auto dry-run for local runs.
	if !cfg.DryRun {
		t.Error("expected DryRun = true (auto dry-run for local run)")
	}
}

func TestApplyFlagAndEnvOverrides_BranchesOverride(t *testing.T) {
	cfg := domain.Config{CI: true}
	opts := &rootOptions{branches: []string{"develop"}, ciFlag: true}
	applyFlagAndEnvOverrides(&cfg, opts)
	if len(cfg.Branches) != 1 || cfg.Branches[0].Name != "develop" {
		t.Errorf("Branches = %v, want [{develop false}]", cfg.Branches)
	}
}

func TestApplyFlagAndEnvOverrides_RepositoryURL(t *testing.T) {
	cfg := domain.Config{CI: true}
	opts := &rootOptions{repositoryURL: "https://github.com/example/repo", ciFlag: true}
	applyFlagAndEnvOverrides(&cfg, opts)
	if cfg.RepositoryURL != "https://github.com/example/repo" {
		t.Errorf("RepositoryURL = %q, want https://github.com/example/repo", cfg.RepositoryURL)
	}
}

func TestApplyFlagAndEnvOverrides_TagFormat(t *testing.T) {
	cfg := domain.Config{CI: true}
	opts := &rootOptions{tagFormat: "v{{.Version}}", ciFlag: true}
	applyFlagAndEnvOverrides(&cfg, opts)
	if cfg.TagFormat != "v{{.Version}}" {
		t.Errorf("TagFormat = %q, want v{{.Version}}", cfg.TagFormat)
	}
}

func TestApplyFlagAndEnvOverrides_Plugins(t *testing.T) {
	cfg := domain.Config{CI: true}
	opts := &rootOptions{plugins: []string{"@semantic-release/git"}, ciFlag: true}
	applyFlagAndEnvOverrides(&cfg, opts)
	if len(cfg.Plugins) != 1 || cfg.Plugins[0] != "@semantic-release/git" {
		t.Errorf("Plugins = %v, want [@semantic-release/git]", cfg.Plugins)
	}
}

func TestApplyFlagAndEnvOverrides_Extends(t *testing.T) {
	cfg := domain.Config{CI: true}
	opts := &rootOptions{extends: []string{"./shared.yaml"}, ciFlag: true}
	applyFlagAndEnvOverrides(&cfg, opts)
	if len(cfg.Extends) != 1 || cfg.Extends[0] != "./shared.yaml" {
		t.Errorf("Extends = %v, want [./shared.yaml]", cfg.Extends)
	}
}

// ---------------------------------------------------------------------------
// modeString
// ---------------------------------------------------------------------------

func TestModeString_Stable(t *testing.T) {
	plan := &domain.ReleasePlan{}
	if got := modeString(plan); got != "stable" {
		t.Errorf("modeString = %q, want stable", got)
	}
}

func TestModeString_Prerelease(t *testing.T) {
	plan := &domain.ReleasePlan{
		Policy: &domain.BranchPolicy{Prerelease: true, Channel: "beta"},
	}
	if got := modeString(plan); got != "prerelease (beta)" {
		t.Errorf("modeString = %q, want prerelease (beta)", got)
	}
}

func TestModeString_NonPrereleasePolicy(t *testing.T) {
	plan := &domain.ReleasePlan{
		Policy: &domain.BranchPolicy{Prerelease: false, Channel: "release"},
	}
	if got := modeString(plan); got != "stable" {
		t.Errorf("modeString = %q, want stable", got)
	}
}

// ---------------------------------------------------------------------------
// displayProjectName
// ---------------------------------------------------------------------------

func TestDisplayProjectName_Empty(t *testing.T) {
	p := domain.Project{Name: ""}
	if got := displayProjectName(p); got != "(repository)" {
		t.Errorf("displayProjectName = %q, want (repository)", got)
	}
}

func TestDisplayProjectName_Root(t *testing.T) {
	p := domain.Project{Name: "root"}
	if got := displayProjectName(p); got != "(repository)" {
		t.Errorf("displayProjectName = %q, want (repository)", got)
	}
}

func TestDisplayProjectName_Named(t *testing.T) {
	p := domain.Project{Name: "my-service"}
	if got := displayProjectName(p); got != "my-service" {
		t.Errorf("displayProjectName = %q, want my-service", got)
	}
}

// ---------------------------------------------------------------------------
// projectName
// ---------------------------------------------------------------------------

func TestProjectName_Empty(t *testing.T) {
	pr := domain.ProjectReleaseResult{Project: domain.Project{Name: ""}}
	if got := projectName(pr); got != "repo" {
		t.Errorf("projectName = %q, want repo", got)
	}
}

func TestProjectName_Named(t *testing.T) {
	pr := domain.ProjectReleaseResult{Project: domain.Project{Name: "api"}}
	if got := projectName(pr); got != "api" {
		t.Errorf("projectName = %q, want api", got)
	}
}

// ---------------------------------------------------------------------------
// filterProject
// ---------------------------------------------------------------------------

func TestFilterProject_Found(t *testing.T) {
	projects := []domain.Project{
		{Name: "alpha"},
		{Name: "beta"},
	}
	got := filterProject(projects, "beta")
	if len(got) != 1 || got[0].Name != "beta" {
		t.Errorf("filterProject = %v, want [{beta}]", got)
	}
}

func TestFilterProject_NotFound(t *testing.T) {
	projects := []domain.Project{{Name: "alpha"}}
	got := filterProject(projects, "gamma")
	if got != nil {
		t.Errorf("filterProject = %v, want nil", got)
	}
}

func TestFilterProject_Empty(t *testing.T) {
	got := filterProject(nil, "anything")
	if got != nil {
		t.Errorf("filterProject(nil) = %v, want nil", got)
	}
}

// ---------------------------------------------------------------------------
// shouldPrompt
// ---------------------------------------------------------------------------

func TestShouldPrompt_NoInteractive(t *testing.T) {
	cfg := domain.Config{}
	opts := &rootOptions{noInteractive: true}
	if shouldPrompt(cfg, opts) {
		t.Error("shouldPrompt = true, want false when noInteractive=true")
	}
}

func TestShouldPrompt_Interactive(t *testing.T) {
	cfg := domain.Config{}
	opts := &rootOptions{interactive: true}
	if !shouldPrompt(cfg, opts) {
		t.Error("shouldPrompt = false, want true when interactive=true")
	}
}

func boolPtr(b bool) *bool { return &b }

func TestShouldPrompt_CfgInteractiveTrue(t *testing.T) {
	cfg := domain.Config{Interactive: boolPtr(true)}
	opts := &rootOptions{}
	if !shouldPrompt(cfg, opts) {
		t.Error("shouldPrompt = false, want true when cfg.Interactive=true")
	}
}

func TestShouldPrompt_CfgInteractiveFalse(t *testing.T) {
	cfg := domain.Config{Interactive: boolPtr(false)}
	opts := &rootOptions{}
	if shouldPrompt(cfg, opts) {
		t.Error("shouldPrompt = true, want false when cfg.Interactive=false")
	}
}

func TestShouldPrompt_CIMode(t *testing.T) {
	// In CI mode, cfg.CI=true; IsTerminal() is false in test environments.
	// Result: !cfg.CI && IsTerminal() = false.
	cfg := domain.Config{CI: true}
	opts := &rootOptions{}
	if shouldPrompt(cfg, opts) {
		t.Error("shouldPrompt = true, want false in CI mode with no terminal")
	}
}

// ---------------------------------------------------------------------------
// printPlan
// ---------------------------------------------------------------------------

func TestPrintPlan_JSON(t *testing.T) {
	plan := &domain.ReleasePlan{
		Branch: "main",
		Projects: []domain.ProjectReleasePlan{
			{
				Project:        domain.Project{Name: "svc"},
				ShouldRelease:  true,
				NextVersion:    domain.NewVersion(1, 2, 0),
				CurrentVersion: domain.NewVersion(1, 1, 0),
				ReleaseType:    domain.ReleaseMinor,
				Reason:         "feat commit",
			},
		},
	}

	var buf bytes.Buffer
	if err := printPlan(&buf, plan, true); err != nil {
		t.Fatalf("printPlan JSON error: %v", err)
	}

	var decoded domain.ReleasePlan
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
}

func TestPrintPlan_Text_NoReleasable(t *testing.T) {
	plan := &domain.ReleasePlan{
		Branch: "main",
		Projects: []domain.ProjectReleasePlan{
			{Project: domain.Project{Name: "svc"}, ShouldRelease: false, Reason: "no changes"},
		},
	}

	var buf bytes.Buffer
	if err := printPlan(&buf, plan, false); err != nil {
		t.Fatalf("printPlan text error: %v", err)
	}
	if !strings.Contains(buf.String(), "No releasable changes found.") {
		t.Errorf("output = %q, want to contain 'No releasable changes found.'", buf.String())
	}
}

func TestPrintPlan_Text_WithProjects(t *testing.T) {
	plan := &domain.ReleasePlan{
		Branch: "main",
		Projects: []domain.ProjectReleasePlan{
			{
				Project:        domain.Project{Name: "my-service"},
				ShouldRelease:  true,
				CurrentVersion: domain.NewVersion(1, 0, 0),
				NextVersion:    domain.NewVersion(1, 1, 0),
				ReleaseType:    domain.ReleaseMinor,
				Commits:        []domain.Commit{{Message: "feat: add something"}},
				Reason:         "new feature",
			},
		},
	}

	var buf bytes.Buffer
	if err := printPlan(&buf, plan, false); err != nil {
		t.Fatalf("printPlan text error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "my-service") {
		t.Errorf("output missing project name; got: %s", out)
	}
	if !strings.Contains(out, "1.0.0") {
		t.Errorf("output missing current version; got: %s", out)
	}
	if !strings.Contains(out, "1.1.0") {
		t.Errorf("output missing next version; got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// printReleaseResult
// ---------------------------------------------------------------------------

func TestPrintReleaseResult_JSON(t *testing.T) {
	result := &domain.ReleaseResult{
		Projects: []domain.ProjectReleaseResult{
			{
				Project: domain.Project{Name: "api"},
				Version: domain.NewVersion(2, 0, 0),
				TagName: "v2.0.0",
			},
		},
	}

	var buf bytes.Buffer
	if err := printReleaseResult(&buf, io.Discard, result, true); err != nil {
		t.Fatalf("printReleaseResult JSON error: %v", err)
	}

	var decoded domain.ReleaseResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
}

func TestPrintReleaseResult_Skipped(t *testing.T) {
	result := &domain.ReleaseResult{
		Projects: []domain.ProjectReleaseResult{
			{
				Project:    domain.Project{Name: "svc"},
				Skipped:    true,
				SkipReason: "dry-run",
				Version:    domain.NewVersion(1, 0, 0),
				TagName:    "v1.0.0",
			},
		},
	}

	var buf bytes.Buffer
	if err := printReleaseResult(&buf, io.Discard, result, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "dry-run") {
		t.Errorf("output = %q, want to contain skip reason 'dry-run'", buf.String())
	}
}

func TestPrintReleaseResult_Error(t *testing.T) {
	result := &domain.ReleaseResult{
		Projects: []domain.ProjectReleaseResult{
			func() domain.ProjectReleaseResult {
				pr := domain.ProjectReleaseResult{Project: domain.Project{Name: "svc"}}
				pr.SetError(errors.New("tag already exists"))
				return pr
			}(),
		},
	}

	var errBuf bytes.Buffer
	_ = printReleaseResult(io.Discard, &errBuf, result, false)
	if !strings.Contains(errBuf.String(), "ERROR") {
		t.Errorf("errW = %q, want to contain 'ERROR'", errBuf.String())
	}
}

func TestPrintReleaseResult_Success(t *testing.T) {
	result := &domain.ReleaseResult{
		Projects: []domain.ProjectReleaseResult{
			{
				Project: domain.Project{Name: "api"},
				Version: domain.NewVersion(1, 2, 3),
				TagName: "v1.2.3",
			},
		},
	}

	var buf bytes.Buffer
	if err := printReleaseResult(&buf, io.Discard, result, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Released") {
		t.Errorf("output = %q, want to contain 'Released'", buf.String())
	}
}

func TestPrintReleaseResult_HasErrors(t *testing.T) {
	result := &domain.ReleaseResult{
		Projects: []domain.ProjectReleaseResult{
			func() domain.ProjectReleaseResult {
				pr := domain.ProjectReleaseResult{Project: domain.Project{Name: "svc"}}
				pr.SetError(errors.New("push failed"))
				return pr
			}(),
		},
	}

	err := printReleaseResult(io.Discard, io.Discard, result, false)
	if !errors.Is(err, ErrQuietExit) {
		t.Errorf("error = %v, want ErrQuietExit", err)
	}
}

func TestPrintReleaseResult_PublishURL(t *testing.T) {
	result := &domain.ReleaseResult{
		Projects: []domain.ProjectReleaseResult{
			{
				Project:    domain.Project{Name: "api"},
				Version:    domain.NewVersion(1, 0, 0),
				TagName:    "v1.0.0",
				PublishURL: "https://github.com/org/repo/releases/tag/v1.0.0",
			},
		},
	}

	var buf bytes.Buffer
	if err := printReleaseResult(&buf, io.Discard, result, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "https://github.com/org/repo/releases/tag/v1.0.0") {
		t.Errorf("output = %q, want to contain publish URL", buf.String())
	}
}
