package git_test

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	adaptergit "github.com/jedi-knights/go-semantic-release/internal/adapters/git"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestWorkspaceDiscoverer_Discover(t *testing.T) {
	tests := []struct {
		name          string
		goWorkExists  bool
		goWorkContent string
		goModContents map[string]string
		wantCount     int
		wantNames     []string
	}{
		{
			name:         "no go.work file",
			goWorkExists: false,
			wantCount:    0,
		},
		{
			name:         "go.work with block use",
			goWorkExists: true,
			goWorkContent: `go 1.21

use (
	./svc-api
	./svc-worker
	./pkg/shared
)
`,
			goModContents: map[string]string{
				"/repo/svc-api/go.mod":    "module github.com/org/repo/svc-api",
				"/repo/svc-worker/go.mod": "module github.com/org/repo/svc-worker",
				"/repo/pkg/shared/go.mod": "module github.com/org/repo/pkg/shared",
			},
			wantCount: 3,
			wantNames: []string{"./svc-api", "./svc-worker", "./pkg/shared"},
		},
		{
			name:         "go.work with single use",
			goWorkExists: true,
			goWorkContent: `go 1.21

use ./mylib
`,
			goModContents: map[string]string{
				"/repo/mylib/go.mod": "module github.com/org/repo/mylib",
			},
			wantCount: 1,
			wantNames: []string{"./mylib"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockFS := mocks.NewMockFileSystem(ctrl)

			mockFS.EXPECT().Exists("/repo/go.work").Return(tt.goWorkExists)

			if tt.goWorkExists {
				mockFS.EXPECT().ReadFile("/repo/go.work").Return([]byte(tt.goWorkContent), nil)

				for path, content := range tt.goModContents {
					mockFS.EXPECT().ReadFile(path).Return([]byte(content), nil).AnyTimes()
				}
			}

			discoverer := adaptergit.NewWorkspaceDiscoverer(mockFS)
			projects, err := discoverer.Discover(context.Background(), "/repo")
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}

			if len(projects) != tt.wantCount {
				t.Errorf("got %d projects, want %d", len(projects), tt.wantCount)
			}

			for i, name := range tt.wantNames {
				if i < len(projects) && projects[i].Name != name {
					t.Errorf("project[%d].Name = %q, want %q", i, projects[i].Name, name)
				}
				if i < len(projects) && projects[i].Type != domain.ProjectTypeGoWorkspace {
					t.Errorf("project[%d].Type = %v, want %v", i, projects[i].Type, domain.ProjectTypeGoWorkspace)
				}
			}
		})
	}
}

func TestModuleDiscoverer_Discover(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Glob("/repo/**/go.mod").Return([]string{
		"/repo/go.mod",
		"/repo/services/api/go.mod",
		"/repo/services/worker/go.mod",
	}, nil)

	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/repo"), nil)
	mockFS.EXPECT().ReadFile("/repo/services/api/go.mod").Return(
		[]byte("module github.com/org/repo/services/api"), nil)
	mockFS.EXPECT().ReadFile("/repo/services/worker/go.mod").Return(
		[]byte("module github.com/org/repo/services/worker"), nil)

	discoverer := adaptergit.NewModuleDiscoverer(mockFS)
	projects, err := discoverer.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 3 {
		t.Fatalf("got %d projects, want 3", len(projects))
	}

	// Root project.
	if projects[0].Type != domain.ProjectTypeRoot {
		t.Errorf("root project type = %v, want %v", projects[0].Type, domain.ProjectTypeRoot)
	}

	// Nested modules.
	if projects[1].Type != domain.ProjectTypeGoModule {
		t.Errorf("nested project type = %v, want %v", projects[1].Type, domain.ProjectTypeGoModule)
	}
}

func TestConfiguredDiscoverer_Discover(t *testing.T) {
	configs := []domain.ProjectConfig{
		{Name: "api", Path: "services/api", TagPrefix: "api/"},
		{Name: "worker", Path: "services/worker"},
	}

	discoverer := adaptergit.NewConfiguredDiscoverer(configs)
	projects, err := discoverer.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(projects))
	}

	if projects[0].TagPrefix != "api/" {
		t.Errorf("project[0].TagPrefix = %q, want %q", projects[0].TagPrefix, "api/")
	}
	if projects[1].TagPrefix != "worker/" {
		t.Errorf("project[1].TagPrefix = %q, want %q", projects[1].TagPrefix, "worker/")
	}
}

func TestCompositeDiscoverer_FirstNonEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)

	empty := mocks.NewMockProjectDiscoverer(ctrl)
	empty.EXPECT().Discover(gomock.Any(), "/repo").Return(nil, nil)

	withProjects := mocks.NewMockProjectDiscoverer(ctrl)
	withProjects.EXPECT().Discover(gomock.Any(), "/repo").Return([]domain.Project{
		{Name: "found"},
	}, nil)

	// Third should not be called.
	neverCalled := mocks.NewMockProjectDiscoverer(ctrl)

	_ = neverCalled // suppress unused warning

	composite := adaptergit.NewCompositeDiscoverer(empty, withProjects)
	projects, err := composite.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 1 || projects[0].Name != "found" {
		t.Errorf("expected 'found' project, got %v", projects)
	}
}
