package cli

import (
	"strings"
	"testing"

	"github.com/charleslr/jig/internal/tracker"
)

func TestCreatePlanFromIssue(t *testing.T) {
	t.Run("returns nil for nil issue", func(t *testing.T) {
		result := createPlanFromIssue(nil)
		if result != nil {
			t.Error("expected nil for nil issue")
		}
	})

	t.Run("creates plan with correct fields", func(t *testing.T) {
		issue := &tracker.Issue{
			ID:          "internal-id-123",
			Identifier:  "NUM-41",
			Title:       "Implement feature X",
			Description: "We need to implement feature X because...",
			Assignee:    "testuser",
		}

		result := createPlanFromIssue(issue)

		if result == nil {
			t.Fatal("expected non-nil plan")
		}

		// Check ID is a local plan ID
		if !strings.HasPrefix(result.ID, "PLAN-") {
			t.Errorf("expected ID to start with 'PLAN-', got %q", result.ID)
		}

		// Check IssueID is set to the issue identifier
		if result.IssueID != "NUM-41" {
			t.Errorf("expected IssueID 'NUM-41', got %q", result.IssueID)
		}

		// Check title
		if result.Title != "Implement feature X" {
			t.Errorf("expected Title 'Implement feature X', got %q", result.Title)
		}

		// Check author is set from assignee
		if result.Author != "testuser" {
			t.Errorf("expected Author 'testuser', got %q", result.Author)
		}

		// Check problem statement is set from description
		if result.ProblemStatement != "We need to implement feature X because..." {
			t.Errorf("expected ProblemStatement to match description, got %q", result.ProblemStatement)
		}
	})

	t.Run("handles issue with empty fields", func(t *testing.T) {
		issue := &tracker.Issue{
			ID:         "id-456",
			Identifier: "ENG-1",
			Title:      "",
			Assignee:   "",
		}

		result := createPlanFromIssue(issue)

		if result == nil {
			t.Fatal("expected non-nil plan")
		}

		if result.IssueID != "ENG-1" {
			t.Errorf("expected IssueID 'ENG-1', got %q", result.IssueID)
		}

		// Empty fields should be preserved
		if result.Title != "" {
			t.Errorf("expected empty Title, got %q", result.Title)
		}
		if result.Author != "" {
			t.Errorf("expected empty Author, got %q", result.Author)
		}
	})

	t.Run("HasLinkedIssue returns true for created plan", func(t *testing.T) {
		issue := &tracker.Issue{
			ID:         "id-789",
			Identifier: "BUG-42",
			Title:      "Fix bug",
		}

		result := createPlanFromIssue(issue)

		if !result.HasLinkedIssue() {
			t.Error("expected HasLinkedIssue() to return true")
		}
	})
}
