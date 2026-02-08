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

func TestListWorktrees(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns main worktree for new repo", func(t *testing.T) {
		repo.InDir(func() {
			worktrees, err := ListWorktrees()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(worktrees) != 1 {
				t.Errorf("expected 1 worktree, got %d", len(worktrees))
			}

			if worktrees[0].Path != repo.Path {
				t.Errorf("expected main worktree path %s, got %s", repo.Path, worktrees[0].Path)
			}

			if worktrees[0].Branch != "main" {
				t.Errorf("expected main branch, got %s", worktrees[0].Branch)
			}
		})
	})

	t.Run("returns multiple worktrees", func(t *testing.T) {
		repo.CreateBranch("feature-wt")
		wtPath := repo.CreateWorktreeDir("feature-wt-dir", "feature-wt")
		defer os.RemoveAll(filepath.Dir(wtPath))

		repo.InDir(func() {
			worktrees, err := ListWorktrees()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(worktrees) != 2 {
				t.Errorf("expected 2 worktrees, got %d", len(worktrees))
			}

			foundFeature := false
			for _, wt := range worktrees {
				if wt.Branch == "feature-wt" {
					foundFeature = true
					if wt.Path != wtPath {
						t.Errorf("expected worktree path %s, got %s", wtPath, wt.Path)
					}
				}
			}
			if !foundFeature {
				t.Error("expected to find feature-wt worktree")
			}
		})
	})
}

func TestGetWorktree(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns nil for non-existent branch worktree", func(t *testing.T) {
		repo.InDir(func() {
			wt, err := GetWorktree("nonexistent")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if wt != nil {
				t.Error("expected nil worktree for nonexistent branch")
			}
		})
	})

	t.Run("returns worktree for existing branch", func(t *testing.T) {
		repo.CreateBranch("feature-get-wt")
		wtPath := repo.CreateWorktreeDir("get-wt-dir", "feature-get-wt")
		defer os.RemoveAll(filepath.Dir(wtPath))

		repo.InDir(func() {
			wt, err := GetWorktree("feature-get-wt")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if wt == nil {
				t.Fatal("expected worktree, got nil")
			}
			if wt.Branch != "feature-get-wt" {
				t.Errorf("expected branch feature-get-wt, got %s", wt.Branch)
			}
			if wt.Path != wtPath {
				t.Errorf("expected path %s, got %s", wtPath, wt.Path)
			}
		})
	})

	t.Run("returns main worktree", func(t *testing.T) {
		repo.InDir(func() {
			wt, err := GetWorktree("main")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if wt == nil {
				t.Fatal("expected main worktree, got nil")
			}
			if wt.Branch != "main" {
				t.Errorf("expected branch main, got %s", wt.Branch)
			}
		})
	})
}

func TestGetWorktreeByPath(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns nil for non-worktree path", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "jig-test-not-wt-*")
		defer os.RemoveAll(tmpDir)

		repo.InDir(func() {
			wt, err := GetWorktreeByPath(tmpDir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if wt != nil {
				t.Error("expected nil worktree for non-worktree path")
			}
		})
	})

	t.Run("returns worktree for valid path", func(t *testing.T) {
		repo.CreateBranch("feature-by-path")
		wtPath := repo.CreateWorktreeDir("by-path-dir", "feature-by-path")
		defer os.RemoveAll(filepath.Dir(wtPath))

		repo.InDir(func() {
			wt, err := GetWorktreeByPath(wtPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if wt == nil {
				t.Fatal("expected worktree, got nil")
			}
			if wt.Branch != "feature-by-path" {
				t.Errorf("expected branch feature-by-path, got %s", wt.Branch)
			}
		})
	})

	t.Run("returns main repo worktree", func(t *testing.T) {
		repo.InDir(func() {
			wt, err := GetWorktreeByPath(repo.Path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if wt == nil {
				t.Fatal("expected main worktree, got nil")
			}
			if wt.Branch != "main" {
				t.Errorf("expected branch main, got %s", wt.Branch)
			}
		})
	})
}

func TestRemoveWorktree(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("removes existing worktree", func(t *testing.T) {
		repo.CreateBranch("feature-remove")
		wtPath := repo.CreateWorktreeDir("remove-dir", "feature-remove")
		parentDir := filepath.Dir(wtPath)
		defer os.RemoveAll(parentDir)

		repo.InDir(func() {
			// Verify worktree exists
			wt, _ := GetWorktree("feature-remove")
			if wt == nil {
				t.Fatal("worktree should exist before removal")
			}

			err := RemoveWorktree(wtPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify worktree is removed
			wt, _ = GetWorktree("feature-remove")
			if wt != nil {
				t.Error("worktree should not exist after removal")
			}
		})
	})
}

func TestPruneWorktrees(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("prunes stale worktree references", func(t *testing.T) {
		repo.CreateBranch("feature-prune")
		wtPath := repo.CreateWorktreeDir("prune-dir", "feature-prune")
		parentDir := filepath.Dir(wtPath)

		// Manually remove the worktree directory (simulating stale reference)
		os.RemoveAll(parentDir)

		repo.InDir(func() {
			// Prune should not error
			err := PruneWorktrees()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
}

func TestIsInWorktree(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns true in git repo", func(t *testing.T) {
		repo.InDir(func() {
			if !IsInWorktree() {
				t.Error("expected to be in worktree")
			}
		})
	})

	t.Run("returns false outside git repo", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "jig-test-no-git-*")
		defer os.RemoveAll(tmpDir)

		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		if IsInWorktree() {
			t.Error("expected to not be in worktree")
		}
	})
}

func TestGetWorktreeRoot(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns repo root in main worktree", func(t *testing.T) {
		repo.InDir(func() {
			root, err := GetWorktreeRoot()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if root != repo.Path {
				t.Errorf("expected %s, got %s", repo.Path, root)
			}
		})
	})

	t.Run("returns worktree root in secondary worktree", func(t *testing.T) {
		repo.CreateBranch("feature-root")
		wtPath := repo.CreateWorktreeDir("root-dir", "feature-root")
		defer os.RemoveAll(filepath.Dir(wtPath))

		origDir, _ := os.Getwd()
		os.Chdir(wtPath)
		defer os.Chdir(origDir)

		root, err := GetWorktreeRoot()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != wtPath {
			t.Errorf("expected %s, got %s", wtPath, root)
		}
	})
}

func TestGetMainRepoRootWithWorktrees(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns main repo root from secondary worktree", func(t *testing.T) {
		repo.CreateBranch("feature-main-root")
		wtPath := repo.CreateWorktreeDir("main-root-dir", "feature-main-root")
		defer os.RemoveAll(filepath.Dir(wtPath))

		origDir, _ := os.Getwd()
		os.Chdir(wtPath)
		defer os.Chdir(origDir)

		root, err := GetMainRepoRoot()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != repo.Path {
			t.Errorf("expected main repo root %s, got %s", repo.Path, root)
		}
	})
}
