// Package users provides user management functionality.
package users

import (
	"fmt"
	"unicode/utf8"

	"github.com/axonops/schema-registry-ui/internal/htpasswd"
)

const (
	minUsernameLen = 2
	maxUsernameLen = 64
	minPasswordLen = 4
	maxPasswordLen = 128
)

// UserInfo represents public user information (no hash).
type UserInfo struct {
	Username string `json:"username"`
	Enabled  bool   `json:"enabled"`
}

// Service provides user management operations.
type Service struct {
	htpasswd *htpasswd.File
}

// NewService creates a new user management service.
func NewService(hp *htpasswd.File) *Service {
	return &Service{htpasswd: hp}
}

// List returns all users.
func (s *Service) List() ([]UserInfo, error) {
	entries, err := s.htpasswd.List()
	if err != nil {
		return nil, err
	}

	users := make([]UserInfo, len(entries))
	for i, e := range entries {
		users[i] = UserInfo{
			Username: e.Username,
			Enabled:  !e.Disabled,
		}
	}
	return users, nil
}

// Create adds a new user with the given username and password.
func (s *Service) Create(username, password string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	return s.htpasswd.Add(username, password)
}

// SetPassword changes a user's password.
func (s *Service) SetPassword(username, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	return s.htpasswd.SetPassword(username, password)
}

// SetEnabled enables or disables a user.
func (s *Service) SetEnabled(username string, enabled bool) error {
	// Prevent disabling the last active user
	if !enabled {
		entries, err := s.htpasswd.List()
		if err != nil {
			return err
		}
		activeCount := 0
		for _, e := range entries {
			if !e.Disabled {
				activeCount++
			}
		}
		if activeCount <= 1 {
			return fmt.Errorf("cannot disable the last active user")
		}
	}
	return s.htpasswd.SetEnabled(username, enabled)
}

// Delete removes a user.
func (s *Service) Delete(username string) error {
	entries, err := s.htpasswd.List()
	if err != nil {
		return err
	}

	// Check existence first
	found := false
	for _, e := range entries {
		if e.Username == username {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user %q not found", username)
	}

	// Prevent deleting the last user
	if len(entries) <= 1 {
		return fmt.Errorf("cannot delete the last user")
	}

	return s.htpasswd.Remove(username)
}

// Exists returns true if the user exists.
func (s *Service) Exists(username string) (bool, error) {
	return s.htpasswd.UserExists(username)
}

func validateUsername(username string) error {
	n := utf8.RuneCountInString(username)
	if n < minUsernameLen || n > maxUsernameLen {
		return fmt.Errorf("username must be %d-%d characters", minUsernameLen, maxUsernameLen)
	}
	for _, r := range username {
		if !isValidUsernameRune(r) {
			return fmt.Errorf("username contains invalid character: %c", r)
		}
	}
	return nil
}

func isValidUsernameRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.'
}

func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	if n > maxPasswordLen {
		return fmt.Errorf("password must be at most %d characters", maxPasswordLen)
	}
	return nil
}
