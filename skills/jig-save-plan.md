# Save Plan to Jig

Use this skill to save an implementation plan to jig after completing a planning session.

## When to Use

- After creating a detailed implementation plan during a `jig new` session
- When the plan is complete and ready to be saved
- Automatically at the end of a planning session

## Plan Format

The plan should be a markdown document with YAML frontmatter:

```markdown
---
id: PLAN-123
title: Your Plan Title
status: draft
author: username
phases:
  - id: phase-1
    title: Phase 1 Title
    status: pending
    depends_on: []
  - id: phase-2
    title: Phase 2 Title
    status: pending
    depends_on: [phase-1]
---

# Your Plan Title

## Problem Statement

[Description of the problem being solved]

## Proposed Solution

[High-level approach]

## Phases

### Phase 1: Phase 1 Title

**Dependencies:** None

#### Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

#### Implementation Details

[Details for this phase]

### Phase 2: Phase 2 Title

**Dependencies:** Phase 1

#### Acceptance Criteria

- [ ] Criterion 1

#### Implementation Details

[Details for this phase]
```

## How to Save

1. Write the plan to a file (e.g., `plan.md`)
2. Run `jig plan save plan.md`

**Development mode:** If jig is not installed globally, use the local binary:
```bash
./bin/jig plan save plan.md
```

Or pipe directly:

```bash
cat << 'EOF' | ./bin/jig plan save
---
id: PLAN-123
title: My Plan
status: draft
...
---
# My Plan
...
EOF
```

## After Saving

The plan will be cached and you can:
- View it: `jig plan show PLAN-123`
- Start implementation: `jig implement PLAN-123`
