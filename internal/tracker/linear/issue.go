package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/charleslr/jig/internal/tracker"
)

// Ensure Client implements tracker.Tracker
var _ tracker.Tracker = (*Client)(nil)

// CreateIssue creates a new issue in Linear
func (c *Client) CreateIssue(ctx context.Context, issue *tracker.Issue) (*tracker.Issue, error) {
	query := `
		mutation CreateIssue($input: IssueCreateInput!) {
			issueCreate(input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					priority
					url
					createdAt
					updatedAt
					state {
						id
						name
						type
					}
					team {
						id
						key
					}
				}
			}
		}
	`

	input := map[string]interface{}{
		"title": issue.Title,
	}

	if issue.Description != "" {
		input["description"] = issue.Description
	}

	teamID := issue.TeamID
	if teamID == "" {
		teamID = c.teamID
	}
	if teamID != "" {
		input["teamId"] = teamID
	}

	if c.projectID != "" {
		input["projectId"] = c.projectID
	}

	if issue.Priority != tracker.PriorityNone {
		input["priority"] = int(issue.Priority)
	}

	if issue.ParentID != "" {
		input["parentId"] = issue.ParentID
	}

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"input": input,
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		IssueCreate struct {
			Success bool        `json:"success"`
			Issue   LinearIssue `json:"issue"`
		} `json:"issueCreate"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !result.IssueCreate.Success {
		return nil, fmt.Errorf("failed to create issue")
	}

	return linearIssueToTracker(&result.IssueCreate.Issue), nil
}

// UpdateIssue updates an existing issue in Linear
func (c *Client) UpdateIssue(ctx context.Context, id string, updates *tracker.IssueUpdate) error {
	query := `
		mutation UpdateIssue($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
			}
		}
	`

	input := make(map[string]interface{})

	if updates.Title != nil {
		input["title"] = *updates.Title
	}
	if updates.Description != nil {
		input["description"] = *updates.Description
	}
	if updates.Priority != nil {
		input["priority"] = int(*updates.Priority)
	}
	if updates.ParentID != nil {
		input["parentId"] = *updates.ParentID
	}

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id":    id,
			"input": input,
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
		return fmt.Errorf("failed to update issue")
	}

	return nil
}

// GetIssue retrieves an issue by ID or identifier
func (c *Client) GetIssue(ctx context.Context, id string) (*tracker.Issue, error) {
	// Try by identifier first (e.g., "ENG-123")
	if strings.Contains(id, "-") {
		return c.getIssueByIdentifier(ctx, id)
	}

	query := `
		query GetIssue($id: String!) {
			issue(id: $id) {
				id
				identifier
				title
				description
				priority
				url
				createdAt
				updatedAt
				state {
					id
					name
					type
				}
				assignee {
					id
					name
				}
				team {
					id
					key
				}
				project {
					id
					name
				}
				parent {
					id
					identifier
				}
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
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Issue LinearIssue `json:"issue"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return linearIssueToTracker(&result.Issue), nil
}

// getIssueByIdentifier retrieves an issue by its human-readable identifier
func (c *Client) getIssueByIdentifier(ctx context.Context, identifier string) (*tracker.Issue, error) {
	query := `
		query GetIssueByIdentifier($filter: IssueFilter!) {
			issues(filter: $filter, first: 1) {
				nodes {
					id
					identifier
					title
					description
					priority
					url
					createdAt
					updatedAt
					state {
						id
						name
						type
					}
					assignee {
						id
						name
					}
					team {
						id
						key
					}
					project {
						id
						name
					}
					parent {
						id
						identifier
					}
					labels {
						nodes {
							id
							name
						}
					}
				}
			}
		}
	`

	// Parse the identifier (e.g., "ENG-123" -> team key and number)
	parts := strings.SplitN(identifier, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid identifier format: %s", identifier)
	}

	issueNumber, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid issue number in identifier %s: %w", identifier, err)
	}

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"filter": map[string]interface{}{
				"team": map[string]interface{}{
					"key": map[string]interface{}{
						"eq": parts[0],
					},
				},
				"number": map[string]interface{}{
					"eq": issueNumber,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Issues struct {
			Nodes []LinearIssue `json:"nodes"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Issues.Nodes) == 0 {
		return nil, fmt.Errorf("issue not found: %s", identifier)
	}

	return linearIssueToTracker(&result.Issues.Nodes[0]), nil
}

// SearchIssues searches for issues matching a query
func (c *Client) SearchIssues(ctx context.Context, searchQuery string) ([]*tracker.Issue, error) {
	query := `
		query SearchIssues($filter: IssueFilter!, $first: Int) {
			issues(filter: $filter, first: $first) {
				nodes {
					id
					identifier
					title
					description
					priority
					url
					createdAt
					updatedAt
					state {
						id
						name
						type
					}
					assignee {
						id
						name
					}
					team {
						id
						key
					}
				}
			}
		}
	`

	filter := map[string]interface{}{
		"or": []map[string]interface{}{
			{
				"title": map[string]interface{}{
					"containsIgnoreCase": searchQuery,
				},
			},
			{
				"description": map[string]interface{}{
					"containsIgnoreCase": searchQuery,
				},
			},
		},
	}

	if c.teamID != "" {
		filter["team"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": c.teamID,
			},
		}
	}

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"filter": filter,
			"first":  50,
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Issues struct {
			Nodes []LinearIssue `json:"nodes"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	issues := make([]*tracker.Issue, len(result.Issues.Nodes))
	for i, node := range result.Issues.Nodes {
		issues[i] = linearIssueToTracker(&node)
	}

	return issues, nil
}

// CreateSubIssue creates a sub-issue under a parent issue
func (c *Client) CreateSubIssue(ctx context.Context, parentID string, issue *tracker.Issue) (*tracker.Issue, error) {
	issue.ParentID = parentID
	return c.CreateIssue(ctx, issue)
}

// GetSubIssues retrieves all sub-issues for a parent issue
func (c *Client) GetSubIssues(ctx context.Context, parentID string) ([]*tracker.Issue, error) {
	query := `
		query GetSubIssues($id: String!) {
			issue(id: $id) {
				children {
					nodes {
						id
						identifier
						title
						description
						priority
						url
						createdAt
						updatedAt
						state {
							id
							name
							type
						}
					}
				}
			}
		}
	`

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": parentID,
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Issue struct {
			Children struct {
				Nodes []LinearIssue `json:"nodes"`
			} `json:"children"`
		} `json:"issue"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	issues := make([]*tracker.Issue, len(result.Issue.Children.Nodes))
	for i, node := range result.Issue.Children.Nodes {
		issues[i] = linearIssueToTracker(&node)
	}

	return issues, nil
}

// SetBlocking sets a blocking relationship between two issues
func (c *Client) SetBlocking(ctx context.Context, blockerID, blockedID string) error {
	query := `
		mutation CreateIssueRelation($input: IssueRelationCreateInput!) {
			issueRelationCreate(input: $input) {
				success
			}
		}
	`

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
				"issueId":        blockedID,
				"relatedIssueId": blockerID,
				"type":           "blocks",
			},
		},
	})
	if err != nil {
		return err
	}

	var result struct {
		IssueRelationCreate struct {
			Success bool `json:"success"`
		} `json:"issueRelationCreate"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !result.IssueRelationCreate.Success {
		return fmt.Errorf("failed to create blocking relation")
	}

	return nil
}

// GetBlockedBy retrieves issues that are blocking the given issue
func (c *Client) GetBlockedBy(ctx context.Context, issueID string) ([]*tracker.Issue, error) {
	query := `
		query GetBlockedBy($id: String!) {
			issue(id: $id) {
				relations {
					nodes {
						type
						relatedIssue {
							id
							identifier
							title
							state {
								name
								type
							}
						}
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
		return nil, err
	}

	var result struct {
		Issue struct {
			Relations struct {
				Nodes []struct {
					Type         string      `json:"type"`
					RelatedIssue LinearIssue `json:"relatedIssue"`
				} `json:"nodes"`
			} `json:"relations"`
		} `json:"issue"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var blockers []*tracker.Issue
	for _, rel := range result.Issue.Relations.Nodes {
		if rel.Type == "blocks" {
			blockers = append(blockers, linearIssueToTracker(&rel.RelatedIssue))
		}
	}

	return blockers, nil
}

// AddComment adds a comment to an issue
func (c *Client) AddComment(ctx context.Context, issueID string, body string) (*tracker.Comment, error) {
	query := `
		mutation CreateComment($input: CommentCreateInput!) {
			commentCreate(input: $input) {
				success
				comment {
					id
					body
					createdAt
					updatedAt
					user {
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
			"input": map[string]interface{}{
				"issueId": issueID,
				"body":    body,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		CommentCreate struct {
			Success bool          `json:"success"`
			Comment LinearComment `json:"comment"`
		} `json:"commentCreate"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !result.CommentCreate.Success {
		return nil, fmt.Errorf("failed to create comment")
	}

	return linearCommentToTracker(&result.CommentCreate.Comment), nil
}

// GetComments retrieves all comments for an issue
func (c *Client) GetComments(ctx context.Context, issueID string) ([]*tracker.Comment, error) {
	query := `
		query GetComments($id: String!) {
			issue(id: $id) {
				comments {
					nodes {
						id
						body
						createdAt
						updatedAt
						user {
							id
							name
						}
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
		return nil, err
	}

	var result struct {
		Issue struct {
			Comments struct {
				Nodes []LinearComment `json:"nodes"`
			} `json:"comments"`
		} `json:"issue"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	comments := make([]*tracker.Comment, len(result.Issue.Comments.Nodes))
	for i, node := range result.Issue.Comments.Nodes {
		comments[i] = linearCommentToTracker(&node)
	}

	return comments, nil
}

// TransitionIssue changes the status of an issue
func (c *Client) TransitionIssue(ctx context.Context, id string, status tracker.Status) error {
	// First, get available workflow states
	states, err := c.getWorkflowStates(ctx, id)
	if err != nil {
		return err
	}

	// Find the matching state
	var stateID string
	for _, state := range states {
		if statusMatches(state, status) {
			stateID = state.ID
			break
		}
	}

	if stateID == "" {
		return fmt.Errorf("no matching workflow state for status: %s", status)
	}

	query := `
		mutation UpdateIssueState($id: String!, $stateId: String!) {
			issueUpdate(id: $id, input: { stateId: $stateId }) {
				success
			}
		}
	`

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id":      id,
			"stateId": stateID,
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
		return fmt.Errorf("failed to update issue state")
	}

	return nil
}

// GetAvailableStatuses returns available status transitions
func (c *Client) GetAvailableStatuses(ctx context.Context, id string) ([]tracker.Status, error) {
	states, err := c.getWorkflowStates(ctx, id)
	if err != nil {
		return nil, err
	}

	statuses := make([]tracker.Status, 0, len(states))
	for _, state := range states {
		status := linearStateToStatus(state.Type)
		// Avoid duplicates
		found := false
		for _, s := range statuses {
			if s == status {
				found = true
				break
			}
		}
		if !found {
			statuses = append(statuses, status)
		}
	}

	return statuses, nil
}

// getWorkflowStates retrieves workflow states for an issue's team
func (c *Client) getWorkflowStates(ctx context.Context, issueID string) ([]LinearWorkflowState, error) {
	query := `
		query GetWorkflowStates($id: String!) {
			issue(id: $id) {
				team {
					states {
						nodes {
							id
							name
							type
						}
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
		return nil, err
	}

	var result struct {
		Issue struct {
			Team struct {
				States struct {
					Nodes []LinearWorkflowState `json:"nodes"`
				} `json:"states"`
			} `json:"team"`
		} `json:"issue"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Issue.Team.States.Nodes, nil
}

// GetTeams retrieves all teams accessible by the API key
func (c *Client) GetTeams(ctx context.Context) ([]tracker.Team, error) {
	query := `
		query GetTeams {
			teams {
				nodes {
					id
					name
					key
				}
			}
		}
	`

	resp, err := c.execute(ctx, &GraphQLRequest{
		Query: query,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Teams struct {
			Nodes []LinearTeam `json:"nodes"`
		} `json:"teams"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	teams := make([]tracker.Team, len(result.Teams.Nodes))
	for i, node := range result.Teams.Nodes {
		teams[i] = tracker.Team{
			ID:   node.ID,
			Name: node.Name,
			Key:  node.Key,
		}
	}

	return teams, nil
}

// GetProjects retrieves all projects for a team
func (c *Client) GetProjects(ctx context.Context, teamID string) ([]tracker.Project, error) {
	query := `
		query GetProjects($teamId: String!) {
			team(id: $teamId) {
				projects {
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
			Projects struct {
				Nodes []LinearProject `json:"nodes"`
			} `json:"projects"`
		} `json:"team"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	projects := make([]tracker.Project, len(result.Team.Projects.Nodes))
	for i, node := range result.Team.Projects.Nodes {
		projects[i] = tracker.Project{
			ID:     node.ID,
			Name:   node.Name,
			TeamID: teamID,
		}
	}

	return projects, nil
}

// Helper functions

func linearIssueToTracker(li *LinearIssue) *tracker.Issue {
	issue := &tracker.Issue{
		ID:          li.ID,
		Identifier:  li.Identifier,
		Title:       li.Title,
		Description: li.Description,
		Priority:    tracker.Priority(li.Priority),
		URL:         li.URL,
		CreatedAt:   li.CreatedAt,
		UpdatedAt:   li.UpdatedAt,
		Status:      linearStateToStatus(li.State.Type),
	}

	if li.Assignee != nil {
		issue.Assignee = li.Assignee.Name
	}

	if li.Team != nil {
		issue.TeamID = li.Team.ID
	}

	if li.Project != nil {
		issue.ProjectID = li.Project.ID
	}

	if li.Parent != nil {
		issue.ParentID = li.Parent.ID
	}

	for _, label := range li.Labels.Nodes {
		issue.Labels = append(issue.Labels, label.Name)
	}

	return issue
}

func linearCommentToTracker(lc *LinearComment) *tracker.Comment {
	return &tracker.Comment{
		ID:        lc.ID,
		Body:      lc.Body,
		Author:    lc.User.Name,
		CreatedAt: lc.CreatedAt,
		UpdatedAt: lc.UpdatedAt,
	}
}

func linearStateToStatus(stateType string) tracker.Status {
	switch stateType {
	case "backlog":
		return tracker.StatusBacklog
	case "unstarted":
		return tracker.StatusTodo
	case "started":
		return tracker.StatusInProgress
	case "completed":
		return tracker.StatusDone
	case "canceled":
		return tracker.StatusCanceled
	default:
		return tracker.StatusTodo
	}
}

func statusMatches(state LinearWorkflowState, status tracker.Status) bool {
	switch status {
	case tracker.StatusBacklog:
		return state.Type == "backlog"
	case tracker.StatusTodo:
		return state.Type == "unstarted"
	case tracker.StatusInProgress:
		return state.Type == "started"
	case tracker.StatusInReview:
		return state.Type == "started" && strings.Contains(strings.ToLower(state.Name), "review")
	case tracker.StatusDone:
		return state.Type == "completed"
	case tracker.StatusCanceled:
		return state.Type == "canceled"
	default:
		return false
	}
}
