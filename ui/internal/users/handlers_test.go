package users

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/axonops/schema-registry-ui/internal/auth"
	"github.com/axonops/schema-registry-ui/internal/htpasswd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	dir := t.TempDir()
	hp := htpasswd.New(filepath.Join(dir, "htpasswd"))
	_, err := hp.Bootstrap("admin", "admin")
	require.NoError(t, err)
	svc := NewService(hp)
	return NewHandlers(svc)
}

func withUser(r *http.Request, username string) *http.Request {
	ctx := context.WithValue(r.Context(), auth.TestContextKey(), username)
	return r.WithContext(ctx)
}

func TestListUsersHandler(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	h.ListUsers(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var users []UserInfo
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&users))
	require.Len(t, users, 1)
	assert.Equal(t, "admin", users[0].Username)
}

func TestCreateUserHandler(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(createUserRequest{Username: "alice", Password: "pass1234"})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req = withUser(req, "admin")
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var user UserInfo
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&user))
	assert.Equal(t, "alice", user.Username)
	assert.True(t, user.Enabled)
}

func TestCreateUserDuplicate(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(createUserRequest{Username: "admin", Password: "pass1234"})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestCreateUserMissingFields(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(createUserRequest{Username: "alice"})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateUserInvalidUsername(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(createUserRequest{Username: "a", Password: "pass1234"})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateUserPassword(t *testing.T) {
	h := setupTestHandlers(t)

	newPass := "newpassword"
	body, _ := json.Marshal(updateUserRequest{Password: &newPass})
	req := httptest.NewRequest("PUT", "/api/users/admin", bytes.NewReader(body))
	req = withUser(req, "admin")
	rec := httptest.NewRecorder()
	h.UpdateUser(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestUpdateUserEnabled(t *testing.T) {
	h := setupTestHandlers(t)

	// Create another user first so we can disable admin
	h.service.Create("alice", "pass1234")

	disabled := false
	body, _ := json.Marshal(updateUserRequest{Enabled: &disabled})
	req := httptest.NewRequest("PUT", "/api/users/admin", bytes.NewReader(body))
	req = withUser(req, "alice")
	rec := httptest.NewRecorder()
	h.UpdateUser(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestUpdateUserNotFound(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(updateUserRequest{})
	req := httptest.NewRequest("PUT", "/api/users/nobody", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.UpdateUser(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteUserHandler(t *testing.T) {
	h := setupTestHandlers(t)
	h.service.Create("alice", "pass1234")

	req := httptest.NewRequest("DELETE", "/api/users/alice", nil)
	req = withUser(req, "admin")
	rec := httptest.NewRecorder()
	h.DeleteUser(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteLastUser(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("DELETE", "/api/users/admin", nil)
	rec := httptest.NewRecorder()
	h.DeleteUser(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestDeleteUserNotFound(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("DELETE", "/api/users/nobody", nil)
	rec := httptest.NewRecorder()
	h.DeleteUser(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestChangeMyPassword(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(changePasswordRequest{
		CurrentPassword: "admin",
		NewPassword:     "newpass1234",
	})
	req := httptest.NewRequest("POST", "/api/users/me/password", bytes.NewReader(body))
	req = withUser(req, "admin")
	rec := httptest.NewRecorder()
	h.ChangeMyPassword(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestChangeMyPasswordUnauthorized(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(changePasswordRequest{NewPassword: "newpass"})
	req := httptest.NewRequest("POST", "/api/users/me/password", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ChangeMyPassword(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestChangeMyPasswordMissing(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(changePasswordRequest{})
	req := httptest.NewRequest("POST", "/api/users/me/password", bytes.NewReader(body))
	req = withUser(req, "admin")
	rec := httptest.NewRecorder()
	h.ChangeMyPassword(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
