package linear

import (
	"context"
	"fmt"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/tracker"
)

// SyncPlan synchronizes a plan to Linear, creating/updating issues as needed
func (c *Client) SyncPlan(ctx context.Context, p *plan.Plan) error {
	// Check if main issue exists
	var mainIssue *tracker.Issue
	var err error

	if p.ID != "" {
		mainIssue, err = c.GetIssue(ctx, p.ID)
		if err != nil {
			// Issue doesn't exist, create it
			mainIssue = nil
		}
	}

	if mainIssue == nil {
		// Create the main issue
		mainIssue, err = c.CreateIssue(ctx, &tracker.Issue{
			Title:       p.Title,
			Description: buildPlanDescription(p),
			TeamID:      c.teamID,
		})
		if err != nil {
			return fmt.Errorf("failed to create main issue: %w", err)
		}
		p.ID = mainIssue.Identifier
	} else {
		// Update the main issue
		desc := buildPlanDescription(p)
		err = c.UpdateIssue(ctx, mainIssue.ID, &tracker.IssueUpdate{
			Title:       &p.Title,
			Description: &desc,
		})
		if err != nil {
			return fmt.Errorf("failed to update main issue: %w", err)
		}
	}

	// Create/update phase sub-issues
	phaseIssueIDs := make(map[string]string) // phase ID -> issue ID

	for _, phase := range p.Phases {
		var phaseIssue *tracker.Issue

		if phase.IssueID != "" {
			phaseIssue, err = c.GetIssue(ctx, phase.IssueID)
			if err != nil {
				phaseIssue = nil
			}
		}

		if phaseIssue == nil {
			// Create sub-issue for the phase
			phaseIssue, err = c.CreateSubIssue(ctx, mainIssue.ID, &tracker.Issue{
				Title:       phase.Title,
				Description: buildPhaseDescription(phase),
				TeamID:      c.teamID,
			})
			if err != nil {
				return fmt.Errorf("failed to create phase issue: %w", err)
			}
			phase.IssueID = phaseIssue.Identifier
		} else {
			// Update the phase issue
			desc := buildPhaseDescription(phase)
			err = c.UpdateIssue(ctx, phaseIssue.ID, &tracker.IssueUpdate{
				Title:       &phase.Title,
				Description: &desc,
			})
			if err != nil {
				return fmt.Errorf("failed to update phase issue: %w", err)
			}
		}

		phaseIssueIDs[phase.ID] = phaseIssue.ID
	}

	// Set up blocking relationships based on dependencies
	for _, phase := range p.Phases {
		if len(phase.DependsOn) == 0 {
			continue
		}

		blockedID := phaseIssueIDs[phase.ID]
		for _, depID := range phase.DependsOn {
			blockerID := phaseIssueIDs[depID]
			if blockerID == "" {
				continue
			}

			err = c.SetBlocking(ctx, blockerID, blockedID)
			if err != nil {
				// Blocking relationship might already exist, log but don't fail
				// TODO: Add proper logging
				continue
			}
		}
	}

	return nil
}

// SyncPhaseStatus syncs a phase's status to Linear
func (c *Client) SyncPhaseStatus(ctx context.Context, phase *plan.Phase) error {
	if phase.IssueID == "" {
		return fmt.Errorf("phase has no associated issue ID")
	}

	status := phaseStatusToTrackerStatus(phase.Status)
	return c.TransitionIssue(ctx, phase.IssueID, status)
}

// SyncPlanStatus syncs a plan's status to Linear
func (c *Client) SyncPlanStatus(ctx context.Context, p *plan.Plan) error {
	if p.ID == "" {
		return fmt.Errorf("plan has no associated issue ID")
	}

	status := planStatusToTrackerStatus(p.Status)
	return c.TransitionIssue(ctx, p.ID, status)
}

// planStatusToTrackerStatus converts a plan status to a tracker status
func planStatusToTrackerStatus(status plan.Status) tracker.Status {
	switch status {
	case plan.StatusDraft, plan.StatusReviewing:
		return tracker.StatusTodo
	case plan.StatusApproved:
		return tracker.StatusTodo
	case plan.StatusInProgress:
		return tracker.StatusInProgress
	case plan.StatusInReview:
		return tracker.StatusInReview
	case plan.StatusComplete:
		return tracker.StatusDone
	default:
		return tracker.StatusTodo
	}
}

// buildPlanDescription creates a description for the main Linear issue
func buildPlanDescription(p *plan.Plan) string {
	desc := ""

	if p.ProblemStatement != "" {
		desc += "## Problem Statement\n\n" + p.ProblemStatement + "\n\n"
	}

	if p.ProposedSolution != "" {
		desc += "## Proposed Solution\n\n" + p.ProposedSolution + "\n\n"
	}

	if len(p.Phases) > 0 {
		desc += "## Phases\n\n"
		for i, phase := range p.Phases {
			status := "⬜"
			switch phase.Status {
			case plan.PhaseStatusInProgress:
				status = "🔄"
			case plan.PhaseStatusComplete:
				status = "✅"
			case plan.PhaseStatusBlocked:
				status = "🚫"
			}
			desc += fmt.Sprintf("%d. %s %s\n", i+1, status, phase.Title)
		}
		desc += "\n"
	}

	return desc
}

// buildPhaseDescription creates a description for a phase sub-issue
func buildPhaseDescription(phase *plan.Phase) string {
	desc := ""

	if phase.Description != "" {
		desc += phase.Description + "\n\n"
	}

	if len(phase.DependsOn) > 0 {
		desc += "**Dependencies:** " + fmt.Sprintf("%v", phase.DependsOn) + "\n\n"
	}

	if len(phase.Acceptance) > 0 {
		desc += "## Acceptance Criteria\n\n"
		for _, ac := range phase.Acceptance {
			desc += fmt.Sprintf("- [ ] %s\n", ac)
		}
	}

	return desc
}

// phaseStatusToTrackerStatus converts a plan phase status to a tracker status
func phaseStatusToTrackerStatus(status plan.PhaseStatus) tracker.Status {
	switch status {
	case plan.PhaseStatusPending:
		return tracker.StatusTodo
	case plan.PhaseStatusInProgress:
		return tracker.StatusInProgress
	case plan.PhaseStatusBlocked:
		return tracker.StatusTodo // Linear doesn't have a blocked state
	case plan.PhaseStatusComplete:
		return tracker.StatusDone
	default:
		return tracker.StatusTodo
	}
}
