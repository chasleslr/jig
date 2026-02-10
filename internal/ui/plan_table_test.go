package ui

import (
	"testing"
	"time"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/state"
)

func TestCachedPlanTableColumns(t *testing.T) {
	columns := CachedPlanTableColumns()

	if len(columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(columns))
	}

	expectedTitles := []string{"ID", "Title", "Status", "Synced"}
	for i, col := range columns {
		if col.Title != expectedTitles[i] {
			t.Errorf("column %d: expected title %q, got %q", i, expectedTitles[i], col.Title)
		}
		if col.Width <= 0 {
			t.Errorf("column %d: expected positive width, got %d", i, col.Width)
		}
	}
}

func TestCachedPlanToTableRow(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	tests := []struct {
		name           string
		cached         *state.CachedPlan
		expectedCells  []string
		expectedNil    bool
	}{
		{
			name:        "nil cached plan",
			cached:      nil,
			expectedNil: true,
		},
		{
			name: "nil plan inside cached",
			cached: &state.CachedPlan{
				Plan: nil,
			},
			expectedNil: true,
		},
		{
			name: "plan with no linked issue",
			cached: &state.CachedPlan{
				Plan: &plan.Plan{
					ID:      "PLAN-1",
					Title:   "Test Plan",
					Status:  plan.StatusDraft,
					IssueID: "",
				},
				UpdatedAt: now,
			},
			expectedCells: []string{"PLAN-1", "Test Plan", "draft", "-"},
		},
		{
			name: "plan never synced (needs sync)",
			cached: &state.CachedPlan{
				Plan: &plan.Plan{
					ID:      "PLAN-2",
					Title:   "Unsynced Plan",
					Status:  plan.StatusInProgress,
					IssueID: "NUM-123",
				},
				UpdatedAt: now,
				SyncedAt:  nil,
			},
			expectedCells: []string{"PLAN-2", "Unsynced Plan", "in-progress", "no"},
		},
		{
			name: "plan synced and up to date",
			cached: &state.CachedPlan{
				Plan: &plan.Plan{
					ID:      "PLAN-3",
					Title:   "Synced Plan",
					Status:  plan.StatusComplete,
					IssueID: "NUM-456",
				},
				UpdatedAt: past,
				SyncedAt:  &now,
			},
			expectedCells: []string{"PLAN-3", "Synced Plan", "complete", "yes"},
		},
		{
			name: "plan updated after sync (needs sync)",
			cached: &state.CachedPlan{
				Plan: &plan.Plan{
					ID:      "PLAN-4",
					Title:   "Updated Plan",
					Status:  plan.StatusDraft,
					IssueID: "NUM-789",
				},
				UpdatedAt: now,
				SyncedAt:  &past,
			},
			expectedCells: []string{"PLAN-4", "Updated Plan", "draft", "no"},
		},
		{
			name: "plan with empty status defaults to draft",
			cached: &state.CachedPlan{
				Plan: &plan.Plan{
					ID:      "PLAN-5",
					Title:   "No Status Plan",
					Status:  "",
					IssueID: "",
				},
				UpdatedAt: now,
			},
			expectedCells: []string{"PLAN-5", "No Status Plan", "draft", "-"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := CachedPlanToTableRow(tt.cached)

			if tt.expectedNil {
				if len(row.Cells) != 0 {
					t.Errorf("expected empty row for nil input, got %v", row.Cells)
				}
				return
			}

			if len(row.Cells) != len(tt.expectedCells) {
				t.Fatalf("expected %d cells, got %d", len(tt.expectedCells), len(row.Cells))
			}

			for i, cell := range row.Cells {
				if cell != tt.expectedCells[i] {
					t.Errorf("cell %d: expected %q, got %q", i, tt.expectedCells[i], cell)
				}
			}

			// Verify Value is set to the plan
			if row.Value == nil {
				t.Error("expected Value to be set")
			} else if p, ok := row.Value.(*plan.Plan); !ok {
				t.Error("expected Value to be *plan.Plan")
			} else if p.ID != tt.cached.Plan.ID {
				t.Errorf("expected Value.ID %q, got %q", tt.cached.Plan.ID, p.ID)
			}
		})
	}
}

func TestCachedPlansToTableRows(t *testing.T) {
	now := time.Now()

	t.Run("empty slice", func(t *testing.T) {
		rows := CachedPlansToTableRows([]*state.CachedPlan{})
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("nil slice", func(t *testing.T) {
		rows := CachedPlansToTableRows(nil)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("multiple plans", func(t *testing.T) {
		plans := []*state.CachedPlan{
			{
				Plan: &plan.Plan{
					ID:      "PLAN-A",
					Title:   "Plan A",
					Status:  plan.StatusDraft,
					IssueID: "NUM-1",
				},
				UpdatedAt: now,
			},
			{
				Plan: &plan.Plan{
					ID:      "PLAN-B",
					Title:   "Plan B",
					Status:  plan.StatusComplete,
					IssueID: "",
				},
				UpdatedAt: now,
			},
		}

		rows := CachedPlansToTableRows(plans)

		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}

		// Verify first row
		if rows[0].Cells[0] != "PLAN-A" {
			t.Errorf("row 0: expected ID 'PLAN-A', got %q", rows[0].Cells[0])
		}
		if rows[0].Cells[3] != "no" { // Has issue, never synced
			t.Errorf("row 0: expected synced 'no', got %q", rows[0].Cells[3])
		}

		// Verify second row
		if rows[1].Cells[0] != "PLAN-B" {
			t.Errorf("row 1: expected ID 'PLAN-B', got %q", rows[1].Cells[0])
		}
		if rows[1].Cells[3] != "-" { // No issue
			t.Errorf("row 1: expected synced '-', got %q", rows[1].Cells[3])
		}
	})

	t.Run("handles nil plan in slice", func(t *testing.T) {
		plans := []*state.CachedPlan{
			{
				Plan: &plan.Plan{
					ID:    "PLAN-VALID",
					Title: "Valid Plan",
				},
				UpdatedAt: now,
			},
			nil, // nil entry
			{
				Plan: nil, // nil plan inside
			},
		}

		rows := CachedPlansToTableRows(plans)

		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(rows))
		}

		// First row should be valid
		if rows[0].Cells[0] != "PLAN-VALID" {
			t.Errorf("row 0: expected valid plan, got %v", rows[0].Cells)
		}

		// Second and third rows should be empty (nil handling)
		if len(rows[1].Cells) != 0 {
			t.Errorf("row 1: expected empty cells for nil, got %v", rows[1].Cells)
		}
		if len(rows[2].Cells) != 0 {
			t.Errorf("row 2: expected empty cells for nil plan, got %v", rows[2].Cells)
		}
	})
}

func TestRunCachedPlanTable_EmptyPlans(t *testing.T) {
	result, ok, err := RunCachedPlanTable("Test", []*state.CachedPlan{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false for empty plans")
	}
	if result != nil {
		t.Error("expected nil result for empty plans")
	}
}
