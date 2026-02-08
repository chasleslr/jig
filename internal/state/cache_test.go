package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	// Verify the content matches exactly
	if string(savedContent) != rawContent {
		t.Error("saved content does not match original RawContent")
		t.Errorf("expected:\n%s", rawContent)
		t.Errorf("got:\n%s", string(savedContent))
	}

	// Verify custom sections are preserved
	if !strings.Contains(string(savedContent), "Extra Custom Section") {
		t.Error("saved content should contain 'Extra Custom Section'")
	}
	if !strings.Contains(string(savedContent), "This extra content should be preserved exactly as-is") {
		t.Error("saved content should contain the extra content text")
	}
}

func TestSavePlanRequiresRawContent(t *testing.T) {
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

	// Create a plan without raw content
	p := &plan.Plan{
		ID:         "test-no-raw",
		Title:      "Test No Raw Content",
		Status:     plan.StatusDraft,
		Author:     "testuser",
		RawContent: "", // Empty raw content
	}

	// Save should fail
	err = cache.SavePlan(p)
	if err == nil {
		t.Error("SavePlan() should error when RawContent is empty")
	}
	if !strings.Contains(err.Error(), "no raw content") {
		t.Errorf("error should mention 'no raw content', got: %v", err)
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

	if content != rawContent {
		t.Error("GetPlanMarkdown() content does not match original")
	}

	if !strings.Contains(content, "Custom Content") {
		t.Error("GetPlanMarkdown() should return content with custom sections")
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

func TestSyncPRForIssueNoMetadata(t *testing.T) {
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

	// Try to sync an issue with no metadata
	_, err = cache.SyncPRForIssue("nonexistent")
	if err == nil {
		t.Error("SyncPRForIssue() should error when no metadata exists")
	}
	if !strings.Contains(err.Error(), "no metadata found") {
		t.Errorf("error should mention 'no metadata found', got: %v", err)
	}
}

func TestSyncPRForIssueNoBranch(t *testing.T) {
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

	// Create metadata without a branch name
	meta := &IssueMetadata{
		IssueID:    "test-issue",
		BranchName: "", // No branch name
	}
	if err := cache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// Try to sync
	_, err = cache.SyncPRForIssue("test-issue")
	if err == nil {
		t.Error("SyncPRForIssue() should error when no branch name exists")
	}
	if !strings.Contains(err.Error(), "no branch name") {
		t.Errorf("error should mention 'no branch name', got: %v", err)
	}
}

func TestSyncPRForIssueAlreadyHasPR(t *testing.T) {
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

	// Create metadata with existing PR number
	meta := &IssueMetadata{
		IssueID:    "test-issue",
		BranchName: "feature-branch",
		PRNumber:   42,
		PRURL:      "https://github.com/test/repo/pull/42",
	}
	if err := cache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// Sync should return existing PR number without calling GitHub
	prNumber, err := cache.SyncPRForIssue("test-issue")
	if err != nil {
		t.Fatalf("SyncPRForIssue() error = %v", err)
	}
	if prNumber != 42 {
		t.Errorf("SyncPRForIssue() = %d, want 42", prNumber)
	}
}

func TestListIssueMetadata(t *testing.T) {
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

	// Test empty list
	metadata, err := cache.ListIssueMetadata()
	if err != nil {
		t.Fatalf("ListIssueMetadata() error = %v", err)
	}
	if len(metadata) != 0 {
		t.Errorf("ListIssueMetadata() = %d items, want 0", len(metadata))
	}

	// Add some metadata
	for _, id := range []string{"NUM-1", "NUM-2", "NUM-3"} {
		meta := &IssueMetadata{
			IssueID:    id,
			BranchName: id + "-branch",
		}
		if err := cache.SaveIssueMetadata(meta); err != nil {
			t.Fatalf("SaveIssueMetadata() error = %v", err)
		}
	}

	// Test list with items
	metadata, err = cache.ListIssueMetadata()
	if err != nil {
		t.Fatalf("ListIssueMetadata() error = %v", err)
	}
	if len(metadata) != 3 {
		t.Errorf("ListIssueMetadata() = %d items, want 3", len(metadata))
	}
}

func TestListIssueMetadataSkipsNonJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	issueDir := filepath.Join(tmpDir, "issues")
	if err := os.MkdirAll(issueDir, 0755); err != nil {
		t.Fatalf("failed to create issues dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "plans"), 0755); err != nil {
		t.Fatalf("failed to create plans dir: %v", err)
	}

	cache := &Cache{dir: tmpDir}

	// Add valid metadata
	meta := &IssueMetadata{IssueID: "NUM-1", BranchName: "branch-1"}
	if err := cache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// Add a non-JSON file that should be skipped
	if err := os.WriteFile(filepath.Join(issueDir, "readme.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}

	// Add a subdirectory that should be skipped
	if err := os.MkdirAll(filepath.Join(issueDir, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	metadata, err := cache.ListIssueMetadata()
	if err != nil {
		t.Fatalf("ListIssueMetadata() error = %v", err)
	}
	if len(metadata) != 1 {
		t.Errorf("ListIssueMetadata() = %d items, want 1", len(metadata))
	}
}

func TestDeleteIssueMetadata(t *testing.T) {
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

	// Save metadata
	meta := &IssueMetadata{IssueID: "NUM-1", BranchName: "branch-1"}
	if err := cache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// Verify it exists
	got, err := cache.GetIssueMetadata("NUM-1")
	if err != nil || got == nil {
		t.Fatalf("metadata should exist before delete")
	}

	// Delete it
	if err := cache.DeleteIssueMetadata("NUM-1"); err != nil {
		t.Fatalf("DeleteIssueMetadata() error = %v", err)
	}

	// Verify it's gone
	got, err = cache.GetIssueMetadata("NUM-1")
	if err != nil {
		t.Fatalf("GetIssueMetadata() error = %v", err)
	}
	if got != nil {
		t.Error("metadata should be nil after delete")
	}
}

func TestDeleteIssueMetadataNotExists(t *testing.T) {
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

	// Delete non-existent metadata should not error
	if err := cache.DeleteIssueMetadata("nonexistent"); err != nil {
		t.Errorf("DeleteIssueMetadata() should not error for non-existent: %v", err)
	}
}

func TestDeletePlan(t *testing.T) {
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

	// Save a plan
	p := &plan.Plan{
		ID:         "test-plan",
		Title:      "Test Plan",
		Status:     plan.StatusDraft,
		Author:     "test",
		RawContent: "---\nid: test-plan\ntitle: Test Plan\nstatus: draft\nauthor: test\n---\n# Test",
	}
	if err := cache.SavePlan(p); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Verify files exist
	jsonPath := filepath.Join(tmpDir, "plans", "test-plan.json")
	mdPath := filepath.Join(tmpDir, "plans", "test-plan.md")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("JSON file should exist before delete")
	}
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Fatal("MD file should exist before delete")
	}

	// Delete it
	if err := cache.DeletePlan("test-plan"); err != nil {
		t.Fatalf("DeletePlan() error = %v", err)
	}

	// Verify files are gone
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Error("JSON file should not exist after delete")
	}
	if _, err := os.Stat(mdPath); !os.IsNotExist(err) {
		t.Error("MD file should not exist after delete")
	}
}

func TestListPlans(t *testing.T) {
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

	// Test empty list
	plans, err := cache.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans() error = %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("ListPlans() = %d items, want 0", len(plans))
	}

	// Add some plans
	for i, id := range []string{"plan-1", "plan-2"} {
		p := &plan.Plan{
			ID:         id,
			Title:      "Test Plan " + id,
			Status:     plan.StatusDraft,
			Author:     "test",
			RawContent: "---\nid: " + id + "\ntitle: Test\nstatus: draft\nauthor: test\n---\n# Test " + string(rune('A'+i)),
		}
		if err := cache.SavePlan(p); err != nil {
			t.Fatalf("SavePlan() error = %v", err)
		}
	}

	// Test list with items
	plans, err = cache.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans() error = %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("ListPlans() = %d items, want 2", len(plans))
	}
}

func TestListPlansSkipsNonJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	planDir := filepath.Join(tmpDir, "plans")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("failed to create plans dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "issues"), 0755); err != nil {
		t.Fatalf("failed to create issues dir: %v", err)
	}

	cache := &Cache{dir: tmpDir}

	// Add valid plan
	p := &plan.Plan{
		ID:         "plan-1",
		Title:      "Test Plan",
		Status:     plan.StatusDraft,
		Author:     "test",
		RawContent: "---\nid: plan-1\ntitle: Test\nstatus: draft\nauthor: test\n---\n# Test",
	}
	if err := cache.SavePlan(p); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Add a non-JSON file that should be skipped (note: .md files are created by SavePlan)
	if err := os.WriteFile(filepath.Join(planDir, "readme.txt"), []byte("ignore"), 0644); err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}

	// Add a subdirectory that should be skipped
	if err := os.MkdirAll(filepath.Join(planDir, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	plans, err := cache.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans() error = %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("ListPlans() = %d items, want 1", len(plans))
	}
}

func TestClear(t *testing.T) {
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

	// Add some data
	p := &plan.Plan{
		ID:         "plan-1",
		Title:      "Test Plan",
		Status:     plan.StatusDraft,
		Author:     "test",
		RawContent: "---\nid: plan-1\ntitle: Test\nstatus: draft\nauthor: test\n---\n# Test",
	}
	if err := cache.SavePlan(p); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	meta := &IssueMetadata{IssueID: "NUM-1", BranchName: "branch-1"}
	if err := cache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	// Clear
	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify everything is gone
	plans, _ := cache.ListPlans()
	if len(plans) != 0 {
		t.Errorf("ListPlans() after Clear() = %d, want 0", len(plans))
	}

	metadata, _ := cache.ListIssueMetadata()
	if len(metadata) != 0 {
		t.Errorf("ListIssueMetadata() after Clear() = %d, want 0", len(metadata))
	}

	// Verify directories still exist (recreated by Clear)
	if _, err := os.Stat(filepath.Join(tmpDir, "plans")); os.IsNotExist(err) {
		t.Error("plans directory should exist after Clear()")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "issues")); os.IsNotExist(err) {
		t.Error("issues directory should exist after Clear()")
	}
}

func TestSaveIssueMetadataRequiresID(t *testing.T) {
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

	meta := &IssueMetadata{IssueID: "", BranchName: "branch"}
	err = cache.SaveIssueMetadata(meta)
	if err == nil {
		t.Error("SaveIssueMetadata() should error when IssueID is empty")
	}
	if !strings.Contains(err.Error(), "issue ID is required") {
		t.Errorf("error should mention 'issue ID is required', got: %v", err)
	}
}

func TestGetPlanNotFound(t *testing.T) {
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

	p, err := cache.GetPlan("nonexistent")
	if err != nil {
		t.Fatalf("GetPlan() unexpected error = %v", err)
	}
	if p != nil {
		t.Error("GetPlan() should return nil for non-existent plan")
	}
}

func TestGetIssueMetadataNotFound(t *testing.T) {
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

	meta, err := cache.GetIssueMetadata("nonexistent")
	if err != nil {
		t.Fatalf("GetIssueMetadata() unexpected error = %v", err)
	}
	if meta != nil {
		t.Error("GetIssueMetadata() should return nil for non-existent metadata")
	}
}

func TestNewCacheWithDir(t *testing.T) {
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

	cache := NewCacheWithDir(tmpDir)
	if cache == nil {
		t.Fatal("NewCacheWithDir() returned nil")
	}

	// Verify cache works by saving and retrieving metadata
	meta := &IssueMetadata{IssueID: "TEST-1", BranchName: "test-branch"}
	if err := cache.SaveIssueMetadata(meta); err != nil {
		t.Fatalf("SaveIssueMetadata() error = %v", err)
	}

	got, err := cache.GetIssueMetadata("TEST-1")
	if err != nil {
		t.Fatalf("GetIssueMetadata() error = %v", err)
	}
	if got == nil || got.IssueID != "TEST-1" {
		t.Error("NewCacheWithDir() cache should be functional")
	}
}

func TestSyncPRForIssueGetMetadataError(t *testing.T) {
	// Test the case where GetIssueMetadata returns an error
	tmpDir, err := os.MkdirTemp("", "jig-cache-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Only create plans dir, not issues dir - this will cause issues when reading
	if err := os.MkdirAll(filepath.Join(tmpDir, "plans"), 0755); err != nil {
		t.Fatalf("failed to create plans dir: %v", err)
	}
	issuesDir := filepath.Join(tmpDir, "issues")
	if err := os.MkdirAll(issuesDir, 0755); err != nil {
		t.Fatalf("failed to create issues dir: %v", err)
	}

	cache := &Cache{dir: tmpDir}

	// Write invalid JSON to trigger parse error
	invalidJSON := filepath.Join(issuesDir, "BAD-1.json")
	if err := os.WriteFile(invalidJSON, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}

	_, err = cache.SyncPRForIssue("BAD-1")
	if err == nil {
		t.Error("SyncPRForIssue() should error on invalid JSON")
	}
}
