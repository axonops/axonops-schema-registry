package memory

import (
	"context"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	err := store.CreateSchema(ctx, ".", record)
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
	got, err := store.GetSchemaByID(ctx, ".", record.ID)
	if err != nil {
		t.Fatalf("GetSchemaByID failed: %v", err)
	}
	if got.Schema != record.Schema {
		t.Errorf("Schema mismatch: got %s, want %s", got.Schema, record.Schema)
	}

	// Get by subject/version
	got, err = store.GetSchemaBySubjectVersion(ctx, ".", "test-subject", 1)
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

	err := store.CreateSchema(ctx, ".", record1)
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

	err = store.CreateSchema(ctx, ".", record2)
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
	if err := store.CreateSchema(ctx, ".", record1); err != nil {
		t.Fatalf("CreateSchema v1 failed: %v", err)
	}

	// Create version 2
	record2 := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp2",
	}
	if err := store.CreateSchema(ctx, ".", record2); err != nil {
		t.Fatalf("CreateSchema v2 failed: %v", err)
	}

	if record2.Version != 2 {
		t.Errorf("Expected version 2, got %d", record2.Version)
	}

	// Get latest
	latest, err := store.GetLatestSchema(ctx, ".", "test-subject")
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
		if err := store.CreateSchema(ctx, ".", record); err != nil {
			t.Fatalf("CreateSchema failed: %v", err)
		}
	}

	got, err := store.ListSubjects(ctx, ".", false)
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

	if err := store.CreateSchema(ctx, ".", record); err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}

	// Soft delete
	versions, err := store.DeleteSubject(ctx, ".", "test-subject", false)
	if err != nil {
		t.Fatalf("DeleteSubject failed: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("Expected 1 deleted version, got %d", len(versions))
	}

	// Subject should not appear in list
	subjects, _ := store.ListSubjects(ctx, ".", false)
	if len(subjects) != 0 {
		t.Errorf("Expected 0 subjects, got %d", len(subjects))
	}

	// But should appear with deleted=true
	subjects, _ = store.ListSubjects(ctx, ".", true)
	if len(subjects) != 1 {
		t.Errorf("Expected 1 subject with deleted=true, got %d", len(subjects))
	}
}

func TestStore_Config(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Get global config (no default seeded — returns ErrNotFound)
	_, err := store.GetGlobalConfig(ctx, ".")
	if err != storage.ErrNotFound {
		t.Fatalf("Expected ErrNotFound for unset global config, got %v", err)
	}

	// Set global config
	err = store.SetGlobalConfig(ctx, ".", &storage.ConfigRecord{CompatibilityLevel: "FULL"})
	if err != nil {
		t.Fatalf("SetGlobalConfig failed: %v", err)
	}

	config, _ := store.GetGlobalConfig(ctx, ".")
	if config.CompatibilityLevel != "FULL" {
		t.Errorf("Expected FULL, got %s", config.CompatibilityLevel)
	}

	// Set subject config
	err = store.SetConfig(ctx, ".", "test-subject", &storage.ConfigRecord{CompatibilityLevel: "NONE"})
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	config, _ = store.GetConfig(ctx, ".", "test-subject")
	if config.CompatibilityLevel != "NONE" {
		t.Errorf("Expected NONE, got %s", config.CompatibilityLevel)
	}
}

func TestStore_NotFound(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	_, err := store.GetSchemaByID(ctx, ".", 999)
	if err != storage.ErrSchemaNotFound {
		t.Errorf("Expected ErrSchemaNotFound, got %v", err)
	}

	_, err = store.GetSchemaBySubjectVersion(ctx, ".", "nonexistent", 1)
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

	err := store.ImportSchema(ctx, ".", record)
	if err != nil {
		t.Fatalf("ImportSchema failed: %v", err)
	}

	// Verify the schema was imported with the correct ID
	got, err := store.GetSchemaByID(ctx, ".", 42)
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
	got, err = store.GetSchemaBySubjectVersion(ctx, ".", "test-subject", 1)
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
	if err := store.ImportSchema(ctx, ".", record1); err != nil {
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
	err := store.ImportSchema(ctx, ".", record2)
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
	if err := store.ImportSchema(ctx, ".", record1); err != nil {
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
	err := store.ImportSchema(ctx, ".", record2)
	if err != storage.ErrSchemaExists {
		t.Errorf("Expected ErrSchemaExists, got %v", err)
	}
}

func TestStore_SetNextID(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Set next ID to 100
	err := store.SetNextID(ctx, ".", 100)
	if err != nil {
		t.Fatalf("SetNextID failed: %v", err)
	}

	// Next ID should be 100
	id, err := store.NextID(ctx, ".")
	if err != nil {
		t.Fatalf("NextID failed: %v", err)
	}
	if id != 100 {
		t.Errorf("Expected NextID to return 100, got %d", id)
	}

	// Next call should return 101
	id, err = store.NextID(ctx, ".")
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
		if err := store.ImportSchema(ctx, ".", record); err != nil {
			t.Fatalf("ImportSchema failed for id=%d: %v", s.id, err)
		}
	}

	// Set next ID to be after the highest imported ID
	if err := store.SetNextID(ctx, ".", 31); err != nil {
		t.Fatalf("SetNextID failed: %v", err)
	}

	// Create a new schema, should get ID 31
	newRecord := &storage.SchemaRecord{
		Subject:     "subject-c",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "new",
	}
	if err := store.CreateSchema(ctx, ".", newRecord); err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}
	if newRecord.ID != 31 {
		t.Errorf("Expected new schema ID to be 31, got %d", newRecord.ID)
	}

	// Verify all subjects exist
	subjects, err := store.ListSubjects(ctx, ".", false)
	if err != nil {
		t.Fatalf("ListSubjects failed: %v", err)
	}
	if len(subjects) != 3 {
		t.Errorf("Expected 3 subjects, got %d", len(subjects))
	}
}

func TestStore_ContextIsolation_Schemas(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register a schema in context ".ctxA"
	record := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-iso-schema-1",
	}
	err := store.CreateSchema(ctx, ".ctxA", record)
	require.NoError(t, err)
	require.NotZero(t, record.ID)

	// Verify the schema IS visible in ".ctxA"
	got, err := store.GetSchemaByID(ctx, ".ctxA", record.ID)
	require.NoError(t, err)
	assert.Equal(t, record.Schema, got.Schema)

	// Verify the schema is NOT visible in ".ctxB"
	_, err = store.GetSchemaByID(ctx, ".ctxB", record.ID)
	assert.ErrorIs(t, err, storage.ErrSchemaNotFound)

	// Also verify it's not visible in the default context "."
	_, err = store.GetSchemaByID(ctx, ".", record.ID)
	assert.ErrorIs(t, err, storage.ErrSchemaNotFound)
}

func TestStore_ContextIsolation_Subjects(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register schemas in ".ctxA" under subject "shared-name"
	recA := &storage.SchemaRecord{
		Subject:     "shared-name",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-iso-subj-a",
	}
	err := store.CreateSchema(ctx, ".ctxA", recA)
	require.NoError(t, err)

	// Register a schema in ".ctxA" under a different subject
	recA2 := &storage.SchemaRecord{
		Subject:     "only-in-a",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-iso-subj-a2",
	}
	err = store.CreateSchema(ctx, ".ctxA", recA2)
	require.NoError(t, err)

	// Register a schema in ".ctxB" under the same "shared-name" subject
	recB := &storage.SchemaRecord{
		Subject:     "shared-name",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "long"}`,
		Fingerprint: "fp-iso-subj-b",
	}
	err = store.CreateSchema(ctx, ".ctxB", recB)
	require.NoError(t, err)

	// ListSubjects in ".ctxA" should return both subjects
	subjectsA, err := store.ListSubjects(ctx, ".ctxA", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"only-in-a", "shared-name"}, subjectsA)

	// ListSubjects in ".ctxB" should return only "shared-name"
	subjectsB, err := store.ListSubjects(ctx, ".ctxB", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"shared-name"}, subjectsB)

	// ListSubjects in default context "." should return nothing (no schemas registered there)
	subjectsDefault, err := store.ListSubjects(ctx, ".", false)
	require.NoError(t, err)
	assert.Empty(t, subjectsDefault)
}

func TestStore_ContextIsolation_Config(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Set global config in ".ctxA"
	err := store.SetGlobalConfig(ctx, ".ctxA", &storage.ConfigRecord{CompatibilityLevel: "FULL"})
	require.NoError(t, err)

	// Set subject config in ".ctxA"
	err = store.SetConfig(ctx, ".ctxA", "test-subject", &storage.ConfigRecord{CompatibilityLevel: "NONE"})
	require.NoError(t, err)

	// Verify ".ctxA" global config is FULL
	configA, err := store.GetGlobalConfig(ctx, ".ctxA")
	require.NoError(t, err)
	assert.Equal(t, "FULL", configA.CompatibilityLevel)

	// Verify ".ctxA" subject config is NONE
	subjConfigA, err := store.GetConfig(ctx, ".ctxA", "test-subject")
	require.NoError(t, err)
	assert.Equal(t, "NONE", subjConfigA.CompatibilityLevel)

	// Verify ".ctxB" global config returns ErrNotFound (context doesn't exist yet)
	_, err = store.GetGlobalConfig(ctx, ".ctxB")
	assert.ErrorIs(t, err, storage.ErrNotFound)

	// Verify ".ctxB" subject config returns ErrNotFound
	_, err = store.GetConfig(ctx, ".ctxB", "test-subject")
	assert.ErrorIs(t, err, storage.ErrNotFound)

	// Verify default context "." has no explicit config set (ErrNotFound)
	_, err = store.GetGlobalConfig(ctx, ".")
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func TestStore_ContextIsolation_Mode(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Set global mode in ".ctxA" to IMPORT
	err := store.SetGlobalMode(ctx, ".ctxA", &storage.ModeRecord{Mode: "IMPORT"})
	require.NoError(t, err)

	// Set subject mode in ".ctxA"
	err = store.SetMode(ctx, ".ctxA", "test-subject", &storage.ModeRecord{Mode: "READONLY"})
	require.NoError(t, err)

	// Verify ".ctxA" global mode is IMPORT
	modeA, err := store.GetGlobalMode(ctx, ".ctxA")
	require.NoError(t, err)
	assert.Equal(t, "IMPORT", modeA.Mode)

	// Verify ".ctxA" subject mode is READONLY
	subjModeA, err := store.GetMode(ctx, ".ctxA", "test-subject")
	require.NoError(t, err)
	assert.Equal(t, "READONLY", subjModeA.Mode)

	// Verify ".ctxB" global mode returns ErrNotFound (context doesn't exist yet)
	_, err = store.GetGlobalMode(ctx, ".ctxB")
	assert.ErrorIs(t, err, storage.ErrNotFound)

	// Verify ".ctxB" subject mode returns ErrNotFound
	_, err = store.GetMode(ctx, ".ctxB", "test-subject")
	assert.ErrorIs(t, err, storage.ErrNotFound)

	// Verify default context "." has no explicit mode set (ErrNotFound)
	_, err = store.GetGlobalMode(ctx, ".")
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func TestStore_PerContextIDs(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register a schema in ".ctxA"
	recA := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-id-a",
	}
	err := store.CreateSchema(ctx, ".ctxA", recA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), recA.ID, "first schema in .ctxA should get ID 1")

	// Register a schema in ".ctxB"
	recB := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-id-b",
	}
	err = store.CreateSchema(ctx, ".ctxB", recB)
	require.NoError(t, err)
	assert.Equal(t, int64(1), recB.ID, "first schema in .ctxB should also get ID 1")

	// Register a second schema in ".ctxA"
	recA2 := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "long"}`,
		Fingerprint: "fp-id-a2",
	}
	err = store.CreateSchema(ctx, ".ctxA", recA2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), recA2.ID, "second schema in .ctxA should get ID 2")

	// ".ctxB" ID sequence should still be at 2 (next after 1)
	recB2 := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "float"}`,
		Fingerprint: "fp-id-b2",
	}
	err = store.CreateSchema(ctx, ".ctxB", recB2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), recB2.ID, "second schema in .ctxB should get ID 2")
}

func TestStore_PerContextFingerprints(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register identical schema content in ".ctxA"
	recA := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "shared-fingerprint",
	}
	err := store.CreateSchema(ctx, ".ctxA", recA)
	require.NoError(t, err)

	// Register identical schema content in ".ctxB"
	recB := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "shared-fingerprint",
	}
	err = store.CreateSchema(ctx, ".ctxB", recB)
	require.NoError(t, err)

	// Each context should have assigned its own ID independently
	assert.Equal(t, int64(1), recA.ID, ".ctxA should assign ID 1")
	assert.Equal(t, int64(1), recB.ID, ".ctxB should assign ID 1")

	// Verify each context can retrieve its own schema by ID
	gotA, err := store.GetSchemaByID(ctx, ".ctxA", recA.ID)
	require.NoError(t, err)
	assert.Equal(t, `{"type": "string"}`, gotA.Schema)

	gotB, err := store.GetSchemaByID(ctx, ".ctxB", recB.ID)
	require.NoError(t, err)
	assert.Equal(t, `{"type": "string"}`, gotB.Schema)

	// Fingerprint dedup is per-context: registering the same fingerprint
	// under a different subject in the same context should reuse the ID
	recA2 := &storage.SchemaRecord{
		Subject:     "other-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "shared-fingerprint",
	}
	err = store.CreateSchema(ctx, ".ctxA", recA2)
	require.NoError(t, err)
	assert.Equal(t, recA.ID, recA2.ID, "same fingerprint in same context should reuse ID")
}

func TestStore_ListContexts(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register schemas in multiple contexts to create them
	contexts := []string{".ctxC", ".ctxA", ".ctxB"}
	for i, regCtx := range contexts {
		rec := &storage.SchemaRecord{
			Subject:     "test-subject",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type": "string"}`,
			Fingerprint: string(rune('x' + i)),
		}
		err := store.CreateSchema(ctx, regCtx, rec)
		require.NoError(t, err)
	}

	// ListContexts should return all contexts sorted alphabetically
	got, err := store.ListContexts(ctx)
	require.NoError(t, err)

	// Should include the default context "." plus the three created contexts
	expected := []string{".", ".ctxA", ".ctxB", ".ctxC"}
	assert.Equal(t, expected, got)
}

func TestStore_DefaultContextAlwaysPresent(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// A fresh store should have the default context "."
	contexts, err := store.ListContexts(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"."}, contexts)
}

func TestStore_ContextIsolation_Delete(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register a schema in ".ctxA" under "shared-subject"
	recA := &storage.SchemaRecord{
		Subject:     "shared-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-del-a",
	}
	err := store.CreateSchema(ctx, ".ctxA", recA)
	require.NoError(t, err)

	// Register a schema in ".ctxB" under the same "shared-subject"
	recB := &storage.SchemaRecord{
		Subject:     "shared-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-del-b",
	}
	err = store.CreateSchema(ctx, ".ctxB", recB)
	require.NoError(t, err)

	// Soft-delete the subject in ".ctxA"
	deletedVersions, err := store.DeleteSubject(ctx, ".ctxA", "shared-subject", false)
	require.NoError(t, err)
	assert.Equal(t, []int{1}, deletedVersions)

	// Verify subject is gone from ".ctxA" (not listed without includeDeleted)
	subjectsA, err := store.ListSubjects(ctx, ".ctxA", false)
	require.NoError(t, err)
	assert.Empty(t, subjectsA)

	// Verify subject still exists in ".ctxB"
	subjectsB, err := store.ListSubjects(ctx, ".ctxB", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"shared-subject"}, subjectsB)

	// Verify the schema in ".ctxB" is still accessible
	gotB, err := store.GetSchemaBySubjectVersion(ctx, ".ctxB", "shared-subject", 1)
	require.NoError(t, err)
	assert.Equal(t, `{"type": "int"}`, gotB.Schema)

	// Verify the deleted subject in ".ctxA" still shows with includeDeleted=true
	subjectsADeleted, err := store.ListSubjects(ctx, ".ctxA", true)
	require.NoError(t, err)
	assert.Equal(t, []string{"shared-subject"}, subjectsADeleted)
}

// =============================================================================
// Context Isolation Tests for Methods at 0% Coverage
// =============================================================================

func TestStore_GetSchemasBySubject_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register two versions in ".ctx1"
	rec1a := &storage.SchemaRecord{
		Subject:     "shared-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-gsbs-1a",
	}
	err := store.CreateSchema(ctx, ".ctx1", rec1a)
	require.NoError(t, err)

	rec1b := &storage.SchemaRecord{
		Subject:     "shared-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-gsbs-1b",
	}
	err = store.CreateSchema(ctx, ".ctx1", rec1b)
	require.NoError(t, err)

	// Register one version in ".ctx2"
	rec2a := &storage.SchemaRecord{
		Subject:     "shared-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "long"}`,
		Fingerprint: "fp-gsbs-2a",
	}
	err = store.CreateSchema(ctx, ".ctx2", rec2a)
	require.NoError(t, err)

	// GetSchemasBySubject in ".ctx1" should return 2 schemas
	schemas1, err := store.GetSchemasBySubject(ctx, ".ctx1", "shared-subject", false)
	require.NoError(t, err)
	assert.Len(t, schemas1, 2)

	// GetSchemasBySubject in ".ctx2" should return 1 schema
	schemas2, err := store.GetSchemasBySubject(ctx, ".ctx2", "shared-subject", false)
	require.NoError(t, err)
	assert.Len(t, schemas2, 1)
	assert.Equal(t, `{"type": "long"}`, schemas2[0].Schema)

	// GetSchemasBySubject in an unused context should return ErrSubjectNotFound
	_, err = store.GetSchemasBySubject(ctx, ".ctx3", "shared-subject", false)
	assert.ErrorIs(t, err, storage.ErrSubjectNotFound)
}

func TestStore_DeleteSchema_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register schemas in two contexts under the same subject
	recA := &storage.SchemaRecord{
		Subject:     "shared-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-ds-a",
	}
	err := store.CreateSchema(ctx, ".ctx1", recA)
	require.NoError(t, err)

	recB := &storage.SchemaRecord{
		Subject:     "shared-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-ds-b",
	}
	err = store.CreateSchema(ctx, ".ctx2", recB)
	require.NoError(t, err)

	// Soft-delete version 1 in ".ctx1"
	err = store.DeleteSchema(ctx, ".ctx1", "shared-subject", 1, false)
	require.NoError(t, err)

	// Verify it is soft-deleted in ".ctx1" (not returned without includeDeleted)
	_, err = store.GetSchemasBySubject(ctx, ".ctx1", "shared-subject", false)
	assert.ErrorIs(t, err, storage.ErrSubjectNotFound)

	// Verify it still shows with includeDeleted=true
	schemasDeleted, err := store.GetSchemasBySubject(ctx, ".ctx1", "shared-subject", true)
	require.NoError(t, err)
	assert.Len(t, schemasDeleted, 1)
	assert.True(t, schemasDeleted[0].Deleted)

	// Verify ".ctx2" is completely unaffected
	schemas2, err := store.GetSchemasBySubject(ctx, ".ctx2", "shared-subject", false)
	require.NoError(t, err)
	assert.Len(t, schemas2, 1)
	assert.False(t, schemas2[0].Deleted)
	assert.Equal(t, `{"type": "int"}`, schemas2[0].Schema)
}

func TestStore_SubjectExists_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register a schema in ".ctx1"
	rec := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-se-1",
	}
	err := store.CreateSchema(ctx, ".ctx1", rec)
	require.NoError(t, err)

	// SubjectExists should return true for ".ctx1"
	exists, err := store.SubjectExists(ctx, ".ctx1", "test-subject")
	require.NoError(t, err)
	assert.True(t, exists)

	// SubjectExists should return false for ".ctx2" (never registered there)
	exists, err = store.SubjectExists(ctx, ".ctx2", "test-subject")
	require.NoError(t, err)
	assert.False(t, exists)

	// SubjectExists should return false for default context "."
	exists, err = store.SubjectExists(ctx, ".", "test-subject")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestStore_DeleteConfig_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Set subject config in ".ctx1"
	err := store.SetConfig(ctx, ".ctx1", "test-subject", &storage.ConfigRecord{CompatibilityLevel: "FULL"})
	require.NoError(t, err)

	// Set same subject config in ".ctx2"
	err = store.SetConfig(ctx, ".ctx2", "test-subject", &storage.ConfigRecord{CompatibilityLevel: "NONE"})
	require.NoError(t, err)

	// Delete config in ".ctx1"
	err = store.DeleteConfig(ctx, ".ctx1", "test-subject")
	require.NoError(t, err)

	// Verify config is gone in ".ctx1"
	_, err = store.GetConfig(ctx, ".ctx1", "test-subject")
	assert.ErrorIs(t, err, storage.ErrNotFound)

	// Verify config still exists in ".ctx2"
	config2, err := store.GetConfig(ctx, ".ctx2", "test-subject")
	require.NoError(t, err)
	assert.Equal(t, "NONE", config2.CompatibilityLevel)
}

func TestStore_DeleteMode_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Set subject mode in ".ctx1"
	err := store.SetMode(ctx, ".ctx1", "test-subject", &storage.ModeRecord{Mode: "READONLY"})
	require.NoError(t, err)

	// Set same subject mode in ".ctx2"
	err = store.SetMode(ctx, ".ctx2", "test-subject", &storage.ModeRecord{Mode: "IMPORT"})
	require.NoError(t, err)

	// Delete mode in ".ctx1"
	err = store.DeleteMode(ctx, ".ctx1", "test-subject")
	require.NoError(t, err)

	// Verify mode is gone in ".ctx1"
	_, err = store.GetMode(ctx, ".ctx1", "test-subject")
	assert.ErrorIs(t, err, storage.ErrNotFound)

	// Verify mode still exists in ".ctx2"
	mode2, err := store.GetMode(ctx, ".ctx2", "test-subject")
	require.NoError(t, err)
	assert.Equal(t, "IMPORT", mode2.Mode)
}

func TestStore_GetSchemaByFingerprint_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register the same schema (same fingerprint) in two contexts under the same subject
	recA := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "shared-fp",
	}
	err := store.CreateSchema(ctx, ".ctx1", recA)
	require.NoError(t, err)

	recB := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "shared-fp",
	}
	err = store.CreateSchema(ctx, ".ctx2", recB)
	require.NoError(t, err)

	// GetSchemaByFingerprint in ".ctx1" should return ".ctx1"'s schema
	gotA, err := store.GetSchemaByFingerprint(ctx, ".ctx1", "test-subject", "shared-fp", false)
	require.NoError(t, err)
	assert.Equal(t, recA.ID, gotA.ID)

	// GetSchemaByFingerprint in ".ctx2" should return ".ctx2"'s schema
	gotB, err := store.GetSchemaByFingerprint(ctx, ".ctx2", "test-subject", "shared-fp", false)
	require.NoError(t, err)
	assert.Equal(t, recB.ID, gotB.ID)

	// GetSchemaByFingerprint in an unused context should fail
	_, err = store.GetSchemaByFingerprint(ctx, ".ctx3", "test-subject", "shared-fp", false)
	assert.ErrorIs(t, err, storage.ErrSubjectNotFound)
}

func TestStore_GetSchemaByGlobalFingerprint_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register schemas with same fingerprint in two contexts
	recA := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "global-fp",
	}
	err := store.CreateSchema(ctx, ".ctx1", recA)
	require.NoError(t, err)

	recB := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "global-fp",
	}
	err = store.CreateSchema(ctx, ".ctx2", recB)
	require.NoError(t, err)

	// GetSchemaByGlobalFingerprint in ".ctx1"
	gotA, err := store.GetSchemaByGlobalFingerprint(ctx, ".ctx1", "global-fp")
	require.NoError(t, err)
	assert.Equal(t, recA.ID, gotA.ID)

	// GetSchemaByGlobalFingerprint in ".ctx2"
	gotB, err := store.GetSchemaByGlobalFingerprint(ctx, ".ctx2", "global-fp")
	require.NoError(t, err)
	assert.Equal(t, recB.ID, gotB.ID)

	// GetSchemaByGlobalFingerprint in an unused context should fail
	_, err = store.GetSchemaByGlobalFingerprint(ctx, ".ctx3", "global-fp")
	assert.ErrorIs(t, err, storage.ErrSchemaNotFound)
}

func TestStore_GetReferencedBy_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// In ".ctx1": register a base schema and a referencing schema
	base := &storage.SchemaRecord{
		Subject:     "base-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-ref-base",
	}
	err := store.CreateSchema(ctx, ".ctx1", base)
	require.NoError(t, err)

	referencing := &storage.SchemaRecord{
		Subject:     "ref-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-ref-referencing",
		References: []storage.Reference{
			{Name: "base", Subject: "base-subject", Version: 1},
		},
	}
	err = store.CreateSchema(ctx, ".ctx1", referencing)
	require.NoError(t, err)

	// GetReferencedBy in ".ctx1" should find the referencing schema
	refs1, err := store.GetReferencedBy(ctx, ".ctx1", "base-subject", 1)
	require.NoError(t, err)
	assert.Len(t, refs1, 1)
	assert.Equal(t, "ref-subject", refs1[0].Subject)

	// GetReferencedBy in ".ctx2" should find nothing (no schemas there)
	refs2, err := store.GetReferencedBy(ctx, ".ctx2", "base-subject", 1)
	require.NoError(t, err)
	assert.Empty(t, refs2)
}

func TestStore_GetSubjectsBySchemaID_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register schemas in two contexts — both will get ID 1 in their context
	recA := &storage.SchemaRecord{
		Subject:     "subject-in-ctx1",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-sbsid-a",
	}
	err := store.CreateSchema(ctx, ".ctx1", recA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), recA.ID)

	recB := &storage.SchemaRecord{
		Subject:     "subject-in-ctx2",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-sbsid-b",
	}
	err = store.CreateSchema(ctx, ".ctx2", recB)
	require.NoError(t, err)
	assert.Equal(t, int64(1), recB.ID)

	// GetSubjectsBySchemaID(1) in ".ctx1" should return "subject-in-ctx1"
	subjects1, err := store.GetSubjectsBySchemaID(ctx, ".ctx1", 1, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"subject-in-ctx1"}, subjects1)

	// GetSubjectsBySchemaID(1) in ".ctx2" should return "subject-in-ctx2"
	subjects2, err := store.GetSubjectsBySchemaID(ctx, ".ctx2", 1, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"subject-in-ctx2"}, subjects2)

	// GetSubjectsBySchemaID(1) in ".ctx3" should return ErrSchemaNotFound
	_, err = store.GetSubjectsBySchemaID(ctx, ".ctx3", 1, false)
	assert.ErrorIs(t, err, storage.ErrSchemaNotFound)
}

func TestStore_GetVersionsBySchemaID_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register schemas in two contexts
	recA := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-vbsid-a",
	}
	err := store.CreateSchema(ctx, ".ctx1", recA)
	require.NoError(t, err)

	recB := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-vbsid-b",
	}
	err = store.CreateSchema(ctx, ".ctx2", recB)
	require.NoError(t, err)

	// GetVersionsBySchemaID in ".ctx1" for ID 1 should return version info from ".ctx1"
	versions1, err := store.GetVersionsBySchemaID(ctx, ".ctx1", 1, false)
	require.NoError(t, err)
	assert.Len(t, versions1, 1)
	assert.Equal(t, "test-subject", versions1[0].Subject)
	assert.Equal(t, 1, versions1[0].Version)

	// GetVersionsBySchemaID in ".ctx2" for ID 1 should return version info from ".ctx2"
	versions2, err := store.GetVersionsBySchemaID(ctx, ".ctx2", 1, false)
	require.NoError(t, err)
	assert.Len(t, versions2, 1)
	assert.Equal(t, "test-subject", versions2[0].Subject)
	assert.Equal(t, 1, versions2[0].Version)

	// GetVersionsBySchemaID in ".ctx3" should return ErrSchemaNotFound
	_, err = store.GetVersionsBySchemaID(ctx, ".ctx3", 1, false)
	assert.ErrorIs(t, err, storage.ErrSchemaNotFound)
}

func TestStore_ListSchemas_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register two schemas in ".ctx1"
	rec1a := &storage.SchemaRecord{
		Subject:     "subject-a",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "string"}`,
		Fingerprint: "fp-ls-1a",
	}
	err := store.CreateSchema(ctx, ".ctx1", rec1a)
	require.NoError(t, err)

	rec1b := &storage.SchemaRecord{
		Subject:     "subject-b",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-ls-1b",
	}
	err = store.CreateSchema(ctx, ".ctx1", rec1b)
	require.NoError(t, err)

	// Register one schema in ".ctx2"
	rec2a := &storage.SchemaRecord{
		Subject:     "subject-c",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "long"}`,
		Fingerprint: "fp-ls-2a",
	}
	err = store.CreateSchema(ctx, ".ctx2", rec2a)
	require.NoError(t, err)

	// ListSchemas in ".ctx1" should return 2 schemas
	schemas1, err := store.ListSchemas(ctx, ".ctx1", &storage.ListSchemasParams{})
	require.NoError(t, err)
	assert.Len(t, schemas1, 2)

	// ListSchemas in ".ctx2" should return 1 schema
	schemas2, err := store.ListSchemas(ctx, ".ctx2", &storage.ListSchemasParams{})
	require.NoError(t, err)
	assert.Len(t, schemas2, 1)
	assert.Equal(t, "subject-c", schemas2[0].Subject)

	// ListSchemas in ".ctx3" (unused) should return empty
	schemas3, err := store.ListSchemas(ctx, ".ctx3", &storage.ListSchemasParams{})
	require.NoError(t, err)
	assert.Empty(t, schemas3)
}

func TestStore_DeleteGlobalConfig_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Set global config in ".ctx1" to FULL
	err := store.SetGlobalConfig(ctx, ".ctx1", &storage.ConfigRecord{CompatibilityLevel: "FULL"})
	require.NoError(t, err)

	// Set global config in ".ctx2" to NONE
	err = store.SetGlobalConfig(ctx, ".ctx2", &storage.ConfigRecord{CompatibilityLevel: "NONE"})
	require.NoError(t, err)

	// Delete global config in ".ctx1" (removes it entirely)
	err = store.DeleteGlobalConfig(ctx, ".ctx1")
	require.NoError(t, err)

	// Verify ".ctx1" global config returns ErrNotFound after deletion
	_, err = store.GetGlobalConfig(ctx, ".ctx1")
	assert.ErrorIs(t, err, storage.ErrNotFound)

	// Verify ".ctx2" global config is still NONE (unaffected)
	config2, err := store.GetGlobalConfig(ctx, ".ctx2")
	require.NoError(t, err)
	assert.Equal(t, "NONE", config2.CompatibilityLevel)
}

func TestStore_GetMaxSchemaID_ContextIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register 3 schemas in ".ctx1"
	for i := 0; i < 3; i++ {
		rec := &storage.SchemaRecord{
			Subject:     "test-subject",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type": "string"}`,
			Fingerprint: string(rune('a' + i)),
		}
		err := store.CreateSchema(ctx, ".ctx1", rec)
		require.NoError(t, err)
	}

	// Register 1 schema in ".ctx2"
	rec := &storage.SchemaRecord{
		Subject:     "test-subject",
		SchemaType:  storage.SchemaTypeAvro,
		Schema:      `{"type": "int"}`,
		Fingerprint: "fp-max-2",
	}
	err := store.CreateSchema(ctx, ".ctx2", rec)
	require.NoError(t, err)

	// GetMaxSchemaID in ".ctx1" should return 3
	maxID1, err := store.GetMaxSchemaID(ctx, ".ctx1")
	require.NoError(t, err)
	assert.Equal(t, int64(3), maxID1)

	// GetMaxSchemaID in ".ctx2" should return 1
	maxID2, err := store.GetMaxSchemaID(ctx, ".ctx2")
	require.NoError(t, err)
	assert.Equal(t, int64(1), maxID2)

	// GetMaxSchemaID in ".ctx3" (unused) should return 0
	maxID3, err := store.GetMaxSchemaID(ctx, ".ctx3")
	require.NoError(t, err)
	assert.Equal(t, int64(0), maxID3)
}
