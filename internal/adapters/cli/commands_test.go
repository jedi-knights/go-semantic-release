package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// newChangelogCmd
// ---------------------------------------------------------------------------

func TestNewChangelogCmd_NonNil(t *testing.T) {
	cmd := newChangelogCmd(&rootOptions{})
	if cmd == nil {
		t.Fatal("newChangelogCmd returned nil")
	}
}

func TestNewChangelogCmd_Use(t *testing.T) {
	cmd := newChangelogCmd(&rootOptions{})
	if cmd.Use != "changelog" {
		t.Errorf("Use = %q, want %q", cmd.Use, "changelog")
	}
}

func TestNewChangelogCmd_HasRunE(t *testing.T) {
	cmd := newChangelogCmd(&rootOptions{})
	if cmd.RunE == nil {
		t.Error("RunE should be set on changelog command")
	}
}

// ---------------------------------------------------------------------------
// newConfigCmd
// ---------------------------------------------------------------------------

func TestNewConfigCmd_NonNil(t *testing.T) {
	cmd := newConfigCmd()
	if cmd == nil {
		t.Fatal("newConfigCmd returned nil")
	}
}

func TestNewConfigCmd_Use(t *testing.T) {
	cmd := newConfigCmd()
	if cmd.Use != "config" {
		t.Errorf("Use = %q, want %q", cmd.Use, "config")
	}
}

func TestNewConfigCmd_HasInitSubcommand(t *testing.T) {
	cmd := newConfigCmd()
	sub, _, err := cmd.Find([]string{"init"})
	if err != nil {
		t.Fatalf("Find(init) error: %v", err)
	}
	if sub == nil || sub.Use != "init" {
		t.Errorf("expected 'init' subcommand, got %v", sub)
	}
}

// ---------------------------------------------------------------------------
// newConfigInitCmd
// ---------------------------------------------------------------------------

func TestNewConfigInitCmd_NonNil(t *testing.T) {
	cmd := newConfigInitCmd()
	if cmd == nil {
		t.Fatal("newConfigInitCmd returned nil")
	}
}

func TestNewConfigInitCmd_Use(t *testing.T) {
	cmd := newConfigInitCmd()
	if cmd.Use != "init" {
		t.Errorf("Use = %q, want %q", cmd.Use, "init")
	}
}

func TestNewConfigInitCmd_HasRunE(t *testing.T) {
	cmd := newConfigInitCmd()
	if cmd.RunE == nil {
		t.Error("RunE should be set on config init command")
	}
}

// ---------------------------------------------------------------------------
// runConfigInit
// ---------------------------------------------------------------------------

// runConfigInit hardcodes ".semantic-release.yaml" relative to the process
// working directory, so we chdir to a temp dir and restore afterwards.
func TestRunConfigInit_WritesFile(t *testing.T) {
	t.Helper()

	dir := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restoring cwd: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}

	cmd := newConfigInitCmd()
	if err := runConfigInit(cmd, nil); err != nil {
		t.Fatalf("runConfigInit error: %v", err)
	}

	expected := filepath.Join(dir, ".semantic-release.yaml")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Errorf("expected file %q to be created, but it does not exist", expected)
	}
}

// ---------------------------------------------------------------------------
// newDetectProjectsCmd
// ---------------------------------------------------------------------------

func TestNewDetectProjectsCmd_NonNil(t *testing.T) {
	cmd := newDetectProjectsCmd(&rootOptions{})
	if cmd == nil {
		t.Fatal("newDetectProjectsCmd returned nil")
	}
}

func TestNewDetectProjectsCmd_Use(t *testing.T) {
	cmd := newDetectProjectsCmd(&rootOptions{})
	if cmd.Use != "detect-projects" {
		t.Errorf("Use = %q, want %q", cmd.Use, "detect-projects")
	}
}

func TestNewDetectProjectsCmd_HasRunE(t *testing.T) {
	cmd := newDetectProjectsCmd(&rootOptions{})
	if cmd.RunE == nil {
		t.Error("RunE should be set on detect-projects command")
	}
}

// ---------------------------------------------------------------------------
// newLintCmd
// ---------------------------------------------------------------------------

func TestNewLintCmd_NonNil(t *testing.T) {
	cmd := newLintCmd(&rootOptions{})
	if cmd == nil {
		t.Fatal("newLintCmd returned nil")
	}
}

func TestNewLintCmd_Use(t *testing.T) {
	cmd := newLintCmd(&rootOptions{})
	if cmd.Use != "lint" {
		t.Errorf("Use = %q, want %q", cmd.Use, "lint")
	}
}

func TestNewLintCmd_HasRunE(t *testing.T) {
	cmd := newLintCmd(&rootOptions{})
	if cmd.RunE == nil {
		t.Error("RunE should be set on lint command")
	}
}

// ---------------------------------------------------------------------------
// newPlanCmd
// ---------------------------------------------------------------------------

func TestNewPlanCmd_NonNil(t *testing.T) {
	cmd := newPlanCmd(&rootOptions{})
	if cmd == nil {
		t.Fatal("newPlanCmd returned nil")
	}
}

func TestNewPlanCmd_Use(t *testing.T) {
	cmd := newPlanCmd(&rootOptions{})
	if cmd.Use != "plan" {
		t.Errorf("Use = %q, want %q", cmd.Use, "plan")
	}
}

func TestNewPlanCmd_HasRunE(t *testing.T) {
	cmd := newPlanCmd(&rootOptions{})
	if cmd.RunE == nil {
		t.Error("RunE should be set on plan command")
	}
}

// ---------------------------------------------------------------------------
// newVerifyCmd
// ---------------------------------------------------------------------------

func TestNewVerifyCmd_NonNil(t *testing.T) {
	cmd := newVerifyCmd(&rootOptions{})
	if cmd == nil {
		t.Fatal("newVerifyCmd returned nil")
	}
}

func TestNewVerifyCmd_Use(t *testing.T) {
	cmd := newVerifyCmd(&rootOptions{})
	if cmd.Use != "verify" {
		t.Errorf("Use = %q, want %q", cmd.Use, "verify")
	}
}

func TestNewVerifyCmd_HasRunE(t *testing.T) {
	cmd := newVerifyCmd(&rootOptions{})
	if cmd.RunE == nil {
		t.Error("RunE should be set on verify command")
	}
}

// ---------------------------------------------------------------------------
// newVersionCmd
// ---------------------------------------------------------------------------

func TestNewVersionCmd_NonNil(t *testing.T) {
	cmd := newVersionCmd(&rootOptions{})
	if cmd == nil {
		t.Fatal("newVersionCmd returned nil")
	}
}

func TestNewVersionCmd_Use(t *testing.T) {
	cmd := newVersionCmd(&rootOptions{})
	if cmd.Use != "version" {
		t.Errorf("Use = %q, want %q", cmd.Use, "version")
	}
}

func TestNewVersionCmd_HasRunE(t *testing.T) {
	cmd := newVersionCmd(&rootOptions{})
	if cmd.RunE == nil {
		t.Error("RunE should be set on version command")
	}
}

// ---------------------------------------------------------------------------
// NewRootCmd
// ---------------------------------------------------------------------------

func TestNewRootCmd_NonNil(t *testing.T) {
	cmd := NewRootCmd()
	if cmd == nil {
		t.Fatal("NewRootCmd returned nil")
	}
}

func TestNewRootCmd_HasSubcommands(t *testing.T) {
	cmd := NewRootCmd()
	if len(cmd.Commands()) == 0 {
		t.Error("NewRootCmd should register at least one subcommand")
	}
}

func TestNewRootCmd_HasVersionSubcommand(t *testing.T) {
	cmd := NewRootCmd()
	sub, _, err := cmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("Find(version) error: %v", err)
	}
	// cmd.Find returns the root command itself when nothing matches;
	// a successful match returns a command whose Use is the subcommand name.
	if sub == nil || sub == cmd {
		t.Fatal("version subcommand not found under root")
	}
	if sub.Use != "version" {
		t.Errorf("found command Use = %q, want %q", sub.Use, "version")
	}
}

func TestNewRootCmd_ExpectedSubcommands(t *testing.T) {
	root := NewRootCmd()

	wants := []string{
		"plan",
		"version",
		"changelog",
		"detect-projects",
		"verify",
		"config",
		"lint",
	}

	for _, name := range wants {
		t.Run(name, func(t *testing.T) {
			found := false
			for _, sub := range root.Commands() {
				if sub.Use == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("subcommand %q not registered on root", name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildContainerWithWorkDir
// ---------------------------------------------------------------------------

// buildContainerWithWorkDir reads the config relative to the process working
// directory via os.Getwd(), so we chdir to a temp dir before calling it.
// The temp dir has no config file; the provider falls back to defaults so the
// container should still be constructed successfully.
func TestBuildContainerWithWorkDir_NonNil(t *testing.T) {
	dir := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	t.Cleanup(func() {
		if chErr := os.Chdir(orig); chErr != nil {
			t.Errorf("restoring cwd: %v", chErr)
		}
	})
	if chErr := os.Chdir(dir); chErr != nil {
		t.Fatalf("chdir to temp dir: %v", chErr)
	}

	// ciFlag=true prevents the auto dry-run guard from interfering; cfgFile=""
	// lets the provider search for ".semantic-release.yaml" (absent here) and
	// fall back to a default config.
	opts := &rootOptions{ciFlag: true}
	container, workDir, err := buildContainerWithWorkDir(opts)
	if err != nil {
		t.Fatalf("buildContainerWithWorkDir error: %v", err)
	}
	if container == nil {
		t.Error("expected non-nil container")
	}
	if workDir == "" {
		t.Error("expected non-empty workDir")
	}
}

// Ensure that *cobra.Command is the correct type returned by all new*Cmd
// constructors. This compile-time assertion verifies the return types without
// an explicit var block that would only be evaluated at runtime.
var (
	_ *cobra.Command = newChangelogCmd(&rootOptions{})
	_ *cobra.Command = newConfigCmd()
	_ *cobra.Command = newConfigInitCmd()
	_ *cobra.Command = newDetectProjectsCmd(&rootOptions{})
	_ *cobra.Command = newLintCmd(&rootOptions{})
	_ *cobra.Command = newPlanCmd(&rootOptions{})
	_ *cobra.Command = newVerifyCmd(&rootOptions{})
	_ *cobra.Command = newVersionCmd(&rootOptions{})
	_ *cobra.Command = NewRootCmd()
)

// ---------------------------------------------------------------------------
// Integration helpers — set up a real git repo in a temp dir and chdir into it.
// ---------------------------------------------------------------------------

// setupCLITestRepo creates a real git repository in a temp directory, writes a
// conventional commit, and os.Chdir's the process into it. The original working
// directory is restored via t.Cleanup.
func setupCLITestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restoring cwd: %v", err)
		}
	})

	gitCmds := [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		if out, err := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "README.md"}, {"commit", "-m", "feat: initial commit"}} {
		if out, err := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return dir
}

// ---------------------------------------------------------------------------
// runDetectProjects integration tests
// ---------------------------------------------------------------------------

func TestRunDetectProjects_Text(t *testing.T) {
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true}
	cmd := newDetectProjectsCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runDetectProjects(cmd, nil, opts); err != nil {
		t.Fatalf("runDetectProjects: %v", err)
	}
	if !strings.Contains(buf.String(), "Discovered") {
		t.Errorf("output = %q, want to contain 'Discovered'", buf.String())
	}
}

func TestRunDetectProjects_JSON(t *testing.T) {
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true, jsonOut: true}
	cmd := newDetectProjectsCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runDetectProjects(cmd, nil, opts); err != nil {
		t.Fatalf("runDetectProjects JSON: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	// JSON output starts with '[' (array) or '{' (object).
	if !strings.HasPrefix(out, "[") && !strings.HasPrefix(out, "{") {
		t.Errorf("output does not look like JSON: %q", out)
	}
}

// ---------------------------------------------------------------------------
// runLint integration tests
// ---------------------------------------------------------------------------

func TestRunLint_PassingCommit(t *testing.T) {
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true}
	cmd := newLintCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runLint(cmd, nil, opts); err != nil {
		t.Fatalf("runLint: %v", err)
	}
	if !strings.Contains(buf.String(), "pass lint checks") {
		t.Errorf("output = %q, want 'pass lint checks'", buf.String())
	}
}

func TestRunLint_WithViolation(t *testing.T) {
	// Add a non-conventional commit on top of the initial one.
	dir := setupCLITestRepo(t)

	if err := os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "extra.txt"}, {"commit", "-m", "bad commit message"}} {
		if out, err := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}

	opts := &rootOptions{ciFlag: true}
	cmd := newLintCmd(opts)
	var buf, errBuf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&errBuf)
	cmd.SetContext(context.Background())

	err := runLint(cmd, nil, opts)
	// A violation at error severity returns ErrQuietExit.
	if err != nil && !errors.Is(err, ErrQuietExit) {
		t.Errorf("runLint unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// runPlan integration tests
// ---------------------------------------------------------------------------

func TestRunPlan_NoTags(t *testing.T) {
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true}
	cmd := newPlanCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runPlan(cmd, nil, opts); err != nil {
		t.Fatalf("runPlan: %v", err)
	}
	if !strings.Contains(buf.String(), "Branch:") {
		t.Errorf("output = %q, want to contain 'Branch:'", buf.String())
	}
}

func TestRunPlan_JSON(t *testing.T) {
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true, jsonOut: true}
	cmd := newPlanCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runPlan(cmd, nil, opts); err != nil {
		t.Fatalf("runPlan JSON: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(out, "{") {
		t.Errorf("output does not look like JSON: %q", out)
	}
}

// ---------------------------------------------------------------------------
// runChangelog integration tests
// ---------------------------------------------------------------------------

func TestRunChangelog_NoTags(t *testing.T) {
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true}
	cmd := newChangelogCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runChangelog(cmd, nil, opts); err != nil {
		t.Fatalf("runChangelog: %v", err)
	}
	// Either produces changelog content or "No releasable changes found."
	out := buf.String()
	if out == "" {
		t.Error("runChangelog produced empty output")
	}
}

// ---------------------------------------------------------------------------
// runVersion integration tests
// ---------------------------------------------------------------------------

func TestRunVersion_NoTags(t *testing.T) {
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true}
	cmd := newVersionCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runVersion(cmd, nil, opts); err != nil {
		t.Fatalf("runVersion: %v", err)
	}
	// Output should contain a version number.
	out := buf.String()
	if out == "" {
		t.Error("runVersion produced empty output")
	}
}

// ---------------------------------------------------------------------------
// runVerify integration tests
// ---------------------------------------------------------------------------

func TestRunVerify_NoBranchPolicy(t *testing.T) {
	// With no branches configured the verifier reports a failure and runVerify
	// returns ErrQuietExit — that is the expected behavior, not an error.
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true}
	cmd := newVerifyCmd(opts)
	var buf, errBuf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&errBuf)
	cmd.SetContext(context.Background())

	err := runVerify(cmd, nil, opts)
	if err != nil && !errors.Is(err, ErrQuietExit) {
		t.Errorf("runVerify unexpected error: %v", err)
	}
}

// setupCLIChoreRepo is like setupCLITestRepo but commits a chore: message so
// that the release planner finds no releasable changes.
func setupCLIChoreRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restoring cwd: %v", err)
		}
	})

	gitCmds := [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range gitCmds {
		if out, err := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "README.md"}, {"commit", "-m", "chore: initial commit"}} {
		if out, err := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return dir
}

// ---------------------------------------------------------------------------
// runRelease integration tests
// ---------------------------------------------------------------------------

func TestRunRelease_AutoDryRun(t *testing.T) {
	// noCIFlag=true causes the auto-dry-run guard to engage: cfg.DryRun=true,
	// cfg.CI=false. The release plan has a releasable project (feat commit), so
	// the executor runs in dry-run mode and prints a "[dry run]" line.
	setupCLITestRepo(t)

	opts := &rootOptions{noCIFlag: true}
	cmd := newVersionCmd(opts) // use any command that has RunE — we call runRelease directly
	var buf, errBuf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&errBuf)
	cmd.SetContext(context.Background())

	err := runRelease(cmd, nil, opts)
	if err != nil {
		t.Fatalf("runRelease AutoDryRun: %v", err)
	}
	// printReleaseResult should have printed at least one line.
	if buf.String() == "" {
		t.Error("runRelease produced empty output")
	}
}

func TestRunRelease_DryRunExecution(t *testing.T) {
	// noInteractive=true skips the confirmation prompt so the release executes.
	// noCIFlag=true triggers auto-dry-run (cfg.DryRun=true, cfg.CI=false), which
	// covers the "note" warning line and runs the executor in dry-run mode without
	// touching git remotes.
	setupCLITestRepo(t)

	opts := &rootOptions{noCIFlag: true, noInteractive: true}
	cmd := newVersionCmd(opts)
	var buf, errBuf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&errBuf)
	cmd.SetContext(context.Background())

	err := runRelease(cmd, nil, opts)
	if err != nil {
		t.Fatalf("runRelease DryRunExecution: %v", err)
	}
	// dry-run prints "[dry run] repo: ..."
	if buf.String() == "" {
		t.Error("expected output from dry-run execution, got none")
	}
}

func TestRunRelease_NoReleasable(t *testing.T) {
	// A chore commit produces no releasable changes; runRelease should print
	// "No releasable changes found." and return nil.
	setupCLIChoreRepo(t)

	opts := &rootOptions{ciFlag: true}
	cmd := newVersionCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runRelease(cmd, nil, opts); err != nil {
		t.Fatalf("runRelease NoReleasable: %v", err)
	}
	if !strings.Contains(buf.String(), "No releasable changes found.") {
		t.Errorf("output = %q, want 'No releasable changes found.'", buf.String())
	}
}

// ---------------------------------------------------------------------------
// Additional runChangelog / runVersion / runPlan branches
// ---------------------------------------------------------------------------

func TestRunChangelog_NoReleasable(t *testing.T) {
	setupCLIChoreRepo(t)

	opts := &rootOptions{ciFlag: true}
	cmd := newChangelogCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runChangelog(cmd, nil, opts); err != nil {
		t.Fatalf("runChangelog NoReleasable: %v", err)
	}
	if !strings.Contains(buf.String(), "No releasable changes found.") {
		t.Errorf("output = %q, want 'No releasable changes found.'", buf.String())
	}
}

func TestRunVersion_NoChange(t *testing.T) {
	// A chore commit results in no version bump; runVersion prints "(no change)".
	setupCLIChoreRepo(t)

	opts := &rootOptions{ciFlag: true}
	cmd := newVersionCmd(opts)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())

	if err := runVersion(cmd, nil, opts); err != nil {
		t.Fatalf("runVersion NoChange: %v", err)
	}
	if !strings.Contains(buf.String(), "no change") {
		t.Errorf("output = %q, want to contain '(no change)'", buf.String())
	}
}

func TestRunPlan_ProjectFilterNotFound(t *testing.T) {
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true, project: "nonexistent"}
	cmd := newPlanCmd(opts)
	cmd.SetContext(context.Background())

	err := runPlan(cmd, nil, opts)
	if err == nil {
		t.Fatal("expected error for nonexistent project, got nil")
	}
}

func TestRunVersion_ProjectFilterNotFound(t *testing.T) {
	setupCLITestRepo(t)

	opts := &rootOptions{ciFlag: true, project: "nonexistent"}
	cmd := newVersionCmd(opts)
	cmd.SetContext(context.Background())

	err := runVersion(cmd, nil, opts)
	if err == nil {
		t.Fatal("expected error for nonexistent project, got nil")
	}
}
