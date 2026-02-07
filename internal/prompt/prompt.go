package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/plan"
)

//go:embed templates/*.md
var embeddedPrompts embed.FS

// Type identifies the type of prompt
type Type string

const (
	TypePlan           Type = "plan"
	TypeImplement      Type = "implement"
	TypeReview         Type = "review"
	TypeLeadReview     Type = "lead_review"
	TypeSecurityReview Type = "security_review"
)

// Vars contains variables for template rendering
type Vars struct {
	Plan         *plan.Plan
	Phase        *plan.Phase
	IssueContext string
	PRComments   []string
	PRNumber     string
	PRTitle      string
	PRBody       string
	BranchName   string
	RepoName     string
	Custom       map[string]string
}

// Manager handles prompt loading and rendering
type Manager struct {
	userPromptsDir string
}

// NewManager creates a new prompt manager
func NewManager() (*Manager, error) {
	promptsDir, err := config.PromptsDir()
	if err != nil {
		return nil, err
	}

	return &Manager{
		userPromptsDir: promptsDir,
	}, nil
}

// Load returns the prompt content for a given type
// It first checks for user overrides, then falls back to embedded defaults
func (m *Manager) Load(promptType Type) (string, error) {
	// Check for user override
	userPath := filepath.Join(m.userPromptsDir, string(promptType)+".md")
	if data, err := os.ReadFile(userPath); err == nil {
		return string(data), nil
	}

	// Fall back to embedded default
	embeddedPath := fmt.Sprintf("templates/%s.md", promptType)
	data, err := embeddedPrompts.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("prompt not found: %s", promptType)
	}

	return string(data), nil
}

// Render applies template variables to a prompt
func (m *Manager) Render(promptContent string, vars *Vars) (string, error) {
	tmpl, err := template.New("prompt").Funcs(templateFuncs()).Parse(promptContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// LoadAndRender loads a prompt and renders it with variables
func (m *Manager) LoadAndRender(promptType Type, vars *Vars) (string, error) {
	content, err := m.Load(promptType)
	if err != nil {
		return "", err
	}

	return m.Render(content, vars)
}

// SaveUserPrompt saves a user-defined prompt override
func (m *Manager) SaveUserPrompt(promptType Type, content string) error {
	if err := os.MkdirAll(m.userPromptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create prompts directory: %w", err)
	}

	path := filepath.Join(m.userPromptsDir, string(promptType)+".md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	return nil
}

// ListUserPrompts returns all user-defined prompts
func (m *Manager) ListUserPrompts() ([]Type, error) {
	entries, err := os.ReadDir(m.userPromptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var prompts []Type
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".md" {
			prompts = append(prompts, Type(name[:len(name)-3]))
		}
	}

	return prompts, nil
}

// DeleteUserPrompt removes a user-defined prompt override
func (m *Manager) DeleteUserPrompt(promptType Type) error {
	path := filepath.Join(m.userPromptsDir, string(promptType)+".md")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete prompt file: %w", err)
	}
	return nil
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"join": func(items []string, sep string) string {
			result := ""
			for i, item := range items {
				if i > 0 {
					result += sep
				}
				result += item
			}
			return result
		},
		"hasPhases": func(p *plan.Plan) bool {
			return p != nil && len(p.Phases) > 0
		},
		"phaseStatus": func(status plan.PhaseStatus) string {
			switch status {
			case plan.PhaseStatusPending:
				return "⬜ Pending"
			case plan.PhaseStatusInProgress:
				return "🔄 In Progress"
			case plan.PhaseStatusComplete:
				return "✅ Complete"
			case plan.PhaseStatusBlocked:
				return "🚫 Blocked"
			default:
				return string(status)
			}
		},
	}
}

// DefaultManager is a convenience instance
var DefaultManager *Manager

// Init initializes the default manager
func Init() error {
	var err error
	DefaultManager, err = NewManager()
	return err
}

// Load loads a prompt using the default manager
func Load(promptType Type) (string, error) {
	if DefaultManager == nil {
		if err := Init(); err != nil {
			return "", err
		}
	}
	return DefaultManager.Load(promptType)
}

// Render renders a prompt using the default manager
func Render(promptContent string, vars *Vars) (string, error) {
	if DefaultManager == nil {
		if err := Init(); err != nil {
			return "", err
		}
	}
	return DefaultManager.Render(promptContent, vars)
}

// LoadAndRender loads and renders a prompt using the default manager
func LoadAndRender(promptType Type, vars *Vars) (string, error) {
	if DefaultManager == nil {
		if err := Init(); err != nil {
			return "", err
		}
	}
	return DefaultManager.LoadAndRender(promptType, vars)
}
