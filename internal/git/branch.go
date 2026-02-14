package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charleslr/jig/internal/config"
)

// BranchExists checks if a branch exists locally or remotely
func BranchExists(name string) (bool, error) {
	// Check local branches
	cmd := exec.Command("git", "branch", "--list", name)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list branches: %w", err)
	}
	if strings.TrimSpace(string(output)) != "" {
		return true, nil
	}

	// Check remote branches
	cmd = exec.Command("git", "branch", "-r", "--list", "origin/"+name)
	output, err = cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list remote branches: %w", err)
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetDefaultBranch returns the default branch (main or master)
func GetDefaultBranch() (string, error) {
	// Try to get from remote HEAD
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err == nil {
		ref := strings.TrimSpace(string(output))
		// refs/remotes/origin/main -> main
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fall back to checking if main or master exists
	for _, branch := range []string{"main", "master"} {
		exists, err := BranchExists(branch)
		if err == nil && exists {
			return branch, nil
		}
	}

	return "main", nil
}

// CreateBranch creates a new branch
func CreateBranch(name, startPoint string) error {
	var cmd *exec.Cmd
	if startPoint != "" {
		cmd = exec.Command("git", "branch", name, startPoint)
	} else {
		cmd = exec.Command("git", "branch", name)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch: %s: %w", string(output), err)
	}
	return nil
}

// DeleteBranch deletes a branch
func DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}

	cmd := exec.Command("git", "branch", flag, name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete branch: %s: %w", string(output), err)
	}
	return nil
}

// CheckoutBranch switches to a branch
func CheckoutBranch(name string) error {
	cmd := exec.Command("git", "checkout", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout branch: %s: %w", string(output), err)
	}
	return nil
}

// GenerateBranchName creates a branch name from an issue ID and title
func GenerateBranchName(issueID, title string) string {
	cfg := config.Get()
	pattern := cfg.Git.BranchPattern
	if pattern == "" {
		pattern = "{issue_id}-{slug}"
	}

	slug := slugify(title)

	name := strings.ReplaceAll(pattern, "{issue_id}", issueID)
	name = strings.ReplaceAll(name, "{slug}", slug)

	return name
}

// slugify converts a string to a URL-friendly slug
func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile("[^a-z0-9-]")
	s = reg.ReplaceAllString(s, "")

	// Replace multiple hyphens with single hyphen
	reg = regexp.MustCompile("-+")
	s = reg.ReplaceAllString(s, "-")

	// Trim hyphens from ends
	s = strings.Trim(s, "-")

	// Truncate to reasonable length
	if len(s) > 50 {
		s = s[:50]
		// Don't end with a hyphen
		s = strings.TrimRight(s, "-")
	}

	return s
}

// GetBranchUpstream returns the upstream tracking branch
func GetBranchUpstream(name string) (string, error) {
	cmd := exec.Command("git", "config", "--get", fmt.Sprintf("branch.%s.remote", name))
	remote, err := cmd.Output()
	if err != nil {
		return "", nil // No upstream configured
	}

	cmd = exec.Command("git", "config", "--get", fmt.Sprintf("branch.%s.merge", name))
	merge, err := cmd.Output()
	if err != nil {
		return "", nil
	}

	remoteName := strings.TrimSpace(string(remote))
	mergeName := strings.TrimSpace(string(merge))

	// refs/heads/main -> main
	mergeName = strings.TrimPrefix(mergeName, "refs/heads/")

	return fmt.Sprintf("%s/%s", remoteName, mergeName), nil
}

// SetBranchUpstream sets the upstream tracking branch
func SetBranchUpstream(name, upstream string) error {
	cmd := exec.Command("git", "branch", "--set-upstream-to", upstream, name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set upstream: %s: %w", string(output), err)
	}
	return nil
}

// IsBranchMerged checks if a branch has been merged into the default branch
func IsBranchMerged(name string) (bool, error) {
	defaultBranch, err := GetDefaultBranch()
	if err != nil {
		return false, err
	}

	cmd := exec.Command("git", "branch", "--merged", defaultBranch)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check merged branches: %w", err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		branch := strings.TrimSpace(line)
		// Strip "* " (current branch) or "+ " (checked out in worktree) prefixes
		branch = strings.TrimPrefix(branch, "* ")
		branch = strings.TrimPrefix(branch, "+ ")
		if branch == name {
			return true, nil
		}
	}

	return false, nil
}

// PushBranch pushes a branch to the remote
func PushBranch(name string, setUpstream bool) error {
	args := []string{"push"}
	if setUpstream {
		args = append(args, "-u", "origin", name)
	} else {
		args = append(args, "origin", name)
	}

	cmd := exec.Command("git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push branch: %s: %w", string(output), err)
	}
	return nil
}

// FetchBranch fetches a specific branch from the remote
func FetchBranch(name string) error {
	cmd := exec.Command("git", "fetch", "origin", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch branch: %s: %w", string(output), err)
	}
	return nil
}

// ListBranches returns all local branches
func ListBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []string
	for _, line := range strings.Split(string(output), "\n") {
		branch := strings.TrimSpace(line)
		if branch != "" {
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

// HasUniqueCommits checks if a branch has commits not in the target branch
// Returns true if the branch has work that isn't in target
func HasUniqueCommits(branch, target string) (bool, error) {
	// Use git rev-list to count commits in branch that aren't in target
	cmd := exec.Command("git", "rev-list", "--count", fmt.Sprintf("%s..%s", target, branch))
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to count unique commits: %w", err)
	}

	count := strings.TrimSpace(string(output))
	// Return true if count > 0
	return count != "0" && count != "", nil
}

// HasUncommittedWork checks if a worktree has staged or unstaged changes
// worktreePath is the absolute path to the worktree directory
func HasUncommittedWork(worktreePath string) (bool, error) {
	// Run git status --porcelain in the worktree directory
	cmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check worktree status: %w", err)
	}

	// If output is non-empty, there are uncommitted changes
	return strings.TrimSpace(string(output)) != "", nil
}

// IsRemoteBranchGone checks if a branch's remote tracking branch no longer exists
// Note: This makes a network call. For batch operations, use GetGoneBranches() instead.
func IsRemoteBranchGone(branch string) (bool, error) {
	goneBranches, err := GetGoneBranches()
	if err != nil {
		return false, err
	}
	return goneBranches[branch], nil
}

// GetGoneBranches returns a set of local branch names whose upstream tracking
// branch has been deleted. This uses a local-only operation (git branch -vv)
// and is much faster than checking each branch individually with ls-remote.
// Note: You should run `git fetch --prune` first if you want up-to-date info.
func GetGoneBranches() (map[string]bool, error) {
	// git branch -vv shows tracking info including "[gone]" for deleted upstreams
	// Example output:
	//   feature-branch abc1234 [origin/feature-branch: gone] commit message
	//   main            def5678 [origin/main] commit message
	cmd := exec.Command("git", "branch", "-vv")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	goneBranches := make(map[string]bool)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove leading "* " or "+ " for current/worktree branches
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "+ ")

		// Extract branch name (first field)
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		branchName := fields[0]

		// Check if the line contains ": gone]" which indicates deleted upstream
		if strings.Contains(line, ": gone]") {
			goneBranches[branchName] = true
		}
	}

	return goneBranches, nil
}
