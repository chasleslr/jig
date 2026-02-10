package functional

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charleslr/jig/tests/functional/testenv"
)

func TestVerify_RequiresWorktreeContext(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Run verify from the repo root (not a worktree)
	result := env.RunJig("verify", "--no-launch")

	// Should fail because we're not in a worktree
	env.AssertFailure(result)
	env.AssertOutputContains(result, "not in a jig worktree")
}

func TestVerify_RequiresPlanFile(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Create a .jig directory manually without a plan
	jigDir := filepath.Join(env.RepoDir, ".jig")
	if err := os.MkdirAll(jigDir, 0755); err != nil {
		t.Fatalf("failed to create .jig dir: %v", err)
	}

	// Create issue.json
	issueJSON := `{"issue_id": "TEST-001"}`
	if err := os.WriteFile(filepath.Join(jigDir, "issue.json"), []byte(issueJSON), 0644); err != nil {
		t.Fatalf("failed to write issue.json: %v", err)
	}

	// Run verify - should fail because plan.md is missing
	result := env.RunJig("verify", "--no-launch")
	env.AssertFailure(result)
	env.AssertOutputContains(result, "no plan found")
}

func TestVerify_WithValidContext(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// First save a plan
	plan := testenv.ValidPlan("VERIFY-001", "Verification Test")
	saveResult := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
	env.AssertSuccess(saveResult)

	// Run implement to set up the worktree
	implResult := env.RunJig("implement", "--no-launch", "VERIFY-001")
	env.AssertSuccess(implResult)

	// Find the worktree directory
	wtDir := env.WorktreeDir()
	entries, err := os.ReadDir(wtDir)
	if err != nil {
		t.Fatalf("failed to read worktree dir: %v", err)
	}

	var worktreePath string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "VERIFY-001") {
			worktreePath = filepath.Join(wtDir, entry.Name())
			break
		}
	}

	if worktreePath == "" {
		t.Fatal("could not find worktree for VERIFY-001")
	}

	// Run verify from the worktree directory with --no-launch
	result := env.RunJigInDir(worktreePath, "verify", "--no-launch")
	env.AssertSuccess(result)

	// Should indicate it's ready for verification
	env.AssertOutputContains(result, "Ready for verification")
	env.AssertOutputContains(result, "VERIFY-001")
}

func TestVerify_AcceptsIssueArg(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// First save a plan
	plan := testenv.ValidPlan("VERIFY-ARG-001", "Verification Arg Test")
	saveResult := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
	env.AssertSuccess(saveResult)

	// Run implement to set up the worktree
	implResult := env.RunJig("implement", "--no-launch", "VERIFY-ARG-001")
	env.AssertSuccess(implResult)

	// Find the worktree directory
	wtDir := env.WorktreeDir()
	entries, err := os.ReadDir(wtDir)
	if err != nil {
		t.Fatalf("failed to read worktree dir: %v", err)
	}

	var worktreePath string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "VERIFY-ARG-001") {
			worktreePath = filepath.Join(wtDir, entry.Name())
			break
		}
	}

	if worktreePath == "" {
		t.Fatal("could not find worktree for VERIFY-ARG-001")
	}

	// Run verify from the worktree with an explicit issue arg
	result := env.RunJigInDir(worktreePath, "verify", "--no-launch", "VERIFY-ARG-001")
	env.AssertSuccess(result)

	// Output should contain the issue ID
	env.AssertOutputContains(result, "VERIFY-ARG-001")
}

func TestVerify_OutputsInstructions(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// First save a plan
	plan := testenv.ValidPlan("VERIFY-OUT-001", "Verification Output Test")
	saveResult := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
	env.AssertSuccess(saveResult)

	// Run implement to set up the worktree
	implResult := env.RunJig("implement", "--no-launch", "VERIFY-OUT-001")
	env.AssertSuccess(implResult)

	// Find the worktree directory
	wtDir := env.WorktreeDir()
	entries, err := os.ReadDir(wtDir)
	if err != nil {
		t.Fatalf("failed to read worktree dir: %v", err)
	}

	var worktreePath string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "VERIFY-OUT-001") {
			worktreePath = filepath.Join(wtDir, entry.Name())
			break
		}
	}

	if worktreePath == "" {
		t.Fatal("could not find worktree for VERIFY-OUT-001")
	}

	// Run verify with --no-launch
	result := env.RunJigInDir(worktreePath, "verify", "--no-launch")
	env.AssertSuccess(result)

	// Output should contain verify instructions
	env.AssertOutputContains(result, "Plan:")
	env.AssertOutputContains(result, "To verify:")
}
