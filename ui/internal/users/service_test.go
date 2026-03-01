package users

import (
	"path/filepath"
	"testing"

	"github.com/axonops/schema-registry-ui/internal/htpasswd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	hp := htpasswd.New(filepath.Join(dir, "htpasswd"))
	_, err := hp.Bootstrap("admin", "admin")
	require.NoError(t, err)
	return NewService(hp)
}

func TestListUsers(t *testing.T) {
	svc := newTestService(t)
	users, err := svc.List()
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "admin", users[0].Username)
	assert.True(t, users[0].Enabled)
}

func TestCreateUser(t *testing.T) {
	svc := newTestService(t)
	require.NoError(t, svc.Create("alice", "pass1234"))

	users, err := svc.List()
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestCreateDuplicateUser(t *testing.T) {
	svc := newTestService(t)
	err := svc.Create("admin", "pass1234")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCreateUserShortUsername(t *testing.T) {
	svc := newTestService(t)
	err := svc.Create("a", "pass1234")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2-64 characters")
}

func TestCreateUserInvalidChars(t *testing.T) {
	svc := newTestService(t)
	err := svc.Create("user@name", "pass1234")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestCreateUserShortPassword(t *testing.T) {
	svc := newTestService(t)
	err := svc.Create("alice", "ab")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 4")
}

func TestSetPassword(t *testing.T) {
	svc := newTestService(t)
	require.NoError(t, svc.SetPassword("admin", "newpass1234"))
}

func TestSetPasswordShort(t *testing.T) {
	svc := newTestService(t)
	err := svc.SetPassword("admin", "ab")
	assert.Error(t, err)
}

func TestSetEnabled(t *testing.T) {
	svc := newTestService(t)
	require.NoError(t, svc.Create("alice", "pass1234"))

	require.NoError(t, svc.SetEnabled("alice", false))

	users, err := svc.List()
	require.NoError(t, err)
	for _, u := range users {
		if u.Username == "alice" {
			assert.False(t, u.Enabled)
		}
	}
}

func TestSetEnabledPreventLastUser(t *testing.T) {
	svc := newTestService(t)
	err := svc.SetEnabled("admin", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "last active user")
}

func TestDeleteUser(t *testing.T) {
	svc := newTestService(t)
	require.NoError(t, svc.Create("alice", "pass1234"))

	require.NoError(t, svc.Delete("alice"))

	users, err := svc.List()
	require.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestDeletePreventLastUser(t *testing.T) {
	svc := newTestService(t)
	err := svc.Delete("admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "last user")
}

func TestExists(t *testing.T) {
	svc := newTestService(t)

	exists, err := svc.Exists("admin")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = svc.Exists("nobody")
	require.NoError(t, err)
	assert.False(t, exists)
}
