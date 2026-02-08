package linear

import (
	"context"
	"fmt"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/tracker"
)

// SyncPlan synchronizes a plan to Linear, creating/updating the main issue
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

	return nil
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

// buildPlanDescription creates a description for the Linear issue
func buildPlanDescription(p *plan.Plan) string {
	desc := ""

	if p.ProblemStatement != "" {
		desc += "## Problem Statement\n\n" + p.ProblemStatement + "\n\n"
	}

	if p.ProposedSolution != "" {
		desc += "## Proposed Solution\n\n" + p.ProposedSolution + "\n\n"
	}

	return desc
}
