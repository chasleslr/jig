---
description: Implement a plan from a jig issue or cached plan
argument-hint: "[<issue-id>]"
---

# /jig:implement

Implement a plan - either from a cached plan, a linked tracker issue, or the current worktree context.

This is the primary implementation workflow - it orchestrates:

1. Reading the plan and setting up implementation context
2. Executing the implementation with proper tracking
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
- **Acceptance Criteria**: What defines success?
- **Implementation Details**: What specific changes are needed?

### Step 2: Create TodoWrite Entries

Create todo entries for the implementation using TodoWrite:

1. Break down the implementation details into logical tasks
2. Include a todo for each acceptance criterion to verify
3. Include a final todo for "Run tests and verify"
4. Include a todo for "Create PR" (if applicable)

Example:
```
- Implement data models
- Add API endpoints
- Update frontend components
- Verify all acceptance criteria
- Run tests and verify all changes
- Create PR
```

### Step 3: Execute Implementation

For each task:

1. **Mark task as in_progress** (in TodoWrite)

2. **Read task requirements** carefully:
   - What must be true when done
   - Implementation details from the plan

3. **Implement the task**:
   - Write clean, well-tested code
   - Follow project conventions (check AGENTS.md if present)
   - Make atomic commits as you go
   - Ensure tests pass before moving on

4. **Verify the work**:
   - Check that the task is complete
   - Run relevant tests
   - Fix any issues before proceeding

5. **Mark task as completed** (in TodoWrite)

6. **Report progress**:
   - What was implemented
   - What files were changed
   - Any notes or deviations from the plan

### Step 4: Implementation Guidelines

Follow these principles:

1. **Follow the plan exactly**: The plan was carefully designed. Don't deviate unless absolutely necessary.

2. **One task at a time**: Complete each task fully before starting the next.

3. **Test as you go**: Run tests after each significant change. Don't accumulate untested changes.

4. **Commit logically**: Make commits that match the logical units of work. Use descriptive commit messages.

5. **Don't over-engineer**: Implement exactly what's specified. Don't add extra features or refactoring.

6. **Ask if unclear**: If something in the plan is ambiguous, ask for clarification rather than guessing.

### Step 5: Run Tests and Verify

After all tasks are complete:

1. **Run the full test suite**:
   - Check AGENTS.md or project docs for test commands
   - Common: `make test`, `npm test`, `go test ./...`, `pytest`

2. **Run linting/formatting**:
   - Ensure code passes all linting rules
   - Fix any formatting issues

3. **Verify all acceptance criteria**:
   - Go through each acceptance criterion from the plan
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

Suggest the user create a PR:

```bash
# Stage and commit any remaining changes
git add -A
git commit -m "feat: implement <feature-name>

<summary of what was done>

Implements: <issue-id>"

# Push the branch
git push -u origin HEAD

# Create the PR
gh pr create --draft --title "<title>" --body "<body>"
```

---

## Output Format

- **Start**: "Starting implementation for [plan title]..."
- **Progress**: Show what's being done, files being modified
- **End**: "Implementation complete. Summary: [changes made]"

---

## Important Rules

1. **Never modify the plan**: The plan is immutable during implementation
2. **Always run tests**: Don't consider a task complete until tests pass
3. **Commit frequently**: Make atomic commits for each logical unit
4. **Report blockers**: If blocked, report clearly and wait for guidance
