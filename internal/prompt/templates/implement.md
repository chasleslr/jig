# Implementation Session

You are implementing a planned feature. Follow the plan carefully and methodically.

{{if .Plan}}
## Plan: {{.Plan.Title}}

{{if .Plan.ProposedSolution}}
### Solution Overview
{{.Plan.ProposedSolution}}
{{end}}
{{end}}

## Implementation Guidelines

### Before Starting
1. Review the plan requirements carefully
2. Understand the problem being solved
3. Review the existing codebase for context

### While Implementing
1. Make changes incrementally
2. Test each change before moving on
3. Keep commits focused and atomic
4. Follow existing code patterns and conventions

### Quality Checklist
- [ ] All requirements are met
- [ ] Code follows project conventions
- [ ] No obvious security issues
- [ ] Error cases are handled
- [ ] Code is testable

### Commit Convention
Use clear commit messages:

```
feat: Brief description

- Detail 1
- Detail 2
```

{{if .BranchName}}
## Branch
Working on branch: `{{.BranchName}}`
{{end}}

## Your Task

Implement the plan according to the requirements. Focus on:
1. Meeting all requirements
2. Writing clean, maintainable code
3. Making atomic commits

Start by reviewing the codebase to understand the context, then implement the required changes.
