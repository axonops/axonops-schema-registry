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

	// Add references by name
	for _, ref := range refs {
		// In a full implementation, we would load the referenced schema content
		// For now, store the reference name for import resolution
		if ref.Name != "" {
			newResolver.refs[ref.Name] = "" // Content would be loaded on demand
		}
	}

	return newResolver
}

// FindFileByPath implements protocompile.Resolver.
func (r *referenceResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	// Check well-known types first
	if content, ok := r.wellKnown[path]; ok {
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
		"google/protobuf/descriptor.proto": descriptorProto,
	}
}

// descriptorProto is a minimal descriptor.proto for self-describing messages
const descriptorProto = `
syntax = "proto2";
package google.protobuf;

message FileDescriptorSet {
  repeated FileDescriptorProto file = 1;
}

message FileDescriptorProto {
  optional string name = 1;
  optional string package = 2;
  repeated string dependency = 3;
  repeated int32 public_dependency = 10;
  repeated int32 weak_dependency = 11;
  repeated DescriptorProto message_type = 4;
  repeated EnumDescriptorProto enum_type = 5;
  repeated ServiceDescriptorProto service = 6;
  repeated FieldDescriptorProto extension = 7;
  optional FileOptions options = 8;
  optional SourceCodeInfo source_code_info = 9;
  optional string syntax = 12;
}

message DescriptorProto {
  optional string name = 1;
  repeated FieldDescriptorProto field = 2;
  repeated FieldDescriptorProto extension = 6;
  repeated DescriptorProto nested_type = 3;
  repeated EnumDescriptorProto enum_type = 4;
  repeated OneofDescriptorProto oneof_decl = 8;
  optional MessageOptions options = 7;
}

message FieldDescriptorProto {
  optional string name = 1;
  optional int32 number = 3;
  optional Label label = 4;
  optional Type type = 5;
  optional string type_name = 6;
  optional string extendee = 2;
  optional string default_value = 7;
  optional int32 oneof_index = 9;
  optional string json_name = 10;
  optional FieldOptions options = 8;

  enum Type {
    TYPE_DOUBLE = 1;
    TYPE_FLOAT = 2;
    TYPE_INT64 = 3;
    TYPE_UINT64 = 4;
    TYPE_INT32 = 5;
    TYPE_FIXED64 = 6;
    TYPE_FIXED32 = 7;
    TYPE_BOOL = 8;
    TYPE_STRING = 9;
    TYPE_GROUP = 10;
    TYPE_MESSAGE = 11;
    TYPE_BYTES = 12;
    TYPE_UINT32 = 13;
    TYPE_ENUM = 14;
    TYPE_SFIXED32 = 15;
    TYPE_SFIXED64 = 16;
    TYPE_SINT32 = 17;
    TYPE_SINT64 = 18;
  }

  enum Label {
    LABEL_OPTIONAL = 1;
    LABEL_REQUIRED = 2;
    LABEL_REPEATED = 3;
  }
}

message OneofDescriptorProto {
  optional string name = 1;
  optional OneofOptions options = 2;
}

message EnumDescriptorProto {
  optional string name = 1;
  repeated EnumValueDescriptorProto value = 2;
  optional EnumOptions options = 3;
}

message EnumValueDescriptorProto {
  optional string name = 1;
  optional int32 number = 2;
  optional EnumValueOptions options = 3;
}

message ServiceDescriptorProto {
  optional string name = 1;
  repeated MethodDescriptorProto method = 2;
  optional ServiceOptions options = 3;
}

message MethodDescriptorProto {
  optional string name = 1;
  optional string input_type = 2;
  optional string output_type = 3;
  optional MethodOptions options = 4;
  optional bool client_streaming = 5;
  optional bool server_streaming = 6;
}

message FileOptions {}
message MessageOptions {}
message FieldOptions {}
message OneofOptions {}
message EnumOptions {}
message EnumValueOptions {}
message ServiceOptions {}
message MethodOptions {}
message SourceCodeInfo {}
`

// Ensure referenceResolver implements the required interface
var _ protocompile.Resolver = (*referenceResolver)(nil)

// Satisfy unused import
var _ io.Reader = nil
