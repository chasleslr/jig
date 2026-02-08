---
description: Verify implementation against plan acceptance criteria
argument-hint: "[<issue-id>]"
---

# /jig:verify

Verify that an implementation meets all acceptance criteria defined in the plan.

This skill uses **sub-agents** to ensure objective verification with isolated context - the verification agents don't inherit any context from an ongoing implementation session.

## Verification Pipeline

1. **Orchestrator (you)**: Load plan, extract criteria, coordinate sub-agents, generate report
2. **Criteria Verification Agent**: Inspect code to verify each acceptance criterion
3. **Test Runner Agent**: Execute test suite and report results
4. *(Future: Security Agent, Performance Agent, etc.)*

## Prerequisites

- Must be in a git worktree with `.jig/plan.md`
- Implementation should be complete (or near complete)

## Usage

```bash
/jig:verify              # Verify current worktree implementation
/jig:verify NUM-123      # Verify specific issue
```

---

## Agent Instructions

**IMPORTANT**: You are the verification orchestrator. Your job is to coordinate sub-agents that perform objective verification. Sub-agents run with isolated context, ensuring unbiased assessment.

### Step 0: Load Context

1. **Check for .jig directory**:
   ```bash
   ls -la .jig/ 2>/dev/null || echo "No .jig directory"
   ```

2. **Read the plan file**:
   ```bash
   cat .jig/plan.md
   ```

3. **Read issue metadata**:
   ```bash
   cat .jig/issue.json
   ```

4. **Parse $ARGUMENTS**:
   - If provided, use it as the issue ID (for reference)
   - If empty, use context from `.jig/`

### Step 1: Extract Acceptance Criteria

Parse the acceptance criteria from the plan. They may appear in various formats:

**Checkbox format**:
```markdown
- [ ] Criterion one
- [x] Criterion two (already done)
```

**Bullet format**:
```markdown
- Criterion one
- Criterion two
```

**Numbered format**:
```markdown
1. Criterion one
2. Criterion two
```

Create a numbered list of all acceptance criteria to verify.

### Step 2: Create Verification Checklist

Use TodoWrite to track the verification pipeline:

1. "Spawn criteria verification agent"
2. "Spawn test runner agent"
3. "Collect and analyze results"
4. "Generate verification report"

### Step 3: Spawn Criteria Verification Agent

Use the **Task tool** to spawn a sub-agent for criteria verification. This ensures the verification happens with fresh context, free from any implementation bias.

**Task parameters**:
- `subagent_type`: `"Explore"` (uses code exploration capabilities)
- `description`: `"Verify acceptance criteria"`
- `prompt`: See template below

**Prompt template for criteria verification agent**:

```
You are a code verification agent. Your task is to objectively verify whether an implementation meets the specified acceptance criteria.

## Plan Context
Title: [plan title]
Issue: [issue ID]

## Acceptance Criteria to Verify

[List each criterion with a number]
1. [criterion 1]
2. [criterion 2]
...

## Instructions

For EACH acceptance criterion:

1. Search for relevant files and code that would satisfy the criterion
2. Read and analyze the implementation
3. Determine the status:
   - **PASS**: Criterion is fully met
   - **PARTIAL**: Criterion is partially met (explain what's missing)
   - **FAIL**: Criterion is not met
   - **UNTESTABLE**: Cannot verify (explain why)
4. Record evidence: file paths, line numbers, brief code references

## Output Format

Return your findings as a structured list:

### Criterion 1: [criterion text]
**Status**: [PASS/PARTIAL/FAIL/UNTESTABLE]
**Evidence**: [what you found]
**Files checked**: [list of files]

### Criterion 2: [criterion text]
...

### Summary
- Total criteria: X
- Passed: X
- Partial: X
- Failed: X
- Untestable: X
```

**Important**: Pass ONLY the criteria and plan context to the sub-agent. Do NOT include implementation history or prior conversation context.

### Step 4: Spawn Test Runner Agent

Use the **Task tool** to spawn a sub-agent for running tests.

**Task parameters**:
- `subagent_type`: `"Bash"` (command execution specialist)
- `description`: `"Run project tests"`
- `prompt`: See template below

**Prompt template for test runner agent**:

```
You are a test execution agent. Run the project's test suite and report results.

## Instructions

1. Check for test configuration:
   - Look for AGENTS.md, CLAUDE.md, Makefile, package.json, etc.
   - Identify the test command (e.g., `go test ./...`, `npm test`, `pytest`, `make test`)

2. Run the test suite

3. Report results in this format:

### Test Results
- **Command**: [command that was run]
- **Exit code**: [0 for success, non-zero for failure]
- **Total tests**: [count]
- **Passed**: [count]
- **Failed**: [count]
- **Skipped**: [count]

### Failures (if any)
[List each failing test with brief error message]

### Notes
[Any relevant observations about test coverage or issues]
```

### Step 5: Collect and Analyze Results

After sub-agents complete:

1. **Collect criteria verification results** from the Explore agent
2. **Collect test results** from the Bash agent
3. **Determine overall status**:
   - **PASS**: All criteria passed AND tests pass
   - **PARTIAL**: Some criteria partial/failed OR some tests failed
   - **FAIL**: Critical criteria failed OR test suite fails completely

### Step 6: Generate Verification Report

Create a structured report combining all sub-agent findings:

```markdown
# Verification Report: [Issue ID]

## Summary
- **Plan**: [Plan title]
- **Date**: [Current date]
- **Overall Status**: [PASS/PARTIAL/FAIL]

## Acceptance Criteria Results

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | [criterion] | PASS/FAIL/PARTIAL | [brief evidence] |
| 2 | [criterion] | PASS/FAIL/PARTIAL | [brief evidence] |
...

## Test Results
- **Command**: [test command]
- **Total**: X tests
- **Passed**: X
- **Failed**: X
- **Skipped**: X

[If failures, list them briefly]

## Detailed Findings

[Include full sub-agent findings here]

## Recommendation

[Based on results, one of:]
- All criteria met and tests pass. Ready for PR creation.
- X criteria need attention before PR. [List what needs fixing]
- Implementation incomplete. [List major gaps]
```

### Step 7: Suggest Next Steps

Based on the verification results:

**If ALL PASS**:
```
All acceptance criteria verified. Ready to create PR:

1. Stage and commit any final changes:
   git add -A && git commit -m "feat: [description]"

2. Push the branch:
   git push -u origin HEAD

3. Create the PR:
   gh pr create --draft --title "[title]" --body "[body]"
```

**If PARTIAL or FAIL**:
```
Verification found issues that need attention:

[List specific issues from sub-agent findings]

After fixing:
1. Run tests to confirm fixes
2. Run `jig verify` again to re-verify
```

---

## Sub-Agent Architecture

This skill uses sub-agents for isolation and extensibility:

```
┌─────────────────────────────────────────────────────┐
│                  Orchestrator (you)                  │
│  - Loads plan and extracts criteria                 │
│  - Spawns sub-agents                                │
│  - Collects results and generates report            │
└──────────────┬────────────────┬─────────────────────┘
               │                │
               ▼                ▼
┌──────────────────┐  ┌──────────────────┐
│ Criteria Agent   │  │ Test Runner      │
│ (Explore)        │  │ (Bash)           │
│                  │  │                  │
│ - Code inspection│  │ - Run test suite │
│ - PASS/FAIL eval │  │ - Report results │
└──────────────────┘  └──────────────────┘

Future agents (not yet implemented):
┌──────────────────┐  ┌──────────────────┐
│ Security Agent   │  │ Performance Agent│
│ - Vuln scanning  │  │ - Benchmarks     │
│ - OWASP checks   │  │ - Profiling      │
└──────────────────┘  └──────────────────┘
```

**Why sub-agents?**
1. **Isolated context**: Sub-agents don't inherit implementation session history
2. **Objective verification**: Fresh perspective without implementation bias
3. **Parallel execution**: Multiple agents can run concurrently
4. **Extensibility**: Easy to add specialized verification agents

---

## Important Rules

1. **Use sub-agents for verification**: Don't verify criteria yourself - delegate to sub-agents
2. **Pass minimal context**: Only give sub-agents what they need (criteria, not implementation history)
3. **Be objective**: Report sub-agent findings accurately, even if unexpected
4. **Aggregate fairly**: Combine results without bias toward pass or fail
5. **Focus on acceptance criteria**: Don't add requirements not in the plan
