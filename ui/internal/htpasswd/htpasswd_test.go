package htpasswd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFile(t *testing.T) *File {
	t.Helper()
	dir := t.TempDir()
	return New(filepath.Join(dir, "htpasswd"))
}

func TestBootstrapCreatesFile(t *testing.T) {
	f := newTestFile(t)
	created, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)
	assert.True(t, created)

	// File should exist
	_, err = os.Stat(f.path)
	require.NoError(t, err)

	// Verify the admin user can authenticate
	ok, err := f.Verify("admin", "admin")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestBootstrapNoopIfExists(t *testing.T) {
	f := newTestFile(t)
	created, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)
	assert.True(t, created)

	// Second call should not recreate
	created, err = f.Bootstrap("admin", "different")
	require.NoError(t, err)
	assert.False(t, created)

	// Original password still works
	ok, err := f.Verify("admin", "admin")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyCorrectPassword(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "secret")
	require.NoError(t, err)

	ok, err := f.Verify("admin", "secret")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyWrongPassword(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "secret")
	require.NoError(t, err)

	ok, err := f.Verify("admin", "wrong")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerifyNonexistentUser(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "secret")
	require.NoError(t, err)

	ok, err := f.Verify("nobody", "secret")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerifyDisabledUser(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "secret")
	require.NoError(t, err)

	require.NoError(t, f.SetEnabled("admin", false))

	ok, err := f.Verify("admin", "secret")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAddUser(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)

	require.NoError(t, f.Add("alice", "pass123"))

	ok, err := f.Verify("alice", "pass123")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestAddDuplicateUser(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)

	err = f.Add("admin", "newpass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestSetPassword(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "old")
	require.NoError(t, err)

	require.NoError(t, f.SetPassword("admin", "new"))

	ok, err := f.Verify("admin", "old")
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = f.Verify("admin", "new")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestSetPasswordNonexistentUser(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)

	err = f.SetPassword("nobody", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSetEnabledDisable(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)

	require.NoError(t, f.SetEnabled("admin", false))

	entries, err := f.List()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Disabled)
}

func TestSetEnabledReEnable(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)

	require.NoError(t, f.SetEnabled("admin", false))
	require.NoError(t, f.SetEnabled("admin", true))

	ok, err := f.Verify("admin", "admin")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRemoveUser(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)
	require.NoError(t, f.Add("alice", "pass"))

	require.NoError(t, f.Remove("alice"))

	entries, err := f.List()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "admin", entries[0].Username)
}

func TestRemoveNonexistentUser(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)

	err = f.Remove("nobody")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserExists(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)

	exists, err := f.UserExists("admin")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = f.UserExists("nobody")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestListMultipleUsers(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)
	require.NoError(t, f.Add("alice", "pass1"))
	require.NoError(t, f.Add("bob", "pass2"))

	entries, err := f.List()
	require.NoError(t, err)
	require.Len(t, entries, 3)

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Username
	}
	assert.Contains(t, names, "admin")
	assert.Contains(t, names, "alice")
	assert.Contains(t, names, "bob")
}

func TestConcurrentAccess(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make(chan error, 20)

	// 10 concurrent verifies
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.Verify("admin", "admin")
			if err != nil {
				errs <- err
			}
		}()
	}

	// 10 concurrent adds (different users)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			err := f.Add(fmt.Sprintf("user%d", n), "pass")
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent operation failed: %v", err)
	}

	entries, err := f.List()
	require.NoError(t, err)
	assert.Len(t, entries, 11) // admin + 10 users
}

func TestAtomicWritePermissions(t *testing.T) {
	f := newTestFile(t)
	_, err := f.Bootstrap("admin", "admin")
	require.NoError(t, err)

	info, err := os.Stat(f.path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestEmptyFileReturnsNoEntries(t *testing.T) {
	f := newTestFile(t)
	// Don't bootstrap — file doesn't exist
	entries, err := f.List()
	require.NoError(t, err)
	assert.Empty(t, entries)
}
