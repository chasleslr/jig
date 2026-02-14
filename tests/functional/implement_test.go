package functional

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charleslr/jig/tests/functional/testenv"
)

func TestImplement_CreatesWorktree(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// First save a plan
	plan := testenv.ValidPlan("IMPL-001", "Test Implementation")
	saveResult := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
	env.AssertSuccess(saveResult)

	// Run implement with --no-launch
	result := env.RunJig("implement", "--no-launch", "IMPL-001")
	env.AssertSuccess(result)

	// Verify the worktree directory exists
	wtDir := env.WorktreeDir()
	entries, err := os.ReadDir(wtDir)
	if err != nil {
		t.Fatalf("failed to read worktree dir: %v", err)
	}

	// Should have at least one worktree entry
	if len(entries) == 0 {
		t.Error("expected at least one worktree directory")
	}

	// Find the worktree for IMPL-001
	var foundWorktree string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "IMPL-001") {
			foundWorktree = filepath.Join(wtDir, entry.Name())
			break
		}
	}

	if foundWorktree == "" {
		t.Error("expected to find worktree for IMPL-001")
	} else {
		// Verify .jig directory exists in worktree
		jigDir := filepath.Join(foundWorktree, ".jig")
		if _, err := os.Stat(jigDir); os.IsNotExist(err) {
			t.Error("expected .jig directory in worktree")
		}

		// Verify plan.md exists in .jig
		planFile := filepath.Join(jigDir, "plan.md")
		if _, err := os.Stat(planFile); os.IsNotExist(err) {
			t.Error("expected plan.md in worktree .jig directory")
		}
	}
}

func TestImplement_ReusesExistingWorktree(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// First save a plan
	plan := testenv.ValidPlan("REUSE-001", "Reuse Test")
	saveResult := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
	env.AssertSuccess(saveResult)

	// Run implement first time
	result1 := env.RunJig("implement", "--no-launch", "REUSE-001")
	env.AssertSuccess(result1)

	// Run implement second time - should succeed and reuse existing worktree
	result2 := env.RunJig("implement", "--no-launch", "REUSE-001")
	env.AssertSuccess(result2)
}

func TestImplement_NoPlanError(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Try to implement a non-existent plan
	// Without a plan cached or on the tracker, this should fail
	// With remote fallback, it will first try to fetch from tracker
	result := env.RunJig("implement", "--no-launch", "NONEXISTENT-001")

	// The command should fail because no plan exists
	// Accept either "no plan found" (when tracker returns nil) or tracker error (when tracker not configured)
	env.AssertFailure(result)
	combinedOutput := result.Stdout + result.Stderr
	if !strings.Contains(combinedOutput, "no plan found") && !strings.Contains(combinedOutput, "tracker") && !strings.Contains(combinedOutput, "failed to load plan") {
		t.Errorf("expected output to contain 'no plan found' or 'tracker' or 'failed to load plan', got:\n%s", combinedOutput)
	}
}

func TestImplement_OutputsWorktreePath(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Save a plan
	plan := testenv.ValidPlan("OUT-001", "Output Test")
	saveResult := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
	env.AssertSuccess(saveResult)

	// Run implement
	result := env.RunJig("implement", "--no-launch", "OUT-001")
	env.AssertSuccess(result)

	// Output should contain worktree path info
	env.AssertOutputContains(result, "Worktree:")
	env.AssertOutputContains(result, "Branch:")
}

func TestImplement_RequiresIssueArg(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Running implement without an issue in non-interactive mode should fail
	// Since our test environment is non-interactive (no tty)
	result := env.RunJig("implement", "--no-launch")

	// Should fail because no ISSUE argument in non-interactive mode
	env.AssertFailure(result)
	env.AssertOutputContains(result, "required")
}

func TestImplement_CreatesIsolatedWorktrees(t *testing.T) {
	// Create two separate environments
	env1 := testenv.New(t)
	defer env1.Cleanup()

	env2 := testenv.New(t)
	defer env2.Cleanup()

	// Save and implement a plan in env1
	plan1 := testenv.ValidPlan("ISO-IMPL-001", "Isolated Impl 1")
	env1.RunJigWithStdin(plan1, "plan", "save", "--no-sync")
	result1 := env1.RunJig("implement", "--no-launch", "ISO-IMPL-001")
	env1.AssertSuccess(result1)

	// Save and implement a different plan in env2
	plan2 := testenv.ValidPlan("ISO-IMPL-002", "Isolated Impl 2")
	env2.RunJigWithStdin(plan2, "plan", "save", "--no-sync")
	result2 := env2.RunJig("implement", "--no-launch", "ISO-IMPL-002")
	env2.AssertSuccess(result2)

	// Verify env1 has its worktree
	wt1Dir := env1.WorktreeDir()
	entries1, err := os.ReadDir(wt1Dir)
	if err != nil {
		t.Fatalf("failed to read env1 worktree dir: %v", err)
	}
	foundEnv1 := false
	for _, entry := range entries1 {
		if strings.Contains(entry.Name(), "ISO-IMPL-001") {
			foundEnv1 = true
		}
		if strings.Contains(entry.Name(), "ISO-IMPL-002") {
			t.Error("env1 should not have ISO-IMPL-002 worktree")
		}
	}
	if !foundEnv1 {
		t.Error("env1 should have ISO-IMPL-001 worktree")
	}

	// Verify env2 has its worktree
	wt2Dir := env2.WorktreeDir()
	entries2, err := os.ReadDir(wt2Dir)
	if err != nil {
		t.Fatalf("failed to read env2 worktree dir: %v", err)
	}
	foundEnv2 := false
	for _, entry := range entries2 {
		if strings.Contains(entry.Name(), "ISO-IMPL-002") {
			foundEnv2 = true
		}
		if strings.Contains(entry.Name(), "ISO-IMPL-001") {
			t.Error("env2 should not have ISO-IMPL-001 worktree")
		}
	}
	if !foundEnv2 {
		t.Error("env2 should have ISO-IMPL-002 worktree")
	}
}
