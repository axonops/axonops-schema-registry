package avro

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func TestParser_ParsePrimitiveTypes(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name   string
		schema string
	}{
		{"null", `"null"`},
		{"boolean", `"boolean"`},
		{"int", `"int"`},
		{"long", `"long"`},
		{"float", `"float"`},
		{"double", `"double"`},
		{"bytes", `"bytes"`},
		{"string", `"string"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.schema, nil)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if parsed.Type() != storage.SchemaTypeAvro {
				t.Errorf("Expected AVRO type, got %s", parsed.Type())
			}
			if parsed.Fingerprint() == "" {
				t.Error("Expected non-empty fingerprint")
			}
		})
	}
}

func TestParser_ParseRecord(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "record",
		"name": "User",
		"namespace": "com.example",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "name", "type": "string"},
			{"name": "email", "type": ["null", "string"], "default": null}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Type() != storage.SchemaTypeAvro {
		t.Errorf("Expected AVRO type, got %s", parsed.Type())
	}

	canonical := parsed.CanonicalString()
	if canonical == "" {
		t.Error("Expected non-empty canonical form")
	}

	fingerprint := parsed.Fingerprint()
	if fingerprint == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseEnum(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "enum",
		"name": "Status",
		"symbols": ["ACTIVE", "INACTIVE", "PENDING"]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseArray(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "array",
		"items": "string"
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseMap(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "map",
		"values": "int"
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseUnion(t *testing.T) {
	parser := NewParser()

	schema := `["null", "string"]`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseFixed(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "fixed",
		"name": "md5",
		"size": 16
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_InvalidSchema(t *testing.T) {
	parser := NewParser()

	invalidSchemas := []string{
		`{"type": "invalid"}`,
		`{invalid json`,
		`{"type": "record"}`, // missing name and fields
	}

	for _, schema := range invalidSchemas {
		_, err := parser.Parse(schema, nil)
		if err == nil {
			t.Errorf("Expected error for invalid schema: %s", schema)
		}
	}
}

func TestParser_SameFingerprintForEquivalentSchemas(t *testing.T) {
	parser := NewParser()

	// Same schema with different whitespace
	schema1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	schema2 := `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "id", "type": "long"}
		]
	}`

	parsed1, err := parser.Parse(schema1, nil)
	if err != nil {
		t.Fatalf("Parse schema1 failed: %v", err)
	}

	parsed2, err := parser.Parse(schema2, nil)
	if err != nil {
		t.Fatalf("Parse schema2 failed: %v", err)
	}

	if parsed1.Fingerprint() != parsed2.Fingerprint() {
		t.Errorf("Expected same fingerprint for equivalent schemas")
	}
}
