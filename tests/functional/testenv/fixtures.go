package testenv

import "fmt"

// ValidPlan generates a valid plan markdown with all required sections.
func ValidPlan(id, title string) string {
	return fmt.Sprintf(`---
id: %s
title: %s
status: draft
created: "2024-01-01T00:00:00Z"
author: test-user
reviewers: {}
---

# %s

## Problem Statement

This is a test problem statement for the plan.

## Proposed Solution

This is a test proposed solution.

## Acceptance Criteria

- [ ] First acceptance criterion
- [ ] Second acceptance criterion

## Implementation Details

### Phase 1: Setup

Create the necessary infrastructure.

### Phase 2: Implementation

Implement the core functionality.

## Files to Modify/Create

| File | Action |
|------|--------|
| src/main.go | Modify |
| src/utils.go | Create |

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Test risk | Test mitigation |
`, id, title, title)
}

// MinimalPlan generates a minimal valid plan with required fields only.
func MinimalPlan(id string) string {
	return fmt.Sprintf(`---
id: %s
title: Minimal Plan
status: draft
created: "2024-01-01T00:00:00Z"
author: test-user
reviewers: {}
---

# Minimal Plan

## Problem Statement

Minimal problem.

## Proposed Solution

Minimal solution.

## Acceptance Criteria

- [ ] It works
`, id)
}

// InvalidPlan generates an invalid plan (missing required fields).
func InvalidPlan() string {
	return `# Not a valid plan

This is just some random markdown without proper frontmatter.
`
}

// PlanWithCustomStatus generates a plan with a specific status.
func PlanWithCustomStatus(id, title, status string) string {
	return fmt.Sprintf(`---
id: %s
title: %s
status: %s
created: "2024-01-01T00:00:00Z"
author: test-user
reviewers: {}
---

# %s

## Problem Statement

Test problem.

## Proposed Solution

Test solution.

## Acceptance Criteria

- [ ] It works
`, id, title, status, title)
}
