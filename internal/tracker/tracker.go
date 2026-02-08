package tracker

import (
	"context"
	"time"
)

// Status represents the status of an issue in the tracker
type Status string

const (
	StatusBacklog    Status = "backlog"
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusInReview   Status = "in_review"
	StatusDone       Status = "done"
	StatusCanceled   Status = "canceled"
)

// Priority represents issue priority
type Priority int

const (
	PriorityNone   Priority = 0
	PriorityUrgent Priority = 1
	PriorityHigh   Priority = 2
	PriorityMedium Priority = 3
	PriorityLow    Priority = 4
)

// Issue represents an issue in the tracker system
type Issue struct {
	ID          string
	Identifier  string // Human-readable ID (e.g., "ENG-123")
	Title       string
	Description string
	Status      Status
	Priority    Priority
	Assignee    string
	Labels      []string
	ParentID    string // For sub-issues
	ProjectID   string
	TeamID      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	URL         string // Web URL to the issue
}

// IssueUpdate contains fields to update on an issue
type IssueUpdate struct {
	Title       *string
	Description *string
	Status      *Status
	Priority    *Priority
	Assignee    *string
	Labels      []string
	ParentID    *string
}

// Comment represents a comment on an issue
type Comment struct {
	ID        string
	Body      string
	Author    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Tracker defines the interface for issue tracking systems
type Tracker interface {
	// Issue management
	CreateIssue(ctx context.Context, issue *Issue) (*Issue, error)
	UpdateIssue(ctx context.Context, id string, updates *IssueUpdate) error
	GetIssue(ctx context.Context, id string) (*Issue, error)
	SearchIssues(ctx context.Context, query string) ([]*Issue, error)

	// Sub-issues
	CreateSubIssue(ctx context.Context, parentID string, issue *Issue) (*Issue, error)
	GetSubIssues(ctx context.Context, parentID string) ([]*Issue, error)

	// Comments for Q&A and updates
	AddComment(ctx context.Context, issueID string, body string) (*Comment, error)
	GetComments(ctx context.Context, issueID string) ([]*Comment, error)

	// Status management
	TransitionIssue(ctx context.Context, id string, status Status) error
	GetAvailableStatuses(ctx context.Context, id string) ([]Status, error)

	// Team and project info
	GetTeams(ctx context.Context) ([]Team, error)
	GetProjects(ctx context.Context, teamID string) ([]Project, error)
}

// Team represents a team in the tracker
type Team struct {
	ID   string
	Name string
	Key  string // e.g., "ENG"
}

// Project represents a project in the tracker
type Project struct {
	ID     string
	Name   string
	TeamID string
}
