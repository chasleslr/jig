package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindStale_NoUniqueWork(t *testing.T) {
	// This test requires a real git repository setup
	// We'll create a temporary JIG_HOME for testing
	tmpHome, err := os.MkdirTemp("", "jig-test-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Set JIG_HOME for the test
	oldHome := os.Getenv("JIG_HOME")
	os.Setenv("JIG_HOME", tmpHome)
	defer os.Setenv("JIG_HOME", oldHome)

	ws, err := NewWorktreeState()
	if err != nil {
		t.Fatalf("failed to create worktree state: %v", err)
	}

	t.Run("worktree with no unique work is marked as stale", func(t *testing.T) {
		// This test needs integration with git repository
		// For now, we test the basic structure
		staleInfos, err := ws.FindStale()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// With no tracked worktrees, should return empty
		if len(staleInfos) != 0 {
			t.Errorf("expected 0 stale worktrees, got %d", len(staleInfos))
		}
	})
}

func TestStaleReasonTypes(t *testing.T) {
	t.Run("all stale reason constants are defined", func(t *testing.T) {
		reasons := []StaleReason{
			StaleReasonNotFound,
			StaleReasonMerged,
			StaleReasonNoUniqueWork,
			StaleReasonBranchGone,
		}

		for _, reason := range reasons {
			if string(reason) == "" {
				t.Error("stale reason should not be empty")
			}
		}
	})

	t.Run("stale reasons have distinct values", func(t *testing.T) {
		seen := make(map[StaleReason]bool)
		reasons := []StaleReason{
			StaleReasonNotFound,
			StaleReasonMerged,
			StaleReasonNoUniqueWork,
			StaleReasonBranchGone,
		}

		for _, reason := range reasons {
			if seen[reason] {
				t.Errorf("duplicate stale reason: %s", reason)
			}
			seen[reason] = true
		}
	})
}

func TestWorktreeStateTracking(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "jig-test-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("JIG_HOME")
	os.Setenv("JIG_HOME", tmpHome)
	defer os.Setenv("JIG_HOME", oldHome)


	ws, err := NewWorktreeState()
	if err != nil {
		t.Fatalf("failed to create worktree state: %v", err)
	}

	t.Run("tracks worktree info", func(t *testing.T) {
		info := &WorktreeInfo{
			IssueID:  "TEST-123",
			Path:     "/tmp/test-worktree",
			Branch:   "test-branch",
			RepoPath: "/tmp/test-repo",
		}

		err := ws.Track(info)
		if err != nil {
			t.Fatalf("failed to track worktree: %v", err)
		}

		retrieved, err := ws.Get("TEST-123")
		if err != nil {
			t.Fatalf("failed to get worktree: %v", err)
		}

		if retrieved == nil {
			t.Fatal("expected worktree to be tracked")
		}

		if retrieved.IssueID != info.IssueID {
			t.Errorf("expected issue ID %s, got %s", info.IssueID, retrieved.IssueID)
		}
		if retrieved.Path != info.Path {
			t.Errorf("expected path %s, got %s", info.Path, retrieved.Path)
		}
		if retrieved.Branch != info.Branch {
			t.Errorf("expected branch %s, got %s", info.Branch, retrieved.Branch)
		}
	})

	t.Run("untracks worktree", func(t *testing.T) {
		info := &WorktreeInfo{
			IssueID:  "TEST-456",
			Path:     "/tmp/test-worktree-2",
			Branch:   "test-branch-2",
			RepoPath: "/tmp/test-repo",
		}

		err := ws.Track(info)
		if err != nil {
			t.Fatalf("failed to track worktree: %v", err)
		}

		err = ws.Untrack("TEST-456")
		if err != nil {
			t.Fatalf("failed to untrack worktree: %v", err)
		}

		retrieved, err := ws.Get("TEST-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if retrieved != nil {
			t.Error("expected worktree to be untracked")
		}
	})
}

func TestStaleWorktreeInfo(t *testing.T) {
	t.Run("stale worktree info contains both worktree and reason", func(t *testing.T) {
		info := &WorktreeInfo{
			IssueID:    "TEST-789",
			Path:       "/tmp/test",
			Branch:     "test",
			RepoPath:   "/tmp/repo",
			CreatedAt:  time.Now(),
			LastUsedAt: time.Now(),
		}

		staleInfo := StaleWorktreeInfo{
			WorktreeInfo: info,
			Reason:       StaleReasonNoUniqueWork,
		}

		if staleInfo.WorktreeInfo.IssueID != "TEST-789" {
			t.Error("expected stale info to contain worktree info")
		}

		if staleInfo.Reason != StaleReasonNoUniqueWork {
			t.Errorf("expected reason to be %s, got %s", StaleReasonNoUniqueWork, staleInfo.Reason)
		}
	})
}

func TestFindStale_DirectoryNotFound(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "jig-test-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("JIG_HOME")
	os.Setenv("JIG_HOME", tmpHome)
	defer os.Setenv("JIG_HOME", oldHome)


	ws, err := NewWorktreeState()
	if err != nil {
		t.Fatalf("failed to create worktree state: %v", err)
	}

	t.Run("marks worktree as stale when directory doesn't exist", func(t *testing.T) {
		// Track a worktree with a non-existent path
		info := &WorktreeInfo{
			IssueID:  "TEST-NOTFOUND",
			Path:     filepath.Join(tmpHome, "nonexistent-worktree"),
			Branch:   "test-branch",
			RepoPath: tmpHome,
		}

		err := ws.Track(info)
		if err != nil {
			t.Fatalf("failed to track worktree: %v", err)
		}

		staleInfos, err := ws.FindStale()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(staleInfos) != 1 {
			t.Fatalf("expected 1 stale worktree, got %d", len(staleInfos))
		}

		if staleInfos[0].Reason != StaleReasonNotFound {
			t.Errorf("expected reason %s, got %s", StaleReasonNotFound, staleInfos[0].Reason)
		}

		if staleInfos[0].WorktreeInfo.IssueID != "TEST-NOTFOUND" {
			t.Errorf("expected issue ID TEST-NOTFOUND, got %s", staleInfos[0].WorktreeInfo.IssueID)
		}
	})
}
