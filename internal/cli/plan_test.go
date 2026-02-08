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
