package avro

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
)

// s creates a SchemaWithRefs with no references for convenience.
func s(schema string) compatibility.SchemaWithRefs {
	return compatibility.SchemaWithRefs{Schema: schema}
}

func TestChecker_BackwardCompatible_AddOptionalField(t *testing.T) {
	checker := NewChecker()

	// Original schema
	writerSchema := `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "id", "type": "long"}
		]
	}`

	// New schema with optional field (has default)
	readerSchema := `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "name", "type": "string", "default": ""}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible, got incompatible: %v", result.Messages)
	}
}

func TestChecker_BackwardIncompatible_AddRequiredField(t *testing.T) {
	checker := NewChecker()

	// Original schema
	writerSchema := `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "id", "type": "long"}
		]
	}`

	// New schema with required field (no default)
	readerSchema := `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "name", "type": "string"}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if result.IsCompatible {
		t.Error("Expected incompatible, got compatible")
	}
	if len(result.Messages) == 0 {
		t.Error("Expected error messages")
	}
}

func TestChecker_BackwardCompatible_RemoveField(t *testing.T) {
	checker := NewChecker()

	// Original schema with two fields
	writerSchema := `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "name", "type": "string"}
		]
	}`

	// New schema with one field removed (backward compatible - reader ignores extra fields)
	readerSchema := `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "id", "type": "long"}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible, got incompatible: %v", result.Messages)
	}
}

func TestChecker_TypePromotion_IntToLong(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": "int"}
		]
	}`

	readerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": "long"}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible (int->long promotion), got incompatible: %v", result.Messages)
	}
}

func TestChecker_TypePromotion_IntToDouble(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": "int"}
		]
	}`

	readerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": "double"}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible (int->double promotion), got incompatible: %v", result.Messages)
	}
}

func TestChecker_TypePromotion_FloatToDouble(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": "float"}
		]
	}`

	readerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": "double"}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible (float->double promotion), got incompatible: %v", result.Messages)
	}
}

func TestChecker_IncompatibleTypeChange(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": "string"}
		]
	}`

	readerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": "int"}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if result.IsCompatible {
		t.Error("Expected incompatible (string->int), got compatible")
	}
}

func TestChecker_UnionCompatibility_AddTypeToUnion(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": ["null", "string"]}
		]
	}`

	// Reader has more types in union - backward compatible
	readerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": ["null", "string", "int"]}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible, got incompatible: %v", result.Messages)
	}
}

func TestChecker_UnionCompatibility_RemoveTypeFromUnion(t *testing.T) {
	checker := NewChecker()

	// Writer has more types
	writerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": ["null", "string", "int"]}
		]
	}`

	// Reader has fewer types - incompatible for backward
	readerSchema := `{
		"type": "record",
		"name": "Data",
		"fields": [
			{"name": "value", "type": ["null", "string"]}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if result.IsCompatible {
		t.Error("Expected incompatible (removed type from union), got compatible")
	}
}

func TestChecker_EnumCompatibility_AddSymbol(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "enum",
		"name": "Status",
		"symbols": ["ACTIVE", "INACTIVE"]
	}`

	// Reader has more symbols - backward compatible
	readerSchema := `{
		"type": "enum",
		"name": "Status",
		"symbols": ["ACTIVE", "INACTIVE", "PENDING"]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible, got incompatible: %v", result.Messages)
	}
}

func TestChecker_EnumCompatibility_RemoveSymbol(t *testing.T) {
	checker := NewChecker()

	// Writer has symbol that reader doesn't have
	writerSchema := `{
		"type": "enum",
		"name": "Status",
		"symbols": ["ACTIVE", "INACTIVE", "PENDING"]
	}`

	readerSchema := `{
		"type": "enum",
		"name": "Status",
		"symbols": ["ACTIVE", "INACTIVE"]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if result.IsCompatible {
		t.Error("Expected incompatible (removed enum symbol), got compatible")
	}
}

func TestChecker_ArrayCompatibility(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "array",
		"items": "string"
	}`

	readerSchema := `{
		"type": "array",
		"items": "string"
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible, got incompatible: %v", result.Messages)
	}
}

func TestChecker_MapCompatibility(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "map",
		"values": "int"
	}`

	readerSchema := `{
		"type": "map",
		"values": "long"
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible (int->long promotion in map values), got incompatible: %v", result.Messages)
	}
}

func TestChecker_NameMismatch(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "record",
		"name": "User",
		"fields": [{"name": "id", "type": "long"}]
	}`

	readerSchema := `{
		"type": "record",
		"name": "Person",
		"fields": [{"name": "id", "type": "long"}]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if result.IsCompatible {
		t.Error("Expected incompatible (name mismatch), got compatible")
	}
}

func TestChecker_NestedRecordCompatibility(t *testing.T) {
	checker := NewChecker()

	writerSchema := `{
		"type": "record",
		"name": "Order",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "user", "type": {
				"type": "record",
				"name": "User",
				"fields": [{"name": "id", "type": "long"}]
			}}
		]
	}`

	readerSchema := `{
		"type": "record",
		"name": "Order",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "user", "type": {
				"type": "record",
				"name": "User",
				"fields": [
					{"name": "id", "type": "long"},
					{"name": "name", "type": "string", "default": ""}
				]
			}}
		]
	}`

	result := checker.Check(s(readerSchema), s(writerSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible (nested record with optional field), got incompatible: %v", result.Messages)
	}
}

func TestChecker_PrimitiveTypes(t *testing.T) {
	checker := NewChecker()

	primitives := []string{
		`"null"`,
		`"boolean"`,
		`"int"`,
		`"long"`,
		`"float"`,
		`"double"`,
		`"bytes"`,
		`"string"`,
	}

	for _, p := range primitives {
		result := checker.Check(s(p), s(p))
		if !result.IsCompatible {
			t.Errorf("Expected primitive %s to be self-compatible, got incompatible: %v", p, result.Messages)
		}
	}
}
