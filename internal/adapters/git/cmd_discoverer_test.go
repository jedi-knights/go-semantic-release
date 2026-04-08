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

// cmdTestDirEntry wraps testDirEntry; reused from the same test package.

func TestCmdDiscoverer_NoGoMod(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	// No go.mod at root → skip entirely.
	mockFS.EXPECT().Exists("/repo/go.mod").Return(false)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	projects, err := d.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestCmdDiscoverer_GoWorkPresent(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	// go.work present → not a single-module monorepo, skip.
	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(true)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	projects, err := d.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestCmdDiscoverer_NoCmdDir(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(false)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	projects, err := d.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestCmdDiscoverer_SingleService_NoSharedPkg(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(true)

	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/myapp\n\ngo 1.21\n"), nil,
	)

	// Walk cmd/ surfaces cmd/api/main.go
	mockFS.EXPECT().Walk("/repo/cmd", gomock.Any()).DoAndReturn(
		func(_ string, fn func(string, fs.DirEntry, error) error) error {
			_ = fn("/repo/cmd/api", testDirEntry{name: "api", isDir: true}, nil)
			_ = fn("/repo/cmd/api/main.go", testDirEntry{name: "main.go", isDir: false}, nil)
			return nil
		},
	)

	// main.go imports — only internal package from its own service, no shared pkg.
	mockFS.EXPECT().ReadFile("/repo/cmd/api/main.go").Return([]byte(`package main

import (
	"github.com/org/myapp/internal/api"
)

func main() {}
`), nil)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	projects, err := d.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d: %v", len(projects), projects)
	}

	svc := projects[0]
	if svc.Name != "api" {
		t.Errorf("Name = %q, want %q", svc.Name, "api")
	}
	if svc.Path != "cmd/api" {
		t.Errorf("Path = %q, want %q", svc.Path, "cmd/api")
	}
	if svc.Type != domain.ProjectTypeCmdService {
		t.Errorf("Type = %v, want %v", svc.Type, domain.ProjectTypeCmdService)
	}
	if svc.TagPrefix != "api/" {
		t.Errorf("TagPrefix = %q, want %q", svc.TagPrefix, "api/")
	}
	if svc.ModulePath != "github.com/org/myapp" {
		t.Errorf("ModulePath = %q, want %q", svc.ModulePath, "github.com/org/myapp")
	}
}

func TestCmdDiscoverer_MultipleServices_SharedPkg(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(true)

	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/mono\n\ngo 1.21\n"), nil,
	)

	// Walk cmd/ surfaces two services.
	mockFS.EXPECT().Walk("/repo/cmd", gomock.Any()).DoAndReturn(
		func(_ string, fn func(string, fs.DirEntry, error) error) error {
			_ = fn("/repo/cmd/api", testDirEntry{name: "api", isDir: true}, nil)
			_ = fn("/repo/cmd/api/main.go", testDirEntry{name: "main.go", isDir: false}, nil)
			_ = fn("/repo/cmd/worker", testDirEntry{name: "worker", isDir: true}, nil)
			_ = fn("/repo/cmd/worker/main.go", testDirEntry{name: "main.go", isDir: false}, nil)
			return nil
		},
	)

	// api imports shared pkg/queue and its own internal.
	mockFS.EXPECT().ReadFile("/repo/cmd/api/main.go").Return([]byte(`package main

import (
	"github.com/org/mono/internal/api"
	"github.com/org/mono/pkg/queue"
)

func main() {}
`), nil)

	// worker imports shared pkg/queue and its own internal.
	mockFS.EXPECT().ReadFile("/repo/cmd/worker/main.go").Return([]byte(`package main

import (
	"github.com/org/mono/internal/worker"
	"github.com/org/mono/pkg/queue"
)

func main() {}
`), nil)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	projects, err := d.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Expect: api, worker (services) + pkg/queue (library used by >1 service)
	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d: %v", len(projects), projects)
	}

	byName := make(map[string]domain.Project, len(projects))
	for _, p := range projects {
		byName[p.Name] = p
	}

	// Services
	for _, svcName := range []string{"api", "worker"} {
		svc, ok := byName[svcName]
		if !ok {
			t.Errorf("missing project %q", svcName)
			continue
		}
		if svc.Type != domain.ProjectTypeCmdService {
			t.Errorf("%s.Type = %v, want %v", svcName, svc.Type, domain.ProjectTypeCmdService)
		}
		if svc.Path != "cmd/"+svcName {
			t.Errorf("%s.Path = %q, want %q", svcName, svc.Path, "cmd/"+svcName)
		}
		if svc.TagPrefix != svcName+"/" {
			t.Errorf("%s.TagPrefix = %q, want %q", svcName, svc.TagPrefix, svcName+"/")
		}
		// Both services depend on the queue library.
		if len(svc.Dependencies) != 1 || svc.Dependencies[0] != "queue" {
			t.Errorf("%s.Dependencies = %v, want [queue]", svcName, svc.Dependencies)
		}
	}

	// Shared library
	lib, ok := byName["queue"]
	if !ok {
		t.Fatal("missing project 'queue'")
	}
	if lib.Type != domain.ProjectTypeCmdLibrary {
		t.Errorf("queue.Type = %v, want %v", lib.Type, domain.ProjectTypeCmdLibrary)
	}
	if lib.Path != "pkg/queue" {
		t.Errorf("queue.Path = %q, want %q", lib.Path, "pkg/queue")
	}
	if lib.TagPrefix != "queue/" {
		t.Errorf("queue.TagPrefix = %q, want %q", lib.TagPrefix, "queue/")
	}
}

func TestCmdDiscoverer_SharedPkgUsedByOnlyOneService_NotPromoted(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(true)

	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/mono\n"), nil,
	)

	mockFS.EXPECT().Walk("/repo/cmd", gomock.Any()).DoAndReturn(
		func(_ string, fn func(string, fs.DirEntry, error) error) error {
			_ = fn("/repo/cmd/api", testDirEntry{name: "api", isDir: true}, nil)
			_ = fn("/repo/cmd/api/main.go", testDirEntry{name: "main.go", isDir: false}, nil)
			return nil
		},
	)

	// Only one service, so pkg/utils is not shared — no library project created.
	mockFS.EXPECT().ReadFile("/repo/cmd/api/main.go").Return([]byte(`package main

import (
	"github.com/org/mono/pkg/utils"
)

func main() {}
`), nil)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	projects, err := d.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Only the api service; utils not promoted (used by only one service).
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d: %v", len(projects), projects)
	}
	if projects[0].Name != "api" {
		t.Errorf("Name = %q, want api", projects[0].Name)
	}
	// No dependencies since the library isn't promoted.
	if len(projects[0].Dependencies) != 0 {
		t.Errorf("expected no dependencies, got %v", projects[0].Dependencies)
	}
}

func TestCmdDiscoverer_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(true)
	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/mono\n"), nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	mockFS.EXPECT().Walk("/repo/cmd", gomock.Any()).DoAndReturn(
		func(_ string, fn func(string, fs.DirEntry, error) error) error {
			return ctx.Err()
		},
	)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	_, err := d.Discover(ctx, "/repo")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// TestCmdDiscoverer_ServiceNameFromPath verifies that the service name is derived
// from the path components, not from shared state set by a preceding directory entry.
// Walk implementations are not required to visit a directory before its children,
// so the discoverer must be robust to entries arriving out of order.
func TestCmdDiscoverer_ServiceNameFromPath(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(true)
	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/mono\n"), nil,
	)

	// Deliver main.go without a preceding directory entry for "api".
	mockFS.EXPECT().Walk("/repo/cmd", gomock.Any()).DoAndReturn(
		func(_ string, fn func(string, fs.DirEntry, error) error) error {
			_ = fn("/repo/cmd/api/main.go", testDirEntry{name: "main.go", isDir: false}, nil)
			return nil
		},
	)

	mockFS.EXPECT().ReadFile("/repo/cmd/api/main.go").Return([]byte(`package main

func main() {}
`), nil)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	projects, err := d.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d: %v", len(projects), projects)
	}
	if projects[0].Name != "api" {
		t.Errorf("Name = %q, want %q", projects[0].Name, "api")
	}
	if projects[0].Path != "cmd/api" {
		t.Errorf("Path = %q, want %q", projects[0].Path, "cmd/api")
	}
}

// TestCmdDiscoverer_ParseError verifies that a main.go with invalid Go syntax
// causes Discover to return a non-nil error rather than silently ignoring it.
func TestCmdDiscoverer_ParseError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(true)
	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/mono\n"), nil,
	)

	// Propagate the callback return value so that parse errors abort the walk.
	mockFS.EXPECT().Walk("/repo/cmd", gomock.Any()).DoAndReturn(
		func(_ string, fn func(string, fs.DirEntry, error) error) error {
			if err := fn("/repo/cmd/api", testDirEntry{name: "api", isDir: true}, nil); err != nil {
				return err
			}
			return fn("/repo/cmd/api/main.go", testDirEntry{name: "main.go", isDir: false}, nil)
		},
	)

	// Deliberately invalid Go — missing package clause causes a parse error that
	// go/parser cannot recover from even in ImportsOnly mode.
	mockFS.EXPECT().ReadFile("/repo/cmd/api/main.go").Return(
		[]byte("this is not valid go code"), nil,
	)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	_, err := d.Discover(context.Background(), "/repo")
	if err == nil {
		t.Fatal("expected error for invalid Go syntax, got nil")
	}
}

// TestCmdDiscoverer_WalkError verifies that a generic Walk I/O error is propagated.
func TestCmdDiscoverer_WalkError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(true)
	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/mono\n"), nil,
	)

	walkErr := errors.New("disk I/O error")
	mockFS.EXPECT().Walk("/repo/cmd", gomock.Any()).Return(walkErr)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	_, err := d.Discover(context.Background(), "/repo")
	if err == nil {
		t.Fatal("expected error from Walk failure, got nil")
	}
	if !errors.Is(err, walkErr) {
		t.Errorf("error chain does not contain walkErr: %v", err)
	}
}

// TestCmdDiscoverer_DuplicatePkgImport verifies that a service importing the same
// pkg/ package twice (e.g. under two aliases) is counted only once toward the
// shared-library promotion threshold.
func TestCmdDiscoverer_DuplicatePkgImport(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(true)
	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/mono\n"), nil,
	)

	mockFS.EXPECT().Walk("/repo/cmd", gomock.Any()).DoAndReturn(
		func(_ string, fn func(string, fs.DirEntry, error) error) error {
			_ = fn("/repo/cmd/api", testDirEntry{name: "api", isDir: true}, nil)
			_ = fn("/repo/cmd/api/main.go", testDirEntry{name: "main.go", isDir: false}, nil)
			return nil
		},
	)

	// Single service imports pkg/queue twice via different aliases.
	mockFS.EXPECT().ReadFile("/repo/cmd/api/main.go").Return([]byte(`package main

import (
	q1 "github.com/org/mono/pkg/queue"
	q2 "github.com/org/mono/pkg/queue"
)

func main() { _ = q1.New(); _ = q2.New() }
`), nil)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	projects, err := d.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// queue is used by only 1 unique service — must NOT be promoted.
	if len(projects) != 1 {
		t.Fatalf("expected 1 project (api only), got %d: %v", len(projects), projects)
	}
	if projects[0].Name != "api" {
		t.Errorf("Name = %q, want api", projects[0].Name)
	}
	// No dependencies wired since queue wasn't promoted.
	if len(projects[0].Dependencies) != 0 {
		t.Errorf("expected no dependencies, got %v", projects[0].Dependencies)
	}
}

// TestCmdDiscoverer_ContextCancelledMidWalk verifies the in-callback context check:
// if the context is cancelled between two walk callbacks, the subsequent callback
// fires the ctx.Err() guard and aborts.
func TestCmdDiscoverer_ContextCancelledMidWalk(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/go.mod").Return(true)
	mockFS.EXPECT().Exists("/repo/go.work").Return(false)
	mockFS.EXPECT().Exists("/repo/cmd").Return(true)
	mockFS.EXPECT().ReadFile("/repo/go.mod").Return(
		[]byte("module github.com/org/mono\n"), nil,
	)

	ctx, cancel := context.WithCancel(context.Background())

	mockFS.EXPECT().Walk("/repo/cmd", gomock.Any()).DoAndReturn(
		func(_ string, fn func(string, fs.DirEntry, error) error) error {
			// First entry succeeds.
			if err := fn("/repo/cmd/api", testDirEntry{name: "api", isDir: true}, nil); err != nil {
				return err
			}
			// Cancel between entries — next callback should trigger the in-callback guard.
			cancel()
			return fn("/repo/cmd/api/main.go", testDirEntry{name: "main.go", isDir: false}, nil)
		},
	)

	d := adaptergit.NewCmdDiscoverer(mockFS)
	_, err := d.Discover(ctx, "/repo")
	if err == nil {
		t.Fatal("expected error from mid-walk cancellation, got nil")
	}
}
