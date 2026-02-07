# Implementation Session

You are implementing a planned feature. Follow the plan carefully and methodically.

{{if .Plan}}
## Plan: {{.Plan.Title}}

{{if .Plan.ProposedSolution}}
### Solution Overview
{{.Plan.ProposedSolution}}
{{end}}

{{if hasPhases .Plan}}
### All Phases
{{range $i, $phase := .Plan.Phases}}
{{$i | printf "%d"}}. **{{$phase.Title}}** - {{phaseStatus $phase.Status}}
{{end}}
{{end}}
{{end}}

{{if .Phase}}
## Current Phase: {{.Phase.Title}}

{{if .Phase.Description}}
### Description
{{.Phase.Description}}
{{end}}

{{if .Phase.Acceptance}}
### Acceptance Criteria
{{range .Phase.Acceptance}}
- [ ] {{.}}
{{end}}
{{end}}

{{if .Phase.DependsOn}}
### Dependencies
This phase depends on: {{join .Phase.DependsOn ", "}}
Ensure these are complete before proceeding.
{{end}}
{{end}}

## Implementation Guidelines

### Before Starting
1. Review the acceptance criteria carefully
2. Understand how this phase fits into the larger plan
3. Check that any dependencies are satisfied

### While Implementing
1. Make changes incrementally
2. Test each change before moving on
3. Keep commits focused and atomic
4. Follow existing code patterns and conventions

### Quality Checklist
- [ ] All acceptance criteria are met
- [ ] Code follows project conventions
- [ ] No obvious security issues
- [ ] Error cases are handled
- [ ] Code is testable

### Commit Convention
Use clear commit messages that reference the phase:

```
feat(phase-id): Brief description

- Detail 1
- Detail 2
```

{{if .BranchName}}
## Branch
Working on branch: `{{.BranchName}}`
{{end}}

## Your Task

Implement the current phase according to the plan. Focus on:
1. Meeting all acceptance criteria
2. Writing clean, maintainable code
3. Making atomic commits

Start by reviewing the codebase to understand the context, then implement the required changes.
