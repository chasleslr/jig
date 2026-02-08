package state

import (
	"context"
	"fmt"

	"github.com/charleslr/jig/internal/plan"
)

// TrackerSyncer defines the interface for syncing plan status to a tracker
type TrackerSyncer interface {
	SyncPlanStatus(ctx context.Context, p *plan.Plan) error
}

// PlanStatusManager handles plan status transitions with automatic
// synchronization to both local cache and remote tracker.
// This ensures consistency between local and remote state.
type PlanStatusManager struct {
	cache  *Cache
	syncer TrackerSyncer
}

// NewPlanStatusManager creates a new PlanStatusManager.
// The syncer parameter is optional - if nil, only local cache is updated.
func NewPlanStatusManager(cache *Cache, syncer TrackerSyncer) *PlanStatusManager {
	return &PlanStatusManager{
		cache:  cache,
		syncer: syncer,
	}
}

// TransitionResult contains the result of a status transition
type TransitionResult struct {
	PreviousStatus plan.Status
	NewStatus      plan.Status
	CacheSaved     bool
	TrackerSynced  bool
	TrackerError   error
}

// TransitionTo transitions a plan to a new status, updating both
// the local cache and remote tracker. Cache update happens first,
// then tracker sync (which can fail without rolling back the cache).
//
// The plan object is modified in place with the new status.
func (m *PlanStatusManager) TransitionTo(ctx context.Context, p *plan.Plan, status plan.Status) (*TransitionResult, error) {
	if p == nil {
		return nil, fmt.Errorf("plan is nil")
	}

	result := &TransitionResult{
		PreviousStatus: p.Status,
		NewStatus:      status,
	}

	// Transition the plan status (validates the transition)
	if err := p.TransitionTo(status); err != nil {
		return result, err
	}

	// Save to local cache
	if m.cache != nil {
		if err := m.cache.SavePlan(p); err != nil {
			// Roll back status on cache failure
			p.Status = result.PreviousStatus
			return result, fmt.Errorf("failed to save plan to cache: %w", err)
		}
		result.CacheSaved = true
	}

	// Sync to tracker (non-blocking - failures are recorded but don't fail the operation)
	if m.syncer != nil {
		if err := m.syncer.SyncPlanStatus(ctx, p); err != nil {
			result.TrackerError = err
		} else {
			result.TrackerSynced = true
		}
	}

	return result, nil
}

// Complete transitions a plan to the complete status.
// This is a convenience method for the common operation of marking a plan as done.
func (m *PlanStatusManager) Complete(ctx context.Context, p *plan.Plan) (*TransitionResult, error) {
	return m.TransitionTo(ctx, p, plan.StatusComplete)
}

// StartProgress transitions a plan to the in-progress status.
// This is a convenience method for the common operation of starting implementation.
func (m *PlanStatusManager) StartProgress(ctx context.Context, p *plan.Plan) (*TransitionResult, error) {
	return m.TransitionTo(ctx, p, plan.StatusInProgress)
}
