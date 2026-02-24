// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"net/http"
	"strings"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// Role represents a user role.
type Role string

const (
	// RoleSuperAdmin has full access to everything.
	RoleSuperAdmin Role = "super_admin"
	// RoleAdmin can manage schemas and configuration.
	RoleAdmin Role = "admin"
	// RoleDeveloper can register and read schemas.
	RoleDeveloper Role = "developer"
	// RoleReadOnly can only read schemas.
	RoleReadOnly Role = "readonly"
)

// Permission represents an action on a resource.
type Permission string

const (
	// Schema permissions
	PermissionSchemaRead   Permission = "schema:read"
	PermissionSchemaWrite  Permission = "schema:write"
	PermissionSchemaDelete Permission = "schema:delete"

	// Config permissions
	PermissionConfigRead  Permission = "config:read"
	PermissionConfigWrite Permission = "config:write"

	// Mode permissions
	PermissionModeRead  Permission = "mode:read"
	PermissionModeWrite Permission = "mode:write"

	// Import permissions (for migration)
	PermissionImport Permission = "import:write"

	// Encryption permissions (KEK/DEK management)
	PermissionEncryptionRead  Permission = "encryption:read"
	PermissionEncryptionWrite Permission = "encryption:write"

	// Exporter permissions
	PermissionExporterRead  Permission = "exporter:read"
	PermissionExporterWrite Permission = "exporter:write"

	// Admin permissions
	PermissionAdminRead  Permission = "admin:read"
	PermissionAdminWrite Permission = "admin:write"
)

// rolePermissions defines permissions for each role.
var rolePermissions = map[Role][]Permission{
	RoleSuperAdmin: {
		PermissionSchemaRead, PermissionSchemaWrite, PermissionSchemaDelete,
		PermissionConfigRead, PermissionConfigWrite,
		PermissionModeRead, PermissionModeWrite,
		PermissionImport,
		PermissionAdminRead, PermissionAdminWrite,
		PermissionEncryptionRead, PermissionEncryptionWrite,
		PermissionExporterRead, PermissionExporterWrite,
	},
	RoleAdmin: {
		PermissionSchemaRead, PermissionSchemaWrite, PermissionSchemaDelete,
		PermissionConfigRead, PermissionConfigWrite,
		PermissionModeRead, PermissionModeWrite,
		PermissionImport,
		PermissionAdminRead,
		PermissionEncryptionRead, PermissionEncryptionWrite,
		PermissionExporterRead, PermissionExporterWrite,
	},
	RoleDeveloper: {
		PermissionSchemaRead, PermissionSchemaWrite,
		PermissionConfigRead,
		PermissionModeRead,
		PermissionEncryptionRead,
		PermissionExporterRead,
	},
	RoleReadOnly: {
		PermissionSchemaRead,
		PermissionConfigRead,
		PermissionModeRead,
		PermissionEncryptionRead,
		PermissionExporterRead,
	},
}

// Authorizer handles authorization.
type Authorizer struct {
	config      config.RBACConfig
	superAdmins map[string]bool
}

// NewAuthorizer creates a new authorizer.
func NewAuthorizer(cfg config.RBACConfig) *Authorizer {
	superAdmins := make(map[string]bool)
	for _, admin := range cfg.SuperAdmins {
		superAdmins[admin] = true
	}

	return &Authorizer{
		config:      cfg,
		superAdmins: superAdmins,
	}
}

// HasPermission checks if a user has a specific permission.
func (a *Authorizer) HasPermission(user *User, perm Permission) bool {
	if user == nil {
		return false
	}

	// Super admins have all permissions
	if a.superAdmins[user.Username] {
		return true
	}

	role := Role(user.Role)
	if role == "" {
		role = Role(a.config.DefaultRole)
	}

	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}

	for _, p := range perms {
		if p == perm {
			return true
		}
	}

	return false
}

// RequirePermission returns middleware that requires a specific permission.
func (a *Authorizer) RequirePermission(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !a.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			user := GetUser(r.Context())
			if user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !a.HasPermission(user, perm) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// EndpointPermission maps HTTP methods and paths to required permissions.
type EndpointPermission struct {
	Method     string
	PathPrefix string
	Permission Permission
}

// DefaultEndpointPermissions returns the default endpoint permission mappings.
func DefaultEndpointPermissions() []EndpointPermission {
	return []EndpointPermission{
		// Schema read operations
		{Method: "GET", PathPrefix: "/subjects", Permission: PermissionSchemaRead},
		{Method: "GET", PathPrefix: "/schemas", Permission: PermissionSchemaRead},

		// Schema write operations
		{Method: "POST", PathPrefix: "/subjects", Permission: PermissionSchemaWrite},
		{Method: "POST", PathPrefix: "/compatibility", Permission: PermissionSchemaRead},

		// Schema delete operations
		{Method: "DELETE", PathPrefix: "/subjects", Permission: PermissionSchemaDelete},

		// Config operations
		{Method: "GET", PathPrefix: "/config", Permission: PermissionConfigRead},
		{Method: "PUT", PathPrefix: "/config", Permission: PermissionConfigWrite},
		{Method: "DELETE", PathPrefix: "/config", Permission: PermissionConfigWrite},

		// Mode operations
		{Method: "GET", PathPrefix: "/mode", Permission: PermissionModeRead},
		{Method: "PUT", PathPrefix: "/mode", Permission: PermissionModeWrite},

		// Import operations (migration)
		{Method: "POST", PathPrefix: "/import", Permission: PermissionImport},

		// DEK Registry (encryption key management)
		{Method: "GET", PathPrefix: "/dek-registry", Permission: PermissionEncryptionRead},
		{Method: "POST", PathPrefix: "/dek-registry", Permission: PermissionEncryptionWrite},
		{Method: "PUT", PathPrefix: "/dek-registry", Permission: PermissionEncryptionWrite},
		{Method: "DELETE", PathPrefix: "/dek-registry", Permission: PermissionEncryptionWrite},

		// Exporter operations
		{Method: "GET", PathPrefix: "/exporters", Permission: PermissionExporterRead},
		{Method: "POST", PathPrefix: "/exporters", Permission: PermissionExporterWrite},
		{Method: "PUT", PathPrefix: "/exporters", Permission: PermissionExporterWrite},
		{Method: "DELETE", PathPrefix: "/exporters", Permission: PermissionExporterWrite},

		// Mode delete operations
		{Method: "DELETE", PathPrefix: "/mode", Permission: PermissionModeWrite},

		// Admin operations (user management, API keys, roles)
		{Method: "GET", PathPrefix: "/admin", Permission: PermissionAdminRead},
		{Method: "POST", PathPrefix: "/admin", Permission: PermissionAdminWrite},
		{Method: "PUT", PathPrefix: "/admin", Permission: PermissionAdminWrite},
		{Method: "DELETE", PathPrefix: "/admin", Permission: PermissionAdminWrite},

		// Self-service account operations (any authenticated user)
		{Method: "GET", PathPrefix: "/me", Permission: PermissionSchemaRead},
		{Method: "POST", PathPrefix: "/me", Permission: PermissionSchemaRead},

		// Contexts and metadata (read-only, any authenticated user)
		{Method: "GET", PathPrefix: "/contexts", Permission: PermissionSchemaRead},
		{Method: "GET", PathPrefix: "/v1/metadata", Permission: PermissionSchemaRead},
	}
}

// normalizePathForRBAC strips the /contexts/{context} prefix from a URL path
// so that context-scoped routes match the same RBAC permissions as root routes.
// For example, /contexts/.TestContext/subjects/foo → /subjects/foo.
func normalizePathForRBAC(path string) string {
	const prefix = "/contexts/"
	if strings.HasPrefix(path, prefix) {
		// Find the end of the context name (next slash after /contexts/)
		rest := path[len(prefix):]
		idx := strings.Index(rest, "/")
		if idx >= 0 {
			return rest[idx:] // Return everything after /contexts/{context}
		}
		return "/" // Context path with no sub-path
	}
	return path
}

// AuthorizeEndpoint returns middleware that checks endpoint-based permissions.
func (a *Authorizer) AuthorizeEndpoint(permissions []EndpointPermission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !a.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			user := GetUser(r.Context())

			// Normalize path to strip /contexts/{context} prefix so that
			// context-scoped routes match the same permissions as root routes.
			normalizedPath := normalizePathForRBAC(r.URL.Path)

			// Find matching permission
			matched := false
			for _, ep := range permissions {
				if r.Method == ep.Method && strings.HasPrefix(normalizedPath, ep.PathPrefix) {
					matched = true
					if user == nil {
						http.Error(w, "Unauthorized", http.StatusUnauthorized)
						return
					}

					if !a.HasPermission(user, ep.Permission) {
						http.Error(w, "Forbidden", http.StatusForbidden)
						return
					}
					break
				}
			}

			// Deny-by-default: if RBAC is enabled and a user is authenticated
			// but no permission entry matched, block access. This prevents
			// newly added routes from being accidentally unprotected.
			if !matched && user != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IsSuperAdmin checks if a user is a super admin.
func (a *Authorizer) IsSuperAdmin(username string) bool {
	return a.superAdmins[username]
}

// GetRolePermissions returns the permissions for a role.
func GetRolePermissions(role Role) []Permission {
	return rolePermissions[role]
}

// ValidRole checks if a role is valid.
func ValidRole(role string) bool {
	switch Role(role) {
	case RoleSuperAdmin, RoleAdmin, RoleDeveloper, RoleReadOnly:
		return true
	default:
		return false
	}
}
