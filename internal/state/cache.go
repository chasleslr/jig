package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/plan"
)

// Cache provides local caching for plans and metadata
type Cache struct {
	dir string
}

// CachedPlan stores plan data with metadata
type CachedPlan struct {
	Plan      *plan.Plan `json:"plan"`
	IssueID   string     `json:"issue_id"`
	CachedAt  time.Time  `json:"cached_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// NewCache creates a new cache instance
func NewCache() (*Cache, error) {
	cacheDir, err := config.CacheDir()
	if err != nil {
		return nil, err
	}

	// Create cache directories
	for _, subdir := range []string{"plans", "issues"} {
		path := filepath.Join(cacheDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	return &Cache{dir: cacheDir}, nil
}

// SavePlan caches a plan locally
func (c *Cache) SavePlan(p *plan.Plan) error {
	if p.ID == "" {
		return fmt.Errorf("plan ID is required for caching")
	}

	cached := &CachedPlan{
		Plan:      p,
		IssueID:   p.ID,
		CachedAt:  time.Now(),
		UpdatedAt: p.Updated,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize plan: %w", err)
	}

	path := filepath.Join(c.dir, "plans", p.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan cache: %w", err)
	}

	// Also save the plan markdown
	mdPath := filepath.Join(c.dir, "plans", p.ID+".md")
	mdData, err := plan.Serialize(p)
	if err != nil {
		return fmt.Errorf("failed to serialize plan markdown: %w", err)
	}
	if err := os.WriteFile(mdPath, mdData, 0644); err != nil {
		return fmt.Errorf("failed to write plan markdown: %w", err)
	}

	return nil
}

// GetPlan retrieves a cached plan by ID
func (c *Cache) GetPlan(id string) (*plan.Plan, error) {
	path := filepath.Join(c.dir, "plans", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plan cache: %w", err)
	}

	var cached CachedPlan
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("failed to parse plan cache: %w", err)
	}

	return cached.Plan, nil
}

// GetPlanMarkdown retrieves the raw markdown for a cached plan
func (c *Cache) GetPlanMarkdown(id string) (string, error) {
	path := filepath.Join(c.dir, "plans", id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read plan markdown: %w", err)
	}
	return string(data), nil
}

// DeletePlan removes a cached plan
func (c *Cache) DeletePlan(id string) error {
	jsonPath := filepath.Join(c.dir, "plans", id+".json")
	mdPath := filepath.Join(c.dir, "plans", id+".md")

	os.Remove(jsonPath)
	os.Remove(mdPath)

	return nil
}

// ListPlans returns all cached plans
func (c *Cache) ListPlans() ([]*plan.Plan, error) {
	planDir := filepath.Join(c.dir, "plans")
	entries, err := os.ReadDir(planDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plans directory: %w", err)
	}

	var plans []*plan.Plan
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5] // Remove .json
		p, err := c.GetPlan(id)
		if err != nil {
			continue
		}
		if p != nil {
			plans = append(plans, p)
		}
	}

	return plans, nil
}

// IssueMetadata stores additional metadata about an issue
type IssueMetadata struct {
	IssueID      string    `json:"issue_id"`
	PlanID       string    `json:"plan_id,omitempty"`
	WorktreePath string    `json:"worktree_path,omitempty"`
	BranchName   string    `json:"branch_name,omitempty"`
	PRNumber     int       `json:"pr_number,omitempty"`
	PRURL        string    `json:"pr_url,omitempty"`
	LastActive   time.Time `json:"last_active"`
}

// SaveIssueMetadata caches issue metadata
func (c *Cache) SaveIssueMetadata(meta *IssueMetadata) error {
	if meta.IssueID == "" {
		return fmt.Errorf("issue ID is required")
	}

	meta.LastActive = time.Now()

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	path := filepath.Join(c.dir, "issues", meta.IssueID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// GetIssueMetadata retrieves cached issue metadata
func (c *Cache) GetIssueMetadata(issueID string) (*IssueMetadata, error) {
	path := filepath.Join(c.dir, "issues", issueID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var meta IssueMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &meta, nil
}

// ListIssueMetadata returns all cached issue metadata
func (c *Cache) ListIssueMetadata() ([]*IssueMetadata, error) {
	issueDir := filepath.Join(c.dir, "issues")
	entries, err := os.ReadDir(issueDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read issues directory: %w", err)
	}

	var metadata []*IssueMetadata
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5]
		meta, err := c.GetIssueMetadata(id)
		if err != nil {
			continue
		}
		if meta != nil {
			metadata = append(metadata, meta)
		}
	}

	return metadata, nil
}

// DeleteIssueMetadata removes cached issue metadata
func (c *Cache) DeleteIssueMetadata(issueID string) error {
	path := filepath.Join(c.dir, "issues", issueID+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}
	return nil
}

// Clear removes all cached data
func (c *Cache) Clear() error {
	for _, subdir := range []string{"plans", "issues"} {
		path := filepath.Join(c.dir, subdir)
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to recreate cache directory: %w", err)
		}
	}
	return nil
}

// DefaultCache is a convenience instance
var DefaultCache *Cache

// Init initializes the default cache
func Init() error {
	var err error
	DefaultCache, err = NewCache()
	return err
}
