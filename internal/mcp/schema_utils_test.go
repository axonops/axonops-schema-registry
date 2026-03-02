package mcp

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func TestExtractAvroFields(t *testing.T) {
	schema := `{"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}`

	fields := ExtractFields(schema, storage.SchemaTypeAvro)
	if len(fields) == 0 {
		t.Fatal("expected fields from Avro schema")
	}

	fieldMap := make(map[string]FieldInfo)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	if f, ok := fieldMap["id"]; !ok {
		t.Error("expected 'id' field")
	} else if f.Type != "int" {
		t.Errorf("expected id type 'int', got %q", f.Type)
	}

	if f, ok := fieldMap["name"]; !ok {
		t.Error("expected 'name' field")
	} else if f.Type != "string" {
		t.Errorf("expected name type 'string', got %q", f.Type)
	}

	if f, ok := fieldMap["email"]; !ok {
		t.Error("expected 'email' field")
	} else {
		if f.Required {
			t.Error("expected email to be optional (nullable union)")
		}
		if !f.HasDefault {
			t.Error("expected email to have a default")
		}
	}
}

func TestExtractAvroNestedFields(t *testing.T) {
	schema := `{"type":"record","name":"User","fields":[{"name":"address","type":{"type":"record","name":"Address","fields":[{"name":"street","type":"string"},{"name":"city","type":"string"}]}}]}`

	fields := ExtractFields(schema, storage.SchemaTypeAvro)
	paths := make(map[string]bool)
	for _, f := range fields {
		paths[f.Path] = true
	}

	if !paths["address"] {
		t.Error("expected 'address' path")
	}
	if !paths["address.street"] {
		t.Error("expected 'address.street' path")
	}
	if !paths["address.city"] {
		t.Error("expected 'address.city' path")
	}
}

func TestExtractJSONSchemaFields(t *testing.T) {
	schema := `{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string","description":"User name"}},"required":["id"]}`

	fields := ExtractFields(schema, storage.SchemaTypeJSON)
	if len(fields) == 0 {
		t.Fatal("expected fields from JSON Schema")
	}

	fieldMap := make(map[string]FieldInfo)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	if f, ok := fieldMap["id"]; !ok {
		t.Error("expected 'id' field")
	} else {
		if !f.Required {
			t.Error("expected id to be required")
		}
		if f.Type != "integer" {
			t.Errorf("expected type 'integer', got %q", f.Type)
		}
	}

	if f, ok := fieldMap["name"]; !ok {
		t.Error("expected 'name' field")
	} else {
		if f.Required {
			t.Error("expected name to be optional")
		}
		if f.Doc != "User name" {
			t.Errorf("expected doc 'User name', got %q", f.Doc)
		}
	}
}

func TestExtractProtobufFields(t *testing.T) {
	schema := `syntax = "proto3";
message User {
  int32 id = 1;
  string name = 2;
  optional bool active = 3;
}`
	fields := ExtractFields(schema, storage.SchemaTypeProtobuf)
	if len(fields) == 0 {
		t.Fatal("expected fields from Protobuf schema")
	}

	fieldMap := make(map[string]FieldInfo)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	if f, ok := fieldMap["id"]; !ok {
		t.Error("expected 'id' field")
	} else if f.Type != "int32" {
		t.Errorf("expected type 'int32', got %q", f.Type)
	}

	if _, ok := fieldMap["name"]; !ok {
		t.Error("expected 'name' field")
	}
}

func TestNormalizeFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"userName", "user_name"},
		{"UserName", "user_name"},
		{"user_name", "user_name"},
		{"user-name", "user_name"},
		{"user.name", "user_name"},
		{"firstName", "first_name"},
		{"HTTPRequest", "httprequest"}, // consecutive uppercase stays lowercase without separators
		{"id", "id"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizeFieldName(tc.input)
			if got != tc.expected {
				t.Errorf("NormalizeFieldName(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestExtractFieldsEmptySchema(t *testing.T) {
	fields := ExtractFields("", storage.SchemaTypeAvro)
	if fields != nil {
		t.Errorf("expected nil for empty schema, got %d fields", len(fields))
	}
}

func TestExtractFieldsUnknownType(t *testing.T) {
	fields := ExtractFields(`{"type":"string"}`, "UNKNOWN")
	if fields != nil {
		t.Errorf("expected nil for unknown type, got %d fields", len(fields))
	}
}
