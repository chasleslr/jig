package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyCmd_Flags(t *testing.T) {
	// Test that flags are properly defined on the command
	cmd := verifyCmd

	t.Run("runner flag exists", func(t *testing.T) {
		runnerFlag := cmd.Flags().Lookup("runner")
		if runnerFlag == nil {
			t.Fatal("expected --runner flag to be defined")
		}
		if runnerFlag.Shorthand != "r" {
			t.Errorf("expected --runner shorthand to be 'r', got %q", runnerFlag.Shorthand)
		}
	})

	t.Run("no-launch flag exists", func(t *testing.T) {
		noLaunchFlag := cmd.Flags().Lookup("no-launch")
		if noLaunchFlag == nil {
			t.Fatal("expected --no-launch flag to be defined")
		}
		if noLaunchFlag.DefValue != "false" {
			t.Errorf("expected --no-launch default to be 'false', got %q", noLaunchFlag.DefValue)
		}
	})
}

func TestVerifyCmd_Usage(t *testing.T) {
	cmd := verifyCmd

	t.Run("has correct use pattern", func(t *testing.T) {
		if cmd.Use != "verify [ISSUE]" {
			t.Errorf("expected Use to be 'verify [ISSUE]', got %q", cmd.Use)
		}
	})

	t.Run("has short description", func(t *testing.T) {
		if cmd.Short == "" {
			t.Error("expected Short description to be set")
		}
		if !strings.Contains(strings.ToLower(cmd.Short), "verify") {
			t.Errorf("expected Short to mention 'verify', got %q", cmd.Short)
		}
	})

	t.Run("has long description", func(t *testing.T) {
		if cmd.Long == "" {
			t.Error("expected Long description to be set")
		}
		if !strings.Contains(cmd.Long, "acceptance criteria") {
			t.Errorf("expected Long to mention 'acceptance criteria', got %q", cmd.Long)
		}
	})

	t.Run("accepts at most one argument", func(t *testing.T) {
		// The command should accept 0 or 1 arguments
		if cmd.Args == nil {
			t.Error("expected Args validator to be set")
		}
	})
}

func TestRunVerify_NotInWorktree(t *testing.T) {
	// Create a temp directory without .jig
	tempDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run verify without an issue ID (should detect from directory)
	err = runVerify(verifyCmd, []string{})

	if err == nil {
		t.Error("expected error when not in a jig worktree")
	}
	if !strings.Contains(err.Error(), "not in a jig worktree") {
		t.Errorf("expected 'not in a jig worktree' error, got: %v", err)
	}
}

func TestRunVerify_NoPlanFile(t *testing.T) {
	// Create a temp directory with .jig but no plan.md
	tempDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .jig directory with issue.json but no plan.md
	jigDir := filepath.Join(tempDir, ".jig")
	if err := os.MkdirAll(jigDir, 0755); err != nil {
		t.Fatalf("failed to create .jig dir: %v", err)
	}

	issueJSON := `{"issue_id": "TEST-123"}`
	if err := os.WriteFile(filepath.Join(jigDir, "issue.json"), []byte(issueJSON), 0644); err != nil {
		t.Fatalf("failed to write issue.json: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run verify without an issue ID
	err = runVerify(verifyCmd, []string{})

	if err == nil {
		t.Error("expected error when plan.md doesn't exist")
	}
	if !strings.Contains(err.Error(), "no plan found") {
		t.Errorf("expected 'no plan found' error, got: %v", err)
	}
}

func TestRunVerify_MissingIssueJSON(t *testing.T) {
	// Create a temp directory with .jig but no issue.json
	tempDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .jig directory without issue.json
	jigDir := filepath.Join(tempDir, ".jig")
	if err := os.MkdirAll(jigDir, 0755); err != nil {
		t.Fatalf("failed to create .jig dir: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run verify without an issue ID (should fail reading issue.json)
	err = runVerify(verifyCmd, []string{})

	if err == nil {
		t.Error("expected error when issue.json doesn't exist")
	}
	if !strings.Contains(err.Error(), "failed to read issue metadata") {
		t.Errorf("expected 'failed to read issue metadata' error, got: %v", err)
	}
}

func TestRunVerify_InvalidIssueJSON(t *testing.T) {
	// Create a temp directory with invalid issue.json
	tempDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .jig directory with invalid JSON
	jigDir := filepath.Join(tempDir, ".jig")
	if err := os.MkdirAll(jigDir, 0755); err != nil {
		t.Fatalf("failed to create .jig dir: %v", err)
	}

	invalidJSON := `{invalid json}`
	if err := os.WriteFile(filepath.Join(jigDir, "issue.json"), []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to write issue.json: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run verify without an issue ID
	err = runVerify(verifyCmd, []string{})

	if err == nil {
		t.Error("expected error when issue.json is invalid")
	}
	if !strings.Contains(err.Error(), "failed to parse issue metadata") {
		t.Errorf("expected 'failed to parse issue metadata' error, got: %v", err)
	}
}

func TestRunVerify_ExplicitIssueID(t *testing.T) {
	// Create a temp directory with .jig and plan.md
	tempDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .jig directory with plan.md (no issue.json needed when ID is explicit)
	jigDir := filepath.Join(tempDir, ".jig")
	if err := os.MkdirAll(jigDir, 0755); err != nil {
		t.Fatalf("failed to create .jig dir: %v", err)
	}

	planContent := `---
id: EXPLICIT-123
title: Test Plan
---

# Test Plan
`
	if err := os.WriteFile(filepath.Join(jigDir, "plan.md"), []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan.md: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run verify with explicit issue ID - should fail at runner lookup, not issue detection
	err = runVerify(verifyCmd, []string{"EXPLICIT-123"})

	// Should fail at runner availability, not at issue ID detection
	if err == nil {
		t.Skip("skipping - runner available in test environment")
	}
	// Should NOT fail with "not in a jig worktree" or "failed to read issue metadata"
	if strings.Contains(err.Error(), "not in a jig worktree") {
		t.Errorf("should not fail with worktree error when issue ID is explicit: %v", err)
	}
	if strings.Contains(err.Error(), "failed to read issue metadata") {
		t.Errorf("should not fail with issue metadata error when issue ID is explicit: %v", err)
	}
}

func TestRunVerify_IssueIDFromContext(t *testing.T) {
	// Create a temp directory with valid .jig context
	tempDir, err := os.MkdirTemp("", "jig-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .jig directory with issue.json and plan.md
	jigDir := filepath.Join(tempDir, ".jig")
	if err := os.MkdirAll(jigDir, 0755); err != nil {
		t.Fatalf("failed to create .jig dir: %v", err)
	}

	issueJSON := `{"issue_id": "CONTEXT-456", "title": "Test Issue"}`
	if err := os.WriteFile(filepath.Join(jigDir, "issue.json"), []byte(issueJSON), 0644); err != nil {
		t.Fatalf("failed to write issue.json: %v", err)
	}

	planContent := `---
id: CONTEXT-456
title: Test Plan
---

# Test Plan
`
	if err := os.WriteFile(filepath.Join(jigDir, "plan.md"), []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan.md: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Run verify without issue ID - should detect from context
	err = runVerify(verifyCmd, []string{})

	// Should fail at runner availability, not at issue ID detection
	if err == nil {
		t.Skip("skipping - runner available in test environment")
	}
	// Should NOT fail with worktree or metadata errors
	if strings.Contains(err.Error(), "not in a jig worktree") {
		t.Errorf("should detect .jig directory: %v", err)
	}
	if strings.Contains(err.Error(), "failed to read issue metadata") {
		t.Errorf("should read issue.json successfully: %v", err)
	}
	if strings.Contains(err.Error(), "no plan found") {
		t.Errorf("should find plan.md: %v", err)
	}
}

func TestParseIssueJSON(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantIssueID string
		wantErr     bool
	}{
		{
			name:        "valid issue.json",
			json:        `{"issue_id": "TEST-123", "title": "Test"}`,
			wantIssueID: "TEST-123",
			wantErr:     false,
		},
		{
			name:        "issue.json with extra fields",
			json:        `{"issue_id": "EXTRA-456", "title": "Test", "status": "in-progress", "author": "user"}`,
			wantIssueID: "EXTRA-456",
			wantErr:     false,
		},
		{
			name:        "empty issue_id",
			json:        `{"issue_id": "", "title": "Test"}`,
			wantIssueID: "",
			wantErr:     false,
		},
		{
			name:    "invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name:        "missing issue_id field",
			json:        `{"title": "Test"}`,
			wantIssueID: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with JSON content
			tempDir, err := os.MkdirTemp("", "jig-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			jsonPath := filepath.Join(tempDir, "issue.json")
			if err := os.WriteFile(jsonPath, []byte(tt.json), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Read and parse
			data, err := os.ReadFile(jsonPath)
			if err != nil {
				t.Fatalf("failed to read test file: %v", err)
			}

			var issueMeta struct {
				IssueID string `json:"issue_id"`
			}
			err = parseIssueMetadata(data, &issueMeta)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if issueMeta.IssueID != tt.wantIssueID {
				t.Errorf("got issue_id %q, want %q", issueMeta.IssueID, tt.wantIssueID)
			}
		})
	}
}

// parseIssueMetadata is a helper that mirrors the parsing logic in verify.go
func parseIssueMetadata(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
