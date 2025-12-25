//go:build oidc

// Package integration provides integration tests for the schema registry.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/api"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsoncompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

var (
	oidcTestServer   *httptest.Server
	oidcTestStore    storage.Storage
	oidcProvider     *auth.OIDCProvider
	oidcIssuerURL    string
	oidcClientID     = "schema-registry"
	oidcClientSecret = "schema-registry-secret"
)

// OIDC test users (must match Keycloak realm configuration)
const (
	// Admin user - member of schema-registry-admins group
	oidcAdminUser     = "admin"
	oidcAdminPassword = "adminpass"

	// Developer user - member of developers group
	oidcDevUser     = "developer"
	oidcDevPassword = "devpass"

	// Readonly user - member of readonly-users group
	oidcReadonlyUser     = "readonly"
	oidcReadonlyPassword = "readonlypass"
)

// Token cache to avoid repeated token requests
var tokenCache = make(map[string]string)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Create in-memory storage
	store := memory.NewStore()
	oidcTestStore = store

	// Get Keycloak issuer URL from environment
	oidcIssuerURL = getEnvOrDefault("OIDC_ISSUER_URL", "http://localhost:8080/realms/schema-registry")

	// Wait for Keycloak to be ready
	if err := waitForOIDCProvider(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "OIDC provider is not ready: %v\n", err)
		os.Exit(1)
	}

	// Create schema parser registry and register parsers
	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	// Create compatibility checker and register type-specific checkers
	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	// Create registry
	reg := registry.New(store, schemaRegistry, compatChecker, "BACKWARD")

	// Create OIDC configuration
	oidcCfg := config.OIDCConfig{
		Enabled:       true,
		IssuerURL:     oidcIssuerURL,
		ClientID:      oidcClientID,
		ClientSecret:  oidcClientSecret,
		UsernameClaim: "preferred_username",
		RolesClaim:    "groups",
		RoleMapping: map[string]string{
			"/schema-registry-admins": "admin",
			"/developers":             "developer",
			"/readonly-users":         "readonly",
		},
		DefaultRole:     "readonly",
		SkipIssuerCheck: false,
		SkipExpiryCheck: false,
	}

	// Create OIDC provider
	var err error
	oidcProvider, err = auth.NewOIDCProvider(ctx, oidcCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create OIDC provider: %v\n", err)
		os.Exit(1)
	}

	// Create authenticator with OIDC
	authCfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"oidc"},
		OIDC:    oidcCfg,
		RBAC: config.RBACConfig{
			Enabled:     true,
			DefaultRole: "readonly",
		},
	}

	authenticator := auth.NewAuthenticator(authCfg)
	authenticator.SetOIDCProvider(oidcProvider)

	// Create authorizer
	authorizer := auth.NewAuthorizer(authCfg.RBAC)

	// Create server with auth
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8082,
		},
		Security: config.SecurityConfig{
			Auth: authCfg,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := api.NewServer(cfg, reg, logger, api.WithAuth(authenticator, authorizer, nil))
	oidcTestServer = httptest.NewServer(server)

	// Pre-fetch tokens for test users
	if err := prefetchTokens(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to prefetch tokens: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	oidcTestServer.Close()
	store.Close()

	os.Exit(code)
}

// waitForOIDCProvider waits for the OIDC provider to be ready.
func waitForOIDCProvider(ctx context.Context) error {
	discoveryURL := oidcIssuerURL + "/.well-known/openid-configuration"
	maxRetries := 30

	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(discoveryURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("OIDC provider not ready after %d retries", maxRetries)
}

// prefetchTokens gets tokens for all test users.
func prefetchTokens() error {
	users := []struct {
		username string
		password string
	}{
		{oidcAdminUser, oidcAdminPassword},
		{oidcDevUser, oidcDevPassword},
		{oidcReadonlyUser, oidcReadonlyPassword},
	}

	for _, user := range users {
		token, err := getOIDCToken(user.username, user.password)
		if err != nil {
			return fmt.Errorf("failed to get token for %s: %w", user.username, err)
		}
		tokenCache[user.username] = token
	}
	return nil
}

// getOIDCToken gets an access token from Keycloak using the password grant.
func getOIDCToken(username, password string) (string, error) {
	// Check cache first
	if token, ok := tokenCache[username]; ok {
		return token, nil
	}

	// Keycloak token endpoint
	tokenURL := oidcIssuerURL + "/protocol/openid-connect/token"

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", username)
	data.Set("password", password)
	data.Set("client_id", oidcClientID)
	data.Set("client_secret", oidcClientSecret)
	data.Set("scope", "openid")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		IDToken     string `json:"id_token"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	// Use ID token for OIDC authentication (contains the claims we need)
	token := tokenResp.IDToken
	if token == "" {
		token = tokenResp.AccessToken
	}

	tokenCache[username] = token
	return token, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// bearerAuth returns the Authorization header value for bearer token auth.
func bearerAuth(token string) string {
	return "Bearer " + token
}

// TestOIDCAuthentication tests that OIDC authentication works for all user types.
func TestOIDCAuthentication(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		expectStatus int
	}{
		{
			name:         "AdminUserCanAuthenticate",
			email:        oidcAdminUser,
			expectStatus: http.StatusOK,
		},
		{
			name:         "DeveloperUserCanAuthenticate",
			email:        oidcDevUser,
			expectStatus: http.StatusOK,
		},
		{
			name:         "ReadonlyUserCanAuthenticate",
			email:        oidcReadonlyUser,
			expectStatus: http.StatusOK,
		},
		{
			name:         "InvalidTokenFails",
			email:        "", // Will use empty token
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", oidcTestServer.URL+"/subjects", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.email != "" {
				token := tokenCache[tt.email]
				req.Header.Set("Authorization", bearerAuth(token))
			} else {
				req.Header.Set("Authorization", bearerAuth("invalid-token"))
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d: %s", tt.expectStatus, resp.StatusCode, string(body))
			}
		})
	}
}

// TestOIDCRoleBasedAccess tests that RBAC is properly enforced based on OIDC groups.
func TestOIDCRoleBasedAccess(t *testing.T) {
	// First, register a schema as admin for later tests
	avroSchema := `{"type":"record","name":"TestOIDC","fields":[{"name":"id","type":"int"}]}`
	schemaReq := map[string]interface{}{
		"schema":     avroSchema,
		"schemaType": "AVRO",
	}
	schemaBody, _ := json.Marshal(schemaReq)

	t.Run("AdminCanRegisterSchema", func(t *testing.T) {
		req, _ := http.NewRequest("POST", oidcTestServer.URL+"/subjects/oidc-test-subject/versions", bytes.NewBuffer(schemaBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcAdminUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("DeveloperCanRegisterSchema", func(t *testing.T) {
		devSchema := `{"type":"record","name":"TestOIDCDev","fields":[{"name":"id","type":"int"}]}`
		devReq := map[string]interface{}{
			"schema":     devSchema,
			"schemaType": "AVRO",
		}
		devBody, _ := json.Marshal(devReq)

		req, _ := http.NewRequest("POST", oidcTestServer.URL+"/subjects/oidc-dev-subject/versions", bytes.NewBuffer(devBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcDevUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("ReadonlyCannotRegisterSchema", func(t *testing.T) {
		roSchema := `{"type":"record","name":"TestOIDCRO","fields":[{"name":"id","type":"int"}]}`
		roReq := map[string]interface{}{
			"schema":     roSchema,
			"schemaType": "AVRO",
		}
		roBody, _ := json.Marshal(roReq)

		req, _ := http.NewRequest("POST", oidcTestServer.URL+"/subjects/oidc-readonly-subject/versions", bytes.NewBuffer(roBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcReadonlyUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 403 Forbidden, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("ReadonlyCanReadSchema", func(t *testing.T) {
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/subjects/oidc-test-subject/versions/1", nil)
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcReadonlyUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("ReadonlyCanListSubjects", func(t *testing.T) {
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/subjects", nil)
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcReadonlyUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("ReadonlyCannotDeleteSubject", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", oidcTestServer.URL+"/subjects/oidc-test-subject", nil)
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcReadonlyUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 403 Forbidden, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("DeveloperCannotDeleteSubject", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", oidcTestServer.URL+"/subjects/oidc-dev-subject", nil)
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcDevUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Developers don't have delete permission
		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 403 Forbidden, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("AdminCanDeleteSubject", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", oidcTestServer.URL+"/subjects/oidc-test-subject", nil)
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcAdminUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})
}

// TestOIDCConfigAccess tests access to configuration endpoints.
func TestOIDCConfigAccess(t *testing.T) {
	t.Run("AdminCanReadConfig", func(t *testing.T) {
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/config", nil)
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcAdminUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("AdminCanSetConfig", func(t *testing.T) {
		configReq := map[string]string{
			"compatibility": "FULL",
		}
		configBody, _ := json.Marshal(configReq)

		req, _ := http.NewRequest("PUT", oidcTestServer.URL+"/config", bytes.NewBuffer(configBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcAdminUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("ReadonlyCanReadConfig", func(t *testing.T) {
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/config", nil)
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcReadonlyUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("ReadonlyCannotSetConfig", func(t *testing.T) {
		configReq := map[string]string{
			"compatibility": "NONE",
		}
		configBody, _ := json.Marshal(configReq)

		req, _ := http.NewRequest("PUT", oidcTestServer.URL+"/config", bytes.NewBuffer(configBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcReadonlyUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 403 Forbidden, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("DeveloperCanReadConfig", func(t *testing.T) {
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/config", nil)
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcDevUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("DeveloperCannotSetConfig", func(t *testing.T) {
		configReq := map[string]string{
			"compatibility": "BACKWARD_TRANSITIVE",
		}
		configBody, _ := json.Marshal(configReq)

		req, _ := http.NewRequest("PUT", oidcTestServer.URL+"/config", bytes.NewBuffer(configBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcDevUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 403 Forbidden, got %d: %s", resp.StatusCode, string(body))
		}
	})
}

// TestOIDCModeAccess tests access to mode endpoints.
func TestOIDCModeAccess(t *testing.T) {
	t.Run("AdminCanReadMode", func(t *testing.T) {
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/mode", nil)
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcAdminUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("AdminCanSetMode", func(t *testing.T) {
		modeReq := map[string]string{
			"mode": "READWRITE",
		}
		modeBody, _ := json.Marshal(modeReq)

		req, _ := http.NewRequest("PUT", oidcTestServer.URL+"/mode", bytes.NewBuffer(modeBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcAdminUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("ReadonlyCannotSetMode", func(t *testing.T) {
		modeReq := map[string]string{
			"mode": "READONLY",
		}
		modeBody, _ := json.Marshal(modeReq)

		req, _ := http.NewRequest("PUT", oidcTestServer.URL+"/mode", bytes.NewBuffer(modeBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerAuth(tokenCache[oidcReadonlyUser]))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 403 Forbidden, got %d: %s", resp.StatusCode, string(body))
		}
	})
}

// TestOIDCTokenValidation tests various token validation scenarios.
func TestOIDCTokenValidation(t *testing.T) {
	t.Run("ExpiredTokenFails", func(t *testing.T) {
		// Using a clearly invalid/expired token
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/subjects", nil)
		req.Header.Set("Authorization", bearerAuth("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjU1NTYvZGV4Iiwic3ViIjoiZXhwaXJlZCIsImF1ZCI6InNjaGVtYS1yZWdpc3RyeSIsImV4cCI6MTAwMDAwMDAwMH0.invalid"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 401, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("MalformedTokenFails", func(t *testing.T) {
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/subjects", nil)
		req.Header.Set("Authorization", bearerAuth("not-a-valid-jwt-token"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 401, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("MissingAuthorizationHeader", func(t *testing.T) {
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/subjects", nil)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 401, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("WrongAuthScheme", func(t *testing.T) {
		req, _ := http.NewRequest("GET", oidcTestServer.URL+"/subjects", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // Basic auth instead of Bearer

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should fail since only OIDC auth is enabled
		if resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 401, got %d: %s", resp.StatusCode, string(body))
		}
	})
}
