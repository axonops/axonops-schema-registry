// Package types provides API request and response types.
package types

import "github.com/axonops/axonops-schema-registry/internal/storage"

// RegisterSchemaRequest is the request body for registering a schema.
type RegisterSchemaRequest struct {
	Schema     string              `json:"schema"`
	SchemaType string              `json:"schemaType,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
}

// RegisterSchemaResponse is the response for registering a schema.
type RegisterSchemaResponse struct {
	ID int64 `json:"id"`
}

// SchemaResponse is the response for getting a schema.
type SchemaResponse struct {
	Schema     string              `json:"schema"`
	SchemaType string              `json:"schemaType,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
}

// SchemaByIDResponse is the response for getting a schema by ID.
type SchemaByIDResponse struct {
	Schema     string              `json:"schema"`
	SchemaType string              `json:"schemaType,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
	MaxId      *int64              `json:"maxId,omitempty"`
}

// SubjectVersionResponse is the response for getting a subject version.
type SubjectVersionResponse struct {
	Subject    string              `json:"subject"`
	ID         int64               `json:"id"`
	Version    int                 `json:"version"`
	SchemaType string              `json:"schemaType,omitempty"`
	Schema     string              `json:"schema"`
	References []storage.Reference `json:"references,omitempty"`
}

// LookupSchemaRequest is the request body for looking up a schema.
type LookupSchemaRequest struct {
	Schema     string              `json:"schema"`
	SchemaType string              `json:"schemaType,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
}

// LookupSchemaResponse is the response for looking up a schema.
type LookupSchemaResponse struct {
	Subject    string              `json:"subject"`
	ID         int64               `json:"id"`
	Version    int                 `json:"version"`
	SchemaType string              `json:"schemaType,omitempty"`
	Schema     string              `json:"schema"`
	References []storage.Reference `json:"references,omitempty"`
}

// ConfigResponse is the response for getting configuration.
type ConfigResponse struct {
	CompatibilityLevel string `json:"compatibilityLevel"`
	Normalize          *bool  `json:"normalize,omitempty"`
}

// ConfigRequest is the request body for setting configuration.
type ConfigRequest struct {
	Compatibility string `json:"compatibility"`
	Normalize     *bool  `json:"normalize,omitempty"`
}

// ModeResponse is the response for getting mode.
type ModeResponse struct {
	Mode string `json:"mode"`
}

// ModeRequest is the request body for setting mode.
type ModeRequest struct {
	Mode string `json:"mode"`
}

// CompatibilityCheckRequest is the request for checking compatibility.
type CompatibilityCheckRequest struct {
	Schema     string              `json:"schema"`
	SchemaType string              `json:"schemaType,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
}

// CompatibilityCheckResponse is the response for checking compatibility.
type CompatibilityCheckResponse struct {
	IsCompatible bool     `json:"is_compatible"`
	Messages     []string `json:"messages,omitempty"`
}

// ErrorResponse is the error response format.
type ErrorResponse struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}

// SubjectVersionPair is a subject-version tuple returned by various endpoints.
type SubjectVersionPair struct {
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

// SchemaListItem is a schema in the list response.
type SchemaListItem struct {
	Subject    string              `json:"subject"`
	Version    int                 `json:"version"`
	ID         int64               `json:"id"`
	SchemaType string              `json:"schemaType,omitempty"`
	Schema     string              `json:"schema"`
	References []storage.Reference `json:"references,omitempty"`
}

// ServerClusterIDResponse is the response for getting cluster ID.
type ServerClusterIDResponse struct {
	ID string `json:"id"`
}

// ServerVersionResponse is the response for getting server version.
type ServerVersionResponse struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
}

// Error codes matching Confluent Schema Registry
const (
	ErrorCodeSubjectNotFound           = 40401
	ErrorCodeVersionNotFound           = 40402
	ErrorCodeSchemaNotFound            = 40403
	ErrorCodeSubjectSoftDeleted        = 40404
	ErrorCodeSubjectNotSoftDeleted     = 40405
	ErrorCodeSchemaVersionSoftDeleted  = 40406
	ErrorCodeIncompatibleSchema        = 409
	ErrorCodeInvalidSchema             = 42201
	ErrorCodeInvalidSchemaType         = 42202
	ErrorCodeInvalidVersion            = 42202 // Confluent uses 42202 for both invalid schema type and invalid version
	ErrorCodeInvalidCompatibilityLevel = 42203
	ErrorCodeInvalidMode               = 42204
	ErrorCodeOperationNotPermitted     = 42205
	ErrorCodeReferenceExists           = 42206
	ErrorCodeInternalServerError       = 50001
	ErrorCodeStorageError              = 50002

	// Admin error codes
	ErrorCodeUnauthorized    = 40101
	ErrorCodeForbidden       = 40301
	ErrorCodeUserNotFound    = 40404
	ErrorCodeUserExists      = 40901
	ErrorCodeAPIKeyNotFound  = 40405
	ErrorCodeAPIKeyExists    = 40902
	ErrorCodeInvalidRole     = 42207
	ErrorCodeInvalidPassword = 42208
	ErrorCodeAPIKeyExpired   = 40103
	ErrorCodeAPIKeyDisabled  = 40104
	ErrorCodeUserDisabled    = 40105
)

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Enabled  *bool  `json:"enabled,omitempty"`
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
}

// UserResponse is the response for user operations.
type UserResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	Role      string `json:"role"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// UsersListResponse is the response for listing users.
type UsersListResponse struct {
	Users []UserResponse `json:"users"`
}

// ChangePasswordRequest is the request body for changing password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// CreateAPIKeyRequest is the request body for creating an API key.
type CreateAPIKeyRequest struct {
	Name      string `json:"name"`                  // Required, must be unique per user
	Role      string `json:"role"`                  // Required: super_admin, admin, developer, readonly
	ExpiresIn int64  `json:"expires_in"`            // Required, duration in seconds (e.g., 2592000 for 30 days)
	ForUserID *int64 `json:"for_user_id,omitempty"` // Optional: super_admin can create keys for other users
}

// UpdateAPIKeyRequest is the request body for updating an API key.
type UpdateAPIKeyRequest struct {
	Name    *string `json:"name,omitempty"`
	Role    *string `json:"role,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

// APIKeyResponse is the response for API key operations (without the raw key).
type APIKeyResponse struct {
	ID        int64   `json:"id"`
	KeyPrefix string  `json:"key_prefix"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	UserID    int64   `json:"user_id"`  // User who owns this API key
	Username  string  `json:"username"` // Username of the owner
	Enabled   bool    `json:"enabled"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt string  `json:"expires_at"`
	LastUsed  *string `json:"last_used,omitempty"`
}

// CreateAPIKeyResponse is the response for creating an API key (includes raw key).
type CreateAPIKeyResponse struct {
	ID        int64  `json:"id"`
	Key       string `json:"key"` // Raw key, only shown once
	KeyPrefix string `json:"key_prefix"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	UserID    int64  `json:"user_id"`  // User who owns this API key
	Username  string `json:"username"` // Username of the owner
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

// APIKeysListResponse is the response for listing API keys.
type APIKeysListResponse struct {
	APIKeys []APIKeyResponse `json:"api_keys"`
}

// RotateAPIKeyResponse is the response for rotating an API key.
type RotateAPIKeyResponse struct {
	NewKey    CreateAPIKeyResponse `json:"new_key"`
	RevokedID int64                `json:"revoked_id"`
}

// RolesListResponse is the response for listing available roles.
type RolesListResponse struct {
	Roles []RoleInfo `json:"roles"`
}

// RoleInfo describes a role and its permissions.
type RoleInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// ImportSchemaRequest is the request for importing a single schema with a specific ID.
type ImportSchemaRequest struct {
	ID         int64               `json:"id"`
	Subject    string              `json:"subject"`
	Version    int                 `json:"version"`
	SchemaType string              `json:"schemaType,omitempty"`
	Schema     string              `json:"schema"`
	References []storage.Reference `json:"references,omitempty"`
}

// ImportSchemasRequest is the request for importing multiple schemas.
type ImportSchemasRequest struct {
	Schemas []ImportSchemaRequest `json:"schemas"`
}

// ImportSchemaResult is the result for a single schema import.
type ImportSchemaResult struct {
	ID      int64  `json:"id"`
	Subject string `json:"subject"`
	Version int    `json:"version"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ImportSchemasResponse is the response for importing schemas.
type ImportSchemasResponse struct {
	Imported int                  `json:"imported"`
	Errors   int                  `json:"errors"`
	Results  []ImportSchemaResult `json:"results"`
}
