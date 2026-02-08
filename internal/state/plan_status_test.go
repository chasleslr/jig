package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/charleslr/jig/internal/plan"
)

// mockSyncer is a mock TrackerSyncer for testing
type mockSyncer struct {
	syncCalled bool
	syncErr    error
	lastPlan   *plan.Plan
}

func (m *mockSyncer) SyncPlanStatus(ctx context.Context, p *plan.Plan) error {
	m.syncCalled = true
	m.lastPlan = p
	return m.syncErr
}

func createTestCache(t *testing.T) (*Cache, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "jig-plan-status-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}
	cleanup := func() { os.RemoveAll(tmpDir) }
	return cache, cleanup
}

func createTestPlan(id string, status plan.Status) *plan.Plan {
	return &plan.Plan{
		ID:         id,
		Title:      "Test Plan",
		Status:     status,
		Author:     "testuser",
		RawContent: "---\nid: " + id + "\ntitle: Test Plan\nstatus: " + string(status) + "\nauthor: testuser\n---\n\n# Test\n",
	}
}

func TestPlanStatusManager_TransitionTo_Success(t *testing.T) {
	cache, cleanup := createTestCache(t)
	defer cleanup()

	syncer := &mockSyncer{}
	mgr := NewPlanStatusManager(cache, syncer)

	p := createTestPlan("test-1", plan.StatusInProgress)

	result, err := mgr.TransitionTo(context.Background(), p, plan.StatusComplete)
	if err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}

	// Check result
	if result.PreviousStatus != plan.StatusInProgress {
		t.Errorf("PreviousStatus = %v, want %v", result.PreviousStatus, plan.StatusInProgress)
	}
	if result.NewStatus != plan.StatusComplete {
		t.Errorf("NewStatus = %v, want %v", result.NewStatus, plan.StatusComplete)
	}
	if !result.CacheSaved {
		t.Error("CacheSaved should be true")
	}
	if !result.TrackerSynced {
		t.Error("TrackerSynced should be true")
	}
	if result.TrackerError != nil {
		t.Errorf("TrackerError should be nil, got %v", result.TrackerError)
	}

	// Check plan was updated
	if p.Status != plan.StatusComplete {
		t.Errorf("plan.Status = %v, want %v", p.Status, plan.StatusComplete)
	}

	// Check syncer was called
	if !syncer.syncCalled {
		t.Error("syncer.SyncPlanStatus should have been called")
	}
	if syncer.lastPlan != p {
		t.Error("syncer should have received the plan")
	}

	// Check plan was saved to cache
	cached, err := cache.GetPlan("test-1")
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if cached == nil {
		t.Fatal("cached plan should not be nil")
	}
	if cached.Status != plan.StatusComplete {
		t.Errorf("cached.Status = %v, want %v", cached.Status, plan.StatusComplete)
	}
}

func TestPlanStatusManager_TransitionTo_InvalidTransition(t *testing.T) {
	cache, cleanup := createTestCache(t)
	defer cleanup()

	mgr := NewPlanStatusManager(cache, nil)

	// Complete status cannot transition to anything
	p := createTestPlan("test-2", plan.StatusComplete)

	result, err := mgr.TransitionTo(context.Background(), p, plan.StatusInProgress)
	if err == nil {
		t.Error("TransitionTo() should return error for invalid transition")
	}

	// Status should not change on error
	if p.Status != plan.StatusComplete {
		t.Errorf("plan.Status should remain %v, got %v", plan.StatusComplete, p.Status)
	}

	// Result should have correct previous status
	if result.PreviousStatus != plan.StatusComplete {
		t.Errorf("PreviousStatus = %v, want %v", result.PreviousStatus, plan.StatusComplete)
	}
}

func TestPlanStatusManager_TransitionTo_NilPlan(t *testing.T) {
	mgr := NewPlanStatusManager(nil, nil)

	_, err := mgr.TransitionTo(context.Background(), nil, plan.StatusComplete)
	if err == nil {
		t.Error("TransitionTo() should return error for nil plan")
	}
}

func TestPlanStatusManager_TransitionTo_TrackerError(t *testing.T) {
	cache, cleanup := createTestCache(t)
	defer cleanup()

	trackerErr := errors.New("tracker connection failed")
	syncer := &mockSyncer{syncErr: trackerErr}
	mgr := NewPlanStatusManager(cache, syncer)

	p := createTestPlan("test-3", plan.StatusInProgress)

	result, err := mgr.TransitionTo(context.Background(), p, plan.StatusComplete)
	// Should not return error for tracker failure (non-blocking)
	if err != nil {
		t.Fatalf("TransitionTo() error = %v (should succeed despite tracker error)", err)
	}

	// Check result
	if !result.CacheSaved {
		t.Error("CacheSaved should be true")
	}
	if result.TrackerSynced {
		t.Error("TrackerSynced should be false when syncer errors")
	}
	if result.TrackerError != trackerErr {
		t.Errorf("TrackerError = %v, want %v", result.TrackerError, trackerErr)
	}

	// Plan should still be updated
	if p.Status != plan.StatusComplete {
		t.Errorf("plan.Status = %v, want %v", p.Status, plan.StatusComplete)
	}
}

func TestPlanStatusManager_TransitionTo_NilSyncer(t *testing.T) {
	cache, cleanup := createTestCache(t)
	defer cleanup()

	mgr := NewPlanStatusManager(cache, nil)

	p := createTestPlan("test-4", plan.StatusInProgress)

	result, err := mgr.TransitionTo(context.Background(), p, plan.StatusComplete)
	if err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}

	// Should save to cache but not sync to tracker
	if !result.CacheSaved {
		t.Error("CacheSaved should be true")
	}
	if result.TrackerSynced {
		t.Error("TrackerSynced should be false when syncer is nil")
	}
	if result.TrackerError != nil {
		t.Errorf("TrackerError should be nil, got %v", result.TrackerError)
	}
}

func TestPlanStatusManager_TransitionTo_NilCache(t *testing.T) {
	syncer := &mockSyncer{}
	mgr := NewPlanStatusManager(nil, syncer)

	p := createTestPlan("test-5", plan.StatusInProgress)

	result, err := mgr.TransitionTo(context.Background(), p, plan.StatusComplete)
	if err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}

	// Should sync to tracker but not save to cache
	if result.CacheSaved {
		t.Error("CacheSaved should be false when cache is nil")
	}
	if !result.TrackerSynced {
		t.Error("TrackerSynced should be true")
	}
}

func TestPlanStatusManager_Complete(t *testing.T) {
	cache, cleanup := createTestCache(t)
	defer cleanup()

	syncer := &mockSyncer{}
	mgr := NewPlanStatusManager(cache, syncer)

	p := createTestPlan("test-6", plan.StatusInProgress)

	result, err := mgr.Complete(context.Background(), p)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if result.NewStatus != plan.StatusComplete {
		t.Errorf("NewStatus = %v, want %v", result.NewStatus, plan.StatusComplete)
	}
	if p.Status != plan.StatusComplete {
		t.Errorf("plan.Status = %v, want %v", p.Status, plan.StatusComplete)
	}
}

func TestPlanStatusManager_StartProgress(t *testing.T) {
	cache, cleanup := createTestCache(t)
	defer cleanup()

	syncer := &mockSyncer{}
	mgr := NewPlanStatusManager(cache, syncer)

	p := createTestPlan("test-7", plan.StatusDraft)

	result, err := mgr.StartProgress(context.Background(), p)
	if err != nil {
		t.Fatalf("StartProgress() error = %v", err)
	}

	if result.NewStatus != plan.StatusInProgress {
		t.Errorf("NewStatus = %v, want %v", result.NewStatus, plan.StatusInProgress)
	}
	if p.Status != plan.StatusInProgress {
		t.Errorf("plan.Status = %v, want %v", p.Status, plan.StatusInProgress)
	}
}

func TestPlanStatusManager_InProgressToComplete(t *testing.T) {
	// This tests the transition that was previously not allowed (the bug)
	cache, cleanup := createTestCache(t)
	defer cleanup()

	mgr := NewPlanStatusManager(cache, nil)

	p := createTestPlan("test-8", plan.StatusInProgress)

	result, err := mgr.Complete(context.Background(), p)
	if err != nil {
		t.Fatalf("Complete() from InProgress should succeed, got error = %v", err)
	}

	if result.NewStatus != plan.StatusComplete {
		t.Errorf("NewStatus = %v, want %v", result.NewStatus, plan.StatusComplete)
	}
	if p.Status != plan.StatusComplete {
		t.Errorf("plan.Status = %v, want %v", p.Status, plan.StatusComplete)
	}
}
