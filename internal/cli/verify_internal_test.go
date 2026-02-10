package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/plan"
)

func TestRunVerify_NoJigDirectory(t *testing.T) {
	// Create a temp directory without .jig
	tmpDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to the temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Reset flags
	verifyRunner = ""
	verifyNoLaunch = false

	cmd := &cobra.Command{}
	cmd.Flags().StringVarP(&verifyRunner, "runner", "r", "", "")
	cmd.Flags().BoolVar(&verifyNoLaunch, "no-launch", false, "")

	err = runVerify(cmd, []string{})
	if err == nil {
		t.Error("expected error when not in jig worktree")
	}
	if err != nil && !strings.Contains(err.Error(), "not in a jig worktree") {
		t.Errorf("expected 'not in a jig worktree' error, got: %v", err)
	}
}

func TestRunVerify_MissingPlanFile(t *testing.T) {
	// Create a temp directory with .jig but no plan.md
	tmpDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .jig directory with issue.json
	jigDir := filepath.Join(tmpDir, ".jig")
	os.MkdirAll(jigDir, 0755)
	os.WriteFile(filepath.Join(jigDir, "issue.json"), []byte(`{"issue_id":"TEST-1"}`), 0644)

	// Change to the temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Reset flags
	verifyRunner = ""
	verifyNoLaunch = false

	cmd := &cobra.Command{}
	cmd.Flags().StringVarP(&verifyRunner, "runner", "r", "", "")
	cmd.Flags().BoolVar(&verifyNoLaunch, "no-launch", false, "")

	err = runVerify(cmd, []string{})
	if err == nil {
		t.Error("expected error when plan.md is missing")
	}
	if err != nil && !strings.Contains(err.Error(), "no plan found") {
		t.Errorf("expected 'no plan found' error, got: %v", err)
	}
}

func TestRunVerify_NoLaunch_Success(t *testing.T) {
	// Create a temp directory with full .jig context
	tmpDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .jig directory with issue.json and plan.md
	jigDir := filepath.Join(tmpDir, ".jig")
	os.MkdirAll(jigDir, 0755)
	os.WriteFile(filepath.Join(jigDir, "issue.json"), []byte(`{"issue_id":"TEST-1"}`), 0644)

	// Create a valid plan
	p := plan.NewPlan("TEST-1", "Test Plan", "testuser")
	p.ProblemStatement = "Test problem"
	p.ProposedSolution = "Test solution"
	content, _ := plan.Serialize(p)
	os.WriteFile(filepath.Join(jigDir, "plan.md"), content, 0644)

	// Change to the temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Reset flags - use --no-launch to skip runner check
	verifyRunner = "claude"
	verifyNoLaunch = true

	cmd := &cobra.Command{}
	cmd.Flags().StringVarP(&verifyRunner, "runner", "r", "claude", "")
	cmd.Flags().BoolVar(&verifyNoLaunch, "no-launch", true, "")

	err = runVerify(cmd, []string{})
	if err != nil {
		t.Errorf("expected success with --no-launch, got: %v", err)
	}
}

func TestRunVerify_RunnerNotFound(t *testing.T) {
	// Save original deps and restore after test
	origDeps := deps
	defer func() { deps = origDeps }()

	// Create mock registry that returns error
	mockReg := newTestMockRegistry()
	mockReg.getErr = fmt.Errorf("unknown runner: nonexistent")
	SetDeps(&Deps{RunnerRegistry: mockReg})

	// Create a temp directory with full .jig context
	tmpDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .jig directory with issue.json and plan.md
	jigDir := filepath.Join(tmpDir, ".jig")
	os.MkdirAll(jigDir, 0755)
	os.WriteFile(filepath.Join(jigDir, "issue.json"), []byte(`{"issue_id":"TEST-1"}`), 0644)

	// Create a valid plan
	p := plan.NewPlan("TEST-1", "Test Plan", "testuser")
	p.ProblemStatement = "Test problem"
	p.ProposedSolution = "Test solution"
	content, _ := plan.Serialize(p)
	os.WriteFile(filepath.Join(jigDir, "plan.md"), content, 0644)

	// Change to the temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Reset flags - DON'T use --no-launch so we hit the runner check
	verifyRunner = "nonexistent"
	verifyNoLaunch = false

	cmd := &cobra.Command{}
	cmd.Flags().StringVarP(&verifyRunner, "runner", "r", "nonexistent", "")
	cmd.Flags().BoolVar(&verifyNoLaunch, "no-launch", false, "")

	err = runVerify(cmd, []string{})
	if err == nil {
		t.Error("expected error for unknown runner")
	}
	if err != nil && !strings.Contains(err.Error(), "runner not found") {
		t.Errorf("expected 'runner not found' error, got: %v", err)
	}
}
