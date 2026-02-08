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

// PlanSyncer defines the interface for syncing full plan content to a tracker
type PlanSyncer interface {
	SyncPlan(ctx context.Context, p *plan.Plan) error
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

// SyncResult contains the result of syncing a plan to the tracker
type SyncResult struct {
	IssueID      string // The issue ID (may be new if created)
	IssueURL     string // URL to the issue in the tracker
	Created      bool   // True if a new issue was created
	Updated      bool   // True if an existing issue was updated
	TrackerError error  // Any error that occurred during sync
}

// SyncPlanToTracker syncs the full plan content to a tracker.
// This creates or updates the issue with the plan's problem statement and solution.
// The plan's ID is updated if a new issue is created.
func SyncPlanToTracker(ctx context.Context, p *plan.Plan, syncer PlanSyncer) (*SyncResult, error) {
	if p == nil {
		return nil, fmt.Errorf("plan is nil")
	}
	if syncer == nil {
		return nil, fmt.Errorf("syncer is nil")
	}

	result := &SyncResult{}

	// Check if this is a new issue or an update
	existingID := p.ID

	// Sync the plan to the tracker
	if err := syncer.SyncPlan(ctx, p); err != nil {
		result.TrackerError = err
		return result, err
	}

	// Determine if it was created or updated
	if existingID == "" || existingID != p.ID {
		result.Created = true
	} else {
		result.Updated = true
	}
	result.IssueID = p.ID

	return result, nil
}
