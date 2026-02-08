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
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title"`
	Status    Status    `yaml:"status"`
	Created   time.Time `yaml:"created"`
	Updated   time.Time `yaml:"updated,omitempty"`
	Author    string    `yaml:"author"`
	Reviewers Reviewers `yaml:"reviewers"`
	Phases    []*Phase  `yaml:"phases"`

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
		Phases:           make([]*Phase, 0),
		QuestionsAnswers: make(map[string]string),
		ReviewNotes:      make(map[ReviewerType]string),
	}
}

// GetPhase returns a phase by ID
func (p *Plan) GetPhase(id string) *Phase {
	for _, phase := range p.Phases {
		if phase.ID == id {
			return phase
		}
	}
	return nil
}

// GetPhaseByIssueID returns a phase by its tracker issue ID
func (p *Plan) GetPhaseByIssueID(issueID string) *Phase {
	for _, phase := range p.Phases {
		if phase.IssueID == issueID {
			return phase
		}
	}
	return nil
}

// AddPhase adds a new phase to the plan
func (p *Plan) AddPhase(phase *Phase) {
	p.Phases = append(p.Phases, phase)
	p.Updated = time.Now()
}

// GetNextPhases returns phases that can be started
func (p *Plan) GetNextPhases() []*Phase {
	var ready []*Phase
	for _, phase := range p.Phases {
		if phase.CanStart(p.Phases) {
			ready = append(ready, phase)
		}
	}
	return ready
}

// GetBlockedPhases returns phases that are blocked by dependencies
func (p *Plan) GetBlockedPhases() []*Phase {
	var blocked []*Phase
	for _, phase := range p.Phases {
		if phase.Status == PhaseStatusPending && phase.IsBlocked(p.Phases) {
			blocked = append(blocked, phase)
		}
	}
	return blocked
}

// GetInProgressPhases returns phases that are currently in progress
func (p *Plan) GetInProgressPhases() []*Phase {
	var inProgress []*Phase
	for _, phase := range p.Phases {
		if phase.Status == PhaseStatusInProgress {
			inProgress = append(inProgress, phase)
		}
	}
	return inProgress
}

// GetCompletedPhases returns phases that are complete
func (p *Plan) GetCompletedPhases() []*Phase {
	var completed []*Phase
	for _, phase := range p.Phases {
		if phase.Status == PhaseStatusComplete {
			completed = append(completed, phase)
		}
	}
	return completed
}

// Progress returns the completion percentage
func (p *Plan) Progress() float64 {
	if len(p.Phases) == 0 {
		return 0
	}
	completed := len(p.GetCompletedPhases())
	return float64(completed) / float64(len(p.Phases)) * 100
}

// IsReadyForReview returns true if the plan can be reviewed
func (p *Plan) IsReadyForReview() bool {
	return p.Status == StatusDraft && len(p.Phases) > 0
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

	// Validate phases
	phaseIDs := make(map[string]bool)
	for _, phase := range p.Phases {
		if phase.ID == "" {
			return fmt.Errorf("phase ID is required")
		}
		if phaseIDs[phase.ID] {
			return fmt.Errorf("duplicate phase ID: %s", phase.ID)
		}
		phaseIDs[phase.ID] = true

		// Validate dependencies exist
		for _, depID := range phase.DependsOn {
			if !phaseIDs[depID] {
				// Dependency might be defined later, check all phases
				found := false
				for _, ph := range p.Phases {
					if ph.ID == depID {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("phase %s has unknown dependency: %s", phase.ID, depID)
				}
			}
		}
	}

	// Check for circular dependencies
	if hasCycle(p.Phases) {
		return fmt.Errorf("plan has circular dependencies")
	}

	return nil
}

// hasCycle checks for circular dependencies using DFS
func hasCycle(phases []*Phase) bool {
	phaseMap := make(map[string]*Phase)
	for _, p := range phases {
		phaseMap[p.ID] = p
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(id string) bool
	dfs = func(id string) bool {
		visited[id] = true
		recStack[id] = true

		phase := phaseMap[id]
		if phase != nil {
			for _, depID := range phase.DependsOn {
				if !visited[depID] {
					if dfs(depID) {
						return true
					}
				} else if recStack[depID] {
					return true
				}
			}
		}

		recStack[id] = false
		return false
	}

	for _, p := range phases {
		if !visited[p.ID] {
			if dfs(p.ID) {
				return true
			}
		}
	}

	return false
}

// TransitionTo changes the plan status with validation
func (p *Plan) TransitionTo(status Status) error {
	validTransitions := map[Status][]Status{
		StatusDraft:      {StatusReviewing, StatusInProgress}, // Allow direct to in-progress for quick implementation
		StatusReviewing:  {StatusDraft, StatusApproved},
		StatusApproved:   {StatusInProgress, StatusDraft}, // Can go back to draft for amendments
		StatusInProgress: {StatusInReview, StatusApproved},
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

// SetPhaseStatus updates a phase status
func (p *Plan) SetPhaseStatus(phaseID string, status PhaseStatus) error {
	phase := p.GetPhase(phaseID)
	if phase == nil {
		return fmt.Errorf("phase not found: %s", phaseID)
	}

	phase.Status = status
	p.Updated = time.Now()

	// Update plan status based on phase progress
	if status == PhaseStatusInProgress && (p.Status == StatusDraft || p.Status == StatusApproved) {
		p.Status = StatusInProgress
	}

	// Check if all phases are complete
	allComplete := true
	for _, ph := range p.Phases {
		if ph.Status != PhaseStatusComplete {
			allComplete = false
			break
		}
	}
	if allComplete && len(p.Phases) > 0 {
		p.Status = StatusInReview
	}

	return nil
}
