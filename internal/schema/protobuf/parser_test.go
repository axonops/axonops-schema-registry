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

func TestParser_Parse_DeeplyNestedMessages(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message Level1 {
  string name = 1;

  message Level2 {
    int32 value = 1;

    message Level3 {
      string data = 1;

      message Level4 {
        bool flag = 1;
        int64 count = 2;
      }

      Level4 deep = 2;
    }

    Level3 nested = 2;
  }

  Level2 child = 2;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse deeply nested messages: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "message Level1") {
		t.Errorf("Should contain Level1: %s", canonical)
	}
	if !strings.Contains(canonical, "message Level4") {
		t.Errorf("Should contain Level4: %s", canonical)
	}
}

func TestParser_Parse_MapOfComplexTypes(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message Metadata {
  string key = 1;
  string value = 2;
}

message Container {
  map<string, Metadata> entries = 1;
  map<int32, string> labels = 2;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse map of complex types: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_MultipleTopLevelMessages(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";
package com.example.events;

message UserCreated {
  string user_id = 1;
  string email = 2;
  int64 created_at = 3;
}

message UserUpdated {
  string user_id = 1;
  string field = 2;
  string old_value = 3;
  string new_value = 4;
}

message UserDeleted {
  string user_id = 1;
  int64 deleted_at = 2;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse multiple top-level messages: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "message UserCreated") {
		t.Errorf("Should contain UserCreated: %s", canonical)
	}
	if !strings.Contains(canonical, "message UserUpdated") {
		t.Errorf("Should contain UserUpdated: %s", canonical)
	}
	if !strings.Contains(canonical, "message UserDeleted") {
		t.Errorf("Should contain UserDeleted: %s", canonical)
	}
}

func TestParser_Parse_ComplexRealWorld(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";
package com.example.payments;

import "google/protobuf/timestamp.proto";

enum Currency {
  CURRENCY_UNSPECIFIED = 0;
  USD = 1;
  EUR = 2;
  GBP = 3;
}

enum PaymentStatus {
  STATUS_UNSPECIFIED = 0;
  PENDING = 1;
  COMPLETED = 2;
  FAILED = 3;
  REFUNDED = 4;
}

message Address {
  string street = 1;
  string city = 2;
  string country = 3;
  string zip = 4;
}

message Customer {
  int64 id = 1;
  string name = 2;
  string email = 3;
  Address address = 4;
}

message LineItem {
  string product_id = 1;
  int32 quantity = 2;
  int64 unit_price_cents = 3;
}

message PaymentEvent {
  string event_id = 1;
  google.protobuf.Timestamp timestamp = 2;
  int64 amount_cents = 3;
  Currency currency = 4;
  PaymentStatus status = 5;
  Customer customer = 6;
  repeated LineItem items = 7;
  map<string, string> metadata = 8;
  string notes = 9;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse complex real-world schema: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "message PaymentEvent") {
		t.Errorf("Should contain PaymentEvent: %s", canonical)
	}
	if !strings.Contains(canonical, "enum Currency") {
		t.Errorf("Should contain Currency enum: %s", canonical)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Fingerprint should not be empty")
	}
}

func TestParser_Parse_Proto3OptionalField(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message UserProfile {
  string name = 1;
  optional string nickname = 2;
  optional int32 age = 3;
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse proto3 optional fields: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_StreamingService(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";

message Request {
  string query = 1;
}

message Response {
  string result = 1;
}

service StreamService {
  rpc ServerStream(Request) returns (stream Response);
  rpc ClientStream(stream Request) returns (Response);
  rpc BidiStream(stream Request) returns (stream Response);
}
`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse streaming service: %v", err)
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "service StreamService") {
		t.Errorf("Should contain StreamService: %s", canonical)
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

// --- Schema Reference Tests ---

func TestParser_Parse_WithEmptyReferences(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";
message Simple { string id = 1; }
`

	parsed, err := parser.Parse(schema, []storage.Reference{})
	if err != nil {
		t.Fatalf("Parse with empty references should not fail: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_CrossSubjectImport(t *testing.T) {
	parser := NewParser()

	// Referenced schema defines CommonMessage
	commonSchema := `
syntax = "proto3";
package common;
message CommonMessage {
  string id = 1;
  string name = 2;
}
`

	// Main schema imports the referenced proto
	mainSchema := `
syntax = "proto3";
import "common.proto";
message Event {
  string event_id = 1;
  common.CommonMessage payload = 2;
}
`

	refs := []storage.Reference{
		{Name: "common.proto", Subject: "common-value", Version: 1, Schema: commonSchema},
	}

	parsed, err := parser.Parse(mainSchema, refs)
	if err != nil {
		t.Fatalf("Parse with cross-subject import failed: %v", err)
	}
	if parsed.Type() != storage.SchemaTypeProtobuf {
		t.Errorf("Expected PROTOBUF type, got %s", parsed.Type())
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}

	canonical := parsed.CanonicalString()
	if !strings.Contains(canonical, "message Event") {
		t.Errorf("Canonical should contain Event message: %s", canonical)
	}
}

func TestParser_Parse_CrossSubjectImport_MultipleRefs(t *testing.T) {
	parser := NewParser()

	addressSchema := `
syntax = "proto3";
package types;
message Address {
  string street = 1;
  string city = 2;
}
`

	customerSchema := `
syntax = "proto3";
package types;
message Customer {
  string id = 1;
  string name = 2;
}
`

	mainSchema := `
syntax = "proto3";
import "address.proto";
import "customer.proto";
message Order {
  string order_id = 1;
  types.Customer buyer = 2;
  types.Address shipping = 3;
}
`

	refs := []storage.Reference{
		{Name: "address.proto", Subject: "address-value", Version: 1, Schema: addressSchema},
		{Name: "customer.proto", Subject: "customer-value", Version: 1, Schema: customerSchema},
	}

	parsed, err := parser.Parse(mainSchema, refs)
	if err != nil {
		t.Fatalf("Parse with multiple cross-subject imports failed: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_WellKnownTypesWithReferences(t *testing.T) {
	parser := NewParser()

	commonSchema := `
syntax = "proto3";
package common;
message Metadata {
  string key = 1;
  string value = 2;
}
`

	// Imports both well-known types and a custom reference
	mainSchema := `
syntax = "proto3";
import "google/protobuf/timestamp.proto";
import "common.proto";
message Event {
  string id = 1;
  google.protobuf.Timestamp created_at = 2;
  common.Metadata meta = 3;
}
`

	refs := []storage.Reference{
		{Name: "common.proto", Subject: "common-value", Version: 1, Schema: commonSchema},
	}

	parsed, err := parser.Parse(mainSchema, refs)
	if err != nil {
		t.Fatalf("Parse with well-known types and references failed: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_ImportFailsWhenRefContentMissing(t *testing.T) {
	parser := NewParser()

	// Schema imports a file but reference has no content
	schema := `
syntax = "proto3";
import "common.proto";
message Event {
  string id = 1;
}
`

	refs := []storage.Reference{
		{Name: "common.proto", Subject: "common-value", Version: 1},
	}

	_, err := parser.Parse(schema, refs)
	if err == nil {
		t.Error("Expected error when importing reference with no content")
	}
}

func TestParser_Parse_ImportUnknownFileWithoutReference(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";
import "unknown.proto";
message Event {
  string id = 1;
}
`

	_, err := parser.Parse(schema, nil)
	if err == nil {
		t.Error("Expected error when importing unknown file")
	}
}

func TestParser_Parse_UnusedReferencesGraceful(t *testing.T) {
	parser := NewParser()

	commonSchema := `
syntax = "proto3";
message Unused { string x = 1; }
`

	// Schema doesn't import the referenced proto
	schema := `
syntax = "proto3";
message Simple { string id = 1; }
`

	refs := []storage.Reference{
		{Name: "common.proto", Subject: "common-value", Version: 1, Schema: commonSchema},
	}

	parsed, err := parser.Parse(schema, refs)
	if err != nil {
		t.Fatalf("Parse with unused references should not fail: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_ReferencesStoredOnParsedSchema(t *testing.T) {
	parser := NewParser()

	schema := `
syntax = "proto3";
message Simple { string id = 1; }
`

	refs := []storage.Reference{
		{Name: "common.proto", Subject: "common-value", Version: 1, Schema: `syntax = "proto3"; message C { string x = 1; }`},
		{Name: "types.proto", Subject: "types-value", Version: 2, Schema: `syntax = "proto3"; message T { string y = 1; }`},
	}

	parsed, err := parser.Parse(schema, refs)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	protoParsed, ok := parsed.(*ParsedProtobuf)
	if !ok {
		t.Fatal("Expected *ParsedProtobuf")
	}

	if len(protoParsed.references) != 2 {
		t.Errorf("Expected 2 stored references, got %d", len(protoParsed.references))
	}
}

// --- Resolver Tests ---

func TestResolver_FindFileByPath_WellKnownTypes(t *testing.T) {
	resolver := newReferenceResolver()

	wellKnownFiles := []string{
		"google/protobuf/timestamp.proto",
		"google/protobuf/any.proto",
		"google/protobuf/duration.proto",
		"google/protobuf/empty.proto",
		"google/protobuf/struct.proto",
		"google/protobuf/wrappers.proto",
		"google/protobuf/field_mask.proto",
		"google/protobuf/descriptor.proto",
	}

	for _, path := range wellKnownFiles {
		t.Run(path, func(t *testing.T) {
			result, err := resolver.FindFileByPath(path)
			if err != nil {
				t.Fatalf("FindFileByPath(%s) failed: %v", path, err)
			}
			if result.Source == nil {
				t.Errorf("Expected non-nil source for %s", path)
			}
		})
	}
}

func TestResolver_FindFileByPath_NotFound(t *testing.T) {
	resolver := newReferenceResolver()

	_, err := resolver.FindFileByPath("nonexistent.proto")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	fnfErr, ok := err.(*fileNotFoundError)
	if !ok {
		t.Errorf("Expected *fileNotFoundError, got %T", err)
	}
	if fnfErr.path != "nonexistent.proto" {
		t.Errorf("Expected path nonexistent.proto, got %s", fnfErr.path)
	}
}

func TestResolver_WithReferencesAndSchema_ResolvesContent(t *testing.T) {
	baseResolver := newReferenceResolver()
	refs := []storage.Reference{
		{Name: "common.proto", Subject: "common", Version: 1, Schema: `syntax = "proto3"; message Foo {}`},
	}
	resolver := baseResolver.withReferencesAndSchema("syntax = \"proto3\"; message Bar {}", refs)

	// Main schema resolves
	result, err := resolver.FindFileByPath("schema.proto")
	if err != nil {
		t.Fatalf("FindFileByPath(schema.proto) failed: %v", err)
	}
	if result.Source == nil {
		t.Error("Expected non-nil source for schema.proto")
	}

	// Referenced schema resolves with content
	result, err = resolver.FindFileByPath("common.proto")
	if err != nil {
		t.Fatalf("FindFileByPath(common.proto) failed: %v", err)
	}
	if result.Source == nil {
		t.Error("Expected non-nil source for common.proto")
	}
}

func TestResolver_WithReferencesAndSchema_EmptyContentRef(t *testing.T) {
	baseResolver := newReferenceResolver()
	refs := []storage.Reference{
		{Name: "common.proto", Subject: "common", Version: 1},
	}
	resolver := baseResolver.withReferencesAndSchema("syntax = \"proto3\";", refs)

	// Reference with empty content won't resolve
	_, err := resolver.FindFileByPath("common.proto")
	if err == nil {
		t.Error("Expected error for reference with empty content")
	}
}
