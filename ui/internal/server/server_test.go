package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/axonops/schema-registry-ui/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 0},
		Registry: config.RegistryConfig{
			URL: "http://localhost:8081",
		},
		Auth: config.AuthConfig{
			HtpasswdFile:  filepath.Join(dir, "htpasswd"),
			SessionSecret: "test-secret-for-server-tests",
			SessionTTL:    3600,
			CookieName:    "sr_session",
			CookieSecure:  false,
		},
	}
}

func testSPAFS() *fstest.MapFS {
	return &fstest.MapFS{
		"index.html": {Data: []byte("<html>SPA</html>")},
	}
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := testConfig(t)
	srv, err := New(cfg, testSPAFS())
	require.NoError(t, err)

	// Use httptest.Server instead of real server for testing
	return httptest.NewServer(srv.httpServer.Handler)
}

func TestHealthEndpoint(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "UP", body["status"])
}

func TestAuthConfigEndpoint(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/auth/config")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]bool
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.True(t, body["auth_enabled"])
}

func TestLoginEndpoint(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	payload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin",
	})
	resp, err := http.Post(ts.URL+"/api/auth/login", "application/json", bytes.NewReader(payload))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "admin", body["username"])

	// Should have a session cookie
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "sr_session" {
			sessionCookie = c
		}
	}
	require.NotNil(t, sessionCookie)
}

func TestUnauthenticatedAPIReturns401(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/auth/session")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthenticatedSession(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// Login first
	payload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin",
	})
	loginResp, err := http.Post(ts.URL+"/api/auth/login", "application/json", bytes.NewReader(payload))
	require.NoError(t, err)
	defer loginResp.Body.Close()
	require.Equal(t, http.StatusOK, loginResp.StatusCode)

	var sessionCookie *http.Cookie
	for _, c := range loginResp.Cookies() {
		if c.Name == "sr_session" {
			sessionCookie = c
		}
	}
	require.NotNil(t, sessionCookie)

	// Check session
	req, _ := http.NewRequest("GET", ts.URL+"/api/auth/session", nil)
	req.AddCookie(sessionCookie)
	client := &http.Client{}
	sessionResp, err := client.Do(req)
	require.NoError(t, err)
	defer sessionResp.Body.Close()

	assert.Equal(t, http.StatusOK, sessionResp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(sessionResp.Body).Decode(&body))
	assert.Equal(t, "admin", body["username"])
}

func TestSPARoute(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/ui/dashboard")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "<html>SPA</html>")
}

func TestUserManagementEndpoints(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// Login
	payload, _ := json.Marshal(map[string]string{"username": "admin", "password": "admin"})
	loginResp, err := http.Post(ts.URL+"/api/auth/login", "application/json", bytes.NewReader(payload))
	require.NoError(t, err)
	defer loginResp.Body.Close()

	var sessionCookie *http.Cookie
	for _, c := range loginResp.Cookies() {
		if c.Name == "sr_session" {
			sessionCookie = c
		}
	}
	require.NotNil(t, sessionCookie)

	client := &http.Client{}

	// List users
	req, _ := http.NewRequest("GET", ts.URL+"/api/users", nil)
	req.AddCookie(sessionCookie)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var users []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&users))
	assert.Len(t, users, 1)

	// Create user
	createPayload, _ := json.Marshal(map[string]string{"username": "testuser", "password": "pass1234"})
	req, _ = http.NewRequest("POST", ts.URL+"/api/users", bytes.NewReader(createPayload))
	req.AddCookie(sessionCookie)
	req.Header.Set("Content-Type", "application/json")
	resp2, err := client.Do(req)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusCreated, resp2.StatusCode)

	// List users again — should be 2
	req, _ = http.NewRequest("GET", ts.URL+"/api/users", nil)
	req.AddCookie(sessionCookie)
	resp3, err := client.Do(req)
	require.NoError(t, err)
	defer resp3.Body.Close()

	var users2 []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp3.Body).Decode(&users2))
	assert.Len(t, users2, 2)
}

func TestLogoutEndpoint(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// Login
	payload, _ := json.Marshal(map[string]string{"username": "admin", "password": "admin"})
	loginResp, err := http.Post(ts.URL+"/api/auth/login", "application/json", bytes.NewReader(payload))
	require.NoError(t, err)
	defer loginResp.Body.Close()

	var sessionCookie *http.Cookie
	for _, c := range loginResp.Cookies() {
		if c.Name == "sr_session" {
			sessionCookie = c
		}
	}
	require.NotNil(t, sessionCookie)

	client := &http.Client{}

	// Logout
	req, _ := http.NewRequest("POST", ts.URL+"/api/auth/logout", nil)
	req.AddCookie(sessionCookie)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Check that the logout cleared the cookie
	for _, c := range resp.Cookies() {
		if c.Name == "sr_session" {
			assert.Equal(t, -1, c.MaxAge)
		}
	}
}

func TestNewServerWithNilSPA(t *testing.T) {
	cfg := testConfig(t)
	srv, err := New(cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, srv)

	_ = fmt.Sprintf // suppress unused import
}
