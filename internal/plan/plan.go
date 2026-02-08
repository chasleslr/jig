package plan

import (
	"fmt"
	"time"
)

// Status represents the overall status of a plan
type Status string

const (
	StatusDraft      Status = "draft"
	StatusReviewing  Status = "reviewing"
	StatusApproved   Status = "approved"
	StatusInProgress Status = "in-progress"
	StatusInReview   Status = "in-review"
	StatusComplete   Status = "complete"
)

// ReviewerType categorizes reviewers
type ReviewerType string

const (
	ReviewerLead          ReviewerType = "lead"
	ReviewerSecurity      ReviewerType = "security"
	ReviewerPerformance   ReviewerType = "performance"
	ReviewerAccessibility ReviewerType = "accessibility"
)

// Reviewers tracks the review status
type Reviewers struct {
	Default  []ReviewerType `yaml:"default,omitempty"`
	Optional []ReviewerType `yaml:"optional,omitempty"`
	OptedOut []ReviewerType `yaml:"opted_out,omitempty"`
}

// Plan represents a complete work plan
type Plan struct {
	// Frontmatter fields
	ID        string    `yaml:"id"`                      // Internal plan ID (e.g., PLAN-1234567890)
	IssueID   string    `yaml:"issue_id,omitempty"`      // Optional linked issue ID (e.g., NUM-41)
	Title     string    `yaml:"title"`
	Status    Status    `yaml:"status"`
	Created   time.Time `yaml:"created"`
	Updated   time.Time `yaml:"updated,omitempty"`
	Author    string    `yaml:"author"`
	Reviewers Reviewers `yaml:"reviewers"`

	// Parsed from markdown body
	ProblemStatement string                  `yaml:"-"`
	ProposedSolution string                  `yaml:"-"`
	QuestionsAnswers map[string]string       `yaml:"-"`
	ReviewNotes      map[ReviewerType]string `yaml:"-"`

	// Raw content
	RawContent string `yaml:"-"`
	FilePath   string `yaml:"-"`
}

// NewPlan creates a new plan with defaults
func NewPlan(id, title, author string) *Plan {
	now := time.Now()
	return &Plan{
		ID:      id,
		Title:   title,
		Status:  StatusDraft,
		Created: now,
		Updated: now,
		Author:  author,
		Reviewers: Reviewers{
			Default: []ReviewerType{ReviewerLead, ReviewerSecurity},
		},
		QuestionsAnswers: make(map[string]string),
		ReviewNotes:      make(map[ReviewerType]string),
	}
}

// IsReadyForReview returns true if the plan can be reviewed
func (p *Plan) IsReadyForReview() bool {
	return p.Status == StatusDraft
}

// CanBeImplemented returns true if the plan has been approved
func (p *Plan) CanBeImplemented() bool {
	return p.Status == StatusApproved || p.Status == StatusInProgress
}

// Validate checks if the plan is well-formed
func (p *Plan) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("plan ID is required")
	}
	if p.Title == "" {
		return fmt.Errorf("plan title is required")
	}
	if p.Author == "" {
		return fmt.Errorf("plan author is required")
	}
	return nil
}

// HasLinkedIssue returns true if the plan is linked to an external issue
func (p *Plan) HasLinkedIssue() bool {
	return p.IssueID != ""
}

// TransitionTo changes the plan status with validation
func (p *Plan) TransitionTo(status Status) error {
	validTransitions := map[Status][]Status{
		StatusDraft:      {StatusReviewing, StatusInProgress}, // Allow direct to in-progress for quick implementation
		StatusReviewing:  {StatusDraft, StatusApproved},
		StatusApproved:   {StatusInProgress, StatusDraft}, // Can go back to draft for amendments
		StatusInProgress: {StatusInReview, StatusComplete, StatusApproved},
		StatusInReview:   {StatusComplete, StatusInProgress}, // Can go back to in-progress for changes
		StatusComplete:   {},                                 // Terminal state
	}

	allowed := validTransitions[p.Status]
	for _, s := range allowed {
		if s == status {
			p.Status = status
			p.Updated = time.Now()
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", p.Status, status)
}
