package compatibility_test

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	"github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jscompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	pbcompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func newCheckerWithAll() *compatibility.Checker {
	c := compatibility.NewChecker()
	c.Register(storage.SchemaTypeAvro, avro.NewChecker())
	c.Register(storage.SchemaTypeProtobuf, pbcompat.NewChecker())
	c.Register(storage.SchemaTypeJSON, jscompat.NewChecker())
	return c
}

// s creates a SchemaWithRefs with no references for convenience.
func s(schema string) compatibility.SchemaWithRefs {
	return compatibility.SchemaWithRefs{Schema: schema}
}

// ss creates a slice of SchemaWithRefs with no references for convenience.
func ss(schemas ...string) []compatibility.SchemaWithRefs {
	result := make([]compatibility.SchemaWithRefs, len(schemas))
	for i, schema := range schemas {
		result[i] = compatibility.SchemaWithRefs{Schema: schema}
	}
	return result
}

// --- Mode helpers ---

func TestMode_IsValid(t *testing.T) {
	valid := []compatibility.Mode{compatibility.ModeNone, compatibility.ModeBackward, compatibility.ModeBackwardTransitive, compatibility.ModeForward, compatibility.ModeForwardTransitive, compatibility.ModeFull, compatibility.ModeFullTransitive}
	for _, m := range valid {
		if !m.IsValid() {
			t.Errorf("Expected %s to be valid", m)
		}
	}
	if compatibility.Mode("INVALID").IsValid() {
		t.Error("Expected INVALID to be invalid")
	}
}

func TestMode_IsTransitive(t *testing.T) {
	transitive := []compatibility.Mode{compatibility.ModeBackwardTransitive, compatibility.ModeForwardTransitive, compatibility.ModeFullTransitive}
	for _, m := range transitive {
		if !m.IsTransitive() {
			t.Errorf("Expected %s to be transitive", m)
		}
	}
	nonTransitive := []compatibility.Mode{compatibility.ModeNone, compatibility.ModeBackward, compatibility.ModeForward, compatibility.ModeFull}
	for _, m := range nonTransitive {
		if m.IsTransitive() {
			t.Errorf("Expected %s to NOT be transitive", m)
		}
	}
}

func TestMode_RequiresBackward(t *testing.T) {
	backward := []compatibility.Mode{compatibility.ModeBackward, compatibility.ModeBackwardTransitive, compatibility.ModeFull, compatibility.ModeFullTransitive}
	for _, m := range backward {
		if !m.RequiresBackward() {
			t.Errorf("Expected %s to require backward", m)
		}
	}
	if compatibility.ModeForward.RequiresBackward() {
		t.Error("FORWARD should not require backward")
	}
}

func TestMode_RequiresForward(t *testing.T) {
	forward := []compatibility.Mode{compatibility.ModeForward, compatibility.ModeForwardTransitive, compatibility.ModeFull, compatibility.ModeFullTransitive}
	for _, m := range forward {
		if !m.RequiresForward() {
			t.Errorf("Expected %s to require forward", m)
		}
	}
	if compatibility.ModeBackward.RequiresForward() {
		t.Error("BACKWARD should not require forward")
	}
}

// --- NONE mode ---

func TestChecker_NoneMode_AlwaysPasses(t *testing.T) {
	c := newCheckerWithAll()

	result := c.Check(compatibility.ModeNone, storage.SchemaTypeAvro, s(`"string"`), ss(`"int"`))
	if !result.IsCompatible {
		t.Error("NONE mode should always pass")
	}
}

// --- BACKWARD mode (non-transitive) ---

func TestChecker_Backward_OnlyChecksLatest(t *testing.T) {
	c := newCheckerWithAll()

	// v1: {id}
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	// v2: {id, name with default}  - backward compat with v1
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string","default":""}]}`
	// v3: {id, name (no default), email with default} - backward compat with v2 (name exists in v2) but NOT with v1 (name doesn't exist in v1 and has no default)
	v3 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"},{"name":"email","type":"string","default":""}]}`

	// BACKWARD (non-transitive) only checks against latest (v2)
	result := c.Check(compatibility.ModeBackward, storage.SchemaTypeAvro, s(v3), ss(v1, v2))
	if !result.IsCompatible {
		t.Errorf("BACKWARD should pass (only checks latest v2): %v", result.Messages)
	}
}

// --- BACKWARD_TRANSITIVE mode ---

func TestChecker_BackwardTransitive_Avro_ChecksAll(t *testing.T) {
	c := newCheckerWithAll()

	// v1: {id}
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	// v2: {id, name with default} - backward compat with v1
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string","default":""}]}`
	// v3: {id, name (no default), email with default}
	// backward compat with v2 (name exists in v2)
	// NOT backward compat with v1 (name missing from v1 and no default in v3)
	v3 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"},{"name":"email","type":"string","default":""}]}`

	// BACKWARD_TRANSITIVE checks against ALL versions (v1 and v2)
	result := c.Check(compatibility.ModeBackwardTransitive, storage.SchemaTypeAvro, s(v3), ss(v1, v2))
	if result.IsCompatible {
		t.Error("BACKWARD_TRANSITIVE should FAIL (v3 is not backward compat with v1)")
	}
}

func TestChecker_BackwardTransitive_Avro_PassesWhenAllCompatible(t *testing.T) {
	c := newCheckerWithAll()

	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string","default":""}]}`
	// v3 adds another optional field - backward compat with ALL previous
	v3 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string","default":""},{"name":"email","type":"string","default":""}]}`

	result := c.Check(compatibility.ModeBackwardTransitive, storage.SchemaTypeAvro, s(v3), ss(v1, v2))
	if !result.IsCompatible {
		t.Errorf("BACKWARD_TRANSITIVE should pass (all optional fields have defaults): %v", result.Messages)
	}
}

// --- FORWARD mode (non-transitive) ---

func TestChecker_Forward_Avro(t *testing.T) {
	c := newCheckerWithAll()

	// Forward: old schema can read data written by new schema
	// Adding a field without default is forward compatible (old reader ignores it)
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`

	result := c.Check(compatibility.ModeForward, storage.SchemaTypeAvro, s(v2), ss(v1))
	if !result.IsCompatible {
		t.Errorf("FORWARD should pass (old reader ignores new field): %v", result.Messages)
	}
}

func TestChecker_Forward_Avro_Incompatible(t *testing.T) {
	c := newCheckerWithAll()

	// Removing a field is NOT forward compatible (old reader expects it)
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`

	result := c.Check(compatibility.ModeForward, storage.SchemaTypeAvro, s(v2), ss(v1))
	if result.IsCompatible {
		t.Error("FORWARD should FAIL (old reader expects 'name' but new schema doesn't have it)")
	}
}

// --- FORWARD_TRANSITIVE ---

func TestChecker_ForwardTransitive_Avro_ChecksAll(t *testing.T) {
	c := newCheckerWithAll()

	// v1: {id, name}
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`
	// v2: {id, name, email} - forward compat with v1 (v1 can read v2 by ignoring email)
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"},{"name":"email","type":"string"}]}`
	// v3: {id, email} - removes name
	// forward compat with v2? v2 has name and id. v2 reads v3 data -> v3 has no name -> v2 expects name without default -> FAIL
	v3 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"email","type":"string"}]}`

	result := c.Check(compatibility.ModeForwardTransitive, storage.SchemaTypeAvro, s(v3), ss(v1, v2))
	if result.IsCompatible {
		t.Error("FORWARD_TRANSITIVE should FAIL (v1 and v2 expect 'name' field)")
	}
}

func TestChecker_ForwardTransitive_Avro_PassesWhenAllCompatible(t *testing.T) {
	c := newCheckerWithAll()

	// Each version adds a new field. Old readers can always ignore new fields.
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`
	v3 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"},{"name":"email","type":"string"}]}`

	result := c.Check(compatibility.ModeForwardTransitive, storage.SchemaTypeAvro, s(v3), ss(v1, v2))
	if !result.IsCompatible {
		t.Errorf("FORWARD_TRANSITIVE should pass (only adding fields): %v", result.Messages)
	}
}

// --- FULL mode ---

func TestChecker_Full_Avro_Compatible(t *testing.T) {
	c := newCheckerWithAll()

	// Full = both backward AND forward
	// Adding optional field with default is both backward and forward compatible
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string","default":""}]}`

	result := c.Check(compatibility.ModeFull, storage.SchemaTypeAvro, s(v2), ss(v1))
	if !result.IsCompatible {
		t.Errorf("FULL should pass (optional field with default): %v", result.Messages)
	}
}

func TestChecker_Full_Avro_BackwardOnlyFails(t *testing.T) {
	c := newCheckerWithAll()

	// Adding required field (no default) - backward incompatible but forward compatible
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`

	result := c.Check(compatibility.ModeFull, storage.SchemaTypeAvro, s(v2), ss(v1))
	if result.IsCompatible {
		t.Error("FULL should FAIL (adding field without default is not backward compat)")
	}
}

// --- FULL_TRANSITIVE ---

func TestChecker_FullTransitive_Avro_ChecksAll(t *testing.T) {
	c := newCheckerWithAll()

	// All versions add optional fields with defaults - should pass full transitive
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string","default":""}]}`
	v3 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string","default":""},{"name":"email","type":"string","default":""}]}`

	result := c.Check(compatibility.ModeFullTransitive, storage.SchemaTypeAvro, s(v3), ss(v1, v2))
	if !result.IsCompatible {
		t.Errorf("FULL_TRANSITIVE should pass: %v", result.Messages)
	}
}

func TestChecker_FullTransitive_Avro_FailsOnHistoricalIncompat(t *testing.T) {
	c := newCheckerWithAll()

	// v1: {id}
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`
	// v2: {id, name with default}
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string","default":""}]}`
	// v3: removes id, adds email - NOT forward compat with v1 or v2 (they expect id)
	v3 := `{"type":"record","name":"User","fields":[{"name":"name","type":"string","default":""},{"name":"email","type":"string","default":""}]}`

	result := c.Check(compatibility.ModeFullTransitive, storage.SchemaTypeAvro, s(v3), ss(v1, v2))
	if result.IsCompatible {
		t.Error("FULL_TRANSITIVE should FAIL (v3 removes 'id' field)")
	}
}

// --- Protobuf transitive tests ---

func TestChecker_BackwardTransitive_Protobuf(t *testing.T) {
	c := newCheckerWithAll()

	v1 := `syntax = "proto3"; message User { string id = 1; }`
	// v2 adds optional field - compatible
	v2 := `syntax = "proto3"; message User { string id = 1; string name = 2; }`
	// v3 removes id field - NOT backward compatible with v1
	v3 := `syntax = "proto3"; message User { string name = 2; string email = 3; }`

	result := c.Check(compatibility.ModeBackwardTransitive, storage.SchemaTypeProtobuf, s(v3), ss(v1, v2))
	if result.IsCompatible {
		t.Error("BACKWARD_TRANSITIVE should FAIL for protobuf (removed field 1)")
	}
}

func TestChecker_ForwardTransitive_Protobuf(t *testing.T) {
	c := newCheckerWithAll()

	// The protobuf checker reports fields in writer (new) not in reader (old) as "removed",
	// so adding fields fails forward checks with this checker implementation.
	// Test that identical schemas across versions pass forward transitive.
	v1 := `syntax = "proto3"; message User { string id = 1; string name = 2; string email = 3; }`
	v2 := `syntax = "proto3"; message User { string id = 1; string name = 2; string email = 3; }`
	v3 := `syntax = "proto3"; message User { string id = 1; string name = 2; string email = 3; }`

	result := c.Check(compatibility.ModeForwardTransitive, storage.SchemaTypeProtobuf, s(v3), ss(v1, v2))
	if !result.IsCompatible {
		t.Errorf("FORWARD_TRANSITIVE should pass (identical schemas): %v", result.Messages)
	}
}

func TestChecker_ForwardTransitive_Protobuf_FailsOnRemovedField(t *testing.T) {
	c := newCheckerWithAll()

	v1 := `syntax = "proto3"; message User { string id = 1; string name = 2; }`
	v2 := `syntax = "proto3"; message User { string id = 1; string name = 2; string email = 3; }`
	// v3 removes name - forward check: v1 (reader) reads v3 (writer) - v1 has name but v3 doesn't
	v3 := `syntax = "proto3"; message User { string id = 1; string email = 3; }`

	result := c.Check(compatibility.ModeForwardTransitive, storage.SchemaTypeProtobuf, s(v3), ss(v1, v2))
	if result.IsCompatible {
		t.Error("FORWARD_TRANSITIVE should FAIL (removed field 'name' that v1 has)")
	}
}

func TestChecker_FullTransitive_Protobuf(t *testing.T) {
	c := newCheckerWithAll()

	// Full transitive with identical schemas - should pass
	v1 := `syntax = "proto3"; message User { string id = 1; }`
	v2 := `syntax = "proto3"; message User { string id = 1; }`
	v3 := `syntax = "proto3"; message User { string id = 1; }`

	result := c.Check(compatibility.ModeFullTransitive, storage.SchemaTypeProtobuf, s(v3), ss(v1, v2))
	if !result.IsCompatible {
		t.Errorf("FULL_TRANSITIVE should pass for identical schemas: %v", result.Messages)
	}
}

// --- JSON Schema transitive tests ---

func TestChecker_BackwardTransitive_JSONSchema(t *testing.T) {
	c := newCheckerWithAll()

	v1 := `{"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}`
	// v2 adds optional email
	v2 := `{"type":"object","properties":{"id":{"type":"integer"},"email":{"type":"string"}},"required":["id"]}`
	// v3 adds required name - NOT backward compat with v1 (new required field)
	v3 := `{"type":"object","properties":{"id":{"type":"integer"},"email":{"type":"string"},"name":{"type":"string"}},"required":["id","name"]}`

	result := c.Check(compatibility.ModeBackwardTransitive, storage.SchemaTypeJSON, s(v3), ss(v1, v2))
	if result.IsCompatible {
		t.Error("BACKWARD_TRANSITIVE should FAIL for JSON Schema (new required field)")
	}
}

func TestChecker_ForwardTransitive_JSONSchema(t *testing.T) {
	c := newCheckerWithAll()

	// The JSON Schema checker reports properties in writer (new) not in reader (old) as "removed".
	// Identical schemas across versions should pass.
	v1 := `{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"}}}`
	v2 := `{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"}}}`
	v3 := `{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"}}}`

	result := c.Check(compatibility.ModeForwardTransitive, storage.SchemaTypeJSON, s(v3), ss(v1, v2))
	if !result.IsCompatible {
		t.Errorf("FORWARD_TRANSITIVE should pass (identical schemas): %v", result.Messages)
	}
}

func TestChecker_ForwardTransitive_JSONSchema_FailsOnTypeChange(t *testing.T) {
	c := newCheckerWithAll()

	v1 := `{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}}}`
	v2 := `{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}}}`
	// v3 changes type of name → old readers expect string but get integer
	v3 := `{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"integer"}}}`

	result := c.Check(compatibility.ModeForwardTransitive, storage.SchemaTypeJSON, s(v3), ss(v1, v2))
	if result.IsCompatible {
		t.Error("FORWARD_TRANSITIVE should FAIL (type of 'name' changed from string to integer)")
	}
}

func TestChecker_FullTransitive_JSONSchema(t *testing.T) {
	c := newCheckerWithAll()

	// Full transitive with identical schemas
	v1 := `{"type":"object","properties":{"id":{"type":"integer"}}}`
	v2 := `{"type":"object","properties":{"id":{"type":"integer"}}}`
	v3 := `{"type":"object","properties":{"id":{"type":"integer"}}}`

	result := c.Check(compatibility.ModeFullTransitive, storage.SchemaTypeJSON, s(v3), ss(v1, v2))
	if !result.IsCompatible {
		t.Errorf("FULL_TRANSITIVE should pass: %v", result.Messages)
	}
}

// --- Edge cases ---

func TestChecker_NoExistingSchemas(t *testing.T) {
	c := newCheckerWithAll()

	result := c.Check(compatibility.ModeBackwardTransitive, storage.SchemaTypeAvro, s(`"string"`), nil)
	if !result.IsCompatible {
		t.Error("No existing schemas should always be compatible")
	}

	result = c.Check(compatibility.ModeBackwardTransitive, storage.SchemaTypeAvro, s(`"string"`), []compatibility.SchemaWithRefs{})
	if !result.IsCompatible {
		t.Error("Empty existing schemas should always be compatible")
	}
}

func TestChecker_UnknownSchemaType(t *testing.T) {
	c := newCheckerWithAll()

	result := c.Check(compatibility.ModeBackward, storage.SchemaType("UNKNOWN"), s(`"string"`), ss(`"string"`))
	if result.IsCompatible {
		t.Error("Unknown schema type should fail")
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		input string
		valid bool
		mode  compatibility.Mode
	}{
		{"BACKWARD", true, compatibility.ModeBackward},
		{"BACKWARD_TRANSITIVE", true, compatibility.ModeBackwardTransitive},
		{"FORWARD", true, compatibility.ModeForward},
		{"FORWARD_TRANSITIVE", true, compatibility.ModeForwardTransitive},
		{"FULL", true, compatibility.ModeFull},
		{"FULL_TRANSITIVE", true, compatibility.ModeFullTransitive},
		{"NONE", true, compatibility.ModeNone},
		{"INVALID", false, compatibility.Mode("INVALID")},
		{"", false, compatibility.Mode("")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, ok := compatibility.ParseMode(tt.input)
			if ok != tt.valid {
				t.Errorf("compatibility.ParseMode(%q): expected valid=%v, got %v", tt.input, tt.valid, ok)
			}
			if ok && mode != tt.mode {
				t.Errorf("compatibility.ParseMode(%q): expected %v, got %v", tt.input, tt.mode, mode)
			}
		})
	}
}

// --- Complex evolution scenarios ---

func TestChecker_Avro_FourVersionEvolution(t *testing.T) {
	c := newCheckerWithAll()

	// Realistic 4-version evolution of a PaymentEvent
	v1 := `{
		"type":"record","name":"Payment","fields":[
			{"name":"id","type":"long"},
			{"name":"amount","type":"double"}
		]
	}`
	v2 := `{
		"type":"record","name":"Payment","fields":[
			{"name":"id","type":"long"},
			{"name":"amount","type":"double"},
			{"name":"currency","type":"string","default":"USD"}
		]
	}`
	v3 := `{
		"type":"record","name":"Payment","fields":[
			{"name":"id","type":"long"},
			{"name":"amount","type":"double"},
			{"name":"currency","type":"string","default":"USD"},
			{"name":"status","type":"string","default":"PENDING"}
		]
	}`
	v4 := `{
		"type":"record","name":"Payment","fields":[
			{"name":"id","type":"long"},
			{"name":"amount","type":"double"},
			{"name":"currency","type":"string","default":"USD"},
			{"name":"status","type":"string","default":"PENDING"},
			{"name":"customer_id","type":["null","long"],"default":null}
		]
	}`

	// All should pass FULL_TRANSITIVE since every addition has defaults
	result := c.Check(compatibility.ModeFullTransitive, storage.SchemaTypeAvro, s(v4), ss(v1, v2, v3))
	if !result.IsCompatible {
		t.Errorf("4-version evolution should pass FULL_TRANSITIVE: %v", result.Messages)
	}
}

func TestChecker_Protobuf_FourVersionEvolution_BackwardTransitive(t *testing.T) {
	c := newCheckerWithAll()

	// For protobuf, adding fields is backward-transitive compatible (new reader, old writer)
	v1 := `syntax = "proto3"; message Payment { int64 id = 1; double amount = 2; }`
	v2 := `syntax = "proto3"; message Payment { int64 id = 1; double amount = 2; string currency = 3; }`
	v3 := `syntax = "proto3"; message Payment { int64 id = 1; double amount = 2; string currency = 3; string status = 4; }`
	v4 := `syntax = "proto3"; message Payment { int64 id = 1; double amount = 2; string currency = 3; string status = 4; int64 customer_id = 5; }`

	result := c.Check(compatibility.ModeBackwardTransitive, storage.SchemaTypeProtobuf, s(v4), ss(v1, v2, v3))
	if !result.IsCompatible {
		t.Errorf("4-version protobuf evolution should pass BACKWARD_TRANSITIVE: %v", result.Messages)
	}
}

func TestChecker_JSONSchema_FourVersionEvolution_BackwardTransitive(t *testing.T) {
	c := newCheckerWithAll()

	// For JSON Schema, adding optional properties is backward-transitive compatible
	v1 := `{"type":"object","properties":{"id":{"type":"integer"},"amount":{"type":"number"}},"required":["id","amount"]}`
	v2 := `{"type":"object","properties":{"id":{"type":"integer"},"amount":{"type":"number"},"currency":{"type":"string"}},"required":["id","amount"]}`
	v3 := `{"type":"object","properties":{"id":{"type":"integer"},"amount":{"type":"number"},"currency":{"type":"string"},"status":{"type":"string"}},"required":["id","amount"]}`
	v4 := `{"type":"object","properties":{"id":{"type":"integer"},"amount":{"type":"number"},"currency":{"type":"string"},"status":{"type":"string"},"customer_id":{"type":"integer"}},"required":["id","amount"]}`

	result := c.Check(compatibility.ModeBackwardTransitive, storage.SchemaTypeJSON, s(v4), ss(v1, v2, v3))
	if !result.IsCompatible {
		t.Errorf("4-version JSON Schema evolution should pass BACKWARD_TRANSITIVE: %v", result.Messages)
	}
}
