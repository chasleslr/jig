package cli

import (
	"context"
	"fmt"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/tracker"
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

// LookupPlanOptions configures plan lookup behavior
type LookupPlanOptions struct {
	FetchFromRemote bool
	Config          *config.Config
	Context         context.Context
	// TrackerGetter is an optional function to get the tracker. If nil, uses getTracker.
	// This is primarily used for testing.
	TrackerGetter func(*config.Config) (tracker.Tracker, error)
}

// lookupPlanByIDWithFallback looks up a plan with optional remote fallback.
// If the plan is not found in the local cache and FetchFromRemote is enabled,
// it attempts to fetch from the remote tracker (e.g., Linear) and caches the result.
func lookupPlanByIDWithFallback(id string, opts *LookupPlanOptions) (*plan.Plan, string, error) {
	// 1. Try local cache first
	p, planID, err := lookupPlanByID(id)
	if err != nil {
		return nil, "", err
	}

	// 2. Return if found or if remote fallback is disabled
	if p != nil || opts == nil || !opts.FetchFromRemote {
		return p, planID, nil
	}

	// 3. Attempt remote fetch
	printInfo(fmt.Sprintf("Fetching plan for %s...", id))

	// Use injected tracker getter if provided (for testing), otherwise use default
	trackerGetter := opts.TrackerGetter
	if trackerGetter == nil {
		trackerGetter = getTracker
	}

	t, err := trackerGetter(opts.Config)
	if err != nil {
		return nil, "", fmt.Errorf("could not connect to tracker: %w", err)
	}

	fetcher, ok := t.(tracker.PlanFetcher)
	if !ok {
		return nil, "", nil // Tracker doesn't support plan fetching
	}

	fetchedPlan, err := fetcher.FetchPlanFromIssue(opts.Context, id)
	if err != nil {
		return nil, "", fmt.Errorf("could not fetch plan: %w", err)
	}

	if fetchedPlan == nil {
		return nil, "", nil // Not found remotely either
	}

	// 4. Cache the fetched plan
	if err := state.DefaultCache.SavePlan(fetchedPlan); err != nil {
		printWarning(fmt.Sprintf("Could not cache plan: %v", err))
	}

	return fetchedPlan, fetchedPlan.ID, nil
}
