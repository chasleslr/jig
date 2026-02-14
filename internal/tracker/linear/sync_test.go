package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/tracker"
)

func TestFormatPlanComment(t *testing.T) {
	t.Run("includes all plan sections", func(t *testing.T) {
		p := &plan.Plan{
			ID:               "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "This is the problem we're solving.",
			ProposedSolution: "This is how we'll solve it.",
		}

		result := formatPlanComment(p)

		// Check header
		if !strings.Contains(result, "## 📋 Implementation Plan") {
			t.Error("expected header '## 📋 Implementation Plan'")
		}

		// Check synced timestamp
		if !strings.Contains(result, "**Synced:**") {
			t.Error("expected synced timestamp")
		}

		// Check problem statement
		if !strings.Contains(result, "### Problem Statement") {
			t.Error("expected '### Problem Statement' section")
		}
		if !strings.Contains(result, "This is the problem we're solving.") {
			t.Error("expected problem statement content")
		}

		// Check proposed solution
		if !strings.Contains(result, "### Proposed Solution") {
			t.Error("expected '### Proposed Solution' section")
		}
		if !strings.Contains(result, "This is how we'll solve it.") {
			t.Error("expected proposed solution content")
		}

		// Check footer
		if !strings.Contains(result, "*This plan was synced by [jig]") {
			t.Error("expected jig attribution footer")
		}
	})

	t.Run("handles empty problem statement", func(t *testing.T) {
		p := &plan.Plan{
			ID:               "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "",
			ProposedSolution: "This is how we'll solve it.",
		}

		result := formatPlanComment(p)

		// Should not contain problem statement section when empty
		if strings.Contains(result, "### Problem Statement") {
			t.Error("should not include empty problem statement section")
		}

		// Should still contain proposed solution
		if !strings.Contains(result, "### Proposed Solution") {
			t.Error("expected '### Proposed Solution' section")
		}
	})

	t.Run("handles empty proposed solution", func(t *testing.T) {
		p := &plan.Plan{
			ID:               "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "This is the problem.",
			ProposedSolution: "",
		}

		result := formatPlanComment(p)

		// Should contain problem statement
		if !strings.Contains(result, "### Problem Statement") {
			t.Error("expected '### Problem Statement' section")
		}

		// Should not contain proposed solution section when empty
		if strings.Contains(result, "### Proposed Solution") {
			t.Error("should not include empty proposed solution section")
		}
	})

	t.Run("includes additional sections from raw content", func(t *testing.T) {
		p := &plan.Plan{
			ID:               "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "Problem",
			ProposedSolution: "Solution",
			RawContent: `## Problem Statement
Problem

## Proposed Solution
Solution

## Acceptance Criteria
- Criterion 1
- Criterion 2

## Implementation Details
Step by step details here.
`,
		}

		result := formatPlanComment(p)

		// Should include acceptance criteria
		if !strings.Contains(result, "## Acceptance Criteria") {
			t.Error("expected '## Acceptance Criteria' section from raw content")
		}
		if !strings.Contains(result, "Criterion 1") {
			t.Error("expected acceptance criteria content")
		}

		// Should include implementation details
		if !strings.Contains(result, "## Implementation Details") {
			t.Error("expected '## Implementation Details' section from raw content")
		}
	})
}

func TestExtractAdditionalSections(t *testing.T) {
	t.Run("extracts sections other than problem and solution", func(t *testing.T) {
		rawContent := `## Problem Statement
Problem content here.

## Proposed Solution
Solution content here.

## Acceptance Criteria
- AC 1
- AC 2

## Testing Strategy
Unit tests and integration tests.
`
		result := extractAdditionalSections(rawContent)

		// Should not include problem statement
		if strings.Contains(result, "Problem Statement") {
			t.Error("should not include Problem Statement section")
		}
		if strings.Contains(result, "Problem content here") {
			t.Error("should not include problem statement content")
		}

		// Should not include proposed solution
		if strings.Contains(result, "Proposed Solution") {
			t.Error("should not include Proposed Solution section")
		}
		if strings.Contains(result, "Solution content here") {
			t.Error("should not include proposed solution content")
		}

		// Should include acceptance criteria
		if !strings.Contains(result, "## Acceptance Criteria") {
			t.Error("expected Acceptance Criteria section")
		}
		if !strings.Contains(result, "AC 1") {
			t.Error("expected acceptance criteria content")
		}

		// Should include testing strategy
		if !strings.Contains(result, "## Testing Strategy") {
			t.Error("expected Testing Strategy section")
		}
		if !strings.Contains(result, "Unit tests and integration tests") {
			t.Error("expected testing strategy content")
		}
	})

	t.Run("handles triple hash headers", func(t *testing.T) {
		rawContent := `### Problem Statement
Problem content.

### Proposed Solution
Solution content.

### Notes
Additional notes here.
`
		result := extractAdditionalSections(rawContent)

		// Should extract notes section
		if !strings.Contains(result, "### Notes") {
			t.Error("expected Notes section")
		}
		if !strings.Contains(result, "Additional notes here") {
			t.Error("expected notes content")
		}

		// Should not include skipped sections
		if strings.Contains(result, "Problem Statement") {
			t.Error("should not include Problem Statement")
		}
		if strings.Contains(result, "Proposed Solution") {
			t.Error("should not include Proposed Solution")
		}
	})

	t.Run("handles case insensitive matching", func(t *testing.T) {
		rawContent := `## PROBLEM STATEMENT
Problem.

## proposed solution
Solution.

## Other Section
Content.
`
		result := extractAdditionalSections(rawContent)

		// Should extract other section
		if !strings.Contains(result, "## Other Section") {
			t.Error("expected Other Section")
		}

		// Should not include uppercase problem statement
		if strings.Contains(result, "PROBLEM STATEMENT") {
			t.Error("should not include PROBLEM STATEMENT")
		}
	})

	t.Run("returns empty for no additional sections", func(t *testing.T) {
		rawContent := `## Problem Statement
Just problem.

## Proposed Solution
Just solution.
`
		result := extractAdditionalSections(rawContent)

		if result != "" {
			t.Errorf("expected empty result, got %q", result)
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		result := extractAdditionalSections("")

		if result != "" {
			t.Errorf("expected empty result for empty input, got %q", result)
		}
	})

	t.Run("preserves content between sections", func(t *testing.T) {
		rawContent := `## Problem Statement
Problem.

## Implementation
Step 1
Step 2
Step 3

More details here.

## Testing
Test approach.
`
		result := extractAdditionalSections(rawContent)

		// Should include all implementation content
		if !strings.Contains(result, "Step 1") {
			t.Error("expected Step 1")
		}
		if !strings.Contains(result, "Step 2") {
			t.Error("expected Step 2")
		}
		if !strings.Contains(result, "More details here") {
			t.Error("expected 'More details here'")
		}

		// Should include testing section
		if !strings.Contains(result, "## Testing") {
			t.Error("expected Testing section")
		}
	})
}

func TestGetIssueLabelIDs(t *testing.T) {
	t.Run("returns empty slice on error", func(t *testing.T) {
		// Test with invalid context/client that will fail
		// The function handles errors gracefully and returns nil
		client := NewClient("invalid", "", "")
		result := getIssueLabelIDs(nil, client, "invalid-id")
		if result != nil {
			t.Errorf("expected nil on error, got %v", result)
		}
	})

	t.Run("returns label IDs from issue", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issue": {
						"labels": {
							"nodes": [
								{"id": "label-1"},
								{"id": "label-2"},
								{"id": "label-3"}
							]
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		result := getIssueLabelIDs(context.Background(), client, "issue-123")

		if len(result) != 3 {
			t.Errorf("expected 3 label IDs, got %d", len(result))
		}
		if result[0] != "label-1" || result[1] != "label-2" || result[2] != "label-3" {
			t.Errorf("unexpected label IDs: %v", result)
		}
	})

	t.Run("returns empty slice for issue with no labels", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issue": {
						"labels": {
							"nodes": []
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		result := getIssueLabelIDs(context.Background(), client, "issue-123")

		if len(result) != 0 {
			t.Errorf("expected 0 label IDs, got %d", len(result))
		}
	})
}

func TestCreateIssueFromPlan(t *testing.T) {
	t.Run("creates issue from plan successfully", func(t *testing.T) {
		var capturedTitle string
		var capturedDescription string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req GraphQLRequest
			json.NewDecoder(r.Body).Decode(&req)

			if input, ok := req.Variables["input"].(map[string]interface{}); ok {
				if title, ok := input["title"].(string); ok {
					capturedTitle = title
				}
				if desc, ok := input["description"].(string); ok {
					capturedDescription = desc
				}
			}

			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueCreate": {
						"success": true,
						"issue": {
							"id": "issue-uuid-123",
							"identifier": "NUM-99",
							"title": "Test Plan Title",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"},
							"team": {"id": "team-123", "key": "NUM"}
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		p := &plan.Plan{
			ID:               "PLAN-123",
			Title:            "Test Plan Title",
			ProblemStatement: "This is the problem we need to solve.",
			ProposedSolution: "Here is how we plan to solve it.",
		}

		issue, err := client.CreateIssueFromPlan(context.Background(), p)
		if err != nil {
			t.Fatalf("CreateIssueFromPlan failed: %v", err)
		}

		// Verify the created issue
		if issue.Identifier != "NUM-99" {
			t.Errorf("expected identifier 'NUM-99', got '%s'", issue.Identifier)
		}

		// Verify the title was passed correctly
		if capturedTitle != "Test Plan Title" {
			t.Errorf("expected title 'Test Plan Title', got '%s'", capturedTitle)
		}

		// Verify the description includes problem statement
		if !strings.Contains(capturedDescription, "Problem Statement") {
			t.Error("expected description to contain 'Problem Statement'")
		}
		if !strings.Contains(capturedDescription, "This is the problem we need to solve.") {
			t.Error("expected description to contain problem statement content")
		}

		// Verify the description includes proposed solution
		if !strings.Contains(capturedDescription, "Proposed Solution") {
			t.Error("expected description to contain 'Proposed Solution'")
		}
		if !strings.Contains(capturedDescription, "Here is how we plan to solve it.") {
			t.Error("expected description to contain proposed solution content")
		}
	})

	t.Run("returns error when plan has no title", func(t *testing.T) {
		client := NewClient("test-key", "team-id", "")
		p := &plan.Plan{
			ID:               "PLAN-123",
			Title:            "", // Empty title
			ProblemStatement: "Problem",
			ProposedSolution: "Solution",
		}

		_, err := client.CreateIssueFromPlan(context.Background(), p)
		if err == nil {
			t.Error("expected error when plan has no title")
		}
		if !strings.Contains(err.Error(), "plan title is required") {
			t.Errorf("expected 'plan title is required' error, got: %v", err)
		}
	})

	t.Run("handles API error gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := GraphQLResponse{
				Errors: []GraphQLError{
					{Message: "Internal server error"},
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		p := &plan.Plan{
			ID:    "PLAN-123",
			Title: "Test Plan",
		}

		_, err := client.CreateIssueFromPlan(context.Background(), p)
		if err == nil {
			t.Error("expected error on API failure")
		}
	})

	t.Run("creates issue with empty problem statement", func(t *testing.T) {
		var capturedDescription string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req GraphQLRequest
			json.NewDecoder(r.Body).Decode(&req)

			if input, ok := req.Variables["input"].(map[string]interface{}); ok {
				if desc, ok := input["description"].(string); ok {
					capturedDescription = desc
				}
			}

			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueCreate": {
						"success": true,
						"issue": {
							"id": "issue-uuid-123",
							"identifier": "NUM-100",
							"title": "Test Plan",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"},
							"team": {"id": "team-123", "key": "NUM"}
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		p := &plan.Plan{
			ID:               "PLAN-123",
			Title:            "Test Plan",
			ProblemStatement: "", // Empty problem statement
			ProposedSolution: "This is the solution.",
		}

		issue, err := client.CreateIssueFromPlan(context.Background(), p)
		if err != nil {
			t.Fatalf("CreateIssueFromPlan failed: %v", err)
		}

		if issue.Identifier != "NUM-100" {
			t.Errorf("expected identifier 'NUM-100', got '%s'", issue.Identifier)
		}

		// Description should only contain proposed solution
		if strings.Contains(capturedDescription, "Problem Statement") {
			t.Error("description should not contain 'Problem Statement' when empty")
		}
		if !strings.Contains(capturedDescription, "Proposed Solution") {
			t.Error("description should contain 'Proposed Solution'")
		}
	})

	t.Run("uses client teamID when creating issue", func(t *testing.T) {
		var capturedTeamID string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req GraphQLRequest
			json.NewDecoder(r.Body).Decode(&req)

			if input, ok := req.Variables["input"].(map[string]interface{}); ok {
				if teamID, ok := input["teamId"].(string); ok {
					capturedTeamID = teamID
				}
			}

			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueCreate": {
						"success": true,
						"issue": {
							"id": "issue-uuid-123",
							"identifier": "NUM-101",
							"title": "Test Plan",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"},
							"team": {"id": "team-custom", "key": "NUM"}
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		// Create client with specific teamID
		client := newTestClientWithTeam(server.URL, "team-custom-123")
		p := &plan.Plan{
			ID:    "PLAN-123",
			Title: "Test Plan",
		}

		_, err := client.CreateIssueFromPlan(context.Background(), p)
		if err != nil {
			t.Fatalf("CreateIssueFromPlan failed: %v", err)
		}

		if capturedTeamID != "team-custom-123" {
			t.Errorf("expected teamId 'team-custom-123', got '%s'", capturedTeamID)
		}
	})
}

func TestSyncPlanToIssue(t *testing.T) {
	t.Run("returns error when plan has no linked issue", func(t *testing.T) {
		client := NewClient("test-key", "team-id", "")
		p := &plan.Plan{
			ID:      "PLAN-123",
			IssueID: "", // No linked issue
		}

		err := client.SyncPlanToIssue(context.Background(), p, "jig-plan")
		if err == nil {
			t.Error("expected error when plan has no linked issue")
		}
		if !strings.Contains(err.Error(), "no linked issue") {
			t.Errorf("expected 'no linked issue' error, got: %v", err)
		}
	})

	t.Run("syncs plan to issue successfully", func(t *testing.T) {
		callCount := 0
		var capturedCommentBody string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req GraphQLRequest
			json.NewDecoder(r.Body).Decode(&req)
			callCount++

			// Determine which query is being made based on call order
			switch callCount {
			case 1:
				// GetIssue query
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"issues": {
							"nodes": [{
								"id": "internal-issue-id",
								"identifier": "NUM-41",
								"title": "Test Issue",
								"team": {"id": "team-123"},
								"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
							}]
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
			case 2:
				// AddComment mutation
				if input, ok := req.Variables["input"].(map[string]interface{}); ok {
					if body, ok := input["body"].(string); ok {
						capturedCommentBody = body
					}
				}
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"commentCreate": {
							"success": true,
							"comment": {
								"id": "comment-123",
								"body": "test"
							}
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
			case 3:
				// GetTeamLabels query
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"team": {
							"labels": {
								"nodes": [
									{"id": "existing-label", "name": "bug"},
									{"id": "jig-plan-label", "name": "jig-plan"}
								]
							}
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
			case 4:
				// GetIssueLabels query
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"issue": {
							"labels": {
								"nodes": [
									{"id": "existing-label"}
								]
							}
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
			case 5:
				// AddLabelToIssue mutation
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"issueUpdate": {
							"success": true
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		p := &plan.Plan{
			ID:               "PLAN-123",
			IssueID:          "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "Test problem",
			ProposedSolution: "Test solution",
		}

		err := client.SyncPlanToIssue(context.Background(), p, "jig-plan")
		if err != nil {
			t.Fatalf("SyncPlanToIssue failed: %v", err)
		}

		// Verify comment was created with plan content
		if !strings.Contains(capturedCommentBody, "Implementation Plan") {
			t.Error("expected comment to contain 'Implementation Plan'")
		}
		if !strings.Contains(capturedCommentBody, "Test problem") {
			t.Error("expected comment to contain problem statement")
		}

		// Verify all expected calls were made
		if callCount != 5 {
			t.Errorf("expected 5 API calls, got %d", callCount)
		}
	})

	t.Run("returns error when issue not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": []
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		p := &plan.Plan{
			ID:      "PLAN-123",
			IssueID: "NONEXISTENT-999",
		}

		err := client.SyncPlanToIssue(context.Background(), p, "jig-plan")
		if err == nil {
			t.Error("expected error when issue not found")
		}
		if !strings.Contains(err.Error(), "failed to get issue") {
			t.Errorf("expected 'failed to get issue' error, got: %v", err)
		}
	})
}

func TestSyncPlanStatus(t *testing.T) {
	t.Run("returns error when plan has no linked issue", func(t *testing.T) {
		client := NewClient("test-key", "team-id", "")
		p := &plan.Plan{
			ID:      "", // Empty ID (no fallback)
			IssueID: "", // No linked issue
		}

		err := client.SyncPlanStatus(context.Background(), p)
		if err == nil {
			t.Error("expected error when plan has no linked issue")
		}
		if !strings.Contains(err.Error(), "no linked issue") {
			t.Errorf("expected 'no linked issue' error, got: %v", err)
		}
	})

	t.Run("uses IssueID not ID for status sync", func(t *testing.T) {
		var capturedTeamKey string
		var capturedIssueNumber int

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req GraphQLRequest
			json.NewDecoder(r.Body).Decode(&req)

			// First call: resolveIssueID - issues query with filter (team.key and number)
			if strings.Contains(req.Query, "issues") && strings.Contains(req.Query, "filter") {
				if filter, ok := req.Variables["filter"].(map[string]interface{}); ok {
					if team, ok := filter["team"].(map[string]interface{}); ok {
						if key, ok := team["key"].(map[string]interface{}); ok {
							if eq, ok := key["eq"].(string); ok {
								capturedTeamKey = eq
							}
						}
					}
					if number, ok := filter["number"].(map[string]interface{}); ok {
						if eq, ok := number["eq"].(float64); ok {
							capturedIssueNumber = int(eq)
						}
					}
				}
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"issues": {
							"nodes": [{
								"id": "internal-uuid",
								"identifier": "NUM-82",
								"title": "Test Issue",
								"team": {"id": "team-123", "key": "NUM"},
								"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
							}]
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			// Second call: getWorkflowStates - issue(id: $id) query
			if strings.Contains(req.Query, "GetWorkflowStates") || (strings.Contains(req.Query, "issue") && strings.Contains(req.Query, "team") && strings.Contains(req.Query, "states")) {
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"issue": {
							"team": {
								"states": {
									"nodes": [
										{"id": "todo-id", "name": "Todo", "type": "unstarted"},
										{"id": "in-progress-id", "name": "In Progress", "type": "started"},
										{"id": "done-id", "name": "Done", "type": "completed"}
									]
								}
							}
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			// Third call: IssueUpdate mutation
			if strings.Contains(req.Query, "issueUpdate") {
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"issueUpdate": {"success": true}
					}`),
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			// Fallback
			response := GraphQLResponse{
				Data: json.RawMessage(`{}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		p := &plan.Plan{
			ID:      "PLAN-1771026009", // Internal plan ID (should NOT be used)
			IssueID: "NUM-82",          // Linear issue ID (should be used)
			Status:  plan.StatusInProgress,
		}

		err := client.SyncPlanStatus(context.Background(), p)
		if err != nil {
			t.Fatalf("SyncPlanStatus failed: %v", err)
		}

		// Verify IssueID (NUM-82) was used by checking the parsed components
		if capturedTeamKey != "NUM" {
			t.Errorf("expected team key 'NUM', but captured: %q", capturedTeamKey)
		}
		if capturedIssueNumber != 82 {
			t.Errorf("expected issue number 82, but captured: %d", capturedIssueNumber)
		}
	})
}

func TestPlanRoundTrip(t *testing.T) {
	t.Run("preserves all sections through sync and fetch", func(t *testing.T) {
		// Create a plan with all sections
		original := &plan.Plan{
			ID:               "test-plan",
			IssueID:          "TEST-123",
			Title:            "Test Plan",
			ProblemStatement: "The problem",
			ProposedSolution: "The solution",
			RawContent: `---
id: test-plan
---

## Problem Statement

The problem

## Proposed Solution

The solution

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Implementation Details

Details here

## Verification

1. Step 1
2. Step 2
`,
		}

		// Format as comment (simulates sync to Linear)
		comment := formatPlanComment(original)

		// Parse back (simulates fetch from Linear)
		issue := &tracker.Issue{
			Identifier: "TEST-123",
			Title:      "Test Plan",
		}
		fetched, err := parsePlanFromComment(comment, issue)
		if err != nil {
			t.Fatalf("parsePlanFromComment failed: %v", err)
		}

		// Serialize the fetched plan
		serialized, err := plan.Serialize(fetched)
		if err != nil {
			t.Fatalf("plan.Serialize failed: %v", err)
		}

		content := string(serialized)

		// Verify all sections are present
		if !strings.Contains(content, "## Acceptance Criteria") {
			t.Error("expected '## Acceptance Criteria' section")
		}
		if !strings.Contains(content, "Criterion 1") {
			t.Error("expected 'Criterion 1' in acceptance criteria")
		}
		if !strings.Contains(content, "## Implementation Details") {
			t.Error("expected '## Implementation Details' section")
		}
		if !strings.Contains(content, "Details here") {
			t.Error("expected 'Details here' in implementation details")
		}
		if !strings.Contains(content, "## Verification") {
			t.Error("expected '## Verification' section")
		}
		if !strings.Contains(content, "Step 1") {
			t.Error("expected 'Step 1' in verification")
		}

		// Verify problem statement and proposed solution are also preserved
		if !strings.Contains(content, "## Problem Statement") {
			t.Error("expected '## Problem Statement' section")
		}
		if !strings.Contains(content, "The problem") {
			t.Error("expected problem statement content")
		}
		if !strings.Contains(content, "## Proposed Solution") {
			t.Error("expected '## Proposed Solution' section")
		}
		if !strings.Contains(content, "The solution") {
			t.Error("expected proposed solution content")
		}
	})

	t.Run("preserves struct fields after round-trip", func(t *testing.T) {
		original := &plan.Plan{
			ID:               "test-plan",
			IssueID:          "TEST-456",
			Title:            "Another Test",
			ProblemStatement: "A specific problem",
			ProposedSolution: "A specific solution",
			RawContent: `---
id: test-plan
---

## Problem Statement

A specific problem

## Proposed Solution

A specific solution
`,
		}

		comment := formatPlanComment(original)
		issue := &tracker.Issue{
			Identifier: "TEST-456",
			Title:      "Another Test",
		}
		fetched, err := parsePlanFromComment(comment, issue)
		if err != nil {
			t.Fatalf("parsePlanFromComment failed: %v", err)
		}

		// Verify struct fields are populated
		if fetched.ProblemStatement != "A specific problem" {
			t.Errorf("expected ProblemStatement 'A specific problem', got %q", fetched.ProblemStatement)
		}
		if fetched.ProposedSolution != "A specific solution" {
			t.Errorf("expected ProposedSolution 'A specific solution', got %q", fetched.ProposedSolution)
		}
		if fetched.IssueID != "TEST-456" {
			t.Errorf("expected IssueID 'TEST-456', got %q", fetched.IssueID)
		}
	})
}

func TestConvertCommentToBodyContent(t *testing.T) {
	t.Run("extracts content between separators", func(t *testing.T) {
		comment := `## 📋 Implementation Plan

**Synced:** 2024-01-01 12:00 UTC

---

### Problem Statement

The problem here

### Proposed Solution

The solution here

### Acceptance Criteria

- AC 1
- AC 2

---

*This plan was synced by [jig](https://github.com/charleslr/jig)*`

		result := convertCommentToBodyContent(comment)

		// Should convert ### to ##
		if !strings.Contains(result, "## Problem Statement") {
			t.Error("expected '## Problem Statement' (converted from ###)")
		}
		if !strings.Contains(result, "## Acceptance Criteria") {
			t.Error("expected '## Acceptance Criteria' (converted from ###)")
		}

		// Should include content
		if !strings.Contains(result, "The problem here") {
			t.Error("expected problem content")
		}
		if !strings.Contains(result, "AC 1") {
			t.Error("expected acceptance criteria content")
		}

		// Should NOT include header or footer
		if strings.Contains(result, "📋 Implementation Plan") {
			t.Error("should not include header")
		}
		if strings.Contains(result, "Synced:") {
			t.Error("should not include synced metadata")
		}
		if strings.Contains(result, "This plan was synced") {
			t.Error("should not include footer")
		}
	})

	t.Run("handles empty comment", func(t *testing.T) {
		result := convertCommentToBodyContent("")
		if result != "" {
			t.Errorf("expected empty result for empty input, got %q", result)
		}
	})
}

func TestParseSectionsIntoFields(t *testing.T) {
	t.Run("extracts problem and solution", func(t *testing.T) {
		body := `## Problem Statement

This is the problem

## Proposed Solution

This is the solution

## Other Section

Other content`

		p := &plan.Plan{}
		parseSectionsIntoFields(p, body)

		if p.ProblemStatement != "This is the problem" {
			t.Errorf("expected ProblemStatement 'This is the problem', got %q", p.ProblemStatement)
		}
		if p.ProposedSolution != "This is the solution" {
			t.Errorf("expected ProposedSolution 'This is the solution', got %q", p.ProposedSolution)
		}
	})

	t.Run("handles missing sections", func(t *testing.T) {
		body := `## Acceptance Criteria

- AC 1`

		p := &plan.Plan{}
		parseSectionsIntoFields(p, body)

		if p.ProblemStatement != "" {
			t.Errorf("expected empty ProblemStatement, got %q", p.ProblemStatement)
		}
		if p.ProposedSolution != "" {
			t.Errorf("expected empty ProposedSolution, got %q", p.ProposedSolution)
		}
	})
}

func TestComputePlanContentHash(t *testing.T) {
	t.Run("returns consistent hash for same content", func(t *testing.T) {
		p := &plan.Plan{
			ID:               "PLAN-123",
			Title:            "Test Plan",
			ProblemStatement: "This is the problem.",
			ProposedSolution: "This is the solution.",
		}

		hash1 := ComputePlanContentHash(p)
		hash2 := ComputePlanContentHash(p)

		if hash1 != hash2 {
			t.Errorf("hash should be consistent: %q != %q", hash1, hash2)
		}
	})

	t.Run("returns different hash for different content", func(t *testing.T) {
		p1 := &plan.Plan{
			ID:               "PLAN-123",
			ProblemStatement: "Problem A",
		}
		p2 := &plan.Plan{
			ID:               "PLAN-123",
			ProblemStatement: "Problem B",
		}

		hash1 := ComputePlanContentHash(p1)
		hash2 := ComputePlanContentHash(p2)

		if hash1 == hash2 {
			t.Error("hash should be different for different content")
		}
	})

	t.Run("returns valid hex string", func(t *testing.T) {
		p := &plan.Plan{
			ID:    "PLAN-123",
			Title: "Test",
		}

		hash := ComputePlanContentHash(p)

		// SHA256 produces 64 hex characters
		if len(hash) != 64 {
			t.Errorf("expected 64 character hex string, got %d characters", len(hash))
		}

		// Should only contain hex characters
		for _, c := range hash {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("hash contains non-hex character: %c", c)
			}
		}
	})

	t.Run("includes raw content in hash calculation", func(t *testing.T) {
		p1 := &plan.Plan{
			ID:               "PLAN-123",
			ProblemStatement: "Problem",
			RawContent:       "",
		}
		p2 := &plan.Plan{
			ID:               "PLAN-123",
			ProblemStatement: "Problem",
			RawContent: `## Acceptance Criteria
- AC 1
- AC 2`,
		}

		hash1 := ComputePlanContentHash(p1)
		hash2 := ComputePlanContentHash(p2)

		if hash1 == hash2 {
			t.Error("hash should be different when RawContent changes")
		}
	})
}
