package plan

// PhaseStatus represents the status of a plan phase
type PhaseStatus string

const (
	PhaseStatusPending    PhaseStatus = "pending"
	PhaseStatusInProgress PhaseStatus = "in-progress"
	PhaseStatusBlocked    PhaseStatus = "blocked"
	PhaseStatusComplete   PhaseStatus = "complete"
)

// Phase represents a single phase within a plan
type Phase struct {
	ID          string      `yaml:"id"`
	Title       string      `yaml:"title"`
	IssueID     string      `yaml:"issue_id,omitempty"`
	Status      PhaseStatus `yaml:"status"`
	DependsOn   []string    `yaml:"depends_on,omitempty"`
	Branch      string      `yaml:"-"` // Computed from issue ID
	Description string      `yaml:"-"` // Parsed from markdown body
	Acceptance  []string    `yaml:"-"` // Acceptance criteria from markdown
}

// IsBlocked returns true if this phase has unmet dependencies
func (p *Phase) IsBlocked(phases []*Phase) bool {
	if len(p.DependsOn) == 0 {
		return false
	}

	phaseMap := make(map[string]*Phase)
	for _, ph := range phases {
		phaseMap[ph.ID] = ph
	}

	for _, depID := range p.DependsOn {
		dep, ok := phaseMap[depID]
		if !ok {
			// Unknown dependency, treat as blocked
			return true
		}
		if dep.Status != PhaseStatusComplete {
			return true
		}
	}

	return false
}

// CanStart returns true if this phase can be started
func (p *Phase) CanStart(phases []*Phase) bool {
	if p.Status != PhaseStatusPending {
		return false
	}
	return !p.IsBlocked(phases)
}

// GetDependencies returns the Phase objects this phase depends on
func (p *Phase) GetDependencies(phases []*Phase) []*Phase {
	if len(p.DependsOn) == 0 {
		return nil
	}

	phaseMap := make(map[string]*Phase)
	for _, ph := range phases {
		phaseMap[ph.ID] = ph
	}

	var deps []*Phase
	for _, depID := range p.DependsOn {
		if dep, ok := phaseMap[depID]; ok {
			deps = append(deps, dep)
		}
	}

	return deps
}

// TopologicalSort returns phases in dependency order
// Independent phases are grouped together
func TopologicalSort(phases []*Phase) [][]*Phase {
	if len(phases) == 0 {
		return nil
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)
	phaseMap := make(map[string]*Phase)

	for _, p := range phases {
		phaseMap[p.ID] = p
		inDegree[p.ID] = len(p.DependsOn)
		for _, depID := range p.DependsOn {
			dependents[depID] = append(dependents[depID], p.ID)
		}
	}

	// Find phases with no dependencies (in-degree 0)
	var result [][]*Phase
	var queue []string

	for _, p := range phases {
		if inDegree[p.ID] == 0 {
			queue = append(queue, p.ID)
		}
	}

	for len(queue) > 0 {
		// All phases in current queue can run in parallel
		var level []*Phase
		var nextQueue []string

		for _, id := range queue {
			level = append(level, phaseMap[id])
			for _, depID := range dependents[id] {
				inDegree[depID]--
				if inDegree[depID] == 0 {
					nextQueue = append(nextQueue, depID)
				}
			}
		}

		if len(level) > 0 {
			result = append(result, level)
		}
		queue = nextQueue
	}

	return result
}
