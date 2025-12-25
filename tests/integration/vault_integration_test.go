//go:build vault

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
	"github.com/axonops/axonops-schema-registry/internal/storage/vault"
)

var (
	vaultTestServer  *httptest.Server
	vaultStore       *vault.Store
	vaultAuthService *auth.Service
	schemaStore      storage.Storage
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Create in-memory storage for schemas
	schemaStore = memory.NewStore()

	// Create Vault store for auth
	vaultAddr := getEnvOrDefault("VAULT_ADDR", "http://localhost:8200")
	vaultToken := getEnvOrDefault("VAULT_TOKEN", "root")

	vaultCfg := vault.Config{
		Address:   vaultAddr,
		Token:     vaultToken,
		MountPath: "secret",
		BasePath:  "schema-registry-test",
	}

	var err error
	vaultStore, err = vault.NewStore(vaultCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Vault store: %v\n", err)
		os.Exit(1)
	}

	// Wait for Vault to be ready
	if err := waitForVault(ctx, vaultStore); err != nil {
		fmt.Fprintf(os.Stderr, "Vault is not ready: %v\n", err)
		os.Exit(1)
	}

	// Create auth service with Vault storage
	vaultAuthService = auth.NewServiceWithConfig(vaultStore, auth.ServiceConfig{
		APIKeySecret: "vaulttestsecret12345678901234567",
		APIKeyPrefix: "sr_vault_",
	})

	// Run tests
	code := m.Run()

	// Cleanup
	vaultAuthService.Close()
	vaultStore.Close()
	schemaStore.Close()

	os.Exit(code)
}

// waitForVault waits for Vault to be ready.
func waitForVault(ctx context.Context, store *vault.Store) error {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		if store.IsHealthy(ctx) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("Vault not ready after %d retries", maxRetries)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestVaultUserManagement tests user CRUD operations using Vault storage.
func TestVaultUserManagement(t *testing.T) {
	ctx := context.Background()

	// Cleanup any existing test users
	cleanupVaultUsers(t, ctx)

	t.Run("CreateUser", func(t *testing.T) {
		user := &storage.UserRecord{
			Username:     "vault_user1",
			Email:        "vault_user1@example.com",
			PasswordHash: mustHashPasswordVault(t, "password123"),
			Role:         "developer",
			Enabled:      true,
		}

		err := vaultStore.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.ID == 0 {
			t.Error("Expected user ID to be set after creation")
		}
		t.Logf("Created user with ID: %d", user.ID)
	})

	t.Run("GetUserByUsername", func(t *testing.T) {
		user, err := vaultStore.GetUserByUsername(ctx, "vault_user1")
		if err != nil {
			t.Fatalf("Failed to get user by username: %v", err)
		}

		if user.Username != "vault_user1" {
			t.Errorf("Expected username 'vault_user1', got '%s'", user.Username)
		}
		if user.Email != "vault_user1@example.com" {
			t.Errorf("Expected email 'vault_user1@example.com', got '%s'", user.Email)
		}
		if user.Role != "developer" {
			t.Errorf("Expected role 'developer', got '%s'", user.Role)
		}
	})

	t.Run("GetUserByID", func(t *testing.T) {
		user, err := vaultStore.GetUserByUsername(ctx, "vault_user1")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		fetchedUser, err := vaultStore.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}

		if fetchedUser.Username != user.Username {
			t.Errorf("Expected username '%s', got '%s'", user.Username, fetchedUser.Username)
		}
	})

	t.Run("UpdateUser", func(t *testing.T) {
		user, err := vaultStore.GetUserByUsername(ctx, "vault_user1")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		user.Role = "admin"
		user.Email = "vault_user1_updated@example.com"

		err = vaultStore.UpdateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		updated, err := vaultStore.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get updated user: %v", err)
		}

		if updated.Role != "admin" {
			t.Errorf("Expected role 'admin', got '%s'", updated.Role)
		}
		if updated.Email != "vault_user1_updated@example.com" {
			t.Errorf("Expected updated email, got '%s'", updated.Email)
		}
	})

	t.Run("ListUsers", func(t *testing.T) {
		// Create additional user
		user2 := &storage.UserRecord{
			Username:     "vault_user2",
			Email:        "vault_user2@example.com",
			PasswordHash: mustHashPasswordVault(t, "password123"),
			Role:         "readonly",
			Enabled:      true,
		}
		err := vaultStore.CreateUser(ctx, user2)
		if err != nil {
			t.Fatalf("Failed to create second user: %v", err)
		}

		users, err := vaultStore.ListUsers(ctx)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(users) < 2 {
			t.Errorf("Expected at least 2 users, got %d", len(users))
		}
	})

	t.Run("CreateDuplicateUser", func(t *testing.T) {
		user := &storage.UserRecord{
			Username:     "vault_user1",
			Email:        "different@example.com",
			PasswordHash: mustHashPasswordVault(t, "password123"),
			Role:         "developer",
			Enabled:      true,
		}

		err := vaultStore.CreateUser(ctx, user)
		if err != storage.ErrUserExists {
			t.Errorf("Expected ErrUserExists, got: %v", err)
		}
	})

	t.Run("GetNonExistentUser", func(t *testing.T) {
		_, err := vaultStore.GetUserByUsername(ctx, "nonexistent_vault_user")
		if err != storage.ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got: %v", err)
		}
	})

	t.Run("DeleteUser", func(t *testing.T) {
		user, err := vaultStore.GetUserByUsername(ctx, "vault_user2")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		err = vaultStore.DeleteUser(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		_, err = vaultStore.GetUserByID(ctx, user.ID)
		if err != storage.ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound after delete, got: %v", err)
		}
	})
}

// TestVaultAPIKeyManagement tests API key CRUD operations using Vault storage.
func TestVaultAPIKeyManagement(t *testing.T) {
	ctx := context.Background()

	// Create a user for API keys
	user := &storage.UserRecord{
		Username:     "vault_apikey_user",
		Email:        "vault_apikey@example.com",
		PasswordHash: mustHashPasswordVault(t, "password123"),
		Role:         "developer",
		Enabled:      true,
	}

	existingUser, err := vaultStore.GetUserByUsername(ctx, user.Username)
	if err != nil {
		err = vaultStore.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	} else {
		user = existingUser
	}

	var apiKeyID int64

	t.Run("CreateAPIKey", func(t *testing.T) {
		key := &storage.APIKeyRecord{
			UserID:    user.ID,
			KeyHash:   "vault_test_hash_123",
			KeyPrefix: "sr_vault_",
			Name:      "vault-test-key",
			Role:      "developer",
			Enabled:   true,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err := vaultStore.CreateAPIKey(ctx, key)
		if err != nil {
			t.Fatalf("Failed to create API key: %v", err)
		}

		if key.ID == 0 {
			t.Error("Expected API key ID to be set after creation")
		}
		apiKeyID = key.ID
		t.Logf("Created API key with ID: %d", key.ID)
	})

	t.Run("GetAPIKeyByID", func(t *testing.T) {
		key, err := vaultStore.GetAPIKeyByID(ctx, apiKeyID)
		if err != nil {
			t.Fatalf("Failed to get API key by ID: %v", err)
		}

		if key.Name != "vault-test-key" {
			t.Errorf("Expected name 'vault-test-key', got '%s'", key.Name)
		}
		if key.Role != "developer" {
			t.Errorf("Expected role 'developer', got '%s'", key.Role)
		}
	})

	t.Run("GetAPIKeyByHash", func(t *testing.T) {
		key, err := vaultStore.GetAPIKeyByHash(ctx, "vault_test_hash_123")
		if err != nil {
			t.Fatalf("Failed to get API key by hash: %v", err)
		}

		if key.ID != apiKeyID {
			t.Errorf("Expected ID %d, got %d", apiKeyID, key.ID)
		}
	})

	t.Run("GetAPIKeyByUserAndName", func(t *testing.T) {
		key, err := vaultStore.GetAPIKeyByUserAndName(ctx, user.ID, "vault-test-key")
		if err != nil {
			t.Fatalf("Failed to get API key by user and name: %v", err)
		}

		if key.ID != apiKeyID {
			t.Errorf("Expected ID %d, got %d", apiKeyID, key.ID)
		}
	})

	t.Run("UpdateAPIKey", func(t *testing.T) {
		key, err := vaultStore.GetAPIKeyByID(ctx, apiKeyID)
		if err != nil {
			t.Fatalf("Failed to get API key: %v", err)
		}

		key.Role = "admin"
		err = vaultStore.UpdateAPIKey(ctx, key)
		if err != nil {
			t.Fatalf("Failed to update API key: %v", err)
		}

		updated, err := vaultStore.GetAPIKeyByID(ctx, apiKeyID)
		if err != nil {
			t.Fatalf("Failed to get updated API key: %v", err)
		}

		if updated.Role != "admin" {
			t.Errorf("Expected role 'admin', got '%s'", updated.Role)
		}
	})

	t.Run("UpdateAPIKeyLastUsed", func(t *testing.T) {
		err := vaultStore.UpdateAPIKeyLastUsed(ctx, apiKeyID)
		if err != nil {
			t.Fatalf("Failed to update last used: %v", err)
		}

		key, err := vaultStore.GetAPIKeyByID(ctx, apiKeyID)
		if err != nil {
			t.Fatalf("Failed to get API key: %v", err)
		}

		if key.LastUsed == nil {
			t.Error("Expected LastUsed to be set")
		}
	})

	t.Run("ListAPIKeysByUserID", func(t *testing.T) {
		// Create another API key for the same user
		key2 := &storage.APIKeyRecord{
			UserID:    user.ID,
			KeyHash:   "vault_test_hash_456",
			KeyPrefix: "sr_vault_",
			Name:      "vault-test-key-2",
			Role:      "readonly",
			Enabled:   true,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := vaultStore.CreateAPIKey(ctx, key2)
		if err != nil {
			t.Fatalf("Failed to create second API key: %v", err)
		}

		keys, err := vaultStore.ListAPIKeysByUserID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to list API keys by user ID: %v", err)
		}

		if len(keys) < 2 {
			t.Errorf("Expected at least 2 API keys, got %d", len(keys))
		}
	})

	t.Run("ListAPIKeys", func(t *testing.T) {
		keys, err := vaultStore.ListAPIKeys(ctx)
		if err != nil {
			t.Fatalf("Failed to list API keys: %v", err)
		}

		if len(keys) < 2 {
			t.Errorf("Expected at least 2 API keys, got %d", len(keys))
		}
	})

	t.Run("CreateDuplicateAPIKey", func(t *testing.T) {
		key := &storage.APIKeyRecord{
			UserID:    user.ID,
			KeyHash:   "vault_test_hash_123", // Same hash as first key
			KeyPrefix: "sr_vault_",
			Name:      "duplicate-key",
			Role:      "developer",
			Enabled:   true,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err := vaultStore.CreateAPIKey(ctx, key)
		if err != storage.ErrAPIKeyExists {
			t.Errorf("Expected ErrAPIKeyExists, got: %v", err)
		}
	})

	t.Run("GetNonExistentAPIKey", func(t *testing.T) {
		_, err := vaultStore.GetAPIKeyByHash(ctx, "nonexistent_hash")
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("Expected ErrAPIKeyNotFound, got: %v", err)
		}
	})

	t.Run("DeleteAPIKey", func(t *testing.T) {
		err := vaultStore.DeleteAPIKey(ctx, apiKeyID)
		if err != nil {
			t.Fatalf("Failed to delete API key: %v", err)
		}

		_, err = vaultStore.GetAPIKeyByID(ctx, apiKeyID)
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("Expected ErrAPIKeyNotFound after delete, got: %v", err)
		}
	})
}

// TestVaultPasswordAuthentication tests password authentication via Vault-stored users.
func TestVaultPasswordAuthentication(t *testing.T) {
	ctx := context.Background()

	t.Run("CreateUserViaService", func(t *testing.T) {
		_, err := vaultAuthService.CreateUser(ctx, auth.CreateUserRequest{
			Username: "vault_auth_user",
			Email:    "vault_auth@example.com",
			Password: "SecurePassword123!",
			Role:     "developer",
			Enabled:  true,
		})
		if err != nil && err != storage.ErrUserExists {
			t.Fatalf("Failed to create user: %v", err)
		}
	})

	t.Run("ValidCredentials", func(t *testing.T) {
		user, err := vaultAuthService.ValidateCredentials(ctx, "vault_auth_user", "SecurePassword123!")
		if err != nil {
			t.Fatalf("Failed to validate credentials: %v", err)
		}

		if user.Username != "vault_auth_user" {
			t.Errorf("Expected username 'vault_auth_user', got '%s'", user.Username)
		}
		if user.Role != "developer" {
			t.Errorf("Expected role 'developer', got '%s'", user.Role)
		}
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		_, err := vaultAuthService.ValidateCredentials(ctx, "vault_auth_user", "WrongPassword")
		if err == nil {
			t.Error("Expected error for invalid password")
		}
	})

	t.Run("NonExistentUser", func(t *testing.T) {
		_, err := vaultAuthService.ValidateCredentials(ctx, "nonexistent_user", "anypassword")
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})

	t.Run("ChangePassword", func(t *testing.T) {
		user, _ := vaultStore.GetUserByUsername(ctx, "vault_auth_user")
		err := vaultAuthService.ChangePassword(ctx, user.ID, "SecurePassword123!", "NewPassword456!")
		if err != nil {
			t.Fatalf("Failed to change password: %v", err)
		}

		// Verify new password works
		_, err = vaultAuthService.ValidateCredentials(ctx, "vault_auth_user", "NewPassword456!")
		if err != nil {
			t.Errorf("Failed to authenticate with new password: %v", err)
		}

		// Verify old password doesn't work
		_, err = vaultAuthService.ValidateCredentials(ctx, "vault_auth_user", "SecurePassword123!")
		if err == nil {
			t.Error("Old password should not work after change")
		}
	})

	t.Run("DisabledUser", func(t *testing.T) {
		user, _ := vaultStore.GetUserByUsername(ctx, "vault_auth_user")
		user.Enabled = false
		_ = vaultStore.UpdateUser(ctx, user)

		_, err := vaultAuthService.ValidateCredentials(ctx, "vault_auth_user", "NewPassword456!")
		if err == nil {
			t.Error("Expected error for disabled user")
		}

		// Re-enable for cleanup
		user.Enabled = true
		_ = vaultStore.UpdateUser(ctx, user)
	})
}

// TestVaultRoleBasedSchemaAccess tests RBAC using Vault-stored API keys.
func TestVaultRoleBasedSchemaAccess(t *testing.T) {
	ctx := context.Background()

	// Create users for each role
	roles := []string{"admin", "developer", "readonly"}
	users := make(map[string]*storage.UserRecord)
	apiKeys := make(map[string]string)

	for _, role := range roles {
		username := fmt.Sprintf("vault_rbac_%s", role)
		user, err := vaultAuthService.CreateUser(ctx, auth.CreateUserRequest{
			Username: username,
			Email:    fmt.Sprintf("vault_rbac_%s@example.com", role),
			Password: "TestPassword123!",
			Role:     role,
			Enabled:  true,
		})
		if err != nil {
			existing, _ := vaultStore.GetUserByUsername(ctx, username)
			if existing != nil {
				users[role] = existing
			} else {
				t.Fatalf("Failed to create user for role %s: %v", role, err)
			}
		} else {
			users[role] = user
		}

		// Create API key for user
		response, err := vaultAuthService.CreateAPIKey(ctx, auth.CreateAPIKeyRequest{
			UserID:    users[role].ID,
			Name:      fmt.Sprintf("vault-rbac-key-%s", role),
			Role:      role,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("Failed to create API key for role %s: %v", role, err)
		}
		apiKeys[role] = response.Key
		t.Logf("Created API key for role %s: %s...", role, response.Key[:20])
	}

	// Create schema parser registry
	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	// Create compatibility checker
	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	// Create registry with in-memory schema storage
	reg := registry.New(schemaStore, schemaRegistry, compatChecker, "BACKWARD")

	// Create server with API key auth
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8084,
		},
		Security: config.SecurityConfig{
			Auth: config.AuthConfig{
				Enabled: true,
				Methods: []string{"api_key"},
				APIKey: config.APIKeyConfig{
					Header:      "X-API-Key",
					StorageType: "database",
					Secret:      "vaulttestsecret12345678901234567",
					KeyPrefix:   "sr_vault_",
				},
				RBAC: config.RBACConfig{
					Enabled:     true,
					DefaultRole: "readonly",
				},
			},
		},
	}

	authenticator := auth.NewAuthenticator(cfg.Security.Auth)
	authenticator.SetService(vaultAuthService)
	authorizer := auth.NewAuthorizer(cfg.Security.Auth.RBAC)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := api.NewServer(cfg, reg, logger, api.WithAuth(authenticator, authorizer, vaultAuthService))
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Helper function
	makeRequest := func(method, path string, body interface{}, apiKey string) *http.Response {
		var bodyReader io.Reader
		if body != nil {
			bodyBytes, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, _ := http.NewRequest(method, ts.URL+path, bodyReader)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if apiKey != "" {
			req.Header.Set("X-API-Key", apiKey)
		}

		resp, _ := http.DefaultClient.Do(req)
		return resp
	}

	testSchema := `{"type":"record","name":"VaultRBACTest","fields":[{"name":"id","type":"int"}]}`

	// Test schema registration
	t.Run("AdminAPIKeyCanRegisterSchema", func(t *testing.T) {
		resp := makeRequest("POST", "/subjects/vault-admin-subject/versions",
			map[string]string{"schema": testSchema}, apiKeys["admin"])

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected admin to register schema, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("DeveloperAPIKeyCanRegisterSchema", func(t *testing.T) {
		devSchema := `{"type":"record","name":"VaultDevTest","fields":[{"name":"id","type":"int"}]}`
		resp := makeRequest("POST", "/subjects/vault-dev-subject/versions",
			map[string]string{"schema": devSchema}, apiKeys["developer"])

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected developer to register schema, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("ReadonlyAPIKeyCannotRegisterSchema", func(t *testing.T) {
		roSchema := `{"type":"record","name":"VaultROTest","fields":[{"name":"id","type":"int"}]}`
		resp := makeRequest("POST", "/subjects/vault-readonly-subject/versions",
			map[string]string{"schema": roSchema}, apiKeys["readonly"])

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected readonly to be forbidden, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	// Test schema reading
	t.Run("AllRolesCanReadSchema", func(t *testing.T) {
		for role, apiKey := range apiKeys {
			resp := makeRequest("GET", "/subjects/vault-admin-subject/versions/1", nil, apiKey)
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected %s to read schema, got status %d: %s", role, resp.StatusCode, body)
			}
			resp.Body.Close()
		}
	})

	t.Run("AllRolesCanListSubjects", func(t *testing.T) {
		for role, apiKey := range apiKeys {
			resp := makeRequest("GET", "/subjects", nil, apiKey)
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected %s to list subjects, got status %d: %s", role, resp.StatusCode, body)
			}
			resp.Body.Close()
		}
	})

	// Test schema deletion
	t.Run("ReadonlyAPIKeyCannotDeleteSubject", func(t *testing.T) {
		resp := makeRequest("DELETE", "/subjects/vault-admin-subject", nil, apiKeys["readonly"])

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected readonly to be forbidden from delete, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("DeveloperAPIKeyCannotDeleteSubject", func(t *testing.T) {
		resp := makeRequest("DELETE", "/subjects/vault-dev-subject", nil, apiKeys["developer"])

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected developer to be forbidden from delete, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("AdminAPIKeyCanDeleteSubject", func(t *testing.T) {
		resp := makeRequest("DELETE", "/subjects/vault-admin-subject", nil, apiKeys["admin"])

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected admin to delete subject, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	// Test config access
	t.Run("AdminAPIKeyCanSetConfig", func(t *testing.T) {
		resp := makeRequest("PUT", "/config",
			map[string]string{"compatibility": "FULL"}, apiKeys["admin"])

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected admin to set config, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("DeveloperAPIKeyCannotSetConfig", func(t *testing.T) {
		resp := makeRequest("PUT", "/config",
			map[string]string{"compatibility": "NONE"}, apiKeys["developer"])

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected developer to be forbidden from setting config, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("ReadonlyAPIKeyCannotSetConfig", func(t *testing.T) {
		resp := makeRequest("PUT", "/config",
			map[string]string{"compatibility": "BACKWARD_TRANSITIVE"}, apiKeys["readonly"])

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected readonly to be forbidden from setting config, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("AllRolesCanReadConfig", func(t *testing.T) {
		for role, apiKey := range apiKeys {
			resp := makeRequest("GET", "/config", nil, apiKey)
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected %s to read config, got status %d: %s", role, resp.StatusCode, body)
			}
			resp.Body.Close()
		}
	})

	// Test mode access
	t.Run("AdminAPIKeyCanSetMode", func(t *testing.T) {
		resp := makeRequest("PUT", "/mode",
			map[string]string{"mode": "READWRITE"}, apiKeys["admin"])

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected admin to set mode, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("ReadonlyAPIKeyCannotSetMode", func(t *testing.T) {
		resp := makeRequest("PUT", "/mode",
			map[string]string{"mode": "READONLY"}, apiKeys["readonly"])

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected readonly to be forbidden from setting mode, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	// Test invalid API key
	t.Run("InvalidAPIKeyFails", func(t *testing.T) {
		resp := makeRequest("GET", "/subjects", nil, "invalid-api-key")

		if resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected invalid API key to fail with 401, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("MissingAPIKeyFails", func(t *testing.T) {
		resp := makeRequest("GET", "/subjects", nil, "")

		if resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected missing API key to fail with 401, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	// Cleanup
	t.Cleanup(func() {
		for _, user := range users {
			keys, _ := vaultAuthService.ListAPIKeysByUserID(ctx, user.ID)
			for _, key := range keys {
				_ = vaultStore.DeleteAPIKey(ctx, key.ID)
			}
			_ = vaultStore.DeleteUser(ctx, user.ID)
		}
	})
}

// TestVaultHealthCheck tests the Vault health check functionality.
func TestVaultHealthCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("VaultIsHealthy", func(t *testing.T) {
		if !vaultStore.IsHealthy(ctx) {
			t.Error("Expected Vault to be healthy")
		}
	})
}

// Helper functions

func mustHashPasswordVault(t *testing.T, password string) string {
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	return hash
}

func cleanupVaultUsers(t *testing.T, ctx context.Context) {
	users, err := vaultStore.ListUsers(ctx)
	if err != nil {
		return
	}

	testPrefixes := []string{"vault_user", "vault_apikey", "vault_auth", "vault_rbac"}

	for _, user := range users {
		for _, prefix := range testPrefixes {
			if len(user.Username) >= len(prefix) && user.Username[:len(prefix)] == prefix {
				// Delete API keys first
				keys, _ := vaultStore.ListAPIKeysByUserID(ctx, user.ID)
				for _, key := range keys {
					_ = vaultStore.DeleteAPIKey(ctx, key.ID)
				}
				_ = vaultStore.DeleteUser(ctx, user.ID)
				break
			}
		}
	}
}
