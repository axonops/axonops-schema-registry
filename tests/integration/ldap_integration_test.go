//go:build ldap

// Package integration provides integration tests for the schema registry.
package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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
	ldapTestServer  *httptest.Server
	ldapTestStore   storage.Storage
	ldapAuthService *auth.Service
)

// LDAP test users (must match the LDIF configuration)
const (
	// Admin user - member of SchemaRegistryAdmins group
	ldapAdminUser     = "admin"
	ldapAdminPassword = "adminpass"

	// Developer user - member of Developers group
	ldapDevUser     = "developer"
	ldapDevPassword = "devpass"

	// Readonly user - member of ReadonlyUsers group
	ldapReadonlyUser     = "readonly"
	ldapReadonlyPassword = "readonlypass"

	// User with no groups - should get default role
	ldapNoGroupUser     = "nogroup"
	ldapNoGroupPassword = "nogrouppass"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Create in-memory storage
	store := memory.NewStore()
	ldapTestStore = store

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

	// Create LDAP configuration
	ldapURL := getEnvOrDefault("LDAP_URL", "ldap://localhost:389")
	ldapCfg := config.LDAPConfig{
		Enabled:           true,
		URL:               ldapURL,
		BindDN:            "cn=admin,dc=example,dc=org",
		BindPassword:      "adminpassword",
		BaseDN:            "dc=example,dc=org",
		UserSearchBase:    "ou=Users,dc=example,dc=org",
		UserSearchFilter:  "(uid=%s)",
		UsernameAttribute: "uid",
		EmailAttribute:    "mail",
		GroupAttribute:    "memberOf",
		RoleMapping: map[string]string{
			"cn=SchemaRegistryAdmins,ou=Groups,dc=example,dc=org": "admin",
			"cn=Developers,ou=Groups,dc=example,dc=org":           "developer",
			"cn=ReadonlyUsers,ou=Groups,dc=example,dc=org":        "readonly",
		},
		DefaultRole:       "readonly",
		ConnectionTimeout: 10,
		RequestTimeout:    30,
	}

	// Create LDAP provider
	ldapProvider, err := auth.NewLDAPProvider(ldapCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create LDAP provider: %v\n", err)
		os.Exit(1)
	}

	// Create authenticator with LDAP
	authCfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"basic"},
		LDAP:    ldapCfg,
		RBAC: config.RBACConfig{
			Enabled:     true,
			DefaultRole: "readonly",
		},
	}

	authenticator := auth.NewAuthenticator(authCfg)
	authenticator.SetLDAPProvider(ldapProvider)

	// Create authorizer
	authorizer := auth.NewAuthorizer(authCfg.RBAC)

	// Create server with auth
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8081,
		},
		Security: config.SecurityConfig{
			Auth: authCfg,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := api.NewServer(cfg, reg, logger, api.WithAuth(authenticator, authorizer, nil))
	ldapTestServer = httptest.NewServer(server)

	// Wait for LDAP to be ready
	if err := waitForLDAP(ctx, ldapProvider); err != nil {
		fmt.Fprintf(os.Stderr, "LDAP is not ready: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	ldapTestServer.Close()
	store.Close()
	ldapProvider.Close()

	os.Exit(code)
}

// waitForLDAP waits for the LDAP server to be ready by attempting to authenticate.
func waitForLDAP(ctx context.Context, provider *auth.LDAPProvider) error {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		_, err := provider.Authenticate(ctx, ldapAdminUser, ldapAdminPassword)
		if err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("LDAP not ready after %d retries", maxRetries)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// basicAuth returns the Authorization header value for basic auth.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// TestLDAPAuthentication tests that LDAP authentication works for all user types.
func TestLDAPAuthentication(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		password     string
		expectStatus int
	}{
		{
			name:         "AdminUserCanAuthenticate",
			username:     ldapAdminUser,
			password:     ldapAdminPassword,
			expectStatus: http.StatusOK,
		},
		{
			name:         "DeveloperUserCanAuthenticate",
			username:     ldapDevUser,
			password:     ldapDevPassword,
			expectStatus: http.StatusOK,
		},
		{
			name:         "ReadonlyUserCanAuthenticate",
			username:     ldapReadonlyUser,
			password:     ldapReadonlyPassword,
			expectStatus: http.StatusOK,
		},
		{
			name:         "NoGroupUserCanAuthenticate",
			username:     ldapNoGroupUser,
			password:     ldapNoGroupPassword,
			expectStatus: http.StatusOK,
		},
		{
			name:         "InvalidPasswordFails",
			username:     ldapAdminUser,
			password:     "wrongpassword",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "NonExistentUserFails",
			username:     "nonexistent",
			password:     "anypassword",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "NoCredentialsFails",
			username:     "",
			password:     "",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", ldapTestServer.URL+"/subjects", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.username != "" {
				req.Header.Set("Authorization", basicAuth(tt.username, tt.password))
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

// TestLDAPRoleBasedAccess tests that RBAC is properly enforced based on LDAP group membership.
func TestLDAPRoleBasedAccess(t *testing.T) {
	// First, register a schema as admin for later tests
	avroSchema := `{"type":"record","name":"TestLDAP","fields":[{"name":"id","type":"int"}]}`
	schemaReq := map[string]interface{}{
		"schema":     avroSchema,
		"schemaType": "AVRO",
	}
	schemaBody, _ := json.Marshal(schemaReq)

	t.Run("AdminCanRegisterSchema", func(t *testing.T) {
		req, _ := http.NewRequest("POST", ldapTestServer.URL+"/subjects/ldap-test-subject/versions", bytes.NewBuffer(schemaBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", basicAuth(ldapAdminUser, ldapAdminPassword))

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
		devSchema := `{"type":"record","name":"TestLDAPDev","fields":[{"name":"id","type":"int"}]}`
		devReq := map[string]interface{}{
			"schema":     devSchema,
			"schemaType": "AVRO",
		}
		devBody, _ := json.Marshal(devReq)

		req, _ := http.NewRequest("POST", ldapTestServer.URL+"/subjects/ldap-dev-subject/versions", bytes.NewBuffer(devBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", basicAuth(ldapDevUser, ldapDevPassword))

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
		roSchema := `{"type":"record","name":"TestLDAPRO","fields":[{"name":"id","type":"int"}]}`
		roReq := map[string]interface{}{
			"schema":     roSchema,
			"schemaType": "AVRO",
		}
		roBody, _ := json.Marshal(roReq)

		req, _ := http.NewRequest("POST", ldapTestServer.URL+"/subjects/ldap-readonly-subject/versions", bytes.NewBuffer(roBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", basicAuth(ldapReadonlyUser, ldapReadonlyPassword))

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

	t.Run("NoGroupUserCannotRegisterSchema", func(t *testing.T) {
		ngSchema := `{"type":"record","name":"TestLDAPNG","fields":[{"name":"id","type":"int"}]}`
		ngReq := map[string]interface{}{
			"schema":     ngSchema,
			"schemaType": "AVRO",
		}
		ngBody, _ := json.Marshal(ngReq)

		req, _ := http.NewRequest("POST", ldapTestServer.URL+"/subjects/ldap-nogroup-subject/versions", bytes.NewBuffer(ngBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", basicAuth(ldapNoGroupUser, ldapNoGroupPassword))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// User with no groups gets default role (readonly), so should be forbidden
		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 403 Forbidden, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("ReadonlyCanReadSchema", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ldapTestServer.URL+"/subjects/ldap-test-subject/versions/1", nil)
		req.Header.Set("Authorization", basicAuth(ldapReadonlyUser, ldapReadonlyPassword))

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
		req, _ := http.NewRequest("GET", ldapTestServer.URL+"/subjects", nil)
		req.Header.Set("Authorization", basicAuth(ldapReadonlyUser, ldapReadonlyPassword))

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
		req, _ := http.NewRequest("DELETE", ldapTestServer.URL+"/subjects/ldap-test-subject", nil)
		req.Header.Set("Authorization", basicAuth(ldapReadonlyUser, ldapReadonlyPassword))

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
		req, _ := http.NewRequest("DELETE", ldapTestServer.URL+"/subjects/ldap-dev-subject", nil)
		req.Header.Set("Authorization", basicAuth(ldapDevUser, ldapDevPassword))

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
		req, _ := http.NewRequest("DELETE", ldapTestServer.URL+"/subjects/ldap-test-subject", nil)
		req.Header.Set("Authorization", basicAuth(ldapAdminUser, ldapAdminPassword))

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

// TestLDAPConfigAccess tests access to configuration endpoints.
func TestLDAPConfigAccess(t *testing.T) {
	t.Run("AdminCanReadConfig", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ldapTestServer.URL+"/config", nil)
		req.Header.Set("Authorization", basicAuth(ldapAdminUser, ldapAdminPassword))

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

		req, _ := http.NewRequest("PUT", ldapTestServer.URL+"/config", bytes.NewBuffer(configBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", basicAuth(ldapAdminUser, ldapAdminPassword))

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
		req, _ := http.NewRequest("GET", ldapTestServer.URL+"/config", nil)
		req.Header.Set("Authorization", basicAuth(ldapReadonlyUser, ldapReadonlyPassword))

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

		req, _ := http.NewRequest("PUT", ldapTestServer.URL+"/config", bytes.NewBuffer(configBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", basicAuth(ldapReadonlyUser, ldapReadonlyPassword))

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
		req, _ := http.NewRequest("GET", ldapTestServer.URL+"/config", nil)
		req.Header.Set("Authorization", basicAuth(ldapDevUser, ldapDevPassword))

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

		req, _ := http.NewRequest("PUT", ldapTestServer.URL+"/config", bytes.NewBuffer(configBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", basicAuth(ldapDevUser, ldapDevPassword))

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

// TestLDAPModeAccess tests access to mode endpoints.
func TestLDAPModeAccess(t *testing.T) {
	t.Run("AdminCanReadMode", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ldapTestServer.URL+"/mode", nil)
		req.Header.Set("Authorization", basicAuth(ldapAdminUser, ldapAdminPassword))

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

		req, _ := http.NewRequest("PUT", ldapTestServer.URL+"/mode", bytes.NewBuffer(modeBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", basicAuth(ldapAdminUser, ldapAdminPassword))

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

		req, _ := http.NewRequest("PUT", ldapTestServer.URL+"/mode", bytes.NewBuffer(modeBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", basicAuth(ldapReadonlyUser, ldapReadonlyPassword))

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
