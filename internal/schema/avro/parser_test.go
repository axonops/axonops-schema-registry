package avro

import (
	"strings"
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

// --- Schema Reference Tests ---

func TestParser_ParseWithEmptyReferences(t *testing.T) {
	parser := NewParser()

	schema := `{
		"type": "record",
		"name": "Simple",
		"fields": [{"name": "id", "type": "long"}]
	}`

	// Empty slice should behave like nil
	parsed, err := parser.Parse(schema, []storage.Reference{})
	if err != nil {
		t.Fatalf("Parse with empty references should not fail: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseInlineNamedTypeReference(t *testing.T) {
	parser := NewParser()

	// Avro supports inline named type references within the same schema.
	schema := `{
		"type": "record",
		"name": "Order",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "billing", "type": {
				"type": "record",
				"name": "Address",
				"fields": [
					{"name": "street", "type": "string"},
					{"name": "city", "type": "string"}
				]
			}},
			{"name": "shipping", "type": "Address"}
		]
	}`

	parsed, err := parser.Parse(schema, nil)
	if err != nil {
		t.Fatalf("Parse with inline named type reference failed: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseCrossSubjectReference(t *testing.T) {
	parser := NewParser()

	// The "Address" type is defined in a referenced schema (from another subject).
	// The main schema uses "Address" as a field type.
	addressSchema := `{
		"type": "record",
		"name": "Address",
		"namespace": "com.example",
		"fields": [
			{"name": "street", "type": "string"},
			{"name": "city", "type": "string"},
			{"name": "zip", "type": "string"}
		]
	}`

	orderSchema := `{
		"type": "record",
		"name": "Order",
		"namespace": "com.example",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "shipping_address", "type": "com.example.Address"}
		]
	}`

	refs := []storage.Reference{
		{Name: "com.example.Address", Subject: "address-value", Version: 1, Schema: addressSchema},
	}

	parsed, err := parser.Parse(orderSchema, refs)
	if err != nil {
		t.Fatalf("Parse with cross-subject reference failed: %v", err)
	}
	if parsed.Type() != storage.SchemaTypeAvro {
		t.Errorf("Expected AVRO type, got %s", parsed.Type())
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseCrossSubjectReference_MultipleRefs(t *testing.T) {
	parser := NewParser()

	addressSchema := `{
		"type": "record",
		"name": "Address",
		"fields": [
			{"name": "street", "type": "string"},
			{"name": "city", "type": "string"}
		]
	}`

	customerSchema := `{
		"type": "record",
		"name": "Customer",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "name", "type": "string"}
		]
	}`

	orderSchema := `{
		"type": "record",
		"name": "Order",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "customer", "type": "Customer"},
			{"name": "shipping", "type": "Address"}
		]
	}`

	refs := []storage.Reference{
		{Name: "Address", Subject: "address-value", Version: 1, Schema: addressSchema},
		{Name: "Customer", Subject: "customer-value", Version: 1, Schema: customerSchema},
	}

	parsed, err := parser.Parse(orderSchema, refs)
	if err != nil {
		t.Fatalf("Parse with multiple cross-subject references failed: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

func TestParser_ParseCrossSubjectReference_FailsWithoutContent(t *testing.T) {
	parser := NewParser()

	// Schema references "Address" but the reference has no content
	orderSchema := `{
		"type": "record",
		"name": "Order",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "shipping", "type": "Address"}
		]
	}`

	// Reference without content — should fail since Address is unknown
	refs := []storage.Reference{
		{Name: "Address", Subject: "address-value", Version: 1},
	}

	_, err := parser.Parse(orderSchema, refs)
	if err == nil {
		t.Error("Expected error when reference content is missing")
	}
}

func TestParser_ParseUnusedReferencesGraceful(t *testing.T) {
	parser := NewParser()

	// Schema doesn't actually use the referenced type
	schema := `{
		"type": "record",
		"name": "Simple",
		"fields": [{"name": "id", "type": "long"}]
	}`

	addressSchema := `{
		"type": "record",
		"name": "Address",
		"fields": [
			{"name": "street", "type": "string"}
		]
	}`

	refs := []storage.Reference{
		{Name: "Address", Subject: "address-value", Version: 1, Schema: addressSchema},
	}

	parsed, err := parser.Parse(schema, refs)
	if err != nil {
		t.Fatalf("Parse with unused references should not fail: %v", err)
	}
	if parsed.Fingerprint() == "" {
		t.Error("Expected non-empty fingerprint")
	}
}

// --- Avro Namespace Inheritance Tests ---
// Per the Avro specification, a nested named type (record, enum, fixed) without
// an explicit namespace inherits the namespace from its most tightly enclosing
// named type. These tests verify that the canonical form correctly resolves
// inherited namespaces, producing fully-qualified names in all cases.

func TestCanonicalize_NestedRecordInheritsNamespace(t *testing.T) {
	// Inner has no explicit namespace → should inherit "com.example" from Outer.
	schema := `{
		"type": "record",
		"name": "Outer",
		"namespace": "com.example",
		"fields": [
			{"name": "inner", "type": {
				"type": "record",
				"name": "Inner",
				"fields": [
					{"name": "value", "type": "string"}
				]
			}}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"com.example.Inner"`) {
		t.Errorf("Inner should inherit namespace com.example, got: %s", canonical)
	}
	if !strings.Contains(canonical, `"name":"com.example.Outer"`) {
		t.Errorf("Outer should be fully qualified, got: %s", canonical)
	}
}

func TestCanonicalize_NestedEnumInheritsNamespace(t *testing.T) {
	// Enum Status has no explicit namespace → should inherit "com.example" from Outer.
	schema := `{
		"type": "record",
		"name": "Outer",
		"namespace": "com.example",
		"fields": [
			{"name": "status", "type": {
				"type": "enum",
				"name": "Status",
				"symbols": ["ACTIVE", "INACTIVE"]
			}}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"com.example.Status"`) {
		t.Errorf("Enum Status should inherit namespace com.example, got: %s", canonical)
	}
}

func TestCanonicalize_NestedFixedInheritsNamespace(t *testing.T) {
	// Fixed Hash has no explicit namespace → should inherit "com.example" from Outer.
	schema := `{
		"type": "record",
		"name": "Outer",
		"namespace": "com.example",
		"fields": [
			{"name": "hash", "type": {
				"type": "fixed",
				"name": "Hash",
				"size": 16
			}}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"com.example.Hash"`) {
		t.Errorf("Fixed Hash should inherit namespace com.example, got: %s", canonical)
	}
}

func TestCanonicalize_ThreeLevelDeepInheritance(t *testing.T) {
	// L2 and L3 have no explicit namespace → both should inherit "com.example" from L1.
	schema := `{
		"type": "record",
		"name": "L1",
		"namespace": "com.example",
		"fields": [
			{"name": "l2", "type": {
				"type": "record",
				"name": "L2",
				"fields": [
					{"name": "l3", "type": {
						"type": "record",
						"name": "L3",
						"fields": [
							{"name": "value", "type": "string"}
						]
					}}
				]
			}}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"com.example.L1"`) {
		t.Errorf("L1 should be fully qualified, got: %s", canonical)
	}
	if !strings.Contains(canonical, `"name":"com.example.L2"`) {
		t.Errorf("L2 should inherit namespace com.example, got: %s", canonical)
	}
	if !strings.Contains(canonical, `"name":"com.example.L3"`) {
		t.Errorf("L3 should inherit namespace com.example, got: %s", canonical)
	}
}

func TestCanonicalize_NestedTypeOverridesNamespace(t *testing.T) {
	// Inner has its own namespace "com.other" → should NOT inherit from Outer's "com.example".
	schema := `{
		"type": "record",
		"name": "Outer",
		"namespace": "com.example",
		"fields": [
			{"name": "inner", "type": {
				"type": "record",
				"name": "Inner",
				"namespace": "com.other",
				"fields": [
					{"name": "value", "type": "string"}
				]
			}}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"com.example.Outer"`) {
		t.Errorf("Outer should be com.example.Outer, got: %s", canonical)
	}
	if !strings.Contains(canonical, `"name":"com.other.Inner"`) {
		t.Errorf("Inner should use its own namespace com.other, got: %s", canonical)
	}
}

func TestCanonicalize_OverriddenNamespacePropagates(t *testing.T) {
	// Middle overrides namespace to "com.middle". Leaf has no namespace → should inherit "com.middle".
	schema := `{
		"type": "record",
		"name": "Top",
		"namespace": "com.top",
		"fields": [
			{"name": "mid", "type": {
				"type": "record",
				"name": "Middle",
				"namespace": "com.middle",
				"fields": [
					{"name": "leaf", "type": {
						"type": "record",
						"name": "Leaf",
						"fields": [
							{"name": "data", "type": "string"}
						]
					}}
				]
			}}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"com.top.Top"`) {
		t.Errorf("Top should be com.top.Top, got: %s", canonical)
	}
	if !strings.Contains(canonical, `"name":"com.middle.Middle"`) {
		t.Errorf("Middle should be com.middle.Middle, got: %s", canonical)
	}
	if !strings.Contains(canonical, `"name":"com.middle.Leaf"`) {
		t.Errorf("Leaf should inherit com.middle from Middle, got: %s", canonical)
	}
}

func TestCanonicalize_AlreadyQualifiedNameNotDoubled(t *testing.T) {
	// Inner already has a fully-qualified name with dot → should not be re-qualified.
	schema := `{
		"type": "record",
		"name": "Outer",
		"namespace": "com.example",
		"fields": [
			{"name": "inner", "type": {
				"type": "record",
				"name": "org.other.Inner",
				"fields": [
					{"name": "value", "type": "string"}
				]
			}}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"org.other.Inner"`) {
		t.Errorf("Already-qualified name should not be modified, got: %s", canonical)
	}
}

func TestCanonicalize_NoNamespaceAtAll(t *testing.T) {
	// No namespace anywhere → names should remain unqualified (short).
	schema := `{
		"type": "record",
		"name": "Outer",
		"fields": [
			{"name": "inner", "type": {
				"type": "record",
				"name": "Inner",
				"fields": [
					{"name": "value", "type": "string"}
				]
			}}
		]
	}`

	canonical := canonicalize(schema)
	if strings.Contains(canonical, `"name":"."`) {
		t.Errorf("Should not produce empty namespace prefix, got: %s", canonical)
	}
	// Names should just be "Outer" and "Inner" with no dots
	if !strings.Contains(canonical, `"name":"Outer"`) {
		t.Errorf("Outer should be unqualified, got: %s", canonical)
	}
	if !strings.Contains(canonical, `"name":"Inner"`) {
		t.Errorf("Inner should be unqualified, got: %s", canonical)
	}
}

func TestCanonicalize_InheritanceThroughArrayItems(t *testing.T) {
	// Named type inside an array items inherits namespace from enclosing record.
	schema := `{
		"type": "record",
		"name": "Order",
		"namespace": "com.shop",
		"fields": [
			{"name": "items", "type": {
				"type": "array",
				"items": {
					"type": "record",
					"name": "LineItem",
					"fields": [
						{"name": "product", "type": "string"},
						{"name": "qty", "type": "int"}
					]
				}
			}}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"com.shop.Order"`) {
		t.Errorf("Order should be fully qualified, got: %s", canonical)
	}
	if !strings.Contains(canonical, `"name":"com.shop.LineItem"`) {
		t.Errorf("LineItem in array items should inherit com.shop, got: %s", canonical)
	}
}

func TestCanonicalize_InheritanceThroughMapValues(t *testing.T) {
	// Named type inside map values inherits namespace from enclosing record.
	schema := `{
		"type": "record",
		"name": "Registry",
		"namespace": "com.example",
		"fields": [
			{"name": "configs", "type": {
				"type": "map",
				"values": {
					"type": "record",
					"name": "Config",
					"fields": [
						{"name": "key", "type": "string"},
						{"name": "enabled", "type": "boolean"}
					]
				}
			}}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"com.example.Config"`) {
		t.Errorf("Config in map values should inherit com.example, got: %s", canonical)
	}
}

func TestCanonicalize_InheritanceThroughUnion(t *testing.T) {
	// Named type inside a union inherits namespace from enclosing record.
	schema := `{
		"type": "record",
		"name": "Event",
		"namespace": "com.events",
		"fields": [
			{"name": "payload", "type": ["null", {
				"type": "record",
				"name": "Detail",
				"fields": [
					{"name": "info", "type": "string"}
				]
			}]}
		]
	}`

	canonical := canonicalize(schema)
	if !strings.Contains(canonical, `"name":"com.events.Detail"`) {
		t.Errorf("Detail in union should inherit com.events, got: %s", canonical)
	}
}

func TestFingerprint_ExplicitVsInheritedNamespacesMatch(t *testing.T) {
	parser := NewParser()

	// Schema 1: Inner has explicit namespace matching parent
	schemaExplicit := `{
		"type": "record",
		"name": "Outer",
		"namespace": "com.example",
		"fields": [
			{"name": "inner", "type": {
				"type": "record",
				"name": "Inner",
				"namespace": "com.example",
				"fields": [
					{"name": "value", "type": "string"}
				]
			}}
		]
	}`

	// Schema 2: Inner has no namespace (inherits com.example from parent)
	schemaInherited := `{
		"type": "record",
		"name": "Outer",
		"namespace": "com.example",
		"fields": [
			{"name": "inner", "type": {
				"type": "record",
				"name": "Inner",
				"fields": [
					{"name": "value", "type": "string"}
				]
			}}
		]
	}`

	parsed1, err := parser.Parse(schemaExplicit, nil)
	if err != nil {
		t.Fatalf("Parse explicit failed: %v", err)
	}
	parsed2, err := parser.Parse(schemaInherited, nil)
	if err != nil {
		t.Fatalf("Parse inherited failed: %v", err)
	}

	if parsed1.Fingerprint() != parsed2.Fingerprint() {
		t.Errorf("Explicit and inherited namespaces should produce same fingerprint.\n  Explicit:  %s\n  Inherited: %s",
			parsed1.CanonicalString(), parsed2.CanonicalString())
	}
}

func TestFingerprint_DifferentInheritedNamespacesAreDifferent(t *testing.T) {
	parser := NewParser()

	// Schema 1: Inner inherits "com.alpha" from parent
	schema1 := `{
		"type": "record",
		"name": "Outer",
		"namespace": "com.alpha",
		"fields": [
			{"name": "inner", "type": {
				"type": "record",
				"name": "Inner",
				"fields": [{"name": "v", "type": "string"}]
			}}
		]
	}`

	// Schema 2: Inner inherits "com.beta" from parent
	schema2 := `{
		"type": "record",
		"name": "Outer",
		"namespace": "com.beta",
		"fields": [
			{"name": "inner", "type": {
				"type": "record",
				"name": "Inner",
				"fields": [{"name": "v", "type": "string"}]
			}}
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

	if parsed1.Fingerprint() == parsed2.Fingerprint() {
		t.Errorf("Different inherited namespaces should produce different fingerprints.\n  Schema1: %s\n  Schema2: %s",
			parsed1.CanonicalString(), parsed2.CanonicalString())
	}
}

func TestCanonicalize_RealWorldPaymentEventInheritance(t *testing.T) {
	// Real-world schema: PaymentEvent has namespace "com.example.payments".
	// Nested types Currency (enum), PaymentStatus (enum), Customer (record),
	// Address (record), LineItem (record) all lack explicit namespaces and
	// must inherit from their enclosing types.
	schema := `{
		"type": "record",
		"name": "PaymentEvent",
		"namespace": "com.example.payments",
		"fields": [
			{"name": "event_id", "type": "string"},
			{"name": "currency", "type": {"type": "enum", "name": "Currency", "symbols": ["USD", "EUR", "GBP"]}},
			{"name": "status", "type": {"type": "enum", "name": "PaymentStatus", "symbols": ["PENDING", "COMPLETED"]}},
			{"name": "customer", "type": {
				"type": "record",
				"name": "Customer",
				"fields": [
					{"name": "id", "type": "long"},
					{"name": "name", "type": "string"},
					{"name": "address", "type": {
						"type": "record",
						"name": "Address",
						"fields": [
							{"name": "street", "type": "string"},
							{"name": "city", "type": "string"}
						]
					}}
				]
			}},
			{"name": "items", "type": {"type": "array", "items": {
				"type": "record",
				"name": "LineItem",
				"fields": [
					{"name": "product_id", "type": "string"},
					{"name": "quantity", "type": "int"}
				]
			}}}
		]
	}`

	canonical := canonicalize(schema)

	// All named types should be qualified with com.example.payments
	expectedNames := []string{
		`"name":"com.example.payments.PaymentEvent"`,
		`"name":"com.example.payments.Currency"`,
		`"name":"com.example.payments.PaymentStatus"`,
		`"name":"com.example.payments.Customer"`,
		`"name":"com.example.payments.Address"`,
		`"name":"com.example.payments.LineItem"`,
	}
	for _, expected := range expectedNames {
		if !strings.Contains(canonical, expected) {
			t.Errorf("Expected %s in canonical form, got: %s", expected, canonical)
		}
	}
}

func TestCanonicalize_MixedExplicitAndInheritedNamespaces(t *testing.T) {
	// Top-level: com.example
	// Customer: explicit com.customers (overrides)
	// Preference: no namespace → inherits com.customers from Customer
	// LineItem: no namespace → inherits com.example from Order (top-level)
	schema := `{
		"type": "record",
		"name": "Order",
		"namespace": "com.example",
		"fields": [
			{"name": "customer", "type": {
				"type": "record",
				"name": "Customer",
				"namespace": "com.customers",
				"fields": [
					{"name": "id", "type": "long"},
					{"name": "pref", "type": {
						"type": "record",
						"name": "Preference",
						"fields": [
							{"name": "key", "type": "string"},
							{"name": "value", "type": "string"}
						]
					}}
				]
			}},
			{"name": "item", "type": {
				"type": "record",
				"name": "LineItem",
				"fields": [
					{"name": "product", "type": "string"}
				]
			}}
		]
	}`

	canonical := canonicalize(schema)

	// Order → com.example.Order
	if !strings.Contains(canonical, `"name":"com.example.Order"`) {
		t.Errorf("Order should be com.example.Order, got: %s", canonical)
	}
	// Customer → com.customers.Customer (explicit override)
	if !strings.Contains(canonical, `"name":"com.customers.Customer"`) {
		t.Errorf("Customer should be com.customers.Customer, got: %s", canonical)
	}
	// Preference → com.customers.Preference (inherits from Customer)
	if !strings.Contains(canonical, `"name":"com.customers.Preference"`) {
		t.Errorf("Preference should inherit com.customers from Customer, got: %s", canonical)
	}
	// LineItem → com.example.LineItem (inherits from Order, not from Customer)
	if !strings.Contains(canonical, `"name":"com.example.LineItem"`) {
		t.Errorf("LineItem should inherit com.example from Order, got: %s", canonical)
	}
}
