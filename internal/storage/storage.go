// Package storage provides storage interfaces and implementations for the schema registry.
package storage

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrNotFound         = errors.New("not found")
	ErrSubjectNotFound  = errors.New("subject not found")
	ErrSchemaNotFound   = errors.New("schema not found")
	ErrVersionNotFound  = errors.New("version not found")
	ErrInvalidVersion   = errors.New("invalid version")
	ErrSubjectDeleted        = errors.New("subject has been deleted")
	ErrSubjectNotSoftDeleted = errors.New("subject must be soft-deleted before being permanently deleted")
	ErrVersionNotSoftDeleted = errors.New("version must be soft-deleted before being permanently deleted")
	ErrSchemaExists          = errors.New("schema already exists")
	ErrUserNotFound     = errors.New("user not found")
	ErrUserExists       = errors.New("user already exists")
	ErrAPIKeyNotFound   = errors.New("API key not found")
	ErrAPIKeyExists     = errors.New("API key already exists")
	ErrAPIKeyNameExists = errors.New("API key name already exists for this user")
	ErrInvalidAPIKey    = errors.New("invalid API key")
	ErrAPIKeyExpired    = errors.New("API key has expired")
	ErrAPIKeyDisabled   = errors.New("API key is disabled")
	ErrUserDisabled     = errors.New("user is disabled")
	ErrInvalidRole      = errors.New("invalid role")
	ErrPermissionDenied = errors.New("permission denied")
	ErrSchemaIDConflict      = errors.New("schema ID already exists")
	ErrOperationNotPermitted = errors.New("Cannot import since found existing subjects")
)

// SchemaType represents the type of schema.
type SchemaType string

const (
	SchemaTypeAvro     SchemaType = "AVRO"
	SchemaTypeProtobuf SchemaType = "PROTOBUF"
	SchemaTypeJSON     SchemaType = "JSON"
)

// SchemaRecord represents a stored schema.
type SchemaRecord struct {
	ID          int64       `json:"id"`
	Subject     string      `json:"subject"`
	Version     int         `json:"version"`
	SchemaType  SchemaType  `json:"schemaType"`
	Schema      string      `json:"schema"`
	References  []Reference `json:"references,omitempty"`
	Fingerprint string      `json:"-"`
	Deleted     bool        `json:"-"`
	CreatedAt   time.Time   `json:"-"`
}

// Reference represents a schema reference.
type Reference struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Version int    `json:"version"`
	Schema  string `json:"-"` // Resolved schema content; not serialized to API responses
}

// SubjectVersion represents a subject-version pair.
type SubjectVersion struct {
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

// ConfigRecord represents a compatibility configuration.
type ConfigRecord struct {
	Subject            string `json:"subject,omitempty"` // Empty for global config
	CompatibilityLevel string `json:"compatibilityLevel"`
	Normalize          *bool  `json:"normalize,omitempty"`
}

// ModeRecord represents a mode configuration.
type ModeRecord struct {
	Subject string `json:"subject,omitempty"` // Empty for global mode
	Mode    string `json:"mode"`
}

// UserRecord represents a stored user.
type UserRecord struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"` // Never exposed in JSON
	Role         string    `json:"role"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// APIKeyRecord represents a stored API key.
type APIKeyRecord struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`    // User who owns this API key (required)
	KeyHash   string     `json:"-"`          // SHA-256 hash of the key, never exposed
	KeyPrefix string     `json:"key_prefix"` // First 8 chars for display/identification
	Name      string     `json:"name"`       // Unique per user
	Role      string     `json:"role"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"` // Required expiration time
	LastUsed  *time.Time `json:"last_used,omitempty"`
}

// AuthStorage defines the interface for authentication storage backends.
// This can be implemented by database backends or secrets managers like Vault.
type AuthStorage interface {
	// User management
	CreateUser(ctx context.Context, user *UserRecord) error
	GetUserByID(ctx context.Context, id int64) (*UserRecord, error)
	GetUserByUsername(ctx context.Context, username string) (*UserRecord, error)
	UpdateUser(ctx context.Context, user *UserRecord) error
	DeleteUser(ctx context.Context, id int64) error
	ListUsers(ctx context.Context) ([]*UserRecord, error)

	// API Key management
	CreateAPIKey(ctx context.Context, key *APIKeyRecord) error
	GetAPIKeyByID(ctx context.Context, id int64) (*APIKeyRecord, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKeyRecord, error)
	GetAPIKeyByUserAndName(ctx context.Context, userID int64, name string) (*APIKeyRecord, error)
	UpdateAPIKey(ctx context.Context, key *APIKeyRecord) error
	DeleteAPIKey(ctx context.Context, id int64) error
	ListAPIKeys(ctx context.Context) ([]*APIKeyRecord, error)
	ListAPIKeysByUserID(ctx context.Context, userID int64) ([]*APIKeyRecord, error)
	UpdateAPIKeyLastUsed(ctx context.Context, id int64) error
}

// Storage defines the interface for schema storage backends.
// It embeds AuthStorage so database backends can implement both.
type Storage interface {
	AuthStorage

	// Schema operations
	CreateSchema(ctx context.Context, record *SchemaRecord) error
	GetSchemaByID(ctx context.Context, id int64) (*SchemaRecord, error)
	GetSchemaBySubjectVersion(ctx context.Context, subject string, version int) (*SchemaRecord, error)
	GetSchemasBySubject(ctx context.Context, subject string, includeDeleted bool) ([]*SchemaRecord, error)
	GetSchemaByFingerprint(ctx context.Context, subject, fingerprint string, includeDeleted bool) (*SchemaRecord, error)
	GetSchemaByGlobalFingerprint(ctx context.Context, fingerprint string) (*SchemaRecord, error)
	GetLatestSchema(ctx context.Context, subject string) (*SchemaRecord, error)
	DeleteSchema(ctx context.Context, subject string, version int, permanent bool) error

	// Subject operations
	ListSubjects(ctx context.Context, includeDeleted bool) ([]string, error)
	DeleteSubject(ctx context.Context, subject string, permanent bool) ([]int, error)
	SubjectExists(ctx context.Context, subject string) (bool, error)

	// Config operations
	GetConfig(ctx context.Context, subject string) (*ConfigRecord, error)
	SetConfig(ctx context.Context, subject string, config *ConfigRecord) error
	DeleteConfig(ctx context.Context, subject string) error
	GetGlobalConfig(ctx context.Context) (*ConfigRecord, error)
	SetGlobalConfig(ctx context.Context, config *ConfigRecord) error

	// Mode operations
	GetMode(ctx context.Context, subject string) (*ModeRecord, error)
	SetMode(ctx context.Context, subject string, mode *ModeRecord) error
	DeleteMode(ctx context.Context, subject string) error
	GetGlobalMode(ctx context.Context) (*ModeRecord, error)
	SetGlobalMode(ctx context.Context, mode *ModeRecord) error

	// ID generation
	NextID(ctx context.Context) (int64, error)
	GetMaxSchemaID(ctx context.Context) (int64, error)

	// Import operations (for migration from other schema registries)
	// ImportSchema inserts a schema with a specified ID (for migration).
	// Returns ErrSchemaIDConflict if the ID already exists.
	ImportSchema(ctx context.Context, record *SchemaRecord) error
	// SetNextID sets the ID sequence to start from the given value.
	// Used after import to prevent ID conflicts.
	SetNextID(ctx context.Context, id int64) error

	// References
	GetReferencedBy(ctx context.Context, subject string, version int) ([]SubjectVersion, error)

	// Schema ID lookups
	GetSubjectsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]string, error)
	GetVersionsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]SubjectVersion, error)

	// Schema listing
	ListSchemas(ctx context.Context, params *ListSchemasParams) ([]*SchemaRecord, error)

	// Global config delete
	DeleteGlobalConfig(ctx context.Context) error

	// Lifecycle
	Close() error
	IsHealthy(ctx context.Context) bool
}

// ListSchemasParams contains parameters for listing schemas.
type ListSchemasParams struct {
	SubjectPrefix string
	Deleted       bool
	LatestOnly    bool
	Offset        int
	Limit         int
}
