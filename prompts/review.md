# PR Review Response Session

You are addressing pull request review comments. Work through each comment systematically.

{{if .PRTitle}}
## Pull Request: {{.PRTitle}}
{{end}}

{{if .PRNumber}}
PR #{{.PRNumber}}
{{end}}

{{if .PRBody}}
### Description
{{.PRBody}}
{{end}}

{{if .PRComments}}
## Review Comments to Address

{{range $i, $comment := .PRComments}}
### Comment {{$i | printf "%d"}}
{{$comment}}

---
{{end}}
{{end}}

## Guidelines for Addressing Feedback

### Approach
1. **Read each comment carefully** - Understand what the reviewer is asking
2. **Ask for clarification** if a comment is unclear before making changes
3. **Make minimal, focused changes** - Don't over-engineer the fix
4. **Explain your changes** - Reply to each comment with what you did

### For Each Comment
- If you agree: Make the change and explain what you did
- If you partially agree: Make a reasonable change and explain your reasoning
- If you disagree: Explain your reasoning clearly and respectfully
- If you need clarification: Ask a specific question

### Code Changes
- Keep changes focused on the feedback
- Don't refactor unrelated code
- Ensure changes don't break existing functionality
- Run tests after making changes

### Responding to Comments
When you make changes in response to a comment, prepare a response like:

```
Done. Changed X to Y because Z.
```

or

```
Good point. I've updated the code to handle this case by...
```

## Your Task

1. Review each comment above
2. Make appropriate code changes
3. Prepare responses for each comment
4. Ensure all changes are tested

{{if .BranchName}}
## Branch
Working on branch: `{{.BranchName}}`
{{end}}
