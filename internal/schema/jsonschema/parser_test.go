package jsonschema

import (
	"strings"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func TestParser_Parse_SimpleObject(t *testing.T) {
	parser := NewParser()

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}

	if parsed.Type() != storage.SchemaTypeJSON {
		t.Errorf("Expected type %s, got %s", storage.SchemaTypeJSON, parsed.Type())
	}

	// Fingerprint should be non-empty
	fingerprint := parsed.Fingerprint()
	if fingerprint == "" {
		t.Error("Fingerprint should not be empty")
	}
}

func TestParser_Parse_NestedObject(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"address": {
						"type": "object",
						"properties": {
							"street": {"type": "string"},
							"city": {"type": "string"}
						}
					}
				}
			}
		}
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse nested schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_ArrayType(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "array",
		"items": {
			"type": "string"
		}
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse array schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_AllPrimitiveTypes(t *testing.T) {
	parser := NewParser()

	schemas := []struct {
		name   string
		schema string
	}{
		{"string", `{"type": "string"}`},
		{"integer", `{"type": "integer"}`},
		{"number", `{"type": "number"}`},
		{"boolean", `{"type": "boolean"}`},
		{"null", `{"type": "null"}`},
		{"array", `{"type": "array", "items": {"type": "string"}}`},
		{"object", `{"type": "object"}`},
	}

	for _, tt := range schemas {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.schema, nil)
			if err != nil {
				t.Fatalf("Failed to parse %s schema: %v", tt.name, err)
			}
			if parsed == nil {
				t.Fatalf("Parsed schema is nil for %s", tt.name)
			}
		})
	}
}

func TestParser_Parse_WithEnum(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "string",
		"enum": ["red", "green", "blue"]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse enum schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_WithPatternProperties(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "object",
		"patternProperties": {
			"^S_": {"type": "string"},
			"^I_": {"type": "integer"}
		}
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse patternProperties schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_WithAdditionalProperties(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name   string
		schema string
	}{
		{
			name:   "additionalProperties true",
			schema: `{"type": "object", "additionalProperties": true}`,
		},
		{
			name:   "additionalProperties false",
			schema: `{"type": "object", "additionalProperties": false}`,
		},
		{
			name:   "additionalProperties schema",
			schema: `{"type": "object", "additionalProperties": {"type": "string"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.schema, nil)
			if err != nil {
				t.Fatalf("Failed to parse schema: %v", err)
			}
			if parsed == nil {
				t.Fatal("Parsed schema is nil")
			}
		})
	}
}

func TestParser_Parse_WithConstraints(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"minLength": 1,
				"maxLength": 100
			},
			"age": {
				"type": "integer",
				"minimum": 0,
				"maximum": 150
			},
			"tags": {
				"type": "array",
				"minItems": 1,
				"maxItems": 10
			}
		}
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema with constraints: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_WithDefinitions(t *testing.T) {
	parser := NewParser()

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"definitions": {
			"address": {
				"type": "object",
				"properties": {
					"street": {"type": "string"},
					"city": {"type": "string"}
				}
			}
		},
		"type": "object",
		"properties": {
			"billing_address": {"$ref": "#/definitions/address"},
			"shipping_address": {"$ref": "#/definitions/address"}
		}
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse schema with definitions: %v", err)
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
			name:   "invalid JSON",
			schema: `{not valid json}`,
		},
		{
			name:   "missing closing brace",
			schema: `{"type": "object"`,
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

	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		}
	}`

	parsed1, _ := parser.Parse(schema, nil)
	parsed2, _ := parser.Parse(schema, nil)

	if parsed1.Fingerprint() != parsed2.Fingerprint() {
		t.Error("Fingerprints should be identical for same schema")
	}
}

func TestParser_Fingerprint_DifferentForDifferentSchemas(t *testing.T) {
	parser := NewParser()

	schema1 := `{"type": "object", "properties": {"name": {"type": "string"}}}`
	schema2 := `{"type": "object", "properties": {"name": {"type": "integer"}}}`

	parsed1, _ := parser.Parse(schema1, nil)
	parsed2, _ := parser.Parse(schema2, nil)

	if parsed1.Fingerprint() == parsed2.Fingerprint() {
		t.Error("Fingerprints should be different for different schemas")
	}
}

func TestParser_CanonicalString_KeysSorted(t *testing.T) {
	parser := NewParser()

	// Parse schema with keys in different order
	schema1 := `{"type": "object", "properties": {"a": {"type": "string"}, "b": {"type": "integer"}}}`
	schema2 := `{"properties": {"b": {"type": "integer"}, "a": {"type": "string"}}, "type": "object"}`

	parsed1, _ := parser.Parse(schema1, nil)
	parsed2, _ := parser.Parse(schema2, nil)

	if parsed1.CanonicalString() != parsed2.CanonicalString() {
		t.Error("Canonical strings should be identical regardless of key order")
	}
}

func TestParser_Type(t *testing.T) {
	parser := NewParser()

	if parser.Type() != storage.SchemaTypeJSON {
		t.Errorf("Expected type %s, got %s", storage.SchemaTypeJSON, parser.Type())
	}
}

func TestParser_Parse_Draft07Features(t *testing.T) {
	parser := NewParser()

	// Draft-07 specific features
	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"optional_field": {"type": "string"}
		},
		"required": ["name"],
		"if": {
			"properties": {"name": {"const": "special"}}
		},
		"then": {
			"required": ["optional_field"]
		}
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse Draft-07 schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_OneOf(t *testing.T) {
	parser := NewParser()

	schema := `{
		"oneOf": [
			{"type": "string"},
			{"type": "number"}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse oneOf schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_AnyOf(t *testing.T) {
	parser := NewParser()

	schema := `{
		"anyOf": [
			{"type": "string"},
			{"type": "null"}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse anyOf schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_AllOf(t *testing.T) {
	parser := NewParser()

	schema := `{
		"allOf": [
			{"type": "object", "properties": {"name": {"type": "string"}}},
			{"type": "object", "properties": {"age": {"type": "integer"}}}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse allOf schema: %v", err)
	}

	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestGetSchemaType(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected string
	}{
		{
			name:     "string type",
			schema:   map[string]interface{}{"type": "string"},
			expected: "string",
		},
		{
			name:     "array of types",
			schema:   map[string]interface{}{"type": []interface{}{"string", "null"}},
			expected: "string",
		},
		{
			name:     "no type",
			schema:   map[string]interface{}{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSchemaType(tt.schema)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetProperties(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
	}

	props := GetProperties(schema)
	if props == nil {
		t.Fatal("Properties should not be nil")
	}

	if _, ok := props["name"]; !ok {
		t.Error("Properties should contain 'name'")
	}
}

func TestGetRequired(t *testing.T) {
	schema := map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"name", "age"},
	}

	required := GetRequired(schema)
	if len(required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(required))
	}

	if required[0] != "name" || required[1] != "age" {
		t.Error("Required fields not as expected")
	}
}

func TestCanonicalizeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"null", nil, "null"},
		{"true", true, "true"},
		{"false", false, "false"},
		{"integer", float64(42), "42"},
		{"float", float64(3.14), "3.14"},
		{"string", "hello", `"hello"`},
		{"empty array", []interface{}{}, "[]"},
		{"string array", []interface{}{"a", "b"}, `["a","b"]`},
		{"empty object", map[string]interface{}{}, "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := canonicalizeValue(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCanonicalizeValue_ObjectKeysSorted(t *testing.T) {
	input := map[string]interface{}{
		"z": "last",
		"a": "first",
		"m": "middle",
	}

	result, _ := canonicalizeValue(input)

	// Keys should be sorted alphabetically
	if !strings.HasPrefix(result, `{"a":`) {
		t.Errorf("Object should start with 'a' key: %s", result)
	}
}
