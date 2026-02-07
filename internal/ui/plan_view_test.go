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
			name: "frontmatter with phases",
			rawContent: `---
id: test-plan
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

Problem here.

## Proposed Solution

Solution here.

## Phases

### Phase 1

Details about phase 1.
`,
			want: `# Test Plan

## Problem Statement

Problem here.

## Proposed Solution

Solution here.

## Phases

### Phase 1

Details about phase 1.`,
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
	if view.height != 20 {
		t.Errorf("expected default height 20, got %d", view.height)
	}
	if view.scroll != 0 {
		t.Errorf("expected initial scroll 0, got %d", view.scroll)
	}
	if view.quitting {
		t.Error("expected quitting to be false initially")
	}
}

func TestPlanViewModelView(t *testing.T) {
	tests := []struct {
		name           string
		plan           *plan.Plan
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "displays title and status",
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
		{
			name: "displays progress bar when phases exist",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusInProgress,
				Author: "testuser",
				Phases: []*plan.Phase{
					{ID: "phase-1", Title: "Phase 1", Status: plan.PhaseStatusComplete},
					{ID: "phase-2", Title: "Phase 2", Status: plan.PhaseStatusPending},
				},
			},
			wantContains: []string{
				"Test Plan",
				"Progress:",
				"50%", // 1 of 2 phases complete
			},
		},
		{
			name: "displays full markdown body",
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

This is the full problem statement that should be displayed.

## Proposed Solution

This is the complete solution.

## Custom Section

This custom content must appear in the view.
`,
			},
			wantContains: []string{
				"Test Plan",
				"Problem Statement",
				"This is the full problem statement that should be displayed.",
				"Proposed Solution",
				"This is the complete solution.",
				"Custom Section",
				"This custom content must appear in the view.",
			},
		},
		{
			name: "no progress bar without phases",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusDraft,
				Author: "testuser",
				Phases: []*plan.Phase{},
			},
			wantContains: []string{
				"Test Plan",
				"DRAFT",
			},
			wantNotContain: []string{
				"Progress:",
			},
		},
		{
			name: "displays reviewing status",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusReviewing,
				Author: "testuser",
			},
			wantContains: []string{
				"REVIEWING",
			},
		},
		{
			name: "displays approved status",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusApproved,
				Author: "testuser",
			},
			wantContains: []string{
				"APPROVED",
			},
		},
		{
			name: "displays in progress status",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusInProgress,
				Author: "testuser",
			},
			wantContains: []string{
				"IN PROGRESS",
			},
		},
		{
			name: "displays complete status",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusComplete,
				Author: "testuser",
			},
			wantContains: []string{
				"COMPLETE",
			},
		},
		{
			name: "displays help text",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusDraft,
				Author: "testuser",
			},
			wantContains: []string{
				"scroll",
				"quit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewPlanView(tt.plan)
			output := view.View()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("View() output should contain %q, got:\n%s", want, output)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("View() output should NOT contain %q, got:\n%s", notWant, output)
				}
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

func TestRenderPlanSummary(t *testing.T) {
	tests := []struct {
		name         string
		plan         *plan.Plan
		wantContains []string
	}{
		{
			name: "basic plan without phases",
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
		{
			name: "plan with phases shows progress",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Test Plan",
				Status: plan.StatusInProgress,
				Author: "testuser",
				Phases: []*plan.Phase{
					{ID: "phase-1", Title: "Phase 1", Status: plan.PhaseStatusComplete},
					{ID: "phase-2", Title: "Phase 2", Status: plan.PhaseStatusComplete},
					{ID: "phase-3", Title: "Phase 3", Status: plan.PhaseStatusPending},
					{ID: "phase-4", Title: "Phase 4", Status: plan.PhaseStatusPending},
				},
			},
			wantContains: []string{
				"Plan: Test Plan",
				"Status: in-progress",
				"Progress: 50%",
				"2/4 phases",
			},
		},
		{
			name: "plan with all phases complete",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "Completed Plan",
				Status: plan.StatusComplete,
				Author: "testuser",
				Phases: []*plan.Phase{
					{ID: "phase-1", Title: "Phase 1", Status: plan.PhaseStatusComplete},
					{ID: "phase-2", Title: "Phase 2", Status: plan.PhaseStatusComplete},
				},
			},
			wantContains: []string{
				"Plan: Completed Plan",
				"Status: complete",
				"Progress: 100%",
				"2/2 phases",
			},
		},
		{
			name: "plan with no phases complete",
			plan: &plan.Plan{
				ID:     "test-plan",
				Title:  "New Plan",
				Status: plan.StatusApproved,
				Author: "testuser",
				Phases: []*plan.Phase{
					{ID: "phase-1", Title: "Phase 1", Status: plan.PhaseStatusPending},
					{ID: "phase-2", Title: "Phase 2", Status: plan.PhaseStatusPending},
					{ID: "phase-3", Title: "Phase 3", Status: plan.PhaseStatusPending},
				},
			},
			wantContains: []string{
				"Plan: New Plan",
				"Status: approved",
				"Progress: 0%",
				"0/3 phases",
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

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		width   int
	}{
		{"0 percent", 0, 10},
		{"50 percent", 50, 10},
		{"100 percent", 100, 10},
		{"25 percent", 25, 20},
		{"75 percent", 75, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderProgressBar(tt.percent, tt.width)
			// The progress bar contains filled (█) and empty (░) characters
			// Just verify it returns something non-empty
			if got == "" {
				t.Error("renderProgressBar() should return non-empty string")
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

func TestPlanViewModelUpdate(t *testing.T) {
	p := &plan.Plan{
		ID:     "test-plan",
		Title:  "Test Plan",
		Status: plan.StatusDraft,
		Author: "testuser",
	}

	tests := []struct {
		name          string
		initialScroll int
		key           tea.KeyType
		runes         string
		wantScroll    int
		wantQuitting  bool
		wantCmdNotNil bool
	}{
		{
			name:          "scroll down with j",
			initialScroll: 0,
			runes:         "j",
			wantScroll:    1,
			wantQuitting:  false,
		},
		{
			name:          "scroll down with down arrow",
			initialScroll: 0,
			key:           tea.KeyDown,
			wantScroll:    1,
			wantQuitting:  false,
		},
		{
			name:          "scroll up with k",
			initialScroll: 5,
			runes:         "k",
			wantScroll:    4,
			wantQuitting:  false,
		},
		{
			name:          "scroll up with up arrow",
			initialScroll: 5,
			key:           tea.KeyUp,
			wantScroll:    4,
			wantQuitting:  false,
		},
		{
			name:          "scroll up at 0 stays at 0",
			initialScroll: 0,
			runes:         "k",
			wantScroll:    0,
			wantQuitting:  false,
		},
		{
			name:          "go to top with g",
			initialScroll: 50,
			runes:         "g",
			wantScroll:    0,
			wantQuitting:  false,
		},
		{
			name:          "go to top with home",
			initialScroll: 50,
			key:           tea.KeyHome,
			wantScroll:    0,
			wantQuitting:  false,
		},
		{
			name:          "go to bottom with G",
			initialScroll: 0,
			runes:         "G",
			wantScroll:    100,
			wantQuitting:  false,
		},
		{
			name:          "go to bottom with end",
			initialScroll: 0,
			key:           tea.KeyEnd,
			wantScroll:    100,
			wantQuitting:  false,
		},
		{
			name:          "quit with q",
			initialScroll: 0,
			runes:         "q",
			wantScroll:    0,
			wantQuitting:  true,
			wantCmdNotNil: true,
		},
		{
			name:          "quit with esc",
			initialScroll: 0,
			key:           tea.KeyEsc,
			wantScroll:    0,
			wantQuitting:  true,
			wantCmdNotNil: true,
		},
		{
			name:          "quit with ctrl+c",
			initialScroll: 0,
			key:           tea.KeyCtrlC,
			wantScroll:    0,
			wantQuitting:  true,
			wantCmdNotNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewPlanView(p)
			view.scroll = tt.initialScroll

			// Create a key message
			var msg tea.KeyMsg
			if tt.runes != "" {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.runes)}
			} else {
				msg = tea.KeyMsg{Type: tt.key}
			}

			model, cmd := view.Update(msg)
			updatedView := model.(PlanViewModel)

			if updatedView.scroll != tt.wantScroll {
				t.Errorf("scroll = %d, want %d", updatedView.scroll, tt.wantScroll)
			}

			if updatedView.quitting != tt.wantQuitting {
				t.Errorf("quitting = %v, want %v", updatedView.quitting, tt.wantQuitting)
			}

			if tt.wantCmdNotNil && cmd == nil {
				t.Error("expected cmd to be non-nil")
			}

			if !tt.wantCmdNotNil && cmd != nil {
				t.Error("expected cmd to be nil")
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

	// Height should be window height - 4
	expectedHeight := 26
	if updatedView.height != expectedHeight {
		t.Errorf("height = %d, want %d", updatedView.height, expectedHeight)
	}
}
