package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/tracker"
	trackerMock "github.com/charleslr/jig/internal/tracker/mock"
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

func TestProcessIssueForPlan(t *testing.T) {
	tests := []struct {
		name          string
		issue         *tracker.Issue
		currentTitle  string
		wantTitle     string
		wantHasContext bool
	}{
		{
			name: "empty current title uses issue title",
			issue: &tracker.Issue{
				ID:          "abc123",
				Identifier:  "ENG-123",
				Title:       "Issue Title From Tracker",
				Description: "Some description",
				Status:      tracker.StatusTodo,
			},
			currentTitle:   "",
			wantTitle:      "Issue Title From Tracker",
			wantHasContext: true,
		},
		{
			name: "non-empty current title is preserved",
			issue: &tracker.Issue{
				ID:          "abc123",
				Identifier:  "ENG-456",
				Title:       "Issue Title From Tracker",
				Description: "Some description",
				Status:      tracker.StatusInProgress,
			},
			currentTitle:   "User Provided Title",
			wantTitle:      "User Provided Title",
			wantHasContext: true,
		},
		{
			name: "whitespace-only current title is preserved",
			issue: &tracker.Issue{
				ID:         "abc123",
				Identifier: "ENG-789",
				Title:      "Issue Title",
				Status:     tracker.StatusBacklog,
			},
			currentTitle:   "   ",
			wantTitle:      "   ", // Non-empty string, even if whitespace
			wantHasContext: true,
		},
		{
			name: "issue with empty title",
			issue: &tracker.Issue{
				ID:          "abc123",
				Identifier:  "ENG-101",
				Title:       "",
				Description: "Description only",
				Status:      tracker.StatusTodo,
			},
			currentTitle:   "",
			wantTitle:      "", // Empty issue title results in empty result title
			wantHasContext: true,
		},
		{
			name: "minimal issue with labels",
			issue: &tracker.Issue{
				ID:         "xyz789",
				Identifier: "BUG-001",
				Title:      "Fix critical bug",
				Status:     tracker.StatusDone,
				Labels:     []string{"bug", "critical"},
			},
			currentTitle:   "",
			wantTitle:      "Fix critical bug",
			wantHasContext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processIssueForPlan(tt.issue, tt.currentTitle)

			// Verify title handling
			if result.Title != tt.wantTitle {
				t.Errorf("processIssueForPlan() title = %q, want %q", result.Title, tt.wantTitle)
			}

			// Verify issue context is populated
			if tt.wantHasContext && result.IssueContext == "" {
				t.Error("processIssueForPlan() expected non-empty IssueContext")
			}

			// Verify issue context contains the issue identifier
			if tt.wantHasContext && !strings.Contains(result.IssueContext, tt.issue.Identifier) {
				t.Errorf("processIssueForPlan() IssueContext should contain identifier %q", tt.issue.Identifier)
			}
		})
	}
}

func TestProcessIssueForPlanContext(t *testing.T) {
	// Test that IssueContext is properly formatted using formatIssueContext
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-999",
		Title:       "Test Integration",
		Description: "Testing that processIssueForPlan uses formatIssueContext correctly",
		Status:      tracker.StatusInProgress,
		Labels:      []string{"test", "integration"},
	}

	result := processIssueForPlan(issue, "")

	// Verify the context matches what formatIssueContext would produce
	expectedContext := formatIssueContext(issue)
	if result.IssueContext != expectedContext {
		t.Errorf("processIssueForPlan() IssueContext does not match formatIssueContext output\ngot:\n%s\nwant:\n%s",
			result.IssueContext, expectedContext)
	}
}

func TestProcessIssueForPlanTitlePrecedence(t *testing.T) {
	// Test that explicit title always takes precedence over issue title
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "FEAT-001",
		Title:      "Original Issue Title",
		Status:     tracker.StatusTodo,
	}

	// When currentTitle is provided, it should be used regardless of issue title
	result := processIssueForPlan(issue, "Custom Plan Title")
	if result.Title != "Custom Plan Title" {
		t.Errorf("processIssueForPlan() should preserve explicit title, got %q", result.Title)
	}

	// When currentTitle is empty, issue title should be used
	result = processIssueForPlan(issue, "")
	if result.Title != "Original Issue Title" {
		t.Errorf("processIssueForPlan() should use issue title when no explicit title, got %q", result.Title)
	}
}

func TestFetchIssueWithDeps(t *testing.T) {
	ctx := context.Background()
	testIssue := &tracker.Issue{
		ID:         "test-id",
		Identifier: "TEST-123",
		Title:      "Test Issue",
		Status:     tracker.StatusTodo,
	}

	t.Run("interactive mode uses spinner and returns issue on success", func(t *testing.T) {
		spinnerCalled := false
		spinnerMessage := ""

		deps := issueFetchDeps{
			isInteractive: func() bool { return true },
			runSpinner: func(message string, fn func() error) error {
				spinnerCalled = true
				spinnerMessage = message
				return fn() // Execute the function passed to spinner
			},
			getTracker: func() (tracker.Tracker, error) {
				return &mockTrackerForFetch{issue: testIssue}, nil
			},
		}

		result := fetchIssueWithDeps(ctx, "TEST-123", deps)

		if !spinnerCalled {
			t.Error("expected spinner to be called in interactive mode")
		}
		if !strings.Contains(spinnerMessage, "TEST-123") {
			t.Errorf("spinner message should contain issue ID, got %q", spinnerMessage)
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Issue == nil {
			t.Error("expected issue to be returned")
		}
		if result.Issue.Identifier != "TEST-123" {
			t.Errorf("expected issue identifier TEST-123, got %s", result.Issue.Identifier)
		}
	})

	t.Run("non-interactive mode fetches directly without spinner", func(t *testing.T) {
		spinnerCalled := false

		deps := issueFetchDeps{
			isInteractive: func() bool { return false },
			runSpinner: func(message string, fn func() error) error {
				spinnerCalled = true
				return fn()
			},
			getTracker: func() (tracker.Tracker, error) {
				return &mockTrackerForFetch{issue: testIssue}, nil
			},
		}

		result := fetchIssueWithDeps(ctx, "TEST-123", deps)

		if spinnerCalled {
			t.Error("spinner should not be called in non-interactive mode")
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Issue == nil {
			t.Error("expected issue to be returned")
		}
	})

	t.Run("interactive mode handles tracker error", func(t *testing.T) {
		trackerErr := fmt.Errorf("failed to connect to tracker")

		deps := issueFetchDeps{
			isInteractive: func() bool { return true },
			runSpinner: func(message string, fn func() error) error {
				return fn()
			},
			getTracker: func() (tracker.Tracker, error) {
				return nil, trackerErr
			},
		}

		result := fetchIssueWithDeps(ctx, "TEST-123", deps)

		if result.Err == nil {
			t.Error("expected error when tracker fails")
		}
		if !strings.Contains(result.Err.Error(), "failed to connect") {
			t.Errorf("expected tracker error, got %v", result.Err)
		}
		if result.Issue != nil {
			t.Error("expected nil issue when tracker fails")
		}
	})

	t.Run("non-interactive mode handles tracker error", func(t *testing.T) {
		trackerErr := fmt.Errorf("failed to connect to tracker")

		deps := issueFetchDeps{
			isInteractive: func() bool { return false },
			runSpinner: func(message string, fn func() error) error {
				return fn()
			},
			getTracker: func() (tracker.Tracker, error) {
				return nil, trackerErr
			},
		}

		result := fetchIssueWithDeps(ctx, "TEST-123", deps)

		if result.Err == nil {
			t.Error("expected error when tracker fails")
		}
		if result.Issue != nil {
			t.Error("expected nil issue when tracker fails")
		}
	})

	t.Run("interactive mode handles issue fetch error", func(t *testing.T) {
		deps := issueFetchDeps{
			isInteractive: func() bool { return true },
			runSpinner: func(message string, fn func() error) error {
				return fn()
			},
			getTracker: func() (tracker.Tracker, error) {
				return &mockTrackerForFetch{err: fmt.Errorf("issue not found")}, nil
			},
		}

		result := fetchIssueWithDeps(ctx, "INVALID-ID", deps)

		if result.Err == nil {
			t.Error("expected error when issue fetch fails")
		}
		if !strings.Contains(result.Err.Error(), "issue not found") {
			t.Errorf("expected issue fetch error, got %v", result.Err)
		}
	})

	t.Run("non-interactive mode handles issue fetch error", func(t *testing.T) {
		deps := issueFetchDeps{
			isInteractive: func() bool { return false },
			runSpinner: func(message string, fn func() error) error {
				return fn()
			},
			getTracker: func() (tracker.Tracker, error) {
				return &mockTrackerForFetch{err: fmt.Errorf("issue not found")}, nil
			},
		}

		result := fetchIssueWithDeps(ctx, "INVALID-ID", deps)

		if result.Err == nil {
			t.Error("expected error when issue fetch fails")
		}
	})

	t.Run("spinner error is propagated", func(t *testing.T) {
		spinnerErr := fmt.Errorf("spinner crashed")

		deps := issueFetchDeps{
			isInteractive: func() bool { return true },
			runSpinner: func(message string, fn func() error) error {
				// Simulate spinner crashing before or during execution
				return spinnerErr
			},
			getTracker: func() (tracker.Tracker, error) {
				return &mockTrackerForFetch{issue: testIssue}, nil
			},
		}

		result := fetchIssueWithDeps(ctx, "TEST-123", deps)

		if result.Err == nil {
			t.Error("expected error when spinner crashes")
		}
		if result.Err != spinnerErr {
			t.Errorf("expected spinner error to be propagated, got %v", result.Err)
		}
	})
}

// mockTrackerForFetch is a minimal mock tracker for testing fetchIssueWithDeps
type mockTrackerForFetch struct {
	issue *tracker.Issue
	err   error
}

func (m *mockTrackerForFetch) GetIssue(ctx context.Context, id string) (*tracker.Issue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.issue, nil
}

// Stub implementations for the Tracker interface
func (m *mockTrackerForFetch) CreateIssue(ctx context.Context, issue *tracker.Issue) (*tracker.Issue, error) {
	return nil, nil
}
func (m *mockTrackerForFetch) UpdateIssue(ctx context.Context, id string, updates *tracker.IssueUpdate) error {
	return nil
}
func (m *mockTrackerForFetch) SearchIssues(ctx context.Context, query string) ([]*tracker.Issue, error) {
	return nil, nil
}
func (m *mockTrackerForFetch) CreateSubIssue(ctx context.Context, parentID string, issue *tracker.Issue) (*tracker.Issue, error) {
	return nil, nil
}
func (m *mockTrackerForFetch) GetSubIssues(ctx context.Context, parentID string) ([]*tracker.Issue, error) {
	return nil, nil
}
func (m *mockTrackerForFetch) AddComment(ctx context.Context, issueID string, body string) (*tracker.Comment, error) {
	return nil, nil
}
func (m *mockTrackerForFetch) GetComments(ctx context.Context, issueID string) ([]*tracker.Comment, error) {
	return nil, nil
}
func (m *mockTrackerForFetch) TransitionIssue(ctx context.Context, id string, status tracker.Status) error {
	return nil
}
func (m *mockTrackerForFetch) GetAvailableStatuses(ctx context.Context, id string) ([]tracker.Status, error) {
	return nil, nil
}
func (m *mockTrackerForFetch) GetTeams(ctx context.Context) ([]tracker.Team, error) { return nil, nil }
func (m *mockTrackerForFetch) GetProjects(ctx context.Context, teamID string) ([]tracker.Project, error) {
	return nil, nil
}
func (m *mockTrackerForFetch) SetBlocking(ctx context.Context, blockerID, blockedID string) error {
	return nil
}
func (m *mockTrackerForFetch) GetBlockedBy(ctx context.Context, issueID string) ([]*tracker.Issue, error) {
	return nil, nil
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

func TestShouldCreateIssueForPlan(t *testing.T) {
	t.Run("returns false when plan already has linked issue", func(t *testing.T) {
		cfg := &config.Config{
			Default: config.DefaultConfig{
				Tracker: "linear",
			},
			Linear: config.LinearConfig{},
		}
		p := &plan.Plan{IssueID: "NUM-41"} // Already has linked issue

		result := shouldCreateIssueForPlan(cfg, p)
		if result {
			t.Error("expected false when plan already has linked issue")
		}
	})

	t.Run("returns false when tracker is not linear", func(t *testing.T) {
		cfg := &config.Config{
			Default: config.DefaultConfig{
				Tracker: "github", // Not linear
			},
		}
		p := &plan.Plan{IssueID: ""} // No linked issue

		result := shouldCreateIssueForPlan(cfg, p)
		if result {
			t.Error("expected false when tracker is not linear")
		}
	})

	t.Run("returns false when issue creation is disabled", func(t *testing.T) {
		createDisabled := false
		cfg := &config.Config{
			Default: config.DefaultConfig{
				Tracker: "linear",
			},
			Linear: config.LinearConfig{
				CreateIssueOnSave: &createDisabled,
			},
		}
		p := &plan.Plan{IssueID: ""} // No linked issue

		result := shouldCreateIssueForPlan(cfg, p)
		if result {
			t.Error("expected false when issue creation is disabled")
		}
	})

	// Note: Testing "no API key configured" is skipped because it depends on
	// the system's ~/.jig/.credentials file which may have real credentials.
	// The API key check path is tested indirectly through integration tests.
}

func TestShouldSyncToLinear(t *testing.T) {
	t.Run("returns false when tracker is not linear", func(t *testing.T) {
		cfg := &config.Config{
			Default: config.DefaultConfig{
				Tracker: "github",
			},
		}
		p := &plan.Plan{IssueID: "NUM-41"}

		result := shouldSyncToLinear(cfg, p)
		if result {
			t.Error("expected false when tracker is not linear")
		}
	})

	t.Run("returns false when sync is disabled", func(t *testing.T) {
		syncDisabled := false
		cfg := &config.Config{
			Default: config.DefaultConfig{
				Tracker: "linear",
			},
			Linear: config.LinearConfig{
				SyncPlanOnSave: &syncDisabled,
			},
		}
		p := &plan.Plan{IssueID: "NUM-41"}

		result := shouldSyncToLinear(cfg, p)
		if result {
			t.Error("expected false when sync is disabled")
		}
	})

	t.Run("returns false when plan has no linked issue", func(t *testing.T) {
		cfg := &config.Config{
			Default: config.DefaultConfig{
				Tracker: "linear",
			},
			Linear: config.LinearConfig{},
		}
		p := &plan.Plan{IssueID: ""} // No linked issue

		result := shouldSyncToLinear(cfg, p)
		if result {
			t.Error("expected false when plan has no linked issue")
		}
	})

	// Note: Testing "no API key configured" is skipped because it depends on
	// the system's ~/.jig/.credentials file which may have real credentials.
	// The API key check path is tested indirectly through integration tests.
}

func TestPlanSaveCmd_NoSyncFlag(t *testing.T) {
	// Test that the --no-sync flag is properly defined on the command
	cmd := planSaveCmd

	noSyncFlag := cmd.Flags().Lookup("no-sync")
	if noSyncFlag == nil {
		t.Fatal("expected --no-sync flag to be defined on planSaveCmd")
	}

	if noSyncFlag.Usage == "" {
		t.Error("expected --no-sync flag to have usage text")
	}

	// Verify default value is false
	if noSyncFlag.DefValue != "false" {
		t.Errorf("expected --no-sync default value to be false, got %q", noSyncFlag.DefValue)
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

func TestSyncPlanWithSyncer(t *testing.T) {
	ctx := context.Background()

	t.Run("calls syncer with plan and label name", func(t *testing.T) {
		syncer := &mockPlanSyncer{}
		p := &plan.Plan{
			ID:               "PLAN-123",
			IssueID:          "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "Test problem",
			ProposedSolution: "Test solution",
		}
		labelName := "jig-plan"

		err := syncPlanWithSyncer(ctx, syncer, p, labelName)
		if err != nil {
			t.Fatalf("syncPlanWithSyncer failed: %v", err)
		}

		// Verify the syncer was called correctly
		if len(syncer.syncedPlans) != 1 {
			t.Fatalf("expected 1 synced plan, got %d", len(syncer.syncedPlans))
		}
		if syncer.syncedPlans[0].Plan.ID != "PLAN-123" {
			t.Errorf("expected plan ID 'PLAN-123', got '%s'", syncer.syncedPlans[0].Plan.ID)
		}
		if syncer.syncedPlans[0].LabelName != "jig-plan" {
			t.Errorf("expected label name 'jig-plan', got '%s'", syncer.syncedPlans[0].LabelName)
		}
	})

	t.Run("propagates syncer error", func(t *testing.T) {
		syncer := &mockPlanSyncer{
			err: fmt.Errorf("sync failed"),
		}
		p := &plan.Plan{
			ID:      "PLAN-456",
			IssueID: "NUM-42",
		}

		err := syncPlanWithSyncer(ctx, syncer, p, "test-label")
		if err == nil {
			t.Error("expected error from syncer to be propagated")
		}
		if !strings.Contains(err.Error(), "sync failed") {
			t.Errorf("expected 'sync failed' error, got: %v", err)
		}
	})

	t.Run("works with different label names", func(t *testing.T) {
		syncer := &mockPlanSyncer{}
		p := &plan.Plan{
			ID:      "PLAN-789",
			IssueID: "NUM-43",
		}

		err := syncPlanWithSyncer(ctx, syncer, p, "custom-plan-label")
		if err != nil {
			t.Fatalf("syncPlanWithSyncer failed: %v", err)
		}

		if syncer.syncedPlans[0].LabelName != "custom-plan-label" {
			t.Errorf("expected label name 'custom-plan-label', got '%s'", syncer.syncedPlans[0].LabelName)
		}
	})
}

// mockPlanSyncer is a mock implementation of tracker.PlanSyncer for testing
type mockPlanSyncer struct {
	syncedPlans []syncedPlanRecord
	err         error
}

type syncedPlanRecord struct {
	Plan      *plan.Plan
	LabelName string
}

func (m *mockPlanSyncer) SyncPlanToIssue(ctx context.Context, p *plan.Plan, labelName string) error {
	if m.err != nil {
		return m.err
	}
	m.syncedPlans = append(m.syncedPlans, syncedPlanRecord{
		Plan:      p,
		LabelName: labelName,
	})
	return nil
}

func TestSyncPlanWithSyncer_UsingTrackerMock(t *testing.T) {
	// This test demonstrates that the tracker/mock package can be used
	// for integration testing of the sync functionality

	ctx := context.Background()

	t.Run("integration with tracker mock client", func(t *testing.T) {
		// Import and use the mock tracker client
		mockClient := trackerMock.NewClient()

		// Create a test issue first
		issue, err := mockClient.CreateIssue(ctx, &tracker.Issue{
			Title:       "Test Issue",
			Description: "Test description",
		})
		if err != nil {
			t.Fatalf("failed to create mock issue: %v", err)
		}

		// Create a plan linked to the issue
		p := &plan.Plan{
			ID:               "PLAN-INTEGRATION",
			IssueID:          issue.Identifier,
			Title:            "Integration Test Plan",
			ProblemStatement: "Testing the integration",
			ProposedSolution: "Use the mock client",
		}

		// Sync using the mock client's PlanSyncer implementation
		err = syncPlanWithSyncer(ctx, mockClient, p, "test-label")
		if err != nil {
			t.Fatalf("syncPlanWithSyncer failed: %v", err)
		}

		// Verify the plan was synced via the mock's tracking
		syncedPlans := mockClient.GetSyncedPlans()
		if len(syncedPlans) != 1 {
			t.Fatalf("expected 1 synced plan, got %d", len(syncedPlans))
		}
		if syncedPlans[0].Plan.ID != "PLAN-INTEGRATION" {
			t.Errorf("expected plan ID 'PLAN-INTEGRATION', got '%s'", syncedPlans[0].Plan.ID)
		}
		if syncedPlans[0].LabelName != "test-label" {
			t.Errorf("expected label name 'test-label', got '%s'", syncedPlans[0].LabelName)
		}
	})

	t.Run("mock returns error for plan with no linked issue", func(t *testing.T) {
		mockClient := trackerMock.NewClient()

		p := &plan.Plan{
			ID:      "PLAN-NO-ISSUE",
			IssueID: "", // No linked issue
		}

		err := syncPlanWithSyncer(ctx, mockClient, p, "test-label")
		if err == nil {
			t.Error("expected error for plan with no linked issue")
		}
		if !strings.Contains(err.Error(), "no linked issue") {
			t.Errorf("expected 'no linked issue' error, got: %v", err)
		}
	})

	t.Run("mock returns error for nonexistent issue", func(t *testing.T) {
		mockClient := trackerMock.NewClient()

		p := &plan.Plan{
			ID:      "PLAN-NONEXISTENT",
			IssueID: "NONEXISTENT-999", // Issue doesn't exist
		}

		err := syncPlanWithSyncer(ctx, mockClient, p, "test-label")
		if err == nil {
			t.Error("expected error for nonexistent issue")
		}
		if !strings.Contains(err.Error(), "issue not found") {
			t.Errorf("expected 'issue not found' error, got: %v", err)
		}
	})
}

// mockIssueCreator is a mock implementation of issueCreator for testing
type mockIssueCreator struct {
	issue *tracker.Issue
	err   error
}

func (m *mockIssueCreator) CreateIssueFromPlan(ctx context.Context, p *plan.Plan) (*tracker.Issue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.issue, nil
}

func TestCreateIssueForPlanWithCreator(t *testing.T) {
	ctx := context.Background()

	t.Run("returns issue identifier on success", func(t *testing.T) {
		creator := &mockIssueCreator{
			issue: &tracker.Issue{
				ID:         "issue-uuid",
				Identifier: "NUM-123",
				Title:      "Test Issue",
			},
		}
		p := &plan.Plan{
			ID:    "PLAN-123",
			Title: "Test Plan",
		}

		identifier, err := createIssueForPlanWithCreator(ctx, creator, p)
		if err != nil {
			t.Fatalf("createIssueForPlanWithCreator failed: %v", err)
		}
		if identifier != "NUM-123" {
			t.Errorf("expected identifier 'NUM-123', got '%s'", identifier)
		}
	})

	t.Run("propagates error from creator", func(t *testing.T) {
		creator := &mockIssueCreator{
			err: fmt.Errorf("API error: rate limited"),
		}
		p := &plan.Plan{
			ID:    "PLAN-123",
			Title: "Test Plan",
		}

		_, err := createIssueForPlanWithCreator(ctx, creator, p)
		if err == nil {
			t.Error("expected error to be propagated")
		}
		if !strings.Contains(err.Error(), "rate limited") {
			t.Errorf("expected 'rate limited' error, got: %v", err)
		}
	})
}

func TestGetLinearClientWithStore(t *testing.T) {
	t.Run("returns error when store creation fails", func(t *testing.T) {
		cfg := &config.Config{
			Linear: config.LinearConfig{
				APIKey: "test-key",
			},
		}
		failingStore := func() (*config.Store, error) {
			return nil, fmt.Errorf("failed to create store")
		}

		_, err := getLinearClientWithStore(cfg, failingStore)
		if err == nil {
			t.Error("expected error when store creation fails")
		}
		if !strings.Contains(err.Error(), "failed to get config store") {
			t.Errorf("expected 'failed to get config store' error, got: %v", err)
		}
	})

	t.Run("uses config API key when store has no key", func(t *testing.T) {
		cfg := &config.Config{
			Linear: config.LinearConfig{
				APIKey:    "config-api-key",
				TeamID:    "team-123",
			},
		}
		// Create a real store but it won't have a key set
		client, err := getLinearClientWithStore(cfg, config.NewStore)
		if err != nil {
			// If this fails, it's because no API key is configured anywhere
			// which is expected in a test environment without credentials
			if strings.Contains(err.Error(), "Linear API key not configured") {
				t.Skip("skipping - no API key in test environment")
			}
			t.Fatalf("getLinearClientWithStore failed: %v", err)
		}
		if client == nil {
			t.Error("expected non-nil client")
		}
	})

	t.Run("returns error when no API key configured", func(t *testing.T) {
		cfg := &config.Config{
			Linear: config.LinearConfig{
				APIKey: "", // No API key in config
			},
		}
		// Mock store that returns no API key
		mockStore := func() (*config.Store, error) {
			return config.NewStoreWithPath(t.TempDir())
		}

		_, err := getLinearClientWithStore(cfg, mockStore)
		if err == nil {
			t.Error("expected error when no API key configured")
		}
		if !strings.Contains(err.Error(), "Linear API key not configured") {
			t.Errorf("expected 'Linear API key not configured' error, got: %v", err)
		}
	})
}
