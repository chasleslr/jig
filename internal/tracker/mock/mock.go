package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/charleslr/jig/internal/tracker"
)

// Client is a mock implementation of the Tracker interface for testing
type Client struct {
	mu        sync.RWMutex
	issues    map[string]*tracker.Issue
	comments  map[string][]*tracker.Comment
	relations map[string][]string // issueID -> blockedByIDs
	counter   int
}

// NewClient creates a new mock tracker client
func NewClient() *Client {
	return &Client{
		issues:    make(map[string]*tracker.Issue),
		comments:  make(map[string][]*tracker.Comment),
		relations: make(map[string][]string),
	}
}

// CreateIssue creates a new mock issue
func (c *Client) CreateIssue(ctx context.Context, issue *tracker.Issue) (*tracker.Issue, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.counter++
	id := fmt.Sprintf("mock-%d", c.counter)
	identifier := fmt.Sprintf("MOCK-%d", c.counter)

	now := time.Now()
	newIssue := &tracker.Issue{
		ID:          id,
		Identifier:  identifier,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      tracker.StatusTodo,
		Priority:    issue.Priority,
		ParentID:    issue.ParentID,
		TeamID:      issue.TeamID,
		ProjectID:   issue.ProjectID,
		CreatedAt:   now,
		UpdatedAt:   now,
		URL:         fmt.Sprintf("https://mock.linear.app/issue/%s", identifier),
	}

	c.issues[id] = newIssue
	return newIssue, nil
}

// UpdateIssue updates an existing mock issue
func (c *Client) UpdateIssue(ctx context.Context, id string, updates *tracker.IssueUpdate) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	issue, ok := c.issues[id]
	if !ok {
		return fmt.Errorf("issue not found: %s", id)
	}

	if updates.Title != nil {
		issue.Title = *updates.Title
	}
	if updates.Description != nil {
		issue.Description = *updates.Description
	}
	if updates.Status != nil {
		issue.Status = *updates.Status
	}
	if updates.Priority != nil {
		issue.Priority = *updates.Priority
	}
	if updates.Assignee != nil {
		issue.Assignee = *updates.Assignee
	}
	if updates.ParentID != nil {
		issue.ParentID = *updates.ParentID
	}

	issue.UpdatedAt = time.Now()
	return nil
}

// GetIssue retrieves a mock issue by ID
func (c *Client) GetIssue(ctx context.Context, id string) (*tracker.Issue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try by ID first
	if issue, ok := c.issues[id]; ok {
		return issue, nil
	}

	// Try by identifier
	for _, issue := range c.issues {
		if issue.Identifier == id {
			return issue, nil
		}
	}

	return nil, fmt.Errorf("issue not found: %s", id)
}

// SearchIssues searches mock issues
func (c *Client) SearchIssues(ctx context.Context, query string) ([]*tracker.Issue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []*tracker.Issue
	for _, issue := range c.issues {
		// Simple substring search
		if containsIgnoreCase(issue.Title, query) || containsIgnoreCase(issue.Description, query) {
			results = append(results, issue)
		}
	}

	return results, nil
}

// CreateSubIssue creates a sub-issue under a parent
func (c *Client) CreateSubIssue(ctx context.Context, parentID string, issue *tracker.Issue) (*tracker.Issue, error) {
	issue.ParentID = parentID
	return c.CreateIssue(ctx, issue)
}

// GetSubIssues retrieves all sub-issues for a parent
func (c *Client) GetSubIssues(ctx context.Context, parentID string) ([]*tracker.Issue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var children []*tracker.Issue
	for _, issue := range c.issues {
		if issue.ParentID == parentID {
			children = append(children, issue)
		}
	}

	return children, nil
}

// SetBlocking sets a blocking relationship
func (c *Client) SetBlocking(ctx context.Context, blockerID, blockedID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.relations[blockedID] = append(c.relations[blockedID], blockerID)
	return nil
}

// GetBlockedBy retrieves blocking issues
func (c *Client) GetBlockedBy(ctx context.Context, issueID string) ([]*tracker.Issue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	blockerIDs := c.relations[issueID]
	var blockers []*tracker.Issue

	for _, id := range blockerIDs {
		if issue, ok := c.issues[id]; ok {
			blockers = append(blockers, issue)
		}
	}

	return blockers, nil
}

// AddComment adds a comment to an issue
func (c *Client) AddComment(ctx context.Context, issueID string, body string) (*tracker.Comment, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.issues[issueID]; !ok {
		return nil, fmt.Errorf("issue not found: %s", issueID)
	}

	c.counter++
	now := time.Now()
	comment := &tracker.Comment{
		ID:        fmt.Sprintf("comment-%d", c.counter),
		Body:      body,
		Author:    "mock-user",
		CreatedAt: now,
		UpdatedAt: now,
	}

	c.comments[issueID] = append(c.comments[issueID], comment)
	return comment, nil
}

// GetComments retrieves all comments for an issue
func (c *Client) GetComments(ctx context.Context, issueID string) ([]*tracker.Comment, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.comments[issueID], nil
}

// TransitionIssue changes an issue's status
func (c *Client) TransitionIssue(ctx context.Context, id string, status tracker.Status) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	issue, ok := c.issues[id]
	if !ok {
		return fmt.Errorf("issue not found: %s", id)
	}

	issue.Status = status
	issue.UpdatedAt = time.Now()
	return nil
}

// GetAvailableStatuses returns all possible statuses
func (c *Client) GetAvailableStatuses(ctx context.Context, id string) ([]tracker.Status, error) {
	return []tracker.Status{
		tracker.StatusBacklog,
		tracker.StatusTodo,
		tracker.StatusInProgress,
		tracker.StatusInReview,
		tracker.StatusDone,
		tracker.StatusCanceled,
	}, nil
}

// GetTeams returns mock teams
func (c *Client) GetTeams(ctx context.Context) ([]tracker.Team, error) {
	return []tracker.Team{
		{ID: "team-1", Name: "Engineering", Key: "ENG"},
		{ID: "team-2", Name: "Design", Key: "DES"},
	}, nil
}

// GetProjects returns mock projects
func (c *Client) GetProjects(ctx context.Context, teamID string) ([]tracker.Project, error) {
	return []tracker.Project{
		{ID: "project-1", Name: "Backend", TeamID: teamID},
		{ID: "project-2", Name: "Frontend", TeamID: teamID},
	}, nil
}

// Helper functions

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsIgnoreCase(s[1:], substr)) ||
		(len(s) >= len(substr) && equalIgnoreCase(s[:len(substr)], substr)))
}

func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
