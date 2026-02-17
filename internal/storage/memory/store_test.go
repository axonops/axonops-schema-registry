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

	// Get global config (seeded default is BACKWARD)
	config, err := store.GetGlobalConfig(ctx, ".")
	if err != nil {
		t.Fatalf("Expected seeded default, got error %v", err)
	}
	if config.CompatibilityLevel != "BACKWARD" {
		t.Errorf("Expected BACKWARD default, got %s", config.CompatibilityLevel)
	}

	// Set global config
	err = store.SetGlobalConfig(ctx, ".", &storage.ConfigRecord{CompatibilityLevel: "FULL"})
	if err != nil {
		t.Fatalf("SetGlobalConfig failed: %v", err)
	}

	config, _ = store.GetGlobalConfig(ctx, ".")
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

	// Verify ".ctxB" global config returns default BACKWARD (context doesn't exist yet)
	configB, err := store.GetGlobalConfig(ctx, ".ctxB")
	require.NoError(t, err)
	assert.Equal(t, "BACKWARD", configB.CompatibilityLevel)

	// Verify ".ctxB" subject config returns ErrNotFound
	_, err = store.GetConfig(ctx, ".ctxB", "test-subject")
	assert.ErrorIs(t, err, storage.ErrNotFound)

	// Verify default context "." still has original BACKWARD default
	configDefault, err := store.GetGlobalConfig(ctx, ".")
	require.NoError(t, err)
	assert.Equal(t, "BACKWARD", configDefault.CompatibilityLevel)
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

	// Verify ".ctxB" global mode returns default READWRITE (context doesn't exist yet)
	modeB, err := store.GetGlobalMode(ctx, ".ctxB")
	require.NoError(t, err)
	assert.Equal(t, "READWRITE", modeB.Mode)

	// Verify ".ctxB" subject mode returns ErrNotFound
	_, err = store.GetMode(ctx, ".ctxB", "test-subject")
	assert.ErrorIs(t, err, storage.ErrNotFound)

	// Verify default context "." still has original READWRITE
	modeDefault, err := store.GetGlobalMode(ctx, ".")
	require.NoError(t, err)
	assert.Equal(t, "READWRITE", modeDefault.Mode)
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
