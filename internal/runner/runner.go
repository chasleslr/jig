package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/charleslr/jig/internal/plan"
)

// PromptType identifies the type of prompt to use
type PromptType string

const (
	PromptTypePlan           PromptType = "plan"
	PromptTypeImplement      PromptType = "implement"
	PromptTypeReview         PromptType = "review"
	PromptTypeLeadReview     PromptType = "lead_review"
	PromptTypeSecurityReview PromptType = "security_review"
)

// PrepareOpts contains options for preparing the execution context
type PrepareOpts struct {
	Plan         *plan.Plan
	Phase        *plan.Phase
	WorktreeDir  string
	PromptType   PromptType
	PlanGoal     string            // User's description of what they want to plan
	IssueContext string            // Context from linked issue (Linear, etc.)
	ExtraVars    map[string]string
}

// LaunchOpts contains options for launching the external tool
type LaunchOpts struct {
	WorktreeDir   string
	Prompt        string // For non-interactive mode (runs and exits)
	InitialPrompt string // For interactive mode - sent as first message, then continues interactively
	SystemPrompt  string // System prompt/instructions (for Claude: --system-prompt)
	Interactive   bool
	PlanMode      bool // Launch in plan mode (for Claude: --permission-mode plan)
	Args          []string
}

// LaunchResult contains information about a completed session
type LaunchResult struct {
	// ExitCode is the exit code of the process (0 = success)
	ExitCode int
	// Duration is how long the session lasted
	Duration time.Duration
}

// Runner defines the interface for external coding tools
type Runner interface {
	// Name returns the runner identifier (e.g., "claude", "codex")
	Name() string

	// Prepare sets up the context for the tool (writes prompts, skills, etc.)
	Prepare(ctx context.Context, opts *PrepareOpts) error

	// Launch starts the tool as a subprocess with TTY passthrough.
	// It blocks until the tool exits and returns information about the session.
	Launch(ctx context.Context, opts *LaunchOpts) (*LaunchResult, error)

	// Available checks if the tool is installed and configured
	Available() bool
}

// Registry holds all available runners
type Registry struct {
	runners map[string]Runner
}

// NewRegistry creates a new runner registry
func NewRegistry() *Registry {
	return &Registry{
		runners: make(map[string]Runner),
	}
}

// Register adds a runner to the registry
func (r *Registry) Register(runner Runner) {
	r.runners[runner.Name()] = runner
}

// Get retrieves a runner by name
func (r *Registry) Get(name string) (Runner, error) {
	runner, ok := r.runners[name]
	if !ok {
		return nil, fmt.Errorf("unknown runner: %s", name)
	}
	return runner, nil
}

// Available returns all available runners
func (r *Registry) Available() []Runner {
	var available []Runner
	for _, runner := range r.runners {
		if runner.Available() {
			available = append(available, runner)
		}
	}
	return available
}

// List returns all registered runner names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.runners))
	for name := range r.runners {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry is the global runner registry
var DefaultRegistry = NewRegistry()

// Register adds a runner to the default registry
func Register(runner Runner) {
	DefaultRegistry.Register(runner)
}

// Get retrieves a runner from the default registry
func Get(name string) (Runner, error) {
	return DefaultRegistry.Get(name)
}
