# Jig CLI - High-Level Architecture Plan

## Overview

Jig is a **workflow orchestrator** for software engineering. It manages the lifecycle of plans, issues, worktrees, and PRs, delegating AI execution to external tools (Claude Code, Codex, OpenCode, etc.).

**Key insight:** Jig does NOT run AI agents internally. It sets up context (worktree, plan, prompts) and hands off to the user's preferred coding tool. Jig maintains the prompts/skills but execution is outsourced.

## Design Principles

1. **Workflow orchestrator**: Jig manages state, context, and coordination - not AI execution
2. **Tool-agnostic**: Integrates with Claude Code, Codex, OpenCode, etc. via configurable interface
3. **Plan-first**: Every change starts with a well-defined, reviewed plan
4. **Interface-driven**: Linear, GitHub, and other integrations are pluggable
5. **Multi-repo aware**: Works across multiple repositories from day one

---

## Core Architecture

### How Jig Works

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

### Directory Structure

```
jig/
├── cmd/
│   └── jig/
│       └── main.go              # Entry point
├── internal/
│   ├── cli/                     # CLI commands (Cobra)
│   │   ├── root.go
│   │   ├── new.go               # Create new plan
│   │   ├── implement.go         # Setup + launch tool
│   │   ├── review.go            # Review comments + launch tool
│   │   ├── merge.go             # Merge PR
│   │   ├── checkout.go          # Manage worktrees
│   │   ├── clean.go             # Clean stale worktrees
│   │   ├── config.go            # Configuration
│   │   ├── status.go            # Current issue status
│   │   ├── list.go              # List active work
│   │   └── amend.go             # Amend approved plan
│   ├── config/                  # Configuration management
│   │   ├── config.go            # Config struct and loading
│   │   └── store.go             # Secure credential storage
│   ├── plan/                    # Plan domain
│   │   ├── plan.go              # Plan struct and methods
│   │   ├── parser.go            # Markdown+frontmatter parsing
│   │   └── phase.go             # Phase and dependency handling
│   ├── tracker/                 # System of record interface
│   │   ├── tracker.go           # Interface definition
│   │   ├── linear/              # Linear implementation
│   │   │   ├── client.go
│   │   │   ├── issue.go
│   │   │   └── sync.go
│   │   └── mock/                # Mock for testing
│   ├── runner/                  # External tool integration
│   │   ├── runner.go            # Interface definition
│   │   ├── claude.go            # Claude Code launcher
│   │   ├── codex.go             # Codex launcher
│   │   └── generic.go           # Generic command launcher
│   ├── prompt/                  # Prompt/skill management
│   │   ├── prompt.go            # Prompt loading and rendering
│   │   ├── embed.go             # Embedded default prompts
│   │   └── templates/           # Default prompt templates
│   │       ├── plan.md
│   │       ├── implement.md
│   │       ├── review.md
│   │       ├── lead_review.md
│   │       └── security_review.md
│   ├── git/                     # Git operations
│   │   ├── worktree.go          # Worktree management
│   │   ├── branch.go            # Branch naming/management
│   │   └── gh.go                # gh CLI wrapper
│   ├── ui/                      # Bubble Tea TUI components
│   │   ├── spinner.go
│   │   ├── select.go
│   │   ├── confirm.go
│   │   └── plan_view.go
│   └── state/                   # Local state management
│       ├── cache.go             # ~/.jig cache
│       └── worktrees.go         # Worktree tracking
├── prompts/                     # Default prompts (embedded)
│   ├── plan.md
│   ├── implement.md
│   ├── review.md
│   ├── lead_review.md
│   └── security_review.md
├── go.mod
├── go.sum
└── README.md
```

---

## Key Interfaces

### Tracker Interface (System of Record)

```go
type Tracker interface {
    // Issue management
    CreateIssue(ctx context.Context, issue *Issue) (*Issue, error)
    UpdateIssue(ctx context.Context, id string, updates *IssueUpdate) error
    GetIssue(ctx context.Context, id string) (*Issue, error)

    // Sub-issues for phases
    CreateSubIssue(ctx context.Context, parentID string, issue *Issue) (*Issue, error)
    SetBlocking(ctx context.Context, blockerID, blockedID string) error

    // Comments for Q&A and updates
    AddComment(ctx context.Context, issueID string, comment string) error

    // Status management
    TransitionIssue(ctx context.Context, id string, status Status) error
}
```

### Runner Interface (External Tools)

```go
type Runner interface {
    // Name returns the runner identifier (e.g., "claude", "codex")
    Name() string

    // Prepare sets up the context for the tool (writes prompts, skills, etc.)
    Prepare(ctx context.Context, opts *PrepareOpts) error

    // Launch starts the tool in the current shell (exec, replaces process)
    // This function does not return on success - it execs into the tool
    Launch(ctx context.Context, opts *LaunchOpts) error

    // Available checks if the tool is installed and configured
    Available() bool
}

type PrepareOpts struct {
    Plan        *plan.Plan
    Phase       *plan.Phase
    WorktreeDir string
    PromptType  PromptType  // plan, implement, review, etc.
}

type LaunchOpts struct {
    WorktreeDir string
    Prompt      string      // The rendered prompt to pass to the tool
    Interactive bool        // Whether to run in interactive mode
}
```

### Prompt Manager

```go
type PromptManager interface {
    // Load returns the prompt for a given type, checking user overrides first
    Load(promptType PromptType) (string, error)

    // Render applies template variables to the prompt
    Render(prompt string, vars *PromptVars) (string, error)
}

type PromptVars struct {
    Plan         *plan.Plan
    Phase        *plan.Phase
    IssueContext string
    PRComments   []string
    // etc.
}
```

---

## Prompt/Skill System

### Default Prompts (Embedded in Binary)

Prompts are stored in `prompts/` and embedded at build time. They use Go template syntax.

### User Overrides

Users can override any prompt by placing files in `~/.jig/prompts/`:

```
~/.jig/
├── config.toml
├── prompts/
│   ├── implement.md    # Overrides default implement prompt
│   └── my_custom.md    # Custom prompt
└── cache/
```

### Claude Code Integration

For Claude Code, jig can register skills. When `jig implement` runs:

1. Writes a `.claude/skills/jig-implement.md` to the worktree (or uses Claude Code's skill system)
2. Launches `claude` with the skill context
3. User is now in Claude Code session with plan context loaded

---

## Plan Document Format

```markdown
---
id: ENG-123
title: Add user authentication
status: draft | reviewing | approved | in-progress | complete
created: 2024-01-15T10:00:00Z
author: charles
reviewers:
  default: [lead, security]
  optional: [performance]
  opted_out: []
phases:
  - id: phase-1
    title: Backend auth service
    issue_id: ENG-124
    status: pending
    depends_on: []
  - id: phase-2
    title: Frontend login flow
    issue_id: ENG-125
    status: pending
    depends_on: [phase-1]
  - id: phase-3
    title: API integration tests
    issue_id: ENG-126
    status: pending
    depends_on: [] # independent, can run parallel
---

# Add User Authentication

## Problem Statement

[Description of the problem being solved]

## Proposed Solution

[High-level approach]

## Phases

### Phase 1: Backend Auth Service

**Dependencies:** None
**Branch:** `ENG-124-backend-auth-service`

#### Acceptance Criteria

- [ ] JWT token generation and validation
- [ ] User model with password hashing
- [ ] Login/logout endpoints

#### Implementation Details

[Detailed implementation plan]

### Phase 2: Frontend Login Flow

**Dependencies:** Phase 1
**Branch:** `ENG-125-frontend-login` (stacked on ENG-124-backend-auth-service)

#### Acceptance Criteria

- [ ] Login form component
- [ ] Auth state management
- [ ] Protected route wrapper

### Phase 3: API Integration Tests

**Dependencies:** None (can run parallel with Phase 2)
**Branch:** `ENG-126-api-integration-tests`

#### Acceptance Criteria

- [ ] Auth endpoint tests
- [ ] Token refresh tests

## Clarifying Questions & Answers

**Q: Should we support OAuth providers?**
A: Not in MVP, but design for extensibility.

**Q: What session duration?**
A: 7 days with refresh tokens.

## Review Notes

### Lead Engineer Review

- Approved overall architecture
- Suggested using middleware pattern for auth

### Security Review

- Recommended bcrypt over argon2 for compatibility
- Flagged: ensure HTTPS-only cookies
```

---

## CLI Commands

### `jig new [ISSUE_ID]`

Creates a new plan, optionally from an existing issue.

**Flow:**

1. If ISSUE_ID provided, fetch context from Linear
2. Prepare planning prompt with context
3. Launch external tool (claude, etc.) for interactive planning
4. User creates plan in the tool session
5. On exit: parse plan, run review (launches tool for each reviewer)
6. On approval: create Linear issue(s), save plan locally

### `jig implement ISSUE [--remote]`

Implements a plan or phase.

**Flow:**

1. Fetch plan from Linear/local cache
2. Create/checkout worktree for the issue
3. Prepare implementation prompt with plan context
4. **Launch external tool in current shell** (exec - jig process replaced)
5. User is now in tool session with full plan context

**Post-implementation (user runs `jig pr` or similar after exiting tool):**

- Create draft PR via `gh`
- Launch tool with review prompt to validate implementation vs plan
- Add review comments to PR

**Remote flag:** Instead of local tool, submits to remote execution service (future)

### `jig review ISSUE [--remote]`

Addresses PR review comments.

**Flow:**

1. Fetch unresolved PR comments via `gh`
2. Create/checkout worktree if needed
3. Prepare review prompt with comments context
4. **Launch external tool in current shell**
5. User addresses comments in tool session

### `jig merge ISSUE`

Merges an approved PR.

**Flow:**

1. Check for unresolved comments
2. Validate CI status
3. If stacked: check dependencies are merged
4. Merge via `gh pr merge`
5. Update Linear issue status
6. Clean up local worktree

### `jig checkout ISSUE`

Manages worktrees.

**Flow:**

1. Check if worktree exists for issue
2. If not: create worktree with correct branch
3. cd to worktree (output path for shell integration)

### `jig clean`

Cleans up stale worktrees.

**Flow:**

1. List all jig-managed worktrees
2. Check each branch status (merged, closed, etc.)
3. Prompt for confirmation
4. Remove stale worktrees

### `jig config`

Manages configuration and credentials.

**Subcommands:**

- `jig config set <key> <value>` - Set config value
- `jig config get <key>` - Get config value
- `jig config init` - Interactive setup wizard

### `jig status`

Shows status of current issue/worktree.

**Flow:**

1. Detect current worktree/branch
2. Fetch associated plan and Linear issue
3. Display: plan status, phase progress, PR status, unresolved comments

### `jig list`

Lists all active plans and worktrees.

**Flow:**

1. Scan ~/.jig for active plans
2. List worktrees and their status
3. Show: issue ID, title, status, branch, worktree path

**Future:** Rich Bubble Tea dashboard with real-time updates

### `jig amend ISSUE`

Amends an approved plan (triggers new review cycle).

**Flow:**

1. Fetch current plan
2. Launch tool for interactive editing
3. Run full review cycle (launches tool for each reviewer)
4. On approval: update Linear issue, notify stakeholders

**Policy:** Any change to an approved plan requires a new review cycle.

---

## Configuration

### ~/.jig/config.toml

```toml
[default]
tracker = "linear"
runner = "claude"          # default tool: claude, codex, opencode, generic

[linear]
api_key = "lin_api_xxx"    # stored securely
team_id = "TEAM-ID"
default_project = "PROJECT-ID"

[github]
# Uses gh CLI auth, no additional config needed

[runners.claude]
command = "claude"         # command to launch
skill_dir = ".claude/skills"

[runners.codex]
command = "codex"

[runners.opencode]
command = "opencode"

[runners.generic]
command = "my-custom-tool"
prompt_arg = "--prompt"    # how to pass prompt

[review]
default_reviewers = ["lead", "security"]
optional_reviewers = ["performance", "accessibility"]

[git]
branch_pattern = "{issue_id}-{slug}"
worktree_dir = "~/.jig/worktrees"

[repos]
# Multi-repo configuration
[repos.backend]
path = "~/workspace/backend"
tracker_project = "BACKEND-PROJECT"

[repos.frontend]
path = "~/workspace/frontend"
tracker_project = "FRONTEND-PROJECT"
```

---

## MVP Implementation Phases

### Phase 1: Foundation

- [ ] Go project setup with Cobra CLI
- [ ] Configuration system (~/.jig/config.toml)
- [ ] Tracker interface + Linear implementation
- [ ] Basic plan parsing (markdown + frontmatter)
- [ ] Git worktree management

### Phase 2: Planning Flow

- [ ] `jig new` command
- [ ] Prompt system (embedded defaults + user overrides)
- [ ] Claude Code runner (launch in current shell)
- [ ] Linear issue creation with sub-issues and blocking relations

### Phase 3: Execution Flow

- [ ] `jig implement` command
- [ ] `jig checkout` command
- [ ] Context preparation (plan → prompt rendering)
- [ ] Post-execution PR creation via `gh`

### Phase 4: Review & Merge

- [ ] `jig review` command
- [ ] `jig merge` command
- [ ] PR comment fetching and context building
- [ ] Stacked PR support (optional Graphite/gt)

### Phase 5: Visibility & Polish

- [ ] `jig status` command
- [ ] `jig list` command
- [ ] `jig clean` command
- [ ] `jig amend` command
- [ ] Additional runners (codex, opencode, generic)
- [ ] Bubble Tea TUI enhancements (future: rich dashboard)
- [ ] Error handling and edge cases

---

## Dependencies

```
github.com/spf13/cobra          # CLI framework
github.com/spf13/viper          # Configuration
github.com/charmbracelet/bubbletea  # TUI
github.com/charmbracelet/lipgloss   # Styling
github.com/charmbracelet/huh       # Forms
gopkg.in/yaml.v3                # YAML parsing
github.com/adrg/frontmatter     # Frontmatter parsing
```

---

## Verification

After implementation:

1. Run `go build ./cmd/jig` to verify compilation
2. Run `go test ./...` to run unit tests
3. Manual test flow:
   - `jig config init` → configure Linear + runner
   - `jig new` → create a plan (launches claude)
   - After planning session: verify Linear issue created
   - `jig implement ENG-123` → launches claude in worktree
   - After implementation: `gh pr create --draft`
   - `jig review ENG-123` → launches claude with PR context
   - `jig merge ENG-123` → merges PR, updates Linear
