package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func TestAuthorizer_HasPermission(t *testing.T) {
	cfg := config.RBACConfig{
		Enabled:     true,
		DefaultRole: "readonly",
		SuperAdmins: []string{"superadmin"},
	}

	auth := NewAuthorizer(cfg)

	tests := []struct {
		name       string
		user       *User
		permission Permission
		expected   bool
	}{
		{
			name:       "super admin has all permissions",
			user:       &User{Username: "superadmin", Role: "readonly"},
			permission: PermissionAdminWrite,
			expected:   true,
		},
		{
			name:       "admin has schema delete",
			user:       &User{Username: "admin1", Role: "admin"},
			permission: PermissionSchemaDelete,
			expected:   true,
		},
		{
			name:       "developer has schema write",
			user:       &User{Username: "dev1", Role: "developer"},
			permission: PermissionSchemaWrite,
			expected:   true,
		},
		{
			name:       "developer cannot delete schema",
			user:       &User{Username: "dev1", Role: "developer"},
			permission: PermissionSchemaDelete,
			expected:   false,
		},
		{
			name:       "readonly can only read",
			user:       &User{Username: "reader1", Role: "readonly"},
			permission: PermissionSchemaRead,
			expected:   true,
		},
		{
			name:       "readonly cannot write",
			user:       &User{Username: "reader1", Role: "readonly"},
			permission: PermissionSchemaWrite,
			expected:   false,
		},
		{
			name:       "nil user has no permissions",
			user:       nil,
			permission: PermissionSchemaRead,
			expected:   false,
		},
		{
			name:       "unknown role uses default",
			user:       &User{Username: "unknown", Role: ""},
			permission: PermissionSchemaRead,
			expected:   true, // Default role is readonly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.HasPermission(tt.user, tt.permission)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAuthorizer_RequirePermission(t *testing.T) {
	cfg := config.RBACConfig{
		Enabled:     true,
		DefaultRole: "readonly",
	}

	auth := NewAuthorizer(cfg)

	t.Run("authorized user passes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)
		user := &User{Username: "reader", Role: "readonly"}
		req = req.WithContext(setUser(req.Context(), user))

		var called bool
		handler := auth.RequirePermission(PermissionSchemaRead)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !called {
			t.Error("Handler should have been called")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	t.Run("unauthorized user blocked", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/subjects/test/versions/1", nil)
		user := &User{Username: "reader", Role: "readonly"}
		req = req.WithContext(setUser(req.Context(), user))

		var called bool
		handler := auth.RequirePermission(PermissionSchemaDelete)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if called {
			t.Error("Handler should not have been called")
		}
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rr.Code)
		}
	})

	t.Run("no user returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subjects", nil)

		var called bool
		handler := auth.RequirePermission(PermissionSchemaRead)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if called {
			t.Error("Handler should not have been called")
		}
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("disabled RBAC passes all", func(t *testing.T) {
		disabledAuth := NewAuthorizer(config.RBACConfig{Enabled: false})

		req := httptest.NewRequest("DELETE", "/subjects/test", nil)
		// No user in context

		var called bool
		handler := disabledAuth.RequirePermission(PermissionSchemaDelete)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !called {
			t.Error("Handler should have been called when RBAC is disabled")
		}
	})
}

func TestAuthorizer_IsSuperAdmin(t *testing.T) {
	cfg := config.RBACConfig{
		SuperAdmins: []string{"admin1", "admin2"},
	}

	auth := NewAuthorizer(cfg)

	if !auth.IsSuperAdmin("admin1") {
		t.Error("admin1 should be super admin")
	}
	if !auth.IsSuperAdmin("admin2") {
		t.Error("admin2 should be super admin")
	}
	if auth.IsSuperAdmin("regular") {
		t.Error("regular should not be super admin")
	}
}

func TestGetRolePermissions(t *testing.T) {
	// Super admin should have all permissions
	superPerms := GetRolePermissions(RoleSuperAdmin)
	if len(superPerms) == 0 {
		t.Error("Super admin should have permissions")
	}

	// Readonly should only have read permissions
	readonlyPerms := GetRolePermissions(RoleReadOnly)
	for _, p := range readonlyPerms {
		switch p {
		case PermissionSchemaRead, PermissionConfigRead, PermissionModeRead:
			// OK
		default:
			t.Errorf("Readonly should not have permission %s", p)
		}
	}
}

func TestValidRole(t *testing.T) {
	if !ValidRole("super_admin") {
		t.Error("super_admin should be valid")
	}
	if !ValidRole("admin") {
		t.Error("admin should be valid")
	}
	if !ValidRole("developer") {
		t.Error("developer should be valid")
	}
	if !ValidRole("readonly") {
		t.Error("readonly should be valid")
	}
	if ValidRole("invalid") {
		t.Error("invalid should not be valid")
	}
}

func TestDefaultEndpointPermissions(t *testing.T) {
	perms := DefaultEndpointPermissions()
	if len(perms) == 0 {
		t.Error("Should have default endpoint permissions")
	}

	// Check for expected mappings
	hasSubjectsGet := false
	hasSubjectsPost := false
	for _, p := range perms {
		if p.Method == "GET" && p.PathPrefix == "/subjects" {
			hasSubjectsGet = true
			if p.Permission != PermissionSchemaRead {
				t.Error("GET /subjects should require schema:read")
			}
		}
		if p.Method == "POST" && p.PathPrefix == "/subjects" {
			hasSubjectsPost = true
			if p.Permission != PermissionSchemaWrite {
				t.Error("POST /subjects should require schema:write")
			}
		}
	}

	if !hasSubjectsGet {
		t.Error("Should have GET /subjects permission")
	}
	if !hasSubjectsPost {
		t.Error("Should have POST /subjects permission")
	}
}

func TestNormalizePathForRBAC(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Root-level paths pass through unchanged
		{"/subjects", "/subjects"},
		{"/subjects/my-topic/versions/1", "/subjects/my-topic/versions/1"},
		{"/schemas/ids/123", "/schemas/ids/123"},
		{"/config", "/config"},
		{"/mode/my-topic", "/mode/my-topic"},

		// Context-scoped paths have /contexts/{context} stripped
		{"/contexts/.TestContext/subjects", "/subjects"},
		{"/contexts/.TestContext/subjects/my-topic", "/subjects/my-topic"},
		{"/contexts/.TestContext/schemas/ids/123", "/schemas/ids/123"},
		{"/contexts/.TestContext/config", "/config"},
		{"/contexts/.TestContext/config/my-topic", "/config/my-topic"},
		{"/contexts/.TestContext/mode/my-topic", "/mode/my-topic"},
		{"/contexts/.TestContext/compatibility/subjects/my-topic/versions/1", "/compatibility/subjects/my-topic/versions/1"},
		{"/contexts/.TestContext/import/schemas", "/import/schemas"},
		{"/contexts/.production/subjects", "/subjects"},
		{"/contexts/:.:/subjects", "/subjects"},

		// Edge cases
		{"/contexts/.TestContext", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePathForRBAC(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePathForRBAC(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAuthorizeEndpoint_ContextScopedRoutes(t *testing.T) {
	cfg := config.RBACConfig{
		Enabled:     true,
		DefaultRole: "readonly",
	}
	auth := NewAuthorizer(cfg)
	permissions := DefaultEndpointPermissions()

	t.Run("readonly user can read context-scoped subjects", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/contexts/.TestContext/subjects", nil)
		user := &User{Username: "reader", Role: "readonly"}
		req = req.WithContext(setUser(req.Context(), user))

		var called bool
		handler := auth.AuthorizeEndpoint(permissions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !called {
			t.Error("Handler should have been called for readonly GET")
		}
	})

	t.Run("readonly user cannot write to context-scoped subjects", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/contexts/.TestContext/subjects/my-topic/versions", nil)
		user := &User{Username: "reader", Role: "readonly"}
		req = req.WithContext(setUser(req.Context(), user))

		var called bool
		handler := auth.AuthorizeEndpoint(permissions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if called {
			t.Error("Handler should not have been called for readonly POST")
		}
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", rr.Code)
		}
	})

	t.Run("no user on context-scoped route returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/contexts/.TestContext/subjects", nil)
		// No user in context

		var called bool
		handler := auth.AuthorizeEndpoint(permissions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if called {
			t.Error("Handler should not have been called without user")
		}
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})
}

// Helper to set user in context
func setUser(ctx context.Context, user *User) context.Context {
	ctx = context.WithValue(ctx, UserContextKey, user)
	ctx = context.WithValue(ctx, RoleContextKey, user.Role)
	return ctx
}
