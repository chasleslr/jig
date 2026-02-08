---
description: Implement a plan from a jig issue or cached plan
argument-hint: "[<issue-id>]"
---

# /jig:implement

Implement a plan - either from a cached plan, a linked tracker issue, or the current worktree context.

This is the primary implementation workflow - it orchestrates:

1. Reading the plan and setting up implementation context
2. Executing each phase with proper tracking
3. Running tests and verifying changes
4. Guiding through PR creation

## Prerequisites

- Must be in a git worktree set up by `jig implement`
- Plan must exist (in cache or fetched from tracker)
- Project should have tests and CI configured

## Usage

```bash
/jig:implement              # Use plan from current worktree context
/jig:implement NUM-123      # Implement specific issue from tracker
```

---

## Agent Instructions

**IMPORTANT**: You are the implementation agent. Your job is to execute the plan, not modify it. Follow the plan exactly.

### Step 0: Understand Context

You are running in a worktree directory set up by `jig implement`. Check the context:

1. **Check for .jig directory**:
   ```bash
   ls -la .jig/ 2>/dev/null || echo "No .jig directory"
   ```

2. **Read any cached plan info** from the worktree or environment.

3. **Parse $ARGUMENTS**:
   - If provided, use it as the issue ID to load the plan
   - If empty, use the plan already loaded in context

### Step 1: Read and Understand the Plan

The plan is stored in `.jig/plan.md` in the current directory. This was set up by `jig implement`.

**Read the plan file**:
```bash
cat .jig/plan.md
```

Also read the issue metadata:
```bash
cat .jig/issue.json
```

Read the plan carefully to understand:

- **Problem Statement**: What problem are we solving?
- **Proposed Solution**: What's the high-level approach?
- **Phases**: What are the implementation phases?
- **Acceptance Criteria**: What defines success for each phase?
- **Dependencies**: Which phases depend on others?

### Step 2: Create TodoWrite Entries

Create todo entries for the implementation using TodoWrite:

1. One todo per phase from the plan
2. Include a final todo for "Run tests and verify"
3. Include a todo for "Create PR" (if applicable)

Example:
```
- Phase 1: Set up data models
- Phase 2: Implement API endpoints
- Phase 3: Add frontend components
- Run tests and verify all changes
- Create PR
```

### Step 3: Execute Each Phase Sequentially

For each phase:

1. **Mark phase as in_progress** (in TodoWrite)

2. **Read phase requirements** carefully:
   - Acceptance criteria (what must be true when done)
   - Implementation details (how to do it)
   - Dependencies (what must be done first)

3. **Implement the phase**:
   - Write clean, well-tested code
   - Follow project conventions (check AGENTS.md if present)
   - Make atomic commits as you go
   - Ensure tests pass before moving on

4. **Verify acceptance criteria**:
   - Check each criterion is met
   - Run relevant tests
   - Fix any issues before proceeding

5. **Mark phase as completed** (in TodoWrite)

6. **Report progress**:
   - What was implemented
   - What files were changed
   - Any notes or deviations from the plan

### Step 4: Implementation Guidelines

Follow these principles:

1. **Follow the plan exactly**: The plan was carefully designed. Don't deviate unless absolutely necessary.

2. **One phase at a time**: Complete each phase fully before starting the next.

3. **Test as you go**: Run tests after each significant change. Don't accumulate untested changes.

4. **Commit logically**: Make commits that match the logical units of work. Use descriptive commit messages.

5. **Don't over-engineer**: Implement exactly what's specified. Don't add extra features or refactoring.

6. **Ask if unclear**: If something in the plan is ambiguous, ask for clarification rather than guessing.

### Step 5: Run Tests and Verify

After all phases are complete:

1. **Run the full test suite**:
   - Check AGENTS.md or project docs for test commands
   - Common: `make test`, `npm test`, `go test ./...`, `pytest`

2. **Run linting/formatting**:
   - Ensure code passes all linting rules
   - Fix any formatting issues

3. **Verify all acceptance criteria**:
   - Go through each phase's acceptance criteria
   - Confirm all are met

4. **Check for regressions**:
   - Ensure existing functionality still works
   - Review any test failures carefully

### Step 6: Final Summary

When implementation is complete, provide a summary:

1. **What was implemented**: List the main changes
2. **Files changed**: Overview of modified files
3. **Tests**: Confirm tests are passing
4. **Deviations**: Note any deviations from the original plan and why
5. **Next steps**: Usually creating a PR

### Step 7: Guide PR Creation

When implementation is complete, guide the user through PR creation using `/jig:pr`:

1. **Ensure changes are committed**:
   ```bash
   git status --porcelain
   ```

   If there are uncommitted changes:
   ```bash
   git add -A
   git commit -m "feat: implement <feature-name>

   <summary of what was done>

   Implements: <issue-id>"
   ```

2. **Push the branch**:
   ```bash
   git push -u origin HEAD
   ```

3. **Suggest using /jig:pr**:

   Tell the user: "Would you like to create a PR now? Run `/jig:pr` to create a draft PR with the plan info and record the PR in metadata for easy merging later."

   Alternatively, they can run directly:
   ```bash
   jig pr
   ```

---

## Output Format

- **Start**: "Starting implementation for [plan title]..."
- **Each phase**: "Phase X: [title] - [brief status]"
- **Progress**: Show what's being done, files being modified
- **End**: "Implementation complete. Summary: [changes made]"

---

## Important Rules

1. **Never skip phases**: Execute all phases in order
2. **Never modify the plan**: The plan is immutable during implementation
3. **Always run tests**: Don't consider a phase complete until tests pass
4. **Commit frequently**: Make atomic commits for each logical unit
5. **Report blockers**: If blocked, report clearly and wait for guidance
