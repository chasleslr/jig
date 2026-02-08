package cli

import (
	"context"
	"fmt"
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
