package linear

import (
	"context"
	"encoding/json"
	"fmt"
)

// LinearLabel represents a label from the Linear API
type LinearLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetTeamLabels retrieves all labels for a team
func (c *Client) GetTeamLabels(ctx context.Context, teamID string) ([]LinearLabel, error) {
	query := `
		query GetTeamLabels($teamId: String!) {
			team(id: $teamId) {
				labels {
					nodes {
						id
						name
					}
				}
			}
		}
	`

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"teamId": teamID,
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Team struct {
			Labels struct {
				Nodes []LinearLabel `json:"nodes"`
			} `json:"labels"`
		} `json:"team"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Team.Labels.Nodes, nil
}

// CreateLabel creates a new label for a team
func (c *Client) CreateLabel(ctx context.Context, teamID, name string) (*LinearLabel, error) {
	query := `
		mutation CreateLabel($input: IssueLabelCreateInput!) {
			issueLabelCreate(input: $input) {
				success
				issueLabel {
					id
					name
				}
			}
		}
	`

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
				"teamId": teamID,
				"name":   name,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		IssueLabelCreate struct {
			Success    bool        `json:"success"`
			IssueLabel LinearLabel `json:"issueLabel"`
		} `json:"issueLabelCreate"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !result.IssueLabelCreate.Success {
		return nil, fmt.Errorf("failed to create label")
	}

	return &result.IssueLabelCreate.IssueLabel, nil
}

// GetOrCreateLabel finds a label by name or creates it if it doesn't exist
func (c *Client) GetOrCreateLabel(ctx context.Context, teamID, name string) (*LinearLabel, error) {
	// Get all labels for the team
	labels, err := c.GetTeamLabels(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team labels: %w", err)
	}

	// Look for existing label
	for _, label := range labels {
		if label.Name == name {
			return &label, nil
		}
	}

	// Create new label
	return c.CreateLabel(ctx, teamID, name)
}

// AddLabelToIssue adds a label to an issue, preserving existing labels
func (c *Client) AddLabelToIssue(ctx context.Context, issueID, labelID string, existingLabelIDs []string) error {
	// Check if label is already present
	for _, id := range existingLabelIDs {
		if id == labelID {
			return nil // Label already on issue
		}
	}

	// Combine existing labels with new label
	allLabelIDs := append(existingLabelIDs, labelID)

	query := `
		mutation UpdateIssueLabels($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
			}
		}
	`

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": issueID,
			"input": map[string]interface{}{
				"labelIds": allLabelIDs,
			},
		},
	})
	if err != nil {
		return err
	}

	var result struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !result.IssueUpdate.Success {
		return fmt.Errorf("failed to update issue labels")
	}

	return nil
}
