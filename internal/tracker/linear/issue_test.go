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
