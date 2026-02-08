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

