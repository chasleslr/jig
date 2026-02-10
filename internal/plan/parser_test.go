package plan

import (
	"strings"
	"testing"
)

func TestValidateStructure(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid plan",
			content: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

This is the problem.

## Proposed Solution

This is the solution.
`,
			wantErr: false,
		},
		{
			name: "valid plan with extra content",
			content: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

This is the problem.

## Proposed Solution

This is the solution.

## Extra Section

This extra content should be allowed.

## Another Custom Section

More custom content here.
`,
			wantErr: false,
		},
		{
			name: "missing id in frontmatter",
			content: `---
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.
`,
			wantErr:     true,
			errContains: "missing required frontmatter fields: id",
		},
		{
			name: "missing title in frontmatter",
			content: `---
id: test-plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.
`,
			wantErr:     true,
			errContains: "missing required frontmatter fields: title",
		},
		{
			name: "missing status in frontmatter",
			content: `---
id: test-plan
title: Test Plan
author: testuser
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.
`,
			wantErr:     true,
			errContains: "missing required frontmatter fields: status",
		},
		{
			name: "missing author in frontmatter",
			content: `---
id: test-plan
title: Test Plan
status: draft
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.
`,
			wantErr:     true,
			errContains: "missing required frontmatter fields: author",
		},
		{
			name: "missing multiple frontmatter fields",
			content: `---
title: Test Plan
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.
`,
			wantErr:     true,
			errContains: "id",
		},
		{
			name: "missing Problem Statement section",
			content: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Proposed Solution

Solution.
`,
			wantErr:     true,
			errContains: "missing required sections: Problem Statement",
		},
		{
			name: "missing Proposed Solution section",
			content: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem.
`,
			wantErr:     true,
			errContains: "missing required sections: Proposed Solution",
		},
		{
			name: "missing multiple sections",
			content: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Some Other Section

Content.
`,
			wantErr:     true,
			errContains: "Problem Statement",
		},
		{
			name: "alternative section naming - problem",
			content: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## The Problem

This uses alternative naming.

## Proposed Solution

Solution.
`,
			wantErr: false,
		},
		{
			name: "alternative section naming - solution",
			content: `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem.

## Solution

This uses alternative naming for solution.
`,
			wantErr: false,
		},
		{
			name: "invalid frontmatter yaml",
			content: `---
id: test-plan
title: Test Plan
status: [invalid yaml
---

# Test Plan
`,
			wantErr:     true,
			errContains: "failed to parse frontmatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStructure([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStructure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateStructure() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestParse(t *testing.T) {
	content := `---
id: test-plan
title: Test Plan Title
status: draft
author: testuser
---

# Test Plan Title

## Problem Statement

This is the problem statement.

## Proposed Solution

This is the proposed solution.
`

	plan, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if plan.ID != "test-plan" {
		t.Errorf("expected ID 'test-plan', got '%s'", plan.ID)
	}
	if plan.Title != "Test Plan Title" {
		t.Errorf("expected Title 'Test Plan Title', got '%s'", plan.Title)
	}
	if plan.Status != StatusDraft {
		t.Errorf("expected Status 'draft', got '%s'", plan.Status)
	}
	if plan.Author != "testuser" {
		t.Errorf("expected Author 'testuser', got '%s'", plan.Author)
	}
	if plan.RawContent != content {
		t.Error("expected RawContent to match original content")
	}
	if !strings.Contains(plan.ProblemStatement, "problem statement") {
		t.Errorf("expected ProblemStatement to contain 'problem statement', got '%s'", plan.ProblemStatement)
	}
	if !strings.Contains(plan.ProposedSolution, "proposed solution") {
		t.Errorf("expected ProposedSolution to contain 'proposed solution', got '%s'", plan.ProposedSolution)
	}
}

func TestParsePreservesRawContent(t *testing.T) {
	content := `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.

## Custom Section

This custom content should be preserved in RawContent.

### Nested Custom Section

Even nested content.
`

	plan, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if plan.RawContent != content {
		t.Error("RawContent should exactly match the original input")
	}

	// Verify the custom content is in RawContent
	if !strings.Contains(plan.RawContent, "Custom Section") {
		t.Error("RawContent should contain 'Custom Section'")
	}
	if !strings.Contains(plan.RawContent, "This custom content should be preserved") {
		t.Error("RawContent should contain the custom content text")
	}
}

func TestSerializePreservesExtraSections(t *testing.T) {
	content := `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem description here.

## Proposed Solution

Solution description here.

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Implementation Details

### File: internal/foo/bar.go

` + "```go" + `
func Example() {
    // code here
}
` + "```" + `

## Verification

1. Build: go build ./...
2. Test: go test ./...
`

	// Parse the plan
	plan, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Serialize it back
	serialized, err := Serialize(plan)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify extra sections are preserved
	if !strings.Contains(string(serialized), "## Acceptance Criteria") {
		t.Error("serialized content should contain 'Acceptance Criteria' section")
	}
	if !strings.Contains(string(serialized), "## Implementation Details") {
		t.Error("serialized content should contain 'Implementation Details' section")
	}
	if !strings.Contains(string(serialized), "## Verification") {
		t.Error("serialized content should contain 'Verification' section")
	}
	if !strings.Contains(string(serialized), "func Example()") {
		t.Error("serialized content should contain code example")
	}
}

func TestSerializeUpdatesStatusInFrontmatter(t *testing.T) {
	content := `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.

## Extra Content

This should be preserved.
`

	// Parse the plan
	plan, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Change the status
	plan.Status = StatusInProgress

	// Serialize it back
	serialized, err := Serialize(plan)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify status is updated in frontmatter
	if !strings.Contains(string(serialized), "status: in-progress") {
		t.Error("serialized content should have updated status in frontmatter")
	}

	// Verify extra content is still preserved
	if !strings.Contains(string(serialized), "## Extra Content") {
		t.Error("serialized content should preserve extra sections")
	}
	if !strings.Contains(string(serialized), "This should be preserved") {
		t.Error("serialized content should preserve extra content text")
	}
}

func TestParseWithIssueID(t *testing.T) {
	content := `---
id: test-plan
issue_id: NUM-42
title: Test Plan With Issue
status: draft
author: testuser
---

# Test Plan With Issue

## Problem Statement

This is the problem statement.

## Proposed Solution

This is the proposed solution.
`

	plan, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if plan.IssueID != "NUM-42" {
		t.Errorf("expected IssueID 'NUM-42', got '%s'", plan.IssueID)
	}
	if plan.ID != "test-plan" {
		t.Errorf("expected ID 'test-plan', got '%s'", plan.ID)
	}
}

func TestParseWithoutIssueID(t *testing.T) {
	content := `---
id: test-plan
title: Test Plan Without Issue
status: draft
author: testuser
---

# Test Plan Without Issue

## Problem Statement

This is the problem statement.

## Proposed Solution

This is the proposed solution.
`

	plan, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if plan.IssueID != "" {
		t.Errorf("expected empty IssueID, got '%s'", plan.IssueID)
	}
}

func TestSerializeWithIssueID(t *testing.T) {
	content := `---
id: test-plan
issue_id: NUM-123
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.
`

	// Parse the plan
	plan, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify IssueID was parsed
	if plan.IssueID != "NUM-123" {
		t.Errorf("expected IssueID 'NUM-123', got '%s'", plan.IssueID)
	}

	// Serialize it back
	serialized, err := Serialize(plan)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify IssueID is in the serialized output
	if !strings.Contains(string(serialized), "issue_id: NUM-123") {
		t.Error("serialized content should contain 'issue_id: NUM-123'")
	}
}

func TestSerializePreservesIssueIDWhenModified(t *testing.T) {
	// Start with a plan without IssueID
	content := `---
id: test-plan
title: Test Plan
status: draft
author: testuser
---

# Test Plan

## Problem Statement

Problem.

## Proposed Solution

Solution.
`

	// Parse the plan
	plan, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Set the IssueID (simulating auto-creation)
	plan.IssueID = "NUM-999"

	// Serialize it back
	serialized, err := Serialize(plan)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify the new IssueID is in the serialized output
	if !strings.Contains(string(serialized), "issue_id: NUM-999") {
		t.Error("serialized content should contain 'issue_id: NUM-999' after modification")
	}
}

func TestExtractBodyFromRawContent(t *testing.T) {
	tests := []struct {
		name        string
		rawContent  string
		wantBody    string
		wantErr     bool
	}{
		{
			name: "normal frontmatter",
			rawContent: `---
id: test
title: Test
---

# Body content here`,
			wantBody: "\n\n# Body content here",
			wantErr:  false,
		},
		{
			name:       "no frontmatter",
			rawContent: "# Just markdown",
			wantErr:    true,
		},
		{
			name: "unclosed frontmatter",
			rawContent: `---
id: test
title: Test
# No closing delimiter`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractBodyFromRawContent(tt.rawContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractBodyFromRawContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantBody {
				t.Errorf("extractBodyFromRawContent() = %q, want %q", got, tt.wantBody)
			}
		})
	}
}
