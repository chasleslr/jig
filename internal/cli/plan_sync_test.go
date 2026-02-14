package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/state"
)

func TestCachedPlanNeedsSync(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name     string
		cached   *state.CachedPlan
		expected bool
	}{
		{
			name: "nil plan returns false",
			cached: &state.CachedPlan{
				Plan:      nil,
				UpdatedAt: now,
			},
			expected: false,
		},
		{
			name: "plan with no issue ID returns false",
			cached: &state.CachedPlan{
				Plan:      &plan.Plan{ID: "PLAN-1", IssueID: ""},
				UpdatedAt: now,
			},
			expected: false,
		},
		{
			name: "plan never synced returns true",
			cached: &state.CachedPlan{
				Plan:      &plan.Plan{ID: "PLAN-2", IssueID: "NUM-123"},
				UpdatedAt: now,
				SyncedAt:  nil,
			},
			expected: true,
		},
		{
			name: "plan updated after last sync returns true",
			cached: &state.CachedPlan{
				Plan:      &plan.Plan{ID: "PLAN-3", IssueID: "NUM-456"},
				UpdatedAt: future,
				SyncedAt:  &now,
			},
			expected: true,
		},
		{
			name: "plan synced after last update returns false",
			cached: &state.CachedPlan{
				Plan:      &plan.Plan{ID: "PLAN-4", IssueID: "NUM-789"},
				UpdatedAt: past,
				SyncedAt:  &now,
			},
			expected: false,
		},
		{
			name: "plan synced at same time as update returns false",
			cached: &state.CachedPlan{
				Plan:      &plan.Plan{ID: "PLAN-5", IssueID: "NUM-101"},
				UpdatedAt: now,
				SyncedAt:  &now,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cached.NeedsSync()
			if result != tt.expected {
				t.Errorf("NeedsSync() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPlanSyncCmd_Definition(t *testing.T) {
	// Verify the sync command is properly defined
	if planSyncCmd.Use != "sync [PLAN_ID]" {
		t.Errorf("expected Use 'sync [PLAN_ID]', got %q", planSyncCmd.Use)
	}

	if planSyncCmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if planSyncCmd.Long == "" {
		t.Error("expected Long description to be set")
	}

	if planSyncCmd.RunE == nil {
		t.Error("expected RunE to be set")
	}

	// Verify Long description mentions key features
	longDesc := planSyncCmd.Long
	if !strings.Contains(longDesc, "interactive") {
		t.Error("Long description should mention interactive mode")
	}
	if !strings.Contains(longDesc, "PLAN_ID") {
		t.Error("Long description should mention PLAN_ID argument")
	}
}

func TestPlanSyncCmdIsRegistered(t *testing.T) {
	// Verify the sync command is registered as a subcommand of plan
	found := false
	for _, cmd := range planCmd.Commands() {
		if cmd.Use == "sync [PLAN_ID]" {
			found = true
			break
		}
	}
	if !found {
		t.Error("planSyncCmd should be registered as a subcommand of planCmd")
	}
}

func TestSyncSinglePlanWithDeps(t *testing.T) {
	ctx := context.Background()
	mockHashFunc := func(p *plan.Plan) string { return "mock-hash-123" }

	t.Run("plan not found returns error", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) {
				return nil, nil // not found
			},
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan:               func(ctx context.Context, p *plan.Plan) error { return nil },
			computeContentHash:     mockHashFunc,
		}

		err := syncSinglePlanWithDeps(ctx, "nonexistent", deps)
		if err == nil {
			t.Error("expected error for nonexistent plan")
		}
		if !strings.Contains(err.Error(), "plan not found") {
			t.Errorf("expected 'plan not found' error, got: %v", err)
		}
	})

	t.Run("cache error is propagated", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) {
				return nil, fmt.Errorf("cache read error")
			},
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan:               func(ctx context.Context, p *plan.Plan) error { return nil },
			computeContentHash:     mockHashFunc,
		}

		err := syncSinglePlanWithDeps(ctx, "test-plan", deps)
		if err == nil {
			t.Error("expected error when cache fails")
		}
		if !strings.Contains(err.Error(), "failed to get plan") {
			t.Errorf("expected 'failed to get plan' error, got: %v", err)
		}
	})

	t.Run("nil plan inside cached returns error", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) {
				return &state.CachedPlan{Plan: nil}, nil
			},
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan:               func(ctx context.Context, p *plan.Plan) error { return nil },
			computeContentHash:     mockHashFunc,
		}

		err := syncSinglePlanWithDeps(ctx, "test-plan", deps)
		if err == nil {
			t.Error("expected error for nil plan")
		}
		if !strings.Contains(err.Error(), "no linked issue") {
			t.Errorf("expected 'no linked issue' error, got: %v", err)
		}
	})

	t.Run("plan with no linked issue returns error", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) {
				return &state.CachedPlan{
					Plan: &plan.Plan{ID: "test", IssueID: ""},
				}, nil
			},
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan:               func(ctx context.Context, p *plan.Plan) error { return nil },
			computeContentHash:     mockHashFunc,
		}

		err := syncSinglePlanWithDeps(ctx, "test-plan", deps)
		if err == nil {
			t.Error("expected error for plan with no linked issue")
		}
		if !strings.Contains(err.Error(), "no linked issue") {
			t.Errorf("expected 'no linked issue' error, got: %v", err)
		}
	})

	t.Run("sync error is propagated", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) {
				return &state.CachedPlan{
					Plan: &plan.Plan{ID: "test", IssueID: "NUM-123"},
				}, nil
			},
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				return fmt.Errorf("sync failed")
			},
			computeContentHash: mockHashFunc,
		}

		err := syncSinglePlanWithDeps(ctx, "test-plan", deps)
		if err == nil {
			t.Error("expected error when sync fails")
		}
		if !strings.Contains(err.Error(), "failed to sync plan") {
			t.Errorf("expected 'failed to sync plan' error, got: %v", err)
		}
	})

	t.Run("successful sync calls markPlanSyncedWithHash", func(t *testing.T) {
		markedSynced := false
		var savedHash string
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) {
				return &state.CachedPlan{
					Plan: &plan.Plan{ID: "test", IssueID: "NUM-123"},
				}, nil
			},
			markPlanSyncedWithHash: func(id, hash string) error {
				markedSynced = true
				savedHash = hash
				return nil
			},
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				return nil
			},
			computeContentHash: mockHashFunc,
		}

		err := syncSinglePlanWithDeps(ctx, "test-plan", deps)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !markedSynced {
			t.Error("expected markPlanSyncedWithHash to be called")
		}
		if savedHash != "mock-hash-123" {
			t.Errorf("expected hash 'mock-hash-123', got %q", savedHash)
		}
	})

	t.Run("markPlanSyncedWithHash error does not fail the sync", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) {
				return &state.CachedPlan{
					Plan: &plan.Plan{ID: "test", IssueID: "NUM-123"},
				}, nil
			},
			markPlanSyncedWithHash: func(id, hash string) error {
				return fmt.Errorf("mark synced failed")
			},
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				return nil
			},
			computeContentHash: mockHashFunc,
		}

		// Should succeed despite markPlanSyncedWithHash error (just logs warning)
		err := syncSinglePlanWithDeps(ctx, "test-plan", deps)
		if err != nil {
			t.Errorf("sync should succeed even if marking fails: %v", err)
		}
	})

	t.Run("skips sync when content hash matches", func(t *testing.T) {
		syncCalled := false
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) {
				return &state.CachedPlan{
					Plan:              &plan.Plan{ID: "test", IssueID: "NUM-123"},
					SyncedContentHash: "mock-hash-123", // Same as what mockHashFunc returns
				}, nil
			},
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				syncCalled = true
				return nil
			},
			computeContentHash: mockHashFunc,
		}

		err := syncSinglePlanWithDeps(ctx, "test-plan", deps)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if syncCalled {
			t.Error("sync should be skipped when content hash matches")
		}
	})

	t.Run("syncs when content hash differs", func(t *testing.T) {
		syncCalled := false
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) {
				return &state.CachedPlan{
					Plan:              &plan.Plan{ID: "test", IssueID: "NUM-123"},
					SyncedContentHash: "different-hash", // Different from mockHashFunc
				}, nil
			},
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				syncCalled = true
				return nil
			},
			computeContentHash: mockHashFunc,
		}

		err := syncSinglePlanWithDeps(ctx, "test-plan", deps)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !syncCalled {
			t.Error("sync should be called when content hash differs")
		}
	})
}

func TestSyncSelectedPlansWithDeps(t *testing.T) {
	ctx := context.Background()
	mockHashFunc := func(p *plan.Plan) string { return "mock-hash-" + p.ID }

	t.Run("empty plan IDs returns success", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan:          func(id string) (*state.CachedPlan, error) { return nil, nil },
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan:               func(ctx context.Context, p *plan.Plan) error { return nil },
			computeContentHash:     mockHashFunc,
		}

		err := syncSelectedPlansWithDeps(ctx, []string{}, map[string]*state.CachedPlan{}, deps)
		if err != nil {
			t.Errorf("expected no error for empty plans, got: %v", err)
		}
	})

	t.Run("plan not found in map returns error", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan:          func(id string) (*state.CachedPlan, error) { return nil, nil },
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan:               func(ctx context.Context, p *plan.Plan) error { return nil },
			computeContentHash:     mockHashFunc,
		}

		// Plan ID not in the map
		err := syncSelectedPlansWithDeps(ctx, []string{"missing-plan"}, map[string]*state.CachedPlan{}, deps)
		if err == nil {
			t.Error("expected error when plan not found in map")
		}
		if !strings.Contains(err.Error(), "failed to sync 1 plan") {
			t.Errorf("expected failure count error, got: %v", err)
		}
	})

	t.Run("sync error for one plan reports failure", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan:          func(id string) (*state.CachedPlan, error) { return nil, nil },
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				if p.ID == "fail-plan" {
					return fmt.Errorf("sync failed for this plan")
				}
				return nil
			},
			computeContentHash: mockHashFunc,
		}

		idToPlan := map[string]*state.CachedPlan{
			"success-plan": {Plan: &plan.Plan{ID: "success-plan", IssueID: "NUM-1"}},
			"fail-plan":    {Plan: &plan.Plan{ID: "fail-plan", IssueID: "NUM-2"}},
		}

		err := syncSelectedPlansWithDeps(ctx, []string{"success-plan", "fail-plan"}, idToPlan, deps)
		if err == nil {
			t.Error("expected error when some plans fail")
		}
		if !strings.Contains(err.Error(), "failed to sync 1 plan") {
			t.Errorf("expected failure count error, got: %v", err)
		}
	})

	t.Run("all plans sync successfully", func(t *testing.T) {
		syncedPlans := make(map[string]bool)
		markedPlans := make(map[string]string)

		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) { return nil, nil },
			markPlanSyncedWithHash: func(id, hash string) error {
				markedPlans[id] = hash
				return nil
			},
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				syncedPlans[p.ID] = true
				return nil
			},
			computeContentHash: mockHashFunc,
		}

		idToPlan := map[string]*state.CachedPlan{
			"plan-1": {Plan: &plan.Plan{ID: "plan-1", IssueID: "NUM-1"}},
			"plan-2": {Plan: &plan.Plan{ID: "plan-2", IssueID: "NUM-2"}},
		}

		err := syncSelectedPlansWithDeps(ctx, []string{"plan-1", "plan-2"}, idToPlan, deps)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify all plans were synced
		if !syncedPlans["plan-1"] || !syncedPlans["plan-2"] {
			t.Error("expected all plans to be synced")
		}

		// Verify all plans were marked as synced with hashes
		if markedPlans["plan-1"] != "mock-hash-plan-1" || markedPlans["plan-2"] != "mock-hash-plan-2" {
			t.Error("expected all plans to be marked as synced with correct hashes")
		}
	})

	t.Run("markPlanSyncedWithHash error does not fail the batch", func(t *testing.T) {
		syncCount := 0
		deps := planSyncDeps{
			getCachedPlan: func(id string) (*state.CachedPlan, error) { return nil, nil },
			markPlanSyncedWithHash: func(id, hash string) error {
				return fmt.Errorf("mark failed")
			},
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				syncCount++
				return nil
			},
			computeContentHash: mockHashFunc,
		}

		idToPlan := map[string]*state.CachedPlan{
			"plan-1": {Plan: &plan.Plan{ID: "plan-1", IssueID: "NUM-1"}},
		}

		err := syncSelectedPlansWithDeps(ctx, []string{"plan-1"}, idToPlan, deps)
		if err != nil {
			t.Errorf("batch should succeed even if marking fails: %v", err)
		}
		if syncCount != 1 {
			t.Errorf("expected 1 sync, got %d", syncCount)
		}
	})

	t.Run("mixed success and failure counts correctly", func(t *testing.T) {
		deps := planSyncDeps{
			getCachedPlan:          func(id string) (*state.CachedPlan, error) { return nil, nil },
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				if strings.HasPrefix(p.ID, "fail") {
					return fmt.Errorf("sync error")
				}
				return nil
			},
			computeContentHash: mockHashFunc,
		}

		idToPlan := map[string]*state.CachedPlan{
			"success-1": {Plan: &plan.Plan{ID: "success-1", IssueID: "NUM-1"}},
			"fail-1":    {Plan: &plan.Plan{ID: "fail-1", IssueID: "NUM-2"}},
			"success-2": {Plan: &plan.Plan{ID: "success-2", IssueID: "NUM-3"}},
			"fail-2":    {Plan: &plan.Plan{ID: "fail-2", IssueID: "NUM-4"}},
		}

		err := syncSelectedPlansWithDeps(ctx, []string{"success-1", "fail-1", "success-2", "fail-2"}, idToPlan, deps)
		if err == nil {
			t.Error("expected error when some plans fail")
		}
		if !strings.Contains(err.Error(), "failed to sync 2 plan") {
			t.Errorf("expected 2 failures, got: %v", err)
		}
	})

	t.Run("skips plans with unchanged content hash", func(t *testing.T) {
		syncedPlans := make(map[string]bool)

		deps := planSyncDeps{
			getCachedPlan:          func(id string) (*state.CachedPlan, error) { return nil, nil },
			markPlanSyncedWithHash: func(id, hash string) error { return nil },
			syncPlan: func(ctx context.Context, p *plan.Plan) error {
				syncedPlans[p.ID] = true
				return nil
			},
			computeContentHash: mockHashFunc,
		}

		idToPlan := map[string]*state.CachedPlan{
			"unchanged-plan": {
				Plan:              &plan.Plan{ID: "unchanged-plan", IssueID: "NUM-1"},
				SyncedContentHash: "mock-hash-unchanged-plan", // Same as mockHashFunc would return
			},
			"changed-plan": {
				Plan:              &plan.Plan{ID: "changed-plan", IssueID: "NUM-2"},
				SyncedContentHash: "old-hash", // Different from mockHashFunc
			},
		}

		err := syncSelectedPlansWithDeps(ctx, []string{"unchanged-plan", "changed-plan"}, idToPlan, deps)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// unchanged-plan should be skipped
		if syncedPlans["unchanged-plan"] {
			t.Error("unchanged-plan should be skipped")
		}

		// changed-plan should be synced
		if !syncedPlans["changed-plan"] {
			t.Error("changed-plan should be synced")
		}
	})
}
