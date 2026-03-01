package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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

	tm := NewTokenManager("test-secret", 3600)
	return NewHandlers(hp, tm, "sr_session", false)
}

func TestLoginSuccess(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(loginRequest{Username: "admin", Password: "admin"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp loginResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "admin", resp.Username)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "sr_session", cookies[0].Name)
	assert.True(t, cookies[0].HttpOnly)
}

func TestLoginWrongPassword(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(loginRequest{Username: "admin", Password: "wrong"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLoginMissingFields(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(loginRequest{Username: "admin"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLoginInvalidBody(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLoginWrongMethod(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("GET", "/api/auth/login", nil)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestLoginNonexistentUser(t *testing.T) {
	h := setupTestHandlers(t)

	body, _ := json.Marshal(loginRequest{Username: "nobody", Password: "pass"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogout(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "sr_session", cookies[0].Name)
	assert.Equal(t, -1, cookies[0].MaxAge)
}

func TestLogoutWrongMethod(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("GET", "/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestSessionWithAuth(t *testing.T) {
	h := setupTestHandlers(t)

	// First login to get a valid cookie
	body, _ := json.Marshal(loginRequest{Username: "admin", Password: "admin"})
	loginReq := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	cookie := loginRec.Result().Cookies()[0]

	// Now check session — this needs the middleware to set context.
	// Simulate by going through middleware chain.
	tm := NewTokenManager("test-secret", 3600)
	handler := Middleware(tm, "sr_session", nil)(http.HandlerFunc(h.Session))

	sessionReq := httptest.NewRequest("GET", "/api/auth/session", nil)
	sessionReq.AddCookie(cookie)
	sessionRec := httptest.NewRecorder()

	handler.ServeHTTP(sessionRec, sessionReq)

	assert.Equal(t, http.StatusOK, sessionRec.Code)

	var resp sessionResponse
	require.NoError(t, json.NewDecoder(sessionRec.Body).Decode(&resp))
	assert.Equal(t, "admin", resp.Username)

	// Session response should also set a refreshed cookie
	assert.Len(t, sessionRec.Result().Cookies(), 1)
}

func TestSessionNoAuth(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("GET", "/api/auth/session", nil)
	rec := httptest.NewRecorder()

	h.Session(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestConfig(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("GET", "/api/auth/config", nil)
	rec := httptest.NewRecorder()

	h.Config(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp configResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.True(t, resp.AuthEnabled)
}

func TestHealth(t *testing.T) {
	h := setupTestHandlers(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "UP", resp["status"])
	assert.NotEmpty(t, resp["time"])
}
