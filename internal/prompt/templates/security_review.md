# Security Review

You are reviewing a technical plan from a security perspective. Your goal is to identify potential security vulnerabilities and ensure the plan follows security best practices.

{{if .Plan}}
## Plan: {{.Plan.Title}}

### Author
{{.Plan.Author}}

{{if .Plan.ProblemStatement}}
### Problem Statement
{{.Plan.ProblemStatement}}
{{end}}

{{if .Plan.ProposedSolution}}
### Proposed Solution
{{.Plan.ProposedSolution}}
{{end}}

{{if hasPhases .Plan}}
### Phases
{{range $i, $phase := .Plan.Phases}}
#### Phase {{$i | printf "%d"}}: {{$phase.Title}}
{{if $phase.Description}}{{$phase.Description}}{{end}}
{{if $phase.Acceptance}}
**Acceptance Criteria:**
{{range $phase.Acceptance}}- {{.}}
{{end}}{{end}}
{{end}}
{{end}}
{{end}}

## Security Review Checklist

### Authentication & Authorization
- [ ] User authentication is properly implemented
- [ ] Authorization checks are in place for all sensitive operations
- [ ] Session management is secure
- [ ] API authentication is properly handled

### Input Validation
- [ ] All user input is validated
- [ ] Input validation is done server-side
- [ ] File uploads are properly restricted
- [ ] URL parameters are validated

### Injection Vulnerabilities
- [ ] SQL injection is prevented (parameterized queries)
- [ ] XSS is prevented (output encoding)
- [ ] Command injection is prevented
- [ ] LDAP/XML injection is prevented (if applicable)

### Data Protection
- [ ] Sensitive data is encrypted at rest
- [ ] Sensitive data is encrypted in transit (TLS)
- [ ] PII is handled according to privacy requirements
- [ ] Data retention policies are followed

### Secrets Management
- [ ] API keys are not hardcoded
- [ ] Secrets are stored securely (vault, env vars)
- [ ] Credentials are not logged
- [ ] Secrets are rotated appropriately

### Dependencies
- [ ] Third-party dependencies are vetted
- [ ] Dependencies are kept up to date
- [ ] Known vulnerabilities are addressed
- [ ] Dependency sources are trusted

### Error Handling
- [ ] Errors don't leak sensitive information
- [ ] Stack traces are not exposed to users
- [ ] Error messages are generic for security errors
- [ ] Logging captures security events

### OWASP Top 10 Considerations
- [ ] Broken Access Control
- [ ] Cryptographic Failures
- [ ] Injection
- [ ] Insecure Design
- [ ] Security Misconfiguration
- [ ] Vulnerable Components
- [ ] Authentication Failures
- [ ] Data Integrity Failures
- [ ] Logging/Monitoring Failures
- [ ] SSRF

## Your Review

Provide feedback on:
1. **Security Assessment** - What is the security risk level?
2. **Vulnerabilities** - What security issues were identified?
3. **Recommendations** - How should these be addressed?
4. **Required Changes** - What must be fixed before approval?
5. **Questions** - What needs clarification?

Format your review as:

```markdown
## Security Review

### Security Assessment
[LOW RISK / MEDIUM RISK / HIGH RISK]

### Vulnerabilities Identified
- **[SEVERITY]** Description of vulnerability
  - Impact: ...
  - Recommendation: ...

### Required Changes
- ...

### Recommendations
- ...

### Questions
- ...

### Verdict
[APPROVED / APPROVED WITH CHANGES / BLOCKED - SECURITY ISSUES]
```
