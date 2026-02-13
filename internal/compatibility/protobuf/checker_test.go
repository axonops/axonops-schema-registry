package protobuf

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
)

// s creates a SchemaWithRefs with no references for convenience.
func s(schema string) compatibility.SchemaWithRefs {
	return compatibility.SchemaWithRefs{Schema: schema}
}

func TestChecker_CompatibleSchemas(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  string name = 1;
  int32 age = 2;
}
`

	newSchema := `
syntax = "proto3";

message User {
  string name = 1;
  int32 age = 2;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible, got messages: %v", result.Messages)
	}
}

func TestChecker_AddOptionalField_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  string name = 1;
}
`

	newSchema := `
syntax = "proto3";

message User {
  string name = 1;
  int32 age = 2;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding optional field should be compatible, got: %v", result.Messages)
	}
}

func TestChecker_AddRequiredField_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto2";

message User {
  required string name = 1;
}
`

	newSchema := `
syntax = "proto2";

message User {
  required string name = 1;
  required int32 age = 2;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Adding required field should be incompatible")
	}
}

func TestChecker_RemoveField_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  string name = 1;
  int32 age = 2;
}
`

	newSchema := `
syntax = "proto3";

message User {
  string name = 1;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Removing field in proto3 should be compatible (wire-safe): %v", result.Messages)
	}
}

func TestChecker_ChangeFieldType_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  string name = 1;
  int32 age = 2;
}
`

	newSchema := `
syntax = "proto3";

message User {
  string name = 1;
  string age = 2;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Changing field type should be incompatible")
	}
}

func TestChecker_TypeChange_Int32ToSint32_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  int32 value = 1;
}
`

	newSchema := `
syntax = "proto3";

message User {
  sint32 value = 1;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("int32 to sint32 should be incompatible (different wire encoding: varint vs zigzag)")
	}
}

func TestChecker_TypeChange_Int64ToSint64_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  int64 value = 1;
}
`

	newSchema := `
syntax = "proto3";

message User {
  sint64 value = 1;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("int64 to sint64 should be incompatible (different wire encoding: varint vs zigzag)")
	}
}

func TestChecker_AddEnum_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  string name = 1;
}
`

	newSchema := `
syntax = "proto3";

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
}

message User {
  string name = 1;
  Status status = 2;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding enum should be compatible: %v", result.Messages)
	}
}

func TestChecker_RemoveEnumValue_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}

message User {
  Status status = 1;
}
`

	newSchema := `
syntax = "proto3";

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
}

message User {
  Status status = 1;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Removing enum value should be compatible (enums are integers on the wire): %v", result.Messages)
	}
}

func TestChecker_AddEnumValue_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
}

message User {
  Status status = 1;
}
`

	newSchema := `
syntax = "proto3";

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}

message User {
  Status status = 1;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding enum value should be compatible: %v", result.Messages)
	}
}

func TestChecker_AddMessage_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  string name = 1;
}
`

	newSchema := `
syntax = "proto3";

message User {
  string name = 1;
}

message Order {
  string id = 1;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding message should be compatible: %v", result.Messages)
	}
}

func TestChecker_RemoveMessage_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  string name = 1;
}

message Order {
  string id = 1;
}
`

	newSchema := `
syntax = "proto3";

message User {
  string name = 1;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Removing message should be incompatible")
	}
}

func TestChecker_NestedMessage_RemoveField_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  string name = 1;

  message Address {
    string street = 1;
    string city = 2;
  }

  Address address = 2;
}
`

	newSchema := `
syntax = "proto3";

message User {
  string name = 1;

  message Address {
    string street = 1;
  }

  Address address = 2;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Removing field from nested message should be compatible (wire-safe in proto3): %v", result.Messages)
	}
}

func TestChecker_Service_AddMethod_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message Request { string query = 1; }
message Response { string result = 1; }
message NewRequest { int32 id = 1; }
message NewResponse { string data = 1; }

service SearchService {
  rpc Search(Request) returns (Response);
}
`

	newSchema := `
syntax = "proto3";

message Request { string query = 1; }
message Response { string result = 1; }
message NewRequest { int32 id = 1; }
message NewResponse { string data = 1; }

service SearchService {
  rpc Search(Request) returns (Response);
  rpc GetById(NewRequest) returns (NewResponse);
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding service method should be compatible: %v", result.Messages)
	}
}

func TestChecker_Service_RemoveMethod_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message Request { string query = 1; }
message Response { string result = 1; }
message NewRequest { int32 id = 1; }
message NewResponse { string data = 1; }

service SearchService {
  rpc Search(Request) returns (Response);
  rpc GetById(NewRequest) returns (NewResponse);
}
`

	newSchema := `
syntax = "proto3";

message Request { string query = 1; }
message Response { string result = 1; }
message NewRequest { int32 id = 1; }
message NewResponse { string data = 1; }

service SearchService {
  rpc Search(Request) returns (Response);
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Service changes should be ignored (gRPC metadata, no wire impact): %v", result.Messages)
	}
}

func TestChecker_Service_ChangeMethodInput_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message Request { string query = 1; }
message OtherRequest { int32 id = 1; }
message Response { string result = 1; }

service SearchService {
  rpc Search(Request) returns (Response);
}
`

	newSchema := `
syntax = "proto3";

message Request { string query = 1; }
message OtherRequest { int32 id = 1; }
message Response { string result = 1; }

service SearchService {
  rpc Search(OtherRequest) returns (Response);
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Service changes should be ignored (gRPC metadata, no wire impact): %v", result.Messages)
	}
}

func TestChecker_PackageChange(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";
package com.example.v1;

message User {
  string name = 1;
}
`

	newSchema := `
syntax = "proto3";
package com.example.v2;

message User {
  string name = 1;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	// Package change is noted but may or may not be breaking
	// depending on usage
	if len(result.Messages) == 0 {
		t.Log("Package change was not noted")
	}
}

func TestChecker_InvalidSchema(t *testing.T) {
	checker := NewChecker()

	validSchema := `
syntax = "proto3";
message User { string name = 1; }
`

	invalidSchema := `this is not valid protobuf`

	result := checker.Check(s(invalidSchema), s(validSchema))
	if result.IsCompatible {
		t.Error("Invalid schema should return incompatible")
	}

	result = checker.Check(s(validSchema), s(invalidSchema))
	if result.IsCompatible {
		t.Error("Invalid old schema should return incompatible")
	}
}

func TestChecker_MapField_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message Config {
  map<string, string> settings = 1;
}
`

	newSchema := `
syntax = "proto3";

message Config {
  map<string, string> settings = 1;
  map<int32, string> codes = 2;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding map field should be compatible: %v", result.Messages)
	}
}

func TestChecker_Oneof_AddField_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message Event {
  string id = 1;
  oneof payload {
    string text = 2;
  }
}
`

	newSchema := `
syntax = "proto3";

message Event {
  string id = 1;
  oneof payload {
    string text = 2;
    int32 number = 3;
  }
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding field to oneof should be compatible: %v", result.Messages)
	}
}

func TestChecker_FieldNumberReuse_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  string name = 1;
  int32 age = 2;
}
`

	// Field 2 changed from int32 to string
	newSchema := `
syntax = "proto3";

message User {
  string name = 1;
  string email = 2;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Reusing field number with different type should be incompatible")
	}
}

func TestChecker_RepeatedToSingular_String_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `
syntax = "proto3";

message User {
  repeated string tags = 1;
}
`

	newSchema := `
syntax = "proto3";

message User {
  string tags = 1;
}
`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Repeated to singular for string should be compatible (same wire encoding): %v", result.Messages)
	}
}

// ============================================================================
// NEW TESTS: Correct wire-type compatible groups (P1)
// ============================================================================

func TestChecker_TypeChange_Int32ToUint32_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { int32 f = 1; }`
	newSchema := `syntax = "proto3"; message M { uint32 f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("int32 to uint32 should be compatible (both varint): %v", result.Messages)
	}
}

func TestChecker_TypeChange_Int32ToInt64_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { int32 f = 1; }`
	newSchema := `syntax = "proto3"; message M { int64 f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("int32 to int64 should be compatible (both varint): %v", result.Messages)
	}
}

func TestChecker_TypeChange_Int32ToBool_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { int32 f = 1; }`
	newSchema := `syntax = "proto3"; message M { bool f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("int32 to bool should be compatible (both varint): %v", result.Messages)
	}
}

func TestChecker_TypeChange_Sint32ToSint64_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { sint32 f = 1; }`
	newSchema := `syntax = "proto3"; message M { sint64 f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("sint32 to sint64 should be compatible (both zigzag varint): %v", result.Messages)
	}
}

func TestChecker_TypeChange_Fixed32ToSfixed32_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { fixed32 f = 1; }`
	newSchema := `syntax = "proto3"; message M { sfixed32 f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("fixed32 to sfixed32 should be compatible (both 32-bit wire type): %v", result.Messages)
	}
}

func TestChecker_TypeChange_Fixed64ToSfixed64_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { fixed64 f = 1; }`
	newSchema := `syntax = "proto3"; message M { sfixed64 f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("fixed64 to sfixed64 should be compatible (both 64-bit wire type): %v", result.Messages)
	}
}

func TestChecker_TypeChange_StringToBytes_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { string f = 1; }`
	newSchema := `syntax = "proto3"; message M { bytes f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("string to bytes should be compatible (both length-delimited): %v", result.Messages)
	}
}

func TestChecker_TypeChange_Int32ToFixed32_Incompatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { int32 f = 1; }`
	newSchema := `syntax = "proto3"; message M { fixed32 f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("int32 to fixed32 should be incompatible (varint vs 32-bit wire type)")
	}
}

func TestChecker_TypeChange_Sint32ToInt32_Incompatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { sint32 f = 1; }`
	newSchema := `syntax = "proto3"; message M { int32 f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("sint32 to int32 should be incompatible (zigzag vs standard varint)")
	}
}

// ============================================================================
// NEW TESTS: Enum-to-integer compatibility (P2)
// ============================================================================

func TestChecker_TypeChange_EnumToInt32_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `
syntax = "proto3";
enum Status { UNKNOWN = 0; ACTIVE = 1; }
message M { Status f = 1; }
`
	newSchema := `
syntax = "proto3";
enum Status { UNKNOWN = 0; ACTIVE = 1; }
message M { int32 f = 1; }
`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("enum to int32 should be compatible (both varint): %v", result.Messages)
	}
}

func TestChecker_TypeChange_Int32ToEnum_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `
syntax = "proto3";
enum Status { UNKNOWN = 0; ACTIVE = 1; }
message M { int32 f = 1; }
`
	newSchema := `
syntax = "proto3";
enum Status { UNKNOWN = 0; ACTIVE = 1; }
message M { Status f = 1; }
`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("int32 to enum should be compatible (both varint): %v", result.Messages)
	}
}

// ============================================================================
// NEW TESTS: Syntax change (P6)
// ============================================================================

func TestChecker_SyntaxChange_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `
syntax = "proto2";
message M { optional string f = 1; }
`
	newSchema := `
syntax = "proto3";
message M { string f = 1; }
`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Syntax change should be compatible (source-level annotation): %v", result.Messages)
	}
}

// ============================================================================
// NEW TESTS: Field removal with oneof exception (P3)
// ============================================================================

func TestChecker_RemoveOneofField_Incompatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `
syntax = "proto3";
message Event {
  string id = 1;
  oneof payload {
    string text = 2;
    int32 number = 3;
  }
}
`
	newSchema := `
syntax = "proto3";
message Event {
  string id = 1;
  oneof payload {
    string text = 2;
  }
}
`
	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Removing a field from a oneof should be incompatible (changes oneof semantics)")
	}
}

// ============================================================================
// NEW TESTS: Service ignored (P5)
// ============================================================================

func TestChecker_ServiceIgnored_ServiceRemoval(t *testing.T) {
	checker := NewChecker()
	oldSchema := `
syntax = "proto3";
message Req { string q = 1; }
message Res { string r = 1; }
service Svc { rpc Do(Req) returns (Res); }
`
	newSchema := `
syntax = "proto3";
message Req { string q = 1; }
message Res { string r = 1; }
`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Service removal should be ignored (gRPC metadata): %v", result.Messages)
	}
}

// ============================================================================
// NEW TESTS: Cardinality (P7)
// ============================================================================

func TestChecker_RepeatedToSingular_Int32_Incompatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { repeated int32 f = 1; }`
	newSchema := `syntax = "proto3"; message M { int32 f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Repeated to singular for int32 should be incompatible (different wire encoding)")
	}
}

func TestChecker_RepeatedToSingular_Message_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `
syntax = "proto3";
message Inner { string v = 1; }
message M { repeated Inner f = 1; }
`
	newSchema := `
syntax = "proto3";
message Inner { string v = 1; }
message M { Inner f = 1; }
`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Repeated to singular for message should be compatible: %v", result.Messages)
	}
}

func TestChecker_RepeatedToSingular_Bytes_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `syntax = "proto3"; message M { repeated bytes f = 1; }`
	newSchema := `syntax = "proto3"; message M { bytes f = 1; }`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Repeated to singular for bytes should be compatible: %v", result.Messages)
	}
}

// ============================================================================
// NEW TESTS: Enum type removal (P8)
// ============================================================================

func TestChecker_EnumTypeRemoval_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `
syntax = "proto3";
enum Status { UNKNOWN = 0; ACTIVE = 1; }
message M { string f = 1; }
`
	newSchema := `
syntax = "proto3";
message M { string f = 1; }
`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Enum type removal should be compatible (integer labels only): %v", result.Messages)
	}
}

func TestChecker_NestedEnumRemoval_Compatible(t *testing.T) {
	checker := NewChecker()
	oldSchema := `
syntax = "proto3";
message M {
  string f = 1;
  enum Status { UNKNOWN = 0; ACTIVE = 1; }
}
`
	newSchema := `
syntax = "proto3";
message M {
  string f = 1;
}
`
	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Nested enum type removal should be compatible: %v", result.Messages)
	}
}
