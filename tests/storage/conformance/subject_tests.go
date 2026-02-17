package conformance

import (
	"context"
	"fmt"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// RunSubjectTests tests all subject operations.
func RunSubjectTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	t.Run("ListSubjects_Empty", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		subjects, err := store.ListSubjects(ctx, false)
		if err != nil {
			t.Fatalf("ListSubjects: %v", err)
		}
		if len(subjects) != 0 {
			t.Errorf("expected 0 subjects, got %d", len(subjects))
		}
	})

	t.Run("ListSubjects_WithSchemas", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		for i, subj := range []string{"alpha", "beta", "gamma"} {
			rec := &storage.SchemaRecord{
				Subject:     subj,
				SchemaType:  storage.SchemaTypeAvro,
				Schema:      `{"type":"string"}`,
				Fingerprint: fmt.Sprintf("fp-ls-%d", i),
			}
			store.CreateSchema(ctx, rec)
		}

		subjects, err := store.ListSubjects(ctx, false)
		if err != nil {
			t.Fatalf("ListSubjects: %v", err)
		}
		if len(subjects) != 3 {
			t.Errorf("expected 3 subjects, got %d", len(subjects))
		}
		// Should be sorted
		if subjects[0] != "alpha" || subjects[1] != "beta" || subjects[2] != "gamma" {
			t.Errorf("subjects not sorted: %v", subjects)
		}
	})

	t.Run("ListSubjects_ExcludesAllDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-sub-del"}
		store.CreateSchema(ctx, rec)
		store.DeleteSubject(ctx, "s", false)

		subjects, err := store.ListSubjects(ctx, false)
		if err != nil {
			t.Fatalf("ListSubjects: %v", err)
		}
		if len(subjects) != 0 {
			t.Errorf("expected 0 subjects (all deleted), got %d", len(subjects))
		}
	})

	t.Run("ListSubjects_IncludeDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-sub-idel"}
		store.CreateSchema(ctx, rec)
		store.DeleteSubject(ctx, "s", false)

		subjects, err := store.ListSubjects(ctx, true)
		if err != nil {
			t.Fatalf("ListSubjects: %v", err)
		}
		if len(subjects) != 1 {
			t.Errorf("expected 1 subject (includeDeleted), got %d", len(subjects))
		}
	})

	t.Run("DeleteSubject_Soft", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-dss-1"}
		r2 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-dss-2"}
		store.CreateSchema(ctx, r1)
		store.CreateSchema(ctx, r2)

		versions, err := store.DeleteSubject(ctx, "s", false)
		if err != nil {
			t.Fatalf("DeleteSubject(soft): %v", err)
		}
		if len(versions) != 2 {
			t.Errorf("expected 2 deleted versions, got %d", len(versions))
		}

		// Subject should not be visible
		exists, _ := store.SubjectExists(ctx, "s")
		if exists {
			t.Error("subject should not exist after soft delete")
		}
	})

	t.Run("DeleteSubject_Permanent", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-dsp"}
		store.CreateSchema(ctx, rec)

		// Must soft-delete before permanent delete (two-step delete)
		_, err := store.DeleteSubject(ctx, "s", false)
		if err != nil {
			t.Fatalf("DeleteSubject(soft): %v", err)
		}
		versions, err := store.DeleteSubject(ctx, "s", true)
		if err != nil {
			t.Fatalf("DeleteSubject(permanent): %v", err)
		}
		if len(versions) != 1 {
			t.Errorf("expected 1 deleted version, got %d", len(versions))
		}

		// Subject should not appear even with includeDeleted
		subjects, _ := store.ListSubjects(ctx, true)
		if len(subjects) != 0 {
			t.Errorf("expected 0 subjects after permanent delete, got %d", len(subjects))
		}
	})

	t.Run("SubjectExists_True", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-se-t"}
		store.CreateSchema(ctx, rec)

		exists, err := store.SubjectExists(ctx, "s")
		if err != nil {
			t.Fatalf("SubjectExists: %v", err)
		}
		if !exists {
			t.Error("expected subject to exist")
		}
	})

	t.Run("SubjectExists_False_Nonexistent", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exists, err := store.SubjectExists(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("SubjectExists: %v", err)
		}
		if exists {
			t.Error("expected subject not to exist")
		}
	})

	t.Run("SubjectExists_False_AllDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-se-ad"}
		store.CreateSchema(ctx, rec)
		store.DeleteSubject(ctx, "s", false)

		exists, err := store.SubjectExists(ctx, "s")
		if err != nil {
			t.Fatalf("SubjectExists: %v", err)
		}
		if exists {
			t.Error("expected subject not to exist (all versions deleted)")
		}
	})
}
