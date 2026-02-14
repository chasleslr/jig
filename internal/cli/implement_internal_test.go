package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/runner"
)

// testMockRunner implements runner.Runner for testing
type testMockRunner struct {
	name         string
	available    bool
	prepareErr   error
	launchErr    error
	prepareCalls int
	launchCalls  int
}

func (m *testMockRunner) Name() string      { return m.name }
func (m *testMockRunner) Available() bool   { return m.available }
func (m *testMockRunner) Prepare(ctx context.Context, opts *runner.PrepareOpts) error {
	m.prepareCalls++
	return m.prepareErr
}
func (m *testMockRunner) Launch(ctx context.Context, opts *runner.LaunchOpts) (*runner.LaunchResult, error) {
	m.launchCalls++
	if m.launchErr != nil {
		return nil, m.launchErr
	}
	return &runner.LaunchResult{ExitCode: 0}, nil
}

// testMockRegistry implements RunnerRegistry for testing
type testMockRegistry struct {
	runners map[string]runner.Runner
	getErr  error
}

func newTestMockRegistry() *testMockRegistry {
	return &testMockRegistry{
		runners: make(map[string]runner.Runner),
	}
}

func (m *testMockRegistry) Get(name string) (runner.Runner, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	r, ok := m.runners[name]
	if !ok {
		return nil, fmt.Errorf("unknown runner: %s", name)
	}
	return r, nil
}

func (m *testMockRegistry) Register(r runner.Runner) {
	m.runners[r.Name()] = r
}

func TestRunImplement_RunnerNotFound(t *testing.T) {
	// Save original deps and restore after test
	origDeps := deps
	defer func() { deps = origDeps }()

	// Create mock registry that returns error
	mockReg := newTestMockRegistry()
	mockReg.getErr = fmt.Errorf("unknown runner: nonexistent")
	SetDeps(&Deps{RunnerRegistry: mockReg})

	// Create a temp git repo for the test
	tmpDir := createTestGitRepo(t)
	defer os.RemoveAll(tmpDir)

	// Set up JIG_HOME with a cached plan
	jigHome := createTestJigHome(t)
	defer os.RemoveAll(jigHome)
	os.Setenv("JIG_HOME", jigHome)
	defer os.Unsetenv("JIG_HOME")

	// Save a plan to cache so we get past the plan lookup stage
	createTestPlanInCache(t, jigHome, "TEST-123")

	// Change to the temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Reset flags
	implRunner = "nonexistent"
	implNoLaunch = true
	implNoAutoAccept = false

	// Create a mock command
	cmd := &cobra.Command{}
	cmd.Flags().StringVarP(&implRunner, "runner", "r", "nonexistent", "")
	cmd.Flags().BoolVar(&implNoLaunch, "no-launch", true, "")
	cmd.Flags().BoolVar(&implNoAutoAccept, "no-auto-accept", false, "")

	// Run the command - should fail with runner not found
	err := runImplement(cmd, []string{"TEST-123"})
	if err == nil {
		t.Error("expected error for unknown runner")
	}
	if err != nil && !strings.Contains(err.Error(), "runner not found") {
		t.Errorf("expected 'runner not found' error, got: %v", err)
	}
}

func TestRunImplement_NoLaunch_SkipsAvailabilityCheck(t *testing.T) {
	// Save original deps and restore after test
	origDeps := deps
	defer func() { deps = origDeps }()

	// Create mock registry with unavailable runner
	mockReg := newTestMockRegistry()
	mockRunner := &testMockRunner{name: "claude", available: false}
	mockReg.Register(mockRunner)
	SetDeps(&Deps{RunnerRegistry: mockReg})

	// Create a temp git repo for the test
	tmpDir := createTestGitRepo(t)
	defer os.RemoveAll(tmpDir)

	// Set up JIG_HOME
	jigHome := createTestJigHome(t)
	defer os.RemoveAll(jigHome)
	os.Setenv("JIG_HOME", jigHome)
	defer os.Unsetenv("JIG_HOME")

	// Save a plan to cache
	createTestPlanInCache(t, jigHome, "TEST-123")

	// Change to the temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Reset flags
	implRunner = "claude"
	implNoLaunch = true
	implNoAutoAccept = false

	// Create a mock command with --no-launch
	cmd := &cobra.Command{}
	cmd.Flags().StringVarP(&implRunner, "runner", "r", "claude", "")
	cmd.Flags().BoolVar(&implNoLaunch, "no-launch", true, "")
	cmd.Flags().BoolVar(&implNoAutoAccept, "no-auto-accept", false, "")

	// Run the command - should succeed even though runner is unavailable
	err := runImplement(cmd, []string{"TEST-123"})
	// Should NOT fail due to runner availability since --no-launch is set
	if err != nil && strings.Contains(err.Error(), "not available") {
		t.Errorf("--no-launch should skip availability check, got: %v", err)
	}
}

// Helper functions

func createTestGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "jig-test-repo-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@test.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	// Create initial commit
	readmePath := filepath.Join(tmpDir, "README.md")
	os.WriteFile(readmePath, []byte("# Test\n"), 0644)
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "init")
	runGit(t, tmpDir, "branch", "-M", "main")

	return tmpDir
}

func createTestJigHome(t *testing.T) string {
	t.Helper()
	jigHome, err := os.MkdirTemp("", "jig-home-*")
	if err != nil {
		t.Fatalf("failed to create jig home: %v", err)
	}
	return jigHome
}

func createTestPlanInCache(t *testing.T, jigHome, planID string) {
	t.Helper()

	// Create cache directories
	cacheDir := filepath.Join(jigHome, "cache", "plans")
	os.MkdirAll(cacheDir, 0755)

	// Create a minimal plan
	p := plan.NewPlan(planID, "Test Plan", "testuser")
	p.ProblemStatement = "Test problem"
	p.ProposedSolution = "Test solution"

	content, _ := plan.Serialize(p)
	os.WriteFile(filepath.Join(cacheDir, planID+".md"), content, 0644)

	// Also create the JSON cache file
	jsonContent := fmt.Sprintf(`{"plan":{"id":"%s","title":"Test Plan","status":"draft","author":"testuser"},"issue_id":"%s"}`, planID, planID)
	os.WriteFile(filepath.Join(cacheDir, planID+".json"), []byte(jsonContent), 0644)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
