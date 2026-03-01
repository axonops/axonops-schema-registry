package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyStripsPrefixAndForwards(t *testing.T) {
	// Mock Schema Registry
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"path": r.URL.Path,
		})
	}))
	defer sr.Close()

	handler, err := New(Config{TargetURL: sr.URL})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "/subjects", resp["path"])
}

func TestProxyInjectsBearerToken(t *testing.T) {
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"auth": r.Header.Get("Authorization"),
		})
	}))
	defer sr.Close()

	handler, err := New(Config{TargetURL: sr.URL, APIToken: "my-token"})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Bearer my-token", resp["auth"])
}

func TestProxyInjectsAPIKey(t *testing.T) {
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"apikey": r.Header.Get("X-API-Key"),
		})
	}))
	defer sr.Close()

	handler, err := New(Config{TargetURL: sr.URL, APIKey: "key-123"})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "key-123", resp["apikey"])
}

func TestProxyRemovesCookie(t *testing.T) {
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"cookie": r.Header.Get("Cookie"),
		})
	}))
	defer sr.Close()

	handler, err := New(Config{TargetURL: sr.URL})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	req.AddCookie(&http.Cookie{Name: "sr_session", Value: "secret-token"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Empty(t, resp["cookie"])
}

func TestProxyErrorHandler(t *testing.T) {
	// Point at a non-existent server
	handler, err := New(Config{TargetURL: "http://127.0.0.1:1"})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "unavailable")
}

func TestProxyNoAuth(t *testing.T) {
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"auth":   r.Header.Get("Authorization"),
			"apikey": r.Header.Get("X-API-Key"),
		})
	}))
	defer sr.Close()

	handler, err := New(Config{TargetURL: sr.URL})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Empty(t, resp["auth"])
	assert.Empty(t, resp["apikey"])
}

func TestProxyTokenPrecedence(t *testing.T) {
	// When both are set, Bearer token takes precedence
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"auth":   r.Header.Get("Authorization"),
			"apikey": r.Header.Get("X-API-Key"),
		})
	}))
	defer sr.Close()

	handler, err := New(Config{TargetURL: sr.URL, APIToken: "tok", APIKey: "key"})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/subjects", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Bearer tok", resp["auth"])
	assert.Empty(t, resp["apikey"])
}

func TestProxyRootPath(t *testing.T) {
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": r.URL.Path})
	}))
	defer sr.Close()

	handler, err := New(Config{TargetURL: sr.URL})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "/", resp["path"])
}

func TestProxyInvalidURL(t *testing.T) {
	_, err := New(Config{TargetURL: "://bad"})
	assert.Error(t, err)
}
