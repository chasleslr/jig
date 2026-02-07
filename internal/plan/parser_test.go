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
