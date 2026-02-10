package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charleslr/jig/internal/plan"
)

func TestSavePlanPreservesRawContent(t *testing.T) {
	// Create a temporary directory for the test cache
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create cache directories
	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Create a plan with raw content that includes extra sections
	rawContent := `---
id: test-preserve
title: Test Preservation
status: draft
author: testuser
---

# Test Preservation

## Problem Statement

The problem to solve.

## Proposed Solution

The proposed solution.

## Extra Custom Section

This extra content should be preserved exactly as-is.

### Nested Section

More custom content that should not be lost.
`

	p := &plan.Plan{
		ID:         "test-preserve",
		Title:      "Test Preservation",
		Status:     plan.StatusDraft,
		Author:     "testuser",
		RawContent: rawContent,
	}

	// Save the plan
	if err := cache.SavePlan(p); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Read the saved markdown file directly
	mdPath := filepath.Join(tmpDir, "plans", "test-preserve.md")
	savedContent, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("failed to read saved markdown: %v", err)
	}

	content := string(savedContent)

	// Verify frontmatter values are correct (format may differ due to Serialize)
	if !strings.Contains(content, "id: test-preserve") {
		t.Error("saved content should contain plan ID")
	}
	if !strings.Contains(content, "title: Test Preservation") {
		t.Error("saved content should contain title")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("saved content should contain status")
	}
	if !strings.Contains(content, "author: testuser") {
		t.Error("saved content should contain author")
	}

	// Verify custom sections are preserved (body content)
	if !strings.Contains(content, "Extra Custom Section") {
		t.Error("saved content should contain 'Extra Custom Section'")
	}
	if !strings.Contains(content, "This extra content should be preserved exactly as-is") {
		t.Error("saved content should contain the extra content text")
	}
	if !strings.Contains(content, "Nested Section") {
		t.Error("saved content should contain nested sections")
	}
	if !strings.Contains(content, "More custom content that should not be lost") {
		t.Error("saved content should preserve nested section content")
	}
}

func TestSavePlanWithoutRawContent(t *testing.T) {
	// Create a temporary directory for the test cache
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create cache directories
	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Create a plan without raw content (simulating a programmatically created plan)
	p := &plan.Plan{
		ID:               "test-no-raw",
		Title:            "Test No Raw Content",
		Status:           plan.StatusDraft,
		Author:           "testuser",
		ProblemStatement: "Test problem",
		ProposedSolution: "Test solution",
		RawContent:       "", // Empty raw content
	}

	// Save should succeed - Serialize will reconstruct markdown from plan fields
	err = cache.SavePlan(p)
	if err != nil {
		t.Errorf("SavePlan() should succeed when RawContent is empty (Serialize reconstructs it): %v", err)
	}

	// Verify the markdown was generated
	mdContent, err := cache.GetPlanMarkdown("test-no-raw")
	if err != nil {
		t.Errorf("GetPlanMarkdown() failed: %v", err)
	}
	if !strings.Contains(mdContent, "Test No Raw Content") {
		t.Error("generated markdown should contain the title")
	}
	if !strings.Contains(mdContent, "status: draft") {
		t.Error("generated markdown should contain the status")
	}
}

func TestSavePlanRequiresID(t *testing.T) {
	// Create a temporary directory for the test cache
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create cache directories
	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Create a plan without ID
	p := &plan.Plan{
		ID:         "", // Empty ID
		Title:      "Test No ID",
		RawContent: "some content",
	}

	// Save should fail
	err = cache.SavePlan(p)
	if err == nil {
		t.Error("SavePlan() should error when ID is empty")
	}
	if !strings.Contains(err.Error(), "ID is required") {
		t.Errorf("error should mention 'ID is required', got: %v", err)
	}
}

func TestGetPlanMarkdown(t *testing.T) {
	// Create a temporary directory for the test cache
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create cache directories
	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	rawContent := `---
id: test-get-md
title: Test Get Markdown
status: draft
author: testuser
---

# Test Get Markdown

## Problem Statement

Problem.

## Proposed Solution

Solution.

## Custom Content

This should be retrievable via GetPlanMarkdown.
`

	p := &plan.Plan{
		ID:         "test-get-md",
		Title:      "Test Get Markdown",
		Status:     plan.StatusDraft,
		Author:     "testuser",
		RawContent: rawContent,
	}

	// Save the plan
	if err := cache.SavePlan(p); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Get the markdown back
	content, err := cache.GetPlanMarkdown("test-get-md")
	if err != nil {
		t.Fatalf("GetPlanMarkdown() error = %v", err)
	}

	// Check frontmatter values are preserved (format may differ due to Serialize)
	if !strings.Contains(content, "id: test-get-md") {
		t.Error("GetPlanMarkdown() should contain the plan ID")
	}
	if !strings.Contains(content, "title: Test Get Markdown") {
		t.Error("GetPlanMarkdown() should contain the title")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("GetPlanMarkdown() should contain the status")
	}
	if !strings.Contains(content, "author: testuser") {
		t.Error("GetPlanMarkdown() should contain the author")
	}

	// Check body content is preserved
	if !strings.Contains(content, "Custom Content") {
		t.Error("GetPlanMarkdown() should return content with custom sections")
	}
	if !strings.Contains(content, "This should be retrievable via GetPlanMarkdown.") {
		t.Error("GetPlanMarkdown() should preserve custom content text")
	}
}

func TestGetPlanMarkdownNotFound(t *testing.T) {
	// Create a temporary directory for the test cache
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create cache directories
	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Try to get a non-existent plan
	content, err := cache.GetPlanMarkdown("nonexistent")
	if err != nil {
		t.Fatalf("GetPlanMarkdown() unexpected error = %v", err)
	}
	if content != "" {
		t.Errorf("GetPlanMarkdown() should return empty string for non-existent plan, got %q", content)
	}
}

// TestImplementFlowPreservesAllSections tests the full implement flow:
// SavePlan -> GetPlan -> Serialize
// This is the exact path used by `jig implement` and must preserve all sections.
func TestImplementFlowPreservesAllSections(t *testing.T) {
	// Create a temporary directory for the test cache
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create cache directories
	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Create a plan with extra sections (like Implementation Details)
	rawContent := `---
id: impl-flow-test
title: Implementation Flow Test
status: draft
author: testuser
---

# Implementation Flow Test

## Problem Statement

Testing the implementation flow.

## Proposed Solution

Use the existing flow.

## Acceptance Criteria

- [ ] All sections preserved
- [ ] Status can be updated

## Implementation Details

### File: internal/foo/bar.go

` + "```go" + `
func Example() {
    // This code block should be preserved
}
` + "```" + `

## Verification

1. Build the project
2. Run tests
`

	// Parse the raw content to get a Plan struct (simulates what happens when plan is saved)
	p, err := plan.Parse([]byte(rawContent))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Save the plan to cache
	if err := cache.SavePlan(p); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Get the plan from cache (what implement.go does)
	retrieved, err := cache.GetPlan("impl-flow-test")
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}

	// Verify RawContent is preserved through JSON serialization
	if retrieved.RawContent == "" {
		t.Fatal("GetPlan() returned plan with empty RawContent")
	}

	// Change status (what implement.go does before calling Prepare)
	retrieved.Status = plan.StatusInProgress

	// Serialize the plan (what runner.Prepare does to write .jig/plan.md)
	serialized, err := plan.Serialize(retrieved)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	serializedStr := string(serialized)

	// Verify all sections are preserved
	requiredSections := []string{
		"## Problem Statement",
		"## Proposed Solution",
		"## Acceptance Criteria",
		"## Implementation Details",
		"## Verification",
		"func Example()",
	}

	for _, section := range requiredSections {
		if !strings.Contains(serializedStr, section) {
			t.Errorf("serialized plan missing section: %s", section)
		}
	}

	// Verify status was updated in frontmatter
	if !strings.Contains(serializedStr, "status: in-progress") {
		t.Error("serialized plan should have updated status in frontmatter")
	}
}

func TestGetCachedPlan(t *testing.T) {
	// Create a temporary directory for the test cache
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create cache directories
	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	rawContent := `---
id: test-get-cached
title: Test Get Cached Plan
status: draft
author: testuser
---

# Test Get Cached Plan

## Problem Statement

Test problem.

## Proposed Solution

Test solution.
`

	p := &plan.Plan{
		ID:         "test-get-cached",
		Title:      "Test Get Cached Plan",
		Status:     plan.StatusDraft,
		Author:     "testuser",
		IssueID:    "NUM-123",
		RawContent: rawContent,
	}

	// Save the plan
	if err := cache.SavePlan(p); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Get the cached plan with metadata
	cached, err := cache.GetCachedPlan("test-get-cached")
	if err != nil {
		t.Fatalf("GetCachedPlan() error = %v", err)
	}
	if cached == nil {
		t.Fatal("GetCachedPlan() returned nil")
	}

	// Verify plan data
	if cached.Plan == nil {
		t.Fatal("GetCachedPlan() returned nil Plan")
	}
	if cached.Plan.ID != "test-get-cached" {
		t.Errorf("expected Plan.ID 'test-get-cached', got %q", cached.Plan.ID)
	}
	if cached.Plan.IssueID != "NUM-123" {
		t.Errorf("expected Plan.IssueID 'NUM-123', got %q", cached.Plan.IssueID)
	}

	// Verify metadata
	if cached.IssueID != "test-get-cached" {
		t.Errorf("expected IssueID 'test-get-cached', got %q", cached.IssueID)
	}
	if cached.CachedAt.IsZero() {
		t.Error("expected CachedAt to be set")
	}

	// SyncedAt should be nil initially
	if cached.SyncedAt != nil {
		t.Errorf("expected SyncedAt to be nil, got %v", cached.SyncedAt)
	}
}

func TestGetCachedPlan_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	cached, err := cache.GetCachedPlan("nonexistent")
	if err != nil {
		t.Fatalf("GetCachedPlan() unexpected error = %v", err)
	}
	if cached != nil {
		t.Error("GetCachedPlan() should return nil for nonexistent plan")
	}
}

func TestListCachedPlans(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Create multiple plans
	plans := []*plan.Plan{
		{
			ID:         "plan-1",
			Title:      "Plan 1",
			IssueID:    "NUM-1",
			RawContent: "---\nid: plan-1\ntitle: Plan 1\nstatus: draft\nauthor: test\n---\n# Plan 1\n## Problem Statement\nP1\n## Proposed Solution\nS1\n",
		},
		{
			ID:         "plan-2",
			Title:      "Plan 2",
			IssueID:    "NUM-2",
			RawContent: "---\nid: plan-2\ntitle: Plan 2\nstatus: draft\nauthor: test\n---\n# Plan 2\n## Problem Statement\nP2\n## Proposed Solution\nS2\n",
		},
		{
			ID:         "plan-3",
			Title:      "Plan 3",
			IssueID:    "", // No linked issue
			RawContent: "---\nid: plan-3\ntitle: Plan 3\nstatus: draft\nauthor: test\n---\n# Plan 3\n## Problem Statement\nP3\n## Proposed Solution\nS3\n",
		},
	}

	for _, p := range plans {
		if err := cache.SavePlan(p); err != nil {
			t.Fatalf("SavePlan(%s) error = %v", p.ID, err)
		}
	}

	// List all cached plans
	cachedPlans, err := cache.ListCachedPlans()
	if err != nil {
		t.Fatalf("ListCachedPlans() error = %v", err)
	}

	if len(cachedPlans) != 3 {
		t.Errorf("expected 3 cached plans, got %d", len(cachedPlans))
	}

	// Verify each plan has proper metadata
	for _, cp := range cachedPlans {
		if cp.Plan == nil {
			t.Error("ListCachedPlans() returned plan with nil Plan")
			continue
		}
		if cp.CachedAt.IsZero() {
			t.Errorf("plan %s has zero CachedAt", cp.Plan.ID)
		}
	}
}

func TestListCachedPlans_DirectoryNotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Don't create the plans directory - only create the cache base
	cache := &Cache{dir: tmpDir}

	plans, err := cache.ListCachedPlans()
	if err != nil {
		t.Fatalf("ListCachedPlans() error = %v", err)
	}
	if plans != nil {
		t.Errorf("expected nil for non-existent directory, got %d plans", len(plans))
	}
}

func TestListCachedPlans_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	cachedPlans, err := cache.ListCachedPlans()
	if err != nil {
		t.Fatalf("ListCachedPlans() error = %v", err)
	}
	if cachedPlans != nil && len(cachedPlans) != 0 {
		t.Errorf("expected empty list for empty cache, got %d plans", len(cachedPlans))
	}
}

func TestGetCachedPlan_InvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Write invalid JSON to the cache file
	invalidJSON := []byte("this is not valid json {{{")
	path := filepath.Join(tmpDir, "plans", "invalid-plan.json")
	if err := os.WriteFile(path, invalidJSON, 0644); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}

	cached, err := cache.GetCachedPlan("invalid-plan")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if cached != nil {
		t.Error("expected nil result for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse plan cache") {
		t.Errorf("expected 'failed to parse plan cache' error, got: %v", err)
	}
}

func TestListCachedPlans_SkipsNonJSONFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Save a valid plan
	validPlan := &plan.Plan{
		ID:         "valid-plan",
		Title:      "Valid Plan",
		IssueID:    "NUM-123",
		RawContent: "---\nid: valid-plan\ntitle: Valid Plan\nstatus: draft\nauthor: test\n---\n# Valid\n## Problem Statement\nP\n## Proposed Solution\nS\n",
	}
	if err := cache.SavePlan(validPlan); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Add a non-JSON file that should be skipped
	txtPath := filepath.Join(tmpDir, "plans", "readme.txt")
	if err := os.WriteFile(txtPath, []byte("readme content"), 0644); err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}

	// Add a subdirectory that should be skipped
	subdir := filepath.Join(tmpDir, "plans", "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	plans, err := cache.ListCachedPlans()
	if err != nil {
		t.Fatalf("ListCachedPlans() error = %v", err)
	}

	// Should only return the valid plan, ignoring .txt and .md files and directories
	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Plan.ID != "valid-plan" {
		t.Errorf("expected plan ID 'valid-plan', got %q", plans[0].Plan.ID)
	}
}

func TestListCachedPlans_SkipsInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Save a valid plan
	validPlan := &plan.Plan{
		ID:         "valid-plan",
		Title:      "Valid Plan",
		IssueID:    "NUM-123",
		RawContent: "---\nid: valid-plan\ntitle: Valid Plan\nstatus: draft\nauthor: test\n---\n# Valid\n## Problem Statement\nP\n## Proposed Solution\nS\n",
	}
	if err := cache.SavePlan(validPlan); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Add an invalid JSON file that should be skipped
	invalidPath := filepath.Join(tmpDir, "plans", "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write invalid json: %v", err)
	}

	plans, err := cache.ListCachedPlans()
	if err != nil {
		t.Fatalf("ListCachedPlans() error = %v", err)
	}

	// Should only return the valid plan, skipping the invalid JSON
	if len(plans) != 1 {
		t.Errorf("expected 1 plan (skipping invalid), got %d", len(plans))
	}
	if plans[0].Plan.ID != "valid-plan" {
		t.Errorf("expected plan ID 'valid-plan', got %q", plans[0].Plan.ID)
	}
}

func TestMarkPlanSynced(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	p := &plan.Plan{
		ID:         "test-mark-synced",
		Title:      "Test Mark Synced",
		IssueID:    "NUM-456",
		RawContent: "---\nid: test-mark-synced\ntitle: Test Mark Synced\nstatus: draft\nauthor: test\n---\n# Test\n## Problem Statement\nP\n## Proposed Solution\nS\n",
	}

	if err := cache.SavePlan(p); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Verify SyncedAt is nil initially
	cached, err := cache.GetCachedPlan("test-mark-synced")
	if err != nil {
		t.Fatalf("GetCachedPlan() error = %v", err)
	}
	if cached.SyncedAt != nil {
		t.Error("expected SyncedAt to be nil initially")
	}

	// Mark as synced
	if err := cache.MarkPlanSynced("test-mark-synced"); err != nil {
		t.Fatalf("MarkPlanSynced() error = %v", err)
	}

	// Verify SyncedAt is now set
	cached, err = cache.GetCachedPlan("test-mark-synced")
	if err != nil {
		t.Fatalf("GetCachedPlan() after mark error = %v", err)
	}
	if cached.SyncedAt == nil {
		t.Fatal("expected SyncedAt to be set after MarkPlanSynced")
	}
	if cached.SyncedAt.IsZero() {
		t.Error("SyncedAt should not be zero")
	}
}

func TestMarkPlanSynced_GetCachedPlanError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Write invalid JSON to the cache file
	invalidJSON := []byte("this is not valid json {{{")
	path := filepath.Join(tmpDir, "plans", "corrupted-plan.json")
	if err := os.WriteFile(path, invalidJSON, 0644); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}

	err = cache.MarkPlanSynced("corrupted-plan")
	if err == nil {
		t.Error("expected error for corrupted plan")
	}
	if !strings.Contains(err.Error(), "failed to get cached plan") {
		t.Errorf("expected 'failed to get cached plan' error, got: %v", err)
	}
}

func TestMarkPlanSynced_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	err = cache.MarkPlanSynced("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}


func TestNeedsSync(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name     string
		cached   *CachedPlan
		expected bool
	}{
		{
			name: "nil plan returns false",
			cached: &CachedPlan{
				Plan:      nil,
				UpdatedAt: now,
			},
			expected: false,
		},
		{
			name: "plan with empty issue ID returns false",
			cached: &CachedPlan{
				Plan:      &plan.Plan{ID: "test", IssueID: ""},
				UpdatedAt: now,
			},
			expected: false,
		},
		{
			name: "plan never synced returns true",
			cached: &CachedPlan{
				Plan:      &plan.Plan{ID: "test", IssueID: "NUM-123"},
				UpdatedAt: now,
				SyncedAt:  nil,
			},
			expected: true,
		},
		{
			name: "plan updated after sync returns true",
			cached: &CachedPlan{
				Plan:      &plan.Plan{ID: "test", IssueID: "NUM-123"},
				UpdatedAt: future,
				SyncedAt:  &now,
			},
			expected: true,
		},
		{
			name: "plan synced after update returns false",
			cached: &CachedPlan{
				Plan:      &plan.Plan{ID: "test", IssueID: "NUM-123"},
				UpdatedAt: past,
				SyncedAt:  &now,
			},
			expected: false,
		},
		{
			name: "plan synced at same time as update returns false",
			cached: &CachedPlan{
				Plan:      &plan.Plan{ID: "test", IssueID: "NUM-123"},
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

func TestNeedsSyncIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, subdir := range []string{"plans", "issues"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, subdir), 0755); err != nil {
			t.Fatalf("failed to create cache subdir: %v", err)
		}
	}

	cache := &Cache{dir: tmpDir}

	// Plan with linked issue - should need sync initially
	p1 := &plan.Plan{
		ID:         "needs-sync-test",
		Title:      "Needs Sync Test",
		IssueID:    "NUM-789",
		RawContent: "---\nid: needs-sync-test\ntitle: Needs Sync Test\nstatus: draft\nauthor: test\n---\n# Test\n## Problem Statement\nP\n## Proposed Solution\nS\n",
	}

	if err := cache.SavePlan(p1); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	cached, err := cache.GetCachedPlan("needs-sync-test")
	if err != nil {
		t.Fatalf("GetCachedPlan() error = %v", err)
	}

	// Should need sync (never synced)
	if !cached.NeedsSync() {
		t.Error("plan with linked issue should need sync initially")
	}

	// Mark as synced
	if err := cache.MarkPlanSynced("needs-sync-test"); err != nil {
		t.Fatalf("MarkPlanSynced() error = %v", err)
	}

	// Re-fetch and check
	cached, err = cache.GetCachedPlan("needs-sync-test")
	if err != nil {
		t.Fatalf("GetCachedPlan() error = %v", err)
	}

	// Should NOT need sync (just synced)
	if cached.NeedsSync() {
		t.Error("plan should not need sync immediately after marking synced")
	}
}
