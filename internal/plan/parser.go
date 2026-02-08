package plan

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// Frontmatter represents the YAML frontmatter of a plan document
type Frontmatter struct {
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title"`
	Status    Status    `yaml:"status"`
	Created   string    `yaml:"created"`
	Author    string    `yaml:"author"`
	Reviewers Reviewers `yaml:"reviewers"`
}

// ParseFile reads and parses a plan from a file
func ParseFile(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	plan, err := Parse(data)
	if err != nil {
		return nil, err
	}
	plan.FilePath = path
	return plan, nil
}

// Parse parses a plan from markdown with frontmatter
func Parse(data []byte) (*Plan, error) {
	var fm Frontmatter
	rest, err := frontmatter.Parse(bytes.NewReader(data), &fm)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	plan := &Plan{
		ID:               fm.ID,
		Title:            fm.Title,
		Status:           fm.Status,
		Author:           fm.Author,
		Reviewers:        fm.Reviewers,
		RawContent:       string(data),
		QuestionsAnswers: make(map[string]string),
		ReviewNotes:      make(map[ReviewerType]string),
	}

	// Parse created date
	if fm.Created != "" {
		// Try multiple date formats
		// The time is stored in the frontmatter
	}

	// Parse markdown body
	body := string(rest)
	parseMarkdownBody(plan, body)

	return plan, nil
}

// parseMarkdownBody extracts structured content from the markdown
func parseMarkdownBody(plan *Plan, body string) {
	sections := splitSections(body)

	for _, section := range sections {
		headerLower := strings.ToLower(section.Header)

		switch {
		case strings.Contains(headerLower, "problem"):
			plan.ProblemStatement = strings.TrimSpace(section.Content)

		case strings.Contains(headerLower, "solution") || strings.Contains(headerLower, "proposed"):
			plan.ProposedSolution = strings.TrimSpace(section.Content)

		case strings.Contains(headerLower, "question") || strings.Contains(headerLower, "q&a"):
			parseQA(plan, section.Content)

		case strings.Contains(headerLower, "review"):
			parseReviewNotes(plan, section.Content)
		}
	}
}

// Section represents a markdown section
type Section struct {
	Header  string
	Level   int
	Content string
}

// splitSections splits markdown into sections by headers
func splitSections(body string) []Section {
	var sections []Section

	// Match markdown headers
	headerRegex := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	matches := headerRegex.FindAllStringSubmatchIndex(body, -1)

	for i, match := range matches {
		level := match[3] - match[2] // Length of # characters
		header := body[match[4]:match[5]]

		// Content is from end of header to start of next header (or end)
		contentStart := match[1]
		contentEnd := len(body)
		if i+1 < len(matches) {
			contentEnd = matches[i+1][0]
		}

		content := strings.TrimSpace(body[contentStart:contentEnd])

		sections = append(sections, Section{
			Header:  header,
			Level:   level,
			Content: content,
		})
	}

	return sections
}

// parseQA extracts Q&A pairs from a section
func parseQA(plan *Plan, content string) {
	// Look for Q: ... A: ... patterns
	qaRegex := regexp.MustCompile(`(?m)\*\*Q:\s*(.+?)\*\*\s*\n\s*A:\s*(.+?)(?:\n\n|$)`)
	matches := qaRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			question := strings.TrimSpace(match[1])
			answer := strings.TrimSpace(match[2])
			plan.QuestionsAnswers[question] = answer
		}
	}
}

// parseReviewNotes extracts review notes by reviewer type
func parseReviewNotes(plan *Plan, content string) {
	sections := splitSections(content)

	for _, section := range sections {
		headerLower := strings.ToLower(section.Header)

		switch {
		case strings.Contains(headerLower, "lead"):
			plan.ReviewNotes[ReviewerLead] = strings.TrimSpace(section.Content)
		case strings.Contains(headerLower, "security"):
			plan.ReviewNotes[ReviewerSecurity] = strings.TrimSpace(section.Content)
		case strings.Contains(headerLower, "performance"):
			plan.ReviewNotes[ReviewerPerformance] = strings.TrimSpace(section.Content)
		case strings.Contains(headerLower, "accessibility"):
			plan.ReviewNotes[ReviewerAccessibility] = strings.TrimSpace(section.Content)
		}
	}
}

// ValidateStructure validates that the plan markdown has the required structure
func ValidateStructure(data []byte) error {
	// Parse frontmatter
	var fm Frontmatter
	rest, err := frontmatter.Parse(bytes.NewReader(data), &fm)
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Validate required frontmatter fields
	var missing []string
	if fm.ID == "" {
		missing = append(missing, "id")
	}
	if fm.Title == "" {
		missing = append(missing, "title")
	}
	if fm.Status == "" {
		missing = append(missing, "status")
	}
	if fm.Author == "" {
		missing = append(missing, "author")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required frontmatter fields: %s", strings.Join(missing, ", "))
	}

	// Parse markdown body and validate required sections
	body := string(rest)
	sections := splitSections(body)

	var foundProblem, foundSolution bool
	for _, section := range sections {
		headerLower := strings.ToLower(section.Header)
		if strings.Contains(headerLower, "problem") {
			foundProblem = true
		}
		if strings.Contains(headerLower, "solution") || strings.Contains(headerLower, "proposed") {
			foundSolution = true
		}
	}

	var missingSections []string
	if !foundProblem {
		missingSections = append(missingSections, "Problem Statement")
	}
	if !foundSolution {
		missingSections = append(missingSections, "Proposed Solution")
	}
	if len(missingSections) > 0 {
		return fmt.Errorf("missing required sections: %s", strings.Join(missingSections, ", "))
	}

	return nil
}

// Serialize converts a plan back to markdown with frontmatter.
// If RawContent is available, it preserves the original markdown body
// while updating the frontmatter with current field values.
func Serialize(plan *Plan) ([]byte, error) {
	var buf bytes.Buffer

	// Write frontmatter with current field values
	fm := Frontmatter{
		ID:        plan.ID,
		Title:     plan.Title,
		Status:    plan.Status,
		Created:   plan.Created.Format("2006-01-02T15:04:05Z"),
		Author:    plan.Author,
		Reviewers: plan.Reviewers,
	}

	buf.WriteString("---\n")
	yamlData, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize frontmatter: %w", err)
	}
	buf.Write(yamlData)
	buf.WriteString("---\n")

	// If we have raw content, extract and preserve the original body
	if plan.RawContent != "" {
		body, err := extractBodyFromRawContent(plan.RawContent)
		if err != nil {
			return nil, err
		}
		buf.WriteString(body)
		return buf.Bytes(), nil
	}

	// Fallback: reconstruct from parsed fields (for programmatically created plans)
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("# %s\n\n", plan.Title))

	if plan.ProblemStatement != "" {
		buf.WriteString("## Problem Statement\n\n")
		buf.WriteString(plan.ProblemStatement)
		buf.WriteString("\n\n")
	}

	if plan.ProposedSolution != "" {
		buf.WriteString("## Proposed Solution\n\n")
		buf.WriteString(plan.ProposedSolution)
		buf.WriteString("\n\n")
	}

	if len(plan.QuestionsAnswers) > 0 {
		buf.WriteString("## Clarifying Questions & Answers\n\n")
		for q, a := range plan.QuestionsAnswers {
			buf.WriteString(fmt.Sprintf("**Q: %s**\n", q))
			buf.WriteString(fmt.Sprintf("A: %s\n\n", a))
		}
	}

	if len(plan.ReviewNotes) > 0 {
		buf.WriteString("## Review Notes\n\n")
		for reviewer, notes := range plan.ReviewNotes {
			buf.WriteString(fmt.Sprintf("### %s Review\n\n", strings.Title(string(reviewer))))
			buf.WriteString(notes)
			buf.WriteString("\n\n")
		}
	}

	return buf.Bytes(), nil
}

// extractBodyFromRawContent extracts everything after the frontmatter
func extractBodyFromRawContent(rawContent string) (string, error) {
	// Find the closing frontmatter delimiter
	const delimiter = "---"

	// Skip the opening delimiter
	start := strings.Index(rawContent, delimiter)
	if start == -1 {
		return "", fmt.Errorf("no frontmatter found in raw content")
	}

	// Find the closing delimiter
	rest := rawContent[start+len(delimiter):]
	end := strings.Index(rest, delimiter)
	if end == -1 {
		return "", fmt.Errorf("unclosed frontmatter in raw content")
	}

	// Return everything after the closing delimiter
	body := rest[end+len(delimiter):]
	return body, nil
}

// SaveFile writes a plan to a file
func SaveFile(plan *Plan, path string) error {
	data, err := Serialize(plan)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	plan.FilePath = path
	return nil
}
