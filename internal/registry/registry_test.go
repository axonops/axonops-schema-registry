package registry

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsonschemacompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protobufcompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

// setupTestRegistry creates a test registry with memory storage and Avro support.
func setupTestRegistry(defaultCompatibility string) *Registry {
	store := memory.NewStore()
	// Set the global config to match the requested default so the store's
	// seeded BACKWARD doesn't override what the test expects.
	store.SetGlobalConfig(context.Background(), ".", &storage.ConfigRecord{CompatibilityLevel: defaultCompatibility})

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())

	return New(store, schemaRegistry, compatChecker, defaultCompatibility)
}

// setupMultiTypeRegistry creates a test registry supporting all three schema types.
func setupMultiTypeRegistry(defaultCompatibility string) *Registry {
	store := memory.NewStore()
	store.SetGlobalConfig(context.Background(), ".", &storage.ConfigRecord{CompatibilityLevel: defaultCompatibility})

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())
	schemaRegistry.Register(protobuf.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsonschemacompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protobufcompat.NewChecker())

	return New(store, schemaRegistry, compatChecker, defaultCompatibility)
}

// --- RegisterSchema tests ---

func TestRegisterSchema_DefaultsToAvro(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	record, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, "", nil) // empty type
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if record.SchemaType != storage.SchemaTypeAvro {
		t.Errorf("expected Avro type, got %s", record.SchemaType)
	}
}

func TestRegisterSchema_UnsupportedType(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.RegisterSchema(ctx, ".", "test-subject", "{}", "UNKNOWN_TYPE", nil)
	if err == nil {
		t.Error("expected error for unsupported schema type")
	}
}

func TestRegisterSchema_InvalidSchema(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.RegisterSchema(ctx, ".", "test-subject", "not valid avro", storage.SchemaTypeAvro, nil)
	if err == nil {
		t.Error("expected error for invalid schema")
	}
}

func TestRegisterSchema_DuplicateReturnsExisting(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`

	rec1, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	rec2, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("second register failed: %v", err)
	}

	if rec1.ID != rec2.ID {
		t.Errorf("duplicate register should return same ID: got %d and %d", rec1.ID, rec2.ID)
	}
}

func TestRegisterSchema_CompatibilityRejection(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	schema1 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", schema1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register v1: %v", err)
	}

	// Incompatible: adds required field (no default)
	schema2 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}`
	_, err = reg.RegisterSchema(ctx, ".", "test-subject", schema2, storage.SchemaTypeAvro, nil)
	if err == nil {
		t.Error("expected compatibility error")
	}
	if !errors.Is(err, ErrIncompatibleSchema) {
		t.Errorf("expected ErrIncompatibleSchema, got: %v", err)
	}
}

func TestRegisterSchema_CompatibilityNone(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema1 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", schema1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register v1: %v", err)
	}

	// With NONE, any schema should pass (even incompatible ones)
	schema2 := `{"type":"record","name":"Test","fields":[{"name":"name","type":"string"}]}`
	_, err = reg.RegisterSchema(ctx, ".", "test-subject", schema2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Errorf("NONE compatibility should allow any schema: %v", err)
	}
}

func TestRegisterSchema_JSONSchema(t *testing.T) {
	reg := setupMultiTypeRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"object","properties":{"name":{"type":"string"}}}`
	record, err := reg.RegisterSchema(ctx, ".", "json-subject", schema, storage.SchemaTypeJSON, nil)
	if err != nil {
		t.Fatalf("failed to register JSON schema: %v", err)
	}
	if record.SchemaType != storage.SchemaTypeJSON {
		t.Errorf("expected JSON type, got %s", record.SchemaType)
	}
}

func TestRegisterSchema_Protobuf(t *testing.T) {
	reg := setupMultiTypeRegistry("NONE")
	ctx := context.Background()

	schema := `syntax = "proto3"; message User { string name = 1; }`
	record, err := reg.RegisterSchema(ctx, ".", "proto-subject", schema, storage.SchemaTypeProtobuf, nil)
	if err != nil {
		t.Fatalf("failed to register Protobuf schema: %v", err)
	}
	if record.SchemaType != storage.SchemaTypeProtobuf {
		t.Errorf("expected PROTOBUF type, got %s", record.SchemaType)
	}
}

// --- GetSchemaByID tests ---

func TestGetSchemaByID(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	record, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	found, err := reg.GetSchemaByID(ctx, ".", record.ID)
	if err != nil {
		t.Fatalf("failed to get by ID: %v", err)
	}
	if found.ID != record.ID {
		t.Errorf("expected ID %d, got %d", record.ID, found.ID)
	}
	if found.Schema != schema {
		t.Errorf("expected schema content to match")
	}
	if found.SchemaType != storage.SchemaTypeAvro {
		t.Errorf("expected type AVRO, got %s", found.SchemaType)
	}
}

func TestGetSchemaByID_NotFound(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.GetSchemaByID(ctx, ".", 99999)
	if err == nil {
		t.Error("expected error for non-existent ID")
	}
}

// --- GetSchemaBySubjectVersion tests ---

func TestGetSchemaBySubjectVersion(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	found, err := reg.GetSchemaBySubjectVersion(ctx, ".", "test-subject", 1)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if found.Version != 1 {
		t.Errorf("expected version 1, got %d", found.Version)
	}
}

func TestGetSchemaBySubjectVersion_NotFound(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.GetSchemaBySubjectVersion(ctx, ".", "nonexistent", 1)
	if err == nil {
		t.Error("expected error for non-existent subject/version")
	}
}

// --- GetVersions tests ---

func TestGetVersions(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	s1 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	s2 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"f","type":"string","default":""}]}`

	_, err := reg.RegisterSchema(ctx, ".", "test-subject", s1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register v1: %v", err)
	}
	_, err = reg.RegisterSchema(ctx, ".", "test-subject", s2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register v2: %v", err)
	}

	versions, err := reg.GetVersions(ctx, ".", "test-subject", false)
	if err != nil {
		t.Fatalf("failed to get versions: %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
}

func TestGetVersions_NonexistentSubject(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.GetVersions(ctx, ".", "nonexistent", false)
	if err == nil {
		t.Error("expected error for nonexistent subject")
	}
}

// --- ListSubjects tests ---

func TestListSubjects(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	subjects, err := reg.ListSubjects(ctx, ".", false)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(subjects) != 0 {
		t.Errorf("expected 0 subjects, got %d", len(subjects))
	}

	s := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err = reg.RegisterSchema(ctx, ".", "subject-1", s, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	subjects, err = reg.ListSubjects(ctx, ".", false)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(subjects) != 1 {
		t.Errorf("expected 1 subject, got %d", len(subjects))
	}
}

// --- DeleteVersion tests ---

func TestDeleteVersion(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	s1 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	s2 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"f","type":"string","default":""}]}`

	_, err := reg.RegisterSchema(ctx, ".", "test-subject", s1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register v1: %v", err)
	}
	_, err = reg.RegisterSchema(ctx, ".", "test-subject", s2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register v2: %v", err)
	}

	ver, err := reg.DeleteVersion(ctx, ".", "test-subject", 1, false)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}
	if ver != 1 {
		t.Errorf("expected deleted version 1, got %d", ver)
	}
}

func TestDeleteVersion_NonexistentVersion(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	s := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", s, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	_, err = reg.DeleteVersion(ctx, ".", "test-subject", 999, false)
	if err == nil {
		t.Error("expected error for non-existent version")
	}
}

func TestDeleteVersion_ReferencedBlocked(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register base schema
	base := `{"type":"record","name":"Base","namespace":"test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "base-subject", base, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register base: %v", err)
	}

	// Register referencing schema
	referencing := `{"type":"record","name":"Ref","namespace":"test","fields":[{"name":"base","type":"test.Base"}]}`
	refs := []storage.Reference{{Name: "test.Base", Subject: "base-subject", Version: 1}}
	_, err = reg.RegisterSchema(ctx, ".", "ref-subject", referencing, storage.SchemaTypeAvro, refs)
	if err != nil {
		t.Fatalf("failed to register referencing: %v", err)
	}

	// Attempt to delete referenced version - should be blocked
	_, err = reg.DeleteVersion(ctx, ".", "base-subject", 1, false)
	if err == nil {
		t.Error("expected error when deleting referenced version")
	}
}

// --- Config tests ---

func TestGetConfig_SubjectFallsBackToGlobal(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	level, err := reg.GetConfig(ctx, ".", "nonexistent-subject")
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}
	if level != "BACKWARD" {
		t.Errorf("expected BACKWARD (default), got %s", level)
	}
}

func TestGetConfig_GlobalReturnsStoredValue(t *testing.T) {
	reg := setupTestRegistry("FULL")
	ctx := context.Background()

	// setupTestRegistry sets the global config to match the requested default.
	level, err := reg.GetConfig(ctx, ".", "")
	if err != nil {
		t.Fatalf("failed to get global config: %v", err)
	}
	if level != "FULL" {
		t.Errorf("expected FULL, got %s", level)
	}
}

func TestSetConfig_SubjectLevel(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	err := reg.SetConfig(ctx, ".", "my-subject", "FULL", nil)
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	level, err := reg.GetConfig(ctx, ".", "my-subject")
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}
	if level != "FULL" {
		t.Errorf("expected FULL, got %s", level)
	}
}

func TestSetConfig_GlobalLevel(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	err := reg.SetConfig(ctx, ".", "", "NONE", nil)
	if err != nil {
		t.Fatalf("failed to set global config: %v", err)
	}

	level, err := reg.GetConfig(ctx, ".", "")
	if err != nil {
		t.Fatalf("failed to get global config: %v", err)
	}
	if level != "NONE" {
		t.Errorf("expected NONE, got %s", level)
	}
}

func TestSetConfig_InvalidLevel(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	err := reg.SetConfig(ctx, ".", "test", "INVALID", nil)
	if err == nil {
		t.Error("expected error for invalid compatibility level")
	}
}

func TestSetConfig_CaseInsensitive(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	err := reg.SetConfig(ctx, ".", "test", "backward", nil)
	if err != nil {
		t.Fatalf("expected lowercase to be accepted: %v", err)
	}
}

func TestSetConfig_AllValidLevels(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	validLevels := []string{
		"NONE", "BACKWARD", "BACKWARD_TRANSITIVE",
		"FORWARD", "FORWARD_TRANSITIVE",
		"FULL", "FULL_TRANSITIVE",
	}

	for _, level := range validLevels {
		err := reg.SetConfig(ctx, ".", "", level, nil)
		if err != nil {
			t.Errorf("level %s should be valid: %v", level, err)
		}
	}
}

func TestDeleteConfig(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	err := reg.SetConfig(ctx, ".", "my-subject", "FULL", nil)
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	prev, err := reg.DeleteConfig(ctx, ".", "my-subject")
	if err != nil {
		t.Fatalf("failed to delete config: %v", err)
	}
	if prev != "FULL" {
		t.Errorf("expected previous level FULL, got %s", prev)
	}

	// After delete, should fall back to global/default
	level, err := reg.GetConfig(ctx, ".", "my-subject")
	if err != nil {
		t.Fatalf("failed to get config after delete: %v", err)
	}
	if level != "BACKWARD" {
		t.Errorf("expected fallback to BACKWARD, got %s", level)
	}
}

func TestDeleteConfig_NotFound(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	_, err := reg.DeleteConfig(ctx, ".", "nonexistent")
	if err == nil {
		t.Error("expected error when deleting non-existent config")
	}
}

func TestDeleteGlobalConfig(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	err := reg.SetConfig(ctx, ".", "", "FULL", nil)
	if err != nil {
		t.Fatalf("failed to set global config: %v", err)
	}

	prev, err := reg.DeleteGlobalConfig(ctx, ".")
	if err != nil {
		t.Fatalf("failed to delete global config: %v", err)
	}
	if prev != "FULL" {
		t.Errorf("expected previous level FULL, got %s", prev)
	}
}

func TestDeleteGlobalConfig_NoExistingConfig(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	// When no global config is set, should return the default
	prev, err := reg.DeleteGlobalConfig(ctx, ".")
	if err != nil {
		t.Fatalf("delete global config should not error: %v", err)
	}
	if prev != "BACKWARD" {
		t.Errorf("expected default BACKWARD, got %s", prev)
	}
}

// --- Mode tests ---

func TestGetMode_DefaultReadWrite(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	mode, err := reg.GetMode(ctx, ".", "")
	if err != nil {
		t.Fatalf("failed to get mode: %v", err)
	}
	if mode != "READWRITE" {
		t.Errorf("expected default READWRITE, got %s", mode)
	}
}

func TestGetMode_SubjectFallsBackToGlobal(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	mode, err := reg.GetMode(ctx, ".", "some-subject")
	if err != nil {
		t.Fatalf("failed to get mode: %v", err)
	}
	if mode != "READWRITE" {
		t.Errorf("expected default READWRITE, got %s", mode)
	}
}

func TestSetMode(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	err := reg.SetMode(ctx, ".", "my-subject", "READONLY", false)
	if err != nil {
		t.Fatalf("failed to set mode: %v", err)
	}

	mode, err := reg.GetMode(ctx, ".", "my-subject")
	if err != nil {
		t.Fatalf("failed to get mode: %v", err)
	}
	if mode != "READONLY" {
		t.Errorf("expected READONLY, got %s", mode)
	}
}

func TestSetMode_Global(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	err := reg.SetMode(ctx, ".", "", "IMPORT", true)
	if err != nil {
		t.Fatalf("failed to set global mode: %v", err)
	}

	mode, err := reg.GetMode(ctx, ".", "")
	if err != nil {
		t.Fatalf("failed to get global mode: %v", err)
	}
	if mode != "IMPORT" {
		t.Errorf("expected IMPORT, got %s", mode)
	}
}

func TestSetMode_Invalid(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	err := reg.SetMode(ctx, ".", "", "INVALID", false)
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestSetMode_AllValidModes(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	validModes := []string{"READWRITE", "READONLY", "IMPORT"}
	for _, mode := range validModes {
		err := reg.SetMode(ctx, ".", "", mode, true)
		if err != nil {
			t.Errorf("mode %s should be valid: %v", mode, err)
		}
	}
}

func TestSetMode_CaseInsensitive(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	err := reg.SetMode(ctx, ".", "", "readonly", false)
	if err != nil {
		t.Fatalf("expected lowercase to be accepted: %v", err)
	}
}

func TestDeleteMode(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	err := reg.SetMode(ctx, ".", "my-subject", "READONLY", false)
	if err != nil {
		t.Fatalf("failed to set mode: %v", err)
	}

	prev, err := reg.DeleteMode(ctx, ".", "my-subject")
	if err != nil {
		t.Fatalf("failed to delete mode: %v", err)
	}
	if prev != "READONLY" {
		t.Errorf("expected previous mode READONLY, got %s", prev)
	}
}

func TestDeleteMode_NotFound(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.DeleteMode(ctx, ".", "nonexistent")
	if err == nil {
		t.Error("expected error when deleting non-existent mode")
	}
}

// --- LookupSchema tests ---

func TestLookupSchema(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	record, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	found, err := reg.LookupSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil, false)
	if err != nil {
		t.Fatalf("failed to lookup: %v", err)
	}
	if found.ID != record.ID {
		t.Errorf("expected ID %d, got %d", record.ID, found.ID)
	}
}

func TestLookupSchema_NotFound(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.LookupSchema(ctx, ".", "nonexistent", schema, storage.SchemaTypeAvro, nil, false)
	if err == nil {
		t.Error("expected error for nonexistent subject")
	}
}

func TestLookupSchema_InvalidSchema(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.LookupSchema(ctx, ".", "test", "not valid avro", storage.SchemaTypeAvro, nil, false)
	if err == nil {
		t.Error("expected error for invalid schema")
	}
}

func TestLookupSchema_UnsupportedType(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.LookupSchema(ctx, ".", "test", "{}", "UNKNOWN_TYPE", nil, false)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestLookupSchema_WithDeleted(t *testing.T) {
	reg := setupTestRegistry("NONE")

	ctx := context.Background()

	// Register a schema
	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	record, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema: %v", err)
	}

	// Verify lookup works before delete
	found, err := reg.LookupSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil, false)
	if err != nil {
		t.Fatalf("failed to lookup schema: %v", err)
	}
	if found.ID != record.ID {
		t.Errorf("expected ID %d, got %d", record.ID, found.ID)
	}

	// Soft delete the subject
	_, err = reg.DeleteSubject(ctx, ".", "test-subject", false)
	if err != nil {
		t.Fatalf("failed to soft delete subject: %v", err)
	}

	// Lookup without deleted flag should fail
	_, err = reg.LookupSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil, false)
	if err == nil {
		t.Error("expected error when looking up deleted schema without deleted flag")
	}

	// Lookup with deleted flag should succeed
	found, err = reg.LookupSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil, true)
	if err != nil {
		t.Fatalf("failed to lookup deleted schema: %v", err)
	}
	if found.ID != record.ID {
		t.Errorf("expected ID %d, got %d", record.ID, found.ID)
	}
}

// --- DeleteSubject tests ---

func TestDeleteSubject_Soft(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	s := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", s, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	versions, err := reg.DeleteSubject(ctx, ".", "test-subject", false)
	if err != nil {
		t.Fatalf("failed to soft delete: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 deleted version, got %d", len(versions))
	}

	// Subject should not appear in list
	subjects, err := reg.ListSubjects(ctx, ".", false)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(subjects) != 0 {
		t.Errorf("expected 0 subjects after soft delete, got %d", len(subjects))
	}

	// Should appear with deleted=true
	subjects, err = reg.ListSubjects(ctx, ".", true)
	if err != nil {
		t.Fatalf("failed to list with deleted: %v", err)
	}
	if len(subjects) != 1 {
		t.Errorf("expected 1 subject with deleted=true, got %d", len(subjects))
	}
}

// --- GetRawSchemaByID tests ---

func TestGetRawSchemaByID(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	record, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	raw, err := reg.GetRawSchemaByID(ctx, ".", record.ID)
	if err != nil {
		t.Fatalf("failed to get raw: %v", err)
	}
	if raw != schema {
		t.Errorf("raw schema mismatch: got %q", raw)
	}
}

func TestGetRawSchemaByID_NotFound(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.GetRawSchemaByID(ctx, ".", 99999)
	if err == nil {
		t.Error("expected error for non-existent ID")
	}
}

// --- GetRawSchemaBySubjectVersion tests ---

func TestGetRawSchemaBySubjectVersion(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	raw, err := reg.GetRawSchemaBySubjectVersion(ctx, ".", "test-subject", 1)
	if err != nil {
		t.Fatalf("failed to get raw: %v", err)
	}
	if raw != schema {
		t.Errorf("raw schema mismatch")
	}
}

func TestGetRawSchemaBySubjectVersion_NotFound(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	_, err := reg.GetRawSchemaBySubjectVersion(ctx, ".", "nonexistent", 1)
	if err == nil {
		t.Error("expected error")
	}
}

// --- GetSchemaTypes tests ---

func TestGetSchemaTypes(t *testing.T) {
	reg := setupMultiTypeRegistry("NONE")

	types := reg.GetSchemaTypes()
	if len(types) != 3 {
		t.Errorf("expected 3 types, got %d: %v", len(types), types)
	}

	// Check that all three types are present
	typeSet := make(map[string]bool)
	for _, tp := range types {
		typeSet[tp] = true
	}
	for _, expected := range []string{"AVRO", "JSON", "PROTOBUF"} {
		if !typeSet[expected] {
			t.Errorf("expected type %s in list", expected)
		}
	}
}

func TestGetSchemaTypes_SingleType(t *testing.T) {
	reg := setupTestRegistry("NONE")

	types := reg.GetSchemaTypes()
	if len(types) != 1 {
		t.Errorf("expected 1 type, got %d", len(types))
	}
}

// --- GetSubjectsBySchemaID tests ---

func TestGetSubjectsBySchemaID(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	record, err := reg.RegisterSchema(ctx, ".", "subject-1", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	subjects, err := reg.GetSubjectsBySchemaID(ctx, ".", record.ID, false)
	if err != nil {
		t.Fatalf("failed to get subjects: %v", err)
	}
	if len(subjects) != 1 || subjects[0] != "subject-1" {
		t.Errorf("expected [subject-1], got %v", subjects)
	}
}

// --- GetVersionsBySchemaID tests ---

func TestGetVersionsBySchemaID(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	record, err := reg.RegisterSchema(ctx, ".", "subject-1", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	svs, err := reg.GetVersionsBySchemaID(ctx, ".", record.ID, false)
	if err != nil {
		t.Fatalf("failed to get versions: %v", err)
	}
	if len(svs) != 1 {
		t.Errorf("expected 1 subject-version pair, got %d", len(svs))
	}
}

// --- ImportSchemas tests ---

func TestImportSchemas_Success(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schemas := []ImportSchemaRequest{
		{
			ID:         100,
			Subject:    "import-subject",
			Version:    1,
			SchemaType: storage.SchemaTypeAvro,
			Schema:     `{"type":"record","name":"Imp","fields":[{"name":"id","type":"int"}]}`,
		},
		{
			ID:         101,
			Subject:    "import-subject",
			Version:    2,
			SchemaType: storage.SchemaTypeAvro,
			Schema:     `{"type":"record","name":"Imp","fields":[{"name":"id","type":"int"},{"name":"f","type":"string","default":""}]}`,
		},
	}

	result, err := reg.ImportSchemas(ctx, ".", schemas)
	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}
	if result.Imported != 2 {
		t.Errorf("expected 2 imported, got %d", result.Imported)
	}
	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}

	// Verify imported schemas are retrievable by subject/version
	record, err := reg.GetSchemaBySubjectVersion(ctx, ".", "import-subject", 1)
	if err != nil {
		t.Fatalf("failed to get imported schema: %v", err)
	}
	if record.ID != 100 {
		t.Errorf("expected ID 100, got %d", record.ID)
	}

	// Verify second schema
	record2, err := reg.GetSchemaBySubjectVersion(ctx, ".", "import-subject", 2)
	if err != nil {
		t.Fatalf("failed to get imported schema v2: %v", err)
	}
	if record2.ID != 101 {
		t.Errorf("expected ID 101, got %d", record2.ID)
	}
}

func TestImportSchemas_ValidationErrors(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schemas := []ImportSchemaRequest{
		{ID: 0, Subject: "test", Version: 1, Schema: "{}"}, // invalid ID
		{ID: 1, Subject: "", Version: 1, Schema: "{}"},     // missing subject
		{ID: 2, Subject: "test", Version: 0, Schema: "{}"}, // invalid version
		{ID: 3, Subject: "test", Version: 1, Schema: ""},   // empty schema
	}

	result, err := reg.ImportSchemas(ctx, ".", schemas)
	if err != nil {
		t.Fatalf("import should not return error for validation failures: %v", err)
	}
	if result.Errors != 4 {
		t.Errorf("expected 4 errors, got %d", result.Errors)
	}
	if result.Imported != 0 {
		t.Errorf("expected 0 imported, got %d", result.Imported)
	}
}

func TestImportSchemas_InvalidSchemaContent(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schemas := []ImportSchemaRequest{
		{
			ID:         1,
			Subject:    "test",
			Version:    1,
			SchemaType: storage.SchemaTypeAvro,
			Schema:     "not valid avro",
		},
	}

	result, err := reg.ImportSchemas(ctx, ".", schemas)
	if err != nil {
		t.Fatalf("import should not return error: %v", err)
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
}

func TestImportSchemas_UnsupportedType(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schemas := []ImportSchemaRequest{
		{
			ID:         1,
			Subject:    "test",
			Version:    1,
			SchemaType: "UNKNOWN",
			Schema:     "{}",
		},
	}

	result, err := reg.ImportSchemas(ctx, ".", schemas)
	if err != nil {
		t.Fatalf("import should not return error: %v", err)
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
}

func TestImportSchemas_DefaultsToAvro(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schemas := []ImportSchemaRequest{
		{
			ID:      1,
			Subject: "test",
			Version: 1,
			Schema:  `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`,
			// SchemaType is empty
		},
	}

	result, err := reg.ImportSchemas(ctx, ".", schemas)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

func TestImportSchemas_DuplicateID(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schemas := []ImportSchemaRequest{
		{
			ID:         1,
			Subject:    "test",
			Version:    1,
			SchemaType: storage.SchemaTypeAvro,
			Schema:     `{"type":"record","name":"A","fields":[{"name":"id","type":"int"}]}`,
		},
	}

	// First import
	_, err := reg.ImportSchemas(ctx, ".", schemas)
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}

	// Second import with same ID
	schemas2 := []ImportSchemaRequest{
		{
			ID:         1,
			Subject:    "test2",
			Version:    1,
			SchemaType: storage.SchemaTypeAvro,
			Schema:     `{"type":"record","name":"B","fields":[{"name":"id","type":"int"}]}`,
		},
	}

	result, err := reg.ImportSchemas(ctx, ".", schemas2)
	if err != nil {
		t.Fatalf("second import should not error: %v", err)
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error for duplicate ID, got %d", result.Errors)
	}
}

// --- IsHealthy tests ---

func TestIsHealthy(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	if !reg.IsHealthy(ctx) {
		t.Error("memory store should be healthy")
	}
}

// --- CheckCompatibility tests ---

func TestCheckCompatibility_ExplicitVersion(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")

	ctx := context.Background()

	// Register three versions of a schema
	schema1 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	schema2 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}`
	schema3 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"age","type":"int","default":0}]}`

	_, err := reg.RegisterSchema(ctx, ".", "test-subject", schema1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema v1: %v", err)
	}

	_, err = reg.RegisterSchema(ctx, ".", "test-subject", schema2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema v2: %v", err)
	}

	_, err = reg.RegisterSchema(ctx, ".", "test-subject", schema3, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema v3: %v", err)
	}

	// Test: Check compatibility against explicit version 1
	// This schema is backward compatible with v1 (adds field with default)
	newSchema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"extra","type":"string","default":""}]}`

	result, err := reg.CheckCompatibility(ctx, ".", "test-subject", newSchema, storage.SchemaTypeAvro, nil, "1")
	if err != nil {
		t.Fatalf("failed to check compatibility: %v", err)
	}

	if !result.IsCompatible {
		t.Error("expected schema to be compatible with version 1")
	}

	// Test: Check compatibility against "latest" (version 3)
	result, err = reg.CheckCompatibility(ctx, ".", "test-subject", newSchema, storage.SchemaTypeAvro, nil, "latest")
	if err != nil {
		t.Fatalf("failed to check compatibility with latest: %v", err)
	}

	if !result.IsCompatible {
		t.Error("expected schema to be compatible with latest version")
	}

	// Test: Check compatibility with empty version (all versions)
	result, err = reg.CheckCompatibility(ctx, ".", "test-subject", newSchema, storage.SchemaTypeAvro, nil, "")
	if err != nil {
		t.Fatalf("failed to check compatibility with all versions: %v", err)
	}

	if !result.IsCompatible {
		t.Error("expected schema to be compatible with all versions")
	}
}

func TestCheckCompatibility_InvalidVersion(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")

	ctx := context.Background()

	// Register a schema
	schema1 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", schema1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema: %v", err)
	}

	// Test: Check compatibility against non-existent version
	newSchema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}`

	_, err = reg.CheckCompatibility(ctx, ".", "test-subject", newSchema, storage.SchemaTypeAvro, nil, "999")
	if err == nil {
		t.Error("expected error for non-existent version")
	}

	// Test: Check compatibility against invalid version string
	_, err = reg.CheckCompatibility(ctx, ".", "test-subject", newSchema, storage.SchemaTypeAvro, nil, "invalid")
	if err == nil {
		t.Error("expected error for invalid version string")
	}
}

func TestCheckCompatibility_NoExistingSchemas(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`

	// Latest on non-existent subject
	result, err := reg.CheckCompatibility(ctx, ".", "nonexistent", schema, storage.SchemaTypeAvro, nil, "latest")
	if err != nil {
		t.Fatalf("should succeed for non-existent subject: %v", err)
	}
	if !result.IsCompatible {
		t.Error("should be compatible when no existing schemas")
	}

	// All versions on non-existent subject
	result, err = reg.CheckCompatibility(ctx, ".", "nonexistent", schema, storage.SchemaTypeAvro, nil, "")
	if err != nil {
		t.Fatalf("should succeed for non-existent subject: %v", err)
	}
	if !result.IsCompatible {
		t.Error("should be compatible when no existing schemas")
	}
}

func TestCheckCompatibility_ModeNone(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	s := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", s, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// With NONE mode, always compatible
	incompatible := `{"type":"record","name":"Test","fields":[{"name":"name","type":"string"}]}`
	result, err := reg.CheckCompatibility(ctx, ".", "test-subject", incompatible, storage.SchemaTypeAvro, nil, "latest")
	if err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if !result.IsCompatible {
		t.Error("NONE mode should always be compatible")
	}
}

func TestCheckCompatibility_UnsupportedType(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	_, err := reg.CheckCompatibility(ctx, ".", "test", "{}", "UNKNOWN", nil, "latest")
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestCheckCompatibility_InvalidSchema(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	_, err := reg.CheckCompatibility(ctx, ".", "test", "invalid", storage.SchemaTypeAvro, nil, "latest")
	if err == nil {
		t.Error("expected error for invalid schema")
	}
}

func TestCheckCompatibility_DefaultsToAvro(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	s := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", s, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Empty schema type should default to Avro
	newSchema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"f","type":"string","default":""}]}`
	result, err := reg.CheckCompatibility(ctx, ".", "test-subject", newSchema, "", nil, "latest")
	if err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if !result.IsCompatible {
		t.Error("expected compatible")
	}
}

// --- GetReferencedBy tests ---

func TestGetReferencedBy(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register a base schema
	baseSchema := `{"type":"record","name":"Base","namespace":"test","fields":[{"name":"id","type":"int"}]}`
	baseRecord, err := reg.RegisterSchema(ctx, ".", "base-subject", baseSchema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register base schema: %v", err)
	}

	// Register a schema that references the base schema
	referencingSchema := `{"type":"record","name":"Referencing","namespace":"test","fields":[{"name":"base","type":"test.Base"}]}`
	refs := []storage.Reference{
		{Name: "test.Base", Subject: "base-subject", Version: 1},
	}
	refRecord, err := reg.RegisterSchema(ctx, ".", "referencing-subject", referencingSchema, storage.SchemaTypeAvro, refs)
	if err != nil {
		t.Fatalf("failed to register referencing schema: %v", err)
	}

	// Get schemas that reference base-subject version 1
	referencedBy, err := reg.GetReferencedBy(ctx, ".", "base-subject", 1)
	if err != nil {
		t.Fatalf("failed to get referenced by: %v", err)
	}

	if len(referencedBy) != 1 {
		t.Errorf("expected 1 reference, got %d", len(referencedBy))
	}

	if len(referencedBy) > 0 {
		if referencedBy[0].Subject != "referencing-subject" {
			t.Errorf("expected subject 'referencing-subject', got %q", referencedBy[0].Subject)
		}
		if referencedBy[0].Version != 1 {
			t.Errorf("expected version 1, got %d", referencedBy[0].Version)
		}
	}

	// Verify we can get the referencing schema ID from the registry
	refSchema, err := reg.GetSchemaBySubjectVersion(ctx, ".", "referencing-subject", 1)
	if err != nil {
		t.Fatalf("failed to get referencing schema: %v", err)
	}
	if refSchema.ID != refRecord.ID {
		t.Errorf("expected ID %d, got %d", refRecord.ID, refSchema.ID)
	}

	// Verify base schema ID is different
	if baseRecord.ID == refRecord.ID {
		t.Error("base and referencing schemas should have different IDs")
	}
}

func TestGetReferencedBy_NoReferences(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register a schema without references
	schema := `{"type":"record","name":"NoRefs","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "no-refs-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema: %v", err)
	}

	// Get schemas that reference this schema (should be empty)
	referencedBy, err := reg.GetReferencedBy(ctx, ".", "no-refs-subject", 1)
	if err != nil {
		t.Fatalf("failed to get referenced by: %v", err)
	}

	if len(referencedBy) != 0 {
		t.Errorf("expected 0 references, got %d", len(referencedBy))
	}
}

// --- Version number tests ---

func TestVersionNumbers_MonotonicallyIncreasing(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register three schemas (same record name, different optional fields for BACKWARD compatibility)
	schema1 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"}]}`
	schema2 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"},{"name":"f2","type":"string","default":""}]}`
	schema3 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"},{"name":"f2","type":"string","default":""},{"name":"f3","type":"int","default":0}]}`

	rec1, err := reg.RegisterSchema(ctx, ".", "version-test", schema1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema 1: %v", err)
	}
	if rec1.Version != 1 {
		t.Errorf("expected version 1, got %d", rec1.Version)
	}

	rec2, err := reg.RegisterSchema(ctx, ".", "version-test", schema2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema 2: %v", err)
	}
	if rec2.Version != 2 {
		t.Errorf("expected version 2, got %d", rec2.Version)
	}

	rec3, err := reg.RegisterSchema(ctx, ".", "version-test", schema3, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema 3: %v", err)
	}
	if rec3.Version != 3 {
		t.Errorf("expected version 3, got %d", rec3.Version)
	}

	// Delete version 2
	_, err = reg.DeleteVersion(ctx, ".", "version-test", 2, false)
	if err != nil {
		t.Fatalf("failed to delete version 2: %v", err)
	}

	// Register a new schema - should be version 4, not version 2
	schema4 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"},{"name":"f2","type":"string","default":""},{"name":"f3","type":"int","default":0},{"name":"f4","type":"long","default":0}]}`
	rec4, err := reg.RegisterSchema(ctx, ".", "version-test", schema4, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema 4: %v", err)
	}
	if rec4.Version != 4 {
		t.Errorf("expected version 4 (monotonically increasing), got %d", rec4.Version)
	}

	// Delete entire subject
	_, err = reg.DeleteSubject(ctx, ".", "version-test", false)
	if err != nil {
		t.Fatalf("failed to delete subject: %v", err)
	}

	// Re-register a schema - should be version 5, not version 1
	schema5 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"},{"name":"newfield","type":"string","default":""}]}`
	rec5, err := reg.RegisterSchema(ctx, ".", "version-test", schema5, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema 5: %v", err)
	}
	if rec5.Version != 5 {
		t.Errorf("expected version 5 (monotonically increasing after subject delete), got %d", rec5.Version)
	}
}

func TestVersionNumbers_IndependentSubjects(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register schemas in two different subjects (each has its own record name)
	schemaA := `{"type":"record","name":"RecordA","fields":[{"name":"id","type":"int"}]}`
	schemaB := `{"type":"record","name":"RecordB","fields":[{"name":"id","type":"int"}]}`

	recA1, err := reg.RegisterSchema(ctx, ".", "subject-a", schemaA, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema A1: %v", err)
	}
	if recA1.Version != 1 {
		t.Errorf("subject-a: expected version 1, got %d", recA1.Version)
	}

	recB1, err := reg.RegisterSchema(ctx, ".", "subject-b", schemaB, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema B1: %v", err)
	}
	if recB1.Version != 1 {
		t.Errorf("subject-b: expected version 1, got %d", recB1.Version)
	}

	// Register more schemas in subject-a (backward compatible - add field with default)
	schemaA2 := `{"type":"record","name":"RecordA","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}`
	recA2, err := reg.RegisterSchema(ctx, ".", "subject-a", schemaA2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema A2: %v", err)
	}
	if recA2.Version != 2 {
		t.Errorf("subject-a: expected version 2, got %d", recA2.Version)
	}

	// subject-b should still get version 2, not affected by subject-a
	schemaB2 := `{"type":"record","name":"RecordB","fields":[{"name":"id","type":"int"},{"name":"desc","type":"string","default":""}]}`
	recB2, err := reg.RegisterSchema(ctx, ".", "subject-b", schemaB2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema B2: %v", err)
	}
	if recB2.Version != 2 {
		t.Errorf("subject-b: expected version 2, got %d", recB2.Version)
	}
}

// --- Helper function tests ---

func TestIsValidCompatibility(t *testing.T) {
	valid := []string{"NONE", "BACKWARD", "BACKWARD_TRANSITIVE", "FORWARD", "FORWARD_TRANSITIVE", "FULL", "FULL_TRANSITIVE"}
	for _, v := range valid {
		if !isValidCompatibility(v) {
			t.Errorf("expected %s to be valid", v)
		}
	}

	invalid := []string{"", "backward", "INVALID", "BACKWARDS", "TRANSITIVE"}
	for _, v := range invalid {
		if isValidCompatibility(v) {
			t.Errorf("expected %s to be invalid", v)
		}
	}
}

func TestIsValidMode(t *testing.T) {
	valid := []string{"READWRITE", "READONLY", "IMPORT"}
	for _, v := range valid {
		if !isValidMode(v) {
			t.Errorf("expected %s to be valid", v)
		}
	}

	invalid := []string{"", "readwrite", "WRITE", "READ"}
	for _, v := range invalid {
		if isValidMode(v) {
			t.Errorf("expected %s to be invalid", v)
		}
	}
}

// --- Reference Resolution tests ---

func TestResolveReferences_EmptyRefs(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	resolved, err := reg.resolveReferences(ctx, ".", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != nil {
		t.Errorf("expected nil for nil refs, got %v", resolved)
	}

	resolved, err = reg.resolveReferences(ctx, ".", []storage.Reference{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected empty for empty refs, got %d", len(resolved))
	}
}

func TestResolveReferences_NonExistentSubject(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	refs := []storage.Reference{
		{Name: "missing.avsc", Subject: "nonexistent-subject", Version: 1},
	}

	_, err := reg.resolveReferences(ctx, ".", refs)
	if err == nil {
		t.Error("expected error when resolving reference to non-existent subject")
	}
	if !strings.Contains(err.Error(), "nonexistent-subject") {
		t.Errorf("error should mention subject name, got: %v", err)
	}
}

func TestResolveReferences_PopulatesSchemaContent(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register a base schema
	baseSchema := `{"type":"record","name":"Base","namespace":"test","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "base-subject", baseSchema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register base: %v", err)
	}

	// Resolve a reference to the base
	refs := []storage.Reference{
		{Name: "test.Base", Subject: "base-subject", Version: 1},
	}
	resolved, err := reg.resolveReferences(ctx, ".", refs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved ref, got %d", len(resolved))
	}
	if resolved[0].Name != "test.Base" {
		t.Errorf("expected name 'test.Base', got %q", resolved[0].Name)
	}
	if resolved[0].Subject != "base-subject" {
		t.Errorf("expected subject 'base-subject', got %q", resolved[0].Subject)
	}
	if resolved[0].Schema == "" {
		t.Error("expected Schema content to be populated")
	}
	if resolved[0].Schema != baseSchema {
		t.Errorf("expected resolved schema to match base, got %q", resolved[0].Schema)
	}
}

func TestResolveReferences_MultipleRefs(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register two base schemas
	base1 := `{"type":"record","name":"Base1","namespace":"multi","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "multi-base1", base1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register base1: %v", err)
	}

	base2 := `{"type":"record","name":"Base2","namespace":"multi","fields":[{"name":"name","type":"string"}]}`
	_, err = reg.RegisterSchema(ctx, ".", "multi-base2", base2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register base2: %v", err)
	}

	// Resolve both references
	refs := []storage.Reference{
		{Name: "multi.Base1", Subject: "multi-base1", Version: 1},
		{Name: "multi.Base2", Subject: "multi-base2", Version: 1},
	}
	resolved, err := reg.resolveReferences(ctx, ".", refs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved refs, got %d", len(resolved))
	}
	if resolved[0].Schema != base1 {
		t.Errorf("first ref content mismatch")
	}
	if resolved[1].Schema != base2 {
		t.Errorf("second ref content mismatch")
	}
}

func TestRegisterSchema_WithAvroReferences(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register a base schema
	baseSchema := `{"type":"record","name":"Base","namespace":"com.example","fields":[{"name":"id","type":"int"}]}`
	baseRec, err := reg.RegisterSchema(ctx, ".", "base-subject", baseSchema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register base: %v", err)
	}

	// Register a schema that references the base (uses the named type)
	referencingSchema := `{"type":"record","name":"Wrapper","namespace":"com.example","fields":[{"name":"base","type":"com.example.Base"}]}`
	refs := []storage.Reference{
		{Name: "com.example.Base", Subject: "base-subject", Version: 1},
	}
	refRec, err := reg.RegisterSchema(ctx, ".", "wrapper-subject", referencingSchema, storage.SchemaTypeAvro, refs)
	if err != nil {
		t.Fatalf("failed to register referencing schema: %v", err)
	}

	if refRec.ID == baseRec.ID {
		t.Error("referencing schema should have a different ID from base")
	}
	if refRec.Version != 1 {
		t.Errorf("expected version 1, got %d", refRec.Version)
	}

	// Verify stored references
	stored, err := reg.GetSchemaBySubjectVersion(ctx, ".", "wrapper-subject", 1)
	if err != nil {
		t.Fatalf("failed to get stored schema: %v", err)
	}
	if len(stored.References) != 1 {
		t.Fatalf("expected 1 stored reference, got %d", len(stored.References))
	}
	if stored.References[0].Subject != "base-subject" {
		t.Errorf("expected ref to base-subject, got %q", stored.References[0].Subject)
	}
}

func TestRegisterSchema_WithReferences_FailsOnMissingRef(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	refs := []storage.Reference{
		{Name: "missing", Subject: "nonexistent", Version: 1},
	}

	_, err := reg.RegisterSchema(ctx, ".", "test-subject", schema, storage.SchemaTypeAvro, refs)
	if err == nil {
		t.Error("expected error when reference subject doesn't exist")
	}
	if !strings.Contains(err.Error(), "resolve references") {
		t.Errorf("expected 'resolve references' in error, got: %v", err)
	}
}

func TestRegisterSchema_WithProtobufReferences(t *testing.T) {
	reg := setupMultiTypeRegistry("NONE")
	ctx := context.Background()

	// Register a base protobuf schema
	baseSchema := `syntax = "proto3";
package common;
message Address {
  string street = 1;
  string city = 2;
}`
	_, err := reg.RegisterSchema(ctx, ".", "common-address", baseSchema, storage.SchemaTypeProtobuf, nil)
	if err != nil {
		t.Fatalf("failed to register base protobuf: %v", err)
	}

	// Register a schema that imports the base
	referencingSchema := `syntax = "proto3";
import "common/address.proto";
package user;
message User {
  string name = 1;
  common.Address address = 2;
}`
	refs := []storage.Reference{
		{Name: "common/address.proto", Subject: "common-address", Version: 1},
	}
	rec, err := reg.RegisterSchema(ctx, ".", "user-subject", referencingSchema, storage.SchemaTypeProtobuf, refs)
	if err != nil {
		t.Fatalf("failed to register referencing protobuf: %v", err)
	}
	if rec.Version != 1 {
		t.Errorf("expected version 1, got %d", rec.Version)
	}
}

func TestRegisterSchema_WithJSONSchemaReferences(t *testing.T) {
	reg := setupMultiTypeRegistry("NONE")
	ctx := context.Background()

	// Register a base JSON schema
	baseSchema := `{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"]}`
	_, err := reg.RegisterSchema(ctx, ".", "base-json", baseSchema, storage.SchemaTypeJSON, nil)
	if err != nil {
		t.Fatalf("failed to register base JSON schema: %v", err)
	}

	// Register a schema with $ref to the base
	referencingSchema := `{"type":"object","properties":{"user":{"$ref":"base.json"}}}`
	refs := []storage.Reference{
		{Name: "base.json", Subject: "base-json", Version: 1},
	}
	rec, err := reg.RegisterSchema(ctx, ".", "wrapper-json", referencingSchema, storage.SchemaTypeJSON, refs)
	if err != nil {
		t.Fatalf("failed to register referencing JSON schema: %v", err)
	}
	if rec.Version != 1 {
		t.Errorf("expected version 1, got %d", rec.Version)
	}
}

func TestLookupSchema_WithReferences(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register base
	baseSchema := `{"type":"record","name":"LookupBase","namespace":"lookup","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "lookup-base", baseSchema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register base: %v", err)
	}

	// Register referencing schema
	referencingSchema := `{"type":"record","name":"LookupRef","namespace":"lookup","fields":[{"name":"base","type":"lookup.LookupBase"}]}`
	refs := []storage.Reference{
		{Name: "lookup.LookupBase", Subject: "lookup-base", Version: 1},
	}
	registered, err := reg.RegisterSchema(ctx, ".", "lookup-ref", referencingSchema, storage.SchemaTypeAvro, refs)
	if err != nil {
		t.Fatalf("failed to register referencing: %v", err)
	}

	// Lookup the referencing schema with references
	found, err := reg.LookupSchema(ctx, ".", "lookup-ref", referencingSchema, storage.SchemaTypeAvro, refs, false)
	if err != nil {
		t.Fatalf("failed to lookup: %v", err)
	}
	if found.ID != registered.ID {
		t.Errorf("expected ID %d, got %d", registered.ID, found.ID)
	}
}

func TestLookupSchema_WithReferences_FailsOnMissingRef(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	refs := []storage.Reference{
		{Name: "missing", Subject: "nonexistent", Version: 1},
	}

	_, err := reg.LookupSchema(ctx, ".", "test", schema, storage.SchemaTypeAvro, refs, false)
	if err == nil {
		t.Error("expected error when reference can't be resolved")
	}
}

func TestCheckCompatibility_WithReferences(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	// Register base schema
	baseSchema := `{"type":"record","name":"CompatBase","namespace":"compat","fields":[{"name":"id","type":"int"}]}`
	_, err := reg.RegisterSchema(ctx, ".", "compat-base", baseSchema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register base: %v", err)
	}

	// Register v1 that references the base
	v1Schema := `{"type":"record","name":"CompatRef","namespace":"compat","fields":[{"name":"base","type":"compat.CompatBase"}]}`
	refs := []storage.Reference{
		{Name: "compat.CompatBase", Subject: "compat-base", Version: 1},
	}
	_, err = reg.RegisterSchema(ctx, ".", "compat-ref", v1Schema, storage.SchemaTypeAvro, refs)
	if err != nil {
		t.Fatalf("failed to register v1: %v", err)
	}

	// Check compatibility of a new schema version (backward compatible - adds field with default)
	v2Schema := `{"type":"record","name":"CompatRef","namespace":"compat","fields":[{"name":"base","type":"compat.CompatBase"},{"name":"extra","type":"string","default":""}]}`
	result, err := reg.CheckCompatibility(ctx, ".", "compat-ref", v2Schema, storage.SchemaTypeAvro, refs, "latest")
	if err != nil {
		t.Fatalf("failed to check compatibility: %v", err)
	}
	if !result.IsCompatible {
		t.Error("expected schema to be backward compatible")
	}
}

func TestCheckCompatibility_WithReferences_FailsOnMissingRef(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")
	ctx := context.Background()

	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	refs := []storage.Reference{
		{Name: "missing", Subject: "nonexistent", Version: 1},
	}

	_, err := reg.CheckCompatibility(ctx, ".", "test", schema, storage.SchemaTypeAvro, refs, "latest")
	if err == nil {
		t.Error("expected error when reference can't be resolved")
	}
}

func TestImportSchemas_WithReferences(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// First import the base schema
	baseSchemas := []ImportSchemaRequest{
		{
			ID:         100,
			Subject:    "import-base",
			Version:    1,
			SchemaType: storage.SchemaTypeAvro,
			Schema:     `{"type":"record","name":"ImportBase","namespace":"imp","fields":[{"name":"id","type":"int"}]}`,
		},
	}
	result, err := reg.ImportSchemas(ctx, ".", baseSchemas)
	if err != nil {
		t.Fatalf("failed to import base: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("expected 1 imported, got %d", result.Imported)
	}

	// Now import a schema with a reference to the base
	refSchemas := []ImportSchemaRequest{
		{
			ID:         101,
			Subject:    "import-ref",
			Version:    1,
			SchemaType: storage.SchemaTypeAvro,
			Schema:     `{"type":"record","name":"ImportRef","namespace":"imp","fields":[{"name":"base","type":"imp.ImportBase"}]}`,
			References: []storage.Reference{
				{Name: "imp.ImportBase", Subject: "import-base", Version: 1},
			},
		},
	}
	result, err = reg.ImportSchemas(ctx, ".", refSchemas)
	if err != nil {
		t.Fatalf("failed to import referencing schema: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("expected 1 imported, got %d (errors: %d)", result.Imported, result.Errors)
	}

	// Verify stored schema
	stored, err := reg.GetSchemaBySubjectVersion(ctx, ".", "import-ref", 1)
	if err != nil {
		t.Fatalf("failed to get imported schema: %v", err)
	}
	if stored.ID != 101 {
		t.Errorf("expected ID 101, got %d", stored.ID)
	}
	if len(stored.References) != 1 {
		t.Errorf("expected 1 reference, got %d", len(stored.References))
	}
}

func TestImportSchemas_WithReferences_FailsOnMissingRef(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	schemas := []ImportSchemaRequest{
		{
			ID:         1,
			Subject:    "test",
			Version:    1,
			SchemaType: storage.SchemaTypeAvro,
			Schema:     `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`,
			References: []storage.Reference{
				{Name: "missing", Subject: "nonexistent", Version: 1},
			},
		},
	}

	result, err := reg.ImportSchemas(ctx, ".", schemas)
	if err != nil {
		t.Fatalf("import should not return top-level error: %v", err)
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
	if result.Imported != 0 {
		t.Errorf("expected 0 imported, got %d", result.Imported)
	}
	if !strings.Contains(result.Results[0].Error, "resolve references") {
		t.Errorf("expected resolve references error, got: %s", result.Results[0].Error)
	}
}

// --- RegisterSchemaWithID + SetNextID failure tests ---

// failSetNextIDStore wraps a real memory store but makes SetNextID always fail.
type failSetNextIDStore struct {
	*memory.Store
}

func (f *failSetNextIDStore) SetNextID(_ context.Context, _ string, _ int64) error {
	return errors.New("injected SetNextID failure")
}

func TestRegisterSchemaWithID_SetNextIDFailure(t *testing.T) {
	underlying := memory.NewStore()
	underlying.SetGlobalConfig(context.Background(), ".", &storage.ConfigRecord{CompatibilityLevel: "NONE"})

	// Put registry in IMPORT mode
	underlying.SetGlobalMode(context.Background(), ".", &storage.ModeRecord{Mode: "IMPORT"})

	store := &failSetNextIDStore{Store: underlying}

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())

	reg := New(store, schemaRegistry, compatChecker, "NONE")

	ctx := context.Background()
	schemaStr := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`

	record, err := reg.RegisterSchemaWithID(ctx, ".", "test-subject", schemaStr, storage.SchemaTypeAvro, nil, 100)
	if err == nil {
		t.Fatal("expected error when SetNextID fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to advance ID sequence") {
		t.Errorf("expected 'failed to advance ID sequence' error, got: %v", err)
	}
	// The schema record should still be returned (schema was stored successfully)
	if record == nil {
		t.Fatal("expected non-nil record even when SetNextID fails (schema was stored)")
	}
	if record.ID != 100 {
		t.Errorf("expected stored schema ID 100, got %d", record.ID)
	}
}

func TestImportSchemas_SetNextIDFailure(t *testing.T) {
	underlying := memory.NewStore()
	underlying.SetGlobalConfig(context.Background(), ".", &storage.ConfigRecord{CompatibilityLevel: "NONE"})
	underlying.SetGlobalMode(context.Background(), ".", &storage.ModeRecord{Mode: "IMPORT"})

	store := &failSetNextIDStore{Store: underlying}

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())

	reg := New(store, schemaRegistry, compatChecker, "NONE")

	ctx := context.Background()
	schemas := []ImportSchemaRequest{
		{
			ID:         200,
			Subject:    "import-test",
			Version:    1,
			SchemaType: storage.SchemaTypeAvro,
			Schema:     `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`,
		},
	}

	result, err := reg.ImportSchemas(ctx, ".", schemas)
	if err == nil {
		t.Fatal("expected error when SetNextID fails during ImportSchemas")
	}
	if !strings.Contains(err.Error(), "failed to adjust ID sequence") {
		t.Errorf("expected 'failed to adjust ID sequence' error, got: %v", err)
	}
	// Schemas were imported successfully before SetNextID failed
	if result.Imported != 1 {
		t.Errorf("expected 1 imported schema, got %d", result.Imported)
	}
}
