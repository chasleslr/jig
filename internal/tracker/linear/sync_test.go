package linear

import (
	"strings"
	"testing"

	"github.com/charleslr/jig/internal/plan"
)

func TestFormatPlanComment(t *testing.T) {
	t.Run("includes all plan sections", func(t *testing.T) {
		p := &plan.Plan{
			ID:               "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "This is the problem we're solving.",
			ProposedSolution: "This is how we'll solve it.",
		}

		result := formatPlanComment(p)

		// Check header
		if !strings.Contains(result, "## 📋 Implementation Plan") {
			t.Error("expected header '## 📋 Implementation Plan'")
		}

		// Check synced timestamp
		if !strings.Contains(result, "**Synced:**") {
			t.Error("expected synced timestamp")
		}

		// Check problem statement
		if !strings.Contains(result, "### Problem Statement") {
			t.Error("expected '### Problem Statement' section")
		}
		if !strings.Contains(result, "This is the problem we're solving.") {
			t.Error("expected problem statement content")
		}

		// Check proposed solution
		if !strings.Contains(result, "### Proposed Solution") {
			t.Error("expected '### Proposed Solution' section")
		}
		if !strings.Contains(result, "This is how we'll solve it.") {
			t.Error("expected proposed solution content")
		}

		// Check footer
		if !strings.Contains(result, "*This plan was synced by [jig]") {
			t.Error("expected jig attribution footer")
		}
	})

	t.Run("handles empty problem statement", func(t *testing.T) {
		p := &plan.Plan{
			ID:               "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "",
			ProposedSolution: "This is how we'll solve it.",
		}

		result := formatPlanComment(p)

		// Should not contain problem statement section when empty
		if strings.Contains(result, "### Problem Statement") {
			t.Error("should not include empty problem statement section")
		}

		// Should still contain proposed solution
		if !strings.Contains(result, "### Proposed Solution") {
			t.Error("expected '### Proposed Solution' section")
		}
	})

	t.Run("handles empty proposed solution", func(t *testing.T) {
		p := &plan.Plan{
			ID:               "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "This is the problem.",
			ProposedSolution: "",
		}

		result := formatPlanComment(p)

		// Should contain problem statement
		if !strings.Contains(result, "### Problem Statement") {
			t.Error("expected '### Problem Statement' section")
		}

		// Should not contain proposed solution section when empty
		if strings.Contains(result, "### Proposed Solution") {
			t.Error("should not include empty proposed solution section")
		}
	})

	t.Run("includes additional sections from raw content", func(t *testing.T) {
		p := &plan.Plan{
			ID:               "NUM-41",
			Title:            "Test Plan",
			ProblemStatement: "Problem",
			ProposedSolution: "Solution",
			RawContent: `## Problem Statement
Problem

## Proposed Solution
Solution

## Acceptance Criteria
- Criterion 1
- Criterion 2

## Implementation Details
Step by step details here.
`,
		}

		result := formatPlanComment(p)

		// Should include acceptance criteria
		if !strings.Contains(result, "## Acceptance Criteria") {
			t.Error("expected '## Acceptance Criteria' section from raw content")
		}
		if !strings.Contains(result, "Criterion 1") {
			t.Error("expected acceptance criteria content")
		}

		// Should include implementation details
		if !strings.Contains(result, "## Implementation Details") {
			t.Error("expected '## Implementation Details' section from raw content")
		}
	})
}

func TestExtractAdditionalSections(t *testing.T) {
	t.Run("extracts sections other than problem and solution", func(t *testing.T) {
		rawContent := `## Problem Statement
Problem content here.

## Proposed Solution
Solution content here.

## Acceptance Criteria
- AC 1
- AC 2

## Testing Strategy
Unit tests and integration tests.
`
		result := extractAdditionalSections(rawContent)

		// Should not include problem statement
		if strings.Contains(result, "Problem Statement") {
			t.Error("should not include Problem Statement section")
		}
		if strings.Contains(result, "Problem content here") {
			t.Error("should not include problem statement content")
		}

		// Should not include proposed solution
		if strings.Contains(result, "Proposed Solution") {
			t.Error("should not include Proposed Solution section")
		}
		if strings.Contains(result, "Solution content here") {
			t.Error("should not include proposed solution content")
		}

		// Should include acceptance criteria
		if !strings.Contains(result, "## Acceptance Criteria") {
			t.Error("expected Acceptance Criteria section")
		}
		if !strings.Contains(result, "AC 1") {
			t.Error("expected acceptance criteria content")
		}

		// Should include testing strategy
		if !strings.Contains(result, "## Testing Strategy") {
			t.Error("expected Testing Strategy section")
		}
		if !strings.Contains(result, "Unit tests and integration tests") {
			t.Error("expected testing strategy content")
		}
	})

	t.Run("handles triple hash headers", func(t *testing.T) {
		rawContent := `### Problem Statement
Problem content.

### Proposed Solution
Solution content.

### Notes
Additional notes here.
`
		result := extractAdditionalSections(rawContent)

		// Should extract notes section
		if !strings.Contains(result, "### Notes") {
			t.Error("expected Notes section")
		}
		if !strings.Contains(result, "Additional notes here") {
			t.Error("expected notes content")
		}

		// Should not include skipped sections
		if strings.Contains(result, "Problem Statement") {
			t.Error("should not include Problem Statement")
		}
		if strings.Contains(result, "Proposed Solution") {
			t.Error("should not include Proposed Solution")
		}
	})

	t.Run("handles case insensitive matching", func(t *testing.T) {
		rawContent := `## PROBLEM STATEMENT
Problem.

## proposed solution
Solution.

## Other Section
Content.
`
		result := extractAdditionalSections(rawContent)

		// Should extract other section
		if !strings.Contains(result, "## Other Section") {
			t.Error("expected Other Section")
		}

		// Should not include uppercase problem statement
		if strings.Contains(result, "PROBLEM STATEMENT") {
			t.Error("should not include PROBLEM STATEMENT")
		}
	})

	t.Run("returns empty for no additional sections", func(t *testing.T) {
		rawContent := `## Problem Statement
Just problem.

## Proposed Solution
Just solution.
`
		result := extractAdditionalSections(rawContent)

		if result != "" {
			t.Errorf("expected empty result, got %q", result)
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		result := extractAdditionalSections("")

		if result != "" {
			t.Errorf("expected empty result for empty input, got %q", result)
		}
	})

	t.Run("preserves content between sections", func(t *testing.T) {
		rawContent := `## Problem Statement
Problem.

## Implementation
Step 1
Step 2
Step 3

More details here.

## Testing
Test approach.
`
		result := extractAdditionalSections(rawContent)

		// Should include all implementation content
		if !strings.Contains(result, "Step 1") {
			t.Error("expected Step 1")
		}
		if !strings.Contains(result, "Step 2") {
			t.Error("expected Step 2")
		}
		if !strings.Contains(result, "More details here") {
			t.Error("expected 'More details here'")
		}

		// Should include testing section
		if !strings.Contains(result, "## Testing") {
			t.Error("expected Testing section")
		}
	})
}

func TestGetIssueLabelIDs(t *testing.T) {
	// This function is tested indirectly through SyncPlanToIssue tests
	// but we can add a direct test for edge cases
	t.Run("returns empty slice on error", func(t *testing.T) {
		// Test with invalid context/client that will fail
		// The function handles errors gracefully and returns nil
		client := NewClient("invalid", "", "")
		result := getIssueLabelIDs(nil, client, "invalid-id")
		if result != nil {
			t.Errorf("expected nil on error, got %v", result)
		}
	})
}
