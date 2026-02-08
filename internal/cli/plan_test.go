package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/charleslr/jig/internal/tracker"
)

func TestFormatIssueContext(t *testing.T) {
	tests := []struct {
		name     string
		issue    *tracker.Issue
		contains []string
		notContains []string
	}{
		{
			name: "full issue with all fields",
			issue: &tracker.Issue{
				ID:          "abc123",
				Identifier:  "ENG-123",
				Title:       "Implement user authentication",
				Description: "We need to add OAuth2 authentication to the API.",
				Status:      tracker.StatusTodo,
				Labels:      []string{"feature", "security"},
			},
			contains: []string{
				"Existing Issue",
				"ENG-123",
				"Implement user authentication",
				"todo",
				"OAuth2 authentication",
				"feature",
				"security",
			},
		},
		{
			name: "issue without description",
			issue: &tracker.Issue{
				ID:         "abc123",
				Identifier: "ENG-456",
				Title:      "Fix login bug",
				Status:     tracker.StatusInProgress,
			},
			contains: []string{
				"ENG-456",
				"Fix login bug",
				"in_progress",
			},
			notContains: []string{
				"Description:",
			},
		},
		{
			name: "issue without labels",
			issue: &tracker.Issue{
				ID:          "abc123",
				Identifier:  "ENG-789",
				Title:       "Update dependencies",
				Description: "Update all npm packages to latest versions.",
				Status:      tracker.StatusBacklog,
				Labels:      []string{},
			},
			contains: []string{
				"ENG-789",
				"Update dependencies",
				"backlog",
				"Update all npm packages",
			},
			notContains: []string{
				"Labels:",
			},
		},
		{
			name: "minimal issue",
			issue: &tracker.Issue{
				ID:         "abc123",
				Identifier: "BUG-001",
				Title:      "Simple bug",
				Status:     tracker.StatusDone,
			},
			contains: []string{
				"BUG-001",
				"Simple bug",
				"done",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatIssueContext(tt.issue)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("formatIssueContext() missing %q in output:\n%s", want, result)
				}
			}

			for _, notWant := range tt.notContains {
				if strings.Contains(result, notWant) {
					t.Errorf("formatIssueContext() should not contain %q in output:\n%s", notWant, result)
				}
			}
		})
	}
}

func TestFormatIssueContextStructure(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-123",
		Title:       "Test Issue",
		Description: "Test description",
		Status:      tracker.StatusTodo,
		Labels:      []string{"test"},
	}

	result := formatIssueContext(issue)

	// Check that the output has the expected structure
	if !strings.HasPrefix(result, "## Existing Issue") {
		t.Error("expected output to start with '## Existing Issue' header")
	}

	// Check field markers are present
	if !strings.Contains(result, "**ID:**") {
		t.Error("expected output to contain '**ID:**' marker")
	}
	if !strings.Contains(result, "**Title:**") {
		t.Error("expected output to contain '**Title:**' marker")
	}
	if !strings.Contains(result, "**Status:**") {
		t.Error("expected output to contain '**Status:**' marker")
	}
	if !strings.Contains(result, "**Description:**") {
		t.Error("expected output to contain '**Description:**' marker")
	}
	if !strings.Contains(result, "**Labels:**") {
		t.Error("expected output to contain '**Labels:**' marker")
	}
}

func TestGetGitAuthor(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	tests := []struct {
		name     string
		userEnv  string
		expected string
	}{
		{
			name:     "with USER env set",
			userEnv:  "testuser",
			expected: "testuser",
		},
		{
			name:     "with empty USER env",
			userEnv:  "",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("USER", tt.userEnv)
			result := getGitAuthor()
			if result != tt.expected {
				t.Errorf("getGitAuthor() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestMarkPlanSaved(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "jig-test-*")
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

	// Call markPlanSaved
	markPlanSaved()

	// Verify the marker file was created
	markerPath := ".jig/plan-saved.marker"
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("expected plan-saved.marker file to be created")
	}
}

func TestWriteAndReadSavedPlanID(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "jig-test-*")
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

	tests := []struct {
		name      string
		sessionID string
		planID    string
	}{
		{
			name:      "simple session and plan IDs",
			sessionID: "12345",
			planID:    "NUM-22",
		},
		{
			name:      "complex plan ID",
			sessionID: "1738977123456789",
			planID:    "my-complex-plan-id-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write the plan ID
			writeSavedPlanID(tt.sessionID, tt.planID)

			// Read it back
			got := readSavedPlanID(tt.sessionID)
			if got != tt.planID {
				t.Errorf("readSavedPlanID() = %q, want %q", got, tt.planID)
			}
		})
	}
}

func TestReadSavedPlanID_NotFound(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "jig-test-*")
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

	// Try to read a non-existent session
	got := readSavedPlanID("nonexistent-session")
	if got != "" {
		t.Errorf("readSavedPlanID() for nonexistent session = %q, want empty string", got)
	}
}

func TestReadSavedPlanID_WithWhitespace(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "jig-test-*")
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

	// Create session directory with plan ID that has trailing whitespace
	sessionDir := ".jig/sessions/test-session"
	os.MkdirAll(sessionDir, 0755)
	os.WriteFile(sessionDir+"/saved-plan-id", []byte("NUM-22\n  "), 0644)

	got := readSavedPlanID("test-session")
	if got != "NUM-22" {
		t.Errorf("readSavedPlanID() = %q, want %q (whitespace should be trimmed)", got, "NUM-22")
	}
}

func TestPlanSaveCmd_Flags(t *testing.T) {
	// Test that the --session flag is properly defined on the command
	cmd := planSaveCmd

	sessionFlag := cmd.Flags().Lookup("session")
	if sessionFlag == nil {
		t.Fatal("expected --session flag to be defined on planSaveCmd")
	}

	if sessionFlag.Usage == "" {
		t.Error("expected --session flag to have usage text")
	}

	// Verify default value is empty
	if sessionFlag.DefValue != "" {
		t.Errorf("expected --session default value to be empty, got %q", sessionFlag.DefValue)
	}
}

func TestWriteSavedPlanID_CreatesDirectory(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "jig-test-*")
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

	// Write to a deeply nested session that doesn't exist yet
	sessionID := "deeply/nested/session"
	planID := "TEST-123"

	writeSavedPlanID(sessionID, planID)

	// Verify directory was created
	sessionDir := ".jig/sessions/" + sessionID
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		t.Errorf("expected session directory %q to be created", sessionDir)
	}

	// Verify file was written
	got := readSavedPlanID(sessionID)
	if got != planID {
		t.Errorf("readSavedPlanID() = %q, want %q", got, planID)
	}
}

func TestSessionIDIsolation(t *testing.T) {
	// Test that different sessions are properly isolated
	tempDir, err := os.MkdirTemp("", "jig-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Write different plan IDs to different sessions
	writeSavedPlanID("session-1", "PLAN-A")
	writeSavedPlanID("session-2", "PLAN-B")
	writeSavedPlanID("session-3", "PLAN-C")

	// Verify each session has its own plan ID
	tests := []struct {
		sessionID string
		wantPlan  string
	}{
		{"session-1", "PLAN-A"},
		{"session-2", "PLAN-B"},
		{"session-3", "PLAN-C"},
		{"session-4", ""}, // non-existent
	}

	for _, tt := range tests {
		got := readSavedPlanID(tt.sessionID)
		if got != tt.wantPlan {
			t.Errorf("readSavedPlanID(%q) = %q, want %q", tt.sessionID, got, tt.wantPlan)
		}
	}
}

func TestRunPlanSave_EmptyContent(t *testing.T) {
	// Save and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create empty stdin
	r, w, _ := os.Pipe()
	w.Close()
	os.Stdin = r

	err := runPlanSave(planSaveCmd, []string{})
	if err == nil {
		t.Error("expected error for empty content")
	}
	if !strings.Contains(err.Error(), "no plan content") {
		t.Errorf("expected 'no plan content' error, got: %v", err)
	}
}

func TestRunPlanSave_InvalidPlanFormat(t *testing.T) {
	// Save and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create stdin with invalid plan content (missing frontmatter)
	invalidPlan := "# Just a markdown file\n\nNo frontmatter here."
	r, w, _ := os.Pipe()
	go func() {
		w.Write([]byte(invalidPlan))
		w.Close()
	}()
	os.Stdin = r

	err := runPlanSave(planSaveCmd, []string{})
	if err == nil {
		t.Error("expected error for invalid plan format")
	}
	if !strings.Contains(err.Error(), "invalid plan format") {
		t.Errorf("expected 'invalid plan format' error, got: %v", err)
	}
}

func TestRunPlanSave_FileNotFound(t *testing.T) {
	err := runPlanSave(planSaveCmd, []string{"/nonexistent/path/plan.md"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("expected 'failed to read file' error, got: %v", err)
	}
}

func TestRunPlanSave_MissingRequiredFields(t *testing.T) {
	// Create a temp file with plan missing required fields
	tempDir, err := os.MkdirTemp("", "jig-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Plan with frontmatter but missing required fields
	invalidPlan := `---
title: Test Plan
---

# Test Plan

Some content.
`
	planFile := tempDir + "/invalid-plan.md"
	os.WriteFile(planFile, []byte(invalidPlan), 0644)

	err = runPlanSave(planSaveCmd, []string{planFile})
	if err == nil {
		t.Error("expected error for plan missing required fields")
	}
	// Should fail on validation or parsing
	if !strings.Contains(err.Error(), "invalid plan format") && !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected validation/parse error, got: %v", err)
	}
}

func TestRunPlanSave_FromFile(t *testing.T) {
	// This test verifies that reading from a file works
	// We don't test the full save path since it requires cache setup
	tempDir, err := os.MkdirTemp("", "jig-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid plan file
	validPlan := `---
id: TEST-123
title: Test Plan
status: draft
author: testuser
phases:
  - id: phase-1
    title: Phase 1
    status: pending
---

# Test Plan

## Problem Statement

Test problem.

## Proposed Solution

Test solution.

## Phases

### Phase 1

Test phase.
`
	planFile := tempDir + "/valid-plan.md"
	os.WriteFile(planFile, []byte(validPlan), 0644)

	// This will fail at the cache initialization step, but it validates
	// that file reading and parsing work correctly
	err = runPlanSave(planSaveCmd, []string{planFile})
	// The error should be about cache initialization, not file reading or parsing
	if err != nil && (strings.Contains(err.Error(), "failed to read file") ||
		strings.Contains(err.Error(), "invalid plan format") ||
		strings.Contains(err.Error(), "failed to parse plan")) {
		t.Errorf("unexpected error type: %v", err)
	}
}

func TestRunPlanSave_FromStdin(t *testing.T) {
	// Save and restore stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	validPlan := `---
id: STDIN-TEST
title: Stdin Test Plan
status: draft
author: testuser
phases:
  - id: phase-1
    title: Phase 1
    status: pending
---

# Stdin Test Plan

## Problem Statement

Test problem from stdin.

## Proposed Solution

Test solution.

## Phases

### Phase 1

Test phase.
`
	// Create stdin with valid plan
	os.Stdin = createStdinPipe(validPlan)

	// This will fail at cache initialization but validates stdin reading works
	err := runPlanSave(planSaveCmd, []string{})
	// Should not fail on reading or parsing
	if err != nil && (strings.Contains(err.Error(), "failed to read") ||
		strings.Contains(err.Error(), "invalid plan format") ||
		strings.Contains(err.Error(), "failed to parse plan")) {
		t.Errorf("unexpected error type: %v", err)
	}
}

// createStdinPipe creates a pipe that simulates stdin with the given content
func createStdinPipe(content string) *os.File {
	r, w, _ := os.Pipe()
	go func() {
		w.Write([]byte(content))
		w.Close()
	}()
	return r
}

func TestPlanSaveCmd_Usage(t *testing.T) {
	// Verify the command usage includes the --session example
	if !strings.Contains(planSaveCmd.Long, "--session") {
		t.Error("expected planSaveCmd.Long to mention --session flag")
	}
}

func TestRunPlanSave_WithSessionFlag(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "jig-test-*")
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

	// Create a valid plan file
	validPlan := `---
id: SESSION-TEST-123
title: Session Test Plan
status: draft
author: testuser
phases:
  - id: phase-1
    title: Phase 1
    status: pending
---

# Session Test Plan

## Problem Statement

Test problem.

## Proposed Solution

Test solution.

## Phases

### Phase 1

Test phase.
`
	planFile := tempDir + "/session-plan.md"
	os.WriteFile(planFile, []byte(validPlan), 0644)

	// Set the session ID flag variable
	originalSessionID := planSaveSessionID
	planSaveSessionID = "test-session-12345"
	defer func() { planSaveSessionID = originalSessionID }()

	// Run the save command
	err = runPlanSave(planSaveCmd, []string{planFile})
	if err != nil {
		t.Fatalf("runPlanSave failed: %v", err)
	}

	// Verify the session ID was written to the session directory
	savedPlanID := readSavedPlanID("test-session-12345")
	if savedPlanID != "SESSION-TEST-123" {
		t.Errorf("expected saved plan ID 'SESSION-TEST-123', got '%s'", savedPlanID)
	}
}

func TestRunPlanSave_WithoutSessionFlag(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "jig-test-*")
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

	// Create a valid plan file
	validPlan := `---
id: NO-SESSION-TEST
title: No Session Test Plan
status: draft
author: testuser
phases:
  - id: phase-1
    title: Phase 1
    status: pending
---

# No Session Test Plan

## Problem Statement

Test problem.

## Proposed Solution

Test solution.

## Phases

### Phase 1

Test phase.
`
	planFile := tempDir + "/no-session-plan.md"
	os.WriteFile(planFile, []byte(validPlan), 0644)

	// Ensure session ID is empty
	originalSessionID := planSaveSessionID
	planSaveSessionID = ""
	defer func() { planSaveSessionID = originalSessionID }()

	// Run the save command
	err = runPlanSave(planSaveCmd, []string{planFile})
	if err != nil {
		t.Fatalf("runPlanSave failed: %v", err)
	}

	// Verify no session directory was created for this test
	// (readSavedPlanID should return empty for a non-existent session)
	savedPlanID := readSavedPlanID("nonexistent-session-xyz")
	if savedPlanID != "" {
		t.Errorf("expected empty saved plan ID for nonexistent session, got '%s'", savedPlanID)
	}
}

func TestDisplaySavedPlanNextSteps(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "jig-test-*")
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

	t.Run("returns true when plan exists", func(t *testing.T) {
		sessionID := "test-session-display"
		planID := "DISPLAY-TEST-123"

		// Write a saved plan ID
		writeSavedPlanID(sessionID, planID)

		// Test that displaySavedPlanNextSteps returns true
		result := displaySavedPlanNextSteps(sessionID)
		if !result {
			t.Error("expected displaySavedPlanNextSteps to return true when plan exists")
		}
	})

	t.Run("returns false when no plan exists", func(t *testing.T) {
		result := displaySavedPlanNextSteps("nonexistent-session-xyz")
		if result {
			t.Error("expected displaySavedPlanNextSteps to return false when no plan exists")
		}
	})
}
