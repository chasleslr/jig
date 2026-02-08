package mock

import (
	"fmt"
	"testing"

	"github.com/charleslr/jig/internal/git"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("NewClient() should not return nil")
	}

	// Check defaults
	if !client.IsAvailable {
		t.Error("IsAvailable should default to true")
	}
	if client.CurrentBranch != "main" {
		t.Errorf("CurrentBranch should default to 'main', got %q", client.CurrentBranch)
	}
	if client.CIStatus != "success" {
		t.Errorf("CIStatus should default to 'success', got %q", client.CIStatus)
	}
	if client.PRsByBranch == nil {
		t.Error("PRsByBranch should be initialized")
	}
	if client.PRsByNumber == nil {
		t.Error("PRsByNumber should be initialized")
	}
	if client.Comments == nil {
		t.Error("Comments should be initialized")
	}
	if client.ReviewThreads == nil {
		t.Error("ReviewThreads should be initialized")
	}
}

func TestClientImplementsInterface(t *testing.T) {
	// Compile-time check that Client implements git.Client
	var _ git.Client = (*Client)(nil)
}

func TestAvailable(t *testing.T) {
	tests := []struct {
		name        string
		isAvailable bool
		err         error
		want        bool
	}{
		{"available", true, nil, true},
		{"not available", false, nil, false},
		{"error makes unavailable", true, fmt.Errorf("error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient()
			client.IsAvailable = tt.isAvailable
			client.AvailableError = tt.err

			if got := client.Available(); got != tt.want {
				t.Errorf("Available() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	client := NewClient()
	client.CurrentBranch = "feature-branch"

	branch, err := client.GetCurrentBranch()
	if err != nil {
		t.Errorf("GetCurrentBranch() error = %v", err)
	}
	if branch != "feature-branch" {
		t.Errorf("GetCurrentBranch() = %q, want %q", branch, "feature-branch")
	}
}

func TestGetCurrentBranchError(t *testing.T) {
	client := NewClient()
	client.GetCurrentBranchError = fmt.Errorf("git error")

	_, err := client.GetCurrentBranch()
	if err == nil {
		t.Error("GetCurrentBranch() should error when GetCurrentBranchError is set")
	}
}

func TestGetPR(t *testing.T) {
	client := NewClient()
	client.CurrentBranch = "feature"

	expectedPR := &git.PR{Number: 42, Title: "Test PR"}
	client.AddPR(&git.PR{Number: 42, Title: "Test PR", HeadRefName: "feature"})

	pr, err := client.GetPR()
	if err != nil {
		t.Errorf("GetPR() error = %v", err)
	}
	if pr.Number != expectedPR.Number {
		t.Errorf("GetPR() Number = %d, want %d", pr.Number, expectedPR.Number)
	}
}

func TestGetPRNotFound(t *testing.T) {
	client := NewClient()
	client.CurrentBranch = "no-pr-branch"

	_, err := client.GetPR()
	if err == nil {
		t.Error("GetPR() should error when no PR exists for current branch")
	}
}

func TestGetPRError(t *testing.T) {
	client := NewClient()
	client.GetPRError = fmt.Errorf("API error")

	_, err := client.GetPR()
	if err == nil {
		t.Error("GetPR() should error when GetPRError is set")
	}
}

func TestGetPRByNumber(t *testing.T) {
	client := NewClient()
	client.AddPR(&git.PR{Number: 123, Title: "PR 123"})

	pr, err := client.GetPRByNumber(123)
	if err != nil {
		t.Errorf("GetPRByNumber() error = %v", err)
	}
	if pr.Number != 123 {
		t.Errorf("GetPRByNumber() Number = %d, want 123", pr.Number)
	}
}

func TestGetPRByNumberNotFound(t *testing.T) {
	client := NewClient()

	_, err := client.GetPRByNumber(999)
	if err == nil {
		t.Error("GetPRByNumber() should error when PR not found")
	}
}

func TestGetPRByNumberError(t *testing.T) {
	client := NewClient()
	client.GetPRByNumberError = fmt.Errorf("API error")

	_, err := client.GetPRByNumber(1)
	if err == nil {
		t.Error("GetPRByNumber() should error when GetPRByNumberError is set")
	}
}

func TestGetPRForBranch(t *testing.T) {
	client := NewClient()
	client.AddPR(&git.PR{Number: 55, Title: "Branch PR", HeadRefName: "feature-x"})

	pr, err := client.GetPRForBranch("feature-x")
	if err != nil {
		t.Errorf("GetPRForBranch() error = %v", err)
	}
	if pr == nil {
		t.Fatal("GetPRForBranch() returned nil")
	}
	if pr.Number != 55 {
		t.Errorf("GetPRForBranch() Number = %d, want 55", pr.Number)
	}
}

func TestGetPRForBranchNotFound(t *testing.T) {
	client := NewClient()

	pr, err := client.GetPRForBranch("nonexistent")
	if err != nil {
		t.Errorf("GetPRForBranch() should not error for missing branch, got %v", err)
	}
	if pr != nil {
		t.Error("GetPRForBranch() should return nil for missing branch")
	}
}

func TestGetPRForBranchError(t *testing.T) {
	client := NewClient()
	client.GetPRForBranchError = fmt.Errorf("API error")

	_, err := client.GetPRForBranch("any")
	if err == nil {
		t.Error("GetPRForBranch() should error when GetPRForBranchError is set")
	}
}

func TestCreatePR(t *testing.T) {
	client := NewClient()
	client.CurrentBranch = "new-feature"

	pr, err := client.CreatePR("Title", "Body", "main", true)
	if err != nil {
		t.Errorf("CreatePR() error = %v", err)
	}

	// Check returned PR
	if pr.Number != 1 {
		t.Errorf("CreatePR() Number = %d, want 1", pr.Number)
	}
	if pr.Title != "Title" {
		t.Errorf("CreatePR() Title = %q, want %q", pr.Title, "Title")
	}
	if pr.Body != "Body" {
		t.Errorf("CreatePR() Body = %q, want %q", pr.Body, "Body")
	}
	if pr.HeadRefName != "new-feature" {
		t.Errorf("CreatePR() HeadRefName = %q, want %q", pr.HeadRefName, "new-feature")
	}
	if pr.BaseRefName != "main" {
		t.Errorf("CreatePR() BaseRefName = %q, want %q", pr.BaseRefName, "main")
	}
	if !pr.IsDraft {
		t.Error("CreatePR() IsDraft should be true")
	}

	// Check call was recorded
	if len(client.CreatePRCalls) != 1 {
		t.Fatalf("CreatePRCalls should have 1 entry, got %d", len(client.CreatePRCalls))
	}
	call := client.CreatePRCalls[0]
	if call.Title != "Title" || call.Body != "Body" || call.BaseBranch != "main" || !call.Draft {
		t.Errorf("CreatePRCalls recorded incorrectly: %+v", call)
	}

	// Check PR was stored
	storedPR, _ := client.GetPRByNumber(1)
	if storedPR == nil {
		t.Error("CreatePR should store PR in PRsByNumber")
	}

	branchPR, _ := client.GetPRForBranch("new-feature")
	if branchPR == nil {
		t.Error("CreatePR should store PR in PRsByBranch")
	}
}

func TestCreatePRError(t *testing.T) {
	client := NewClient()
	client.CreatePRError = fmt.Errorf("create failed")

	_, err := client.CreatePR("Title", "Body", "main", false)
	if err == nil {
		t.Error("CreatePR() should error when CreatePRError is set")
	}

	// Call should still be recorded even on error
	if len(client.CreatePRCalls) != 1 {
		t.Error("CreatePR should record call even when error occurs")
	}
}

func TestCreatePRIncrementingNumbers(t *testing.T) {
	client := NewClient()

	pr1, _ := client.CreatePR("PR 1", "", "main", false)
	pr2, _ := client.CreatePR("PR 2", "", "main", false)

	if pr1.Number != 1 {
		t.Errorf("First PR Number = %d, want 1", pr1.Number)
	}
	if pr2.Number != 2 {
		t.Errorf("Second PR Number = %d, want 2", pr2.Number)
	}
}

func TestMergePR(t *testing.T) {
	client := NewClient()
	client.AddPR(&git.PR{Number: 10, State: "OPEN"})

	err := client.MergePR(10, "squash", true)
	if err != nil {
		t.Errorf("MergePR() error = %v", err)
	}

	// Check call was recorded
	if len(client.MergePRCalls) != 1 {
		t.Fatalf("MergePRCalls should have 1 entry, got %d", len(client.MergePRCalls))
	}
	call := client.MergePRCalls[0]
	if call.Number != 10 || call.Method != "squash" || !call.DeleteAfter {
		t.Errorf("MergePRCalls recorded incorrectly: %+v", call)
	}

	// Check PR state was updated
	pr, _ := client.GetPRByNumber(10)
	if pr.State != "MERGED" {
		t.Errorf("PR State = %q, want %q", pr.State, "MERGED")
	}
}

func TestMergePRError(t *testing.T) {
	client := NewClient()
	client.MergePRError = fmt.Errorf("merge failed")

	err := client.MergePR(1, "merge", false)
	if err == nil {
		t.Error("MergePR() should error when MergePRError is set")
	}

	// Call should still be recorded
	if len(client.MergePRCalls) != 1 {
		t.Error("MergePR should record call even when error occurs")
	}
}

func TestGetPRComments(t *testing.T) {
	client := NewClient()
	client.AddComment(42, git.PRComment{ID: 1, Body: "Comment 1"})
	client.AddComment(42, git.PRComment{ID: 2, Body: "Comment 2"})

	comments, err := client.GetPRComments(42)
	if err != nil {
		t.Errorf("GetPRComments() error = %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("GetPRComments() returned %d comments, want 2", len(comments))
	}
}

func TestGetPRCommentsEmpty(t *testing.T) {
	client := NewClient()

	comments, err := client.GetPRComments(999)
	if err != nil {
		t.Errorf("GetPRComments() error = %v", err)
	}
	if comments != nil && len(comments) != 0 {
		t.Errorf("GetPRComments() should return empty for unknown PR")
	}
}

func TestGetPRCommentsError(t *testing.T) {
	client := NewClient()
	client.GetPRCommentsError = fmt.Errorf("API error")

	_, err := client.GetPRComments(1)
	if err == nil {
		t.Error("GetPRComments() should error when GetPRCommentsError is set")
	}
}

func TestGetPRReviewThreads(t *testing.T) {
	client := NewClient()
	client.AddReviewThread(42, git.PRComment{ID: 1, Body: "Thread 1"})

	threads, err := client.GetPRReviewThreads(42)
	if err != nil {
		t.Errorf("GetPRReviewThreads() error = %v", err)
	}
	if len(threads) != 1 {
		t.Errorf("GetPRReviewThreads() returned %d threads, want 1", len(threads))
	}
}

func TestGetPRReviewThreadsError(t *testing.T) {
	client := NewClient()
	client.GetPRReviewThreadsError = fmt.Errorf("API error")

	_, err := client.GetPRReviewThreads(1)
	if err == nil {
		t.Error("GetPRReviewThreads() should error when GetPRReviewThreadsError is set")
	}
}

func TestGetCIStatus(t *testing.T) {
	client := NewClient()
	client.CIStatus = "pending"

	status, err := client.GetCIStatus()
	if err != nil {
		t.Errorf("GetCIStatus() error = %v", err)
	}
	if status != "pending" {
		t.Errorf("GetCIStatus() = %q, want %q", status, "pending")
	}
}

func TestGetCIStatusError(t *testing.T) {
	client := NewClient()
	client.GetCIStatusError = fmt.Errorf("API error")

	_, err := client.GetCIStatus()
	if err == nil {
		t.Error("GetCIStatus() should error when GetCIStatusError is set")
	}
}

func TestAddPR(t *testing.T) {
	client := NewClient()

	pr := &git.PR{Number: 99, HeadRefName: "test-branch"}
	client.AddPR(pr)

	// Should be findable by number
	byNumber, _ := client.GetPRByNumber(99)
	if byNumber == nil {
		t.Error("AddPR should make PR findable by number")
	}

	// Should be findable by branch
	byBranch, _ := client.GetPRForBranch("test-branch")
	if byBranch == nil {
		t.Error("AddPR should make PR findable by branch")
	}
}

func TestAddPRNoHeadRefName(t *testing.T) {
	client := NewClient()

	pr := &git.PR{Number: 100, HeadRefName: ""}
	client.AddPR(pr)

	// Should be findable by number
	byNumber, _ := client.GetPRByNumber(100)
	if byNumber == nil {
		t.Error("AddPR should make PR findable by number even without HeadRefName")
	}

	// Should NOT be added to PRsByBranch since HeadRefName is empty
	if len(client.PRsByBranch) != 0 {
		t.Error("AddPR should not add to PRsByBranch when HeadRefName is empty")
	}
}

func TestReset(t *testing.T) {
	client := NewClient()

	// Add some state
	client.AddPR(&git.PR{Number: 1, HeadRefName: "branch"})
	client.AddComment(1, git.PRComment{ID: 1})
	client.AddReviewThread(1, git.PRComment{ID: 2})
	client.CreatePR("Title", "Body", "main", false)
	client.MergePR(1, "squash", true)
	client.GetCurrentBranchError = fmt.Errorf("error")
	client.CreatePRError = fmt.Errorf("error")

	// Reset
	client.Reset()

	// Verify all state is cleared
	if len(client.PRsByBranch) != 0 {
		t.Error("Reset should clear PRsByBranch")
	}
	if len(client.PRsByNumber) != 0 {
		t.Error("Reset should clear PRsByNumber")
	}
	if len(client.Comments) != 0 {
		t.Error("Reset should clear Comments")
	}
	if len(client.ReviewThreads) != 0 {
		t.Error("Reset should clear ReviewThreads")
	}
	if len(client.CreatePRCalls) != 0 {
		t.Error("Reset should clear CreatePRCalls")
	}
	if len(client.MergePRCalls) != 0 {
		t.Error("Reset should clear MergePRCalls")
	}
	if client.GetCurrentBranchError != nil {
		t.Error("Reset should clear GetCurrentBranchError")
	}
	if client.CreatePRError != nil {
		t.Error("Reset should clear CreatePRError")
	}
}
