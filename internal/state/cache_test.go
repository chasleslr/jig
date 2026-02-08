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
