package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charleslr/jig/internal/tracker"
)

// TestGetIssueByIdentifier_NumberType verifies that the issue number is sent as
// an integer, not a string. This is a regression test for a bug where "NUM-14"
// would send {"number": {"eq": "14"}} instead of {"number": {"eq": 14}},
// causing the Linear API to reject it with "Float cannot represent non numeric value".
func TestGetIssueByIdentifier_NumberType(t *testing.T) {
	var capturedRequest GraphQLRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		// Return a valid response
		response := GraphQLResponse{
			Data: json.RawMessage(`{
				"issues": {
					"nodes": [{
						"id": "issue-123",
						"identifier": "NUM-14",
						"title": "Test Issue",
						"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
					}]
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetIssue(context.Background(), "NUM-14")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}

	// Verify the filter structure
	filter, ok := capturedRequest.Variables["filter"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected filter to be a map, got %T", capturedRequest.Variables["filter"])
	}

	number, ok := filter["number"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected filter.number to be a map, got %T", filter["number"])
	}

	eq := number["eq"]

	// This is the critical assertion: eq must be a number, not a string
	switch v := eq.(type) {
	case float64:
		if v != 14 {
			t.Errorf("expected number.eq to be 14, got %v", v)
		}
	case int:
		if v != 14 {
			t.Errorf("expected number.eq to be 14, got %v", v)
		}
	default:
		t.Errorf("expected number.eq to be a number type, got %T with value %v", eq, eq)
	}
}

func TestGetIssueByIdentifier_InvalidFormat(t *testing.T) {
	// Test that non-numeric issue numbers are rejected before making API calls.
	// Note: identifiers without hyphens (e.g., "NUM14") are treated as direct IDs,
	// not team-number identifiers, so they go through a different code path.

	client := NewClient("test-key", "", "")

	_, err := client.GetIssue(context.Background(), "NUM-abc")
	if err == nil {
		t.Fatal("expected error for non-numeric issue number, got nil")
	}
	if !contains(err.Error(), "invalid issue number") {
		t.Errorf("expected error containing %q, got %q", "invalid issue number", err.Error())
	}
}

func TestGetIssueByIdentifier_VariousFormats(t *testing.T) {
	tests := []struct {
		name           string
		identifier     string
		expectedTeam   string
		expectedNumber float64
	}{
		{
			name:           "standard format",
			identifier:     "ENG-123",
			expectedTeam:   "ENG",
			expectedNumber: 123,
		},
		{
			name:           "single digit",
			identifier:     "NUM-1",
			expectedTeam:   "NUM",
			expectedNumber: 1,
		},
		{
			name:           "large number",
			identifier:     "PROJ-99999",
			expectedTeam:   "PROJ",
			expectedNumber: 99999,
		},
		{
			name:           "lowercase team key",
			identifier:     "eng-42",
			expectedTeam:   "eng",
			expectedNumber: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequest GraphQLRequest

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewDecoder(r.Body).Decode(&capturedRequest)
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"issues": {
							"nodes": [{
								"id": "id-1",
								"identifier": "` + tt.identifier + `",
								"title": "Test",
								"state": {"type": "unstarted"}
							}]
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := newTestClient(server.URL)
			_, _ = client.GetIssue(context.Background(), tt.identifier)

			filter := capturedRequest.Variables["filter"].(map[string]interface{})

			// Verify team key
			team := filter["team"].(map[string]interface{})
			teamKey := team["key"].(map[string]interface{})
			if teamKey["eq"] != tt.expectedTeam {
				t.Errorf("expected team.key.eq = %q, got %q", tt.expectedTeam, teamKey["eq"])
			}

			// Verify number is numeric (JSON unmarshals numbers as float64)
			number := filter["number"].(map[string]interface{})
			eq := number["eq"]
			switch v := eq.(type) {
			case float64:
				if v != tt.expectedNumber {
					t.Errorf("expected number.eq = %v, got %v", tt.expectedNumber, v)
				}
			case int:
				if float64(v) != tt.expectedNumber {
					t.Errorf("expected number.eq = %v, got %v", tt.expectedNumber, v)
				}
			default:
				t.Errorf("number.eq should be numeric, got %T", eq)
			}
		})
	}
}

// newTestClient creates a client that points to a test server
func newTestClient(serverURL string) *Client {
	client := NewClient("test-api-key", "", "")
	// Override the HTTP client to use our test server
	client.httpClient = &http.Client{
		Transport: &testTransport{baseURL: serverURL},
	}
	return client
}

// newTestClientWithTeam creates a client with a specific teamID that points to a test server
func newTestClientWithTeam(serverURL, teamID string) *Client {
	client := NewClient("test-api-key", teamID, "")
	// Override the HTTP client to use our test server
	client.httpClient = &http.Client{
		Transport: &testTransport{baseURL: serverURL},
	}
	return client
}

// testTransport redirects requests to the test server
type testTransport struct {
	baseURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect the request to our test server
	req.URL.Scheme = "http"
	req.URL.Host = t.baseURL[7:] // strip "http://"
	return http.DefaultTransport.RoundTrip(req)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestStatusMatches_InProgressExcludesReview verifies that StatusInProgress does not match
// states with "review" in the name. This is a regression test for a bug where "In Review"
// states (type: "started", name contains "review") would incorrectly match StatusInProgress.
func TestStatusMatches_InProgressExcludesReview(t *testing.T) {
	tests := []struct {
		name          string
		state         LinearWorkflowState
		status        tracker.Status
		expectedMatch bool
	}{
		{
			name:          "InProgress matches started state without review",
			state:         LinearWorkflowState{ID: "1", Name: "In Progress", Type: "started"},
			status:        tracker.StatusInProgress,
			expectedMatch: true,
		},
		{
			name:          "InProgress does not match state with review in name",
			state:         LinearWorkflowState{ID: "2", Name: "In Review", Type: "started"},
			status:        tracker.StatusInProgress,
			expectedMatch: false,
		},
		{
			name:          "InProgress does not match state with Review (capitalized)",
			state:         LinearWorkflowState{ID: "3", Name: "Code Review", Type: "started"},
			status:        tracker.StatusInProgress,
			expectedMatch: false,
		},
		{
			name:          "InReview matches state with review in name",
			state:         LinearWorkflowState{ID: "4", Name: "In Review", Type: "started"},
			status:        tracker.StatusInReview,
			expectedMatch: true,
		},
		{
			name:          "InReview does not match state without review in name",
			state:         LinearWorkflowState{ID: "5", Name: "In Progress", Type: "started"},
			status:        tracker.StatusInReview,
			expectedMatch: false,
		},
		{
			name:          "InProgress does not match non-started state",
			state:         LinearWorkflowState{ID: "6", Name: "Backlog", Type: "backlog"},
			status:        tracker.StatusInProgress,
			expectedMatch: false,
		},
		{
			name:          "Backlog matches backlog state",
			state:         LinearWorkflowState{ID: "7", Name: "Backlog", Type: "backlog"},
			status:        tracker.StatusBacklog,
			expectedMatch: true,
		},
		{
			name:          "Todo matches unstarted state",
			state:         LinearWorkflowState{ID: "8", Name: "Todo", Type: "unstarted"},
			status:        tracker.StatusTodo,
			expectedMatch: true,
		},
		{
			name:          "Done matches completed state",
			state:         LinearWorkflowState{ID: "9", Name: "Done", Type: "completed"},
			status:        tracker.StatusDone,
			expectedMatch: true,
		},
		{
			name:          "Canceled matches canceled state",
			state:         LinearWorkflowState{ID: "10", Name: "Canceled", Type: "canceled"},
			status:        tracker.StatusCanceled,
			expectedMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusMatches(tt.state, tt.status)
			if got != tt.expectedMatch {
				t.Errorf("statusMatches(%+v, %v) = %v, want %v", tt.state, tt.status, got, tt.expectedMatch)
			}
		})
	}
}

// TestResolveIssueID_Identifier verifies that human-readable identifiers
// (e.g., "NUM-70") are resolved to internal UUIDs via getIssueByIdentifier.
func TestResolveIssueID_Identifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response for getIssueByIdentifier query
		response := GraphQLResponse{
			Data: json.RawMessage(`{
				"issues": {
					"nodes": [{
						"id": "internal-uuid-123",
						"identifier": "NUM-70",
						"title": "Test Issue",
						"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
					}]
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	resolvedID, err := client.resolveIssueID(context.Background(), "NUM-70")
	if err != nil {
		t.Fatalf("resolveIssueID failed: %v", err)
	}

	if resolvedID != "internal-uuid-123" {
		t.Errorf("expected resolved ID to be %q, got %q", "internal-uuid-123", resolvedID)
	}
}

// TestResolveIssueID_UUID verifies that UUIDs (no hyphens in Linear's base62 format)
// are returned unchanged without making API calls.
func TestResolveIssueID_UUID(t *testing.T) {
	// Create a client with a test server that should never be called
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("API should not be called when resolving a UUID")
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	uuid := "abc123def456"
	resolvedID, err := client.resolveIssueID(context.Background(), uuid)
	if err != nil {
		t.Fatalf("resolveIssueID failed: %v", err)
	}

	if resolvedID != uuid {
		t.Errorf("expected resolved ID to be %q, got %q", uuid, resolvedID)
	}
}

// TestTransitionIssue_WithIdentifier verifies that TransitionIssue correctly
// resolves identifiers to UUIDs and uses the internal UUID in the mutation.
func TestTransitionIssue_WithIdentifier(t *testing.T) {
	requestCount := 0
	var capturedMutationRequest GraphQLRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// First request: getIssueByIdentifier (resolve NUM-70 → internal UUID)
		if requestCount == 1 {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [{
							"id": "internal-uuid-123",
							"identifier": "NUM-70",
							"title": "Test Issue",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
						}]
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Second request: getWorkflowStates
		if requestCount == 2 {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issue": {
						"team": {
							"states": {
								"nodes": [
									{"id": "state-1", "name": "Todo", "type": "unstarted"},
									{"id": "state-2", "name": "In Progress", "type": "started"},
									{"id": "state-3", "name": "Done", "type": "completed"}
								]
							}
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Third request: issueUpdate mutation
		if requestCount == 3 {
			capturedMutationRequest = req
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueUpdate": {
						"success": true
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.TransitionIssue(context.Background(), "NUM-70", tracker.StatusInProgress)
	if err != nil {
		t.Fatalf("TransitionIssue failed: %v", err)
	}

	// Verify the mutation used the internal UUID, not the identifier
	mutationID := capturedMutationRequest.Variables["id"]
	if mutationID != "internal-uuid-123" {
		t.Errorf("expected mutation to use internal UUID %q, got %q", "internal-uuid-123", mutationID)
	}

	// Verify we made exactly 3 requests (resolve + get states + update)
	if requestCount != 3 {
		t.Errorf("expected 3 API requests, got %d", requestCount)
	}
}

// TestGetAvailableStatuses_WithIdentifier verifies that GetAvailableStatuses
// correctly resolves identifiers to UUIDs before fetching workflow states.
func TestGetAvailableStatuses_WithIdentifier(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// First request: getIssueByIdentifier (resolve NUM-70 → internal UUID)
		if requestCount == 1 {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [{
							"id": "internal-uuid-456",
							"identifier": "NUM-70",
							"title": "Test Issue",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
						}]
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Second request: getWorkflowStates
		if requestCount == 2 {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issue": {
						"team": {
							"states": {
								"nodes": [
									{"id": "state-1", "name": "Backlog", "type": "backlog"},
									{"id": "state-2", "name": "Todo", "type": "unstarted"},
									{"id": "state-3", "name": "In Progress", "type": "started"},
									{"id": "state-4", "name": "Done", "type": "completed"}
								]
							}
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	statuses, err := client.GetAvailableStatuses(context.Background(), "NUM-70")
	if err != nil {
		t.Fatalf("GetAvailableStatuses failed: %v", err)
	}

	// Verify we got the expected statuses
	expectedStatuses := []tracker.Status{
		tracker.StatusBacklog,
		tracker.StatusTodo,
		tracker.StatusInProgress,
		tracker.StatusDone,
	}

	if len(statuses) != len(expectedStatuses) {
		t.Errorf("expected %d statuses, got %d", len(expectedStatuses), len(statuses))
	}

	// Verify we made exactly 2 requests (resolve + get states)
	if requestCount != 2 {
		t.Errorf("expected 2 API requests, got %d", requestCount)
	}
}

// TestResolveIssueID_ErrorCase verifies that resolveIssueID returns an error
// when the identifier cannot be resolved.
func TestResolveIssueID_ErrorCase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty results to simulate "not found"
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
	_, err := client.resolveIssueID(context.Background(), "INVALID-999")
	if err == nil {
		t.Fatal("expected error for invalid identifier, got nil")
	}

	if !contains(err.Error(), "failed to resolve identifier") {
		t.Errorf("expected error containing %q, got %q", "failed to resolve identifier", err.Error())
	}
}

// TestTransitionIssue_ResolveError verifies that TransitionIssue returns an error
// when identifier resolution fails.
func TestTransitionIssue_ResolveError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty results to simulate "not found"
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
	err := client.TransitionIssue(context.Background(), "INVALID-999", tracker.StatusInProgress)
	if err == nil {
		t.Fatal("expected error when identifier resolution fails, got nil")
	}

	if !contains(err.Error(), "failed to resolve identifier") {
		t.Errorf("expected error containing %q, got %q", "failed to resolve identifier", err.Error())
	}
}

// TestGetAvailableStatuses_ResolveError verifies that GetAvailableStatuses returns
// an error when identifier resolution fails.
func TestGetAvailableStatuses_ResolveError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty results to simulate "not found"
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
	_, err := client.GetAvailableStatuses(context.Background(), "INVALID-999")
	if err == nil {
		t.Fatal("expected error when identifier resolution fails, got nil")
	}

	if !contains(err.Error(), "failed to resolve identifier") {
		t.Errorf("expected error containing %q, got %q", "failed to resolve identifier", err.Error())
	}
}

// TestTransitionIssue_GetWorkflowStatesError verifies error handling when
// getWorkflowStates fails.
func TestTransitionIssue_GetWorkflowStatesError(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if requestCount == 1 {
			// First request: resolve identifier successfully
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [{
							"id": "internal-uuid-123",
							"identifier": "NUM-70",
							"title": "Test Issue",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
						}]
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if requestCount == 2 {
			// Second request: getWorkflowStates returns error
			response := GraphQLResponse{
				Errors: []GraphQLError{{Message: "Team not found"}},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.TransitionIssue(context.Background(), "NUM-70", tracker.StatusInProgress)
	if err == nil {
		t.Fatal("expected error when getWorkflowStates fails, got nil")
	}
}

// TestTransitionIssue_NoMatchingState verifies error handling when no matching
// workflow state is found for the requested status.
func TestTransitionIssue_NoMatchingState(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if requestCount == 1 {
			// Resolve identifier
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [{
							"id": "internal-uuid-123",
							"identifier": "NUM-70",
							"title": "Test Issue",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
						}]
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if requestCount == 2 {
			// Return states that don't match StatusInReview
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issue": {
						"team": {
							"states": {
								"nodes": [
									{"id": "state-1", "name": "Todo", "type": "unstarted"},
									{"id": "state-2", "name": "Done", "type": "completed"}
								]
							}
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.TransitionIssue(context.Background(), "NUM-70", tracker.StatusInReview)
	if err == nil {
		t.Fatal("expected error when no matching state found, got nil")
	}

	if !contains(err.Error(), "no matching workflow state") {
		t.Errorf("expected error containing %q, got %q", "no matching workflow state", err.Error())
	}
}

// TestTransitionIssue_MutationFails verifies error handling when the
// issueUpdate mutation fails.
func TestTransitionIssue_MutationFails(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if requestCount == 1 {
			// Resolve identifier
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [{
							"id": "internal-uuid-123",
							"identifier": "NUM-70",
							"title": "Test Issue",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
						}]
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if requestCount == 2 {
			// Get workflow states
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issue": {
						"team": {
							"states": {
								"nodes": [
									{"id": "state-1", "name": "Todo", "type": "unstarted"},
									{"id": "state-2", "name": "In Progress", "type": "started"}
								]
							}
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if requestCount == 3 {
			// Mutation returns success: false
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueUpdate": {
						"success": false
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.TransitionIssue(context.Background(), "NUM-70", tracker.StatusInProgress)
	if err == nil {
		t.Fatal("expected error when mutation fails, got nil")
	}

	if !contains(err.Error(), "failed to update issue state") {
		t.Errorf("expected error containing %q, got %q", "failed to update issue state", err.Error())
	}
}

// TestGetAvailableStatuses_GetWorkflowStatesError verifies error handling when
// getWorkflowStates fails.
func TestGetAvailableStatuses_GetWorkflowStatesError(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if requestCount == 1 {
			// Resolve identifier successfully
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [{
							"id": "internal-uuid-123",
							"identifier": "NUM-70",
							"title": "Test Issue",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
						}]
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if requestCount == 2 {
			// getWorkflowStates returns error
			response := GraphQLResponse{
				Errors: []GraphQLError{{Message: "Team not found"}},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetAvailableStatuses(context.Background(), "NUM-70")
	if err == nil {
		t.Fatal("expected error when getWorkflowStates fails, got nil")
	}
}

// TestGetAvailableStatuses_DuplicateDetection verifies that GetAvailableStatuses
// properly deduplicates statuses when multiple workflow states map to the same status.
func TestGetAvailableStatuses_DuplicateDetection(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if requestCount == 1 {
			// Resolve identifier
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [{
							"id": "internal-uuid-123",
							"identifier": "NUM-70",
							"title": "Test Issue",
							"state": {"id": "state-1", "name": "Todo", "type": "unstarted"}
						}]
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if requestCount == 2 {
			// Return multiple "started" states that should deduplicate to one StatusInProgress
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issue": {
						"team": {
							"states": {
								"nodes": [
									{"id": "state-1", "name": "Todo", "type": "unstarted"},
									{"id": "state-2", "name": "In Progress", "type": "started"},
									{"id": "state-3", "name": "Working", "type": "started"},
									{"id": "state-4", "name": "In Review", "type": "started"},
									{"id": "state-5", "name": "Done", "type": "completed"}
								]
							}
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	statuses, err := client.GetAvailableStatuses(context.Background(), "NUM-70")
	if err != nil {
		t.Fatalf("GetAvailableStatuses failed: %v", err)
	}

	// Count occurrences of each status to verify deduplication
	statusCount := make(map[tracker.Status]int)
	for _, status := range statuses {
		statusCount[status]++
	}

	// Verify no duplicates
	for status, count := range statusCount {
		if count > 1 {
			t.Errorf("status %v appears %d times, expected 1 (deduplication failed)", status, count)
		}
	}

	// Verify we got the expected unique statuses
	// Note: linearStateToStatus converts based on Type field only, so all "started"
	// states become StatusInProgress regardless of name. InReview is only used in
	// statusMatches for transitions, not in GetAvailableStatuses.
	if len(statuses) != 3 {
		t.Errorf("expected 3 unique statuses (Todo, InProgress, Done), got %d: %v", len(statuses), statuses)
	}

	// Verify deduplication worked for the multiple "started" states
	if statusCount[tracker.StatusInProgress] != 1 {
		t.Errorf("expected StatusInProgress to appear once after deduplication, appeared %d times", statusCount[tracker.StatusInProgress])
	}
}
