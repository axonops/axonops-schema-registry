package protobuf

import (
	"io"
	"strings"

	"github.com/bufbuild/protocompile"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// referenceResolver resolves protobuf imports from schema references.
type referenceResolver struct {
	refs      map[string]string // name -> schema content
	wellKnown map[string]string // well-known type imports
}

// newReferenceResolver creates a new reference resolver.
func newReferenceResolver() *referenceResolver {
	return &referenceResolver{
		refs:      make(map[string]string),
		wellKnown: getWellKnownTypes(),
	}
}

// withReferencesAndSchema returns a new resolver with the schema and references.
func (r *referenceResolver) withReferencesAndSchema(schema string, refs []storage.Reference) protocompile.Resolver {
	newResolver := &referenceResolver{
		refs:      make(map[string]string),
		wellKnown: r.wellKnown,
	}

	// Add the main schema
	newResolver.refs["schema.proto"] = schema

	// Add references with their resolved content
	for _, ref := range refs {
		if ref.Name != "" {
			newResolver.refs[ref.Name] = ref.Schema
		}
	}

	return newResolver
}

// FindFileByPath implements protocompile.Resolver.
func (r *referenceResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	// Check well-known types first
	if content, ok := r.wellKnown[path]; ok && content != "" {
		return protocompile.SearchResult{
			Source: strings.NewReader(content),
		}, nil
	}

	// Check references
	if content, ok := r.refs[path]; ok && content != "" {
		return protocompile.SearchResult{
			Source: strings.NewReader(content),
		}, nil
	}

	// Return not found - protocompile will handle this
	return protocompile.SearchResult{}, &fileNotFoundError{path: path}
}

// fileNotFoundError indicates a file was not found.
type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

// getWellKnownTypes returns proto definitions for well-known types.
// Note: descriptor.proto is intentionally left empty so that
// protocompile.WithStandardImports provides the full definition
// (which includes complete Options messages needed for features
// like allow_alias and packed).
func getWellKnownTypes() map[string]string {
	return map[string]string{
		"google/protobuf/any.proto": `
syntax = "proto3";
package google.protobuf;
message Any {
  string type_url = 1;
  bytes value = 2;
}`,
		"google/protobuf/timestamp.proto": `
syntax = "proto3";
package google.protobuf;
message Timestamp {
  int64 seconds = 1;
  int32 nanos = 2;
}`,
		"google/protobuf/duration.proto": `
syntax = "proto3";
package google.protobuf;
message Duration {
  int64 seconds = 1;
  int32 nanos = 2;
}`,
		"google/protobuf/empty.proto": `
syntax = "proto3";
package google.protobuf;
message Empty {}`,
		"google/protobuf/struct.proto": `
syntax = "proto3";
package google.protobuf;
message Struct {
  map<string, Value> fields = 1;
}
message Value {
  oneof kind {
    NullValue null_value = 1;
    double number_value = 2;
    string string_value = 3;
    bool bool_value = 4;
    Struct struct_value = 5;
    ListValue list_value = 6;
  }
}
message ListValue {
  repeated Value values = 1;
}
enum NullValue {
  NULL_VALUE = 0;
}`,
		"google/protobuf/wrappers.proto": `
syntax = "proto3";
package google.protobuf;
message DoubleValue { double value = 1; }
message FloatValue { float value = 1; }
message Int64Value { int64 value = 1; }
message UInt64Value { uint64 value = 1; }
message Int32Value { int32 value = 1; }
message UInt32Value { uint32 value = 1; }
message BoolValue { bool value = 1; }
message StringValue { string value = 1; }
message BytesValue { bytes value = 1; }`,
		"google/protobuf/field_mask.proto": `
syntax = "proto3";
package google.protobuf;
message FieldMask {
  repeated string paths = 1;
}`,
		// Note: at compile time, descriptor.proto is provided by
		// protocompile.WithStandardImports (via CompositeResolver priority)
		// with the full definition including complete Options messages.
		// This stub exists only for backward compatibility with direct
		// FindFileByPath callers.
		"google/protobuf/descriptor.proto": `syntax = "proto2"; package google.protobuf;`,
	}
}

// notFoundResolver is a resolver that always returns not-found. It is used as
// the inner resolver for protocompile.WithStandardImports so that standard
// imports are always provided from the real protobuf definitions.
type notFoundResolver struct{}

func (notFoundResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	return protocompile.SearchResult{}, &fileNotFoundError{path: path}
}

// Ensure referenceResolver implements the required interface
var _ protocompile.Resolver = (*referenceResolver)(nil)

// Satisfy unused import
var _ io.Reader = nil
