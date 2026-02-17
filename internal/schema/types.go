// Package schema provides schema parsing and handling.
package schema

import (
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// ParsedSchema represents a parsed schema with metadata.
type ParsedSchema interface {
	// Type returns the schema type.
	Type() storage.SchemaType

	// CanonicalString returns the canonical form of the schema.
	CanonicalString() string

	// Fingerprint returns a unique fingerprint for the schema.
	Fingerprint() string

	// RawSchema returns the underlying schema object.
	RawSchema() interface{}

	// FormattedString returns the schema formatted according to the given format.
	// Supported formats vary by schema type (e.g., "resolved" for Avro).
	// Returns canonical string for unknown or empty format values.
	FormattedString(format string) string

	// Normalize returns a normalized copy of this schema with deterministic
	// representation for deduplication and comparison purposes.
	Normalize() ParsedSchema

	// HasTopLevelField reports whether the schema contains a top-level field
	// with the given name. For Avro records this checks record fields, for
	// Protobuf it checks fields across all top-level messages, and for JSON
	// Schema it checks the "properties" object.
	HasTopLevelField(field string) bool
}

// Parser is the interface for schema parsers.
type Parser interface {
	// Parse parses a schema string.
	Parse(schemaStr string, references []storage.Reference) (ParsedSchema, error)

	// Type returns the schema type this parser handles.
	Type() storage.SchemaType
}

// Registry holds registered schema parsers.
type Registry struct {
	parsers map[storage.SchemaType]Parser
}

// NewRegistry creates a new schema registry.
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[storage.SchemaType]Parser),
	}
}

// Register registers a parser for a schema type.
func (r *Registry) Register(parser Parser) {
	r.parsers[parser.Type()] = parser
}

// Get returns the parser for a schema type.
func (r *Registry) Get(schemaType storage.SchemaType) (Parser, bool) {
	parser, ok := r.parsers[schemaType]
	return parser, ok
}

// Types returns all supported schema types.
func (r *Registry) Types() []string {
	types := make([]string, 0, len(r.parsers))
	for t := range r.parsers {
		types = append(types, string(t))
	}
	return types
}
