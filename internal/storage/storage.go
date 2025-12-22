// Package storage provides storage interfaces and implementations for the schema registry.
package storage

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrNotFound        = errors.New("not found")
	ErrSubjectNotFound = errors.New("subject not found")
	ErrSchemaNotFound  = errors.New("schema not found")
	ErrVersionNotFound = errors.New("version not found")
	ErrSubjectDeleted  = errors.New("subject has been deleted")
	ErrSchemaExists    = errors.New("schema already exists")
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
	ID          int64      `json:"id"`
	Subject     string     `json:"subject"`
	Version     int        `json:"version"`
	SchemaType  SchemaType `json:"schemaType"`
	Schema      string     `json:"schema"`
	References  []Reference `json:"references,omitempty"`
	Fingerprint string     `json:"-"`
	Deleted     bool       `json:"-"`
	CreatedAt   time.Time  `json:"-"`
}

// Reference represents a schema reference.
type Reference struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

// SubjectVersion represents a subject-version pair.
type SubjectVersion struct {
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

// ConfigRecord represents a compatibility configuration.
type ConfigRecord struct {
	Subject           string `json:"subject,omitempty"` // Empty for global config
	CompatibilityLevel string `json:"compatibilityLevel"`
}

// ModeRecord represents a mode configuration.
type ModeRecord struct {
	Subject string `json:"subject,omitempty"` // Empty for global mode
	Mode    string `json:"mode"`
}

// Storage defines the interface for schema storage backends.
type Storage interface {
	// Schema operations
	CreateSchema(ctx context.Context, record *SchemaRecord) error
	GetSchemaByID(ctx context.Context, id int64) (*SchemaRecord, error)
	GetSchemaBySubjectVersion(ctx context.Context, subject string, version int) (*SchemaRecord, error)
	GetSchemasBySubject(ctx context.Context, subject string, includeDeleted bool) ([]*SchemaRecord, error)
	GetSchemaByFingerprint(ctx context.Context, subject, fingerprint string) (*SchemaRecord, error)
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
