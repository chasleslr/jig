---
description: Create a detailed implementation plan for a software engineering task
argument-hint: "[<goal>]"
---

# /jig:plan

Create a comprehensive implementation plan for a software engineering task.

This skill guides you through creating a well-structured plan with:
- Problem statement and proposed solution
- Clear acceptance criteria
- Implementation details

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

2. **Categorize your understanding** of each aspect:
   - **Clear**: You understand exactly what's needed and how to implement it
   - **Ambiguous**: Multiple interpretations exist, or details are underspecified
   - **Unknown**: You don't have enough information to proceed

3. **Explore the codebase** to understand:
   - Existing architecture and patterns
   - Related code that might be affected
   - Testing patterns and conventions

#### 1.1 Ask Clarifying Questions (REQUIRED)

**DO NOT proceed to solution design if ANY aspect is Ambiguous or Unknown.**

For each ambiguous or unknown aspect, ask the user using this format:

```
**Question**: [Specific question about the unclear aspect]
**Why this matters**: [Brief explanation of how the answer affects the plan]
**Options** (if applicable):
- Option A: [description]
- Option B: [description]
```

Wait for user responses before continuing to Step 2.

### Step 2: Design the Solution

#### 2.1 Assess Feasibility First (REQUIRED)

Before designing, evaluate feasibility using this checklist:

- [ ] **Technical feasibility**: Can this be built with current technologies and architecture?
- [ ] **Scope clarity**: Is the scope well-defined enough to estimate effort?
- [ ] **Dependencies**: Are all required dependencies available and compatible?
- [ ] **Constraints**: Are there time, performance, or resource constraints that affect approach?
- [ ] **Risks**: Are there significant unknowns or risks that could block implementation?

**If ANY concern is found**: Stop and report to the user before proceeding.

```
**Feasibility Concern**: [What the concern is]
**Impact**: [How this affects the plan]
**Recommendation**: [What you suggest - pause, investigate, adjust scope, etc.]
```

#### 2.2 Present Alternatives When Tradeoffs Exist

When multiple valid approaches exist, present them to the user:

```
**Decision Needed**: [What needs to be decided]

| Approach | Pros | Cons | Best If |
|----------|------|------|---------|
| Option A | [benefits] | [drawbacks] | [when to choose] |
| Option B | [benefits] | [drawbacks] | [when to choose] |

**My recommendation**: [Which option and why]
```

Wait for user input before proceeding with the chosen approach.

#### 2.3 Design with User Input

After user decisions on approach, think through:

1. **High-level approach**: What's the overall strategy based on chosen direction?
2. **Acceptance criteria**: What conditions must be met for success?
3. **Implementation details**: What specific changes are needed?
4. **Risks**: What could go wrong? How to mitigate?

---

### Examples: Clarification and Feasibility

#### Good Example: Asking Clarifying Questions

**User request**: "Add caching to the API"

**Good response**:
```
I've analyzed the request and have some questions before I can create a plan:

**Question**: Which endpoints need caching?
**Why this matters**: Caching strategy differs significantly between read-heavy endpoints (where aggressive caching helps) vs write-heavy endpoints (where cache invalidation becomes complex).
**Options**:
- All GET endpoints
- Only specific high-traffic endpoints (please specify)
- Only endpoints with expensive database queries

**Question**: What cache backend should we use?
**Why this matters**: This affects infrastructure requirements and implementation approach.
**Options**:
- Redis (recommended for distributed systems)
- In-memory (simpler but doesn't share across instances)
- Existing infrastructure (please specify what's available)

**Question**: What are the cache invalidation requirements?
**Why this matters**: Stale data tolerance affects complexity significantly.
**Options**:
- Time-based expiration only (simplest)
- Invalidate on writes (moderate complexity)
- Real-time consistency required (most complex)
```

#### Bad Example: Making Assumptions

**User request**: "Add caching to the API"

**Bad response**:
```
I'll create a plan to add Redis caching to all GET endpoints with a 5-minute TTL
and automatic invalidation on writes.

## Problem Statement
Add caching to improve API performance...
```

*This is bad because*: The agent assumed Redis, assumed all GET endpoints, assumed 5-minute TTL, and assumed write-based invalidation without asking. These assumptions could lead to over-engineering or wrong solutions.

---

#### Good Example: Reporting Feasibility Concerns

**User request**: "Add real-time collaboration to the document editor"

**Good response**:
```
**Feasibility Concern**: The current architecture uses REST APIs with polling.
Real-time collaboration typically requires WebSocket infrastructure.

**Impact**: This would require:
- Adding WebSocket server support
- Implementing operational transformation (OT) or CRDT algorithms
- Significant changes to the document data model
- New infrastructure for connection management

**Recommendation**: Before proceeding with planning, we should:
1. Confirm this scope is acceptable (this is a significant undertaking)
2. Decide on the synchronization approach (OT vs CRDT)
3. Evaluate if existing real-time libraries could be used

Would you like to proceed with a plan of this scope, or should we explore a simpler alternative like periodic auto-save with conflict detection?
```

#### Bad Example: Ignoring Feasibility Concerns

**User request**: "Add real-time collaboration to the document editor"

**Bad response**:
```
## Problem Statement
Add real-time collaboration so multiple users can edit simultaneously.

## Proposed Solution
Implement WebSocket connections and operational transformation...

## Implementation Details
1. Add Socket.io server
2. Implement OT algorithm
3. Update document model...
```

*This is bad because*: The agent jumped straight to planning without flagging that this is a major architectural change requiring significant scope discussion.

---

### Step 3: Write the Plan

Create a plan document with this structure:

```markdown
---
id: <plan-id>
title: <Plan Title>
status: draft
author: <username>
reviewers:
  default: [lead]
---

# <Plan Title>

## Problem Statement

<Clear description of the problem being solved>

## Proposed Solution

<High-level approach to solving the problem>

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2
- [ ] Criterion 3

## Implementation Details

<Specific details for implementing this plan>
```

### Step 4: Plan Guidelines

Follow these principles:

1. **Be specific**: Vague plans lead to vague implementations
2. **Keep it focused**: Each plan should address a single coherent goal
3. **Define clear acceptance criteria**: Testable conditions that define "done"
4. **Consider testing**: Include test requirements in acceptance criteria
5. **Avoid over-engineering**: Plan what's needed, not what might be needed

### Step 5: Save the Plan

When the plan is complete and the user is satisfied:

1. **Write the plan to a file**:
   ```bash
   # Write the plan content to plan.md
   cat > plan.md << 'EOF'
   <your plan content here>
   EOF
   ```

2. **Save to jig's cache** (include the session ID so jig can track which plan was saved):
   ```bash
   jig plan save --session $ARGUMENTS plan.md
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
| `reviewers` | No | Map of reviewer types to usernames |

### Body Sections

1. **Problem Statement**: What problem are we solving and why?
2. **Proposed Solution**: High-level approach (not implementation details)
3. **Acceptance Criteria**: Testable conditions that define success
4. **Implementation Details**: Specific changes needed to implement the plan

---

## Output Format

- **Start**: "Let me help you create an implementation plan for [goal]..."
- **Questions**: Ask clarifying questions before diving into planning
- **Progress**: Share your thinking as you design the solution
- **Draft**: Present the draft plan for review
- **End**: Save the finalized plan with `jig plan save --session $ARGUMENTS`

---

## Important Rules

1. **Understand before planning**: Ask questions first, plan second
2. **Explore the codebase**: Don't plan in a vacuum - understand existing patterns
3. **Get user buy-in**: Present drafts and iterate based on feedback
4. **Keep it actionable**: Every part of the plan should be clearly implementable
5. **Save properly**: Always use `jig plan save --session $ARGUMENTS` to cache the final plan
