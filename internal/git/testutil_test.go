package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// chdirMu protects against concurrent os.Chdir calls in tests.
// Tests that use InDir() should not be run with t.Parallel().
var chdirMu sync.Mutex

// TestRepo is a temporary git repository for testing
type TestRepo struct {
	t    *testing.T
	Path string
}

// NewTestRepo creates an isolated git repo with an initial commit
func NewTestRepo(t *testing.T) *TestRepo {
	t.Helper()

	// Create temp directory with jig-test prefix
	tmpDir, err := os.MkdirTemp("", "jig-test-git-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Resolve symlinks (macOS /tmp -> /private/tmp)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	r := &TestRepo{t: t, Path: tmpDir}

	// Initialize git repo
	r.Git("init")

	// Configure git user (required for commits)
	r.Git("config", "user.email", "test@jig.dev")
	r.Git("config", "user.name", "Jig Test")

	// Create initial commit (required for worktrees to work)
	r.WriteFile("README.md", "# Test Repository\n")
	r.Git("add", ".")
	r.Git("commit", "-m", "Initial commit")

	// Rename default branch to main for consistency
	r.Git("branch", "-M", "main")

	return r
}

// Cleanup removes the test repository
func (r *TestRepo) Cleanup() {
	r.t.Helper()

	// Clean up any worktrees first to avoid issues
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = r.Path
	output, _ := cmd.Output()

	// Parse worktree paths and remove non-main ones
	var worktreePaths []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			wtPath := strings.TrimPrefix(line, "worktree ")
			if wtPath != r.Path {
				worktreePaths = append(worktreePaths, wtPath)
			}
		}
	}

	for _, wtPath := range worktreePaths {
		cmd := exec.Command("git", "worktree", "remove", wtPath, "--force")
		cmd.Dir = r.Path
		cmd.Run()
	}

	os.RemoveAll(r.Path)
}

// Git runs a git command and returns output, failing the test on error
func (r *TestRepo) Git(args ...string) string {
	r.t.Helper()
	output, err := r.GitMayFail(args...)
	if err != nil {
		r.t.Fatalf("git %s failed: %v\nOutput: %s", strings.Join(args, " "), err, output)
	}
	return output
}

// GitMayFail runs a git command and returns output and error
func (r *TestRepo) GitMayFail(args ...string) (string, error) {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Path
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// WriteFile creates or overwrites a file in the repo
func (r *TestRepo) WriteFile(relPath, content string) {
	r.t.Helper()
	fullPath := filepath.Join(r.Path, relPath)

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		r.t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		r.t.Fatalf("failed to write file %s: %v", relPath, err)
	}
}

// Commit stages all changes and creates a commit, returning the SHA
func (r *TestRepo) Commit(message string) string {
	r.t.Helper()
	r.Git("add", ".")
	r.Git("commit", "-m", message)
	return r.Git("rev-parse", "HEAD")
}

// CreateBranch creates a new branch at the current HEAD
func (r *TestRepo) CreateBranch(name string) {
	r.t.Helper()
	r.Git("branch", name)
}

// Checkout switches to a branch
func (r *TestRepo) Checkout(name string) {
	r.t.Helper()
	r.Git("checkout", name)
}

// MergeBranch merges a branch into the current branch
func (r *TestRepo) MergeBranch(name string) {
	r.t.Helper()
	r.Git("merge", name, "--no-edit")
}

// InDir runs a function with the repo as the current working directory.
// This is required because the git functions in branch.go and worktree.go
// operate on the current working directory.
//
// WARNING: Tests using InDir() must NOT use t.Parallel() as they mutate
// the process's current working directory. A mutex is used to serialize
// InDir calls but this doesn't protect against other concurrent directory
// changes.
func (r *TestRepo) InDir(fn func()) {
	r.t.Helper()

	// Serialize directory changes to avoid races between tests
	chdirMu.Lock()
	defer chdirMu.Unlock()

	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		r.t.Fatalf("failed to get current directory: %v", err)
	}

	// Change to repo directory
	if err := os.Chdir(r.Path); err != nil {
		r.t.Fatalf("failed to change to repo directory: %v", err)
	}

	// Restore original directory on exit
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			r.t.Fatalf("failed to restore original directory: %v", err)
		}
	}()

	fn()
}

// CreateWorktreeDir creates a worktree at a specified path for a branch
func (r *TestRepo) CreateWorktreeDir(path, branch string) string {
	r.t.Helper()

	// If path is relative, make it relative to a temp directory
	if !filepath.IsAbs(path) {
		tmpDir, err := os.MkdirTemp("", "jig-test-wt-*")
		if err != nil {
			r.t.Fatalf("failed to create temp dir for worktree: %v", err)
		}
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)
		path = filepath.Join(tmpDir, path)
	}

	r.Git("worktree", "add", path, branch)
	return path
}

// ListWorktreePaths returns paths of all worktrees
func (r *TestRepo) ListWorktreePaths() []string {
	r.t.Helper()

	output := r.Git("worktree", "list", "--porcelain")

	var paths []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			paths = append(paths, strings.TrimPrefix(line, "worktree "))
		}
	}
	return paths
}
