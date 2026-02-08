---
description: Create a PR and record it in issue metadata
argument-hint: "[<issue-id>]"
---

# /jig:pr

Create a pull request and record it in issue metadata for tracking.

This skill helps you create a PR with proper metadata linkage, enabling features like `jig merge <issue>` to work without needing to be on the branch.

## Prerequisites

- Must have changes committed to the branch
- Must have the branch pushed to origin
- GitHub CLI (gh) must be authenticated

## Usage

```bash
/jig:pr              # Create PR, auto-detect issue from branch
/jig:pr NUM-123      # Create PR for specific issue
```

---

## Agent Instructions

### Step 1: Check Prerequisites

Verify the user is ready to create a PR:

1. **Check for uncommitted changes**:
   ```bash
   git status --porcelain
   ```

   If there are uncommitted changes, ask the user if they want to commit first.

2. **Check if branch is pushed**:
   ```bash
   git rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null || echo "NOT_PUSHED"
   ```

   If not pushed, offer to push:
   ```bash
   git push -u origin HEAD
   ```

3. **Detect issue ID** from `$ARGUMENTS` or current branch:
   ```bash
   git rev-parse --abbrev-ref HEAD
   ```

   Extract issue ID from branch name (e.g., `NUM-123-feature-name` -> `NUM-123`).

### Step 2: Gather PR Information

Check if there's a plan cached for this issue:

```bash
jig status
```

If a plan exists, it will be used for the PR title and body by default.

Ask the user:
- Is the default title okay, or should it be customized?
- Should this be a draft PR (default) or ready for review?

### Step 3: Create the PR

Run the `jig pr` command:

```bash
# For draft PR with defaults (most common)
jig pr

# Or with specific issue ID
jig pr NUM-123

# Or with custom title
jig pr --title "feat: implement usage metrics tracking"

# For non-draft PR (ready for review)
jig pr --draft=false
```

### Step 4: Confirm Success

After the PR is created:

1. Display the PR URL
2. Confirm the metadata was updated
3. Suggest next steps:
   - "PR created! You can now use `jig merge NUM-123` to merge it when ready."
   - If draft: "Mark it ready for review with `gh pr ready`"

---

## Output Format

- **Start**: "Let me help you create a PR..."
- **Progress**: Show what's being checked/done
- **End**: "PR created successfully! URL: ..."

---

## Example Session

```
User: /jig:pr