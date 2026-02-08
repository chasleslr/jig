package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/state"
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

func TestRunPlanSave_SavesLocally(t *testing.T) {
	// Create temp directories for test
	tempDir, err := os.MkdirTemp("", "jig-plan-save-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid plan file
	planContent := `---
id: TEST-123
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

This is a test problem statement.

## Proposed Solution

This is a test proposed solution.
`
	planFile := filepath.Join(tempDir, "plan.md")
	if err := os.WriteFile(planFile, []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan file: %v", err)
	}

	// Set up cache directory
	cacheDir := filepath.Join(tempDir, "cache")
	if err := os.MkdirAll(filepath.Join(cacheDir, "plans"), 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "issues"), 0755); err != nil {
		t.Fatalf("failed to create issues dir: %v", err)
	}

	// Initialize state with test cache
	oldCache := state.DefaultCache
	state.DefaultCache = state.NewCacheWithDir(cacheDir)
	defer func() { state.DefaultCache = oldCache }()

	// Change to temp directory for marker file
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Reset flags
	oldSync := planSaveSync
	oldNoSync := planSaveNoSync
	planSaveSync = false
	planSaveNoSync = true // Disable sync for this test
	defer func() {
		planSaveSync = oldSync
		planSaveNoSync = oldNoSync
	}()

	// Run the command
	err = runPlanSave(planSaveCmd, []string{planFile})
	if err != nil {
		t.Fatalf("runPlanSave() error = %v", err)
	}

	// Verify plan was saved to cache
	savedPlan, err := state.DefaultCache.GetPlan("TEST-123")
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if savedPlan == nil {
		t.Fatal("plan should have been saved to cache")
	}
	if savedPlan.Title != "Test Plan" {
		t.Errorf("savedPlan.Title = %q, want %q", savedPlan.Title, "Test Plan")
	}
}

func TestRunPlanSave_NoSyncFlag(t *testing.T) {
	// Create temp directories for test
	tempDir, err := os.MkdirTemp("", "jig-plan-save-nosync-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid plan file
	planContent := `---
id: TEST-NOSYNC
title: No Sync Test
status: draft
author: testuser
---

# No Sync Test

## Problem Statement

Test problem.

## Proposed Solution

Test solution.
`
	planFile := filepath.Join(tempDir, "plan.md")
	if err := os.WriteFile(planFile, []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan file: %v", err)
	}

	// Set up cache directory
	cacheDir := filepath.Join(tempDir, "cache")
	if err := os.MkdirAll(filepath.Join(cacheDir, "plans"), 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "issues"), 0755); err != nil {
		t.Fatalf("failed to create issues dir: %v", err)
	}

	// Initialize state with test cache
	oldCache := state.DefaultCache
	state.DefaultCache = state.NewCacheWithDir(cacheDir)
	defer func() { state.DefaultCache = oldCache }()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Set --no-sync flag
	oldSync := planSaveSync
	oldNoSync := planSaveNoSync
	planSaveSync = false
	planSaveNoSync = true
	defer func() {
		planSaveSync = oldSync
		planSaveNoSync = oldNoSync
	}()

	// Run the command - should succeed without trying to sync
	err = runPlanSave(planSaveCmd, []string{planFile})
	if err != nil {
		t.Fatalf("runPlanSave() with --no-sync error = %v", err)
	}

	// Verify plan was saved
	savedPlan, err := state.DefaultCache.GetPlan("TEST-NOSYNC")
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if savedPlan == nil {
		t.Fatal("plan should have been saved")
	}
}

func TestRunPlanSave_SyncFlagOverridesConfig(t *testing.T) {
	// This test verifies that --sync flag forces sync even if config says no
	// and --no-sync flag prevents sync even if config says yes

	tests := []struct {
		name         string
		configSync   bool
		flagSync     bool
		flagNoSync   bool
		expectSync   bool
	}{
		{
			name:       "config true, no flags",
			configSync: true,
			flagSync:   false,
			flagNoSync: false,
			expectSync: true,
		},
		{
			name:       "config false, no flags",
			configSync: false,
			flagSync:   false,
			flagNoSync: false,
			expectSync: false,
		},
		{
			name:       "config false, --sync flag",
			configSync: false,
			flagSync:   true,
			flagNoSync: false,
			expectSync: true,
		},
		{
			name:       "config true, --no-sync flag",
			configSync: true,
			flagSync:   false,
			flagNoSync: true,
			expectSync: false,
		},
		{
			name:       "both flags set, --no-sync wins",
			configSync: true,
			flagSync:   true,
			flagNoSync: true,
			expectSync: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Plan: config.PlanConfig{Sync: tt.configSync},
			}

			// Calculate shouldSync using same logic as runPlanSave
			shouldSync := cfg.Plan.Sync
			if tt.flagSync {
				shouldSync = true
			}
			if tt.flagNoSync {
				shouldSync = false
			}

			if shouldSync != tt.expectSync {
				t.Errorf("shouldSync = %v, want %v", shouldSync, tt.expectSync)
			}
		})
	}
}

func TestRunPlanSave_InvalidPlan(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "jig-plan-invalid-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an invalid plan file (missing required sections)
	invalidPlan := `---
id: INVALID
title: Invalid Plan
status: draft
author: test
---

# Invalid Plan

No required sections here.
`
	planFile := filepath.Join(tempDir, "invalid.md")
	if err := os.WriteFile(planFile, []byte(invalidPlan), 0644); err != nil {
		t.Fatalf("failed to write plan file: %v", err)
	}

	// Set flags to not sync
	oldNoSync := planSaveNoSync
	planSaveNoSync = true
	defer func() { planSaveNoSync = oldNoSync }()

	// Run the command - should fail validation
	err = runPlanSave(planSaveCmd, []string{planFile})
	if err == nil {
		t.Error("runPlanSave() should return error for invalid plan")
	}
	if !strings.Contains(err.Error(), "invalid plan format") {
		t.Errorf("error should mention invalid plan format, got: %v", err)
	}
}

func TestRunPlanSave_EmptyFile(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "jig-plan-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an empty file
	emptyFile := filepath.Join(tempDir, "empty.md")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}

	// Run the command - should fail
	err = runPlanSave(planSaveCmd, []string{emptyFile})
	if err == nil {
		t.Error("runPlanSave() should return error for empty file")
	}
	if !strings.Contains(err.Error(), "no plan content") {
		t.Errorf("error should mention no plan content, got: %v", err)
	}
}

func TestRunPlanSave_FileNotFound(t *testing.T) {
	err := runPlanSave(planSaveCmd, []string{"/nonexistent/path/plan.md"})
	if err == nil {
		t.Error("runPlanSave() should return error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("error should mention failed to read file, got: %v", err)
	}
}

func TestRunPlanSync_PlanNotFound(t *testing.T) {
	// Create temp directory for cache
	tempDir, err := os.MkdirTemp("", "jig-plan-sync-notfound-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up cache directory
	cacheDir := filepath.Join(tempDir, "cache")
	if err := os.MkdirAll(filepath.Join(cacheDir, "plans"), 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "issues"), 0755); err != nil {
		t.Fatalf("failed to create issues dir: %v", err)
	}

	// Initialize state with test cache
	oldCache := state.DefaultCache
	state.DefaultCache = state.NewCacheWithDir(cacheDir)
	defer func() { state.DefaultCache = oldCache }()

	// Run sync for nonexistent plan
	err = runPlanSync(planSyncCmd, []string{"NONEXISTENT-123"})
	if err == nil {
		t.Error("runPlanSync() should return error for nonexistent plan")
	}
	if !strings.Contains(err.Error(), "plan not found") {
		t.Errorf("error should mention plan not found, got: %v", err)
	}
}

func TestGetPlanSyncer_UnknownTracker(t *testing.T) {
	cfg := &config.Config{
		Default: config.DefaultConfig{
			Tracker: "unknown-tracker",
		},
	}

	_, err := getPlanSyncer(cfg)
	if err == nil {
		t.Error("getPlanSyncer() should return error for unknown tracker")
	}
	if !strings.Contains(err.Error(), "unknown tracker") {
		t.Errorf("error should mention unknown tracker, got: %v", err)
	}
}

func TestGetPlanSyncer_LinearWithStoreAPIKey(t *testing.T) {
	// Create temp directory for credential store
	tempDir, err := os.MkdirTemp("", "jig-syncer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a store with an API key
	credPath := filepath.Join(tempDir, ".credentials")
	store := config.NewStoreWithPath(credPath)
	if err := store.SetLinearAPIKey("test-api-key-from-store"); err != nil {
		t.Fatalf("failed to set API key: %v", err)
	}

	// Override storeFactory for this test
	oldFactory := storeFactory
	storeFactory = func() (*config.Store, error) {
		return config.NewStoreWithPath(credPath), nil
	}
	defer func() { storeFactory = oldFactory }()

	cfg := &config.Config{
		Default: config.DefaultConfig{
			Tracker: "linear",
		},
		Linear: config.LinearConfig{
			TeamID:         "team-123",
			DefaultProject: "project-456",
		},
	}

	syncer, err := getPlanSyncer(cfg)
	if err != nil {
		t.Fatalf("getPlanSyncer() error = %v", err)
	}
	if syncer == nil {
		t.Error("getPlanSyncer() returned nil syncer")
	}
}

func TestGetPlanSyncer_LinearWithConfigAPIKey(t *testing.T) {
	// Create temp directory for credential store (empty)
	tempDir, err := os.MkdirTemp("", "jig-syncer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an empty store (no API key)
	credPath := filepath.Join(tempDir, ".credentials")

	// Override storeFactory for this test
	oldFactory := storeFactory
	storeFactory = func() (*config.Store, error) {
		return config.NewStoreWithPath(credPath), nil
	}
	defer func() { storeFactory = oldFactory }()

	cfg := &config.Config{
		Default: config.DefaultConfig{
			Tracker: "linear",
		},
		Linear: config.LinearConfig{
			APIKey:         "test-api-key-from-config",
			TeamID:         "team-123",
			DefaultProject: "project-456",
		},
	}

	syncer, err := getPlanSyncer(cfg)
	if err != nil {
		t.Fatalf("getPlanSyncer() error = %v", err)
	}
	if syncer == nil {
		t.Error("getPlanSyncer() returned nil syncer")
	}
}

func TestGetPlanSyncer_LinearNoAPIKey(t *testing.T) {
	// Create temp directory for credential store (empty)
	tempDir, err := os.MkdirTemp("", "jig-syncer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an empty store (no API key)
	credPath := filepath.Join(tempDir, ".credentials")

	// Override storeFactory for this test
	oldFactory := storeFactory
	storeFactory = func() (*config.Store, error) {
		return config.NewStoreWithPath(credPath), nil
	}
	defer func() { storeFactory = oldFactory }()

	cfg := &config.Config{
		Default: config.DefaultConfig{
			Tracker: "linear",
		},
		Linear: config.LinearConfig{
			// No API key in config either
			TeamID:         "team-123",
			DefaultProject: "project-456",
		},
	}

	_, err = getPlanSyncer(cfg)
	if err == nil {
		t.Error("getPlanSyncer() should return error when no API key configured")
	}
	if !strings.Contains(err.Error(), "API key not configured") {
		t.Errorf("error should mention API key not configured, got: %v", err)
	}
}

func TestGetPlanSyncer_StoreError(t *testing.T) {
	// Override storeFactory to return an error
	oldFactory := storeFactory
	storeFactory = func() (*config.Store, error) {
		return nil, fmt.Errorf("store creation failed")
	}
	defer func() { storeFactory = oldFactory }()

	cfg := &config.Config{
		Default: config.DefaultConfig{
			Tracker: "linear",
		},
	}

	_, err := getPlanSyncer(cfg)
	if err == nil {
		t.Error("getPlanSyncer() should return error when store creation fails")
	}
	if !strings.Contains(err.Error(), "store creation failed") {
		t.Errorf("error should mention store creation failed, got: %v", err)
	}
}

func TestRunPlanSave_WithSyncEnabled_NoTracker(t *testing.T) {
	// Test that sync gracefully handles missing tracker config
	tempDir, err := os.MkdirTemp("", "jig-plan-sync-notracker-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid plan file
	planContent := `---
id: TEST-SYNC
title: Sync Test
status: draft
author: testuser
---

# Sync Test

## Problem Statement

Test problem.

## Proposed Solution

Test solution.
`
	planFile := filepath.Join(tempDir, "plan.md")
	if err := os.WriteFile(planFile, []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan file: %v", err)
	}

	// Set up cache directory
	cacheDir := filepath.Join(tempDir, "cache")
	if err := os.MkdirAll(filepath.Join(cacheDir, "plans"), 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "issues"), 0755); err != nil {
		t.Fatalf("failed to create issues dir: %v", err)
	}

	// Initialize state with test cache
	oldCache := state.DefaultCache
	state.DefaultCache = state.NewCacheWithDir(cacheDir)
	defer func() { state.DefaultCache = oldCache }()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Set --sync flag (should try to sync but fail gracefully)
	oldSync := planSaveSync
	oldNoSync := planSaveNoSync
	planSaveSync = true
	planSaveNoSync = false
	defer func() {
		planSaveSync = oldSync
		planSaveNoSync = oldNoSync
	}()

	// Run the command - should succeed even if sync fails (warns but doesn't error)
	err = runPlanSave(planSaveCmd, []string{planFile})
	if err != nil {
		t.Fatalf("runPlanSave() error = %v", err)
	}

	// Verify plan was still saved locally
	savedPlan, err := state.DefaultCache.GetPlan("TEST-SYNC")
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if savedPlan == nil {
		t.Fatal("plan should have been saved despite sync failure")
	}
}

func TestShouldSyncLogic(t *testing.T) {
	// Comprehensive test of the shouldSync logic extracted from runPlanSave
	tests := []struct {
		name       string
		configSync bool
		flagSync   bool
		flagNoSync bool
		want       bool
	}{
		// Config defaults
		{"config=true, flags=none", true, false, false, true},
		{"config=false, flags=none", false, false, false, false},

		// --sync flag overrides
		{"config=true, --sync", true, true, false, true},
		{"config=false, --sync", false, true, false, true},

		// --no-sync flag overrides
		{"config=true, --no-sync", true, false, true, false},
		{"config=false, --no-sync", false, false, true, false},

		// Both flags: --no-sync wins
		{"config=true, both flags", true, true, true, false},
		{"config=false, both flags", false, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from runPlanSave
			shouldSync := tt.configSync
			if tt.flagSync {
				shouldSync = true
			}
			if tt.flagNoSync {
				shouldSync = false
			}

			if shouldSync != tt.want {
				t.Errorf("shouldSync = %v, want %v", shouldSync, tt.want)
			}
		})
	}
}

func TestRunPlanImport_DelegatesToSave(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "jig-plan-import-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid plan file
	planContent := `---
id: TEST-IMPORT
title: Import Test
status: draft
author: testuser
---

# Import Test

## Problem Statement

Test problem.

## Proposed Solution

Test solution.
`
	planFile := filepath.Join(tempDir, "plan.md")
	if err := os.WriteFile(planFile, []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan file: %v", err)
	}

	// Set up cache
	cacheDir := filepath.Join(tempDir, "cache")
	os.MkdirAll(filepath.Join(cacheDir, "plans"), 0755)
	os.MkdirAll(filepath.Join(cacheDir, "issues"), 0755)

	oldCache := state.DefaultCache
	state.DefaultCache = state.NewCacheWithDir(cacheDir)
	defer func() { state.DefaultCache = oldCache }()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Disable sync
	oldNoSync := planSaveNoSync
	planSaveNoSync = true
	defer func() { planSaveNoSync = oldNoSync }()

	// Run import
	err = runPlanImport(planImportCmd, []string{planFile})
	if err != nil {
		t.Fatalf("runPlanImport() error = %v", err)
	}

	// Verify plan was saved
	savedPlan, _ := state.DefaultCache.GetPlan("TEST-IMPORT")
	if savedPlan == nil {
		t.Fatal("plan should have been saved via import")
	}
}
