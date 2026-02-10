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

func TestParser_ParseDeeplyNestedRecords(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "record",
		"name": "Level1",
		"fields": [
			{"name": "l2", "type": {
				"type": "record",
				"name": "Level2",
				"fields": [
					{"name": "l3", "type": {
						"type": "record",
						"name": "Level3",
						"fields": [
							{"name": "l4", "type": {
								"type": "record",
								"name": "Level4",
								"fields": [
									{"name": "value", "type": "string"}
								]
							}}
						]
					}}
				]
			}}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed for deeply nested records: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
	if parsed.CanonicalString() == "" {
		t.Error("Expected non-empty canonical string")
	}
}

func TestParser_ParseLogicalTypes(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name   string
		schema string
	}{
		{
			"date",
			`{"type": "record", "name": "WithDate", "fields": [
				{"name": "d", "type": {"type": "int", "logicalType": "date"}}
			]}`,
		},
		{
			"timestamp-millis",
			`{"type": "record", "name": "WithTSMillis", "fields": [
				{"name": "ts", "type": {"type": "long", "logicalType": "timestamp-millis"}}
			]}`,
		},
		{
			"timestamp-micros",
			`{"type": "record", "name": "WithTSMicros", "fields": [
				{"name": "ts", "type": {"type": "long", "logicalType": "timestamp-micros"}}
			]}`,
		},
		{
			"decimal",
			`{"type": "record", "name": "WithDecimal", "fields": [
				{"name": "price", "type": {"type": "bytes", "logicalType": "decimal", "precision": 10, "scale": 2}}
			]}`,
		},
		{
			"uuid",
			`{"type": "record", "name": "WithUUID", "fields": [
				{"name": "id", "type": {"type": "string", "logicalType": "uuid"}}
			]}`,
		},
		{
			"time-millis",
			`{"type": "record", "name": "WithTime", "fields": [
				{"name": "t", "type": {"type": "int", "logicalType": "time-millis"}}
			]}`,
		},
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

func TestParser_ParseRecursiveType(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "record",
		"name": "TreeNode",
		"fields": [
			{"name": "value", "type": "string"},
			{"name": "children", "type": {"type": "array", "items": "TreeNode"}}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed for recursive type: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseRecordWithDefaults(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "record",
		"name": "WithDefaults",
		"fields": [
			{"name": "name", "type": "string", "default": "unknown"},
			{"name": "count", "type": "int", "default": 0},
			{"name": "active", "type": "boolean", "default": true},
			{"name": "score", "type": "double", "default": 0.0},
			{"name": "tags", "type": {"type": "array", "items": "string"}, "default": []},
			{"name": "email", "type": ["null", "string"], "default": null}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed for record with defaults: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseComplexRealWorldSchema(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "record",
		"name": "PaymentEvent",
		"namespace": "com.example.payments",
		"fields": [
			{"name": "event_id", "type": {"type": "string", "logicalType": "uuid"}},
			{"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
			{"name": "amount", "type": {"type": "bytes", "logicalType": "decimal", "precision": 12, "scale": 2}},
			{"name": "currency", "type": {"type": "enum", "name": "Currency", "symbols": ["USD", "EUR", "GBP", "JPY"]}},
			{"name": "status", "type": {"type": "enum", "name": "PaymentStatus", "symbols": ["PENDING", "COMPLETED", "FAILED", "REFUNDED"]}},
			{"name": "customer", "type": {
				"type": "record",
				"name": "Customer",
				"fields": [
					{"name": "id", "type": "long"},
					{"name": "name", "type": "string"},
					{"name": "email", "type": ["null", "string"], "default": null},
					{"name": "address", "type": {
						"type": "record",
						"name": "Address",
						"fields": [
							{"name": "street", "type": "string"},
							{"name": "city", "type": "string"},
							{"name": "country", "type": "string"},
							{"name": "zip", "type": ["null", "string"], "default": null}
						]
					}}
				]
			}},
			{"name": "items", "type": {"type": "array", "items": {
				"type": "record",
				"name": "LineItem",
				"fields": [
					{"name": "product_id", "type": "string"},
					{"name": "quantity", "type": "int"},
					{"name": "unit_price", "type": {"type": "bytes", "logicalType": "decimal", "precision": 10, "scale": 2}}
				]
			}}},
			{"name": "metadata", "type": {"type": "map", "values": "string"}},
			{"name": "notes", "type": ["null", "string"], "default": null}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed for complex real-world schema: %v", err)
	}
	if parsed.Type() != storage.SchemaTypeAvro {
		t.Errorf("Expected AVRO type, got %s", parsed.Type())
	}
	if parsed.CanonicalString() == "" {
		t.Error("Expected non-empty canonical string")
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseNamespacedRecords(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "record",
		"name": "Order",
		"namespace": "com.example.orders",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "customer", "type": {
				"type": "record",
				"name": "Customer",
				"namespace": "com.example.customers",
				"fields": [
					{"name": "id", "type": "long"},
					{"name": "name", "type": "string"}
				]
			}}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed for namespaced records: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseComplexCollections(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name   string
		schema string
	}{
		{
			"map of arrays",
			`{"type": "record", "name": "R1", "fields": [
				{"name": "groups", "type": {"type": "map", "values": {"type": "array", "items": "string"}}}
			]}`,
		},
		{
			"array of maps",
			`{"type": "record", "name": "R2", "fields": [
				{"name": "entries", "type": {"type": "array", "items": {"type": "map", "values": "int"}}}
			]}`,
		},
		{
			"array of records",
			`{"type": "record", "name": "R3", "fields": [
				{"name": "items", "type": {"type": "array", "items": {
					"type": "record", "name": "Item", "fields": [
						{"name": "name", "type": "string"},
						{"name": "value", "type": "int"}
					]
				}}}
			]}`,
		},
		{
			"map with record values",
			`{"type": "record", "name": "R4", "fields": [
				{"name": "configs", "type": {"type": "map", "values": {
					"type": "record", "name": "Config", "fields": [
						{"name": "key", "type": "string"},
						{"name": "enabled", "type": "boolean"}
					]
				}}}
			]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.schema, nil)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if parsed.Fingerprint() == "" {
				t.Error("Expected non-empty fingerprint")
			}
		})
	}
}

func TestParser_ParseComplexUnion(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "record",
		"name": "Event",
		"fields": [
			{"name": "payload", "type": ["null", "string", "int", "double", "boolean"]}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse failed for complex union: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
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
