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
	Schema string `json:"schema"`
}

// SubjectVersionResponse is the response for getting a subject version.
type SubjectVersionResponse struct {
	Subject    string              `json:"subject"`
	ID         int64               `json:"id"`
	Version    int                 `json:"version"`
	SchemaType string              `json:"schemaType"`
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
	SchemaType string              `json:"schemaType"`
	Schema     string              `json:"schema"`
	References []storage.Reference `json:"references,omitempty"`
}

// ConfigResponse is the response for getting configuration.
type ConfigResponse struct {
	CompatibilityLevel string `json:"compatibilityLevel"`
}

// ConfigRequest is the request body for setting configuration.
type ConfigRequest struct {
	Compatibility string `json:"compatibility"`
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
	ErrorCodeInvalidCompatibilityLevel = 42203
	ErrorCodeInvalidMode               = 42204
	ErrorCodeOperationNotPermitted     = 42205
	ErrorCodeReferenceExists           = 42206
	ErrorCodeInternalServerError       = 50001
	ErrorCodeStorageError              = 50002
)
