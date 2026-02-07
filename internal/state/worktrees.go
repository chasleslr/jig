package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/git"
)

// WorktreeState tracks jig-managed worktrees
type WorktreeState struct {
	dir string
}

// WorktreeInfo stores information about a tracked worktree
type WorktreeInfo struct {
	IssueID    string    `json:"issue_id"`
	Path       string    `json:"path"`
	Branch     string    `json:"branch"`
	RepoPath   string    `json:"repo_path"`
	PlanID     string    `json:"plan_id,omitempty"`
	PhaseID    string    `json:"phase_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

// NewWorktreeState creates a new worktree state manager
func NewWorktreeState() (*WorktreeState, error) {
	jigDir, err := config.JigDir()
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(jigDir, "state", "worktrees")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktree state directory: %w", err)
	}

	return &WorktreeState{dir: stateDir}, nil
}

// Track records a worktree as being managed by jig
func (ws *WorktreeState) Track(info *WorktreeInfo) error {
	if info.IssueID == "" {
		return fmt.Errorf("issue ID is required")
	}

	if info.CreatedAt.IsZero() {
		info.CreatedAt = time.Now()
	}
	info.LastUsedAt = time.Now()

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize worktree info: %w", err)
	}

	path := filepath.Join(ws.dir, info.IssueID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write worktree info: %w", err)
	}

	return nil
}

// Get retrieves worktree info by issue ID
func (ws *WorktreeState) Get(issueID string) (*WorktreeInfo, error) {
	path := filepath.Join(ws.dir, issueID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read worktree info: %w", err)
	}

	var info WorktreeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse worktree info: %w", err)
	}

	return &info, nil
}

// GetByPath retrieves worktree info by path
func (ws *WorktreeState) GetByPath(path string) (*WorktreeInfo, error) {
	all, err := ws.List()
	if err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(path)
	for _, info := range all {
		infoAbs, _ := filepath.Abs(info.Path)
		if infoAbs == absPath {
			return info, nil
		}
	}

	return nil, nil
}

// GetByBranch retrieves worktree info by branch name
func (ws *WorktreeState) GetByBranch(branch string) (*WorktreeInfo, error) {
	all, err := ws.List()
	if err != nil {
		return nil, err
	}

	for _, info := range all {
		if info.Branch == branch {
			return info, nil
		}
	}

	return nil, nil
}

// List returns all tracked worktrees
func (ws *WorktreeState) List() ([]*WorktreeInfo, error) {
	entries, err := os.ReadDir(ws.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read worktree state directory: %w", err)
	}

	var worktrees []*WorktreeInfo
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		issueID := entry.Name()[:len(entry.Name())-5]
		info, err := ws.Get(issueID)
		if err != nil {
			continue
		}
		if info != nil {
			worktrees = append(worktrees, info)
		}
	}

	return worktrees, nil
}

// Untrack removes a worktree from tracking
func (ws *WorktreeState) Untrack(issueID string) error {
	path := filepath.Join(ws.dir, issueID+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove worktree info: %w", err)
	}
	return nil
}

// UpdateLastUsed updates the last used time for a worktree
func (ws *WorktreeState) UpdateLastUsed(issueID string) error {
	info, err := ws.Get(issueID)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("worktree not tracked: %s", issueID)
	}

	info.LastUsedAt = time.Now()
	return ws.Track(info)
}

// FindStale returns worktrees that may be stale
// A worktree is considered stale if:
// - The worktree directory no longer exists
// - The branch has been merged
// - The branch no longer exists
func (ws *WorktreeState) FindStale() ([]*WorktreeInfo, error) {
	all, err := ws.List()
	if err != nil {
		return nil, err
	}

	var stale []*WorktreeInfo
	for _, info := range all {
		isStale := false

		// Check if worktree path exists
		if _, err := os.Stat(info.Path); os.IsNotExist(err) {
			isStale = true
		}

		// Check if branch has been merged
		if !isStale && info.Branch != "" {
			merged, err := git.IsBranchMerged(info.Branch)
			if err == nil && merged {
				isStale = true
			}
		}

		// Check if branch still exists
		if !isStale && info.Branch != "" {
			exists, err := git.BranchExists(info.Branch)
			if err == nil && !exists {
				isStale = true
			}
		}

		if isStale {
			stale = append(stale, info)
		}
	}

	return stale, nil
}

// Cleanup removes stale worktrees and their tracking info
func (ws *WorktreeState) Cleanup() ([]string, error) {
	stale, err := ws.FindStale()
	if err != nil {
		return nil, err
	}

	var cleaned []string
	for _, info := range stale {
		// Try to remove the git worktree
		if _, err := os.Stat(info.Path); err == nil {
			if err := git.RemoveWorktree(info.Path); err != nil {
				// Log error but continue
				continue
			}
		}

		// Remove tracking
		if err := ws.Untrack(info.IssueID); err != nil {
			continue
		}

		cleaned = append(cleaned, info.IssueID)
	}

	return cleaned, nil
}

// SyncWithGit syncs tracking state with actual git worktrees
func (ws *WorktreeState) SyncWithGit() error {
	// Get actual git worktrees
	gitWorktrees, err := git.ListWorktrees()
	if err != nil {
		return err
	}

	// Get tracked worktrees
	tracked, err := ws.List()
	if err != nil {
		return err
	}

	// Build maps for lookup
	gitMap := make(map[string]git.Worktree)
	for _, wt := range gitWorktrees {
		gitMap[wt.Path] = wt
	}

	trackedMap := make(map[string]*WorktreeInfo)
	for _, info := range tracked {
		trackedMap[info.Path] = info
	}

	// Remove tracking for worktrees that no longer exist
	for _, info := range tracked {
		if _, exists := gitMap[info.Path]; !exists {
			ws.Untrack(info.IssueID)
		}
	}

	return nil
}

// DefaultWorktreeState is a convenience instance
var DefaultWorktreeState *WorktreeState

// InitWorktreeState initializes the default worktree state
func InitWorktreeState() error {
	var err error
	DefaultWorktreeState, err = NewWorktreeState()
	return err
}
