// Package storage provides storage interfaces and implementations for the schema registry.
package storage

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrNotFound              = errors.New("not found")
	ErrSubjectNotFound       = errors.New("subject not found")
	ErrSchemaNotFound        = errors.New("schema not found")
	ErrVersionNotFound       = errors.New("version not found")
	ErrInvalidVersion        = errors.New("invalid version")
	ErrSubjectDeleted        = errors.New("subject has been deleted")
	ErrSubjectNotSoftDeleted = errors.New("subject must be soft-deleted before being permanently deleted")
	ErrVersionNotSoftDeleted = errors.New("version must be soft-deleted before being permanently deleted")
	ErrSchemaExists          = errors.New("schema already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrUserExists            = errors.New("user already exists")
	ErrAPIKeyNotFound        = errors.New("API key not found")
	ErrAPIKeyExists          = errors.New("API key already exists")
	ErrAPIKeyNameExists      = errors.New("API key name already exists for this user")
	ErrInvalidAPIKey         = errors.New("invalid API key")
	ErrAPIKeyExpired         = errors.New("API key has expired")
	ErrAPIKeyDisabled        = errors.New("API key is disabled")
	ErrUserDisabled          = errors.New("user is disabled")
	ErrInvalidRole           = errors.New("invalid role")
	ErrPermissionDenied      = errors.New("permission denied")
	ErrSchemaIDConflict      = errors.New("schema ID already exists")
	ErrOperationNotPermitted = errors.New("Cannot import since found existing subjects")
	ErrExporterNotFound      = errors.New("exporter not found")
	ErrExporterExists        = errors.New("exporter already exists")
	ErrKEKNotFound           = errors.New("key encryption key not found")
	ErrKEKExists             = errors.New("key encryption key already exists")
	ErrKEKSoftDeleted        = errors.New("key encryption key is soft-deleted")
	ErrDEKNotFound           = errors.New("data encryption key not found")
	ErrDEKExists             = errors.New("data encryption key already exists")
	ErrDEKSoftDeleted        = errors.New("data encryption key is soft-deleted")
)

// SchemaType represents the type of schema.
type SchemaType string

const (
	SchemaTypeAvro     SchemaType = "AVRO"
	SchemaTypeProtobuf SchemaType = "PROTOBUF"
	SchemaTypeJSON     SchemaType = "JSON"
)

// Metadata represents schema metadata for data contracts.
type Metadata struct {
	Tags       map[string][]string `json:"tags,omitempty"`
	Properties map[string]string   `json:"properties,omitempty"`
	Sensitive  []string            `json:"sensitive,omitempty"`
}

// RuleSet represents a set of data contract rules.
type RuleSet struct {
	MigrationRules []Rule `json:"migrationRules,omitempty"`
	DomainRules    []Rule `json:"domainRules,omitempty"`
	EncodingRules  []Rule `json:"encodingRules,omitempty"`
}

// Rule represents a single data contract rule.
type Rule struct {
	Name      string            `json:"name"`
	Doc       string            `json:"doc,omitempty"`
	Kind      string            `json:"kind"`
	Mode      string            `json:"mode"`
	Type      string            `json:"type,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Params    map[string]string `json:"params,omitempty"`
	Expr      string            `json:"expr,omitempty"`
	OnSuccess string            `json:"onSuccess,omitempty"`
	OnFailure string            `json:"onFailure,omitempty"`
	Disabled  bool              `json:"disabled,omitempty"`
}

// SchemaRecord represents a stored schema.
type SchemaRecord struct {
	ID          int64       `json:"id"`
	Subject     string      `json:"subject"`
	Version     int         `json:"version"`
	SchemaType  SchemaType  `json:"schemaType"`
	Schema      string      `json:"schema"`
	References  []Reference `json:"references,omitempty"`
	Metadata    *Metadata   `json:"metadata,omitempty"`
	RuleSet     *RuleSet    `json:"ruleSet,omitempty"`
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
	Subject            string    `json:"subject,omitempty"` // Empty for global config
	CompatibilityLevel string    `json:"compatibilityLevel"`
	Normalize          *bool     `json:"normalize,omitempty"`
	ValidateFields     *bool     `json:"validateFields,omitempty"`
	Alias              string    `json:"alias,omitempty"`
	CompatibilityGroup string    `json:"compatibilityGroup,omitempty"`
	DefaultMetadata    *Metadata `json:"defaultMetadata,omitempty"`
	OverrideMetadata   *Metadata `json:"overrideMetadata,omitempty"`
	DefaultRuleSet     *RuleSet  `json:"defaultRuleSet,omitempty"`
	OverrideRuleSet    *RuleSet  `json:"overrideRuleSet,omitempty"`
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

// ExporterRecord represents a stored exporter (Confluent Schema Linking compatible).
type ExporterRecord struct {
	Name                string            `json:"name"`
	ContextType         string            `json:"contextType,omitempty"`         // CUSTOM, NONE, AUTO (default: AUTO)
	Context             string            `json:"context,omitempty"`             // Custom context path (when ContextType is CUSTOM)
	Subjects            []string          `json:"subjects,omitempty"`            // Subject filter list
	SubjectRenameFormat string            `json:"subjectRenameFormat,omitempty"` // Subject rename format (e.g. "${subject}")
	Config              map[string]string `json:"config,omitempty"`              // Destination configuration
	CreatedAt           time.Time         `json:"-"`
	UpdatedAt           time.Time         `json:"-"`
}

// ExporterStatusRecord represents the status of an exporter.
type ExporterStatusRecord struct {
	Name   string `json:"name"`
	State  string `json:"state"`            // STARTING, RUNNING, PAUSED, ERROR
	Offset int64  `json:"offset,omitempty"` // Last exported offset
	Ts     int64  `json:"ts,omitempty"`     // Timestamp of last state change
	Trace  string `json:"trace,omitempty"`  // Error trace if state is ERROR
}

// KEKRecord represents a Key Encryption Key for CSFLE (Client-Side Field Level Encryption).
type KEKRecord struct {
	Name      string            `json:"name"`
	KmsType   string            `json:"kmsType"`              // aws-kms, azure-kms, gcp-kms, hcvault
	KmsKeyID  string            `json:"kmsKeyId"`             // KMS key identifier
	KmsProps  map[string]string `json:"kmsProps,omitempty"`   // KMS-specific properties
	Doc       string            `json:"doc,omitempty"`        // Documentation string
	Shared    bool              `json:"shared"`               // Whether DEKs under this KEK share key material
	Deleted   bool              `json:"deleted,omitempty"`    // Soft-delete flag
	Ts        int64             `json:"ts,omitempty"`         // Timestamp of last modification
	CreatedAt time.Time         `json:"-"`
	UpdatedAt time.Time         `json:"-"`
}

// DEKRecord represents a Data Encryption Key used for field-level encryption.
type DEKRecord struct {
	KEKName              string `json:"kekName"`
	Subject              string `json:"subject"`
	Version              int    `json:"version"`
	Algorithm            string `json:"algorithm"`                      // AES128_GCM, AES256_GCM, AES256_SIV
	EncryptedKeyMaterial string `json:"encryptedKeyMaterial,omitempty"` // Encrypted DEK material
	KeyMaterial          string `json:"keyMaterial,omitempty"`          // Plaintext DEK (only returned on create if shared=true; never stored)
	Deleted              bool   `json:"deleted,omitempty"`              // Soft-delete flag
	Ts                   int64  `json:"ts,omitempty"`                   // Timestamp of last modification
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
//
// All schema, subject, config, mode, and ID methods accept a registryCtx parameter
// that identifies the schema registry context (namespace). The default context is ".".
// Schema IDs are scoped per-context: the same ID in different contexts represents
// different schemas (Confluent-compatible multi-tenancy).
type Storage interface {
	AuthStorage

	// Schema operations
	CreateSchema(ctx context.Context, registryCtx string, record *SchemaRecord) error
	GetSchemaByID(ctx context.Context, registryCtx string, id int64) (*SchemaRecord, error)
	GetSchemaBySubjectVersion(ctx context.Context, registryCtx string, subject string, version int) (*SchemaRecord, error)
	GetSchemasBySubject(ctx context.Context, registryCtx string, subject string, includeDeleted bool) ([]*SchemaRecord, error)
	GetSchemaByFingerprint(ctx context.Context, registryCtx string, subject, fingerprint string, includeDeleted bool) (*SchemaRecord, error)
	GetSchemaByGlobalFingerprint(ctx context.Context, registryCtx string, fingerprint string) (*SchemaRecord, error)
	GetLatestSchema(ctx context.Context, registryCtx string, subject string) (*SchemaRecord, error)
	DeleteSchema(ctx context.Context, registryCtx string, subject string, version int, permanent bool) error

	// Subject operations
	ListSubjects(ctx context.Context, registryCtx string, includeDeleted bool) ([]string, error)
	DeleteSubject(ctx context.Context, registryCtx string, subject string, permanent bool) ([]int, error)
	SubjectExists(ctx context.Context, registryCtx string, subject string) (bool, error)

	// Config operations (per-context: "global" means all subjects within the context)
	GetConfig(ctx context.Context, registryCtx string, subject string) (*ConfigRecord, error)
	SetConfig(ctx context.Context, registryCtx string, subject string, config *ConfigRecord) error
	DeleteConfig(ctx context.Context, registryCtx string, subject string) error
	GetGlobalConfig(ctx context.Context, registryCtx string) (*ConfigRecord, error)
	SetGlobalConfig(ctx context.Context, registryCtx string, config *ConfigRecord) error

	// Mode operations (per-context: "global" means all subjects within the context)
	GetMode(ctx context.Context, registryCtx string, subject string) (*ModeRecord, error)
	SetMode(ctx context.Context, registryCtx string, subject string, mode *ModeRecord) error
	DeleteMode(ctx context.Context, registryCtx string, subject string) error
	GetGlobalMode(ctx context.Context, registryCtx string) (*ModeRecord, error)
	SetGlobalMode(ctx context.Context, registryCtx string, mode *ModeRecord) error
	DeleteGlobalMode(ctx context.Context, registryCtx string) error

	// ID generation (per-context: each context has its own ID sequence)
	NextID(ctx context.Context, registryCtx string) (int64, error)
	GetMaxSchemaID(ctx context.Context, registryCtx string) (int64, error)

	// Import operations (for migration from other schema registries)
	// ImportSchema inserts a schema with a specified ID (for migration).
	// Returns ErrSchemaIDConflict if the ID already exists.
	ImportSchema(ctx context.Context, registryCtx string, record *SchemaRecord) error
	// SetNextID sets the ID sequence to start from the given value.
	// Used after import to prevent ID conflicts.
	SetNextID(ctx context.Context, registryCtx string, id int64) error

	// References
	GetReferencedBy(ctx context.Context, registryCtx string, subject string, version int) ([]SubjectVersion, error)

	// Schema ID lookups
	GetSubjectsBySchemaID(ctx context.Context, registryCtx string, id int64, includeDeleted bool) ([]string, error)
	GetVersionsBySchemaID(ctx context.Context, registryCtx string, id int64, includeDeleted bool) ([]SubjectVersion, error)

	// Schema listing
	ListSchemas(ctx context.Context, registryCtx string, params *ListSchemasParams) ([]*SchemaRecord, error)

	// Context operations
	ListContexts(ctx context.Context) ([]string, error)

	// Global config delete
	DeleteGlobalConfig(ctx context.Context, registryCtx string) error

	// KEK operations (CSFLE - Client-Side Field Level Encryption)
	CreateKEK(ctx context.Context, kek *KEKRecord) error
	GetKEK(ctx context.Context, name string, includeDeleted bool) (*KEKRecord, error)
	UpdateKEK(ctx context.Context, kek *KEKRecord) error
	DeleteKEK(ctx context.Context, name string, permanent bool) error
	UndeleteKEK(ctx context.Context, name string) error
	ListKEKs(ctx context.Context, includeDeleted bool) ([]*KEKRecord, error)

	// DEK operations (CSFLE - Client-Side Field Level Encryption)
	CreateDEK(ctx context.Context, dek *DEKRecord) error
	GetDEK(ctx context.Context, kekName, subject string, version int, algorithm string, includeDeleted bool) (*DEKRecord, error)
	ListDEKs(ctx context.Context, kekName string, includeDeleted bool) ([]string, error)
	ListDEKVersions(ctx context.Context, kekName, subject string, algorithm string, includeDeleted bool) ([]int, error)
	DeleteDEK(ctx context.Context, kekName, subject string, version int, algorithm string, permanent bool) error
	UndeleteDEK(ctx context.Context, kekName, subject string, version int, algorithm string) error

	// Exporter operations (Confluent Schema Linking compatible)
	CreateExporter(ctx context.Context, exporter *ExporterRecord) error
	GetExporter(ctx context.Context, name string) (*ExporterRecord, error)
	UpdateExporter(ctx context.Context, exporter *ExporterRecord) error
	DeleteExporter(ctx context.Context, name string) error
	ListExporters(ctx context.Context) ([]string, error)
	GetExporterStatus(ctx context.Context, name string) (*ExporterStatusRecord, error)
	SetExporterStatus(ctx context.Context, name string, status *ExporterStatusRecord) error
	GetExporterConfig(ctx context.Context, name string) (map[string]string, error)
	UpdateExporterConfig(ctx context.Context, name string, config map[string]string) error

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
