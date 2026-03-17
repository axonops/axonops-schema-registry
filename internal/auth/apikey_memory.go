// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// MemoryAPIKeyStore holds API keys loaded from the config file.
// Keys are validated using bcrypt comparison against stored hashes.
type MemoryAPIKeyStore struct {
	keys []memoryAPIKey
}

type memoryAPIKey struct {
	name string
	hash string
	role string
}

// NewMemoryAPIKeyStore creates a store from config-defined API keys.
func NewMemoryAPIKeyStore(keys []config.ConfigAPIKey) (*MemoryAPIKeyStore, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no API keys defined in config")
	}

	store := &MemoryAPIKeyStore{
		keys: make([]memoryAPIKey, 0, len(keys)),
	}

	for i, k := range keys {
		if k.Name == "" {
			return nil, fmt.Errorf("API key at index %d has empty name", i)
		}
		if k.KeyHash == "" {
			return nil, fmt.Errorf("API key %q has empty key_hash", k.Name)
		}
		if !isBcryptHash(k.KeyHash) {
			return nil, fmt.Errorf("API key %q has unsupported hash format (only bcrypt is supported)", k.Name)
		}
		if k.Role == "" {
			return nil, fmt.Errorf("API key %q has empty role", k.Name)
		}
		store.keys = append(store.keys, memoryAPIKey{
			name: k.Name,
			hash: k.KeyHash,
			role: k.Role,
		})
	}

	return store, nil
}

// Validate checks a plaintext API key against all stored hashes.
// Returns the key name and role if a match is found.
func (s *MemoryAPIKeyStore) Validate(key string) (name, role string, ok bool) {
	for _, k := range s.keys {
		if bcrypt.CompareHashAndPassword([]byte(k.hash), []byte(key)) == nil {
			return k.name, k.role, true
		}
	}
	return "", "", false
}

// Count returns the number of keys in the store.
func (s *MemoryAPIKeyStore) Count() int {
	return len(s.keys)
}

// isBcryptHash checks if a string looks like a bcrypt hash.
func isBcryptHash(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") ||
		strings.HasPrefix(hash, "$2b$") ||
		strings.HasPrefix(hash, "$2y$")
}
