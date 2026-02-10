package conformance

import (
	"context"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// RunImportTests tests import operations and ID management.
func RunImportTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	t.Run("ImportSchema_WithSpecificID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{
			ID:          100,
			Subject:     "imported-subject",
			Version:     1,
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"string"}`,
			Fingerprint: "fp-import-1",
		}
		if err := store.ImportSchema(ctx, rec); err != nil {
			t.Fatalf("ImportSchema: %v", err)
		}

		got, err := store.GetSchemaByID(ctx, 100)
		if err != nil {
			t.Fatalf("GetSchemaByID: %v", err)
		}
		if got.ID != 100 {
			t.Errorf("expected ID 100, got %d", got.ID)
		}
		if got.Schema != `{"type":"string"}` {
			t.Errorf("schema mismatch")
		}
	})

	t.Run("ImportSchema_BySubjectVersion", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{
			ID:          200,
			Subject:     "imp-sv",
			Version:     3,
			SchemaType:  storage.SchemaTypeProtobuf,
			Schema:      `syntax = "proto3"; message M { string f = 1; }`,
			Fingerprint: "fp-import-sv",
		}
		store.ImportSchema(ctx, rec)

		got, err := store.GetSchemaBySubjectVersion(ctx, "imp-sv", 3)
		if err != nil {
			t.Fatalf("GetSchemaBySubjectVersion: %v", err)
		}
		if got.ID != 200 {
			t.Errorf("expected ID 200, got %d", got.ID)
		}
	})

	t.Run("ImportSchema_IDConflict", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{ID: 50, Subject: "s-a", Version: 1, SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-ic-1"}
		r2 := &storage.SchemaRecord{ID: 50, Subject: "s-b", Version: 1, SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-ic-2"}
		store.ImportSchema(ctx, r1)

		err := store.ImportSchema(ctx, r2)
		if err != storage.ErrSchemaIDConflict {
			t.Errorf("expected ErrSchemaIDConflict, got %v", err)
		}
	})

	t.Run("ImportSchema_VersionConflict", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{ID: 60, Subject: "s", Version: 1, SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-vc-1"}
		r2 := &storage.SchemaRecord{ID: 61, Subject: "s", Version: 1, SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-vc-2"}
		store.ImportSchema(ctx, r1)

		err := store.ImportSchema(ctx, r2)
		if err != storage.ErrSchemaExists {
			t.Errorf("expected ErrSchemaExists for version conflict, got %v", err)
		}
	})

	t.Run("SetNextID_AndNextID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		if err := store.SetNextID(ctx, 500); err != nil {
			t.Fatalf("SetNextID: %v", err)
		}

		id, err := store.NextID(ctx)
		if err != nil {
			t.Fatalf("NextID: %v", err)
		}
		if id != 500 {
			t.Errorf("expected 500, got %d", id)
		}

		id, err = store.NextID(ctx)
		if err != nil {
			t.Fatalf("NextID: %v", err)
		}
		if id != 501 {
			t.Errorf("expected 501, got %d", id)
		}
	})

	t.Run("ImportThenCreate_ContinuesFromCorrectID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Import schemas with specific IDs
		for _, id := range []int64{10, 20, 30} {
			rec := &storage.SchemaRecord{
				ID:          id,
				Subject:     "imp",
				Version:     int(id / 10),
				SchemaType:  storage.SchemaTypeAvro,
				Schema:      `{"type":"string"}`,
				Fingerprint: string(rune(id)),
			}
			if err := store.ImportSchema(ctx, rec); err != nil {
				t.Fatalf("ImportSchema(id=%d): %v", id, err)
			}
		}

		// Set next ID after the highest import
		if err := store.SetNextID(ctx, 31); err != nil {
			t.Fatalf("SetNextID: %v", err)
		}

		// Create a new schema via normal path
		newRec := &storage.SchemaRecord{
			Subject:     "new-subject",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"int"}`,
			Fingerprint: "fp-after-import",
		}
		if err := store.CreateSchema(ctx, newRec); err != nil {
			t.Fatalf("CreateSchema: %v", err)
		}
		if newRec.ID != 31 {
			t.Errorf("expected new schema ID 31, got %d", newRec.ID)
		}
	})

	t.Run("ImportMultiple_ListsCorrectly", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		entries := []struct {
			id      int64
			subject string
			version int
		}{
			{10, "sub-a", 1},
			{20, "sub-a", 2},
			{30, "sub-b", 1},
		}
		for _, e := range entries {
			rec := &storage.SchemaRecord{
				ID:          e.id,
				Subject:     e.subject,
				Version:     e.version,
				SchemaType:  storage.SchemaTypeAvro,
				Schema:      `{"type":"string"}`,
				Fingerprint: string(rune(e.id + 100)),
			}
			store.ImportSchema(ctx, rec)
		}

		subjects, err := store.ListSubjects(ctx, false)
		if err != nil {
			t.Fatalf("ListSubjects: %v", err)
		}
		if len(subjects) != 2 {
			t.Errorf("expected 2 subjects, got %d", len(subjects))
		}

		schemas, err := store.GetSchemasBySubject(ctx, "sub-a", false)
		if err != nil {
			t.Fatalf("GetSchemasBySubject: %v", err)
		}
		if len(schemas) != 2 {
			t.Errorf("expected 2 schemas for sub-a, got %d", len(schemas))
		}
	})

	t.Run("ImportSchema_WithReferences", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		base := &storage.SchemaRecord{
			ID:          1,
			Subject:     "base-type",
			Version:     1,
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"record","name":"Base","fields":[{"name":"id","type":"long"}]}`,
			Fingerprint: "fp-base-imp",
		}
		store.ImportSchema(ctx, base)

		child := &storage.SchemaRecord{
			ID:          2,
			Subject:     "child-type",
			Version:     1,
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"record","name":"Child","fields":[{"name":"base","type":"Base"}]}`,
			Fingerprint: "fp-child-imp",
			References:  []storage.Reference{{Name: "Base", Subject: "base-type", Version: 1}},
		}
		if err := store.ImportSchema(ctx, child); err != nil {
			t.Fatalf("ImportSchema(child): %v", err)
		}

		refs, err := store.GetReferencedBy(ctx, "base-type", 1)
		if err != nil {
			t.Fatalf("GetReferencedBy: %v", err)
		}
		if len(refs) != 1 {
			t.Errorf("expected 1 reference, got %d", len(refs))
		}
	})
}
