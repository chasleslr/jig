package ui

import (
	"strings"
	"testing"

	"github.com/charleslr/jig/internal/tracker"
)

func TestNewIssueView(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-123",
		Title:       "Test Issue",
		Description: "Test description",
		Status:      tracker.StatusInProgress,
		Labels:      []string{"bug", "high-priority"},
		URL:         "https://linear.app/team/issue/ENG-123",
	}

	m := NewIssueView(issue)

	if m.issue != issue {
		t.Error("expected issue to be set")
	}
	if m.height != 20 {
		t.Errorf("expected default height 20, got %d", m.height)
	}
	if m.width != 80 {
		t.Errorf("expected default width 80, got %d", m.width)
	}
	if m.scroll != 0 {
		t.Errorf("expected scroll to be 0, got %d", m.scroll)
	}
	if m.quitting {
		t.Error("expected quitting to be false")
	}
}

func TestIssueViewModelView(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-123",
		Title:       "Test Issue Title",
		Description: "This is a test description for the issue.",
		Status:      tracker.StatusInProgress,
		Labels:      []string{"bug", "critical"},
		URL:         "https://linear.app/team/issue/ENG-123",
	}

	m := NewIssueView(issue)
	view := m.View()

	// Check that key elements are present in the view
	if !strings.Contains(view, "ENG-123") {
		t.Error("expected view to contain issue identifier")
	}
	if !strings.Contains(view, "Test Issue Title") {
		t.Error("expected view to contain issue title")
	}
	if !strings.Contains(view, "This is a test description") {
		t.Error("expected view to contain description")
	}
	if !strings.Contains(view, "bug") {
		t.Error("expected view to contain label 'bug'")
	}
	if !strings.Contains(view, "critical") {
		t.Error("expected view to contain label 'critical'")
	}
}

func TestIssueViewModelViewWhenQuitting(t *testing.T) {
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-123",
		Title:      "Test Issue",
	}

	m := NewIssueView(issue)
	m.quitting = true
	view := m.View()

	if view != "" {
		t.Errorf("expected empty view when quitting, got %q", view)
	}
}

func TestRenderIssueContext(t *testing.T) {
	tests := []struct {
		name     string
		issue    *tracker.Issue
		contains []string
	}{
		{
			name: "full issue",
			issue: &tracker.Issue{
				ID:          "abc123",
				Identifier:  "ENG-123",
				Title:       "Implement feature X",
				Description: "We need to implement feature X for the users.",
				Status:      tracker.StatusTodo,
				Labels:      []string{"feature", "priority-high"},
				URL:         "https://linear.app/team/issue/ENG-123",
			},
			contains: []string{
				"ENG-123",
				"Implement feature X",
				"We need to implement feature X",
				"todo",
				"feature",
				"priority-high",
				"https://linear.app/team/issue/ENG-123",
			},
		},
		{
			name: "minimal issue",
			issue: &tracker.Issue{
				ID:         "abc123",
				Identifier: "ENG-456",
				Title:      "Simple task",
				Status:     tracker.StatusBacklog,
			},
			contains: []string{
				"ENG-456",
				"Simple task",
				"backlog",
			},
		},
		{
			name: "issue without labels",
			issue: &tracker.Issue{
				ID:          "abc123",
				Identifier:  "ENG-789",
				Title:       "Task without labels",
				Description: "Some description here.",
				Status:      tracker.StatusDone,
			},
			contains: []string{
				"ENG-789",
				"Task without labels",
				"Some", "description", "here", // glamour may insert ANSI codes between words
				"done",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderIssueContext(tt.issue)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("RenderIssueContext() missing %q in output:\n%s", want, result)
				}
			}
		})
	}
}

func TestRenderIssueContextWithMarkdown(t *testing.T) {
	// Test that markdown in the description is rendered properly
	issue := &tracker.Issue{
		ID:         "abc123",
		Identifier: "ENG-100",
		Title:      "Markdown Test Issue",
		Description: `## Overview

This is a **bold** statement and some *italic* text.

- Item 1
- Item 2
- Item 3

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```",
		Status: tracker.StatusInProgress,
	}

	result := RenderIssueContext(issue)

	// The markdown should be rendered, so check for content presence
	// Glamour adds ANSI codes, but the text content should still be present
	// Note: glamour may insert ANSI codes between words, so check individual words
	checks := []string{
		"ENG-100",
		"Markdown Test Issue",
		"Overview",
		"bold",
		"italic",
		"Item", // List items - glamour may style them differently
		"Hello",
	}

	for _, want := range checks {
		if !strings.Contains(result, want) {
			t.Errorf("RenderIssueContext() missing %q in markdown output:\n%s", want, result)
		}
	}
}

func TestRenderIssueContextWithWidth(t *testing.T) {
	issue := &tracker.Issue{
		ID:          "abc123",
		Identifier:  "ENG-200",
		Title:       "Width Test Issue",
		Description: "This is a test description that should be wrapped according to the specified width.",
		Status:      tracker.StatusTodo,
	}

	// Test with different widths
	result40 := RenderIssueContextWithWidth(issue, 40)
	result120 := RenderIssueContextWithWidth(issue, 120)

	// Both should contain the essential content
	for _, result := range []string{result40, result120} {
		if !strings.Contains(result, "ENG-200") {
			t.Error("expected output to contain identifier")
		}
		if !strings.Contains(result, "Width Test Issue") {
			t.Error("expected output to contain title")
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	// Test the renderMarkdown function directly
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "plain text",
			input:    "Just some plain text.",
			contains: []string{"Just", "some", "plain", "text"},
		},
		{
			name:     "bold text",
			input:    "This is **bold** text.",
			contains: []string{"bold", "text"},
		},
		{
			name:     "list",
			input:    "- First\n- Second\n- Third",
			contains: []string{"First", "Second", "Third"},
		},
		{
			name:     "code block",
			input:    "```\ncode here\n```",
			contains: []string{"code here"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderMarkdown(tt.input, 80)
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("renderMarkdown(%q) missing %q in output:\n%s", tt.input, want, result)
				}
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{
			name:     "zero width returns original",
			text:     "Hello world",
			width:    0,
			expected: "Hello world",
		},
		{
			name:     "negative width returns original",
			text:     "Hello world",
			width:    -1,
			expected: "Hello world",
		},
		{
			name:     "short text unchanged",
			text:     "Hi",
			width:    80,
			expected: "Hi",
		},
		{
			name:     "text wraps at width",
			text:     "Hello world test",
			width:    10,
			expected: "Hello\nworld test",
		},
		{
			name:     "preserves newlines",
			text:     "Line 1\nLine 2",
			width:    80,
			expected: "Line 1\nLine 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)
			if result != tt.expected {
				t.Errorf("wrapText(%q, %d) = %q, want %q", tt.text, tt.width, result, tt.expected)
			}
		})
	}
}

func TestFormatIssueStatus(t *testing.T) {
	tests := []struct {
		status   tracker.Status
		contains string
	}{
		{tracker.StatusBacklog, "BACKLOG"},
		{tracker.StatusTodo, "TODO"},
		{tracker.StatusInProgress, "IN PROGRESS"},
		{tracker.StatusInReview, "IN REVIEW"},
		{tracker.StatusDone, "DONE"},
		{tracker.StatusCanceled, "CANCELED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := formatIssueStatus(tt.status)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatIssueStatus(%s) = %q, want to contain %q", tt.status, result, tt.contains)
			}
		})
	}
}
