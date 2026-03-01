package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareValidCookie(t *testing.T) {
	tm := NewTokenManager("test-secret", 3600)
	token, _, err := tm.Create("alice")
	require.NoError(t, err)

	handler := Middleware(tm, "sr_session", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := UsernameFromContext(r.Context())
		w.Write([]byte(username))
	}))

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	req.AddCookie(&http.Cookie{Name: "sr_session", Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "alice", rec.Body.String())
}

func TestMiddlewareNoCookie(t *testing.T) {
	tm := NewTokenManager("test-secret", 3600)
	handler := Middleware(tm, "sr_session", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareInvalidToken(t *testing.T) {
	tm := NewTokenManager("test-secret", 3600)
	handler := Middleware(tm, "sr_session", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	req.AddCookie(&http.Cookie{Name: "sr_session", Value: "garbage"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareExpiredToken(t *testing.T) {
	tm := NewTokenManager("test-secret", 1)
	token, _, err := tm.Create("alice")
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	handler := Middleware(tm, "sr_session", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	req.AddCookie(&http.Cookie{Name: "sr_session", Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareSkipPaths(t *testing.T) {
	tm := NewTokenManager("test-secret", 3600)
	skipPaths := map[string]bool{
		"/api/auth/login": true,
	}
	handler := Middleware(tm, "sr_session", skipPaths)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddlewareWrongCookieName(t *testing.T) {
	tm := NewTokenManager("test-secret", 3600)
	token, _, err := tm.Create("alice")
	require.NoError(t, err)

	handler := Middleware(tm, "sr_session", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	req.AddCookie(&http.Cookie{Name: "wrong_cookie", Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUsernameFromContextEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	assert.Equal(t, "", UsernameFromContext(req.Context()))
}

func TestMiddlewareWrongSecret(t *testing.T) {
	tm1 := NewTokenManager("secret-one", 3600)
	tm2 := NewTokenManager("secret-two", 3600)
	token, _, err := tm1.Create("alice")
	require.NoError(t, err)

	handler := Middleware(tm2, "sr_session", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	req.AddCookie(&http.Cookie{Name: "sr_session", Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
