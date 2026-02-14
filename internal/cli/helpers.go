package cli

import (
	"fmt"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/ui"
)

// lookupPlanByID looks up a plan by ID, supporting both plan IDs (PLAN-xxx) and issue IDs (NUM-xxx).
// If both match different plans, prompts the user to select one.
// Returns the plan and the actual ID to use for further lookups (e.g., GetPlanMarkdown).
func lookupPlanByID(id string) (*plan.Plan, string, error) {
	result, err := state.DefaultCache.LookupPlan(id)
	if err != nil {
		return nil, "", fmt.Errorf("failed to lookup plan: %w", err)
	}

	if result.HasConflict() {
		// Both a plan ID and issue ID matched different plans - let user choose
		printWarning(fmt.Sprintf("'%s' matches both a plan ID and an issue ID", id))
		options := []ui.SelectOption{
			{
				Label:       fmt.Sprintf("Plan: %s", result.ByPlanID.Title),
				Value:       "plan",
				Description: fmt.Sprintf("Plan ID: %s, Issue: %s", result.ByPlanID.ID, result.ByPlanID.IssueID),
			},
			{
				Label:       fmt.Sprintf("Plan: %s", result.ByIssueID.Title),
				Value:       "issue",
				Description: fmt.Sprintf("Plan ID: %s, Issue: %s", result.ByIssueID.ID, result.ByIssueID.IssueID),
			},
		}
		selected, err := ui.RunSelect("Which plan do you want to use?", options)
		if err != nil {
			return nil, "", fmt.Errorf("failed to select plan: %w", err)
		}
		if selected == "" {
			return nil, "", nil // User cancelled
		}
		if selected == "plan" {
			return result.ByPlanID, result.ByPlanID.ID, nil
		}
		return result.ByIssueID, result.ByIssueID.ID, nil
	}

	p := result.Plan()
	if p == nil {
		return nil, "", nil
	}
	return p, p.ID, nil
}
