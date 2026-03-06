// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HTPasswdStore holds parsed htpasswd file entries for Basic authentication.
// Only bcrypt-hashed passwords ($2y$, $2a$, $2b$) are supported.
type HTPasswdStore struct {
	entries map[string]string // username -> bcrypt hash
}

// LoadHTPasswdFile reads and parses an Apache-style htpasswd file.
// Each line must be in the format: username:bcrypt_hash
// Lines starting with # and blank lines are ignored.
func LoadHTPasswdFile(filePath string) (*HTPasswdStore, error) {
	// #nosec G304 -- filePath is from trusted server configuration
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open htpasswd file: %w", err)
	}
	defer f.Close()

	store := &HTPasswdStore{
		entries: make(map[string]string),
	}

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("htpasswd line %d: invalid format (expected username:hash)", lineNum)
		}

		username := strings.TrimSpace(parts[0])
		hash := strings.TrimSpace(parts[1])

		if username == "" {
			return nil, fmt.Errorf("htpasswd line %d: empty username", lineNum)
		}

		// Only accept bcrypt hashes
		if !strings.HasPrefix(hash, "$2y$") && !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
			return nil, fmt.Errorf("htpasswd line %d: unsupported hash format for user %q (only bcrypt is supported)", lineNum, username)
		}

		store.entries[username] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read htpasswd file: %w", err)
	}

	if len(store.entries) == 0 {
		return nil, fmt.Errorf("htpasswd file contains no valid entries")
	}

	return store, nil
}

// Verify checks if the given username and password match an entry in the htpasswd store.
func (h *HTPasswdStore) Verify(username, password string) bool {
	hash, ok := h.entries[username]
	if !ok {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Count returns the number of entries in the htpasswd store.
func (h *HTPasswdStore) Count() int {
	return len(h.entries)
}
