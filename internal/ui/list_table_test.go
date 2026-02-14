package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestListActionConstants(t *testing.T) {
	// Verify action constants are defined correctly
	actions := []ListAction{
		ListActionSelect,
		ListActionOpenPR,
		ListActionCheckout,
		ListActionDetails,
		ListActionQuit,
	}

	expected := []string{"select", "open_pr", "checkout", "details", "quit"}
	for i, action := range actions {
		if string(action) != expected[i] {
			t.Errorf("action %d: expected %q, got %q", i, expected[i], string(action))
		}
	}
}

func TestNewListTable(t *testing.T) {
	items := []ListItem{
		{IssueID: "NUM-1", Title: "First item"},
		{IssueID: "NUM-2", Title: "Second item"},
	}

	model := newListTable("Test Title", items)

	if model.title != "Test Title" {
		t.Errorf("expected title %q, got %q", "Test Title", model.title)
	}
	if len(model.items) != 2 {
		t.Errorf("expected 2 items, got %d", len(model.items))
	}
	if model.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", model.cursor)
	}
	if model.height != 10 {
		t.Errorf("expected default height 10, got %d", model.height)
	}
}

func TestListTableInit(t *testing.T) {
	model := newListTable("Test", []ListItem{})
	cmd := model.Init()
	if cmd != nil {
		t.Error("expected nil cmd from Init")
	}
}

func TestListTableUpdateNavigation(t *testing.T) {
	items := []ListItem{
		{IssueID: "NUM-1"},
		{IssueID: "NUM-2"},
		{IssueID: "NUM-3"},
	}

	tests := []struct {
		name           string
		key            string
		initialCursor  int
		expectedCursor int
	}{
		{"down from top", "down", 0, 1},
		{"j from top", "j", 0, 1},
		{"up from middle", "up", 1, 0},
		{"k from middle", "k", 1, 0},
		{"up at top stays at top", "up", 0, 0},
		{"down at bottom stays at bottom", "down", 2, 2},
		{"home goes to start", "home", 2, 0},
		{"g goes to start", "g", 2, 0},
		{"end goes to end", "end", 0, 2},
		{"G goes to end", "G", 0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newListTable("Test", items)
			m.cursor = tt.initialCursor

			newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			result := newModel.(listTableModel)

			if result.cursor != tt.expectedCursor {
				t.Errorf("expected cursor %d, got %d", tt.expectedCursor, result.cursor)
			}
		})
	}
}

func TestListTableUpdateQuit(t *testing.T) {
	items := []ListItem{{IssueID: "NUM-1"}}

	quitKeys := []string{"q", "esc"}
	for _, key := range quitKeys {
		t.Run(key, func(t *testing.T) {
			m := newListTable("Test", items)
			newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			result := newModel.(listTableModel)

			if !result.quitting {
				t.Error("expected quitting to be true")
			}
			if result.result == nil || result.result.Action != ListActionQuit {
				t.Error("expected quit action")
			}
			if cmd == nil {
				t.Error("expected tea.Quit cmd")
			}
		})
	}
}

func TestListTableUpdateActions(t *testing.T) {
	items := []ListItem{
		{IssueID: "NUM-1", Title: "Test", PRNumber: 123, Path: "/path", Exists: true},
	}

	tests := []struct {
		key            string
		expectedAction ListAction
	}{
		{"enter", ListActionSelect},
		{"o", ListActionOpenPR},
		{"c", ListActionCheckout},
		{"d", ListActionDetails},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			model := newListTable("Test", items)
			newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			result := newModel.(listTableModel)

			if !result.quitting {
				t.Error("expected quitting to be true")
			}
			if result.result == nil {
				t.Fatal("expected result to be set")
			}
			if result.result.Action != tt.expectedAction {
				t.Errorf("expected action %v, got %v", tt.expectedAction, result.result.Action)
			}
			if result.result.Item == nil {
				t.Error("expected item to be set")
			}
			if result.result.Item.IssueID != "NUM-1" {
				t.Errorf("expected item IssueID NUM-1, got %s", result.result.Item.IssueID)
			}
			if cmd == nil {
				t.Error("expected tea.Quit cmd")
			}
		})
	}
}

func TestListTableUpdateWindowSize(t *testing.T) {
	model := newListTable("Test", []ListItem{})

	newModel, _ := model.Update(tea.WindowSizeMsg{Height: 20, Width: 80})
	result := newModel.(listTableModel)

	// height should be terminal height - 6 (for header and help)
	expectedHeight := 14
	if result.height != expectedHeight {
		t.Errorf("expected height %d, got %d", expectedHeight, result.height)
	}
}

func TestListTableUpdateWindowSizeMinHeight(t *testing.T) {
	model := newListTable("Test", []ListItem{})

	// Very small terminal
	newModel, _ := model.Update(tea.WindowSizeMsg{Height: 5, Width: 80})
	result := newModel.(listTableModel)

	// Should be capped at minimum of 3
	if result.height != 3 {
		t.Errorf("expected minimum height 3, got %d", result.height)
	}
}

func TestListTableViewEmpty(t *testing.T) {
	model := newListTable("Empty Test", []ListItem{})
	view := model.View()

	if !strings.Contains(view, "Empty Test") {
		t.Error("expected title in view")
	}
	if !strings.Contains(view, "No items") {
		t.Error("expected 'No items' message")
	}
}

func TestListTableViewWithItems(t *testing.T) {
	items := []ListItem{
		{IssueID: "NUM-1", Title: "First item", PlanStatus: "draft", PRNumber: 123},
		{IssueID: "NUM-2", Title: "Second item", PlanStatus: "complete", PRNumber: 0},
	}
	model := newListTable("Test", items)
	view := model.View()

	// Check header columns
	if !strings.Contains(view, "ISSUE") {
		t.Error("expected ISSUE header")
	}
	if !strings.Contains(view, "TITLE") {
		t.Error("expected TITLE header")
	}
	if !strings.Contains(view, "STATUS") {
		t.Error("expected STATUS header")
	}
	if !strings.Contains(view, "PR") {
		t.Error("expected PR header")
	}

	// Check items
	if !strings.Contains(view, "NUM-1") {
		t.Error("expected NUM-1 in view")
	}
	if !strings.Contains(view, "NUM-2") {
		t.Error("expected NUM-2 in view")
	}

	// Check help bar
	if !strings.Contains(view, "open PR") {
		t.Error("expected 'open PR' in help bar")
	}
	if !strings.Contains(view, "checkout") {
		t.Error("expected 'checkout' in help bar")
	}
	if !strings.Contains(view, "details") {
		t.Error("expected 'details' in help bar")
	}
	if !strings.Contains(view, "select") {
		t.Error("expected 'select' in help bar")
	}
	if !strings.Contains(view, "quit") {
		t.Error("expected 'quit' in help bar")
	}
}

func TestListTableViewQuitting(t *testing.T) {
	model := newListTable("Test", []ListItem{{IssueID: "NUM-1"}})
	model.quitting = true
	view := model.View()

	// When quitting, view should be empty
	if view != "" {
		t.Errorf("expected empty view when quitting, got %q", view)
	}
}

func TestListTableRenderRowTruncation(t *testing.T) {
	items := []ListItem{
		{
			IssueID:    "NUM-1",
			Title:      "This is a very long title that should be truncated because it exceeds the column width",
			PlanStatus: "draft",
		},
	}
	model := newListTable("Test", items)
	view := model.View()

	// Title should be truncated (max 35 chars, but we truncate at 33 and add ...)
	if strings.Contains(view, "exceeds the column width") {
		t.Error("expected title to be truncated")
	}
	if !strings.Contains(view, "...") {
		t.Error("expected truncation indicator ...")
	}
}

func TestListTableRenderRowDefaults(t *testing.T) {
	items := []ListItem{
		{
			IssueID: "NUM-1",
			// No title, no status, no PR
		},
	}
	model := newListTable("Test", items)
	view := model.View()

	// Should show "-" for empty fields
	// Count occurrences of " - " or "| -" patterns
	if !strings.Contains(view, "-") {
		t.Error("expected dash placeholders for empty fields")
	}
}

func TestRunListTableEmpty(t *testing.T) {
	result, err := RunListTable("Test", []ListItem{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Action != ListActionQuit {
		t.Errorf("expected quit action for empty list, got %v", result.Action)
	}
}

func TestListItem(t *testing.T) {
	now := time.Now()
	item := ListItem{
		IssueID:    "NUM-123",
		Title:      "Test Item",
		Branch:     "NUM-123-test-branch",
		Path:       "/path/to/worktree",
		PlanStatus: "in-progress",
		PRNumber:   456,
		PRState:    "OPEN",
		PRURL:      "https://github.com/org/repo/pull/456",
		LastActive: now,
		Exists:     true,
	}

	if item.IssueID != "NUM-123" {
		t.Errorf("expected IssueID NUM-123, got %s", item.IssueID)
	}
	if item.PRNumber != 456 {
		t.Errorf("expected PRNumber 456, got %d", item.PRNumber)
	}
	if !item.Exists {
		t.Error("expected Exists to be true")
	}
}

func TestListResult(t *testing.T) {
	item := &ListItem{IssueID: "NUM-1"}
	result := ListResult{
		Action: ListActionSelect,
		Item:   item,
	}

	if result.Action != ListActionSelect {
		t.Errorf("expected select action, got %v", result.Action)
	}
	if result.Item.IssueID != "NUM-1" {
		t.Errorf("expected item IssueID NUM-1, got %s", result.Item.IssueID)
	}
}

func TestListTableColumns(t *testing.T) {
	// Verify column configuration
	if len(listTableColumns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(listTableColumns))
	}

	expectedColumns := []struct {
		title string
		width int
	}{
		{"ISSUE", 15},
		{"TITLE", 35},
		{"STATUS", 12},
		{"PR", 10},
	}

	for i, expected := range expectedColumns {
		if listTableColumns[i].Title != expected.title {
			t.Errorf("column %d: expected title %q, got %q", i, expected.title, listTableColumns[i].Title)
		}
		if listTableColumns[i].Width != expected.width {
			t.Errorf("column %d: expected width %d, got %d", i, expected.width, listTableColumns[i].Width)
		}
	}
}

func TestListTableActionsNoItems(t *testing.T) {
	model := newListTable("Test", []ListItem{})

	// Actions should not cause errors with empty list
	keys := []string{"enter", "o", "c", "d"}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			result := newModel.(listTableModel)

			// Should not panic or set result (no items to act on)
			if result.result != nil {
				t.Errorf("expected no result for action %s on empty list", key)
			}
		})
	}
}
