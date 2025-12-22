package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func TestAuthenticator_BasicAuth(t *testing.T) {
	// Create a password hash
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"basic"},
		Basic: config.BasicAuthConfig{
			Realm: "Test Realm",
			Users: map[string]string{
				"testuser": hash,
			},
		},
		RBAC: config.RBACConfig{
			DefaultRole: "developer",
		},
	}

	auth := NewAuthenticator(cfg)

	// Test successful authentication
	t.Run("valid credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)
		credentials := base64.StdEncoding.EncodeToString([]byte("testuser:secret123"))
		req.Header.Set("Authorization", "Basic "+credentials)

		var capturedUser *User
		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedUser = GetUser(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if capturedUser == nil {
			t.Error("Expected user in context")
		} else if capturedUser.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", capturedUser.Username)
		}
	})

	// Test invalid password
	t.Run("invalid password", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)
		credentials := base64.StdEncoding.EncodeToString([]byte("testuser:wrongpassword"))
		req.Header.Set("Authorization", "Basic "+credentials)

		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	// Test missing authorization header
	t.Run("no credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)

		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})
}

func TestAuthenticator_APIKey(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"api_key"},
		APIKey: config.APIKeyConfig{
			Header:     "X-API-Key",
			QueryParam: "api_key",
		},
		RBAC: config.RBACConfig{
			DefaultRole: "developer",
		},
	}

	auth := NewAuthenticator(cfg)
	auth.AddAPIKey(&APIKey{
		Key:      "test-api-key-123",
		Name:     "Test Key",
		Username: "apiuser",
		Role:     "admin",
	})

	// Test successful authentication with header
	t.Run("valid API key in header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)
		req.Header.Set("X-API-Key", "test-api-key-123")

		var capturedUser *User
		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedUser = GetUser(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if capturedUser == nil {
			t.Error("Expected user in context")
		} else {
			if capturedUser.Username != "apiuser" {
				t.Errorf("Expected username 'apiuser', got '%s'", capturedUser.Username)
			}
			if capturedUser.Role != "admin" {
				t.Errorf("Expected role 'admin', got '%s'", capturedUser.Role)
			}
		}
	})

	// Test successful authentication with query param
	t.Run("valid API key in query", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects?api_key=test-api-key-123", nil)

		var capturedUser *User
		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedUser = GetUser(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if capturedUser == nil {
			t.Error("Expected user in context")
		}
	})

	// Test invalid API key
	t.Run("invalid API key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)
		req.Header.Set("X-API-Key", "invalid-key")

		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})
}

func TestAuthenticator_Disabled(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: false,
	}

	auth := NewAuthenticator(cfg)

	req := httptest.NewRequest("GET", "/subjects", nil)

	var called bool
	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Handler should have been called when auth is disabled")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestGetUser(t *testing.T) {
	user := &User{
		Username: "testuser",
		Role:     "admin",
	}

	ctx := context.WithValue(context.Background(), UserContextKey, user)

	result := GetUser(ctx)
	if result == nil {
		t.Fatal("Expected user from context")
	}
	if result.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", result.Username)
	}

	// Test with no user in context
	emptyCtx := context.Background()
	if GetUser(emptyCtx) != nil {
		t.Error("Expected nil for context without user")
	}
}

func TestGetRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), RoleContextKey, "admin")

	result := GetRole(ctx)
	if result != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", result)
	}

	// Test with no role in context
	emptyCtx := context.Background()
	if GetRole(emptyCtx) != "" {
		t.Error("Expected empty string for context without role")
	}
}

func TestHashPassword(t *testing.T) {
	password := "mysecretpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Hash should start with bcrypt prefix
	if hash[0] != '$' {
		t.Error("Hash should start with bcrypt prefix")
	}
}

func TestConstantTimeCompare(t *testing.T) {
	if !ConstantTimeCompare("test", "test") {
		t.Error("Expected true for equal strings")
	}

	if ConstantTimeCompare("test", "other") {
		t.Error("Expected false for different strings")
	}
}
