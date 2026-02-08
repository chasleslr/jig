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
