package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/tracker"
)

// mockPlanFetcherTracker implements both tracker.Tracker and tracker.PlanFetcher for testing
type mockPlanFetcherTracker struct {
	plan *plan.Plan
	err  error
}

func (m *mockPlanFetcherTracker) FetchPlanFromIssue(ctx context.Context, issueID string) (*plan.Plan, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.plan, nil
}

// Implement tracker.Tracker interface (stub methods)
func (m *mockPlanFetcherTracker) CreateIssue(ctx context.Context, issue *tracker.Issue) (*tracker.Issue, error) {
	return nil, nil
}
func (m *mockPlanFetcherTracker) UpdateIssue(ctx context.Context, id string, updates *tracker.IssueUpdate) error {
	return nil
}
func (m *mockPlanFetcherTracker) GetIssue(ctx context.Context, id string) (*tracker.Issue, error) {
	return nil, nil
}
func (m *mockPlanFetcherTracker) SearchIssues(ctx context.Context, query string) ([]*tracker.Issue, error) {
	return nil, nil
}
func (m *mockPlanFetcherTracker) CreateSubIssue(ctx context.Context, parentID string, issue *tracker.Issue) (*tracker.Issue, error) {
	return nil, nil
}
func (m *mockPlanFetcherTracker) GetSubIssues(ctx context.Context, parentID string) ([]*tracker.Issue, error) {
	return nil, nil
}
func (m *mockPlanFetcherTracker) AddComment(ctx context.Context, issueID string, body string) (*tracker.Comment, error) {
	return nil, nil
}
func (m *mockPlanFetcherTracker) GetComments(ctx context.Context, issueID string) ([]*tracker.Comment, error) {
	return nil, nil
}
func (m *mockPlanFetcherTracker) TransitionIssue(ctx context.Context, id string, status tracker.Status) error {
	return nil
}
func (m *mockPlanFetcherTracker) GetAvailableStatuses(ctx context.Context, id string) ([]tracker.Status, error) {
	return nil, nil
}
func (m *mockPlanFetcherTracker) GetTeams(ctx context.Context) ([]tracker.Team, error) {
	return nil, nil
}
func (m *mockPlanFetcherTracker) GetProjects(ctx context.Context, teamID string) ([]tracker.Project, error) {
	return nil, nil
}

// mockNonPlanFetcherTracker implements only tracker.Tracker (not PlanFetcher)
type mockNonPlanFetcherTracker struct{}

func (m *mockNonPlanFetcherTracker) CreateIssue(ctx context.Context, issue *tracker.Issue) (*tracker.Issue, error) {
	return nil, nil
}
func (m *mockNonPlanFetcherTracker) UpdateIssue(ctx context.Context, id string, updates *tracker.IssueUpdate) error {
	return nil
}
func (m *mockNonPlanFetcherTracker) GetIssue(ctx context.Context, id string) (*tracker.Issue, error) {
	return nil, nil
}
func (m *mockNonPlanFetcherTracker) SearchIssues(ctx context.Context, query string) ([]*tracker.Issue, error) {
	return nil, nil
}
func (m *mockNonPlanFetcherTracker) CreateSubIssue(ctx context.Context, parentID string, issue *tracker.Issue) (*tracker.Issue, error) {
	return nil, nil
}
func (m *mockNonPlanFetcherTracker) GetSubIssues(ctx context.Context, parentID string) ([]*tracker.Issue, error) {
	return nil, nil
}
func (m *mockNonPlanFetcherTracker) AddComment(ctx context.Context, issueID string, body string) (*tracker.Comment, error) {
	return nil, nil
}
func (m *mockNonPlanFetcherTracker) GetComments(ctx context.Context, issueID string) ([]*tracker.Comment, error) {
	return nil, nil
}
func (m *mockNonPlanFetcherTracker) TransitionIssue(ctx context.Context, id string, status tracker.Status) error {
	return nil
}
func (m *mockNonPlanFetcherTracker) GetAvailableStatuses(ctx context.Context, id string) ([]tracker.Status, error) {
	return nil, nil
}
func (m *mockNonPlanFetcherTracker) GetTeams(ctx context.Context) ([]tracker.Team, error) {
	return nil, nil
}
func (m *mockNonPlanFetcherTracker) GetProjects(ctx context.Context, teamID string) ([]tracker.Project, error) {
	return nil, nil
}

// setupTestCache creates a temporary JIG_HOME and initializes the cache for testing
func setupTestCache(t *testing.T) (cleanup func()) {
	t.Helper()

	// Create temp JIG_HOME
	jigHome, err := os.MkdirTemp("", "jig-test-*")
	if err != nil {
		t.Fatalf("failed to create temp JIG_HOME: %v", err)
	}

	// Set JIG_HOME env var
	oldJigHome := os.Getenv("JIG_HOME")
	os.Setenv("JIG_HOME", jigHome)

	// Create cache directories
	cacheDir := filepath.Join(jigHome, "cache", "plans")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	// Initialize state
	if err := state.Init(); err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	return func() {
		os.Setenv("JIG_HOME", oldJigHome)
		os.RemoveAll(jigHome)
	}
}

// savePlanToCache saves a plan to the test cache
func savePlanToCache(t *testing.T, p *plan.Plan) {
	t.Helper()
	if err := state.DefaultCache.SavePlan(p); err != nil {
		t.Fatalf("failed to save plan to cache: %v", err)
	}
}

func TestLookupPlanByIDWithFallback_CacheHit(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Create and cache a plan
	testPlan := plan.NewPlan("PLAN-123", "Test Plan", "testuser")
	testPlan.IssueID = "NUM-100"
	testPlan.ProblemStatement = "Test problem"
	testPlan.ProposedSolution = "Test solution"
	savePlanToCache(t, testPlan)

	ctx := context.Background()
	cfg := &config.Config{}

	// Look up by plan ID - should find in cache without remote call
	p, planID, err := lookupPlanByIDWithFallback("PLAN-123", &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
	})

	if err != nil {
		t.Fatalf("lookupPlanByIDWithFallback failed: %v", err)
	}
	if p == nil {
		t.Fatal("expected plan to be found")
	}
	if p.ID != "PLAN-123" {
		t.Errorf("expected plan ID 'PLAN-123', got '%s'", p.ID)
	}
	if planID != "PLAN-123" {
		t.Errorf("expected returned planID 'PLAN-123', got '%s'", planID)
	}
}

func TestLookupPlanByIDWithFallback_CacheHit_ByIssueID(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Create and cache a plan with an issue ID
	testPlan := plan.NewPlan("PLAN-456", "Test Plan 2", "testuser")
	testPlan.IssueID = "NUM-200"
	testPlan.ProblemStatement = "Test problem"
	testPlan.ProposedSolution = "Test solution"
	savePlanToCache(t, testPlan)

	ctx := context.Background()
	cfg := &config.Config{}

	// Look up by issue ID - should find in cache
	p, _, err := lookupPlanByIDWithFallback("NUM-200", &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
	})

	if err != nil {
		t.Fatalf("lookupPlanByIDWithFallback failed: %v", err)
	}
	if p == nil {
		t.Fatal("expected plan to be found by issue ID")
	}
	if p.IssueID != "NUM-200" {
		t.Errorf("expected issue ID 'NUM-200', got '%s'", p.IssueID)
	}
}

func TestLookupPlanByIDWithFallback_NoRemoteFallback(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	cfg := &config.Config{}

	// Look up non-existent plan with FetchFromRemote=false
	p, planID, err := lookupPlanByIDWithFallback("NON-EXISTENT", &LookupPlanOptions{
		FetchFromRemote: false,
		Config:          cfg,
		Context:         ctx,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("expected nil plan when not in cache and remote disabled")
	}
	if planID != "" {
		t.Errorf("expected empty planID, got '%s'", planID)
	}
}

func TestLookupPlanByIDWithFallback_NilOptions(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Look up non-existent plan with nil options - should not panic or fetch remote
	p, planID, err := lookupPlanByIDWithFallback("NON-EXISTENT", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("expected nil plan when not in cache and no options")
	}
	if planID != "" {
		t.Errorf("expected empty planID, got '%s'", planID)
	}
}

func TestLookupPlanByIDWithFallback_TrackerNotConfigured(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	cfg := &config.Config{
		Default: config.DefaultConfig{
			Tracker: "", // No tracker configured
		},
	}

	// Look up non-existent plan - should fail to connect to tracker
	_, _, err := lookupPlanByIDWithFallback("NON-EXISTENT", &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
	})

	if err == nil {
		t.Error("expected error when tracker not configured")
	}
}

func TestLookupPlanByIDWithFallback_TrackerDoesNotSupportPlanFetching(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	cfg := &config.Config{}

	// Use a tracker that doesn't implement PlanFetcher
	p, planID, err := lookupPlanByIDWithFallback("NON-EXISTENT", &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
		TrackerGetter: func(c *config.Config) (tracker.Tracker, error) {
			return &mockNonPlanFetcherTracker{}, nil
		},
	})

	// Should return nil without error (tracker just doesn't support plan fetching)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("expected nil plan when tracker doesn't support plan fetching")
	}
	if planID != "" {
		t.Errorf("expected empty planID, got '%s'", planID)
	}
}

func TestLookupPlanByIDWithFallback_RemoteFetchSuccess(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	cfg := &config.Config{}

	// Create a plan that the mock tracker will return
	remotePlan := plan.NewPlan("REMOTE-PLAN", "Remote Plan", "remoteuser")
	remotePlan.IssueID = "REMOTE-ISSUE"
	remotePlan.ProblemStatement = "Remote problem"
	remotePlan.ProposedSolution = "Remote solution"

	// Use a mock tracker that returns a plan
	p, planID, err := lookupPlanByIDWithFallback("REMOTE-ISSUE", &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
		TrackerGetter: func(c *config.Config) (tracker.Tracker, error) {
			return &mockPlanFetcherTracker{plan: remotePlan}, nil
		},
	})

	if err != nil {
		t.Fatalf("lookupPlanByIDWithFallback failed: %v", err)
	}
	if p == nil {
		t.Fatal("expected plan to be fetched from remote")
	}
	if p.ID != "REMOTE-PLAN" {
		t.Errorf("expected plan ID 'REMOTE-PLAN', got '%s'", p.ID)
	}
	if planID != "REMOTE-PLAN" {
		t.Errorf("expected returned planID 'REMOTE-PLAN', got '%s'", planID)
	}

	// Verify the plan was cached
	cachedPlan, err := state.DefaultCache.GetPlan("REMOTE-PLAN")
	if err != nil {
		t.Fatalf("failed to get cached plan: %v", err)
	}
	if cachedPlan == nil {
		t.Error("expected plan to be cached after remote fetch")
	}
}

func TestLookupPlanByIDWithFallback_RemoteFetchReturnsNil(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	cfg := &config.Config{}

	// Use a mock tracker that returns nil (plan not found)
	p, planID, err := lookupPlanByIDWithFallback("NOT-FOUND", &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
		TrackerGetter: func(c *config.Config) (tracker.Tracker, error) {
			return &mockPlanFetcherTracker{plan: nil}, nil
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("expected nil plan when remote returns nil")
	}
	if planID != "" {
		t.Errorf("expected empty planID, got '%s'", planID)
	}
}

func TestLookupPlanByIDWithFallback_RemoteFetchReturnsError(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	cfg := &config.Config{}

	// Use a mock tracker that returns an error
	_, _, err := lookupPlanByIDWithFallback("ERROR-ISSUE", &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
		TrackerGetter: func(c *config.Config) (tracker.Tracker, error) {
			return &mockPlanFetcherTracker{err: fmt.Errorf("API error: rate limited")}, nil
		},
	})

	if err == nil {
		t.Error("expected error when remote fetch fails")
	}
	if err != nil && !strings.Contains(err.Error(), "could not fetch plan") {
		t.Errorf("expected 'could not fetch plan' error, got: %v", err)
	}
}

func TestLookupPlanByIDWithFallback_TrackerGetterError(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	cfg := &config.Config{}

	// Use a tracker getter that returns an error
	_, _, err := lookupPlanByIDWithFallback("ANY-ISSUE", &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
		TrackerGetter: func(c *config.Config) (tracker.Tracker, error) {
			return nil, fmt.Errorf("failed to connect")
		},
	})

	if err == nil {
		t.Error("expected error when tracker getter fails")
	}
	if err != nil && !strings.Contains(err.Error(), "could not connect to tracker") {
		t.Errorf("expected 'could not connect to tracker' error, got: %v", err)
	}
}

func TestLookupPlanOptions(t *testing.T) {
	// Test that LookupPlanOptions struct works correctly
	ctx := context.Background()
	cfg := &config.Config{}

	opts := &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
	}

	if !opts.FetchFromRemote {
		t.Error("expected FetchFromRemote to be true")
	}
	if opts.Config != cfg {
		t.Error("expected Config to match")
	}
	if opts.Context != ctx {
		t.Error("expected Context to match")
	}
}

func TestLookupPlanByID_Basic(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Create and cache a plan
	testPlan := plan.NewPlan("PLAN-BASIC", "Basic Test Plan", "testuser")
	testPlan.IssueID = "NUM-BASIC"
	testPlan.ProblemStatement = "Basic problem"
	testPlan.ProposedSolution = "Basic solution"
	savePlanToCache(t, testPlan)

	// Test lookupPlanByID directly
	p, planID, err := lookupPlanByID("PLAN-BASIC")
	if err != nil {
		t.Fatalf("lookupPlanByID failed: %v", err)
	}
	if p == nil {
		t.Fatal("expected plan to be found")
	}
	if p.ID != "PLAN-BASIC" {
		t.Errorf("expected plan ID 'PLAN-BASIC', got '%s'", p.ID)
	}
	if planID != "PLAN-BASIC" {
		t.Errorf("expected returned planID 'PLAN-BASIC', got '%s'", planID)
	}
}

func TestLookupPlanByID_NotFound(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Look up non-existent plan
	p, planID, err := lookupPlanByID("NON-EXISTENT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("expected nil plan for non-existent ID")
	}
	if planID != "" {
		t.Errorf("expected empty planID, got '%s'", planID)
	}
}

func TestLookupPlanByIDWithFallback_RemoteFetchError(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	// Use an invalid tracker config to trigger an error
	cfg := &config.Config{
		Default: config.DefaultConfig{
			Tracker: "unknown-tracker", // Invalid tracker type
		},
	}

	_, _, err := lookupPlanByIDWithFallback("NON-EXISTENT", &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
	})

	if err == nil {
		t.Error("expected error for unknown tracker")
	}
	if err != nil && !strings.Contains(err.Error(), "could not connect to tracker") {
		t.Errorf("expected 'could not connect to tracker' error, got: %v", err)
	}
}

func TestMockPlanFetcherTracker(t *testing.T) {
	// Test the mock tracker implements PlanFetcher correctly
	ctx := context.Background()
	testPlan := plan.NewPlan("MOCK-PLAN", "Mock Plan", "testuser")

	mock := &mockPlanFetcherTracker{plan: testPlan}

	// Verify it implements PlanFetcher
	fetcher, ok := interface{}(mock).(tracker.PlanFetcher)
	if !ok {
		t.Fatal("mockPlanFetcherTracker should implement PlanFetcher")
	}

	p, err := fetcher.FetchPlanFromIssue(ctx, "TEST-123")
	if err != nil {
		t.Fatalf("FetchPlanFromIssue failed: %v", err)
	}
	if p == nil {
		t.Fatal("expected plan to be returned")
	}
	if p.ID != "MOCK-PLAN" {
		t.Errorf("expected plan ID 'MOCK-PLAN', got '%s'", p.ID)
	}
}

func TestMockPlanFetcherTracker_Error(t *testing.T) {
	ctx := context.Background()

	mock := &mockPlanFetcherTracker{err: fmt.Errorf("fetch failed")}

	p, err := mock.FetchPlanFromIssue(ctx, "TEST-123")
	if err == nil {
		t.Error("expected error from mock")
	}
	if p != nil {
		t.Error("expected nil plan on error")
	}
}

func TestMockNonPlanFetcherTracker(t *testing.T) {
	// Verify mockNonPlanFetcherTracker does NOT implement PlanFetcher
	mock := &mockNonPlanFetcherTracker{}

	_, ok := interface{}(mock).(tracker.PlanFetcher)
	if ok {
		t.Error("mockNonPlanFetcherTracker should NOT implement PlanFetcher")
	}

	// But it should implement Tracker
	_, ok = interface{}(mock).(tracker.Tracker)
	if !ok {
		t.Error("mockNonPlanFetcherTracker should implement Tracker")
	}
}
