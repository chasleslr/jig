package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/tracker"
)

// CreateIssueFromPlan creates a new Linear issue from a plan that has no linked issue.
// Returns the created issue with its identifier (e.g., "NUM-123").
func (c *Client) CreateIssueFromPlan(ctx context.Context, p *plan.Plan) (*tracker.Issue, error) {
	if p.Title == "" {
		return nil, fmt.Errorf("plan title is required")
	}

	return c.CreateIssue(ctx, &tracker.Issue{
		Title:       p.Title,
		Description: buildPlanDescription(p),
		TeamID:      c.teamID,
	})
}

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

// SyncPlanToIssue syncs a plan's content to its associated Linear issue as a comment
// and adds a "jig-plan" label to indicate the issue has an implementation plan.
// This is called when saving a plan that has a linked issue (p.IssueID).
func (c *Client) SyncPlanToIssue(ctx context.Context, p *plan.Plan, labelName string) error {
	if p.IssueID == "" {
		return fmt.Errorf("plan has no linked issue")
	}

	// Get the issue to get team ID and existing labels
	issue, err := c.GetIssue(ctx, p.IssueID)
	if err != nil {
		return fmt.Errorf("failed to get issue %s: %w", p.IssueID, err)
	}

	// Add plan content as a comment
	commentBody := formatPlanComment(p)
	if _, err := c.AddComment(ctx, issue.ID, commentBody); err != nil {
		return fmt.Errorf("failed to add plan comment: %w", err)
	}

	// Get or create the jig-plan label
	label, err := c.GetOrCreateLabel(ctx, issue.TeamID, labelName)
	if err != nil {
		return fmt.Errorf("failed to get/create label %q: %w", labelName, err)
	}

	// Get existing label IDs from the issue
	existingLabelIDs := getIssueLabelIDs(ctx, c, issue.ID)

	// Add label to issue if not already present
	if err := c.AddLabelToIssue(ctx, issue.ID, label.ID, existingLabelIDs); err != nil {
		return fmt.Errorf("failed to add label to issue: %w", err)
	}

	return nil
}

// getIssueLabelIDs fetches the current label IDs for an issue
func getIssueLabelIDs(ctx context.Context, c *Client, issueID string) []string {
	// Re-fetch the issue to get current labels with their IDs
	// We need to use a custom query since GetIssue returns label names, not IDs
	query := `
		query GetIssueLabels($id: String!) {
			issue(id: $id) {
				labels {
					nodes {
						id
					}
				}
			}
		}
	`

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": issueID,
		},
	})
	if err != nil {
		return nil
	}

	var result struct {
		Issue struct {
			Labels struct {
				Nodes []struct {
					ID string `json:"id"`
				} `json:"nodes"`
			} `json:"labels"`
		} `json:"issue"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil
	}

	ids := make([]string, len(result.Issue.Labels.Nodes))
	for i, node := range result.Issue.Labels.Nodes {
		ids[i] = node.ID
	}
	return ids
}

// formatPlanComment formats a plan as a markdown comment for Linear
func formatPlanComment(p *plan.Plan) string {
	var sb strings.Builder

	sb.WriteString("## 📋 Implementation Plan\n\n")
	sb.WriteString(fmt.Sprintf("**Synced:** %s\n\n", time.Now().UTC().Format("2006-01-02 15:04 UTC")))
	sb.WriteString("---\n\n")

	if p.ProblemStatement != "" {
		sb.WriteString("### Problem Statement\n\n")
		sb.WriteString(p.ProblemStatement)
		sb.WriteString("\n\n")
	}

	if p.ProposedSolution != "" {
		sb.WriteString("### Proposed Solution\n\n")
		sb.WriteString(p.ProposedSolution)
		sb.WriteString("\n\n")
	}

	// Include raw content sections beyond problem/solution if present
	// This captures acceptance criteria, implementation details, etc.
	if p.RawContent != "" {
		// Extract additional sections from raw content
		additionalSections := extractAdditionalSections(p.RawContent)
		if additionalSections != "" {
			sb.WriteString(additionalSections)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("---\n\n")
	sb.WriteString("*This plan was synced by [jig](https://github.com/charleslr/jig)*")

	return sb.String()
}

// extractAdditionalSections extracts sections from raw content that aren't
// Problem Statement or Proposed Solution (those are already included separately)
func extractAdditionalSections(rawContent string) string {
	var sb strings.Builder
	lines := strings.Split(rawContent, "\n")

	inSection := false
	skipSection := false

	for _, line := range lines {
		// Check for section headers
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			sectionName := strings.TrimPrefix(strings.TrimPrefix(line, "### "), "## ")
			sectionName = strings.TrimSpace(sectionName)

			// Skip sections we already include
			if strings.Contains(strings.ToLower(sectionName), "problem statement") ||
				strings.Contains(strings.ToLower(sectionName), "proposed solution") {
				skipSection = true
				inSection = false
				continue
			}

			// Start a new section
			skipSection = false
			inSection = true
			sb.WriteString(line)
			sb.WriteString("\n")
			continue
		}

		// Write content for non-skipped sections
		if inSection && !skipSection {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return strings.TrimSpace(sb.String())
}
