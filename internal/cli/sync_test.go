package cli

import (
	"testing"

	"github.com/charleslr/jig/internal/state"
)

func TestSyncCmdFlags(t *testing.T) {
	// Test that the sync command has the expected flags
	flags := syncCmd.Flags()

	// Check --all flag
	allFlag := flags.Lookup("all")
	if allFlag == nil {
		t.Error("sync command should have --all flag")
	}
	if allFlag.DefValue != "false" {
		t.Errorf("--all default should be false, got %s", allFlag.DefValue)
	}
}

func TestSyncCmdUsage(t *testing.T) {
	// Test command usage string
	if syncCmd.Use != "sync [ISSUE]" {
		t.Errorf("sync command Use = %q, want %q", syncCmd.Use, "sync [ISSUE]")
	}

	// Test short description
	if syncCmd.Short != "Sync PR info from GitHub" {
		t.Errorf("sync command Short = %q, want %q", syncCmd.Short, "Sync PR info from GitHub")
	}
}

func TestSyncCmdRegistered(t *testing.T) {
	// Verify sync command is registered in root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "sync" {
			found = true
			break
		}
	}
	if !found {
		t.Error("sync command should be registered with root command")
	}
}

func TestSyncAllIssuesEmpty(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// syncAllIssues should handle empty metadata list gracefully
	// This would normally print "No tracked issues found" but shouldn't error
	err := syncAllIssues()
	if err != nil {
		t.Errorf("syncAllIssues() with empty cache should not error, got: %v", err)
	}
}

func TestSyncAllIssuesWithPRAlreadySynced(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Create metadata with PR already synced
	meta := &state.IssueMetadata{
		IssueID:    "NUM-123",
		BranchName: "feature-branch",
		PRNumber:   42,
		PRURL:      "https://github.com/test/repo/pull/42",
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// syncAllIssues should skip already synced issues
	err := syncAllIssues()
	if err != nil {
		t.Errorf("syncAllIssues() should not error, got: %v", err)
	}
}

func TestSyncAllIssuesWithNoBranch(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Create metadata without branch name
	meta := &state.IssueMetadata{
		IssueID: "NUM-456",
		// No BranchName set
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// syncAllIssues should handle issues without branch names
	err := syncAllIssues()
	if err != nil {
		t.Errorf("syncAllIssues() should not error, got: %v", err)
	}
}

func TestSyncIssueMissingMetadata(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// syncIssue should return error for non-existent issue
	err := syncIssue("NONEXISTENT-999")
	if err == nil {
		t.Error("syncIssue() should error for non-existent issue")
	}
}

func TestSyncIssueAlreadyHasPR(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Create metadata with PR already synced
	meta := &state.IssueMetadata{
		IssueID:    "NUM-789",
		BranchName: "feature-branch",
		PRNumber:   55,
		PRURL:      "https://github.com/test/repo/pull/55",
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// syncIssue should return the existing PR number (via SyncPRForIssue which returns early)
	err := syncIssue("NUM-789")
	if err != nil {
		t.Errorf("syncIssue() should not error for already-synced issue, got: %v", err)
	}
}

func TestSyncAllIssuesMixedStates(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Create multiple issues in different states
	issues := []*state.IssueMetadata{
		{IssueID: "SYNCED-1", BranchName: "branch-1", PRNumber: 10, PRURL: "url-1"},   // Already synced
		{IssueID: "SYNCED-2", BranchName: "branch-2", PRNumber: 20, PRURL: "url-2"},   // Already synced
		{IssueID: "NO-BRANCH", BranchName: ""},                                         // No branch
		{IssueID: "HAS-BRANCH", BranchName: "branch-3"},                                // Has branch but no PR (would need gh)
	}

	for _, meta := range issues {
		if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
			t.Fatalf("SaveIssueMetadata() error = %v", err)
		}
	}

	// syncAllIssues should handle all these cases
	err := syncAllIssues()
	if err != nil {
		t.Errorf("syncAllIssues() should not error, got: %v", err)
	}
}

func TestSyncIssueNoBranch(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Create metadata without branch
	meta := &state.IssueMetadata{
		IssueID: "NO-BRANCH-ISSUE",
		// No BranchName
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// syncIssue should error when no branch name
	err := syncIssue("NO-BRANCH-ISSUE")
	if err == nil {
		t.Error("syncIssue() should error when no branch name")
	}
}
