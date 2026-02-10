package cli

import "github.com/charleslr/jig/internal/runner"

// Deps holds dependencies for CLI commands.
// This enables dependency injection for testing.
type Deps struct {
	// RunnerRegistry provides access to code runners (claude, etc.)
	RunnerRegistry RunnerRegistry
}

// RunnerRegistry is the interface for getting runners.
// This allows mocking in tests.
type RunnerRegistry interface {
	Get(name string) (runner.Runner, error)
}

// DefaultDeps returns the default production dependencies.
func DefaultDeps() *Deps {
	return &Deps{
		RunnerRegistry: runner.DefaultRegistry,
	}
}

// deps is the package-level dependencies used by commands.
// It can be replaced in tests via SetDeps.
var deps = DefaultDeps()

// SetDeps replaces the package-level dependencies.
// This is intended for testing only.
func SetDeps(d *Deps) {
	deps = d
}

// ResetDeps restores the default dependencies.
// This should be called in test cleanup.
func ResetDeps() {
	deps = DefaultDeps()
}
