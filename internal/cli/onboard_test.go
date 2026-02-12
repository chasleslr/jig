package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallSkillFiles(t *testing.T) {
	t.Run("installs skills globally", func(t *testing.T) {
		// Create temp home directory
		tmpHome := t.TempDir()
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpHome)
		defer os.Setenv("HOME", originalHome)

		err := InstallSkillFiles("global", false)
		if err != nil {
			t.Fatalf("InstallSkillFiles failed: %v", err)
		}

		// Check skills were installed in global location
		globalDir := filepath.Join(tmpHome, ".claude", "commands", "jig")
		planPath := filepath.Join(globalDir, "plan.md")
		implementPath := filepath.Join(globalDir, "implement.md")

		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			t.Error("plan.md should be installed globally")
		}

		if _, err := os.Stat(implementPath); os.IsNotExist(err) {
			t.Error("implement.md should be installed globally")
		}

		// Verify content
		content, err := os.ReadFile(planPath)
		if err != nil {
			t.Fatalf("failed to read installed plan.md: %v", err)
		}
		if len(content) == 0 {
			t.Error("installed plan.md should not be empty")
		}
	})

	t.Run("installs skills at project level", func(t *testing.T) {
		// Create temp project directory
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(originalDir)

		err := InstallSkillFiles("project", false)
		if err != nil {
			t.Fatalf("InstallSkillFiles failed: %v", err)
		}

		// Check skills were installed in project location
		projectDir := filepath.Join(tmpDir, ".claude", "commands", "jig")
		planPath := filepath.Join(projectDir, "plan.md")
		implementPath := filepath.Join(projectDir, "implement.md")

		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			t.Error("plan.md should be installed in project")
		}

		if _, err := os.Stat(implementPath); os.IsNotExist(err) {
			t.Error("implement.md should be installed in project")
		}
	})

	t.Run("does not overwrite existing files without force", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(originalDir)

		// Create existing file with custom content
		projectDir := filepath.Join(tmpDir, ".claude", "commands", "jig")
		os.MkdirAll(projectDir, 0755)
		customContent := []byte("# Custom plan")
		planPath := filepath.Join(projectDir, "plan.md")
		os.WriteFile(planPath, customContent, 0644)

		// Install without force
		err := InstallSkillFiles("project", false)
		if err != nil {
			t.Fatalf("InstallSkillFiles failed: %v", err)
		}

		// Check file was NOT overwritten
		content, _ := os.ReadFile(planPath)
		if string(content) != string(customContent) {
			t.Error("existing file should not be overwritten without force")
		}
	})

	t.Run("overwrites existing files with force", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(originalDir)

		// Create existing file with custom content
		projectDir := filepath.Join(tmpDir, ".claude", "commands", "jig")
		os.MkdirAll(projectDir, 0755)
		customContent := []byte("# Custom plan")
		planPath := filepath.Join(projectDir, "plan.md")
		os.WriteFile(planPath, customContent, 0644)

		// Install with force
		err := InstallSkillFiles("project", true)
		if err != nil {
			t.Fatalf("InstallSkillFiles failed: %v", err)
		}

		// Check file WAS overwritten
		content, _ := os.ReadFile(planPath)
		if string(content) == string(customContent) {
			t.Error("existing file should be overwritten with force")
		}

		// Should contain actual skill content
		if len(content) < 100 {
			t.Error("overwritten file should contain real skill content")
		}
	})

	t.Run("creates directory if it does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(originalDir)

		err := InstallSkillFiles("project", false)
		if err != nil {
			t.Fatalf("InstallSkillFiles failed: %v", err)
		}

		// Check directory was created
		projectDir := filepath.Join(tmpDir, ".claude", "commands", "jig")
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			t.Error("directory should be created")
		}
	})
}

func TestInstallClaudeSkills(t *testing.T) {
	t.Run("installs skills globally", func(t *testing.T) {
		// Create temp home directory
		tmpHome := t.TempDir()
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpHome)
		defer os.Setenv("HOME", originalHome)

		// Create temp project directory for hooks
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(originalDir)

		// Disable initForce for this test
		originalForce := initForce
		initForce = false
		defer func() { initForce = originalForce }()

		err := InstallClaudeSkills("global")
		if err != nil {
			t.Fatalf("InstallClaudeSkills failed: %v", err)
		}

		// Check skills were installed globally
		globalDir := filepath.Join(tmpHome, ".claude", "commands", "jig")
		planPath := filepath.Join(globalDir, "plan.md")
		implementPath := filepath.Join(globalDir, "implement.md")

		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			t.Error("plan.md should be installed globally")
		}

		if _, err := os.Stat(implementPath); os.IsNotExist(err) {
			t.Error("implement.md should be installed globally")
		}

		// Check hooks were set up in project
		settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
		if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
			t.Error("settings.json should be created")
		}
	})

	t.Run("installs skills at project level", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(originalDir)

		// Disable initForce for this test
		originalForce := initForce
		initForce = false
		defer func() { initForce = originalForce }()

		err := InstallClaudeSkills("project")
		if err != nil {
			t.Fatalf("InstallClaudeSkills failed: %v", err)
		}

		// Check skills were installed in project
		projectDir := filepath.Join(tmpDir, ".claude", "commands", "jig")
		planPath := filepath.Join(projectDir, "plan.md")
		implementPath := filepath.Join(projectDir, "implement.md")

		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			t.Error("plan.md should be installed in project")
		}

		if _, err := os.Stat(implementPath); os.IsNotExist(err) {
			t.Error("implement.md should be installed in project")
		}
	})
}
