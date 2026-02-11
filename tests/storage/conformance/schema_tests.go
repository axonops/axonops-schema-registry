package conformance

import (
	"context"
	"fmt"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// RunSchemaTests tests all schema CRUD operations.
func RunSchemaTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	t.Run("CreateSchema_AssignsIDAndVersion", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{
			Subject:     "test-subject",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"string"}`,
			Fingerprint: "fp-create-1",
		}
		if err := store.CreateSchema(ctx, rec); err != nil {
			t.Fatalf("CreateSchema: %v", err)
		}
		if rec.ID == 0 {
			t.Error("expected non-zero ID")
		}
		if rec.Version != 1 {
			t.Errorf("expected version 1, got %d", rec.Version)
		}
	})

	t.Run("CreateSchema_SecondVersionIncrementsVersion", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-v1"}
		r2 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-v2"}
		if err := store.CreateSchema(ctx, r1); err != nil {
			t.Fatalf("CreateSchema v1: %v", err)
		}
		if err := store.CreateSchema(ctx, r2); err != nil {
			t.Fatalf("CreateSchema v2: %v", err)
		}
		if r2.Version != 2 {
			t.Errorf("expected version 2, got %d", r2.Version)
		}
	})

	t.Run("CreateSchema_GlobalDeduplication", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Same fingerprint within the same subject must return ErrSchemaExists
		r1 := &storage.SchemaRecord{Subject: "sub-a", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "shared-fp"}
		if err := store.CreateSchema(ctx, r1); err != nil {
			t.Fatalf("CreateSchema sub-a: %v", err)
		}
		dup := &storage.SchemaRecord{Subject: "sub-a", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "shared-fp"}
		if err := store.CreateSchema(ctx, dup); err != storage.ErrSchemaExists {
			t.Errorf("expected ErrSchemaExists for duplicate fingerprint in same subject, got %v", err)
		}

		// Same fingerprint across different subjects should succeed
		r2 := &storage.SchemaRecord{Subject: "sub-b", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "shared-fp"}
		if err := store.CreateSchema(ctx, r2); err != nil {
			t.Fatalf("CreateSchema sub-b: %v", err)
		}
		// Both schemas should be retrievable
		got1, err := store.GetSchemaBySubjectVersion(ctx, "sub-a", 1)
		if err != nil {
			t.Fatalf("GetSchemaBySubjectVersion sub-a: %v", err)
		}
		got2, err := store.GetSchemaBySubjectVersion(ctx, "sub-b", 1)
		if err != nil {
			t.Fatalf("GetSchemaBySubjectVersion sub-b: %v", err)
		}
		if got1.Schema != got2.Schema {
			t.Errorf("schemas should have same content")
		}
	})

	t.Run("GetSchemaByID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-get-id"}
		if err := store.CreateSchema(ctx, rec); err != nil {
			t.Fatalf("CreateSchema: %v", err)
		}
		got, err := store.GetSchemaByID(ctx, rec.ID)
		if err != nil {
			t.Fatalf("GetSchemaByID: %v", err)
		}
		if got.Schema != rec.Schema {
			t.Errorf("schema mismatch: %q vs %q", got.Schema, rec.Schema)
		}
	})

	t.Run("GetSchemaBySubjectVersion", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-sv"}
		if err := store.CreateSchema(ctx, rec); err != nil {
			t.Fatalf("CreateSchema: %v", err)
		}
		got, err := store.GetSchemaBySubjectVersion(ctx, "s", 1)
		if err != nil {
			t.Fatalf("GetSchemaBySubjectVersion: %v", err)
		}
		if got.ID != rec.ID {
			t.Errorf("ID mismatch: %d vs %d", got.ID, rec.ID)
		}
		if got.Subject != "s" {
			t.Errorf("subject mismatch: %q", got.Subject)
		}
		if got.Version != 1 {
			t.Errorf("version mismatch: %d", got.Version)
		}
	})

	t.Run("GetSchemaBySubjectVersion_Latest", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-lat-1"}
		r2 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-lat-2"}
		store.CreateSchema(ctx, r1)
		store.CreateSchema(ctx, r2)

		got, err := store.GetSchemaBySubjectVersion(ctx, "s", -1)
		if err != nil {
			t.Fatalf("GetSchemaBySubjectVersion(-1): %v", err)
		}
		if got.Version != 2 {
			t.Errorf("expected latest version 2, got %d", got.Version)
		}
	})

	t.Run("GetSchemasBySubject", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-gs-1"}
		r2 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-gs-2"}
		store.CreateSchema(ctx, r1)
		store.CreateSchema(ctx, r2)

		schemas, err := store.GetSchemasBySubject(ctx, "s", false)
		if err != nil {
			t.Fatalf("GetSchemasBySubject: %v", err)
		}
		if len(schemas) != 2 {
			t.Fatalf("expected 2 schemas, got %d", len(schemas))
		}
		// Should be sorted by version
		if schemas[0].Version > schemas[1].Version {
			t.Error("schemas not sorted by version")
		}
	})

	t.Run("GetSchemasBySubject_IncludeDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-del"}
		store.CreateSchema(ctx, rec)
		store.DeleteSchema(ctx, "s", 1, false) // soft delete

		// Without includeDeleted
		schemas, err := store.GetSchemasBySubject(ctx, "s", false)
		if err != nil {
			t.Fatalf("GetSchemasBySubject(false): %v", err)
		}
		if len(schemas) != 0 {
			t.Errorf("expected 0 non-deleted, got %d", len(schemas))
		}

		// With includeDeleted
		schemas, err = store.GetSchemasBySubject(ctx, "s", true)
		if err != nil {
			t.Fatalf("GetSchemasBySubject(true): %v", err)
		}
		if len(schemas) != 1 {
			t.Errorf("expected 1 with deleted, got %d", len(schemas))
		}
	})

	t.Run("GetSchemaByFingerprint", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-by-fp"}
		store.CreateSchema(ctx, rec)

		got, err := store.GetSchemaByFingerprint(ctx, "s", "fp-by-fp", false)
		if err != nil {
			t.Fatalf("GetSchemaByFingerprint: %v", err)
		}
		if got.ID != rec.ID {
			t.Errorf("ID mismatch: %d vs %d", got.ID, rec.ID)
		}
	})

	t.Run("GetSchemaByFingerprint_ExcludesDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-del-fp"}
		store.CreateSchema(ctx, rec)
		store.DeleteSchema(ctx, "s", 1, false)

		_, err := store.GetSchemaByFingerprint(ctx, "s", "fp-del-fp", false)
		if err != storage.ErrSchemaNotFound {
			t.Errorf("expected ErrSchemaNotFound, got %v", err)
		}

		got, err := store.GetSchemaByFingerprint(ctx, "s", "fp-del-fp", true)
		if err != nil {
			t.Fatalf("GetSchemaByFingerprint(includeDeleted): %v", err)
		}
		if got.ID != rec.ID {
			t.Errorf("ID mismatch")
		}
	})

	t.Run("GetSchemaByGlobalFingerprint", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-global"}
		store.CreateSchema(ctx, rec)

		got, err := store.GetSchemaByGlobalFingerprint(ctx, "fp-global")
		if err != nil {
			t.Fatalf("GetSchemaByGlobalFingerprint: %v", err)
		}
		if got.ID != rec.ID {
			t.Errorf("ID mismatch: %d vs %d", got.ID, rec.ID)
		}
	})

	t.Run("GetLatestSchema", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-lat-a"}
		r2 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-lat-b"}
		store.CreateSchema(ctx, r1)
		store.CreateSchema(ctx, r2)

		latest, err := store.GetLatestSchema(ctx, "s")
		if err != nil {
			t.Fatalf("GetLatestSchema: %v", err)
		}
		if latest.Version != 2 {
			t.Errorf("expected version 2, got %d", latest.Version)
		}
	})

	t.Run("GetLatestSchema_SkipsDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-lsd-1"}
		r2 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-lsd-2"}
		store.CreateSchema(ctx, r1)
		store.CreateSchema(ctx, r2)
		store.DeleteSchema(ctx, "s", 2, false) // soft-delete v2

		latest, err := store.GetLatestSchema(ctx, "s")
		if err != nil {
			t.Fatalf("GetLatestSchema: %v", err)
		}
		if latest.Version != 1 {
			t.Errorf("expected version 1 (v2 deleted), got %d", latest.Version)
		}
	})

	t.Run("DeleteSchema_Soft", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-ds"}
		store.CreateSchema(ctx, rec)

		if err := store.DeleteSchema(ctx, "s", 1, false); err != nil {
			t.Fatalf("DeleteSchema(soft): %v", err)
		}

		// Soft-deleted version should not be returned
		_, err := store.GetSchemaBySubjectVersion(ctx, "s", 1)
		if err != storage.ErrVersionNotFound {
			t.Errorf("expected ErrVersionNotFound, got %v", err)
		}

		// Schema content should still exist (other subjects might use it)
		got, err := store.GetSchemaByID(ctx, rec.ID)
		if err != nil {
			t.Fatalf("GetSchemaByID after soft delete: %v", err)
		}
		if got.Schema != rec.Schema {
			t.Error("schema content should still exist after soft delete")
		}
	})

	t.Run("DeleteSchema_Permanent", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-dp"}
		store.CreateSchema(ctx, rec)

		if err := store.DeleteSchema(ctx, "s", 1, true); err != nil {
			t.Fatalf("DeleteSchema(permanent): %v", err)
		}

		_, err := store.GetSchemaBySubjectVersion(ctx, "s", 1)
		if err == nil {
			t.Error("expected error after permanent delete")
		}
	})

	t.Run("GetSubjectsBySchemaID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "sub-a", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-subj-id"}
		store.CreateSchema(ctx, rec)

		subjects, err := store.GetSubjectsBySchemaID(ctx, rec.ID, false)
		if err != nil {
			t.Fatalf("GetSubjectsBySchemaID: %v", err)
		}
		if len(subjects) < 1 {
			t.Errorf("expected at least 1 subject, got %d", len(subjects))
		}
		found := false
		for _, s := range subjects {
			if s == "sub-a" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected subject 'sub-a' in results: %v", subjects)
		}
	})

	t.Run("GetSubjectsBySchemaID_ExcludesDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "sub-a", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-subj-del-a"}
		r2 := &storage.SchemaRecord{Subject: "sub-b", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-subj-del-b"}
		store.CreateSchema(ctx, r1)
		store.CreateSchema(ctx, r2)

		store.DeleteSchema(ctx, "sub-a", 1, false)

		// sub-a is deleted, so GetSubjectsBySchemaID for sub-a's schema should return empty or error
		subjects, err := store.GetSubjectsBySchemaID(ctx, r1.ID, false)
		if err != nil && err != storage.ErrSchemaNotFound {
			t.Fatalf("GetSubjectsBySchemaID: %v", err)
		}
		for _, s := range subjects {
			if s == "sub-a" {
				t.Errorf("deleted subject 'sub-a' should not appear in results")
			}
		}

		// sub-b should still be present
		subjects2, err := store.GetSubjectsBySchemaID(ctx, r2.ID, false)
		if err != nil {
			t.Fatalf("GetSubjectsBySchemaID for sub-b: %v", err)
		}
		if len(subjects2) != 1 {
			t.Errorf("expected 1 subject for sub-b, got %d", len(subjects2))
		}
	})

	t.Run("GetVersionsBySchemaID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "sub-a", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-ver-id"}
		store.CreateSchema(ctx, rec)

		versions, err := store.GetVersionsBySchemaID(ctx, rec.ID, false)
		if err != nil {
			t.Fatalf("GetVersionsBySchemaID: %v", err)
		}
		if len(versions) < 1 {
			t.Errorf("expected at least 1 subject-version, got %d", len(versions))
		}
	})

	t.Run("GetReferencedBy", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Create a base schema
		base := &storage.SchemaRecord{Subject: "base", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-base-ref"}
		store.CreateSchema(ctx, base)

		// Create a schema that references the base
		ref := &storage.SchemaRecord{
			Subject:     "child",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"record","name":"R","fields":[]}`,
			Fingerprint: "fp-child-ref",
			References:  []storage.Reference{{Name: "base", Subject: "base", Version: 1}},
		}
		store.CreateSchema(ctx, ref)

		refs, err := store.GetReferencedBy(ctx, "base", 1)
		if err != nil {
			t.Fatalf("GetReferencedBy: %v", err)
		}
		if len(refs) != 1 {
			t.Fatalf("expected 1 reference, got %d", len(refs))
		}
		if refs[0].Subject != "child" {
			t.Errorf("expected subject 'child', got %q", refs[0].Subject)
		}
	})

	t.Run("GetReferencedBy_Empty", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		base := &storage.SchemaRecord{Subject: "base", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-noref"}
		store.CreateSchema(ctx, base)

		refs, err := store.GetReferencedBy(ctx, "base", 1)
		if err != nil {
			t.Fatalf("GetReferencedBy: %v", err)
		}
		if len(refs) != 0 {
			t.Errorf("expected 0 references, got %d", len(refs))
		}
	})

	t.Run("ListSchemas_All", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "s1", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-ls-1"}
		r2 := &storage.SchemaRecord{Subject: "s2", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-ls-2"}
		store.CreateSchema(ctx, r1)
		store.CreateSchema(ctx, r2)

		schemas, err := store.ListSchemas(ctx, &storage.ListSchemasParams{})
		if err != nil {
			t.Fatalf("ListSchemas: %v", err)
		}
		if len(schemas) != 2 {
			t.Errorf("expected 2 schemas, got %d", len(schemas))
		}
	})

	t.Run("ListSchemas_SubjectPrefix", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "orders-value", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-pf-1"}
		r2 := &storage.SchemaRecord{Subject: "orders-key", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-pf-2"}
		r3 := &storage.SchemaRecord{Subject: "users-value", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"long"}`, Fingerprint: "fp-pf-3"}
		store.CreateSchema(ctx, r1)
		store.CreateSchema(ctx, r2)
		store.CreateSchema(ctx, r3)

		schemas, err := store.ListSchemas(ctx, &storage.ListSchemasParams{SubjectPrefix: "orders"})
		if err != nil {
			t.Fatalf("ListSchemas: %v", err)
		}
		if len(schemas) != 2 {
			t.Errorf("expected 2 schemas with prefix 'orders', got %d", len(schemas))
		}
	})

	t.Run("ListSchemas_OffsetLimit", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			rec := &storage.SchemaRecord{
				Subject:     "s",
				SchemaType:  storage.SchemaTypeAvro,
				Schema:      `{"type":"string"}`,
				Fingerprint: fmt.Sprintf("fp-ol-%d", i),
			}
			store.CreateSchema(ctx, rec)
		}

		schemas, err := store.ListSchemas(ctx, &storage.ListSchemasParams{Offset: 1, Limit: 2})
		if err != nil {
			t.Fatalf("ListSchemas: %v", err)
		}
		if len(schemas) != 2 {
			t.Errorf("expected 2 schemas with offset/limit, got %d", len(schemas))
		}
	})

	t.Run("ListSchemas_LatestOnly", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-lo-1"}
		r2 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-lo-2"}
		store.CreateSchema(ctx, r1)
		store.CreateSchema(ctx, r2)

		schemas, err := store.ListSchemas(ctx, &storage.ListSchemasParams{LatestOnly: true})
		if err != nil {
			t.Fatalf("ListSchemas: %v", err)
		}
		if len(schemas) != 1 {
			t.Errorf("expected 1 schema (latest only), got %d", len(schemas))
		}
		if schemas[0].Version != 2 {
			t.Errorf("expected latest version 2, got %d", schemas[0].Version)
		}
	})

	t.Run("ListSchemas_ExcludesDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-lsdel"}
		store.CreateSchema(ctx, rec)
		store.DeleteSchema(ctx, "s", 1, false)

		schemas, err := store.ListSchemas(ctx, &storage.ListSchemasParams{})
		if err != nil {
			t.Fatalf("ListSchemas: %v", err)
		}
		if len(schemas) != 0 {
			t.Errorf("expected 0 schemas (deleted excluded), got %d", len(schemas))
		}

		schemas, err = store.ListSchemas(ctx, &storage.ListSchemasParams{Deleted: true})
		if err != nil {
			t.Fatalf("ListSchemas(deleted): %v", err)
		}
		if len(schemas) != 1 {
			t.Errorf("expected 1 schema (deleted included), got %d", len(schemas))
		}
	})
}
