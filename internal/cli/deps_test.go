package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/charleslr/jig/internal/runner"
)

// mockRunner implements runner.Runner for testing
type mockRunner struct {
	name      string
	available bool
}

func (m *mockRunner) Name() string { return m.name }
func (m *mockRunner) Available() bool { return m.available }
func (m *mockRunner) Prepare(ctx context.Context, opts *runner.PrepareOpts) error { return nil }
func (m *mockRunner) Launch(ctx context.Context, opts *runner.LaunchOpts) (*runner.LaunchResult, error) {
	return &runner.LaunchResult{}, nil
}

// mockRegistry implements RunnerRegistry for testing
type mockRegistry struct {
	runners map[string]runner.Runner
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		runners: make(map[string]runner.Runner),
	}
}

func (m *mockRegistry) Get(name string) (runner.Runner, error) {
	r, ok := m.runners[name]
	if !ok {
		return nil, fmt.Errorf("unknown runner: %s", name)
	}
	return r, nil
}

func (m *mockRegistry) Register(r runner.Runner) {
	m.runners[r.Name()] = r
}

func TestDeps_DefaultDeps(t *testing.T) {
	d := DefaultDeps()
	if d == nil {
		t.Fatal("DefaultDeps() returned nil")
	}
	if d.RunnerRegistry == nil {
		t.Error("DefaultDeps().RunnerRegistry is nil")
	}
}

func TestDeps_SetAndReset(t *testing.T) {
	// Save original
	original := deps

	// Create mock deps
	mockReg := newMockRegistry()
	mockDeps := &Deps{RunnerRegistry: mockReg}

	// Set mock deps
	SetDeps(mockDeps)
	if deps != mockDeps {
		t.Error("SetDeps() did not set deps")
	}

	// Reset
	ResetDeps()
	if deps == mockDeps {
		t.Error("ResetDeps() did not reset deps")
	}

	// Restore original for other tests
	deps = original
}

func TestRunnerRegistry_ErrorPath(t *testing.T) {
	// Save original deps
	original := deps
	defer func() { deps = original }()

	// Create mock registry with no runners
	mockReg := newMockRegistry()
	SetDeps(&Deps{RunnerRegistry: mockReg})

	// Try to get a non-existent runner
	_, err := deps.RunnerRegistry.Get("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent runner")
	}
}

func TestRunnerRegistry_SuccessPath(t *testing.T) {
	// Save original deps
	original := deps
	defer func() { deps = original }()

	// Create mock registry with a runner
	mockReg := newMockRegistry()
	mockReg.Register(&mockRunner{name: "test-runner", available: true})
	SetDeps(&Deps{RunnerRegistry: mockReg})

	// Get the runner
	r, err := deps.RunnerRegistry.Get("test-runner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name() != "test-runner" {
		t.Errorf("expected name 'test-runner', got %q", r.Name())
	}
	if !r.Available() {
		t.Error("expected runner to be available")
	}
}
