//go:build integration

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
)

// TestAuthUserManagement tests user CRUD operations for each storage backend.
func TestAuthUserManagement(t *testing.T) {
	if testStore == nil {
		t.Skip("No storage backend available")
	}

	ctx := context.Background()

	// Cleanup any existing test users
	cleanupTestUsers(t, ctx)

	t.Run("CreateUser", func(t *testing.T) {
		user := &storage.UserRecord{
			Username:     "testuser1",
			Email:        "testuser1@example.com",
			PasswordHash: mustHashPassword(t, "password123"),
			Role:         "developer",
			Enabled:      true,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		err := testStore.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.ID == 0 {
			t.Error("Expected user ID to be set after creation")
		}
	})

	t.Run("GetUserByUsername", func(t *testing.T) {
		user, err := testStore.GetUserByUsername(ctx, "testuser1")
		if err != nil {
			t.Fatalf("Failed to get user by username: %v", err)
		}

		if user.Username != "testuser1" {
			t.Errorf("Expected username 'testuser1', got '%s'", user.Username)
		}
		if user.Email != "testuser1@example.com" {
			t.Errorf("Expected email 'testuser1@example.com', got '%s'", user.Email)
		}
		if user.Role != "developer" {
			t.Errorf("Expected role 'developer', got '%s'", user.Role)
		}
	})

	t.Run("GetUserByID", func(t *testing.T) {
		user, err := testStore.GetUserByUsername(ctx, "testuser1")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		fetchedUser, err := testStore.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}

		if fetchedUser.Username != user.Username {
			t.Errorf("Expected username '%s', got '%s'", user.Username, fetchedUser.Username)
		}
	})

	t.Run("UpdateUser", func(t *testing.T) {
		user, err := testStore.GetUserByUsername(ctx, "testuser1")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		user.Email = "updated@example.com"
		user.Role = "admin"
		user.UpdatedAt = time.Now().UTC()

		err = testStore.UpdateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		updated, err := testStore.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get updated user: %v", err)
		}

		if updated.Email != "updated@example.com" {
			t.Errorf("Expected email 'updated@example.com', got '%s'", updated.Email)
		}
		if updated.Role != "admin" {
			t.Errorf("Expected role 'admin', got '%s'", updated.Role)
		}
	})

	t.Run("ListUsers", func(t *testing.T) {
		// Create additional users
		for i := 2; i <= 3; i++ {
			user := &storage.UserRecord{
				Username:     fmt.Sprintf("testuser%d", i),
				Email:        fmt.Sprintf("testuser%d@example.com", i),
				PasswordHash: mustHashPassword(t, "password123"),
				Role:         "readonly",
				Enabled:      true,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			}
			if err := testStore.CreateUser(ctx, user); err != nil {
				t.Fatalf("Failed to create user %d: %v", i, err)
			}
		}

		users, err := testStore.ListUsers(ctx)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		// Count test users
		testUserCount := 0
		for _, u := range users {
			if u.Username == "testuser1" || u.Username == "testuser2" || u.Username == "testuser3" {
				testUserCount++
			}
		}

		if testUserCount < 3 {
			t.Errorf("Expected at least 3 test users, got %d", testUserCount)
		}
	})

	t.Run("DeleteUser", func(t *testing.T) {
		user, err := testStore.GetUserByUsername(ctx, "testuser3")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		err = testStore.DeleteUser(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		_, err = testStore.GetUserByUsername(ctx, "testuser3")
		if err == nil {
			t.Error("Expected error when getting deleted user")
		}
	})

	t.Run("CreateDuplicateUser", func(t *testing.T) {
		user := &storage.UserRecord{
			Username:     "testuser1", // Already exists
			Email:        "duplicate@example.com",
			PasswordHash: mustHashPassword(t, "password123"),
			Role:         "readonly",
			Enabled:      true,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		err := testStore.CreateUser(ctx, user)
		if err == nil {
			t.Error("Expected error when creating duplicate user")
		}
	})

	t.Run("GetNonExistentUser", func(t *testing.T) {
		_, err := testStore.GetUserByUsername(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error when getting non-existent user")
		}
	})
}

// TestAuthAPIKeyManagement tests API key CRUD operations.
func TestAuthAPIKeyManagement(t *testing.T) {
	if testStore == nil {
		t.Skip("No storage backend available")
	}

	ctx := context.Background()

	// Ensure we have a test user
	user, err := testStore.GetUserByUsername(ctx, "testuser1")
	if err != nil {
		// Create one if it doesn't exist
		user = &storage.UserRecord{
			Username:     "testuser1",
			Email:        "testuser1@example.com",
			PasswordHash: mustHashPassword(t, "password123"),
			Role:         "developer",
			Enabled:      true,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}
		if err := testStore.CreateUser(ctx, user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	var createdKeyID int64
	keyHash := "testhash123456789"

	t.Run("CreateAPIKey", func(t *testing.T) {
		apiKey := &storage.APIKeyRecord{
			UserID:    user.ID,
			KeyHash:   keyHash,
			KeyPrefix: "test1234",
			Name:      "test-key",
			Role:      "developer",
			Enabled:   true,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
		}

		err := testStore.CreateAPIKey(ctx, apiKey)
		if err != nil {
			t.Fatalf("Failed to create API key: %v", err)
		}

		if apiKey.ID == 0 {
			t.Error("Expected API key ID to be set after creation")
		}
		createdKeyID = apiKey.ID
	})

	t.Run("GetAPIKeyByHash", func(t *testing.T) {
		apiKey, err := testStore.GetAPIKeyByHash(ctx, keyHash)
		if err != nil {
			t.Fatalf("Failed to get API key by hash: %v", err)
		}

		if apiKey.Name != "test-key" {
			t.Errorf("Expected name 'test-key', got '%s'", apiKey.Name)
		}
		if apiKey.Role != "developer" {
			t.Errorf("Expected role 'developer', got '%s'", apiKey.Role)
		}
	})

	t.Run("GetAPIKeyByID", func(t *testing.T) {
		apiKey, err := testStore.GetAPIKeyByID(ctx, createdKeyID)
		if err != nil {
			t.Fatalf("Failed to get API key by ID: %v", err)
		}

		if apiKey.KeyHash != keyHash {
			t.Errorf("Expected key hash '%s', got '%s'", keyHash, apiKey.KeyHash)
		}
	})

	t.Run("GetAPIKeyByUserAndName", func(t *testing.T) {
		apiKey, err := testStore.GetAPIKeyByUserAndName(ctx, user.ID, "test-key")
		if err != nil {
			t.Fatalf("Failed to get API key by user and name: %v", err)
		}

		if apiKey.ID != createdKeyID {
			t.Errorf("Expected API key ID %d, got %d", createdKeyID, apiKey.ID)
		}
	})

	t.Run("UpdateAPIKey", func(t *testing.T) {
		apiKey, err := testStore.GetAPIKeyByID(ctx, createdKeyID)
		if err != nil {
			t.Fatalf("Failed to get API key: %v", err)
		}

		apiKey.Role = "admin"
		apiKey.Enabled = false

		err = testStore.UpdateAPIKey(ctx, apiKey)
		if err != nil {
			t.Fatalf("Failed to update API key: %v", err)
		}

		updated, err := testStore.GetAPIKeyByID(ctx, createdKeyID)
		if err != nil {
			t.Fatalf("Failed to get updated API key: %v", err)
		}

		if updated.Role != "admin" {
			t.Errorf("Expected role 'admin', got '%s'", updated.Role)
		}
		if updated.Enabled {
			t.Error("Expected Enabled to be false")
		}
	})

	t.Run("UpdateAPIKeyLastUsed", func(t *testing.T) {
		err := testStore.UpdateAPIKeyLastUsed(ctx, createdKeyID)
		if err != nil {
			t.Fatalf("Failed to update last used: %v", err)
		}

		apiKey, err := testStore.GetAPIKeyByID(ctx, createdKeyID)
		if err != nil {
			t.Fatalf("Failed to get API key: %v", err)
		}

		if apiKey.LastUsed == nil {
			t.Error("Expected LastUsed to be set")
		}
	})

	t.Run("ListAPIKeysByUserID", func(t *testing.T) {
		// Create another key for the same user
		apiKey2 := &storage.APIKeyRecord{
			UserID:    user.ID,
			KeyHash:   "anotherhash987654321",
			KeyPrefix: "test5678",
			Name:      "test-key-2",
			Role:      "readonly",
			Enabled:   true,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
		}
		if err := testStore.CreateAPIKey(ctx, apiKey2); err != nil {
			t.Fatalf("Failed to create second API key: %v", err)
		}

		keys, err := testStore.ListAPIKeysByUserID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to list API keys by user: %v", err)
		}

		if len(keys) < 2 {
			t.Errorf("Expected at least 2 API keys, got %d", len(keys))
		}
	})

	t.Run("DeleteAPIKey", func(t *testing.T) {
		err := testStore.DeleteAPIKey(ctx, createdKeyID)
		if err != nil {
			t.Fatalf("Failed to delete API key: %v", err)
		}

		_, err = testStore.GetAPIKeyByID(ctx, createdKeyID)
		if err == nil {
			t.Error("Expected error when getting deleted API key")
		}
	})
}

// TestAuthPasswordAuthentication tests password-based authentication flow.
func TestAuthPasswordAuthentication(t *testing.T) {
	if testStore == nil {
		t.Skip("No storage backend available")
	}

	ctx := context.Background()

	// Create auth service
	authService := auth.NewService(testStore)
	defer authService.Close()

	// Create test user via service
	t.Run("CreateUserViaService", func(t *testing.T) {
		_, err := authService.CreateUser(ctx, auth.CreateUserRequest{
			Username: "authtest_user",
			Email:    "authtest@example.com",
			Password: "SecurePassword123!",
			Role:     "developer",
			Enabled:  true,
		})
		if err != nil {
			t.Fatalf("Failed to create user via service: %v", err)
		}
	})

	t.Run("ValidCredentials", func(t *testing.T) {
		user, err := authService.ValidateCredentials(ctx, "authtest_user", "SecurePassword123!")
		if err != nil {
			t.Fatalf("Failed to validate credentials: %v", err)
		}

		if user.Username != "authtest_user" {
			t.Errorf("Expected username 'authtest_user', got '%s'", user.Username)
		}
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		_, err := authService.ValidateCredentials(ctx, "authtest_user", "WrongPassword")
		if err == nil {
			t.Error("Expected error for invalid password")
		}
	})

	t.Run("NonExistentUser", func(t *testing.T) {
		_, err := authService.ValidateCredentials(ctx, "nonexistent_user", "password")
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})

	t.Run("ChangePassword", func(t *testing.T) {
		user, err := authService.GetUserByUsername(ctx, "authtest_user")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		err = authService.ChangePassword(ctx, user.ID, "SecurePassword123!", "NewPassword456!")
		if err != nil {
			t.Fatalf("Failed to change password: %v", err)
		}

		// Old password should fail
		_, err = authService.ValidateCredentials(ctx, "authtest_user", "SecurePassword123!")
		if err == nil {
			t.Error("Expected old password to fail")
		}

		// New password should work
		_, err = authService.ValidateCredentials(ctx, "authtest_user", "NewPassword456!")
		if err != nil {
			t.Errorf("Expected new password to work: %v", err)
		}
	})

	t.Run("DisabledUser", func(t *testing.T) {
		user, err := authService.GetUserByUsername(ctx, "authtest_user")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		// Disable user
		_, err = authService.UpdateUser(ctx, user.ID, map[string]interface{}{
			"enabled": false,
		})
		if err != nil {
			t.Fatalf("Failed to disable user: %v", err)
		}

		// Should fail authentication
		_, err = authService.ValidateCredentials(ctx, "authtest_user", "NewPassword456!")
		if err == nil {
			t.Error("Expected disabled user authentication to fail")
		}
	})
}

// TestAuthRoleBasedSchemaAccess tests role-based access control for schema operations.
func TestAuthRoleBasedSchemaAccess(t *testing.T) {
	if testStore == nil {
		t.Skip("No storage backend available")
	}

	ctx := context.Background()

	// Create auth service
	authService := auth.NewService(testStore)
	defer authService.Close()

	// Create users with different roles
	roles := []string{"admin", "developer", "readonly"}
	users := make(map[string]*storage.UserRecord)

	for _, role := range roles {
		username := fmt.Sprintf("rbac_%s_user", role)
		user, err := authService.CreateUser(ctx, auth.CreateUserRequest{
			Username: username,
			Email:    fmt.Sprintf("%s@example.com", username),
			Password: "TestPassword123!",
			Role:     role,
			Enabled:  true,
		})
		if err != nil {
			// User might already exist
			user, err = authService.GetUserByUsername(ctx, username)
			if err != nil {
				t.Fatalf("Failed to create/get user with role %s: %v", role, err)
			}
		}
		users[role] = user
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

	// Create registry
	reg := registry.New(testStore, schemaRegistry, compatChecker, "BACKWARD")

	// Create server with auth enabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8082,
		},
		Security: config.SecurityConfig{
			Auth: config.AuthConfig{
				Enabled: true,
				Methods: []string{"basic"},
				Basic: config.BasicAuthConfig{
					Realm: "Test Realm",
				},
				RBAC: config.RBACConfig{
					Enabled:     true,
					DefaultRole: "readonly",
				},
			},
		},
	}

	authenticator := auth.NewAuthenticator(cfg.Security.Auth)
	authenticator.SetService(authService)
	authorizer := auth.NewAuthorizer(cfg.Security.Auth.RBAC)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := api.NewServer(cfg, reg, logger, api.WithAuth(authenticator, authorizer, authService))
	ts := httptest.NewServer(server)
	defer ts.Close()

	// Helper function to make authenticated requests
	makeRequest := func(method, path string, body interface{}, username, password string) *http.Response {
		var bodyReader io.Reader
		if body != nil {
			bodyBytes, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, _ := http.NewRequest(method, ts.URL+path, bodyReader)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if username != "" {
			credentials := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			req.Header.Set("Authorization", "Basic "+credentials)
		}

		resp, _ := http.DefaultClient.Do(req)
		return resp
	}

	testSchema := `{"type":"record","name":"RBACTest","fields":[{"name":"id","type":"int"}]}`

	t.Run("AdminCanRegisterSchema", func(t *testing.T) {
		resp := makeRequest("POST", "/subjects/rbac-test-subject/versions",
			map[string]string{"schema": testSchema},
			"rbac_admin_user", "TestPassword123!")

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected admin to register schema, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("DeveloperCanRegisterSchema", func(t *testing.T) {
		resp := makeRequest("POST", "/subjects/rbac-test-subject-dev/versions",
			map[string]string{"schema": testSchema},
			"rbac_developer_user", "TestPassword123!")

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected developer to register schema, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("ReadonlyCannotRegisterSchema", func(t *testing.T) {
		resp := makeRequest("POST", "/subjects/rbac-test-subject-readonly/versions",
			map[string]string{"schema": testSchema},
			"rbac_readonly_user", "TestPassword123!")

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected readonly user to be forbidden, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("ReadonlyCanReadSchema", func(t *testing.T) {
		resp := makeRequest("GET", "/subjects/rbac-test-subject/versions/latest",
			nil,
			"rbac_readonly_user", "TestPassword123!")

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected readonly user to read schema, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("ReadonlyCannotDeleteSubject", func(t *testing.T) {
		resp := makeRequest("DELETE", "/subjects/rbac-test-subject",
			nil,
			"rbac_readonly_user", "TestPassword123!")

		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected readonly user to be forbidden from delete, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("AdminCanDeleteSubject", func(t *testing.T) {
		resp := makeRequest("DELETE", "/subjects/rbac-test-subject-dev",
			nil,
			"rbac_admin_user", "TestPassword123!")

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected admin to delete subject, got status %d: %s", resp.StatusCode, body)
		}
		resp.Body.Close()
	})

	t.Run("UnauthenticatedRequestFails", func(t *testing.T) {
		resp := makeRequest("GET", "/subjects", nil, "", "")

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected unauthenticated request to fail with 401, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("InvalidCredentialsFails", func(t *testing.T) {
		resp := makeRequest("GET", "/subjects", nil, "rbac_admin_user", "WrongPassword")

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected invalid credentials to fail with 401, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})
}

// TestAuthAPIKeyAuthentication tests API key-based authentication.
func TestAuthAPIKeyAuthentication(t *testing.T) {
	if testStore == nil {
		t.Skip("No storage backend available")
	}

	ctx := context.Background()

	// Create auth service with API key config
	authService := auth.NewServiceWithConfig(testStore, auth.ServiceConfig{
		APIKeySecret: "testsecret1234567890123456789012", // 32 bytes
		APIKeyPrefix: "sr_test_",
	})
	defer authService.Close()

	// Ensure we have a user for API keys
	user, err := authService.GetUserByUsername(ctx, "apikey_test_user")
	if err != nil {
		user, err = authService.CreateUser(ctx, auth.CreateUserRequest{
			Username: "apikey_test_user",
			Email:    "apikey_test@example.com",
			Password: "TestPassword123!",
			Role:     "developer",
			Enabled:  true,
		})
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	var rawAPIKey string

	t.Run("CreateAPIKey", func(t *testing.T) {
		response, err := authService.CreateAPIKey(ctx, auth.CreateAPIKeyRequest{
			UserID:    user.ID,
			Name:      "integration-test-key",
			Role:      "developer",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		if err != nil {
			t.Fatalf("Failed to create API key: %v", err)
		}

		rawAPIKey = response.Key
		if rawAPIKey == "" {
			t.Error("Expected raw API key to be returned")
		}

		// Verify prefix
		if len(rawAPIKey) > 8 && rawAPIKey[:8] != "sr_test_" {
			t.Errorf("Expected API key to have prefix 'sr_test_', got '%s'", rawAPIKey[:8])
		}
	})

	t.Run("ValidateAPIKey", func(t *testing.T) {
		apiKey, err := authService.ValidateAPIKey(ctx, rawAPIKey)
		if err != nil {
			t.Fatalf("Failed to validate API key: %v", err)
		}

		if apiKey.Name != "integration-test-key" {
			t.Errorf("Expected name 'integration-test-key', got '%s'", apiKey.Name)
		}
		if apiKey.Role != "developer" {
			t.Errorf("Expected role 'developer', got '%s'", apiKey.Role)
		}
	})

	t.Run("InvalidAPIKey", func(t *testing.T) {
		_, err := authService.ValidateAPIKey(ctx, "invalid-key-12345")
		if err == nil {
			t.Error("Expected error for invalid API key")
		}
	})

	t.Run("RevokeAPIKey", func(t *testing.T) {
		// Get the API key first
		apiKey, err := authService.ValidateAPIKey(ctx, rawAPIKey)
		if err != nil {
			t.Fatalf("Failed to get API key: %v", err)
		}

		// Revoke it
		err = authService.RevokeAPIKey(ctx, apiKey.ID)
		if err != nil {
			t.Fatalf("Failed to revoke API key: %v", err)
		}

		// Should fail validation now
		_, err = authService.ValidateAPIKey(ctx, rawAPIKey)
		if err == nil {
			t.Error("Expected revoked API key to fail validation")
		}
	})
}

// TestBootstrapAdminWorkflow tests the complete bootstrap flow:
// 1. Bootstrap initial admin user when users table is empty
// 2. Update admin password
// 3. Use admin with updated password to create other users
func TestBootstrapAdminWorkflow(t *testing.T) {
	if testStore == nil {
		t.Skip("No storage backend available")
	}

	ctx := context.Background()

	// Clean up ALL users to simulate fresh deployment
	cleanupAllUsers(t, ctx)

	// Verify users table is empty
	users, err := testStore.ListUsers(ctx)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("Expected empty users table, got %d users", len(users))
	}

	// Create auth service for bootstrap
	authService := auth.NewService(testStore)
	defer authService.Close()

	const (
		bootstrapUsername = "bootstrap_admin"
		bootstrapPassword = "InitialPassword123!"
		updatedPassword   = "UpdatedSecurePassword456!"
	)

	var bootstrapAdminID int64

	t.Run("BootstrapInitialAdmin", func(t *testing.T) {
		// This simulates what happens on server startup with bootstrap enabled
		result, err := authService.BootstrapAdmin(ctx, bootstrapUsername, bootstrapPassword, "admin@example.com")
		if err != nil {
			t.Fatalf("Bootstrap failed: %v", err)
		}

		if !result.Created {
			t.Errorf("Expected admin to be created, got message: %s", result.Message)
		}

		if result.Username != bootstrapUsername {
			t.Errorf("Expected username '%s', got '%s'", bootstrapUsername, result.Username)
		}

		// Verify admin was created with super_admin role
		admin, err := authService.GetUserByUsername(ctx, bootstrapUsername)
		if err != nil {
			t.Fatalf("Failed to get bootstrap admin: %v", err)
		}

		if admin.Role != "super_admin" {
			t.Errorf("Expected role 'super_admin', got '%s'", admin.Role)
		}

		bootstrapAdminID = admin.ID
		t.Logf("Bootstrap admin created with ID: %d", bootstrapAdminID)
	})

	t.Run("BootstrapIsIdempotent", func(t *testing.T) {
		// Running bootstrap again should not create a new user
		result, err := authService.BootstrapAdmin(ctx, "another_admin", "password", "")
		if err != nil {
			t.Fatalf("Second bootstrap call failed: %v", err)
		}

		if result.Created {
			t.Error("Expected bootstrap to skip when users exist")
		}

		// Should still have only 1 user
		users, err := testStore.ListUsers(ctx)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(users) != 1 {
			t.Errorf("Expected 1 user, got %d", len(users))
		}
	})

	t.Run("ValidateBootstrapCredentials", func(t *testing.T) {
		// Validate admin can authenticate with initial password
		user, err := authService.ValidateCredentials(ctx, bootstrapUsername, bootstrapPassword)
		if err != nil {
			t.Fatalf("Failed to validate bootstrap credentials: %v", err)
		}

		if user.Username != bootstrapUsername {
			t.Errorf("Expected username '%s', got '%s'", bootstrapUsername, user.Username)
		}
	})

	t.Run("UpdateAdminPassword", func(t *testing.T) {
		// Change the admin password (simulating admin's first action)
		err := authService.ChangePassword(ctx, bootstrapAdminID, bootstrapPassword, updatedPassword)
		if err != nil {
			t.Fatalf("Failed to change password: %v", err)
		}

		// Old password should no longer work
		_, err = authService.ValidateCredentials(ctx, bootstrapUsername, bootstrapPassword)
		if err == nil {
			t.Error("Expected old password to fail")
		}

		// New password should work
		user, err := authService.ValidateCredentials(ctx, bootstrapUsername, updatedPassword)
		if err != nil {
			t.Fatalf("Failed to validate with new password: %v", err)
		}

		t.Logf("Admin password updated successfully for user: %s", user.Username)
	})

	t.Run("AdminCreatesOtherUsers", func(t *testing.T) {
		// First verify admin is authenticated (simulating API auth check)
		admin, err := authService.ValidateCredentials(ctx, bootstrapUsername, updatedPassword)
		if err != nil {
			t.Fatalf("Admin authentication failed: %v", err)
		}

		if admin.Role != "super_admin" {
			t.Fatalf("Expected super_admin role, got '%s'", admin.Role)
		}

		// Now create other users (as admin would do via API)
		newUsers := []struct {
			username string
			role     string
		}{
			{"developer_user", "developer"},
			{"readonly_user", "readonly"},
			{"another_admin", "admin"},
		}

		for _, u := range newUsers {
			_, err := authService.CreateUser(ctx, auth.CreateUserRequest{
				Username: u.username,
				Email:    u.username + "@example.com",
				Password: "UserPassword123!",
				Role:     u.role,
				Enabled:  true,
			})
			if err != nil {
				t.Fatalf("Failed to create user %s: %v", u.username, err)
			}
			t.Logf("Created user '%s' with role '%s'", u.username, u.role)
		}

		// Verify all users exist
		users, err := testStore.ListUsers(ctx)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(users) != 4 { // bootstrap_admin + 3 new users
			t.Errorf("Expected 4 users, got %d", len(users))
		}
	})

	t.Run("NewUsersCanAuthenticate", func(t *testing.T) {
		// Verify each new user can authenticate
		usersToCheck := []string{"developer_user", "readonly_user", "another_admin"}

		for _, username := range usersToCheck {
			user, err := authService.ValidateCredentials(ctx, username, "UserPassword123!")
			if err != nil {
				t.Errorf("User '%s' failed to authenticate: %v", username, err)
				continue
			}
			t.Logf("User '%s' authenticated successfully with role '%s'", user.Username, user.Role)
		}
	})

	t.Run("RoleBasedAccessWithBootstrappedUsers", func(t *testing.T) {
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

		// Create registry
		reg := registry.New(testStore, schemaRegistry, compatChecker, "BACKWARD")

		// Create server with auth enabled
		cfg := &config.Config{
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8083,
			},
			Security: config.SecurityConfig{
				Auth: config.AuthConfig{
					Enabled: true,
					Methods: []string{"basic"},
					Basic: config.BasicAuthConfig{
						Realm: "Test Realm",
					},
					RBAC: config.RBACConfig{
						Enabled:     true,
						DefaultRole: "readonly",
						SuperAdmins: []string{bootstrapUsername},
					},
				},
			},
		}

		authenticator := auth.NewAuthenticator(cfg.Security.Auth)
		authenticator.SetService(authService)
		authorizer := auth.NewAuthorizer(cfg.Security.Auth.RBAC)

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		server := api.NewServer(cfg, reg, logger, api.WithAuth(authenticator, authorizer, authService))
		ts := httptest.NewServer(server)
		defer ts.Close()

		// Helper to make requests
		makeRequest := func(method, path string, body interface{}, username, password string) *http.Response {
			var bodyReader io.Reader
			if body != nil {
				bodyBytes, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req, _ := http.NewRequest(method, ts.URL+path, bodyReader)
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			credentials := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			req.Header.Set("Authorization", "Basic "+credentials)

			resp, _ := http.DefaultClient.Do(req)
			return resp
		}

		testSchema := `{"type":"record","name":"BootstrapTest","fields":[{"name":"id","type":"int"}]}`

		// Super admin (bootstrap_admin) can do everything
		resp := makeRequest("POST", "/subjects/bootstrap-test/versions",
			map[string]string{"schema": testSchema},
			bootstrapUsername, updatedPassword)
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Super admin failed to create schema: %d - %s", resp.StatusCode, body)
		}
		resp.Body.Close()

		// Developer can create schemas
		resp = makeRequest("POST", "/subjects/bootstrap-test-dev/versions",
			map[string]string{"schema": testSchema},
			"developer_user", "UserPassword123!")
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Developer failed to create schema: %d - %s", resp.StatusCode, body)
		}
		resp.Body.Close()

		// Readonly user cannot create schemas
		resp = makeRequest("POST", "/subjects/bootstrap-test-readonly/versions",
			map[string]string{"schema": testSchema},
			"readonly_user", "UserPassword123!")
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected readonly user to be forbidden, got %d", resp.StatusCode)
		}
		resp.Body.Close()

		// Readonly user can read schemas
		resp = makeRequest("GET", "/subjects/bootstrap-test/versions/latest",
			nil, "readonly_user", "UserPassword123!")
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Readonly user failed to read schema: %d - %s", resp.StatusCode, body)
		}
		resp.Body.Close()

		t.Log("Role-based access control verified for bootstrapped users")
	})

	// Cleanup bootstrap test users
	t.Cleanup(func() {
		cleanupBootstrapUsers(t, ctx)
	})
}

// Helper functions

func mustHashPassword(t *testing.T, password string) string {
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	return hash
}

func cleanupTestUsers(t *testing.T, ctx context.Context) {
	users, err := testStore.ListUsers(ctx)
	if err != nil {
		return // Ignore errors during cleanup
	}

	testUsernames := []string{
		"testuser1", "testuser2", "testuser3",
		"authtest_user", "apikey_test_user",
		"rbac_admin_user", "rbac_developer_user", "rbac_readonly_user",
	}

	for _, user := range users {
		for _, testUsername := range testUsernames {
			if user.Username == testUsername {
				_ = testStore.DeleteUser(ctx, user.ID)
				break
			}
		}
	}
}

func cleanupAllUsers(t *testing.T, ctx context.Context) {
	users, err := testStore.ListUsers(ctx)
	if err != nil {
		return // Ignore errors during cleanup
	}

	for _, user := range users {
		_ = testStore.DeleteUser(ctx, user.ID)
	}
}

func cleanupBootstrapUsers(t *testing.T, ctx context.Context) {
	users, err := testStore.ListUsers(ctx)
	if err != nil {
		return // Ignore errors during cleanup
	}

	bootstrapUsernames := []string{
		"bootstrap_admin", "developer_user", "readonly_user", "another_admin",
	}

	for _, user := range users {
		for _, username := range bootstrapUsernames {
			if user.Username == username {
				_ = testStore.DeleteUser(ctx, user.ID)
				break
			}
		}
	}
}
