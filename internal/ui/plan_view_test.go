package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charleslr/jig/internal/plan"
)

func TestExtractMarkdownBody(t *testing.T) {
	tests := []struct {
		name       string
		rawContent string
		want       string
	}{
		{
			name: "standard frontmatter",
			rawContent: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

This is the problem.

## Proposed Solution

This is the solution.
`,
			want: `# Test Plan

## Problem Statement

This is the problem.

## Proposed Solution

This is the solution.`,
		},
		{
			name:       "no frontmatter",
			rawContent: `# Just Markdown

No frontmatter here.
`,
			want: `# Just Markdown

No frontmatter here.
`,
		},
		{
			name: "malformed frontmatter - missing closing delimiter",
			rawContent: `---
id: test-plan
title: Test Plan

# This should be returned as-is
`,
			want: `---
id: test-plan
title: Test Plan

# This should be returned as-is
`,
		},
		{
			name:       "empty content",
			rawContent: "",
			want:       "",
		},
		{
			name: "frontmatter only",
			rawContent: `---
id: test-plan
title: Test Plan
---`,
			want: "",
		},
		{
			name: "frontmatter with custom sections",
			rawContent: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.

## Custom Section

This custom content should be included.

### Nested Custom

Even nested content.

## Another Custom Section

More content here with **bold** and _italic_.
`,
			want: `# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.

## Custom Section

This custom content should be included.

### Nested Custom

Even nested content.

## Another Custom Section

More content here with **bold** and _italic_.`,
		},
		{
			name: "frontmatter with dashes in content",
			rawContent: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

This has --- dashes in the middle.

## Proposed Solution

More --- dashes --- here.
`,
			want: `# Test Plan

## Problem Statement

This has --- dashes in the middle.

## Proposed Solution

More --- dashes --- here.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMarkdownBody(tt.rawContent)
			if got != tt.want {
				t.Errorf("extractMarkdownBody() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestNewPlanView(t *testing.T) {
	p := &plan.Plan{
		ID:     "test-plan",
		Title:  "Test Plan",
		Status: plan.StatusDraft,
		Author: "testuser",
	}

	view := NewPlanView(p)

	if view.plan != p {
		t.Error("expected plan to be set")
	}
	if view.ready {
		t.Error("expected ready to be false initially")
	}
	if view.quitting {
		t.Error("expected quitting to be false initially")
	}
}

func TestPlanViewModelRenderHeader(t *testing.T) {
	tests := []struct {
		name         string
		plan         *plan.Plan
		wantContains []string
	}{
		{
			name: "header with title and status",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan Title",
				Status: plan.StatusDraft,
				Author: "testuser",
			},
			wantContains: []string{
				"Test Plan Title",
				"DRAFT",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewPlanView(tt.plan)
			output := view.renderHeader()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("renderHeader() should contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

func TestPlanViewModelRenderContent(t *testing.T) {
	tests := []struct {
		name         string
		plan         *plan.Plan
		wantContains []string
	}{
		{
			name: "renders full markdown body",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusDraft,
				Author: "testuser",
				RawContent: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

This is the full problem statement.

## Proposed Solution

This is the solution.

## Custom Section

Custom content here.
`,
			},
			wantContains: []string{
				"Problem Statement",
				"This is the full problem statement.",
				"Proposed Solution",
				"Custom Section",
				"Custom content here.",
			},
		},
		{
			name: "empty content when no raw content",
			plan: &plan.Plan{
				ID:         "test-plan",
				Title:      "Test Plan",
				Status:     plan.StatusDraft,
				Author:     "testuser",
				RawContent: "",
			},
			wantContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewPlanView(tt.plan)
			output := view.renderContent()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("renderContent() should contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

func TestPlanViewModelHeaderHeight(t *testing.T) {
	tests := []struct {
		name       string
		plan       *plan.Plan
		wantHeight int
	}{
		{
			name: "basic plan",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusDraft,
				Author: "testuser",
			},
			wantHeight: 4, // title + separator + blank + status
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewPlanView(tt.plan)
			got := view.headerHeight()
			if got != tt.wantHeight {
				t.Errorf("headerHeight() = %d, want %d", got, tt.wantHeight)
			}
		})
	}
}

func TestPlanViewModelViewQuitting(t *testing.T) {
	p := &plan.Plan{
		ID:     "test-plan",
		Title:  "Test Plan",
		Status: plan.StatusDraft,
		Author: "testuser",
	}

	view := NewPlanView(p)
	view.quitting = true

	output := view.View()
	if output != "" {
		t.Errorf("expected empty string when quitting, got: %s", output)
	}
}

func TestPlanViewModelViewNotReady(t *testing.T) {
	p := &plan.Plan{
		ID:     "test-plan",
		Title:  "Test Plan",
		Status: plan.StatusDraft,
		Author: "testuser",
	}

	view := NewPlanView(p)
	// ready is false by default

	output := view.View()
	if output != "Loading..." {
		t.Errorf("expected 'Loading...' when not ready, got: %s", output)
	}
}

func TestRenderPlanSummary(t *testing.T) {
	tests := []struct {
		name         string
		plan         *plan.Plan
		wantContains []string
	}{
		{
			name: "basic plan",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan Title",
				Status: plan.StatusDraft,
				Author: "testuser",
			},
			wantContains: []string{
				"Plan: Test Plan Title",
				"Status: draft",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderPlanSummary(tt.plan)

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("RenderPlanSummary() should contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

func TestFormatPlanStatus(t *testing.T) {
	tests := []struct {
		status plan.Status
		want   string
	}{
		{plan.StatusDraft, "DRAFT"},
		{plan.StatusReviewing, "REVIEWING"},
		{plan.StatusApproved, "APPROVED"},
		{plan.StatusInProgress, "IN PROGRESS"},
		{plan.StatusComplete, "COMPLETE"},
		{plan.Status("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := formatPlanStatus(tt.status)
			if !strings.Contains(got, tt.want) {
				t.Errorf("formatPlanStatus(%s) should contain %q, got %q", tt.status, tt.want, got)
			}
		})
	}
}

func TestPlanViewModelInit(t *testing.T) {
	p := &plan.Plan{
		ID:     "test-plan",
		Title:  "Test Plan",
		Status: plan.StatusDraft,
		Author: "testuser",
	}

	view := NewPlanView(p)
	cmd := view.Init()

	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestPlanViewModelUpdateQuit(t *testing.T) {
	p := &plan.Plan{
		ID:     "test-plan",
		Title:  "Test Plan",
		Status: plan.StatusDraft,
		Author: "testuser",
	}

	tests := []struct {
		name string
		key  tea.KeyType
		char string
	}{
		{"quit with q", tea.KeyRunes, "q"},
		{"quit with esc", tea.KeyEsc, ""},
		{"quit with ctrl+c", tea.KeyCtrlC, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewPlanView(p)

			var msg tea.KeyMsg
			if tt.char != "" {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.char)}
			} else {
				msg = tea.KeyMsg{Type: tt.key}
			}

			model, cmd := view.Update(msg)
			updatedView := model.(PlanViewModel)

			if !updatedView.quitting {
				t.Error("expected quitting to be true")
			}

			if cmd == nil {
				t.Error("expected cmd to be non-nil (tea.Quit)")
			}
		})
	}
}

func TestPlanViewModelUpdateWindowSize(t *testing.T) {
	p := &plan.Plan{
		ID:     "test-plan",
		Title:  "Test Plan",
		Status: plan.StatusDraft,
		Author: "testuser",
	}

	view := NewPlanView(p)

	// Simulate window size message
	msg := tea.WindowSizeMsg{Height: 30, Width: 80}
	model, _ := view.Update(msg)
	updatedView := model.(PlanViewModel)

	// View should be ready after receiving window size
	if !updatedView.ready {
		t.Error("expected ready to be true after WindowSizeMsg")
	}

	// Viewport should be initialized
	expectedHeight := 30 - updatedView.headerHeight() - 2 // footer
	if updatedView.viewport.Height != expectedHeight {
		t.Errorf("viewport.Height = %d, want %d", updatedView.viewport.Height, expectedHeight)
	}
}
