package plan

import (
	"testing"
)

func TestNewPlan(t *testing.T) {
	p := NewPlan("ENG-123", "Test Plan", "testuser")

	if p.ID != "ENG-123" {
		t.Errorf("expected ID 'ENG-123', got '%s'", p.ID)
	}
	if p.Title != "Test Plan" {
		t.Errorf("expected Title 'Test Plan', got '%s'", p.Title)
	}
	if p.Author != "testuser" {
		t.Errorf("expected Author 'testuser', got '%s'", p.Author)
	}
	if p.Status != StatusDraft {
		t.Errorf("expected Status 'draft', got '%s'", p.Status)
	}
}

func TestPlanValidate(t *testing.T) {
	tests := []struct {
		name    string
		plan    *Plan
		wantErr bool
	}{
		{
			name: "valid plan",
			plan: &Plan{
				ID:     "ENG-123",
				Title:  "Test",
				Author: "user",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			plan: &Plan{
				Title:  "Test",
				Author: "user",
			},
			wantErr: true,
		},
		{
			name: "missing title",
			plan: &Plan{
				ID:     "ENG-123",
				Author: "user",
			},
			wantErr: true,
		},
		{
			name: "missing author",
			plan: &Plan{
				ID:    "ENG-123",
				Title: "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPlanProgress(t *testing.T) {
	p := NewPlan("ENG-123", "Test", "user")
	p.Phases = []*Phase{
		{ID: "phase-1", Title: "Phase 1", Status: PhaseStatusComplete},
		{ID: "phase-2", Title: "Phase 2", Status: PhaseStatusInProgress},
		{ID: "phase-3", Title: "Phase 3", Status: PhaseStatusPending},
		{ID: "phase-4", Title: "Phase 4", Status: PhaseStatusPending},
	}

	progress := p.Progress()
	expected := 25.0 // 1 out of 4 = 25%

	if progress != expected {
		t.Errorf("expected progress %.1f%%, got %.1f%%", expected, progress)
	}
}

func TestPhaseIsBlocked(t *testing.T) {
	phases := []*Phase{
		{ID: "phase-1", Title: "Phase 1", Status: PhaseStatusComplete},
		{ID: "phase-2", Title: "Phase 2", Status: PhaseStatusPending, DependsOn: []string{"phase-1"}},
		{ID: "phase-3", Title: "Phase 3", Status: PhaseStatusPending, DependsOn: []string{"phase-2"}},
	}

	// Phase 1 has no dependencies, not blocked
	if phases[0].IsBlocked(phases) {
		t.Error("phase-1 should not be blocked")
	}

	// Phase 2 depends on phase-1 which is complete, not blocked
	if phases[1].IsBlocked(phases) {
		t.Error("phase-2 should not be blocked (phase-1 is complete)")
	}

	// Phase 3 depends on phase-2 which is pending, blocked
	if !phases[2].IsBlocked(phases) {
		t.Error("phase-3 should be blocked (phase-2 is pending)")
	}
}

func TestTopologicalSort(t *testing.T) {
	phases := []*Phase{
		{ID: "phase-1", Title: "Phase 1", DependsOn: []string{}},
		{ID: "phase-2", Title: "Phase 2", DependsOn: []string{"phase-1"}},
		{ID: "phase-3", Title: "Phase 3", DependsOn: []string{"phase-1"}},
		{ID: "phase-4", Title: "Phase 4", DependsOn: []string{"phase-2", "phase-3"}},
	}

	levels := TopologicalSort(phases)

	if len(levels) != 3 {
		t.Errorf("expected 3 levels, got %d", len(levels))
	}

	// Level 0: phase-1 (no dependencies)
	if len(levels[0]) != 1 || levels[0][0].ID != "phase-1" {
		t.Error("expected level 0 to contain only phase-1")
	}

	// Level 1: phase-2 and phase-3 (both depend only on phase-1)
	if len(levels[1]) != 2 {
		t.Errorf("expected level 1 to contain 2 phases, got %d", len(levels[1]))
	}

	// Level 2: phase-4 (depends on phase-2 and phase-3)
	if len(levels[2]) != 1 || levels[2][0].ID != "phase-4" {
		t.Error("expected level 2 to contain only phase-4")
	}
}

func TestPlanStatusTransition(t *testing.T) {
	p := NewPlan("ENG-123", "Test", "user")

	// Draft -> Reviewing is valid
	if err := p.TransitionTo(StatusReviewing); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Reviewing -> Approved is valid
	if err := p.TransitionTo(StatusApproved); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Approved -> Draft (amend) is valid
	if err := p.TransitionTo(StatusDraft); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Draft -> Complete is invalid
	if err := p.TransitionTo(StatusComplete); err == nil {
		t.Error("expected error for invalid transition draft -> complete")
	}
}

func TestPlanStatusTransitionDraftToInProgress(t *testing.T) {
	p := NewPlan("ENG-123", "Test", "user")

	// Draft -> InProgress is valid (quick implementation bypass)
	if err := p.TransitionTo(StatusInProgress); err != nil {
		t.Errorf("unexpected error for draft -> in-progress: %v", err)
	}

	if p.Status != StatusInProgress {
		t.Errorf("expected status 'in-progress', got '%s'", p.Status)
	}
}

func TestSetPhaseStatusTransitionsDraftPlan(t *testing.T) {
	p := NewPlan("ENG-123", "Test", "user")
	p.Phases = []*Phase{
		{ID: "phase-1", Title: "Phase 1", Status: PhaseStatusPending},
	}

	// Setting a phase to in-progress should transition draft plan to in-progress
	if err := p.SetPhaseStatus("phase-1", PhaseStatusInProgress); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if p.Status != StatusInProgress {
		t.Errorf("expected plan status 'in-progress', got '%s'", p.Status)
	}
}
