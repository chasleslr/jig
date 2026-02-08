package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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

func TestPlanPromptModelUpdateWindowSize(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Test Issue",
	}

	m := NewPlanPrompt(issue)

	// Initially viewport should not be ready
	if m.contextReady {
		t.Error("expected contextReady to be false initially")
	}

	// Send window size message
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = newModel.(PlanPromptModel)

	// Viewport should now be ready
	if !m.contextReady {
		t.Error("expected contextReady to be true after window size message")
	}
	if m.width != 100 {
		t.Errorf("expected width to be 100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("expected height to be 50, got %d", m.height)
	}
}

func TestPlanPromptModelContextContentCaching(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-123",
		Title:       "Test Issue",
		Description: "Test description for caching",
		Status:      tracker.StatusTodo,
	}

	m := NewPlanPrompt(issue)

	// Initially context content should be empty
	if m.contextContent != "" {
		t.Error("expected contextContent to be empty initially")
	}

	// Simulate selecting "View issue context" by sending key message
	// First move cursor to View issue context option (index 1)
	m.cursor = 1
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PlanPromptModel)

	// Context content should now be cached
	if m.contextContent == "" {
		t.Error("expected contextContent to be populated after viewing context")
	}
	if m.state != stateViewContext {
		t.Errorf("expected state to be stateViewContext, got %v", m.state)
	}

	// Content should contain issue details
	if !strings.Contains(m.contextContent, "ENG-123") {
		t.Error("expected contextContent to contain issue identifier")
	}
	if !strings.Contains(m.contextContent, "Test Issue") {
		t.Error("expected contextContent to contain issue title")
	}
}

func TestPlanPromptModelRenderContextContent(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-456",
		Title:       "Render Context Test",
		Description: "Description with **markdown**",
		Status:      tracker.StatusInProgress,
		Labels:      []string{"feature", "priority"},
		URL:         "https://example.com/issue/456",
	}

	m := NewPlanPrompt(issue)
	m.width = 80

	content := m.renderContextContent()

	// Check that content contains expected elements
	if !strings.Contains(content, "ENG-456") {
		t.Error("expected content to contain identifier")
	}
	if !strings.Contains(content, "Render Context Test") {
		t.Error("expected content to contain title")
	}
	if !strings.Contains(content, "markdown") {
		t.Error("expected content to contain description text")
	}
	if !strings.Contains(content, "in_progress") {
		t.Error("expected content to contain status")
	}
}

func TestPlanPromptModelRenderContextContentDefaultWidth(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-789",
		Title:       "Default Width Test",
		Description: "Test description",
		Status:      tracker.StatusTodo,
	}

	m := NewPlanPrompt(issue)
	// Don't set width - should default to 80

	content := m.renderContextContent()

	// Should still render correctly with default width
	if !strings.Contains(content, "ENG-789") {
		t.Error("expected content to contain identifier with default width")
	}
}

func TestPlanPromptModelViewContextWithViewport(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-123",
		Title:       "Viewport Test",
		Description: "Test description for viewport",
		Status:      tracker.StatusTodo,
	}

	m := NewPlanPrompt(issue)

	// Initialize viewport with window size
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(PlanPromptModel)

	// Move to view context
	m.cursor = 1
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PlanPromptModel)

	// Get the view
	view := m.View()

	// Should contain header and content
	if !strings.Contains(view, "Issue Context") {
		t.Error("expected view to contain 'Issue Context' header")
	}
	if !strings.Contains(view, "ENG-123") {
		t.Error("expected view to contain issue identifier")
	}
}

func TestPlanPromptModelViewContextScrolling(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-123",
		Title:       "Scroll Test",
		Description: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
		Status:      tracker.StatusTodo,
	}

	m := NewPlanPrompt(issue)

	// Initialize viewport
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(PlanPromptModel)

	// Move to view context
	m.cursor = 1
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PlanPromptModel)

	// Send scroll down key - should not error
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(PlanPromptModel)

	// Should still be in view context state
	if m.state != stateViewContext {
		t.Errorf("expected state to remain stateViewContext after scroll, got %v", m.state)
	}
}

func TestPlanPromptModelViewContextEscape(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Escape Test",
	}

	m := NewPlanPrompt(issue)
	m.state = stateViewContext

	// Press escape to return to menu
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(PlanPromptModel)

	if m.state != stateMenu {
		t.Errorf("expected state to be stateMenu after escape, got %v", m.state)
	}
}

func TestPlanPromptModelViewContextQuit(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Quit Test",
	}

	m := NewPlanPrompt(issue)
	m.state = stateViewContext

	// Press q to return to menu
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(PlanPromptModel)

	if m.state != stateMenu {
		t.Errorf("expected state to be stateMenu after pressing q, got %v", m.state)
	}
}

func TestPlanPromptModelMenuQuit(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Menu Quit Test",
	}

	m := NewPlanPrompt(issue)

	// Press q to quit from menu
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(PlanPromptModel)

	if !m.quitting {
		t.Error("expected quitting to be true after pressing q")
	}
	if m.result == nil {
		t.Fatal("expected result to be set")
	}
	if m.result.Action != PlanPromptActionCancel {
		t.Errorf("expected action to be Cancel, got %v", m.result.Action)
	}
	if cmd == nil {
		t.Error("expected quit command to be returned")
	}
}

func TestPlanPromptModelMenuNavigation(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Navigation Test",
	}

	m := NewPlanPrompt(issue)

	// Initial cursor should be 0
	if m.cursor != 0 {
		t.Errorf("expected initial cursor to be 0, got %d", m.cursor)
	}

	// Press down to move cursor
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(PlanPromptModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor to be 1 after down, got %d", m.cursor)
	}

	// Press j to move cursor down again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(PlanPromptModel)
	if m.cursor != 2 {
		t.Errorf("expected cursor to be 2 after j, got %d", m.cursor)
	}

	// Press down at bottom - should stay at 2
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(PlanPromptModel)
	if m.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m.cursor)
	}

	// Press up to move cursor back
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(PlanPromptModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor to be 1 after up, got %d", m.cursor)
	}

	// Press k to move cursor up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(PlanPromptModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor to be 0 after k, got %d", m.cursor)
	}

	// Press up at top - should stay at 0
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(PlanPromptModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.cursor)
	}
}

func TestPlanPromptModelStartPlanning(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Start Planning Test",
	}

	m := NewPlanPrompt(issue)
	m.instructions = "Test instructions"

	// Cursor is at 0 (Start planning), press enter
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PlanPromptModel)

	if !m.quitting {
		t.Error("expected quitting to be true")
	}
	if m.result == nil {
		t.Fatal("expected result to be set")
	}
	if m.result.Action != PlanPromptActionStart {
		t.Errorf("expected action to be Start, got %v", m.result.Action)
	}
	if m.result.Instructions != "Test instructions" {
		t.Errorf("expected instructions to be preserved, got %q", m.result.Instructions)
	}
	if cmd == nil {
		t.Error("expected quit command to be returned")
	}
}

func TestPlanPromptModelStartPlanningWithSpace(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Start Planning Space Test",
	}

	m := NewPlanPrompt(issue)

	// Press space to select (same as enter)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = newModel.(PlanPromptModel)

	if !m.quitting {
		t.Error("expected quitting to be true after space")
	}
	if m.result == nil {
		t.Fatal("expected result to be set")
	}
	if m.result.Action != PlanPromptActionStart {
		t.Errorf("expected action to be Start, got %v", m.result.Action)
	}
	if cmd == nil {
		t.Error("expected quit command to be returned")
	}
}

func TestPlanPromptModelViewContextWithCachedContent(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-123",
		Title:       "Cached Content Test",
		Description: "Test description",
		Status:      tracker.StatusTodo,
	}

	m := NewPlanPrompt(issue)

	// Pre-cache the content
	m.contextContent = "Pre-cached content for testing"

	// Move to view context
	m.cursor = 1
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PlanPromptModel)

	// Content should remain as pre-cached (not re-rendered)
	if m.contextContent != "Pre-cached content for testing" {
		t.Errorf("expected cached content to be preserved, got %q", m.contextContent)
	}
}

func TestPlanPromptModelWindowSizeUpdate(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Window Size Update Test",
	}

	m := NewPlanPrompt(issue)

	// First window size to initialize
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(PlanPromptModel)

	if !m.contextReady {
		t.Error("expected contextReady to be true")
	}

	// Update window size
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(PlanPromptModel)

	if m.width != 120 {
		t.Errorf("expected width to be 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height to be 40, got %d", m.height)
	}
}

func TestPlanPromptModelWindowSizeInViewContext(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-123",
		Title:       "Window Size in Context Test",
		Description: "Test description",
		Status:      tracker.StatusTodo,
	}

	m := NewPlanPrompt(issue)

	// Initialize viewport
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(PlanPromptModel)

	// Move to view context and cache content
	m.cursor = 1
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PlanPromptModel)

	// Now send another window size message while in view context
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = newModel.(PlanPromptModel)

	// Should update dimensions
	if m.width != 100 {
		t.Errorf("expected width to be 100, got %d", m.width)
	}
}
