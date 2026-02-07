package ui

import (
	"strings"
	"testing"

	"github.com/charleslr/jig/internal/tracker"
)

func TestNewPlanPrompt(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Test Issue",
	}

	m := NewPlanPrompt(issue)

	if m.issue != issue {
		t.Error("expected issue to be set")
	}
	if m.state != stateMenu {
		t.Errorf("expected initial state to be stateMenu, got %v", m.state)
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor to be 0, got %d", m.cursor)
	}
	if m.instructions != "" {
		t.Errorf("expected instructions to be empty, got %q", m.instructions)
	}
	if m.result != nil {
		t.Error("expected result to be nil initially")
	}
	if m.quitting {
		t.Error("expected quitting to be false")
	}
}

func TestPlanPromptAction(t *testing.T) {
	// Test that action constants are distinct
	actions := []PlanPromptAction{
		PlanPromptActionStart,
		PlanPromptActionViewContext,
		PlanPromptActionAddInstructions,
		PlanPromptActionCancel,
	}

	seen := make(map[PlanPromptAction]bool)
	for _, action := range actions {
		if seen[action] {
			t.Errorf("duplicate action value: %v", action)
		}
		seen[action] = true
	}
}

func TestPlanPromptResult(t *testing.T) {
	result := &PlanPromptResult{
		Action:       PlanPromptActionStart,
		Instructions: "Additional context for planning",
	}

	if result.Action != PlanPromptActionStart {
		t.Errorf("expected Action to be PlanPromptActionStart, got %v", result.Action)
	}
	if result.Instructions != "Additional context for planning" {
		t.Errorf("expected Instructions to be set, got %q", result.Instructions)
	}
}

func TestPlanPromptModelViewMenu(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Implement user authentication",
	}

	m := NewPlanPrompt(issue)
	view := m.View()

	// Check that key elements are present
	if !strings.Contains(view, "Planning:") {
		t.Error("expected view to contain 'Planning:'")
	}
	if !strings.Contains(view, "ENG-123") {
		t.Error("expected view to contain issue identifier")
	}
	if !strings.Contains(view, "Implement user authentication") {
		t.Error("expected view to contain issue title")
	}
	if !strings.Contains(view, "What would you like to do?") {
		t.Error("expected view to contain menu question")
	}
	if !strings.Contains(view, "Start planning") {
		t.Error("expected view to contain 'Start planning' option")
	}
	if !strings.Contains(view, "View issue context") {
		t.Error("expected view to contain 'View issue context' option")
	}
	if !strings.Contains(view, "Add instructions") {
		t.Error("expected view to contain 'Add instructions' option")
	}
}

func TestPlanPromptModelViewMenuWithInstructions(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Test Issue",
	}

	m := NewPlanPrompt(issue)
	m.instructions = "Focus on security aspects"
	view := m.View()

	// Check that instructions indicator is shown
	if !strings.Contains(view, "Instructions added") {
		t.Error("expected view to contain 'Instructions added' indicator")
	}
	if !strings.Contains(view, "Focus on security") {
		t.Error("expected view to contain truncated instructions preview")
	}
}

func TestPlanPromptModelViewMenuWithLongInstructions(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Test Issue",
	}

	m := NewPlanPrompt(issue)
	m.instructions = "This is a very long instruction text that should be truncated when displayed in the menu because it exceeds the maximum display length"
	view := m.View()

	// Check that instructions are truncated with ellipsis
	if !strings.Contains(view, "...") {
		t.Error("expected long instructions to be truncated with ellipsis")
	}
}

func TestPlanPromptModelViewContext(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-123",
		Title:       "Test Issue",
		Description: "Detailed description of the issue",
		Status:      tracker.StatusTodo,
	}

	m := NewPlanPrompt(issue)
	m.state = stateViewContext
	view := m.View()

	// Check that context view shows issue details
	if !strings.Contains(view, "Issue Context") {
		t.Error("expected view to contain 'Issue Context' header")
	}
	if !strings.Contains(view, "ENG-123") {
		t.Error("expected view to contain issue identifier")
	}
	if !strings.Contains(view, "Test Issue") {
		t.Error("expected view to contain issue title")
	}
	if !strings.Contains(view, "return to menu") {
		t.Error("expected view to contain return instruction")
	}
}

func TestPlanPromptModelViewWhenQuitting(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Test Issue",
	}

	m := NewPlanPrompt(issue)
	m.quitting = true
	view := m.View()

	if view != "" {
		t.Errorf("expected empty view when quitting, got %q", view)
	}
}

func TestMenuOptions(t *testing.T) {
	// Verify menu options are configured correctly
	if len(menuOptions) != 3 {
		t.Errorf("expected 3 menu options, got %d", len(menuOptions))
	}

	// Check first option is "Start planning"
	if menuOptions[0].action != PlanPromptActionStart {
		t.Error("expected first option to be 'Start planning'")
	}
	if menuOptions[0].label != "Start planning" {
		t.Errorf("expected first option label 'Start planning', got %q", menuOptions[0].label)
	}

	// Check second option is "View issue context"
	if menuOptions[1].action != PlanPromptActionViewContext {
		t.Error("expected second option to be 'View issue context'")
	}

	// Check third option is "Add instructions"
	if menuOptions[2].action != PlanPromptActionAddInstructions {
		t.Error("expected third option to be 'Add instructions'")
	}
}

func TestPlanPromptModelResult(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Test Issue",
	}

	m := NewPlanPrompt(issue)

	// Initially result should be nil
	if m.Result() != nil {
		t.Error("expected Result() to be nil initially")
	}

	// Set a result
	m.result = &PlanPromptResult{
		Action:       PlanPromptActionStart,
		Instructions: "test instructions",
	}

	result := m.Result()
	if result == nil {
		t.Fatal("expected Result() to return non-nil")
	}
	if result.Action != PlanPromptActionStart {
		t.Errorf("expected Action PlanPromptActionStart, got %v", result.Action)
	}
	if result.Instructions != "test instructions" {
		t.Errorf("expected Instructions 'test instructions', got %q", result.Instructions)
	}
}
