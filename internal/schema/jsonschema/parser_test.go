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

func TestParser_Parse_CrossRefWithinDefinitions(t *testing.T) {
	parser := NewParser()

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"definitions": {
			"street_address": {
				"type": "object",
				"properties": {
					"street": {"type": "string"},
					"city": {"type": "string"},
					"country": {"type": "string"}
				},
				"required": ["street", "city"]
			},
			"person": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"home_address": {"$ref": "#/definitions/street_address"},
					"work_address": {"$ref": "#/definitions/street_address"}
				},
				"required": ["name"]
			}
		},
		"type": "object",
		"properties": {
			"primary_contact": {"$ref": "#/definitions/person"},
			"backup_contact": {"$ref": "#/definitions/person"}
		}
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse cross-ref schema: %v", err)
	}
	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_ComplexRealWorldPaymentEvent(t *testing.T) {
	parser := NewParser()

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"title": "PaymentEvent",
		"definitions": {
			"address": {
				"type": "object",
				"properties": {
					"street": {"type": "string"},
					"city": {"type": "string"},
					"country": {"type": "string"},
					"zip": {"type": "string", "pattern": "^[0-9]{5}(-[0-9]{4})?$"}
				},
				"required": ["street", "city", "country"]
			},
			"customer": {
				"type": "object",
				"properties": {
					"id": {"type": "integer"},
					"name": {"type": "string", "minLength": 1, "maxLength": 200},
					"email": {"type": "string", "format": "email"},
					"address": {"$ref": "#/definitions/address"}
				},
				"required": ["id", "name"]
			},
			"line_item": {
				"type": "object",
				"properties": {
					"product_id": {"type": "string"},
					"quantity": {"type": "integer", "minimum": 1},
					"unit_price": {"type": "number", "exclusiveMinimum": 0}
				},
				"required": ["product_id", "quantity", "unit_price"]
			}
		},
		"properties": {
			"event_id": {"type": "string", "format": "uuid"},
			"timestamp": {"type": "string", "format": "date-time"},
			"amount": {"type": "number", "minimum": 0},
			"currency": {"type": "string", "enum": ["USD", "EUR", "GBP", "JPY"]},
			"status": {"type": "string", "enum": ["PENDING", "COMPLETED", "FAILED", "REFUNDED"]},
			"customer": {"$ref": "#/definitions/customer"},
			"items": {
				"type": "array",
				"items": {"$ref": "#/definitions/line_item"},
				"minItems": 1
			},
			"metadata": {
				"type": "object",
				"additionalProperties": {"type": "string"}
			},
			"notes": {"type": ["string", "null"]}
		},
		"required": ["event_id", "timestamp", "amount", "currency", "status", "customer", "items"]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse PaymentEvent schema: %v", err)
	}
	if parsed.Type() != storage.SchemaTypeJSON {
		t.Errorf("Expected JSON type, got %s", parsed.Type())
	}
	if parsed.CanonicalString() == "" {
		t.Error("Expected non-empty canonical string")
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_ComplexComposition(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name   string
		schema string
	}{
		{
			"oneOf with multiple object schemas",
			`{
				"oneOf": [
					{"type": "object", "properties": {"type": {"const": "email"}, "address": {"type": "string"}}, "required": ["type", "address"]},
					{"type": "object", "properties": {"type": {"const": "phone"}, "number": {"type": "string"}}, "required": ["type", "number"]},
					{"type": "object", "properties": {"type": {"const": "sms"}, "number": {"type": "string"}, "carrier": {"type": "string"}}, "required": ["type", "number"]}
				]
			}`,
		},
		{
			"allOf combining constraints",
			`{
				"allOf": [
					{"type": "object", "properties": {"name": {"type": "string"}}},
					{"type": "object", "properties": {"age": {"type": "integer", "minimum": 0, "maximum": 150}}},
					{"required": ["name", "age"]}
				]
			}`,
		},
		{
			"nested anyOf within oneOf",
			`{
				"oneOf": [
					{"type": "string"},
					{"anyOf": [{"type": "integer"}, {"type": "number"}]}
				]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.schema, nil)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			if parsed == nil {
				t.Fatal("Parsed schema is nil")
			}
			if parsed.Fingerprint() == "" {
				t.Error("Expected non-empty fingerprint")
			}
		})
	}
}

func TestParser_Parse_DeeplyNestedObjects(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "object",
		"properties": {
			"l1": {
				"type": "object",
				"properties": {
					"l2": {
						"type": "object",
						"properties": {
							"l3": {
								"type": "object",
								"properties": {
									"l4": {
										"type": "object",
										"properties": {
											"value": {"type": "string"}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse deeply nested schema: %v", err)
	}
	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_ConditionalIfThenElse(t *testing.T) {
	parser := NewParser()

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"type": {"type": "string", "enum": ["residential", "business"]},
			"address": {"type": "string"}
		},
		"required": ["type"],
		"if": {
			"properties": {"type": {"const": "business"}}
		},
		"then": {
			"properties": {
				"company_name": {"type": "string"},
				"tax_id": {"type": "string"}
			},
			"required": ["company_name", "tax_id"]
		},
		"else": {
			"properties": {
				"resident_name": {"type": "string"}
			},
			"required": ["resident_name"]
		}
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Failed to parse if/then/else schema: %v", err)
	}
	if parsed == nil {
		t.Fatal("Parsed schema is nil")
	}
}

func TestParser_Parse_StandaloneNonObjectTypes(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name   string
		schema string
	}{
		{
			"standalone string with constraints",
			`{"type": "string", "minLength": 1, "maxLength": 255, "pattern": "^[a-zA-Z]+$"}`,
		},
		{
			"standalone array of numbers",
			`{"type": "array", "items": {"type": "number"}, "minItems": 1, "maxItems": 100, "uniqueItems": true}`,
		},
		{
			"standalone integer with constraints",
			`{"type": "integer", "minimum": 0, "maximum": 1000, "multipleOf": 5}`,
		},
		{
			"type union",
			`{"type": ["string", "null"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.schema, nil)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			if parsed == nil {
				t.Fatal("Parsed schema is nil")
			}
			if parsed.Fingerprint() == "" {
				t.Error("Expected non-empty fingerprint")
			}
		})
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

// --- Schema Reference Tests ---

func TestParser_Parse_WithEmptyReferences(t *testing.T) {
	parser := NewParser()

	schema := `{"type": "object", "properties": {"id": {"type": "integer"}}}`

	parsed, err := parser.Parse(schema, []storage.Reference{})
	if err != nil {
		t.Fatalf("Parse with empty references should not fail: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_CrossSubjectRef(t *testing.T) {
	parser := NewParser()

	// External schema for "address.json"
	addressSchema := `{
		"type": "object",
		"properties": {
			"street": {"type": "string"},
			"city": {"type": "string"},
			"zip": {"type": "string"}
		},
		"required": ["street", "city"]
	}`

	// Main schema references the external address schema
	orderSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"id": {"type": "integer"},
			"shipping_address": {"$ref": "address.json"}
		},
		"required": ["id"]
	}`

	refs := []storage.Reference{
		{Name: "address.json", Subject: "address-value", Version: 1, Schema: addressSchema},
	}

	parsed, err := parser.Parse(orderSchema, refs)
	if err != nil {
		t.Fatalf("Parse with cross-subject $ref failed: %v", err)
	}
	if parsed.Type() != storage.SchemaTypeJSON {
		t.Errorf("Expected JSON type, got %s", parsed.Type())
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_CrossSubjectRef_MultipleRefs(t *testing.T) {
	parser := NewParser()

	addressSchema := `{
		"type": "object",
		"properties": {
			"street": {"type": "string"},
			"city": {"type": "string"}
		}
	}`

	customerSchema := `{
		"type": "object",
		"properties": {
			"id": {"type": "integer"},
			"name": {"type": "string"}
		}
	}`

	orderSchema := `{
		"type": "object",
		"properties": {
			"customer": {"$ref": "customer.json"},
			"shipping": {"$ref": "address.json"}
		}
	}`

	refs := []storage.Reference{
		{Name: "address.json", Subject: "address-value", Version: 1, Schema: addressSchema},
		{Name: "customer.json", Subject: "customer-value", Version: 1, Schema: customerSchema},
	}

	parsed, err := parser.Parse(orderSchema, refs)
	if err != nil {
		t.Fatalf("Parse with multiple cross-subject refs failed: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_InternalRefWithExternalReferences(t *testing.T) {
	parser := NewParser()

	addressSchema := `{
		"type": "object",
		"properties": {
			"street": {"type": "string"},
			"city": {"type": "string"}
		}
	}`

	// Schema uses both internal definitions and external references
	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"definitions": {
			"phone": {
				"type": "object",
				"properties": {
					"number": {"type": "string"}
				}
			}
		},
		"type": "object",
		"properties": {
			"phone": {"$ref": "#/definitions/phone"},
			"address": {"$ref": "address.json"}
		}
	}`

	refs := []storage.Reference{
		{Name: "address.json", Subject: "address-value", Version: 1, Schema: addressSchema},
	}

	parsed, err := parser.Parse(schema, refs)
	if err != nil {
		t.Fatalf("Parse with internal $ref and external references failed: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_Parse_ReferencesStoredOnParsedSchema(t *testing.T) {
	parser := NewParser()

	schema := `{"type": "object", "properties": {"id": {"type": "integer"}}}`

	refs := []storage.Reference{
		{Name: "TypeA", Subject: "subject-a", Version: 1, Schema: `{"type": "string"}`},
		{Name: "TypeB", Subject: "subject-b", Version: 3, Schema: `{"type": "integer"}`},
	}

	parsed, err := parser.Parse(schema, refs)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	jsonParsed, ok := parsed.(*ParsedJSONSchema)
	if !ok {
		t.Fatal("Expected *ParsedJSONSchema")
	}

	if len(jsonParsed.references) != 2 {
		t.Errorf("Expected 2 stored references, got %d", len(jsonParsed.references))
	}
}

func TestParser_Parse_UnusedReferencesGraceful(t *testing.T) {
	parser := NewParser()

	schema := `{"type": "object", "properties": {"id": {"type": "integer"}}}`

	refs := []storage.Reference{
		{Name: "unused.json", Subject: "unused-value", Version: 1, Schema: `{"type": "string"}`},
	}

	parsed, err := parser.Parse(schema, refs)
	if err != nil {
		t.Fatalf("Parse with unused references should not fail: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}
