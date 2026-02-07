---
description: Create a detailed implementation plan for a software engineering task
argument-hint: "[<goal>]"
---

# /jig:plan

Create a comprehensive implementation plan for a software engineering task.

This skill guides you through creating a well-structured plan with:
- Problem statement and proposed solution
- Phased implementation approach
- Acceptance criteria for each phase
- Dependency tracking between phases

## Usage

```bash
/jig:plan <session-id>  # Use context from session-specific directory
```

This skill is typically invoked automatically by `jig plan` with a session ID.

---

## Agent Instructions

**IMPORTANT**: You are the planning agent. Your job is to help the user create a detailed, actionable implementation plan.

### Step 0: Gather Context

Context files are stored in a session-specific directory (`.jig/sessions/<session-id>/`) to support parallel planning sessions without conflicts.

1. **Get the session ID from $ARGUMENTS**:
   - The session ID is passed as the first argument
   - If no session ID provided, use "default"

2. **Read the planning context** from the session directory:
   ```bash
   # $ARGUMENTS contains the session ID
   cat .jig/sessions/$ARGUMENTS/planning-context.md 2>/dev/null || echo "No planning context file"
   ```

3. **Read any issue context** from the session directory:
   ```bash
   cat .jig/sessions/$ARGUMENTS/issue-context.md 2>/dev/null || echo "No issue context"
   ```

If no context is found, ask the user what they want to plan.

### Step 1: Understand the Problem

Before creating a plan:

1. **Analyze the request**: What is the user trying to accomplish?
2. **Ask clarifying questions** if requirements are unclear:
   - What problem does this solve?
   - Who are the users/consumers?
   - What are the constraints?
   - Are there existing patterns to follow?
3. **Explore the codebase** to understand:
   - Existing architecture and patterns
   - Related code that might be affected
   - Testing patterns and conventions

### Step 2: Design the Solution

Think through:

1. **High-level approach**: What's the overall strategy?
2. **Phased breakdown**: How can this be split into independent, deliverable phases?
3. **Dependencies**: Which phases depend on others?
4. **Risks**: What could go wrong? How to mitigate?

### Step 3: Write the Plan

Create a plan document with this structure:

```markdown
---
id: <plan-id>
title: <Plan Title>
status: draft
author: <username>
phases:
  - id: phase-1
    title: <Phase 1 Title>
    status: pending
    depends_on: []
  - id: phase-2
    title: <Phase 2 Title>
    status: pending
    depends_on: [phase-1]
---

# <Plan Title>

## Problem Statement

<Clear description of the problem being solved>

## Proposed Solution

<High-level approach to solving the problem>

## Phases

### <Phase 1 Title>

**Dependencies:** None

#### Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

#### Implementation Details

<Specific details for implementing this phase>

### <Phase 2 Title>

**Dependencies:** Phase 1

#### Acceptance Criteria

- [ ] Criterion 1

#### Implementation Details

<Specific details for implementing this phase>
```

### Step 4: Plan Guidelines

Follow these principles:

1. **Be specific**: Vague plans lead to vague implementations
2. **Keep phases small**: Each phase should be completable in one session
3. **Define clear acceptance criteria**: Testable conditions that define "done"
4. **Consider testing**: Include test requirements in acceptance criteria
5. **Document dependencies**: Make phase ordering explicit
6. **Avoid over-engineering**: Plan what's needed, not what might be needed

### Step 5: Save the Plan

When the plan is complete and the user is satisfied:

1. **Write the plan to a file**:
   ```bash
   # Write the plan content to plan.md
   cat > plan.md << 'EOF'
   <your plan content here>
   EOF
   ```

2. **Save to jig's cache**:
   ```bash
   jig plan save plan.md
   ```

This will validate and cache the plan for implementation with `jig implement <plan-id>`.

---

## Plan Format Reference

### Frontmatter Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier (e.g., `add-user-auth`, `PROJ-123`) |
| `title` | Yes | Human-readable title |
| `status` | Yes | One of: `draft`, `approved`, `in_progress`, `completed` |
| `author` | Yes | Username of the plan author |
| `phases` | Yes | Array of phase definitions |

### Phase Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique phase identifier (e.g., `phase-1`) |
| `title` | Yes | Human-readable phase title |
| `status` | Yes | One of: `pending`, `in_progress`, `complete` |
| `depends_on` | No | Array of phase IDs this phase depends on |

### Body Sections

1. **Problem Statement**: What problem are we solving and why?
2. **Proposed Solution**: High-level approach (not implementation details)
3. **Phases**: Detailed breakdown with acceptance criteria and implementation details

---

## Output Format

- **Start**: "Let me help you create an implementation plan for [goal]..."
- **Questions**: Ask clarifying questions before diving into planning
- **Progress**: Share your thinking as you design the solution
- **Draft**: Present the draft plan for review
- **End**: Save the finalized plan with `jig plan save`

---

## Important Rules

1. **Understand before planning**: Ask questions first, plan second
2. **Explore the codebase**: Don't plan in a vacuum - understand existing patterns
3. **Get user buy-in**: Present drafts and iterate based on feedback
4. **Keep it actionable**: Every phase should be clearly implementable
5. **Save properly**: Always use `jig plan save` to cache the final plan
