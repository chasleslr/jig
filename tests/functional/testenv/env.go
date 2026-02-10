// Package testenv provides isolated test environments for functional tests.
package testenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnv represents an isolated test environment with its own JIG_HOME and git repo.
type TestEnv struct {
	t       *testing.T
	JigHome string // Isolated JIG_HOME directory
	RepoDir string // Isolated git repository
	origEnv string // Original JIG_HOME env value
}

// New creates a new isolated test environment.
// It creates a temp directory for JIG_HOME and initializes a git repository.
func New(t *testing.T) *TestEnv {
	t.Helper()

	// Create temp directory for JIG_HOME
	jigHome, err := os.MkdirTemp("", "jig-test-home-*")
	if err != nil {
		t.Fatalf("failed to create temp JIG_HOME: %v", err)
	}

	// Resolve symlinks (macOS /tmp -> /private/tmp)
	jigHome, err = filepath.EvalSymlinks(jigHome)
	if err != nil {
		os.RemoveAll(jigHome)
		t.Fatalf("failed to resolve JIG_HOME symlinks: %v", err)
	}

	// Create temp directory for git repo
	repoDir, err := os.MkdirTemp("", "jig-test-repo-*")
	if err != nil {
		os.RemoveAll(jigHome)
		t.Fatalf("failed to create temp repo: %v", err)
	}

	// Resolve symlinks
	repoDir, err = filepath.EvalSymlinks(repoDir)
	if err != nil {
		os.RemoveAll(jigHome)
		os.RemoveAll(repoDir)
		t.Fatalf("failed to resolve repo symlinks: %v", err)
	}

	env := &TestEnv{
		t:       t,
		JigHome: jigHome,
		RepoDir: repoDir,
		origEnv: os.Getenv("JIG_HOME"),
	}

	// Set JIG_HOME environment variable
	if err := os.Setenv("JIG_HOME", jigHome); err != nil {
		env.cleanup()
		t.Fatalf("failed to set JIG_HOME: %v", err)
	}

	// Initialize git repo
	env.initGitRepo()

	return env
}

// Cleanup removes the test environment and restores original state.
func (e *TestEnv) Cleanup() {
	e.t.Helper()
	e.cleanup()
}

func (e *TestEnv) cleanup() {
	// Restore original JIG_HOME
	if e.origEnv != "" {
		os.Setenv("JIG_HOME", e.origEnv)
	} else {
		os.Unsetenv("JIG_HOME")
	}

	// Clean up worktrees before removing directories
	e.cleanupWorktrees()

	// Remove temp directories
	os.RemoveAll(e.JigHome)
	os.RemoveAll(e.RepoDir)
}

// cleanupWorktrees removes any worktrees created during the test
func (e *TestEnv) cleanupWorktrees() {
	// List worktrees
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = e.RepoDir
	output, err := cmd.Output()
	if err != nil {
		return
	}

	// Parse worktree paths and remove non-main ones
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			wtPath := strings.TrimPrefix(line, "worktree ")
			if wtPath != e.RepoDir {
				removeCmd := exec.Command("git", "worktree", "remove", wtPath, "--force")
				removeCmd.Dir = e.RepoDir
				removeCmd.Run()
				// Also remove the parent directory if it exists
				os.RemoveAll(filepath.Dir(wtPath))
			}
		}
	}
}

// initGitRepo initializes a git repository with an initial commit
func (e *TestEnv) initGitRepo() {
	e.t.Helper()

	// Initialize git repo
	e.Git("init")

	// Configure git user
	e.Git("config", "user.email", "test@jig.dev")
	e.Git("config", "user.name", "Jig Test")

	// Create initial commit
	e.WriteFile("README.md", "# Test Repository\n")
	e.Git("add", ".")
	e.Git("commit", "-m", "Initial commit")

	// Rename default branch to main
	e.Git("branch", "-M", "main")
}

// Git runs a git command in the repo directory and fails the test on error.
func (e *TestEnv) Git(args ...string) string {
	e.t.Helper()
	output, err := e.GitMayFail(args...)
	if err != nil {
		e.t.Fatalf("git %s failed: %v\nOutput: %s", strings.Join(args, " "), err, output)
	}
	return output
}

// GitMayFail runs a git command and returns output and error.
func (e *TestEnv) GitMayFail(args ...string) (string, error) {
	e.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = e.RepoDir
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// WriteFile creates or overwrites a file in the repo.
func (e *TestEnv) WriteFile(relPath, content string) {
	e.t.Helper()
	fullPath := filepath.Join(e.RepoDir, relPath)

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		e.t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		e.t.Fatalf("failed to write file %s: %v", relPath, err)
	}
}

// ReadFile reads a file from the repo.
func (e *TestEnv) ReadFile(relPath string) string {
	e.t.Helper()
	fullPath := filepath.Join(e.RepoDir, relPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		e.t.Fatalf("failed to read file %s: %v", relPath, err)
	}
	return string(content)
}

// FileExists checks if a file exists in the repo.
func (e *TestEnv) FileExists(relPath string) bool {
	fullPath := filepath.Join(e.RepoDir, relPath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// Commit stages all changes and creates a commit.
func (e *TestEnv) Commit(message string) string {
	e.t.Helper()
	e.Git("add", ".")
	e.Git("commit", "-m", message)
	return e.Git("rev-parse", "HEAD")
}

// CreateBranch creates a new branch at the current HEAD.
func (e *TestEnv) CreateBranch(name string) {
	e.t.Helper()
	e.Git("branch", name)
}

// Checkout switches to a branch.
func (e *TestEnv) Checkout(name string) {
	e.t.Helper()
	e.Git("checkout", name)
}

// WriteJigConfig writes a config file to the JIG_HOME directory.
func (e *TestEnv) WriteJigConfig(content string) {
	e.t.Helper()
	configPath := filepath.Join(e.JigHome, "config.toml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		e.t.Fatalf("failed to write config: %v", err)
	}
}

// CacheDir returns the cache directory path.
func (e *TestEnv) CacheDir() string {
	return filepath.Join(e.JigHome, "cache")
}

// WorktreeDir returns the worktrees directory path.
func (e *TestEnv) WorktreeDir() string {
	return filepath.Join(e.JigHome, "worktrees")
}
