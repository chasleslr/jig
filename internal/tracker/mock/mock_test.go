package mock

import (
	"context"
	"testing"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/tracker"
)

func TestClient_SyncPlan_CreateNewIssue(t *testing.T) {
	client := NewClient()
	p := &plan.Plan{
		Title:            "Test Plan",
		ProblemStatement: "The problem to solve",
		ProposedSolution: "The proposed solution",
		Status:           plan.StatusDraft,
		Author:           "testuser",
	}

	err := client.SyncPlan(context.Background(), p)
	if err != nil {
		t.Fatalf("SyncPlan() error = %v", err)
	}

	// Plan should have a new ID assigned
	if p.ID == "" {
		t.Error("plan.ID should be assigned after sync")
	}

	// Verify the issue was created
	issue := client.GetIssueByIdentifier(p.ID)
	if issue == nil {
		t.Fatal("issue should be created in mock tracker")
	}
	if issue.Title != p.Title {
		t.Errorf("issue.Title = %q, want %q", issue.Title, p.Title)
	}
	if issue.Description == "" {
		t.Error("issue.Description should not be empty")
	}
	if !containsString(issue.Description, p.ProblemStatement) {
		t.Error("issue.Description should contain problem statement")
	}
	if !containsString(issue.Description, p.ProposedSolution) {
		t.Error("issue.Description should contain proposed solution")
	}
}

func TestClient_SyncPlan_UpdateExistingIssue(t *testing.T) {
	client := NewClient()

	// First create an issue
	initialIssue, err := client.CreateIssue(context.Background(), &tracker.Issue{
		Title:       "Initial Title",
		Description: "Initial description",
	})
	if err != nil {
		t.Fatalf("CreateIssue() error = %v", err)
	}

	// Now sync a plan with the same ID
	p := &plan.Plan{
		ID:               initialIssue.Identifier,
		Title:            "Updated Title",
		ProblemStatement: "Updated problem",
		ProposedSolution: "Updated solution",
		Status:           plan.StatusDraft,
		Author:           "testuser",
	}

	err = client.SyncPlan(context.Background(), p)
	if err != nil {
		t.Fatalf("SyncPlan() error = %v", err)
	}

	// Verify the issue was updated
	issue, err := client.GetIssue(context.Background(), initialIssue.ID)
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}
	if issue.Title != p.Title {
		t.Errorf("issue.Title = %q, want %q", issue.Title, p.Title)
	}
	if !containsString(issue.Description, p.ProblemStatement) {
		t.Error("issue.Description should contain updated problem statement")
	}
}

func TestClient_SyncPlan_NilPlan(t *testing.T) {
	client := NewClient()

	err := client.SyncPlan(context.Background(), nil)
	if err == nil {
		t.Error("SyncPlan() should return error for nil plan")
	}
}

func TestClient_SyncPlanStatus_Success(t *testing.T) {
	tests := []struct {
		name           string
		planStatus     plan.Status
		expectedStatus tracker.Status
	}{
		{"draft to todo", plan.StatusDraft, tracker.StatusTodo},
		{"reviewing to todo", plan.StatusReviewing, tracker.StatusTodo},
		{"approved to todo", plan.StatusApproved, tracker.StatusTodo},
		{"in-progress to in_progress", plan.StatusInProgress, tracker.StatusInProgress},
		{"in-review to in_review", plan.StatusInReview, tracker.StatusInReview},
		{"complete to done", plan.StatusComplete, tracker.StatusDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient()

			// Create an issue first (fresh client for each test)
			issue, _ := client.CreateIssue(context.Background(), &tracker.Issue{
				Title: "Test Issue",
			})

			p := &plan.Plan{
				ID:     issue.ID, // Use internal ID, not Identifier
				Title:  "Test Plan",
				Status: tt.planStatus,
				Author: "testuser",
			}

			err := client.SyncPlanStatus(context.Background(), p)
			if err != nil {
				t.Fatalf("SyncPlanStatus() error = %v", err)
			}

			// Verify the issue status was updated
			updatedIssue, _ := client.GetIssue(context.Background(), issue.ID)
			if updatedIssue.Status != tt.expectedStatus {
				t.Errorf("issue.Status = %v, want %v", updatedIssue.Status, tt.expectedStatus)
			}
		})
	}
}

func TestClient_SyncPlanStatus_NilPlan(t *testing.T) {
	client := NewClient()

	err := client.SyncPlanStatus(context.Background(), nil)
	if err == nil {
		t.Error("SyncPlanStatus() should return error for nil plan")
	}
}

func TestClient_SyncPlanStatus_NoIssueID(t *testing.T) {
	client := NewClient()
	p := &plan.Plan{
		Title:  "Test Plan",
		Status: plan.StatusDraft,
		Author: "testuser",
	}

	err := client.SyncPlanStatus(context.Background(), p)
	if err == nil {
		t.Error("SyncPlanStatus() should return error when plan has no issue ID")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsString(s[1:], substr)) ||
		(len(s) >= len(substr) && s[:len(substr)] == substr))
}
