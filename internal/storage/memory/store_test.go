package memory

import (
	"context"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func TestStore_CreateAndGetSchema(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	record := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "abc123",
	}

	err := store.CreateSchema(ctx, record)
	if err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}

	if record.ID == 0 {
		t.Error("Expected schema ID to be set")
	}
	if record.Version != 1 {
		t.Errorf("Expected version 1, got %d", record.Version)
	}

	// Get by ID
	got, err := store.GetSchemaByID(ctx, record.ID)
	if err != nil {
		t.Fatalf("GetSchemaByID failed: %v", err)
	}
	if got.Schema != record.Schema {
		t.Errorf("Schema mismatch: got %s, want %s", got.Schema, record.Schema)
	}

	// Get by subject/version
	got, err = store.GetSchemaBySubjectVersion(ctx, "test-subject", 1)
	if err != nil {
		t.Fatalf("GetSchemaBySubjectVersion failed: %v", err)
	}
	if got.ID != record.ID {
		t.Errorf("ID mismatch: got %d, want %d", got.ID, record.ID)
	}
}

func TestStore_DuplicateSchema(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	record1 := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "abc123",
	}

	err := store.CreateSchema(ctx, record1)
	if err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}

	// Try to create duplicate
	record2 := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "abc123",
	}

	err = store.CreateSchema(ctx, record2)
	if err != storage.ErrSchemaExists {
		t.Errorf("Expected ErrSchemaExists, got %v", err)
	}

	// Should have same ID as original
	if record2.ID != record1.ID {
		t.Errorf("Expected same ID %d, got %d", record1.ID, record2.ID)
	}
}

func TestStore_MultipleVersions(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Create version 1
	record1 := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp1",
	}
	if err := store.CreateSchema(ctx, record1); err != nil {
		t.Fatalf("CreateSchema v1 failed: %v", err)
	}

	// Create version 2
	record2 := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp2",
	}
	if err := store.CreateSchema(ctx, record2); err != nil {
		t.Fatalf("CreateSchema v2 failed: %v", err)
	}

	if record2.Version != 2 {
		t.Errorf("Expected version 2, got %d", record2.Version)
	}

	// Get latest
	latest, err := store.GetLatestSchema(ctx, "test-subject")
	if err != nil {
		t.Fatalf("GetLatestSchema failed: %v", err)
	}
	if latest.Version != 2 {
		t.Errorf("Expected latest version 2, got %d", latest.Version)
	}
}

func TestStore_ListSubjects(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Create schemas for multiple subjects
	subjects := []string{"subject-a", "subject-b", "subject-c"}
	for i, subj := range subjects {
		record := &storage.SchemaRecord{
			Subject:     subj,
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type": "string"}`,
			Fingerprint: string(rune('a' + i)),
		}
		if err := store.CreateSchema(ctx, record); err != nil {
			t.Fatalf("CreateSchema failed: %v", err)
		}
	}

	got, err := store.ListSubjects(ctx, false)
	if err != nil {
		t.Fatalf("ListSubjects failed: %v", err)
	}

	if len(got) != len(subjects) {
		t.Errorf("Expected %d subjects, got %d", len(subjects), len(got))
	}
}

func TestStore_DeleteSubject(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	record := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "abc123",
	}

	if err := store.CreateSchema(ctx, record); err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}

	// Soft delete
	versions, err := store.DeleteSubject(ctx, "test-subject", false)
	if err != nil {
		t.Fatalf("DeleteSubject failed: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("Expected 1 deleted version, got %d", len(versions))
	}

	// Subject should not appear in list
	subjects, _ := store.ListSubjects(ctx, false)
	if len(subjects) != 0 {
		t.Errorf("Expected 0 subjects, got %d", len(subjects))
	}

	// But should appear with deleted=true
	subjects, _ = store.ListSubjects(ctx, true)
	if len(subjects) != 1 {
		t.Errorf("Expected 1 subject with deleted=true, got %d", len(subjects))
	}
}

func TestStore_Config(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Get global config (no default — returns ErrNotFound)
	_, err := store.GetGlobalConfig(ctx)
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound for unset global config, got %v", err)
	}

	// Set global config
	err = store.SetGlobalConfig(ctx, &storage.ConfigRecord{CompatibilityLevel: "FULL"})
	if err != nil {
		t.Fatalf("SetGlobalConfig failed: %v", err)
	}

	config, _ := store.GetGlobalConfig(ctx)
	if config.CompatibilityLevel != "FULL" {
		t.Errorf("Expected FULL, got %s", config.CompatibilityLevel)
	}

	// Set subject config
	err = store.SetConfig(ctx, "test-subject", &storage.ConfigRecord{CompatibilityLevel: "NONE"})
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	config, _ = store.GetConfig(ctx, "test-subject")
	if config.CompatibilityLevel != "NONE" {
		t.Errorf("Expected NONE, got %s", config.CompatibilityLevel)
	}
}

func TestStore_NotFound(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	_, err := store.GetSchemaByID(ctx, 999)
	if err != storage.ErrSchemaNotFound {
		t.Errorf("Expected ErrSchemaNotFound, got %v", err)
	}

	_, err = store.GetSchemaBySubjectVersion(ctx, "nonexistent", 1)
	if err != storage.ErrSubjectNotFound {
		t.Errorf("Expected ErrSubjectNotFound, got %v", err)
	}
}

func TestStore_ImportSchema(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Import a schema with a specific ID
	record := &storage.SchemaRecord{
		ID:          42,
		Subject:     "test-subject",
		Version:     1,
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "abc123",
	}

	err := store.ImportSchema(ctx, record)
	if err != nil {
		t.Fatalf("ImportSchema failed: %v", err)
	}

	// Verify the schema was imported with the correct ID
	got, err := store.GetSchemaByID(ctx, 42)
	if err != nil {
		t.Fatalf("GetSchemaByID failed: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("Expected ID 42, got %d", got.ID)
	}
	if got.Schema != record.Schema {
		t.Errorf("Schema mismatch")
	}

	// Verify by subject/version
	got, err = store.GetSchemaBySubjectVersion(ctx, "test-subject", 1)
	if err != nil {
		t.Fatalf("GetSchemaBySubjectVersion failed: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("Expected ID 42, got %d", got.ID)
	}
}

func TestStore_ImportSchema_IDConflict(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Import first schema
	record1 := &storage.SchemaRecord{
		ID:          42,
		Subject:     "subject-a",
		Version:     1,
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp1",
	}
	if err := store.ImportSchema(ctx, record1); err != nil {
		t.Fatalf("ImportSchema failed: %v", err)
	}

	// Try to import another schema with the same ID
	record2 := &storage.SchemaRecord{
		ID:          42,
		Subject:     "subject-b",
		Version:     1,
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp2",
	}
	err := store.ImportSchema(ctx, record2)
	if err != storage.ErrSchemaIDConflict {
		t.Errorf("Expected ErrSchemaIDConflict, got %v", err)
	}
}

func TestStore_ImportSchema_VersionConflict(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Import first schema
	record1 := &storage.SchemaRecord{
		ID:          42,
		Subject:     "test-subject",
		Version:     1,
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp1",
	}
	if err := store.ImportSchema(ctx, record1); err != nil {
		t.Fatalf("ImportSchema failed: %v", err)
	}

	// Try to import another schema with the same subject/version
	record2 := &storage.SchemaRecord{
		ID:          43,
		Subject:     "test-subject",
		Version:     1,
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp2",
	}
	err := store.ImportSchema(ctx, record2)
	if err != storage.ErrSchemaExists {
		t.Errorf("Expected ErrSchemaExists, got %v", err)
	}
}

func TestStore_SetNextID(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Set next ID to 100
	err := store.SetNextID(ctx, 100)
	if err != nil {
		t.Fatalf("SetNextID failed: %v", err)
	}

	// Next ID should be 100
	id, err := store.NextID(ctx)
	if err != nil {
		t.Fatalf("NextID failed: %v", err)
	}
	if id != 100 {
		t.Errorf("Expected NextID to return 100, got %d", id)
	}

	// Next call should return 101
	id, err = store.NextID(ctx)
	if err != nil {
		t.Fatalf("NextID failed: %v", err)
	}
	if id != 101 {
		t.Errorf("Expected NextID to return 101, got %d", id)
	}
}

func TestStore_ImportMultipleSchemas(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Import multiple schemas
	schemas := []struct {
		id      int64
		subject string
		version int
	}{
		{id: 10, subject: "subject-a", version: 1},
		{id: 20, subject: "subject-a", version: 2},
		{id: 30, subject: "subject-b", version: 1},
	}

	for _, s := range schemas {
		record := &storage.SchemaRecord{
			ID:          s.id,
			Subject:     s.subject,
			Version:     s.version,
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type": "string"}`,
			Fingerprint: string(rune(s.id)),
		}
		if err := store.ImportSchema(ctx, record); err != nil {
			t.Fatalf("ImportSchema failed for id=%d: %v", s.id, err)
		}
	}

	// Set next ID to be after the highest imported ID
	if err := store.SetNextID(ctx, 31); err != nil {
		t.Fatalf("SetNextID failed: %v", err)
	}

	// Create a new schema, should get ID 31
	newRecord := &storage.SchemaRecord{
		Subject:     "subject-c",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "new",
	}
	if err := store.CreateSchema(ctx, newRecord); err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}
	if newRecord.ID != 31 {
		t.Errorf("Expected new schema ID to be 31, got %d", newRecord.ID)
	}

	// Verify all subjects exist
	subjects, err := store.ListSubjects(ctx, false)
	if err != nil {
		t.Fatalf("ListSubjects failed: %v", err)
	}
	if len(subjects) != 3 {
		t.Errorf("Expected 3 subjects, got %d", len(subjects))
	}
}
