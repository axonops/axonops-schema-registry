package registry

import (
	"context"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

// setupTestRegistry creates a test registry with memory storage and Avro support.
func setupTestRegistry(defaultCompatibility string) *Registry {
	store := memory.NewStore()

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())

	return New(store, schemaRegistry, compatChecker, defaultCompatibility)
}

func TestCheckCompatibility_ExplicitVersion(t *testing.T) {
	reg := setupTestRegistry("BACKWARD")

	ctx := context.Background()

	// Register three versions of a schema
	schema1 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	schema2 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}`
	schema3 := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"age","type":"int","default":0}]}`

	_, err := reg.RegisterSchema(ctx, "test-subject", schema1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema v1: %v", err)
	}

	_, err = reg.RegisterSchema(ctx, "test-subject", schema2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema v2: %v", err)
	}

	_, err = reg.RegisterSchema(ctx, "test-subject", schema3, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema v3: %v", err)
	}

	// Test: Check compatibility against explicit version 1
	// This schema is backward compatible with v1 (adds field with default)
	newSchema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"extra","type":"string","default":""}]}`

	result, err := reg.CheckCompatibility(ctx, "test-subject", newSchema, storage.SchemaTypeAvro, nil, "1")
	if err != nil {
		t.Fatalf("failed to check compatibility: %v", err)
	}

	if !result.IsCompatible {
		t.Error("expected schema to be compatible with version 1")
	}

	// Test: Check compatibility against "latest" (version 3)
	result, err = reg.CheckCompatibility(ctx, "test-subject", newSchema, storage.SchemaTypeAvro, nil, "latest")
	if err != nil {
		t.Fatalf("failed to check compatibility with latest: %v", err)
	}

	if !result.IsCompatible {
		t.Error("expected schema to be compatible with latest version")
	}

	// Test: Check compatibility with empty version (all versions)
	result, err = reg.CheckCompatibility(ctx, "test-subject", newSchema, storage.SchemaTypeAvro, nil, "")
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
	_, err := reg.RegisterSchema(ctx, "test-subject", schema1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema: %v", err)
	}

	// Test: Check compatibility against non-existent version
	newSchema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}`

	_, err = reg.CheckCompatibility(ctx, "test-subject", newSchema, storage.SchemaTypeAvro, nil, "999")
	if err == nil {
		t.Error("expected error for non-existent version")
	}

	// Test: Check compatibility against invalid version string
	_, err = reg.CheckCompatibility(ctx, "test-subject", newSchema, storage.SchemaTypeAvro, nil, "invalid")
	if err == nil {
		t.Error("expected error for invalid version string")
	}
}

func TestLookupSchema_WithDeleted(t *testing.T) {
	reg := setupTestRegistry("NONE")

	ctx := context.Background()

	// Register a schema
	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`
	record, err := reg.RegisterSchema(ctx, "test-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema: %v", err)
	}

	// Verify lookup works before delete
	found, err := reg.LookupSchema(ctx, "test-subject", schema, storage.SchemaTypeAvro, nil, false)
	if err != nil {
		t.Fatalf("failed to lookup schema: %v", err)
	}
	if found.ID != record.ID {
		t.Errorf("expected ID %d, got %d", record.ID, found.ID)
	}

	// Soft delete the subject
	_, err = reg.DeleteSubject(ctx, "test-subject", false)
	if err != nil {
		t.Fatalf("failed to soft delete subject: %v", err)
	}

	// Lookup without deleted flag should fail
	_, err = reg.LookupSchema(ctx, "test-subject", schema, storage.SchemaTypeAvro, nil, false)
	if err == nil {
		t.Error("expected error when looking up deleted schema without deleted flag")
	}

	// Lookup with deleted flag should succeed
	found, err = reg.LookupSchema(ctx, "test-subject", schema, storage.SchemaTypeAvro, nil, true)
	if err != nil {
		t.Fatalf("failed to lookup deleted schema: %v", err)
	}
	if found.ID != record.ID {
		t.Errorf("expected ID %d, got %d", record.ID, found.ID)
	}
}

func TestGetReferencedBy(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register a base schema
	baseSchema := `{"type":"record","name":"Base","namespace":"test","fields":[{"name":"id","type":"int"}]}`
	baseRecord, err := reg.RegisterSchema(ctx, "base-subject", baseSchema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register base schema: %v", err)
	}

	// Register a schema that references the base schema
	referencingSchema := `{"type":"record","name":"Referencing","namespace":"test","fields":[{"name":"base","type":"test.Base"}]}`
	refs := []storage.Reference{
		{Name: "test.Base", Subject: "base-subject", Version: 1},
	}
	refRecord, err := reg.RegisterSchema(ctx, "referencing-subject", referencingSchema, storage.SchemaTypeAvro, refs)
	if err != nil {
		t.Fatalf("failed to register referencing schema: %v", err)
	}

	// Get schemas that reference base-subject version 1
	referencedBy, err := reg.GetReferencedBy(ctx, "base-subject", 1)
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
	refSchema, err := reg.GetSchemaBySubjectVersion(ctx, "referencing-subject", 1)
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
	_, err := reg.RegisterSchema(ctx, "no-refs-subject", schema, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema: %v", err)
	}

	// Get schemas that reference this schema (should be empty)
	referencedBy, err := reg.GetReferencedBy(ctx, "no-refs-subject", 1)
	if err != nil {
		t.Fatalf("failed to get referenced by: %v", err)
	}

	if len(referencedBy) != 0 {
		t.Errorf("expected 0 references, got %d", len(referencedBy))
	}
}

func TestVersionNumbers_MonotonicallyIncreasing(t *testing.T) {
	reg := setupTestRegistry("NONE")
	ctx := context.Background()

	// Register three schemas (same record name, different optional fields for BACKWARD compatibility)
	schema1 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"}]}`
	schema2 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"},{"name":"f2","type":"string","default":""}]}`
	schema3 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"},{"name":"f2","type":"string","default":""},{"name":"f3","type":"int","default":0}]}`

	rec1, err := reg.RegisterSchema(ctx, "version-test", schema1, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema 1: %v", err)
	}
	if rec1.Version != 1 {
		t.Errorf("expected version 1, got %d", rec1.Version)
	}

	rec2, err := reg.RegisterSchema(ctx, "version-test", schema2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema 2: %v", err)
	}
	if rec2.Version != 2 {
		t.Errorf("expected version 2, got %d", rec2.Version)
	}

	rec3, err := reg.RegisterSchema(ctx, "version-test", schema3, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema 3: %v", err)
	}
	if rec3.Version != 3 {
		t.Errorf("expected version 3, got %d", rec3.Version)
	}

	// Delete version 2
	_, err = reg.DeleteVersion(ctx, "version-test", 2, false)
	if err != nil {
		t.Fatalf("failed to delete version 2: %v", err)
	}

	// Register a new schema - should be version 4, not version 2
	schema4 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"},{"name":"f2","type":"string","default":""},{"name":"f3","type":"int","default":0},{"name":"f4","type":"long","default":0}]}`
	rec4, err := reg.RegisterSchema(ctx, "version-test", schema4, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema 4: %v", err)
	}
	if rec4.Version != 4 {
		t.Errorf("expected version 4 (monotonically increasing), got %d", rec4.Version)
	}

	// Delete entire subject
	_, err = reg.DeleteSubject(ctx, "version-test", false)
	if err != nil {
		t.Fatalf("failed to delete subject: %v", err)
	}

	// Re-register a schema - should be version 5, not version 1
	schema5 := `{"type":"record","name":"MonoTest","fields":[{"name":"id","type":"int"},{"name":"newfield","type":"string","default":""}]}`
	rec5, err := reg.RegisterSchema(ctx, "version-test", schema5, storage.SchemaTypeAvro, nil)
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

	recA1, err := reg.RegisterSchema(ctx, "subject-a", schemaA, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema A1: %v", err)
	}
	if recA1.Version != 1 {
		t.Errorf("subject-a: expected version 1, got %d", recA1.Version)
	}

	recB1, err := reg.RegisterSchema(ctx, "subject-b", schemaB, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema B1: %v", err)
	}
	if recB1.Version != 1 {
		t.Errorf("subject-b: expected version 1, got %d", recB1.Version)
	}

	// Register more schemas in subject-a (backward compatible - add field with default)
	schemaA2 := `{"type":"record","name":"RecordA","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}`
	recA2, err := reg.RegisterSchema(ctx, "subject-a", schemaA2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema A2: %v", err)
	}
	if recA2.Version != 2 {
		t.Errorf("subject-a: expected version 2, got %d", recA2.Version)
	}

	// subject-b should still get version 2, not affected by subject-a
	schemaB2 := `{"type":"record","name":"RecordB","fields":[{"name":"id","type":"int"},{"name":"desc","type":"string","default":""}]}`
	recB2, err := reg.RegisterSchema(ctx, "subject-b", schemaB2, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("failed to register schema B2: %v", err)
	}
	if recB2.Version != 2 {
		t.Errorf("subject-b: expected version 2, got %d", recB2.Version)
	}
}
