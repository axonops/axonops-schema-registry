package protobuf

import (
	"strings"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func TestParser_Parse_SimpleMessage(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message User {
  string name = 1;
  int32 age = 2;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}

	if parsed.Type() != storage.SchemaTypeProtobuf {
		t.Errorf("Expected type %s, got %s", storage.SchemaTypeProtobuf, parsed.Type())
	}

	// Check that canonical string contains expected elements
	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "message User") {
		t.Errorf("Canonical string should contain 'message User': %s", canonical)
	}
	if !strings.Contains(canonical, "string name = 1") {
		t.Errorf("Canonical string should contain 'string name = 1': %s", canonical)
	}
	if !strings.Contains(canonical, "int32 age = 2") {
		t.Errorf("Canonical string should contain 'int32 age = 2': %s", canonical)
	}

	// Fingerprint should be non-empty
	fingerprint := parsed.Fingerprint()
	if fingerprint == "" {
		t.Error("Fingerprint should not be empty")
	}
}

func TestParser_Parse_WithPackage(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";
package com.example;

message Order {
  int64 id = 1;
  string product = 2;
  int32 quantity = 3;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "package com.example") {
		t.Errorf("Canonical string should contain package: %s", canonical)
	}
}

func TestParser_Parse_NestedMessage(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message Outer {
  string name = 1;

  message Inner {
    int32 value = 1;
  }

  Inner inner = 2;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "message Outer") {
		t.Errorf("Should contain Outer message: %s", canonical)
	}
	if !strings.Contains(canonical, "message Inner") {
		t.Errorf("Should contain nested Inner message: %s", canonical)
	}
}

func TestParser_Parse_Enum(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}

message Task {
  string name = 1;
  Status status = 2;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "enum Status") {
		t.Errorf("Should contain enum: %s", canonical)
	}
	if !strings.Contains(canonical, "UNKNOWN = 0") {
		t.Errorf("Should contain enum value: %s", canonical)
	}
}

func TestParser_Parse_Service(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message Request {
  string query = 1;
}

message Response {
  string result = 1;
}

service SearchService {
  rpc Search(Request) returns (Response);
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "service SearchService") {
		t.Errorf("Should contain service: %s", canonical)
	}
	if !strings.Contains(canonical, "rpc Search") {
		t.Errorf("Should contain rpc method: %s", canonical)
	}
}

func TestParser_Parse_RepeatedFields(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message Container {
  repeated string items = 1;
  repeated int32 numbers = 2;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "repeated string items") {
		t.Errorf("Should contain repeated field: %s", canonical)
	}
}

func TestParser_Parse_MapField(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message Config {
  map<string, string> settings = 1;
  map<int32, string> codes = 2;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "map<string, string> settings") {
		t.Errorf("Should contain map field: %s", canonical)
	}
}

func TestParser_Parse_Oneof(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message Event {
  string id = 1;

  oneof payload {
    string text = 2;
    int32 number = 3;
    bytes data = 4;
  }
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "oneof payload") {
		t.Errorf("Should contain oneof: %s", canonical)
	}
}

func TestParser_Parse_Proto2Syntax(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto2";

message LegacyMessage {
  required string id = 1;
  optional string name = 2;
  repeated int32 values = 3;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, `syntax = "proto2"`) {
		t.Errorf("Should indicate proto2 syntax: %s", canonical)
	}
	if !strings.Contains(canonical, "required string id") {
		t.Errorf("Should contain required field: %s", canonical)
	}
}

func TestParser_Parse_WellKnownTypes(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

import "google/protobuf/timestamp.proto";
import "google/protobuf/any.proto";

message Event {
  string id = 1;
  google.protobuf.Timestamp created_at = 2;
  google.protobuf.Any payload = 3;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema with well-known types: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_AllPrimitiveTypes(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message AllTypes {
  double double_val = 1;
  float float_val = 2;
  int32 int32_val = 3;
  int64 int64_val = 4;
  uint32 uint32_val = 5;
  uint64 uint64_val = 6;
  sint32 sint32_val = 7;
  sint64 sint64_val = 8;
  fixed32 fixed32_val = 9;
  fixed64 fixed64_val = 10;
  sfixed32 sfixed32_val = 11;
  sfixed64 sfixed64_val = 12;
  bool bool_val = 13;
  string string_val = 14;
  bytes bytes_val = 15;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema with all primitive types: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_InvalidSchema(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name:   "missing syntax",
			schema: `message User { string name = 1; }`,
		},
		{
			name:   "invalid field number",
			schema: `syntax = "proto3"; message User { string name = -1; }`,
		},
		{
			name:   "duplicate field number",
			schema: `syntax = "proto3"; message User { string name = 1; int32 age = 1; }`,
		},
		{
			name:   "invalid syntax",
			schema: `this is not valid protobuf`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.schema, nil)
			if err == nil {
				t.Error("Expected error for invalid schema")
			}
		})
	}
}

func TestParser_Fingerprint_Deterministic(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message User {
  string name = 1;
  int32 age = 2;
}
`

	parsed1, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	parsed2, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	if parsed1.Fingerprint() != parsed2.Fingerprint() {
		t.Error("Fingerprints should be identical for same schema")
	}
}

func TestParser_Fingerprint_DifferentForDifferentSchemas(t *testing.T) {
	parser := NewParser()

	schema1 := `
syntax = "proto3";
message User { string name = 1; }
`

	schema2 := `
syntax = "proto3";
message User { string name = 1; int32 age = 2; }
`

	parsed1, _ := parser.Parse(schema1, nil)
	parsed2, _ := parser.Parse(schema2, nil)

	if parsed1.Fingerprint() == parsed2.Fingerprint() {
		t.Error("Fingerprints should be different for different schemas")
	}
}

func TestParser_Type(t *testing.T) {
	parser := NewParser()

	if parser.Type() != storage.SchemaTypeProtobuf {
		t.Errorf("Expected type %s, got %s", storage.SchemaTypeProtobuf, parser.Type())
	}
}
