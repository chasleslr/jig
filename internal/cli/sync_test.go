package cli

import (
	"fmt"
	"testing"

	"github.com/charleslr/jig/internal/git"
	gitmock "github.com/charleslr/jig/internal/git/mock"
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

	mockClient := gitmock.NewClient()

	// syncAllIssues should handle empty metadata list gracefully
	err := syncAllIssuesWithClient(mockClient)
	if err != nil {
		t.Errorf("syncAllIssuesWithClient() with empty cache should not error, got: %v", err)
	}
}

func TestSyncAllIssuesWithPRAlreadySynced(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()

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
	err := syncAllIssuesWithClient(mockClient)
	if err != nil {
		t.Errorf("syncAllIssuesWithClient() should not error, got: %v", err)
	}
}

func TestSyncAllIssuesWithNoBranch(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()

	// Create metadata without branch name
	meta := &state.IssueMetadata{
		IssueID: "NUM-456",
		// No BranchName set
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// syncAllIssues should handle issues without branch names
	err := syncAllIssuesWithClient(mockClient)
	if err != nil {
		t.Errorf("syncAllIssuesWithClient() should not error, got: %v", err)
	}
}

func TestSyncIssueMissingMetadata(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()

	// syncIssue should return error for non-existent issue
	err := syncIssueWithClient("NONEXISTENT-999", mockClient)
	if err == nil {
		t.Error("syncIssueWithClient() should error for non-existent issue")
	}
}

func TestSyncIssueAlreadyHasPR(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()

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
	err := syncIssueWithClient("NUM-789", mockClient)
	if err != nil {
		t.Errorf("syncIssueWithClient() should not error for already-synced issue, got: %v", err)
	}
}

func TestSyncAllIssuesMixedStates(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()

	// Add PR for HAS-BRANCH issue
	mockClient.AddPR(&git.PR{
		Number:      99,
		Title:       "Test PR",
		URL:         "https://github.com/test/repo/pull/99",
		HeadRefName: "branch-3",
	})

	// Create multiple issues in different states
	issues := []*state.IssueMetadata{
		{IssueID: "SYNCED-1", BranchName: "branch-1", PRNumber: 10, PRURL: "url-1"}, // Already synced
		{IssueID: "SYNCED-2", BranchName: "branch-2", PRNumber: 20, PRURL: "url-2"}, // Already synced
		{IssueID: "NO-BRANCH", BranchName: ""},                                       // No branch
		{IssueID: "HAS-BRANCH", BranchName: "branch-3"},                              // Has branch with PR
	}

	for _, meta := range issues {
		if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
			t.Fatalf("SaveIssueMetadata() error = %v", err)
		}
	}

	// syncAllIssues should handle all these cases
	err := syncAllIssuesWithClient(mockClient)
	if err != nil {
		t.Errorf("syncAllIssuesWithClient() should not error, got: %v", err)
	}

	// Verify HAS-BRANCH was synced
	meta, _ := state.DefaultCache.GetIssueMetadata("HAS-BRANCH")
	if meta == nil || meta.PRNumber != 99 {
		t.Error("expected HAS-BRANCH to be synced with PR #99")
	}
}

func TestSyncIssueNoBranch(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()

	// Create metadata without branch
	meta := &state.IssueMetadata{
		IssueID: "NO-BRANCH-ISSUE",
		// No BranchName
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// syncIssue should error when no branch name
	err := syncIssueWithClient("NO-BRANCH-ISSUE", mockClient)
	if err == nil {
		t.Error("syncIssueWithClient() should error when no branch name")
	}
}

func TestRunSyncWithClientNotAvailable(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()
	mockClient.IsAvailable = false

	err := runSyncWithClient([]string{"NUM-123"}, mockClient)
	if err == nil {
		t.Error("runSyncWithClient() should error when gh is not available")
	}
}

func TestRunSyncWithClientExplicitIssue(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()
	mockClient.AddPR(&git.PR{
		Number:      42,
		Title:       "Test PR",
		URL:         "https://github.com/test/repo/pull/42",
		HeadRefName: "NUM-123-feature",
	})

	// Create metadata with branch but no PR
	meta := &state.IssueMetadata{
		IssueID:    "NUM-123",
		BranchName: "NUM-123-feature",
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	err := runSyncWithClient([]string{"NUM-123"}, mockClient)
	if err != nil {
		t.Errorf("runSyncWithClient() error = %v", err)
	}

	// Verify PR was synced
	updated, _ := state.DefaultCache.GetIssueMetadata("NUM-123")
	if updated == nil || updated.PRNumber != 42 {
		t.Error("expected PR #42 to be synced")
	}
}

func TestRunSyncWithClientDetectFromBranch(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()
	mockClient.CurrentBranch = "NUM-456-feature"
	mockClient.AddPR(&git.PR{
		Number:      77,
		Title:       "Test PR",
		URL:         "https://github.com/test/repo/pull/77",
		HeadRefName: "NUM-456-feature",
	})

	// Create metadata with branch
	meta := &state.IssueMetadata{
		IssueID:    "NUM-456",
		BranchName: "NUM-456-feature",
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// Run without explicit issue ID - should detect from branch
	err := runSyncWithClient([]string{}, mockClient)
	if err != nil {
		t.Errorf("runSyncWithClient() error = %v", err)
	}

	// Verify PR was synced
	updated, _ := state.DefaultCache.GetIssueMetadata("NUM-456")
	if updated == nil || updated.PRNumber != 77 {
		t.Error("expected PR #77 to be synced")
	}
}

func TestRunSyncWithClientNoIssueDetected(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()
	mockClient.CurrentBranch = "main" // Not an issue branch

	err := runSyncWithClient([]string{}, mockClient)
	if err == nil {
		t.Error("runSyncWithClient() should error when no issue can be detected")
	}
}

func TestSyncIssueWithClientFindsPR(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()
	mockClient.AddPR(&git.PR{
		Number:      88,
		Title:       "New PR",
		URL:         "https://github.com/test/repo/pull/88",
		HeadRefName: "feature-branch",
	})

	// Create metadata with branch but no PR
	meta := &state.IssueMetadata{
		IssueID:    "SYNC-TEST",
		BranchName: "feature-branch",
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	err := syncIssueWithClient("SYNC-TEST", mockClient)
	if err != nil {
		t.Errorf("syncIssueWithClient() error = %v", err)
	}

	// Verify PR was synced
	updated, _ := state.DefaultCache.GetIssueMetadata("SYNC-TEST")
	if updated == nil || updated.PRNumber != 88 {
		t.Error("expected PR #88 to be synced")
	}
}

func TestSyncIssueWithClientNoPRFound(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()
	// No PRs in mock

	// Create metadata with branch but no PR exists
	meta := &state.IssueMetadata{
		IssueID:    "NO-PR-TEST",
		BranchName: "orphan-branch",
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// Should not error, just report no PR found
	err := syncIssueWithClient("NO-PR-TEST", mockClient)
	if err != nil {
		t.Errorf("syncIssueWithClient() should not error when no PR found, got: %v", err)
	}

	// Verify PR number is still 0
	updated, _ := state.DefaultCache.GetIssueMetadata("NO-PR-TEST")
	if updated == nil || updated.PRNumber != 0 {
		t.Error("expected PRNumber to remain 0 when no PR found")
	}
}

func TestSyncIssueWithClientError(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()
	mockClient.GetPRForBranchError = fmt.Errorf("API error")

	// Create metadata with branch
	meta := &state.IssueMetadata{
		IssueID:    "ERROR-TEST",
		BranchName: "error-branch",
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	err := syncIssueWithClient("ERROR-TEST", mockClient)
	if err == nil {
		t.Error("syncIssueWithClient() should error when GetPRForBranch fails")
	}
}
