package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTeamLabels(t *testing.T) {
	t.Run("returns labels from team", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"team": {
						"labels": {
							"nodes": [
								{"id": "label-1", "name": "bug"},
								{"id": "label-2", "name": "feature"},
								{"id": "label-3", "name": "jig-plan"}
							]
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		labels, err := client.GetTeamLabels(context.Background(), "team-123")
		if err != nil {
			t.Fatalf("GetTeamLabels failed: %v", err)
		}

		if len(labels) != 3 {
			t.Errorf("expected 3 labels, got %d", len(labels))
		}

		expectedLabels := []struct {
			id   string
			name string
		}{
			{"label-1", "bug"},
			{"label-2", "feature"},
			{"label-3", "jig-plan"},
		}

		for i, expected := range expectedLabels {
			if labels[i].ID != expected.id {
				t.Errorf("labels[%d].ID = %q, want %q", i, labels[i].ID, expected.id)
			}
			if labels[i].Name != expected.name {
				t.Errorf("labels[%d].Name = %q, want %q", i, labels[i].Name, expected.name)
			}
		}
	})

	t.Run("handles empty labels", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"team": {
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
		labels, err := client.GetTeamLabels(context.Background(), "team-123")
		if err != nil {
			t.Fatalf("GetTeamLabels failed: %v", err)
		}

		if len(labels) != 0 {
			t.Errorf("expected 0 labels, got %d", len(labels))
		}
	})
}

func TestCreateLabel(t *testing.T) {
	t.Run("creates label successfully", func(t *testing.T) {
		var capturedRequest GraphQLRequest

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&capturedRequest)
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueLabelCreate": {
						"success": true,
						"issueLabel": {
							"id": "new-label-id",
							"name": "jig-plan"
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		label, err := client.CreateLabel(context.Background(), "team-123", "jig-plan")
		if err != nil {
			t.Fatalf("CreateLabel failed: %v", err)
		}

		if label.ID != "new-label-id" {
			t.Errorf("label.ID = %q, want %q", label.ID, "new-label-id")
		}
		if label.Name != "jig-plan" {
			t.Errorf("label.Name = %q, want %q", label.Name, "jig-plan")
		}

		// Verify the request payload
		input, ok := capturedRequest.Variables["input"].(map[string]interface{})
		if !ok {
			t.Fatal("expected input in variables")
		}
		if input["teamId"] != "team-123" {
			t.Errorf("input.teamId = %v, want %q", input["teamId"], "team-123")
		}
		if input["name"] != "jig-plan" {
			t.Errorf("input.name = %v, want %q", input["name"], "jig-plan")
		}
	})

	t.Run("returns error on failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueLabelCreate": {
						"success": false,
						"issueLabel": null
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		_, err := client.CreateLabel(context.Background(), "team-123", "jig-plan")
		if err == nil {
			t.Fatal("expected error when label creation fails")
		}
	})
}

func TestGetOrCreateLabel(t *testing.T) {
	t.Run("returns existing label if found", func(t *testing.T) {
		callCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			// Only the GetTeamLabels call should be made
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"team": {
						"labels": {
							"nodes": [
								{"id": "existing-label-id", "name": "jig-plan"},
								{"id": "other-label", "name": "bug"}
							]
						}
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		label, err := client.GetOrCreateLabel(context.Background(), "team-123", "jig-plan")
		if err != nil {
			t.Fatalf("GetOrCreateLabel failed: %v", err)
		}

		if label.ID != "existing-label-id" {
			t.Errorf("label.ID = %q, want %q", label.ID, "existing-label-id")
		}
		if label.Name != "jig-plan" {
			t.Errorf("label.Name = %q, want %q", label.Name, "jig-plan")
		}

		// Should only call GetTeamLabels, not CreateLabel
		if callCount != 1 {
			t.Errorf("expected 1 API call, got %d", callCount)
		}
	})

	t.Run("creates label if not found", func(t *testing.T) {
		callCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			var req GraphQLRequest
			json.NewDecoder(r.Body).Decode(&req)

			if callCount == 1 {
				// First call: GetTeamLabels - return empty
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"team": {
							"labels": {
								"nodes": [
									{"id": "other-label", "name": "bug"}
								]
							}
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
			} else {
				// Second call: CreateLabel
				response := GraphQLResponse{
					Data: json.RawMessage(`{
						"issueLabelCreate": {
							"success": true,
							"issueLabel": {
								"id": "new-label-id",
								"name": "jig-plan"
							}
						}
					}`),
				}
				json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		label, err := client.GetOrCreateLabel(context.Background(), "team-123", "jig-plan")
		if err != nil {
			t.Fatalf("GetOrCreateLabel failed: %v", err)
		}

		if label.ID != "new-label-id" {
			t.Errorf("label.ID = %q, want %q", label.ID, "new-label-id")
		}

		// Should call both GetTeamLabels and CreateLabel
		if callCount != 2 {
			t.Errorf("expected 2 API calls, got %d", callCount)
		}
	})
}

func TestAddLabelToIssue(t *testing.T) {
	t.Run("adds label when not already present", func(t *testing.T) {
		var capturedRequest GraphQLRequest

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&capturedRequest)
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueUpdate": {
						"success": true
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		existingLabelIDs := []string{"label-1", "label-2"}
		err := client.AddLabelToIssue(context.Background(), "issue-123", "new-label", existingLabelIDs)
		if err != nil {
			t.Fatalf("AddLabelToIssue failed: %v", err)
		}

		// Verify the request includes all labels
		input, ok := capturedRequest.Variables["input"].(map[string]interface{})
		if !ok {
			t.Fatal("expected input in variables")
		}
		labelIDs, ok := input["labelIds"].([]interface{})
		if !ok {
			t.Fatal("expected labelIds in input")
		}
		if len(labelIDs) != 3 {
			t.Errorf("expected 3 labelIds, got %d", len(labelIDs))
		}
	})

	t.Run("skips when label already present", func(t *testing.T) {
		apiCalled := false

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiCalled = true
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueUpdate": {
						"success": true
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		existingLabelIDs := []string{"label-1", "existing-label", "label-2"}
		err := client.AddLabelToIssue(context.Background(), "issue-123", "existing-label", existingLabelIDs)
		if err != nil {
			t.Fatalf("AddLabelToIssue failed: %v", err)
		}

		// Should not call API since label is already present
		if apiCalled {
			t.Error("expected no API call when label already present")
		}
	})

	t.Run("returns error on failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueUpdate": {
						"success": false
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		err := client.AddLabelToIssue(context.Background(), "issue-123", "new-label", []string{})
		if err == nil {
			t.Fatal("expected error when update fails")
		}
	})

	t.Run("adds to empty existing labels", func(t *testing.T) {
		var capturedRequest GraphQLRequest

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&capturedRequest)
			response := GraphQLResponse{
				Data: json.RawMessage(`{
					"issueUpdate": {
						"success": true
					}
				}`),
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := newTestClient(server.URL)
		err := client.AddLabelToIssue(context.Background(), "issue-123", "first-label", []string{})
		if err != nil {
			t.Fatalf("AddLabelToIssue failed: %v", err)
		}

		// Verify the request includes only the new label
		input, ok := capturedRequest.Variables["input"].(map[string]interface{})
		if !ok {
			t.Fatal("expected input in variables")
		}
		labelIDs, ok := input["labelIds"].([]interface{})
		if !ok {
			t.Fatal("expected labelIds in input")
		}
		if len(labelIDs) != 1 {
			t.Errorf("expected 1 labelId, got %d", len(labelIDs))
		}
		if labelIDs[0] != "first-label" {
			t.Errorf("expected labelIds[0] = %q, got %v", "first-label", labelIDs[0])
		}
	})
}
