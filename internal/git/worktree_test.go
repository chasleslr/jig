package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsInsidePath(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDirRaw, err := os.MkdirTemp("", "jig-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDirRaw)

	// Resolve symlinks (on macOS, /tmp -> /private/tmp)
	tmpDir, err := filepath.EvalSymlinks(tmpDirRaw)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	// Create subdirectories
	subDir := filepath.Join(tmpDir, "subdir")
	nestedDir := filepath.Join(subDir, "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// Save original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get original dir: %v", err)
	}
	defer os.Chdir(origDir)

	t.Run("returns true when cwd equals target path", func(t *testing.T) {
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		inside, err := IsInsidePath(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !inside {
			t.Error("expected to be inside the target path")
		}
	})

	t.Run("returns true when cwd is subdirectory of target", func(t *testing.T) {
		if err := os.Chdir(subDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		inside, err := IsInsidePath(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !inside {
			t.Error("expected to be inside the target path")
		}
	})

	t.Run("returns true when cwd is nested subdirectory of target", func(t *testing.T) {
		if err := os.Chdir(nestedDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		inside, err := IsInsidePath(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !inside {
			t.Error("expected to be inside the target path")
		}
	})

	t.Run("returns false when cwd is outside target path", func(t *testing.T) {
		// Create another directory outside tmpDir
		otherDirRaw, err := os.MkdirTemp("", "jig-test-other-*")
		if err != nil {
			t.Fatalf("failed to create other temp dir: %v", err)
		}
		defer os.RemoveAll(otherDirRaw)

		// Resolve symlinks
		otherDir, err := filepath.EvalSymlinks(otherDirRaw)
		if err != nil {
			t.Fatalf("failed to resolve symlinks: %v", err)
		}

		if err := os.Chdir(otherDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		inside, err := IsInsidePath(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inside {
			t.Error("expected to be outside the target path")
		}
	})

	t.Run("returns false when paths share prefix but are different", func(t *testing.T) {
		// Create directories with similar prefixes
		dirA := filepath.Join(tmpDir, "foo")
		dirB := filepath.Join(tmpDir, "foobar")
		if err := os.MkdirAll(dirA, 0755); err != nil {
			t.Fatalf("failed to create dirA: %v", err)
		}
		if err := os.MkdirAll(dirB, 0755); err != nil {
			t.Fatalf("failed to create dirB: %v", err)
		}

		if err := os.Chdir(dirB); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		// dirB (/tmp/xxx/foobar) should NOT be considered inside dirA (/tmp/xxx/foo)
		inside, err := IsInsidePath(dirA)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inside {
			t.Error("foobar should not be considered inside foo")
		}
	})
}

func TestGetMainRepoRoot(t *testing.T) {
	// This test runs in the actual jig repository
	// We just verify it returns a valid path without error

	t.Run("returns valid path in git repository", func(t *testing.T) {
		root, err := GetMainRepoRoot()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it's a valid directory
		info, err := os.Stat(root)
		if err != nil {
			t.Fatalf("returned path does not exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("returned path is not a directory")
		}

		// Verify it contains a .git directory (it's the repo root)
		gitDir := filepath.Join(root, ".git")
		if _, err := os.Stat(gitDir); err != nil {
			t.Errorf("expected .git directory at %s", gitDir)
		}
	})
}
