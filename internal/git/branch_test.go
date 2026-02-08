package git

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lowercase",
			input:    "hello world",
			expected: "hello-world",
		},
		{
			name:     "uppercase to lowercase",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "underscores to hyphens",
			input:    "hello_world",
			expected: "hello-world",
		},
		{
			name:     "removes special characters",
			input:    "hello@world!",
			expected: "helloworld",
		},
		{
			name:     "collapses multiple hyphens",
			input:    "hello---world",
			expected: "hello-world",
		},
		{
			name:     "trims leading hyphens",
			input:    "---hello",
			expected: "hello",
		},
		{
			name:     "trims trailing hyphens",
			input:    "hello---",
			expected: "hello",
		},
		{
			name:     "truncates long strings",
			input:    "this is a very long title that should be truncated to fifty characters or less",
			expected: "this-is-a-very-long-title-that-should-be-truncated",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special characters",
			input:    "!@#$%",
			expected: "",
		},
		{
			name:     "preserves numbers",
			input:    "issue 123",
			expected: "issue-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBranchExists(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns true for existing branch", func(t *testing.T) {
		repo.InDir(func() {
			exists, err := BranchExists("main")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !exists {
				t.Error("expected main branch to exist")
			}
		})
	})

	t.Run("returns false for non-existing branch", func(t *testing.T) {
		repo.InDir(func() {
			exists, err := BranchExists("nonexistent")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exists {
				t.Error("expected nonexistent branch to not exist")
			}
		})
	})

	t.Run("returns true for newly created branch", func(t *testing.T) {
		repo.CreateBranch("feature-branch")
		repo.InDir(func() {
			exists, err := BranchExists("feature-branch")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !exists {
				t.Error("expected feature-branch to exist")
			}
		})
	})
}

func TestGetCurrentBranch(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns main on new repo", func(t *testing.T) {
		repo.InDir(func() {
			branch, err := GetCurrentBranch()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if branch != "main" {
				t.Errorf("expected main, got %s", branch)
			}
		})
	})

	t.Run("returns correct branch after checkout", func(t *testing.T) {
		repo.CreateBranch("develop")
		repo.Checkout("develop")
		repo.InDir(func() {
			branch, err := GetCurrentBranch()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if branch != "develop" {
				t.Errorf("expected develop, got %s", branch)
			}
		})
	})
}

func TestCreateBranch(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("creates branch from HEAD", func(t *testing.T) {
		repo.InDir(func() {
			err := CreateBranch("new-branch", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			exists, err := BranchExists("new-branch")
			if err != nil {
				t.Fatalf("unexpected error checking branch: %v", err)
			}
			if !exists {
				t.Error("expected new-branch to exist after creation")
			}
		})
	})

	t.Run("creates branch from specific commit", func(t *testing.T) {
		// Make a new commit on a different branch
		repo.CreateBranch("base")
		repo.Checkout("base")
		repo.WriteFile("base.txt", "base content")
		baseSHA := repo.Commit("base commit")
		repo.Checkout("main")

		repo.InDir(func() {
			err := CreateBranch("from-base", baseSHA)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the branch points to the correct commit
			output := repo.Git("rev-parse", "from-base")
			if output != baseSHA {
				t.Errorf("expected from-base to point to %s, got %s", baseSHA, output)
			}
		})
	})

	t.Run("fails for duplicate branch name", func(t *testing.T) {
		repo.InDir(func() {
			err := CreateBranch("main", "")
			if err == nil {
				t.Error("expected error when creating duplicate branch")
			}
		})
	})
}

func TestDeleteBranch(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("deletes merged branch", func(t *testing.T) {
		// Create and checkout feature branch
		repo.CreateBranch("feature-to-delete")
		repo.Checkout("feature-to-delete")
		repo.WriteFile("feature.txt", "feature content")
		repo.Commit("feature commit")

		// Merge into main
		repo.Checkout("main")
		repo.MergeBranch("feature-to-delete")

		repo.InDir(func() {
			err := DeleteBranch("feature-to-delete", false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			exists, _ := BranchExists("feature-to-delete")
			if exists {
				t.Error("branch should have been deleted")
			}
		})
	})

	t.Run("fails to delete unmerged branch without force", func(t *testing.T) {
		repo.CreateBranch("unmerged-branch")
		repo.Checkout("unmerged-branch")
		repo.WriteFile("unmerged.txt", "unmerged content")
		repo.Commit("unmerged commit")
		repo.Checkout("main")

		repo.InDir(func() {
			err := DeleteBranch("unmerged-branch", false)
			if err == nil {
				t.Error("expected error when deleting unmerged branch without force")
			}
		})
	})

	t.Run("force deletes unmerged branch", func(t *testing.T) {
		repo.CreateBranch("force-delete-branch")
		repo.Checkout("force-delete-branch")
		repo.WriteFile("force.txt", "force content")
		repo.Commit("force commit")
		repo.Checkout("main")

		repo.InDir(func() {
			err := DeleteBranch("force-delete-branch", true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			exists, _ := BranchExists("force-delete-branch")
			if exists {
				t.Error("branch should have been force deleted")
			}
		})
	})

	t.Run("fails to delete current branch", func(t *testing.T) {
		repo.InDir(func() {
			err := DeleteBranch("main", true)
			if err == nil {
				t.Error("expected error when deleting current branch")
			}
		})
	})
}

func TestIsBranchMerged(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns true for merged branch", func(t *testing.T) {
		// Create feature branch with commit
		repo.CreateBranch("merged-feature")
		repo.Checkout("merged-feature")
		repo.WriteFile("merged.txt", "merged content")
		repo.Commit("merged commit")

		// Merge into main
		repo.Checkout("main")
		repo.MergeBranch("merged-feature")

		repo.InDir(func() {
			merged, err := IsBranchMerged("merged-feature")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !merged {
				t.Error("expected branch to be merged")
			}
		})
	})

	t.Run("returns false for unmerged branch", func(t *testing.T) {
		repo.CreateBranch("unmerged-feature")
		repo.Checkout("unmerged-feature")
		repo.WriteFile("unmerged.txt", "unmerged content")
		repo.Commit("unmerged commit")
		repo.Checkout("main")

		repo.InDir(func() {
			merged, err := IsBranchMerged("unmerged-feature")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if merged {
				t.Error("expected branch to not be merged")
			}
		})
	})
}

func TestGetDefaultBranch(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("returns main when main exists", func(t *testing.T) {
		repo.InDir(func() {
			branch, err := GetDefaultBranch()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if branch != "main" {
				t.Errorf("expected main, got %s", branch)
			}
		})
	})
}

func TestListBranches(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("lists all branches", func(t *testing.T) {
		repo.CreateBranch("branch-a")
		repo.CreateBranch("branch-b")

		repo.InDir(func() {
			branches, err := ListBranches()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expected := map[string]bool{
				"main":     true,
				"branch-a": true,
				"branch-b": true,
			}

			if len(branches) != len(expected) {
				t.Errorf("expected %d branches, got %d: %v", len(expected), len(branches), branches)
			}

			for _, branch := range branches {
				if !expected[branch] {
					t.Errorf("unexpected branch: %s", branch)
				}
			}
		})
	})

	t.Run("returns single branch for new repo", func(t *testing.T) {
		newRepo := NewTestRepo(t)
		defer newRepo.Cleanup()

		newRepo.InDir(func() {
			branches, err := ListBranches()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(branches) != 1 {
				t.Errorf("expected 1 branch, got %d: %v", len(branches), branches)
			}
			if branches[0] != "main" {
				t.Errorf("expected main, got %s", branches[0])
			}
		})
	})
}

func TestCheckoutBranch(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("switches to existing branch", func(t *testing.T) {
		repo.CreateBranch("checkout-test")

		repo.InDir(func() {
			err := CheckoutBranch("checkout-test")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			current, _ := GetCurrentBranch()
			if current != "checkout-test" {
				t.Errorf("expected checkout-test, got %s", current)
			}
		})
	})

	t.Run("fails for non-existing branch", func(t *testing.T) {
		repo.InDir(func() {
			err := CheckoutBranch("nonexistent")
			if err == nil {
				t.Error("expected error when checking out non-existing branch")
			}
		})
	})
}
