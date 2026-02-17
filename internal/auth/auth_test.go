package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
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

func TestGetUserID(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDContextKey, int64(42))
	if id := GetUserID(ctx); id != 42 {
		t.Errorf("Expected 42, got %d", id)
	}

	if id := GetUserID(context.Background()); id != 0 {
		t.Errorf("Expected 0 for missing ID, got %d", id)
	}
}

func TestAuthenticator_MTLSAuth(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"mtls"},
		RBAC: config.RBACConfig{
			DefaultRole: "developer",
		},
	}

	auth := NewAuthenticator(cfg)

	t.Run("valid client cert", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)
		req.TLS = &tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{
				{Subject: pkix.Name{CommonName: "client-app"}},
			},
		}

		var capturedUser *User
		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedUser = GetUser(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
		if capturedUser == nil {
			t.Fatal("Expected user in context")
		}
		if capturedUser.Username != "client-app" {
			t.Errorf("Expected username 'client-app', got %s", capturedUser.Username)
		}
		if capturedUser.Method != "mtls" {
			t.Errorf("Expected method 'mtls', got %s", capturedUser.Method)
		}
		if capturedUser.Role != "developer" {
			t.Errorf("Expected role 'developer', got %s", capturedUser.Role)
		}
	})

	t.Run("no TLS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)
		// req.TLS is nil

		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})

	t.Run("TLS but no peer certs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)
		req.TLS = &tls.ConnectionState{}

		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})

	t.Run("empty CN", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)
		req.TLS = &tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{
				{Subject: pkix.Name{CommonName: ""}},
			},
		}

		handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 for empty CN, got %d", rr.Code)
		}
	})
}

func TestAuthenticator_Unauthorized_ChallengeHeaders(t *testing.T) {
	tests := []struct {
		name     string
		methods  []string
		expected []string
	}{
		{
			name:     "basic auth challenge",
			methods:  []string{"basic"},
			expected: []string{"Basic"},
		},
		{
			name:     "basic with custom realm",
			methods:  []string{"basic"},
			expected: []string{`Basic realm="Custom Realm"`},
		},
		{
			name:     "api_key challenge",
			methods:  []string{"api_key"},
			expected: []string{"API-Key"},
		},
		{
			name:     "jwt challenge",
			methods:  []string{"jwt"},
			expected: []string{"Bearer"},
		},
		{
			name:     "oidc challenge",
			methods:  []string{"oidc"},
			expected: []string{"Bearer"},
		},
		{
			name:     "multiple methods",
			methods:  []string{"basic", "api_key", "jwt"},
			expected: []string{"Basic", "API-Key", "Bearer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			realm := ""
			if tt.name == "basic with custom realm" {
				realm = "Custom Realm"
			}
			cfg := config.AuthConfig{
				Enabled: true,
				Methods: tt.methods,
				Basic:   config.BasicAuthConfig{Realm: realm},
			}
			auth := NewAuthenticator(cfg)

			req := httptest.NewRequest("GET", "/subjects", nil)
			handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("Expected 401, got %d", rr.Code)
			}

			authHeaders := rr.Header().Values("WWW-Authenticate")
			if len(authHeaders) != len(tt.expected) {
				t.Errorf("Expected %d WWW-Authenticate headers, got %d: %v", len(tt.expected), len(authHeaders), authHeaders)
			}
		})
	}
}

func TestAuthenticator_Middleware_StoresUserID(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"api_key"},
		APIKey:  config.APIKeyConfig{Header: "X-API-Key"},
	}

	auth := NewAuthenticator(cfg)
	auth.AddAPIKey(&APIKey{
		Key:      "test-key",
		Username: "apiuser",
		Role:     "admin",
	})

	req := httptest.NewRequest("GET", "/subjects", nil)
	req.Header.Set("X-API-Key", "test-key")

	var capturedRole string
	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRole = GetRole(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedRole != "admin" {
		t.Errorf("Expected role 'admin' from context, got '%s'", capturedRole)
	}
}

func TestAuthenticator_UnknownMethod(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"unknown_method"},
	}

	auth := NewAuthenticator(cfg)
	req := httptest.NewRequest("GET", "/subjects", nil)

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for unknown method, got %d", rr.Code)
	}
}

func TestAuthenticator_BasicAuth_InvalidBase64(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"basic"},
	}

	auth := NewAuthenticator(cfg)
	req := httptest.NewRequest("GET", "/subjects", nil)
	req.Header.Set("Authorization", "Basic not-valid-base64!!!")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for invalid base64, got %d", rr.Code)
	}
}

func TestAuthenticator_BasicAuth_NoColon(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"basic"},
	}

	auth := NewAuthenticator(cfg)
	req := httptest.NewRequest("GET", "/subjects", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("no-colon"))
	req.Header.Set("Authorization", "Basic "+credentials)

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for missing colon, got %d", rr.Code)
	}
}

func TestAuthenticator_BasicAuth_APIKeyViaBasic(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"basic"},
		APIKey:  config.APIKeyConfig{Header: "X-API-Key"},
	}

	auth := NewAuthenticator(cfg)
	auth.AddAPIKey(&APIKey{
		Key:      "my-api-key",
		Username: "apiuser",
		Role:     "readwrite",
	})

	// Confluent-compatible: API key as username, any password
	req := httptest.NewRequest("GET", "/subjects", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("my-api-key:any-secret"))
	req.Header.Set("Authorization", "Basic "+credentials)

	var capturedUser *User
	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = GetUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
	if capturedUser == nil {
		t.Fatal("Expected user")
	}
	if capturedUser.Method != "api_key" {
		t.Errorf("Expected method 'api_key', got %s", capturedUser.Method)
	}
}

func TestAuthenticator_JWT_NoBearerPrefix(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"jwt"},
	}

	auth := NewAuthenticator(cfg)
	req := httptest.NewRequest("GET", "/subjects", nil)
	req.Header.Set("Authorization", "Token some-token")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for non-Bearer prefix, got %d", rr.Code)
	}
}

func TestAuthenticator_JWT_NoProvider(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"jwt"},
	}

	auth := NewAuthenticator(cfg)
	req := httptest.NewRequest("GET", "/subjects", nil)
	req.Header.Set("Authorization", "Bearer some-token")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 when JWT provider is nil, got %d", rr.Code)
	}
}

func TestAuthenticator_SetProviders(t *testing.T) {
	auth := NewAuthenticator(config.AuthConfig{})

	// These are simple setters - verify they don't panic
	auth.SetService(nil)
	auth.SetLDAPProvider(nil)
	auth.SetOIDCProvider(nil)
	auth.SetJWTProvider(nil)

	auth.AddAPIKey(&APIKey{Key: "k", Username: "u", Role: "r"})
	if _, ok := auth.apiKeys["k"]; !ok {
		t.Error("Expected API key to be added")
	}
}
