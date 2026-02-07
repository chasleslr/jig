package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charleslr/jig/internal/config"
)

// Worktree represents a git worktree
type Worktree struct {
	Path   string
	Branch string
	Commit string
	Bare   bool
}

// ListWorktrees returns all worktrees in the repository
func ListWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []Worktree
	var current Worktree

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = Worktree{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch refs/heads/") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == "bare" {
			current.Bare = true
		}
	}

	// Don't forget the last worktree
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// CreateWorktree creates a new worktree for an issue
func CreateWorktree(issueID, branchName string) (string, error) {
	cfg := config.Get()

	// Determine worktree directory
	worktreeDir := cfg.Git.WorktreeDir
	if worktreeDir == "" {
		jigDir, err := config.JigDir()
		if err != nil {
			return "", err
		}
		worktreeDir = filepath.Join(jigDir, "worktrees")
	}

	// Expand ~ in path
	if strings.HasPrefix(worktreeDir, "~/") {
		home, _ := os.UserHomeDir()
		worktreeDir = filepath.Join(home, worktreeDir[2:])
	}

	// Create worktree directory if it doesn't exist
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Create a subdirectory for this worktree
	worktreePath := filepath.Join(worktreeDir, issueID)

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return worktreePath, nil
	}

	// Check if branch exists
	branchExists, err := BranchExists(branchName)
	if err != nil {
		return "", err
	}

	if branchExists {
		// Create worktree from existing branch
		cmd := exec.Command("git", "worktree", "add", worktreePath, branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to create worktree: %s: %w", string(output), err)
		}
	} else {
		// Create worktree with new branch from main/master
		baseBranch, err := GetDefaultBranch()
		if err != nil {
			return "", err
		}

		cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to create worktree with new branch: %s: %w", string(output), err)
		}
	}

	return worktreePath, nil
}

// RemoveWorktree removes a worktree
func RemoveWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", path, "--force")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove worktree: %s: %w", string(output), err)
	}
	return nil
}

// GetWorktree returns the worktree for a given branch
func GetWorktree(branchName string) (*Worktree, error) {
	worktrees, err := ListWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branchName {
			return &wt, nil
		}
	}

	return nil, nil
}

// GetWorktreeByPath returns the worktree at a given path
func GetWorktreeByPath(path string) (*Worktree, error) {
	worktrees, err := ListWorktrees()
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		wtAbs, _ := filepath.Abs(wt.Path)
		if wtAbs == absPath {
			return &wt, nil
		}
	}

	return nil, nil
}

// GetCurrentWorktree returns the worktree for the current directory
func GetCurrentWorktree() (*Worktree, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return GetWorktreeByPath(cwd)
}

// PruneWorktrees removes stale worktree references
func PruneWorktrees() error {
	cmd := exec.Command("git", "worktree", "prune")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to prune worktrees: %s: %w", string(output), err)
	}
	return nil
}

// IsInWorktree returns true if the current directory is in a worktree
func IsInWorktree() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// GetWorktreeRoot returns the root directory of the current worktree
func GetWorktreeRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
