package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/charleslr/jig/internal/git"
	gitmock "github.com/charleslr/jig/internal/git/mock"
	"github.com/charleslr/jig/internal/state"
)

// setupTestCache creates a temporary cache directory and initializes state.DefaultCache
func setupTestCache(t *testing.T) func() {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "jig-cli-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create cache directories
	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	// Save original DefaultCache and set up test cache
	originalCache := state.DefaultCache
	state.DefaultCache = state.NewCacheWithDir(tmpDir)

	cleanup := func() {
		state.DefaultCache = originalCache
		os.RemoveAll(tmpDir)
	}

	return cleanup
}

func TestPrCmdFlags(t *testing.T) {
	// Test that the pr command has the expected flags
	flags := prCmd.Flags()

	// Check --draft flag
	draftFlag := flags.Lookup("draft")
	if draftFlag == nil {
		t.Error("pr command should have --draft flag")
	}
	if draftFlag.DefValue != "true" {
		t.Errorf("--draft default should be true, got %s", draftFlag.DefValue)
	}
	if draftFlag.Shorthand != "d" {
		t.Errorf("--draft shorthand should be 'd', got %q", draftFlag.Shorthand)
	}

	// Check --title flag
	titleFlag := flags.Lookup("title")
	if titleFlag == nil {
		t.Error("pr command should have --title flag")
	}
	if titleFlag.Shorthand != "t" {
		t.Errorf("--title shorthand should be 't', got %q", titleFlag.Shorthand)
	}

	// Check --body flag
	bodyFlag := flags.Lookup("body")
	if bodyFlag == nil {
		t.Error("pr command should have --body flag")
	}
	if bodyFlag.Shorthand != "b" {
		t.Errorf("--body shorthand should be 'b', got %q", bodyFlag.Shorthand)
	}

	// Check --base flag
	baseFlag := flags.Lookup("base")
	if baseFlag == nil {
		t.Error("pr command should have --base flag")
	}
}

func TestPrCmdUsage(t *testing.T) {
	// Test command usage string
	if prCmd.Use != "pr [ISSUE]" {
		t.Errorf("pr command Use = %q, want %q", prCmd.Use, "pr [ISSUE]")
	}

	// Test short description
	if prCmd.Short != "Create a PR and record it in metadata" {
		t.Errorf("pr command Short = %q, want %q", prCmd.Short, "Create a PR and record it in metadata")
	}
}

func TestPrCmdRegistered(t *testing.T) {
	// Verify pr command is registered in root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "pr" {
			found = true
			break
		}
	}
	if !found {
		t.Error("pr command should be registered with root command")
	}
}

func TestUpdatePRMetadataCreatesNew(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Call updatePRMetadata for a new issue
	err := updatePRMetadata("NEW-123", "new-branch", 42, "https://github.com/test/repo/pull/42")
	if err != nil {
		t.Fatalf("updatePRMetadata() error = %v", err)
	}

	// Verify metadata was created
	meta, err := state.DefaultCache.GetIssueMetadata("NEW-123")
	if err != nil {
		t.Fatalf("GetIssueMetadata() error = %v", err)
	}
	if meta == nil {
		t.Fatal("expected metadata to be created")
	}
	if meta.IssueID != "NEW-123" {
		t.Errorf("IssueID = %q, want %q", meta.IssueID, "NEW-123")
	}
	if meta.BranchName != "new-branch" {
		t.Errorf("BranchName = %q, want %q", meta.BranchName, "new-branch")
	}
	if meta.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want %d", meta.PRNumber, 42)
	}
	if meta.PRURL != "https://github.com/test/repo/pull/42" {
		t.Errorf("PRURL = %q, want %q", meta.PRURL, "https://github.com/test/repo/pull/42")
	}
}

func TestUpdatePRMetadataUpdatesExisting(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Create existing metadata
	existingMeta := &state.IssueMetadata{
		IssueID:      "EXISTING-456",
		BranchName:   "old-branch",
		WorktreePath: "/path/to/worktree",
	}
	if err := state.DefaultCache.SaveIssueMetadata(existingMeta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// Call updatePRMetadata to update it
	err := updatePRMetadata("EXISTING-456", "new-branch", 99, "https://github.com/test/repo/pull/99")
	if err != nil {
		t.Fatalf("updatePRMetadata() error = %v", err)
	}

	// Verify metadata was updated
	meta, err := state.DefaultCache.GetIssueMetadata("EXISTING-456")
	if err != nil {
		t.Fatalf("GetIssueMetadata() error = %v", err)
	}
	if meta == nil {
		t.Fatal("expected metadata to exist")
	}
	// Check new values
	if meta.BranchName != "new-branch" {
		t.Errorf("BranchName = %q, want %q", meta.BranchName, "new-branch")
	}
	if meta.PRNumber != 99 {
		t.Errorf("PRNumber = %d, want %d", meta.PRNumber, 99)
	}
	if meta.PRURL != "https://github.com/test/repo/pull/99" {
		t.Errorf("PRURL = %q, want %q", meta.PRURL, "https://github.com/test/repo/pull/99")
	}
	// Check preserved values
	if meta.WorktreePath != "/path/to/worktree" {
		t.Errorf("WorktreePath = %q, want %q (should be preserved)", meta.WorktreePath, "/path/to/worktree")
	}
}

func TestRunPRWithClientNotAvailable(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()
	mockClient.IsAvailable = false

	err := runPRWithClient([]string{}, mockClient)
	if err == nil {
		t.Error("runPRWithClient() should error when gh is not available")
	}
}

func TestRunPRWithClientExistingPR(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	mockClient := gitmock.NewClient()
	mockClient.CurrentBranch = "NUM-123-feature"
	mockClient.AddPR(&git.PR{
		Number:      42,
		Title:       "Existing PR",
		URL:         "https://github.com/test/repo/pull/42",
		HeadRefName: "NUM-123-feature",
	})

	err := runPRWithClient([]string{}, mockClient)
	if err != nil {
		t.Errorf("runPRWithClient() error = %v", err)
	}

	// Verify metadata was updated
	meta, _ := state.DefaultCache.GetIssueMetadata("NUM-123")
	if meta == nil {
		t.Fatal("expected metadata to be created for existing PR")
	}
	if meta.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", meta.PRNumber)
	}
}

func TestRunPRWithClientCreatesPR(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Reset global flags to defaults
	prTitle = ""
	prBody = ""
	prBase = ""
	prDraft = true

	mockClient := gitmock.NewClient()
	mockClient.CurrentBranch = "NUM-456-new-feature"

	err := runPRWithClient([]string{}, mockClient)
	if err != nil {
		t.Errorf("runPRWithClient() error = %v", err)
	}

	// Verify CreatePR was called
	if len(mockClient.CreatePRCalls) != 1 {
		t.Fatalf("expected 1 CreatePR call, got %d", len(mockClient.CreatePRCalls))
	}

	call := mockClient.CreatePRCalls[0]
	if call.Draft != true {
		t.Errorf("CreatePR Draft = %v, want true", call.Draft)
	}
	if call.BaseBranch != "main" {
		t.Errorf("CreatePR BaseBranch = %q, want %q", call.BaseBranch, "main")
	}

	// Verify metadata was created
	meta, _ := state.DefaultCache.GetIssueMetadata("NUM-456")
	if meta == nil {
		t.Fatal("expected metadata to be created")
	}
	if meta.PRNumber != 1 { // Mock assigns PR numbers starting from 1
		t.Errorf("PRNumber = %d, want 1", meta.PRNumber)
	}
}

func TestRunPRWithClientExplicitIssueID(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Reset global flags to defaults
	prTitle = ""
	prBody = ""
	prBase = ""
	prDraft = true

	mockClient := gitmock.NewClient()
	mockClient.CurrentBranch = "some-other-branch"

	err := runPRWithClient([]string{"EXPLICIT-789"}, mockClient)
	if err != nil {
		t.Errorf("runPRWithClient() error = %v", err)
	}

	// Verify metadata was created with explicit issue ID
	meta, _ := state.DefaultCache.GetIssueMetadata("EXPLICIT-789")
	if meta == nil {
		t.Fatal("expected metadata to be created with explicit issue ID")
	}
}

func TestRunPRWithClientCreatePRError(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Reset global flags
	prTitle = ""
	prBody = ""
	prBase = ""
	prDraft = true

	mockClient := gitmock.NewClient()
	mockClient.CurrentBranch = "NUM-999-feature"
	mockClient.CreatePRError = fmt.Errorf("network error")

	err := runPRWithClient([]string{}, mockClient)
	if err == nil {
		t.Error("runPRWithClient() should error when CreatePR fails")
	}
}
