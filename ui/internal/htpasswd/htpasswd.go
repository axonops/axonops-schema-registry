// Package htpasswd manages a bcrypt-based htpasswd file with file locking.
package htpasswd

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/crypto/bcrypt"
)

const (
	// bcryptCost is the bcrypt work factor for password hashing.
	bcryptCost = 10
	// disabledPrefix marks a user as disabled in the htpasswd file.
	disabledPrefix = "#!"
)

// Entry represents a single user entry in the htpasswd file.
type Entry struct {
	Username string
	Hash     string
	Disabled bool
}

// File manages read/write access to an htpasswd file with file locking.
type File struct {
	path     string
	lockPath string
	mu       sync.Mutex // process-level mutex
}

// New creates a new File manager for the given path.
func New(path string) *File {
	return &File{
		path:     path,
		lockPath: path + ".lock",
	}
}

// Bootstrap creates the htpasswd file with a default admin user if it doesn't exist.
// Returns true if the file was created.
func (f *File) Bootstrap(username, password string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, err := os.Stat(f.path); err == nil {
		return false, nil
	}

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("creating htpasswd directory: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return false, fmt.Errorf("hashing password: %w", err)
	}

	content := fmt.Sprintf("# Schema Registry UI Users\n%s:%s\n", username, string(hash))
	if err := f.atomicWrite(content); err != nil {
		return false, err
	}

	slog.Warn("htpasswd file created with default user — change the password immediately",
		"file", f.path, "user", username)
	return true, nil
}

// Verify checks if the username/password combination is valid.
// Returns false if the user doesn't exist, is disabled, or the password is wrong.
func (f *File) Verify(username, password string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.readWithLock(false)
	if err != nil {
		return false, err
	}

	for _, e := range entries {
		if e.Username == username {
			if e.Disabled {
				return false, nil
			}
			return bcrypt.CompareHashAndPassword([]byte(e.Hash), []byte(password)) == nil, nil
		}
	}
	return false, nil
}

// List returns all user entries.
func (f *File) List() ([]Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.readWithLock(false)
}

// Add creates a new user. Returns error if the user already exists.
func (f *File) Add(username, password string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.readWithLock(true)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Username == username {
			return fmt.Errorf("user %q already exists", username)
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	entries = append(entries, Entry{Username: username, Hash: string(hash), Disabled: false})
	return f.writeEntries(entries)
}

// SetPassword updates the password for an existing user.
func (f *File) SetPassword(username, password string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.readWithLock(true)
	if err != nil {
		return err
	}

	found := false
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	for i, e := range entries {
		if e.Username == username {
			entries[i].Hash = string(hash)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("user %q not found", username)
	}

	return f.writeEntries(entries)
}

// SetEnabled enables or disables a user.
func (f *File) SetEnabled(username string, enabled bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.readWithLock(true)
	if err != nil {
		return err
	}

	found := false
	for i, e := range entries {
		if e.Username == username {
			entries[i].Disabled = !enabled
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("user %q not found", username)
	}

	return f.writeEntries(entries)
}

// Remove deletes a user from the file.
func (f *File) Remove(username string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.readWithLock(true)
	if err != nil {
		return err
	}

	newEntries := make([]Entry, 0, len(entries))
	found := false
	for _, e := range entries {
		if e.Username == username {
			found = true
			continue
		}
		newEntries = append(newEntries, e)
	}

	if !found {
		return fmt.Errorf("user %q not found", username)
	}

	return f.writeEntries(newEntries)
}

// UserExists returns true if the username exists (enabled or disabled).
func (f *File) UserExists(username string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.readWithLock(false)
	if err != nil {
		return false, err
	}

	for _, e := range entries {
		if e.Username == username {
			return true, nil
		}
	}
	return false, nil
}

// readWithLock reads the htpasswd file with a shared (read) or exclusive (write) file lock.
func (f *File) readWithLock(exclusive bool) ([]Entry, error) {
	file, err := os.OpenFile(f.lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}
	defer file.Close()

	lockType := syscall.LOCK_SH
	if exclusive {
		lockType = syscall.LOCK_EX
	}
	if err := syscall.Flock(int(file.Fd()), lockType); err != nil {
		return nil, fmt.Errorf("acquiring file lock: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	return f.parseFile()
}

// parseFile reads and parses the htpasswd file.
func (f *File) parseFile() ([]Entry, error) {
	data, err := os.Open(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening htpasswd file: %w", err)
	}
	defer data.Close()

	var entries []Entry
	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || (strings.HasPrefix(line, "#") && !strings.HasPrefix(line, disabledPrefix)) {
			continue
		}

		disabled := false
		if strings.HasPrefix(line, disabledPrefix) {
			disabled = true
			line = line[len(disabledPrefix):]
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		entries = append(entries, Entry{
			Username: strings.TrimSpace(parts[0]),
			Hash:     strings.TrimSpace(parts[1]),
			Disabled: disabled,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading htpasswd file: %w", err)
	}
	return entries, nil
}

// writeEntries writes all entries back to the htpasswd file atomically.
func (f *File) writeEntries(entries []Entry) error {
	var b strings.Builder
	b.WriteString("# Schema Registry UI Users\n")
	for _, e := range entries {
		if e.Disabled {
			b.WriteString(disabledPrefix)
		}
		b.WriteString(e.Username)
		b.WriteByte(':')
		b.WriteString(e.Hash)
		b.WriteByte('\n')
	}
	return f.atomicWrite(b.String())
}

// atomicWrite writes content to a temp file then renames it into place.
func (f *File) atomicWrite(content string) error {
	dir := filepath.Dir(f.path)
	tmp, err := os.CreateTemp(dir, ".htpasswd-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting file permissions: %w", err)
	}

	if err := os.Rename(tmpPath, f.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}
