package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStoreWithPath(t *testing.T) {
	store := NewStoreWithPath("/custom/path/.credentials")
	if store == nil {
		t.Fatal("NewStoreWithPath returned nil")
	}
	if store.path != "/custom/path/.credentials" {
		t.Errorf("store.path = %q, want %q", store.path, "/custom/path/.credentials")
	}
}

func TestStore_LoadEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, ".credentials")
	store := NewStoreWithPath(credPath)

	// Load when file doesn't exist should return empty credentials
	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if creds == nil {
		t.Fatal("Load() returned nil")
	}
	if creds.LinearAPIKey != "" {
		t.Errorf("LinearAPIKey should be empty, got %q", creds.LinearAPIKey)
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, ".credentials")
	store := NewStoreWithPath(credPath)

	// Save credentials
	creds := &Credentials{
		LinearAPIKey: "test-api-key-12345",
	}
	if err := store.Save(creds); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(credPath)
	if err != nil {
		t.Fatalf("failed to stat credentials file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions = %o, want %o", info.Mode().Perm(), 0600)
	}

	// Load credentials back
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.LinearAPIKey != "test-api-key-12345" {
		t.Errorf("LinearAPIKey = %q, want %q", loaded.LinearAPIKey, "test-api-key-12345")
	}
}

func TestStore_SetAndGetLinearAPIKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, ".credentials")
	store := NewStoreWithPath(credPath)

	// Set API key
	if err := store.SetLinearAPIKey("my-api-key"); err != nil {
		t.Fatalf("SetLinearAPIKey() error = %v", err)
	}

	// Get API key
	key, err := store.GetLinearAPIKey()
	if err != nil {
		t.Fatalf("GetLinearAPIKey() error = %v", err)
	}
	if key != "my-api-key" {
		t.Errorf("GetLinearAPIKey() = %q, want %q", key, "my-api-key")
	}
}

func TestStore_GetLinearAPIKeyEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, ".credentials")
	store := NewStoreWithPath(credPath)

	// Get API key from non-existent file
	key, err := store.GetLinearAPIKey()
	if err != nil {
		t.Fatalf("GetLinearAPIKey() error = %v", err)
	}
	if key != "" {
		t.Errorf("GetLinearAPIKey() should return empty string, got %q", key)
	}
}

func TestStore_Clear(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, ".credentials")
	store := NewStoreWithPath(credPath)

	// Save credentials first
	if err := store.SetLinearAPIKey("my-api-key"); err != nil {
		t.Fatalf("SetLinearAPIKey() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		t.Fatal("credentials file should exist")
	}

	// Clear credentials
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(credPath); !os.IsNotExist(err) {
		t.Error("credentials file should be removed after Clear()")
	}
}

func TestStore_ClearNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, ".credentials")
	store := NewStoreWithPath(credPath)

	// Clear when file doesn't exist should not error
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() should not error for non-existent file, got: %v", err)
	}
}

func TestStore_LoadInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, ".credentials")

	// Write invalid JSON
	if err := os.WriteFile(credPath, []byte("not valid json{"), 0600); err != nil {
		t.Fatalf("failed to write invalid json: %v", err)
	}

	store := NewStoreWithPath(credPath)

	_, err = store.Load()
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestStore_SetLinearAPIKeyPreservesExisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	credPath := filepath.Join(tmpDir, ".credentials")
	store := NewStoreWithPath(credPath)

	// Set initial key
	if err := store.SetLinearAPIKey("first-key"); err != nil {
		t.Fatalf("SetLinearAPIKey() error = %v", err)
	}

	// Update key
	if err := store.SetLinearAPIKey("second-key"); err != nil {
		t.Fatalf("SetLinearAPIKey() error = %v", err)
	}

	// Verify updated key
	key, err := store.GetLinearAPIKey()
	if err != nil {
		t.Fatalf("GetLinearAPIKey() error = %v", err)
	}
	if key != "second-key" {
		t.Errorf("GetLinearAPIKey() = %q, want %q", key, "second-key")
	}
}

func TestStore_SaveCreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "jig-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a nested path that doesn't exist
	credPath := filepath.Join(tmpDir, "nested", "dir", ".credentials")
	store := NewStoreWithPath(credPath)

	// Save should create the directory
	if err := store.SetLinearAPIKey("test-key"); err != nil {
		t.Fatalf("SetLinearAPIKey() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		t.Error("credentials file should be created")
	}
}
