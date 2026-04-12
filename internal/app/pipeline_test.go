package app_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

// compositePlugin implements multiple lifecycle interfaces for testing.
type compositePlugin struct {
	name           string
	verifyErr      error
	analyzeResult  domain.ReleaseType
	analyzeErr     error
	generateResult string
	generateErr    error
	prepareErr     error
	publishResult  *domain.ProjectReleaseResult
	publishErr     error
	successCalled  bool
	failCalled     bool
	failErr        error
}

func (p *compositePlugin) Name() string { return p.name }

func (p *compositePlugin) VerifyConditions(_ context.Context, _ *domain.ReleaseContext) error {
	return p.verifyErr
}

func (p *compositePlugin) AnalyzeCommits(_ context.Context, _ *domain.ReleaseContext) (domain.ReleaseType, error) {
	return p.analyzeResult, p.analyzeErr
}

func (p *compositePlugin) GenerateNotes(_ context.Context, _ *domain.ReleaseContext) (string, error) {
	return p.generateResult, p.generateErr
}

func (p *compositePlugin) Prepare(_ context.Context, _ *domain.ReleaseContext) error {
	return p.prepareErr
}

func (p *compositePlugin) Publish(_ context.Context, _ *domain.ReleaseContext) (*domain.ProjectReleaseResult, error) {
	return p.publishResult, p.publishErr
}

func (p *compositePlugin) Success(_ context.Context, _ *domain.ReleaseContext) error {
	p.successCalled = true
	return nil
}

func (p *compositePlugin) Fail(_ context.Context, _ *domain.ReleaseContext) error {
	p.failCalled = true
	return p.failErr
}

func TestPipeline_Execute_NoReleasableChanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()

	plugin := &compositePlugin{
		name:          "test",
		analyzeResult: domain.ReleaseNone,
	}

	pipeline := app.NewPipeline([]ports.Plugin{plugin}, mockLogger)
	rc := &domain.ReleaseContext{}

	err := pipeline.Execute(context.Background(), rc)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestPipeline_Execute_DryRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()

	plugin := &compositePlugin{
		name:           "test",
		analyzeResult:  domain.ReleaseMinor,
		generateResult: "## 1.1.0\n\n### Features\n- something new",
	}

	pipeline := app.NewPipeline([]ports.Plugin{plugin}, mockLogger)
	rc := &domain.ReleaseContext{DryRun: true}

	err := pipeline.Execute(context.Background(), rc)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if rc.Notes == "" {
		t.Error("expected notes to be generated even in dry run")
	}
}

func TestPipeline_Execute_VerifyFailureCallsFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	verifyErr := errors.New("bad credentials")
	plugin := &compositePlugin{
		name:      "test",
		verifyErr: verifyErr,
	}

	pipeline := app.NewPipeline([]ports.Plugin{plugin}, mockLogger)
	rc := &domain.ReleaseContext{}

	err := pipeline.Execute(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error")
	}
	if !plugin.failCalled {
		t.Error("Fail() should have been called on verification failure")
	}
}

func TestPipeline_Execute_MultipleAnalyzersHighestWins(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()

	patchPlugin := &compositePlugin{name: "patch", analyzeResult: domain.ReleasePatch}
	majorPlugin := &compositePlugin{name: "major", analyzeResult: domain.ReleaseMajor}

	pipeline := app.NewPipeline([]ports.Plugin{patchPlugin, majorPlugin}, mockLogger)
	rc := &domain.ReleaseContext{DryRun: true}

	err := pipeline.Execute(context.Background(), rc)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	// Both should pass through verifyRelease (no errors), generateNotes should run.
}

func TestPipeline_Execute_FullRelease(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()

	publishResult := &domain.ProjectReleaseResult{
		Published:  true,
		PublishURL: "https://github.com/org/repo/releases/v1.1.0",
	}

	plugin := &compositePlugin{
		name:           "test",
		analyzeResult:  domain.ReleaseMinor,
		generateResult: "## 1.1.0\n- feat",
		publishResult:  publishResult,
	}

	pipeline := app.NewPipeline([]ports.Plugin{plugin}, mockLogger)
	rc := &domain.ReleaseContext{
		Result: &domain.ReleaseResult{},
	}

	err := pipeline.Execute(context.Background(), rc)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !plugin.successCalled {
		t.Error("Success() should have been called")
	}
	if len(rc.Result.Projects) != 1 {
		t.Errorf("expected 1 project result, got %d", len(rc.Result.Projects))
	}
}

// verifyReleasePlugin extends compositePlugin with VerifyRelease support.
type verifyReleasePlugin struct {
	compositePlugin
	verifyReleaseErr error
}

func (p *verifyReleasePlugin) VerifyRelease(_ context.Context, _ *domain.ReleaseContext) error {
	return p.verifyReleaseErr
}

func TestPipeline_Execute_VerifyReleaseFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	plugin := &verifyReleasePlugin{
		compositePlugin: compositePlugin{
			name:          "test",
			analyzeResult: domain.ReleaseMinor,
		},
		verifyReleaseErr: errors.New("release blocked by policy"),
	}

	pipeline := app.NewPipeline([]ports.Plugin{plugin}, mockLogger)
	rc := &domain.ReleaseContext{}

	err := pipeline.Execute(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error from verifyRelease failure, got nil")
	}
	if !plugin.failCalled {
		t.Error("Fail() should have been called when verifyRelease fails")
	}
}

func TestPipeline_Execute_GenerateNotesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	plugin := &compositePlugin{
		name:          "test",
		analyzeResult: domain.ReleaseMinor,
		generateErr:   errors.New("notes generation failed"),
	}

	pipeline := app.NewPipeline([]ports.Plugin{plugin}, mockLogger)
	rc := &domain.ReleaseContext{}

	err := pipeline.Execute(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error from generateNotes failure, got nil")
	}
	if !plugin.failCalled {
		t.Error("Fail() should have been called when generateNotes fails")
	}
}

func TestPipeline_Execute_PrepareError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	plugin := &compositePlugin{
		name:           "test",
		analyzeResult:  domain.ReleaseMinor,
		generateResult: "## 1.1.0",
		prepareErr:     errors.New("prepare step failed"),
	}

	pipeline := app.NewPipeline([]ports.Plugin{plugin}, mockLogger)
	rc := &domain.ReleaseContext{}

	err := pipeline.Execute(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error from prepare failure, got nil")
	}
	if !plugin.failCalled {
		t.Error("Fail() should have been called when prepare fails")
	}
}

// addChannelPlugin exercises the addChannel non-fatal path.
// It implements AnalyzeCommits, GenerateNotes, Prepare, Publish, AddChannel, Success, and Fail.
type addChannelPlugin struct {
	name           string
	analyzeResult  domain.ReleaseType
	generateResult string
	addChannelErr  error
	successCalled  bool
	failCalled     bool
}

func (p *addChannelPlugin) Name() string { return p.name }

func (p *addChannelPlugin) AnalyzeCommits(_ context.Context, _ *domain.ReleaseContext) (domain.ReleaseType, error) {
	return p.analyzeResult, nil
}

func (p *addChannelPlugin) GenerateNotes(_ context.Context, _ *domain.ReleaseContext) (string, error) {
	return p.generateResult, nil
}

func (p *addChannelPlugin) Prepare(_ context.Context, _ *domain.ReleaseContext) error {
	return nil
}

func (p *addChannelPlugin) Publish(_ context.Context, _ *domain.ReleaseContext) (*domain.ProjectReleaseResult, error) {
	return nil, nil
}

func (p *addChannelPlugin) AddChannel(_ context.Context, _ *domain.ReleaseContext) error {
	return p.addChannelErr
}

func (p *addChannelPlugin) Success(_ context.Context, _ *domain.ReleaseContext) error {
	p.successCalled = true
	return nil
}

func (p *addChannelPlugin) Fail(_ context.Context, _ *domain.ReleaseContext) error {
	p.failCalled = true
	return nil
}

// TestPipeline_Execute_AddChannelNonFatal verifies that an addChannel failure is non-fatal:
// the pipeline logs a warning and continues to invoke the success step.
func TestPipeline_Execute_AddChannelNonFatal(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	plugin := &addChannelPlugin{
		name:           "test",
		analyzeResult:  domain.ReleaseMinor,
		generateResult: "## 1.1.0\n- feat",
		addChannelErr:  errors.New("channel add failed"),
	}

	pipeline := app.NewPipeline([]ports.Plugin{plugin}, mockLogger)
	rc := &domain.ReleaseContext{
		Result: &domain.ReleaseResult{},
	}

	err := pipeline.Execute(context.Background(), rc)
	if err != nil {
		t.Fatalf("Execute() should not return error when addChannel fails (non-fatal), got: %v", err)
	}
	if !plugin.successCalled {
		t.Error("Success() should have been called even when addChannel fails")
	}
}

// successErrPlugin implements Success with an error to exercise the non-fatal warning path.
type successErrPlugin struct {
	compositePlugin
	successErr error
}

func (p *successErrPlugin) Success(_ context.Context, _ *domain.ReleaseContext) error {
	return p.successErr
}

// TestPipeline_Execute_SuccessNotificationFailure verifies that a Success() error is non-fatal.
func TestPipeline_Execute_SuccessNotificationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	plugin := &successErrPlugin{
		compositePlugin: compositePlugin{
			name:           "test",
			analyzeResult:  domain.ReleaseMinor,
			generateResult: "## 1.1.0",
		},
		successErr: errors.New("notification delivery failed"),
	}

	pipeline := app.NewPipeline([]ports.Plugin{plugin}, mockLogger)
	rc := &domain.ReleaseContext{}

	err := pipeline.Execute(context.Background(), rc)
	if err != nil {
		t.Fatalf("Execute() should not return error when success notification fails (non-fatal), got: %v", err)
	}
}
