package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndValidate(t *testing.T) {
	tm := NewTokenManager("test-secret", 3600)

	token, expiresAt, err := tm.Create("alice")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))

	claims, err := tm.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, "alice", claims.Username)
	assert.Equal(t, "schema-registry-ui", claims.Issuer)
}

func TestValidateExpiredToken(t *testing.T) {
	tm := NewTokenManager("test-secret", 1) // 1 second TTL

	token, _, err := tm.Create("alice")
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	_, err = tm.Validate(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing token")
}

func TestValidateWrongSecret(t *testing.T) {
	tm1 := NewTokenManager("secret-one", 3600)
	tm2 := NewTokenManager("secret-two", 3600)

	token, _, err := tm1.Create("alice")
	require.NoError(t, err)

	_, err = tm2.Validate(token)
	assert.Error(t, err)
}

func TestValidateGarbageToken(t *testing.T) {
	tm := NewTokenManager("test-secret", 3600)

	_, err := tm.Validate("not-a-valid-token")
	assert.Error(t, err)
}

func TestValidateEmptyToken(t *testing.T) {
	tm := NewTokenManager("test-secret", 3600)

	_, err := tm.Validate("")
	assert.Error(t, err)
}

func TestTTL(t *testing.T) {
	tm := NewTokenManager("test-secret", 7200)
	assert.Equal(t, 2*time.Hour, tm.TTL())
}
