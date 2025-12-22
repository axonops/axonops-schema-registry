package protobuf

import (
	"testing"
)

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

	result := checker.Check(newSchema, oldSchema)
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

	result := checker.Check(newSchema, oldSchema)
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

	result := checker.Check(newSchema, oldSchema)
	if result.IsCompatible {
		t.Error("Adding required field should be incompatible")
	}
}

func TestChecker_RemoveField_Incompatible(t *testing.T) {
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

	result := checker.Check(newSchema, oldSchema)
	if result.IsCompatible {
		t.Error("Removing field should be incompatible")
	}
	if len(result.Messages) == 0 {
		t.Error("Expected message about removed field")
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

	result := checker.Check(newSchema, oldSchema)
	if result.IsCompatible {
		t.Error("Changing field type should be incompatible")
	}
}

func TestChecker_CompatibleTypeChange_Int32ToSint32(t *testing.T) {
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

	result := checker.Check(newSchema, oldSchema)
	if !result.IsCompatible {
		t.Errorf("int32 to sint32 should be compatible: %v", result.Messages)
	}
}

func TestChecker_CompatibleTypeChange_Int64ToSint64(t *testing.T) {
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

	result := checker.Check(newSchema, oldSchema)
	if !result.IsCompatible {
		t.Errorf("int64 to sint64 should be compatible: %v", result.Messages)
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

	result := checker.Check(newSchema, oldSchema)
	if !result.IsCompatible {
		t.Errorf("Adding enum should be compatible: %v", result.Messages)
	}
}

func TestChecker_RemoveEnumValue_Incompatible(t *testing.T) {
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

	result := checker.Check(newSchema, oldSchema)
	if result.IsCompatible {
		t.Error("Removing enum value should be incompatible")
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

	result := checker.Check(newSchema, oldSchema)
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

	result := checker.Check(newSchema, oldSchema)
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

	result := checker.Check(newSchema, oldSchema)
	if result.IsCompatible {
		t.Error("Removing message should be incompatible")
	}
}

func TestChecker_NestedMessage_RemoveField(t *testing.T) {
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

	result := checker.Check(newSchema, oldSchema)
	if result.IsCompatible {
		t.Error("Removing field from nested message should be incompatible")
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

	result := checker.Check(newSchema, oldSchema)
	if !result.IsCompatible {
		t.Errorf("Adding service method should be compatible: %v", result.Messages)
	}
}

func TestChecker_Service_RemoveMethod_Incompatible(t *testing.T) {
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

	result := checker.Check(newSchema, oldSchema)
	if result.IsCompatible {
		t.Error("Removing service method should be incompatible")
	}
}

func TestChecker_Service_ChangeMethodInput_Incompatible(t *testing.T) {
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

	result := checker.Check(newSchema, oldSchema)
	if result.IsCompatible {
		t.Error("Changing service method input type should be incompatible")
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

	result := checker.Check(newSchema, oldSchema)
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

	result := checker.Check(invalidSchema, validSchema)
	if result.IsCompatible {
		t.Error("Invalid schema should return incompatible")
	}

	result = checker.Check(validSchema, invalidSchema)
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

	result := checker.Check(newSchema, oldSchema)
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

	result := checker.Check(newSchema, oldSchema)
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

	result := checker.Check(newSchema, oldSchema)
	if result.IsCompatible {
		t.Error("Reusing field number with different type should be incompatible")
	}
}

func TestChecker_RepeatedToSingular_Incompatible(t *testing.T) {
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

	result := checker.Check(newSchema, oldSchema)
	// This is a type mismatch at the wire level
	if result.IsCompatible {
		t.Error("Changing repeated to singular should be incompatible")
	}
}
