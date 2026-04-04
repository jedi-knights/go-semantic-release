package git_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"go.uber.org/mock/gomock"

	adaptergit "github.com/jedi-knights/go-semantic-release/internal/adapters/git"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

// testDirEntry is a minimal fs.DirEntry implementation for unit tests.
type testDirEntry struct {
	name  string
	isDir bool
}

func (d testDirEntry) Name() string      { return d.name }
func (d testDirEntry) IsDir() bool       { return d.isDir }
func (d testDirEntry) Type() fs.FileMode { return 0 }

// Info returns an error rather than nil so that if a future discoverer calls
// Info() on a test entry, the test fails with a clear message instead of
// silently producing a nil fs.FileInfo that panics at the call site.
func (d testDirEntry) Info() (fs.FileInfo, error) {
	return nil, errors.New("testDirEntry.Info: not implemented")
}

func TestWorkspaceDiscoverer_Discover(t *testing.T) {
	t.Parallel()
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
			// go.work use entries like "./svc-api" are normalised to "svc-api"
			// so project names and tag prefixes do not contain a "./" segment.
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
			wantNames: []string{"svc-api", "svc-worker", "pkg/shared"},
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
			wantNames: []string{"mylib"},
		},
		{
			// use( with multiple spaces between "use" and "(" must still be recognised.
			name:         "go.work use block with extra spaces before paren",
			goWorkExists: true,
			goWorkContent: `go 1.21

use  (
	./svc-api
)
`,
			goModContents: map[string]string{
				"/repo/svc-api/go.mod": "module github.com/org/repo/svc-api",
			},
			wantCount: 1,
			wantNames: []string{"svc-api"},
		},
		{
			// Inline // comments on use-block entries must be stripped.
			name:         "go.work use block with inline comments",
			goWorkExists: true,
			goWorkContent: `go 1.21

use (
	./svc-api    // primary service
	./pkg/shared // shared library
)
`,
			goModContents: map[string]string{
				"/repo/svc-api/go.mod":    "module github.com/org/repo/svc-api",
				"/repo/pkg/shared/go.mod": "module github.com/org/repo/pkg/shared",
			},
			wantCount: 2,
			wantNames: []string{"svc-api", "pkg/shared"},
		},
		{
			// The go toolchain accepts a tab between "use" and the path.
			name:         "go.work single use with tab separator",
			goWorkExists: true,
			goWorkContent: "go 1.21\n\nuse\t./svc-api\n",
			goModContents: map[string]string{
				"/repo/svc-api/go.mod": "module github.com/org/repo/svc-api",
			},
			wantCount: 1,
			wantNames: []string{"svc-api"},
		},
		{
			// Comment-only lines inside a use block must be ignored (not become project paths).
			name:         "go.work use block with comment-only lines",
			goWorkExists: true,
			goWorkContent: `go 1.21

use (
	// this is a comment
	./svc-api
	// another comment
)
`,
			goModContents: map[string]string{
				"/repo/svc-api/go.mod": "module github.com/org/repo/svc-api",
			},
			wantCount: 1,
			wantNames: []string{"svc-api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
	tests := []struct {
		name        string
		modFiles    []string // paths Walk will surface as go.mod files
		modContents map[string]string
		walkErr     error
		readFileErr string // path whose ReadFile call returns an error; "" means no error
		wantCount   int
		wantRootType domain.ProjectType
		wantErr     bool
	}{
		{
			name: "three modules including root",
			modFiles: []string{
				"/repo/go.mod",
				"/repo/services/api/go.mod",
				"/repo/services/worker/go.mod",
			},
			modContents: map[string]string{
				"/repo/go.mod":                "module github.com/org/repo",
				"/repo/services/api/go.mod":   "module github.com/org/repo/services/api",
				"/repo/services/worker/go.mod": "module github.com/org/repo/services/worker",
			},
			wantCount:    3,
			wantRootType: domain.ProjectTypeRoot,
		},
		{
			name:        "no modules found",
			modFiles:    []string{},
			modContents: map[string]string{},
			wantCount:   0,
		},
		{
			name:    "walk error propagates",
			walkErr: errors.New("filesystem error"),
			wantErr: true,
		},
		{
			name: "ReadFile error propagates",
			modFiles: []string{
				"/repo/go.mod",
			},
			modContents: map[string]string{},
			readFileErr: "/repo/go.mod",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			mockFS := mocks.NewMockFileSystem(ctrl)

			// Walk simulates a recursive directory traversal by calling the provided
			// WalkDirFunc for each path in modFiles, or returning walkErr immediately.
			mockFS.EXPECT().Walk("/repo", gomock.Any()).DoAndReturn(
				func(_ string, fn func(string, fs.DirEntry, error) error) error {
					if tt.walkErr != nil {
						return tt.walkErr
					}
					for _, path := range tt.modFiles {
						if err := fn(path, testDirEntry{name: "go.mod"}, nil); err != nil {
							return err
						}
					}
					return nil
				})

			// Register success expectations for all paths except the one designated
			// to fail. The error expectation is set separately so it is always
			// registered regardless of whether readFileErr appears in modContents.
			for path, content := range tt.modContents {
				if path == tt.readFileErr {
					continue
				}
				mockFS.EXPECT().ReadFile(path).Return([]byte(content), nil).AnyTimes()
			}
			if tt.readFileErr != "" {
				mockFS.EXPECT().ReadFile(tt.readFileErr).Return(nil, errors.New("read error"))
			}

			discoverer := adaptergit.NewModuleDiscoverer(mockFS)
			projects, err := discoverer.Discover(context.Background(), "/repo")

			if tt.wantErr {
				if err == nil {
					t.Fatal("Discover() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}

			if len(projects) != tt.wantCount {
				t.Fatalf("got %d projects, want %d", len(projects), tt.wantCount)
			}

			if tt.wantCount > 0 && projects[0].Type != tt.wantRootType {
				t.Errorf("projects[0].Type = %v, want %v", projects[0].Type, tt.wantRootType)
			}
			if tt.wantCount > 1 && projects[1].Type != domain.ProjectTypeGoModule {
				t.Errorf("projects[1].Type = %v, want %v", projects[1].Type, domain.ProjectTypeGoModule)
			}
		})
	}
}

func TestConfiguredDiscoverer_Discover(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		configs       []domain.ProjectConfig
		wantPaths     []string // expected Project.Path after filepath.Clean normalisation
		wantPrefixes  []string // expected Project.TagPrefix
	}{
		{
			// Explicit TagPrefix must be preserved verbatim.
			name: "explicit tag prefix preserved",
			configs: []domain.ProjectConfig{
				{Name: "api", Path: "services/api", TagPrefix: "api/"},
			},
			wantPaths:    []string{"services/api"},
			wantPrefixes: []string{"api/"},
		},
		{
			// Empty TagPrefix defaults to Name + "/".
			name: "default tag prefix from name",
			configs: []domain.ProjectConfig{
				{Name: "worker", Path: "services/worker"},
			},
			wantPaths:    []string{"services/worker"},
			wantPrefixes: []string{"worker/"},
		},
		{
			// Paths with a leading "./" are normalised to match ModuleDiscoverer output.
			name: "path normalisation strips leading ./",
			configs: []domain.ProjectConfig{
				{Name: "shared", Path: "./pkg/shared"},
			},
			wantPaths:    []string{"pkg/shared"},
			wantPrefixes: []string{"shared/"},
		},
		{
			// Multiple projects are all processed in order.
			name: "multiple projects",
			configs: []domain.ProjectConfig{
				{Name: "api", Path: "services/api", TagPrefix: "api/"},
				{Name: "worker", Path: "services/worker"},
			},
			wantPaths:    []string{"services/api", "services/worker"},
			wantPrefixes: []string{"api/", "worker/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			discoverer := adaptergit.NewConfiguredDiscoverer(tt.configs)
			projects, err := discoverer.Discover(context.Background(), "/repo")
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}

			if len(projects) != len(tt.wantPaths) {
				t.Fatalf("got %d projects, want %d", len(projects), len(tt.wantPaths))
			}

			for i := range tt.wantPaths {
				if projects[i].Path != tt.wantPaths[i] {
					t.Errorf("project[%d].Path = %q, want %q", i, projects[i].Path, tt.wantPaths[i])
				}
				if projects[i].TagPrefix != tt.wantPrefixes[i] {
					t.Errorf("project[%d].TagPrefix = %q, want %q", i, projects[i].TagPrefix, tt.wantPrefixes[i])
				}
			}
		})
	}
}

func TestCompositeDiscoverer_FirstNonEmpty(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	empty := mocks.NewMockProjectDiscoverer(ctrl)
	empty.EXPECT().Discover(gomock.Any(), "/repo").Return(nil, nil)

	withProjects := mocks.NewMockProjectDiscoverer(ctrl)
	withProjects.EXPECT().Discover(gomock.Any(), "/repo").Return([]domain.Project{
		{Name: "found"},
	}, nil)

	// neverCalled is registered as the third discoverer. gomock will fail the
	// test if Discover is called on it, enforcing that the composite stops as
	// soon as withProjects returns a non-empty result.
	neverCalled := mocks.NewMockProjectDiscoverer(ctrl)

	composite := adaptergit.NewCompositeDiscoverer(empty, withProjects, neverCalled)
	projects, err := composite.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 1 || projects[0].Name != "found" {
		t.Errorf("expected 'found' project, got %v", projects)
	}
}
