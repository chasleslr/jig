package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Store handles secure credential storage
// For MVP, we use a simple file-based approach
// Future: integrate with system keychain (keyring, Keychain Access, etc.)
type Store struct {
	path string
}

// Credentials holds sensitive API keys and tokens
type Credentials struct {
	LinearAPIKey string `json:"linear_api_key,omitempty"`
}

// NewStore creates a new credential store
func NewStore() (*Store, error) {
	jigDir, err := JigDir()
	if err != nil {
		return nil, err
	}

	credPath := filepath.Join(jigDir, ".credentials")
	return &Store{path: credPath}, nil
}

// Load reads credentials from the store
func (s *Store) Load() (*Credentials, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Credentials{}, nil
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// Save writes credentials to the store
func (s *Store) Save(creds *Credentials) error {
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	// Write with restrictive permissions
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// SetLinearAPIKey stores the Linear API key
func (s *Store) SetLinearAPIKey(key string) error {
	creds, err := s.Load()
	if err != nil {
		creds = &Credentials{}
	}
	creds.LinearAPIKey = key
	return s.Save(creds)
}

// GetLinearAPIKey retrieves the Linear API key
func (s *Store) GetLinearAPIKey() (string, error) {
	creds, err := s.Load()
	if err != nil {
		return "", err
	}
	return creds.LinearAPIKey, nil
}

// Clear removes all stored credentials
func (s *Store) Clear() error {
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove credentials: %w", err)
	}
	return nil
}
