package git

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWorktreeMustBeRemovedBeforeBranchDeletion validates the critical path
// in merge.go where a worktree must be removed before its branch can be deleted.
// This is the core reason for creating this test infrastructure.
func TestWorktreeMustBeRemovedBeforeBranchDeletion(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// 1. Create a feature branch with a commit
	repo.CreateBranch("feature-critical")
	repo.Checkout("feature-critical")
	repo.WriteFile("feature.txt", "feature content")
	repo.Commit("feature commit")

	// Go back to main so we can create a worktree for the feature branch
	// (git won't allow creating a worktree for a branch that's already checked out)
	repo.Checkout("main")

	// 2. Create a worktree for the feature branch
	wtPath := repo.CreateWorktreeDir("critical-wt", "feature-critical")
	defer os.RemoveAll(filepath.Dir(wtPath))

	// 3. Merge the feature into main
	repo.MergeBranch("feature-critical")

	repo.InDir(func() {
		// 4. Verify the branch is merged
		merged, err := IsBranchMerged("feature-critical")
		if err != nil {
			t.Fatalf("unexpected error checking merge status: %v", err)
		}
		if !merged {
			t.Fatal("expected branch to be merged")
		}

		// 5. Try to delete the branch - should FAIL because it's checked out in worktree
		err = DeleteBranch("feature-critical", false)
		if err == nil {
			t.Fatal("expected error when deleting branch checked out in worktree")
		}

		// Verify the error message mentions the branch cannot be deleted
		// (git should say something like "cannot delete branch 'feature-critical' checked out at")

		// 6. Verify the branch still exists
		exists, _ := BranchExists("feature-critical")
		if !exists {
			t.Fatal("branch should still exist after failed deletion")
		}

		// 7. Remove the worktree
		err = RemoveWorktree(wtPath)
		if err != nil {
			t.Fatalf("unexpected error removing worktree: %v", err)
		}

		// 8. Now delete should succeed
		err = DeleteBranch("feature-critical", false)
		if err != nil {
			t.Fatalf("unexpected error deleting branch after worktree removal: %v", err)
		}

		// 9. Verify branch no longer exists
		exists, _ = BranchExists("feature-critical")
		if exists {
			t.Error("branch should have been deleted")
		}
	})
}

// TestWorktreeCleanupFlow simulates the jig clean workflow
func TestWorktreeCleanupFlow(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Create feature branches with commits BEFORE creating worktrees
	repo.CreateBranch("feature-a")
	repo.Checkout("feature-a")
	repo.WriteFile("a.txt", "a content")
	repo.Commit("feature-a commit")

	repo.CreateBranch("feature-b")
	repo.Checkout("feature-b")
	repo.WriteFile("b.txt", "b content")
	repo.Commit("feature-b commit")

	repo.CreateBranch("feature-c")
	repo.Checkout("feature-c")
	repo.WriteFile("c.txt", "c content")
	repo.Commit("feature-c commit")

	// Go back to main so we can create worktrees
	repo.Checkout("main")

	// Create worktrees for each branch
	wtPaths := make(map[string]string)
	for _, name := range []string{"feature-a", "feature-b", "feature-c"} {
		wtPath := repo.CreateWorktreeDir(name+"-wt", name)
		wtPaths[name] = wtPath
		defer os.RemoveAll(filepath.Dir(wtPath))
	}

	// Merge feature-a and feature-b into main (but not feature-c)
	repo.MergeBranch("feature-a")
	repo.MergeBranch("feature-b")

	repo.InDir(func() {
		// List all worktrees
		worktrees, err := ListWorktrees()
		if err != nil {
			t.Fatalf("unexpected error listing worktrees: %v", err)
		}

		// Should have 4 worktrees (main + 3 features)
		if len(worktrees) != 4 {
			t.Errorf("expected 4 worktrees, got %d", len(worktrees))
		}

		// Check which branches are merged
		for _, name := range []string{"feature-a", "feature-b"} {
			merged, err := IsBranchMerged(name)
			if err != nil {
				t.Fatalf("unexpected error checking %s merge status: %v", name, err)
			}
			if !merged {
				t.Errorf("expected %s to be merged", name)
			}
		}

		merged, err := IsBranchMerged("feature-c")
		if err != nil {
			t.Fatalf("unexpected error checking feature-c merge status: %v", err)
		}
		if merged {
			t.Error("expected feature-c to NOT be merged")
		}

		// Clean up merged branches (remove worktree first, then branch)
		for _, name := range []string{"feature-a", "feature-b"} {
			wtPath := wtPaths[name]

			err := RemoveWorktree(wtPath)
			if err != nil {
				t.Fatalf("unexpected error removing %s worktree: %v", name, err)
			}

			err = DeleteBranch(name, false)
			if err != nil {
				t.Fatalf("unexpected error deleting %s branch: %v", name, err)
			}
		}

		// Verify only main and feature-c remain
		branches, err := ListBranches()
		if err != nil {
			t.Fatalf("unexpected error listing branches: %v", err)
		}

		expected := map[string]bool{"main": true, "feature-c": true}
		if len(branches) != 2 {
			t.Errorf("expected 2 branches, got %d: %v", len(branches), branches)
		}
		for _, b := range branches {
			if !expected[b] {
				t.Errorf("unexpected branch: %s", b)
			}
		}

		// Verify worktrees
		worktrees, _ = ListWorktrees()
		if len(worktrees) != 2 {
			t.Errorf("expected 2 worktrees (main + feature-c), got %d", len(worktrees))
		}
	})
}

// TestBranchMergedAfterMerge verifies merge detection works correctly
func TestBranchMergedAfterMerge(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("detects merge with fast-forward", func(t *testing.T) {
		repo.CreateBranch("ff-branch")
		repo.Checkout("ff-branch")
		repo.WriteFile("ff.txt", "ff content")
		repo.Commit("ff commit")
		repo.Checkout("main")
		repo.MergeBranch("ff-branch")

		repo.InDir(func() {
			merged, err := IsBranchMerged("ff-branch")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !merged {
				t.Error("expected ff-branch to be merged")
			}
		})
	})

	t.Run("detects merge with merge commit", func(t *testing.T) {
		// Create divergent history
		repo.WriteFile("main.txt", "main content")
		repo.Commit("main commit")

		repo.CreateBranch("merge-branch")
		repo.Checkout("merge-branch")
		repo.WriteFile("merge.txt", "merge content")
		repo.Commit("merge branch commit")

		repo.Checkout("main")
		repo.WriteFile("main2.txt", "main2 content")
		repo.Commit("main commit 2")

		// This will create a merge commit
		repo.Git("merge", "merge-branch", "--no-edit")

		repo.InDir(func() {
			merged, err := IsBranchMerged("merge-branch")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !merged {
				t.Error("expected merge-branch to be merged")
			}
		})
	})
}

// TestWorktreeBranchTracking verifies worktree-branch relationships
func TestWorktreeBranchTracking(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.CreateBranch("tracked-branch")
	wtPath := repo.CreateWorktreeDir("tracked-wt", "tracked-branch")
	defer os.RemoveAll(filepath.Dir(wtPath))

	repo.InDir(func() {
		// Find worktree by branch
		wt, err := GetWorktree("tracked-branch")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wt == nil {
			t.Fatal("expected to find worktree for tracked-branch")
		}
		if wt.Path != wtPath {
			t.Errorf("expected path %s, got %s", wtPath, wt.Path)
		}

		// Find worktree by path
		wtByPath, err := GetWorktreeByPath(wtPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wtByPath == nil {
			t.Fatal("expected to find worktree by path")
		}
		if wtByPath.Branch != "tracked-branch" {
			t.Errorf("expected branch tracked-branch, got %s", wtByPath.Branch)
		}

		// Verify commit hash is populated
		if wt.Commit == "" {
			t.Error("expected commit hash to be populated")
		}
	})
}

// TestConcurrentWorktreeOperations tests creating multiple worktrees
func TestConcurrentWorktreeOperations(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Create multiple branches
	branches := []string{"wt-1", "wt-2", "wt-3", "wt-4", "wt-5"}
	for _, name := range branches {
		repo.CreateBranch(name)
	}

	// Create worktrees for each
	var wtPaths []string
	for _, name := range branches {
		wtPath := repo.CreateWorktreeDir(name+"-dir", name)
		wtPaths = append(wtPaths, wtPath)
		defer os.RemoveAll(filepath.Dir(wtPath))
	}

	repo.InDir(func() {
		// Verify all worktrees exist
		worktrees, err := ListWorktrees()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have main + 5 feature worktrees
		if len(worktrees) != 6 {
			t.Errorf("expected 6 worktrees, got %d", len(worktrees))
		}

		// Verify each branch has a worktree
		for _, name := range branches {
			wt, err := GetWorktree(name)
			if err != nil {
				t.Fatalf("unexpected error getting worktree for %s: %v", name, err)
			}
			if wt == nil {
				t.Errorf("expected worktree for %s", name)
			}
		}

		// Clean up all worktrees and branches
		for i, name := range branches {
			err := RemoveWorktree(wtPaths[i])
			if err != nil {
				t.Fatalf("unexpected error removing worktree %s: %v", name, err)
			}

			err = DeleteBranch(name, true)
			if err != nil {
				t.Fatalf("unexpected error deleting branch %s: %v", name, err)
			}
		}

		// Verify only main remains
		worktrees, _ = ListWorktrees()
		if len(worktrees) != 1 {
			t.Errorf("expected 1 worktree (main), got %d", len(worktrees))
		}

		branches, _ := ListBranches()
		if len(branches) != 1 || branches[0] != "main" {
			t.Errorf("expected only main branch, got %v", branches)
		}
	})
}
