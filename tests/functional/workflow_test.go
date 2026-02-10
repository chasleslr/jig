package functional

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/charleslr/jig/tests/functional/testenv"
)

// TestFullWorkflow_PlanToImplement tests the complete workflow from plan save
// through implement to verify.
func TestFullWorkflow_PlanToImplement(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Step 1: Save a plan
	plan := testenv.ValidPlan("FLOW-001", "Full Workflow Test")
	saveResult := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
	env.AssertSuccess(saveResult)

	// Step 2: List plans - verify it's there
	listResult := env.RunJig("plan", "list")
	env.AssertSuccess(listResult)
	env.AssertStdoutContains(listResult, "FLOW-001")

	// Step 3: Show the plan
	showResult := env.RunJig("plan", "show", "FLOW-001", "--raw")
	env.AssertSuccess(showResult)
	env.AssertStdoutContains(showResult, "Full Workflow Test")

	// Step 4: Implement the plan
	implResult := env.RunJig("implement", "--no-launch", "FLOW-001")
	env.AssertSuccess(implResult)

	// Step 5: Find and verify the worktree
	wtDir := env.WorktreeDir()
	entries, err := os.ReadDir(wtDir)
	if err != nil {
		t.Fatalf("failed to read worktree dir: %v", err)
	}

	var worktreePath string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "FLOW-001") {
			worktreePath = filepath.Join(wtDir, entry.Name())
			break
		}
	}

	if worktreePath == "" {
		t.Fatal("could not find worktree for FLOW-001")
	}

	// Step 6: Verify the worktree has the expected structure
	jigDir := filepath.Join(worktreePath, ".jig")
	if _, err := os.Stat(jigDir); os.IsNotExist(err) {
		t.Error("expected .jig directory in worktree")
	}

	planFile := filepath.Join(jigDir, "plan.md")
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		t.Error("expected plan.md in worktree")
	}

	issueFile := filepath.Join(jigDir, "issue.json")
	if _, err := os.Stat(issueFile); os.IsNotExist(err) {
		t.Error("expected issue.json in worktree")
	}

	// Step 7: Run verify from the worktree
	verifyResult := env.RunJigInDir(worktreePath, "verify", "--no-launch")
	env.AssertSuccess(verifyResult)
	env.AssertOutputContains(verifyResult, "FLOW-001")
}

// TestCacheIsolation verifies that multiple parallel test environments
// don't interfere with each other.
func TestCacheIsolation(t *testing.T) {
	// Create multiple environments in parallel
	const numEnvs = 5

	var wg sync.WaitGroup
	errors := make(chan error, numEnvs)

	for i := 0; i < numEnvs; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			env := testenv.New(t)
			defer env.Cleanup()

			// Each environment saves a unique plan
			planID := "PARALLEL-" + string(rune('A'+id))
			plan := testenv.ValidPlan(planID, "Parallel Test "+planID)
			saveResult := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")

			if !saveResult.Success() {
				errors <- nil // We can't use t.Error in goroutines
				return
			}

			// Verify only this plan exists in this environment
			listResult := env.RunJig("plan", "list")
			if !listResult.Success() {
				errors <- nil
				return
			}

			// Should have our plan
			if !strings.Contains(listResult.Stdout, planID) {
				errors <- nil
				return
			}

			// Should not have other parallel plans
			for j := 0; j < numEnvs; j++ {
				if j == id {
					continue
				}
				otherID := "PARALLEL-" + string(rune('A'+j))
				if strings.Contains(listResult.Stdout, otherID) {
					errors <- nil
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		if err != nil {
			t.Error(err)
		}
	}
}

// TestWorktreeIsolation verifies that worktrees are created in isolated locations.
func TestWorktreeIsolation(t *testing.T) {
	env1 := testenv.New(t)
	defer env1.Cleanup()

	env2 := testenv.New(t)
	defer env2.Cleanup()

	// Save and implement plans in both environments
	plan1 := testenv.ValidPlan("WT-ISO-001", "Worktree Isolation 1")
	env1.RunJigWithStdin(plan1, "plan", "save", "--no-sync")
	env1.RunJig("implement", "--no-launch", "WT-ISO-001")

	plan2 := testenv.ValidPlan("WT-ISO-002", "Worktree Isolation 2")
	env2.RunJigWithStdin(plan2, "plan", "save", "--no-sync")
	env2.RunJig("implement", "--no-launch", "WT-ISO-002")

	// Verify worktrees are in different locations
	if env1.WorktreeDir() == env2.WorktreeDir() {
		t.Error("worktree directories should be different")
	}

	// Verify each environment only has its own worktree
	entries1, _ := os.ReadDir(env1.WorktreeDir())
	entries2, _ := os.ReadDir(env2.WorktreeDir())

	found1 := false
	found2InEnv1 := false
	for _, e := range entries1 {
		if strings.Contains(e.Name(), "WT-ISO-001") {
			found1 = true
		}
		if strings.Contains(e.Name(), "WT-ISO-002") {
			found2InEnv1 = true
		}
	}

	found2 := false
	found1InEnv2 := false
	for _, e := range entries2 {
		if strings.Contains(e.Name(), "WT-ISO-002") {
			found2 = true
		}
		if strings.Contains(e.Name(), "WT-ISO-001") {
			found1InEnv2 = true
		}
	}

	if !found1 {
		t.Error("env1 should have WT-ISO-001 worktree")
	}
	if !found2 {
		t.Error("env2 should have WT-ISO-002 worktree")
	}
	if found2InEnv1 {
		t.Error("env1 should not have WT-ISO-002 worktree")
	}
	if found1InEnv2 {
		t.Error("env2 should not have WT-ISO-001 worktree")
	}
}

// TestPlanStatusTransition verifies plan status changes during workflow.
func TestPlanStatusTransition(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Save a plan with draft status
	plan := testenv.PlanWithCustomStatus("STATUS-001", "Status Test", "draft")
	saveResult := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
	env.AssertSuccess(saveResult)

	// Show the plan - should be draft
	showResult := env.RunJig("plan", "show", "STATUS-001", "--raw")
	env.AssertSuccess(showResult)
	env.AssertStdoutContains(showResult, "status: draft")

	// Implement the plan - this should transition to in-progress
	implResult := env.RunJig("implement", "--no-launch", "STATUS-001")
	env.AssertSuccess(implResult)

	// Show the plan again - should be in-progress
	showResult2 := env.RunJig("plan", "show", "STATUS-001", "--raw")
	env.AssertSuccess(showResult2)
	env.AssertStdoutContains(showResult2, "status: in-progress")
}

// TestMultiplePlansAndImplements tests handling multiple plans and worktrees.
func TestMultiplePlansAndImplements(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	plans := []struct {
		id    string
		title string
	}{
		{"MULTI-001", "First Feature"},
		{"MULTI-002", "Second Feature"},
		{"MULTI-003", "Third Feature"},
	}

	// Save all plans
	for _, p := range plans {
		plan := testenv.ValidPlan(p.id, p.title)
		result := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
		env.AssertSuccess(result)
	}

	// List should show all plans
	listResult := env.RunJig("plan", "list")
	env.AssertSuccess(listResult)
	for _, p := range plans {
		env.AssertStdoutContains(listResult, p.id)
	}

	// Implement only two of them
	env.RunJig("implement", "--no-launch", "MULTI-001")
	env.RunJig("implement", "--no-launch", "MULTI-003")

	// Verify worktrees exist for implemented plans only
	wtDir := env.WorktreeDir()
	entries, _ := os.ReadDir(wtDir)

	found := map[string]bool{}
	for _, e := range entries {
		for _, p := range plans {
			if strings.Contains(e.Name(), p.id) {
				found[p.id] = true
			}
		}
	}

	if !found["MULTI-001"] {
		t.Error("expected worktree for MULTI-001")
	}
	if !found["MULTI-003"] {
		t.Error("expected worktree for MULTI-003")
	}
	if found["MULTI-002"] {
		t.Error("should not have worktree for MULTI-002 (not implemented)")
	}
}
