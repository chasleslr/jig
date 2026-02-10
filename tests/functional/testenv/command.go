package testenv

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	// jigBinaryPath caches the built jig binary path
	jigBinaryPath string
	buildOnce     sync.Once
	buildErr      error
)

// CommandResult holds the result of a jig command execution.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Success returns true if the command exited with code 0.
func (r *CommandResult) Success() bool {
	return r.ExitCode == 0
}

// Combined returns stdout and stderr combined.
func (r *CommandResult) Combined() string {
	if r.Stderr == "" {
		return r.Stdout
	}
	if r.Stdout == "" {
		return r.Stderr
	}
	return r.Stdout + "\n" + r.Stderr
}

// buildJigBinary builds the jig binary once for all tests
func buildJigBinary(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		projectRoot := findProjectRoot(t)

		// Create a temp file for the binary
		tmpDir, err := os.MkdirTemp("", "jig-test-bin-*")
		if err != nil {
			buildErr = fmt.Errorf("failed to create temp dir for binary: %w", err)
			return
		}

		jigBinaryPath = filepath.Join(tmpDir, "jig")

		// Build the binary
		cmd := exec.Command("go", "build", "-o", jigBinaryPath, "./cmd/jig")
		cmd.Dir = projectRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("failed to build jig binary: %w\nOutput: %s", err, output)
			return
		}
	})

	if buildErr != nil {
		t.Fatal(buildErr)
	}

	return jigBinaryPath
}

// RunJig executes the jig command with the given arguments.
func (e *TestEnv) RunJig(args ...string) *CommandResult {
	e.t.Helper()
	return e.RunJigWithStdin("", args...)
}

// RunJigWithStdin executes the jig command with stdin content.
func (e *TestEnv) RunJigWithStdin(stdin string, args ...string) *CommandResult {
	e.t.Helper()

	binaryPath := buildJigBinary(e.t)

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = e.RepoDir

	// Set up environment with isolated JIG_HOME
	cmd.Env = append(os.Environ(), fmt.Sprintf("JIG_HOME=%s", e.JigHome))

	// Set up stdin if provided
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	result := &CommandResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}

	// Extract exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			// Some other error (e.g., command not found)
			result.ExitCode = -1
			result.Stderr = err.Error()
		}
	}

	return result
}

// RunJigInDir executes the jig command in a specific directory.
func (e *TestEnv) RunJigInDir(dir string, args ...string) *CommandResult {
	e.t.Helper()

	binaryPath := buildJigBinary(e.t)

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir

	// Set up environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("JIG_HOME=%s", e.JigHome))

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &CommandResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
			result.Stderr = err.Error()
		}
	}

	return result
}

// AssertSuccess fails the test if the command did not succeed.
func (e *TestEnv) AssertSuccess(result *CommandResult) {
	e.t.Helper()
	if !result.Success() {
		e.t.Fatalf("expected command to succeed, got exit code %d\nStdout: %s\nStderr: %s",
			result.ExitCode, result.Stdout, result.Stderr)
	}
}

// AssertFailure fails the test if the command succeeded.
func (e *TestEnv) AssertFailure(result *CommandResult) {
	e.t.Helper()
	if result.Success() {
		e.t.Fatalf("expected command to fail, but it succeeded\nStdout: %s\nStderr: %s",
			result.Stdout, result.Stderr)
	}
}

// AssertOutputContains fails if the combined output doesn't contain the expected string.
func (e *TestEnv) AssertOutputContains(result *CommandResult, expected string) {
	e.t.Helper()
	if !strings.Contains(result.Combined(), expected) {
		e.t.Errorf("expected output to contain %q, got:\n%s", expected, result.Combined())
	}
}

// AssertStdoutContains fails if stdout doesn't contain the expected string.
func (e *TestEnv) AssertStdoutContains(result *CommandResult, expected string) {
	e.t.Helper()
	if !strings.Contains(result.Stdout, expected) {
		e.t.Errorf("expected stdout to contain %q, got:\n%s", expected, result.Stdout)
	}
}

// AssertStderrContains fails if stderr doesn't contain the expected string.
func (e *TestEnv) AssertStderrContains(result *CommandResult, expected string) {
	e.t.Helper()
	if !strings.Contains(result.Stderr, expected) {
		e.t.Errorf("expected stderr to contain %q, got:\n%s", expected, result.Stderr)
	}
}

// findProjectRoot finds the jig project root by looking for go.mod
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from the test file's directory
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller information")
	}

	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// CopyFile copies a file from src to dst within the repo.
func (e *TestEnv) CopyFile(src, dst string) {
	e.t.Helper()

	srcPath := filepath.Join(e.RepoDir, src)
	dstPath := filepath.Join(e.RepoDir, dst)

	// Create parent directories if needed
	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		e.t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		e.t.Fatalf("failed to open source file %s: %v", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		e.t.Fatalf("failed to create destination file %s: %v", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		e.t.Fatalf("failed to copy file: %v", err)
	}
}
