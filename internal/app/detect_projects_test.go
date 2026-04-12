package app_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestProjectDetector_Detect_ReturnsDiscoveredProjects(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDiscoverer := mocks.NewMockProjectDiscoverer(ctrl)

	projects := []domain.Project{
		{Name: "service-a", Path: "./service-a", Type: domain.ProjectTypeGoModule},
		{Name: "service-b", Path: "./service-b", Type: domain.ProjectTypeGoModule},
	}
	mockDiscoverer.EXPECT().Discover(gomock.Any(), ".").Return(projects, nil)

	detector := app.NewProjectDetector(mockDiscoverer, noopLogger{})
	got, err := detector.Detect(context.Background(), ".")
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(got) != 2 {
		t.Errorf("Detect() returned %d projects, want 2", len(got))
	}
}

func TestProjectDetector_Detect_FallsBackToRootProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDiscoverer := mocks.NewMockProjectDiscoverer(ctrl)
	mockDiscoverer.EXPECT().Discover(gomock.Any(), ".").Return(nil, nil)

	detector := app.NewProjectDetector(mockDiscoverer, noopLogger{})
	got, err := detector.Detect(context.Background(), ".")
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Detect() returned %d projects, want 1 (root fallback)", len(got))
	}
	if got[0].Type != domain.ProjectTypeRoot {
		t.Errorf("fallback project type = %v, want ProjectTypeRoot", got[0].Type)
	}
	if got[0].Path != "." {
		t.Errorf("fallback project path = %q, want %q", got[0].Path, ".")
	}
}

func TestProjectDetector_Detect_DiscovererError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDiscoverer := mocks.NewMockProjectDiscoverer(ctrl)
	mockDiscoverer.EXPECT().Discover(gomock.Any(), ".").Return(nil, errors.New("scan failed"))

	detector := app.NewProjectDetector(mockDiscoverer, noopLogger{})
	_, err := detector.Detect(context.Background(), ".")
	if err == nil {
		t.Error("Detect() should return error when discoverer fails")
	}
}
