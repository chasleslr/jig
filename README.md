# Jig

A workflow orchestrator for software engineering. Jig manages the lifecycle of plans, issues, worktrees, and PRs, delegating AI execution to external tools.

**Key insight:** Jig does NOT run AI agents internally. It sets up context (worktree, plan, prompts) and hands off to your preferred coding tool (Claude Code, Codex, OpenCode, etc.).

## Installation

```bash
go install github.com/charleslr/jig/cmd/jig@latest
```

Or build from source:

```bash
git clone https://github.com/charleslr/jig.git
cd jig
go build ./cmd/jig
```

## Quick Start

1. **Configure jig:**

```bash
jig config init
```

This will guide you through setting up:

- Linear API key (for issue tracking)
- Default coding tool (claude, codex, opencode)
- Git branch patterns
- Worktree directory

2. **Create a plan:**

```bash
jig plan
# or from an existing issue
jig plan ENG-123
```

This launches your coding tool for an interactive planning session.

# TODO: plan review (review an existing plan)

3. **Implement the plan:**

```bash
jig implement ENG-123
```

Creates a worktree and launches your coding tool with the plan context.

4. **Address PR feedback:**

```bash
jig review ENG-123
```

Fetches unresolved PR comments and launches your tool to address them.

5. **Merge when ready:**

```bash
jig merge ENG-123
```

Validates the PR is ready and merges it.

## Commands

| Command               | Description                          |
| --------------------- | ------------------------------------ |
| `jig new [ISSUE]`     | Create a new plan                    |
| `jig implement ISSUE` | Set up worktree and implement        |
| `jig review [ISSUE]`  | Address PR review comments           |
| `jig merge [ISSUE]`   | Merge an approved PR                 |
| `jig checkout ISSUE`  | Create/switch to an issue's worktree |
| `jig status [ISSUE]`  | Show status of current issue         |
| `jig list`            | List all active plans and worktrees  |
| `jig clean`           | Clean up stale worktrees             |
| `jig amend ISSUE`     | Amend an approved plan               |
| `jig config`          | Manage configuration                 |

## How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                         jig CLI                                 │
├─────────────────────────────────────────────────────────────────┤
│  1. Manage plans (create, review, amend)                        │
│  2. Sync with Linear (issues, sub-issues, status)               │
│  3. Manage worktrees (create, checkout, clean)                  │
│  4. Prepare context (plan, prompts, codebase info)              │
│  5. Launch external tool in current shell → jig exits          │
│  6. Post-execution: create PRs, update status, trigger reviews  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    External Coding Tools                         │
│  • Claude Code (via skills/prompts)                             │
│  • Codex                                                         │
│  • OpenCode                                                      │
│  • Others (configurable)                                         │
└─────────────────────────────────────────────────────────────────┘
```

## Configuration

Configuration is stored in `~/.jig/config.toml`:

```toml
[default]
tracker = "linear"
runner = "claude"

[linear]
team_id = "TEAM-ID"
default_project = "PROJECT-ID"

[runners.claude]
command = "claude"
skill_dir = ".claude/skills"

[git]
branch_pattern = "{issue_id}-{slug}"
worktree_dir = "~/.jig/worktrees"

[review]
default_reviewers = ["lead", "security"]
```

## Prompts

Jig uses prompt templates for different scenarios:

- `plan.md` - Planning new features
- `implement.md` - Implementation guidance
- `review.md` - Addressing PR comments
- `lead_review.md` - Lead engineer review
- `security_review.md` - Security review

Override default prompts by placing files in `~/.jig/prompts/`.

## Plan Document Format

Plans use markdown with YAML frontmatter:

```markdown
---
id: ENG-123
title: Add user authentication
status: draft
author: charles
phases:
  - id: phase-1
    title: Backend auth service
    status: pending
    depends_on: []
  - id: phase-2
    title: Frontend login flow
    status: pending
    depends_on: [phase-1]
---

# Add User Authentication

## Problem Statement

[Description]

## Proposed Solution

[Approach]

## Phases

### Phase 1: Backend Auth Service

...
```

## License

MIT
