package plugins_test

import (
	"context"
	"io/fs"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestPreparePlugin_UpdateVersionFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	mockFS.EXPECT().WriteFile(
		"/repo/VERSION",
		[]byte("2.0.0\n"),
		fs.FileMode(0644),
	).Return(nil)

	plugin := plugins.NewPreparePlugin(mockFS, mockLogger, plugins.PrepareConfig{
		VersionFile: "VERSION",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(2, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_UpdateChangelog(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// Existing changelog.
	mockFS.EXPECT().Exists("/repo/CHANGELOG.md").Return(true)
	mockFS.EXPECT().ReadFile("/repo/CHANGELOG.md").Return(
		[]byte("# Changelog\n\n## 1.0.0\n\n- Initial release\n"), nil)

	mockFS.EXPECT().WriteFile(
		"/repo/CHANGELOG.md",
		gomock.Any(), // We'll verify content structure.
		fs.FileMode(0644),
	).DoAndReturn(func(_ string, data []byte, _ fs.FileMode) error {
		content := string(data)
		if content[:11] != "# Changelog" {
			t.Errorf("expected changelog to start with title, got: %s", content[:20])
		}
		return nil
	})

	plugin := plugins.NewPreparePlugin(mockFS, mockLogger, plugins.PrepareConfig{
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 2.0.0\n\n### Features\n- new stuff",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(2, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_NewChangelog(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	mockFS.EXPECT().Exists("/repo/CHANGELOG.md").Return(false)

	mockFS.EXPECT().WriteFile(
		"/repo/CHANGELOG.md",
		gomock.Any(),
		fs.FileMode(0644),
	).DoAndReturn(func(_ string, data []byte, _ fs.FileMode) error {
		content := string(data)
		if content[:11] != "# Changelog" {
			t.Errorf("expected new changelog to start with title")
		}
		return nil
	})

	plugin := plugins.NewPreparePlugin(mockFS, mockLogger, plugins.PrepareConfig{
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 1.0.0\n\n- first release",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_NilProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)

	plugin := plugins.NewPreparePlugin(mockFS, mockLogger, plugins.PrepareConfig{
		VersionFile:   "VERSION",
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{CurrentProject: nil}

	err := plugin.Prepare(context.Background(), rc)
	if err != nil {
		t.Fatalf("Prepare() with nil project should not error, got: %v", err)
	}
}
