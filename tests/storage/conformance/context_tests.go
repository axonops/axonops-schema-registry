package conformance

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// RunContextTests tests context (namespace) isolation across all storage operations.
func RunContextTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	// --- Schema isolation between contexts ---

	t.Run("SchemaIsolation_IndependentIDsAndVersions", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Create a schema in the default context "."
		recDefault := &storage.SchemaRecord{
			Subject:     "my-subject",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"string"}`,
			Fingerprint: "fp-ctx-default-1",
		}
		if err := store.CreateSchema(ctx, ".", recDefault); err != nil {
			t.Fatalf("CreateSchema in default ctx: %v", err)
		}

		// Create a schema in context ".ctx-a"
		recA := &storage.SchemaRecord{
			Subject:     "my-subject",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"string"}`,
			Fingerprint: "fp-ctx-a-1",
		}
		if err := store.CreateSchema(ctx, ".ctx-a", recA); err != nil {
			t.Fatalf("CreateSchema in .ctx-a: %v", err)
		}

		// Both should have version 1 (independent version counters)
		if recDefault.Version != 1 {
			t.Errorf("default ctx: expected version 1, got %d", recDefault.Version)
		}
		if recA.Version != 1 {
			t.Errorf(".ctx-a: expected version 1, got %d", recA.Version)
		}

		// IDs should be independently assigned (both should be non-zero)
		if recDefault.ID == 0 {
			t.Error("default ctx: expected non-zero ID")
		}
		if recA.ID == 0 {
			t.Error(".ctx-a: expected non-zero ID")
		}
	})

	t.Run("SchemaIsolation_GetSchemaByID_CrossContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Create multiple schemas in ".ctx-a" to advance its ID counter
		recA1 := &storage.SchemaRecord{
			Subject:     "subj-a-1",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"string"}`,
			Fingerprint: "fp-iso-a-1",
		}
		if err := store.CreateSchema(ctx, ".ctx-a", recA1); err != nil {
			t.Fatalf("CreateSchema 1 in .ctx-a: %v", err)
		}
		recA2 := &storage.SchemaRecord{
			Subject:     "subj-a-2",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"long"}`,
			Fingerprint: "fp-iso-a-2",
		}
		if err := store.CreateSchema(ctx, ".ctx-a", recA2); err != nil {
			t.Fatalf("CreateSchema 2 in .ctx-a: %v", err)
		}
		recA3 := &storage.SchemaRecord{
			Subject:     "subj-a-3",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"float"}`,
			Fingerprint: "fp-iso-a-3",
		}
		if err := store.CreateSchema(ctx, ".ctx-a", recA3); err != nil {
			t.Fatalf("CreateSchema 3 in .ctx-a: %v", err)
		}

		// Create only one schema in ".ctx-b"
		recB := &storage.SchemaRecord{
			Subject:     "subj-b",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"int"}`,
			Fingerprint: "fp-iso-b",
		}
		if err := store.CreateSchema(ctx, ".ctx-b", recB); err != nil {
			t.Fatalf("CreateSchema in .ctx-b: %v", err)
		}

		// GetSchemaByID in .ctx-a should return schema 3 from .ctx-a
		gotA, err := store.GetSchemaByID(ctx, ".ctx-a", recA3.ID)
		if err != nil {
			t.Fatalf("GetSchemaByID(%d) in .ctx-a: %v", recA3.ID, err)
		}
		if gotA.Schema != `{"type":"float"}` {
			t.Errorf(".ctx-a: expected float schema, got %q", gotA.Schema)
		}

		// GetSchemaByID in .ctx-b for .ctx-a's highest ID should fail
		// (since .ctx-b only has 1 schema, ID 3 from .ctx-a should not exist there)
		_, err = store.GetSchemaByID(ctx, ".ctx-b", recA3.ID)
		if !errors.Is(err, storage.ErrSchemaNotFound) {
			t.Errorf("GetSchemaByID(%d) in .ctx-b for .ctx-a's ID: expected ErrSchemaNotFound, got %v", recA3.ID, err)
		}

		// GetSchemaByID in .ctx-b for its own ID should succeed
		gotB, err := store.GetSchemaByID(ctx, ".ctx-b", recB.ID)
		if err != nil {
			t.Fatalf("GetSchemaByID(%d) in .ctx-b: %v", recB.ID, err)
		}
		if gotB.Schema != `{"type":"int"}` {
			t.Errorf(".ctx-b: expected int schema, got %q", gotB.Schema)
		}
	})

	t.Run("SchemaIsolation_ListSubjects_ScopedToContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Create schemas in different contexts
		for i, subj := range []string{"alpha", "beta"} {
			rec := &storage.SchemaRecord{
				Subject:     subj,
				SchemaType:  storage.SchemaTypeAvro,
				Schema:      `{"type":"string"}`,
				Fingerprint: fmt.Sprintf("fp-ls-ctx-a-%d", i),
			}
			if err := store.CreateSchema(ctx, ".ctx-a", rec); err != nil {
				t.Fatalf("CreateSchema %s in .ctx-a: %v", subj, err)
			}
		}
		recB := &storage.SchemaRecord{
			Subject:     "gamma",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"int"}`,
			Fingerprint: "fp-ls-ctx-b-0",
		}
		if err := store.CreateSchema(ctx, ".ctx-b", recB); err != nil {
			t.Fatalf("CreateSchema gamma in .ctx-b: %v", err)
		}

		// ListSubjects in .ctx-a should return alpha and beta
		subjectsA, err := store.ListSubjects(ctx, ".ctx-a", false)
		if err != nil {
			t.Fatalf("ListSubjects .ctx-a: %v", err)
		}
		if len(subjectsA) != 2 {
			t.Errorf(".ctx-a: expected 2 subjects, got %d: %v", len(subjectsA), subjectsA)
		}

		// ListSubjects in .ctx-b should return only gamma
		subjectsB, err := store.ListSubjects(ctx, ".ctx-b", false)
		if err != nil {
			t.Fatalf("ListSubjects .ctx-b: %v", err)
		}
		if len(subjectsB) != 1 {
			t.Errorf(".ctx-b: expected 1 subject, got %d: %v", len(subjectsB), subjectsB)
		}
		if len(subjectsB) > 0 && subjectsB[0] != "gamma" {
			t.Errorf(".ctx-b: expected subject 'gamma', got %q", subjectsB[0])
		}

		// ListSubjects in an unused context should return empty
		subjectsEmpty, err := store.ListSubjects(ctx, ".unused", false)
		if err != nil {
			t.Fatalf("ListSubjects .unused: %v", err)
		}
		if len(subjectsEmpty) != 0 {
			t.Errorf(".unused: expected 0 subjects, got %d", len(subjectsEmpty))
		}
	})

	// --- Config isolation between contexts ---

	t.Run("ConfigIsolation_GlobalConfig", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Set global config in .ctx-a to FULL
		if err := store.SetGlobalConfig(ctx, ".ctx-a", &storage.ConfigRecord{CompatibilityLevel: "FULL"}); err != nil {
			t.Fatalf("SetGlobalConfig .ctx-a: %v", err)
		}

		// Set global config in .ctx-b to NONE
		if err := store.SetGlobalConfig(ctx, ".ctx-b", &storage.ConfigRecord{CompatibilityLevel: "NONE"}); err != nil {
			t.Fatalf("SetGlobalConfig .ctx-b: %v", err)
		}

		// Verify each context returns its own config
		configA, err := store.GetGlobalConfig(ctx, ".ctx-a")
		if err != nil {
			t.Fatalf("GetGlobalConfig .ctx-a: %v", err)
		}
		if configA.CompatibilityLevel != "FULL" {
			t.Errorf(".ctx-a: expected FULL, got %q", configA.CompatibilityLevel)
		}

		configB, err := store.GetGlobalConfig(ctx, ".ctx-b")
		if err != nil {
			t.Fatalf("GetGlobalConfig .ctx-b: %v", err)
		}
		if configB.CompatibilityLevel != "NONE" {
			t.Errorf(".ctx-b: expected NONE, got %q", configB.CompatibilityLevel)
		}
	})

	t.Run("ConfigIsolation_PerSubjectConfig", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Set per-subject config in .ctx-a
		if err := store.SetConfig(ctx, ".ctx-a", "my-subject", &storage.ConfigRecord{CompatibilityLevel: "FORWARD"}); err != nil {
			t.Fatalf("SetConfig .ctx-a: %v", err)
		}

		// Verify config exists in .ctx-a
		configA, err := store.GetConfig(ctx, ".ctx-a", "my-subject")
		if err != nil {
			t.Fatalf("GetConfig .ctx-a: %v", err)
		}
		if configA.CompatibilityLevel != "FORWARD" {
			t.Errorf(".ctx-a: expected FORWARD, got %q", configA.CompatibilityLevel)
		}

		// Verify config is absent in .ctx-b
		_, err = store.GetConfig(ctx, ".ctx-b", "my-subject")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("GetConfig .ctx-b: expected ErrNotFound, got %v", err)
		}
	})

	// --- Mode isolation between contexts ---

	t.Run("ModeIsolation_GlobalMode", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Set global mode in .ctx-a to READWRITE
		if err := store.SetGlobalMode(ctx, ".ctx-a", &storage.ModeRecord{Mode: "READWRITE"}); err != nil {
			t.Fatalf("SetGlobalMode .ctx-a: %v", err)
		}

		// Set global mode in .ctx-b to READONLY
		if err := store.SetGlobalMode(ctx, ".ctx-b", &storage.ModeRecord{Mode: "READONLY"}); err != nil {
			t.Fatalf("SetGlobalMode .ctx-b: %v", err)
		}

		// Verify each context returns its own mode
		modeA, err := store.GetGlobalMode(ctx, ".ctx-a")
		if err != nil {
			t.Fatalf("GetGlobalMode .ctx-a: %v", err)
		}
		if modeA.Mode != "READWRITE" {
			t.Errorf(".ctx-a: expected READWRITE, got %q", modeA.Mode)
		}

		modeB, err := store.GetGlobalMode(ctx, ".ctx-b")
		if err != nil {
			t.Fatalf("GetGlobalMode .ctx-b: %v", err)
		}
		if modeB.Mode != "READONLY" {
			t.Errorf(".ctx-b: expected READONLY, got %q", modeB.Mode)
		}
	})

	t.Run("ModeIsolation_DeleteDoesNotAffectOtherContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Set global mode in both contexts
		if err := store.SetGlobalMode(ctx, ".ctx-a", &storage.ModeRecord{Mode: "READWRITE"}); err != nil {
			t.Fatalf("SetGlobalMode .ctx-a: %v", err)
		}
		if err := store.SetGlobalMode(ctx, ".ctx-b", &storage.ModeRecord{Mode: "READONLY"}); err != nil {
			t.Fatalf("SetGlobalMode .ctx-b: %v", err)
		}

		// Delete global mode in .ctx-a
		if err := store.DeleteGlobalMode(ctx, ".ctx-a"); err != nil {
			t.Fatalf("DeleteGlobalMode .ctx-a: %v", err)
		}

		// Verify .ctx-a no longer has a mode
		_, err := store.GetGlobalMode(ctx, ".ctx-a")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("GetGlobalMode .ctx-a after delete: expected ErrNotFound, got %v", err)
		}

		// Verify .ctx-b is unaffected
		modeB, err := store.GetGlobalMode(ctx, ".ctx-b")
		if err != nil {
			t.Fatalf("GetGlobalMode .ctx-b: %v", err)
		}
		if modeB.Mode != "READONLY" {
			t.Errorf(".ctx-b: expected READONLY, got %q", modeB.Mode)
		}
	})

	// --- ListContexts ---

	t.Run("ListContexts_FreshStore_ContainsDefaultContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		contexts, err := store.ListContexts(ctx)
		if err != nil {
			t.Fatalf("ListContexts: %v", err)
		}
		// A fresh store may contain the default context "." or be empty.
		// Both behaviors are acceptable depending on the backend.
		for _, c := range contexts {
			if c != "." {
				t.Errorf("fresh store should only contain default context '.', got %q", c)
			}
		}
	})

	t.Run("ListContexts_ReturnsAllContextsWithData", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Create schemas in multiple contexts (including default ".")
		for i, rctx := range []string{".", ".prod", ".staging"} {
			rec := &storage.SchemaRecord{
				Subject:     "subj",
				SchemaType:  storage.SchemaTypeAvro,
				Schema:      `{"type":"string"}`,
				Fingerprint: fmt.Sprintf("fp-lc-%d", i),
			}
			if err := store.CreateSchema(ctx, rctx, rec); err != nil {
				t.Fatalf("CreateSchema in %s: %v", rctx, err)
			}
		}

		contexts, err := store.ListContexts(ctx)
		if err != nil {
			t.Fatalf("ListContexts: %v", err)
		}
		if len(contexts) < 3 {
			t.Fatalf("expected at least 3 contexts, got %d: %v", len(contexts), contexts)
		}

		// Verify the list is sorted
		if !sort.StringsAreSorted(contexts) {
			t.Errorf("contexts not sorted: %v", contexts)
		}

		// Verify all expected contexts are present
		expected := map[string]bool{".": false, ".prod": false, ".staging": false}
		for _, c := range contexts {
			if _, ok := expected[c]; ok {
				expected[c] = true
			}
		}
		for name, found := range expected {
			if !found {
				t.Errorf("expected context %q not found in %v", name, contexts)
			}
		}
	})

	// --- GetMaxSchemaID per context ---

	t.Run("GetMaxSchemaID_FreshStore", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		maxID, err := store.GetMaxSchemaID(ctx, ".ctx-a")
		if err != nil {
			t.Fatalf("GetMaxSchemaID: %v", err)
		}
		if maxID != 0 {
			t.Errorf("expected 0 for fresh context, got %d", maxID)
		}
	})

	t.Run("GetMaxSchemaID_IndependentPerContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Create 3 schemas in .ctx-a
		for i := 0; i < 3; i++ {
			rec := &storage.SchemaRecord{
				Subject:     fmt.Sprintf("subj-%d", i),
				SchemaType:  storage.SchemaTypeAvro,
				Schema:      `{"type":"string"}`,
				Fingerprint: fmt.Sprintf("fp-max-a-%d", i),
			}
			if err := store.CreateSchema(ctx, ".ctx-a", rec); err != nil {
				t.Fatalf("CreateSchema %d in .ctx-a: %v", i, err)
			}
		}

		// Create 1 schema in .ctx-b
		recB := &storage.SchemaRecord{
			Subject:     "subj-b",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"int"}`,
			Fingerprint: "fp-max-b-0",
		}
		if err := store.CreateSchema(ctx, ".ctx-b", recB); err != nil {
			t.Fatalf("CreateSchema in .ctx-b: %v", err)
		}

		// GetMaxSchemaID for .ctx-a should be 3
		maxA, err := store.GetMaxSchemaID(ctx, ".ctx-a")
		if err != nil {
			t.Fatalf("GetMaxSchemaID .ctx-a: %v", err)
		}
		if maxA != 3 {
			t.Errorf(".ctx-a: expected max ID 3, got %d", maxA)
		}

		// GetMaxSchemaID for .ctx-b should be 1
		maxB, err := store.GetMaxSchemaID(ctx, ".ctx-b")
		if err != nil {
			t.Fatalf("GetMaxSchemaID .ctx-b: %v", err)
		}
		if maxB != 1 {
			t.Errorf(".ctx-b: expected max ID 1, got %d", maxB)
		}

		// Verify .ctx-a still returns 3 (not affected by .ctx-b)
		maxA2, err := store.GetMaxSchemaID(ctx, ".ctx-a")
		if err != nil {
			t.Fatalf("GetMaxSchemaID .ctx-a (recheck): %v", err)
		}
		if maxA2 != 3 {
			t.Errorf(".ctx-a (recheck): expected max ID 3, got %d", maxA2)
		}
	})

	// --- DeleteSubject isolation ---

	t.Run("DeleteSubject_IsolatedBetweenContexts", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Create the same subject name in both contexts
		recA := &storage.SchemaRecord{
			Subject:     "shared-name",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"string"}`,
			Fingerprint: "fp-del-iso-a",
		}
		if err := store.CreateSchema(ctx, ".ctx-a", recA); err != nil {
			t.Fatalf("CreateSchema in .ctx-a: %v", err)
		}

		recB := &storage.SchemaRecord{
			Subject:     "shared-name",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"int"}`,
			Fingerprint: "fp-del-iso-b",
		}
		if err := store.CreateSchema(ctx, ".ctx-b", recB); err != nil {
			t.Fatalf("CreateSchema in .ctx-b: %v", err)
		}

		// Delete subject in .ctx-a
		_, err := store.DeleteSubject(ctx, ".ctx-a", "shared-name", false)
		if err != nil {
			t.Fatalf("DeleteSubject in .ctx-a: %v", err)
		}

		// Verify subject no longer exists in .ctx-a
		existsA, err := store.SubjectExists(ctx, ".ctx-a", "shared-name")
		if err != nil {
			t.Fatalf("SubjectExists .ctx-a: %v", err)
		}
		if existsA {
			t.Error(".ctx-a: subject should not exist after delete")
		}

		// Verify subject still exists in .ctx-b
		existsB, err := store.SubjectExists(ctx, ".ctx-b", "shared-name")
		if err != nil {
			t.Fatalf("SubjectExists .ctx-b: %v", err)
		}
		if !existsB {
			t.Error(".ctx-b: subject should still exist")
		}

		// Verify we can still retrieve the schema in .ctx-b
		got, err := store.GetSchemaBySubjectVersion(ctx, ".ctx-b", "shared-name", 1)
		if err != nil {
			t.Fatalf("GetSchemaBySubjectVersion .ctx-b: %v", err)
		}
		if got.Schema != `{"type":"int"}` {
			t.Errorf(".ctx-b: expected int schema, got %q", got.Schema)
		}
	})

	// --- Import isolation ---

	t.Run("ImportIsolation_SameIDInDifferentContexts", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Import schema with ID 100 in .ctx-a
		recA := &storage.SchemaRecord{
			ID:          100,
			Subject:     "imported-subj",
			Version:     1,
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"string"}`,
			Fingerprint: "fp-imp-iso-a",
		}
		if err := store.ImportSchema(ctx, ".ctx-a", recA); err != nil {
			t.Fatalf("ImportSchema in .ctx-a: %v", err)
		}

		// Import schema with the same ID 100 in .ctx-b (should succeed - different context)
		recB := &storage.SchemaRecord{
			ID:          100,
			Subject:     "imported-subj",
			Version:     1,
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"int"}`,
			Fingerprint: "fp-imp-iso-b",
		}
		if err := store.ImportSchema(ctx, ".ctx-b", recB); err != nil {
			t.Fatalf("ImportSchema in .ctx-b with same ID: %v", err)
		}

		// Verify each context has its own schema at ID 100
		gotA, err := store.GetSchemaByID(ctx, ".ctx-a", 100)
		if err != nil {
			t.Fatalf("GetSchemaByID .ctx-a: %v", err)
		}
		if gotA.Schema != `{"type":"string"}` {
			t.Errorf(".ctx-a: expected string schema, got %q", gotA.Schema)
		}

		gotB, err := store.GetSchemaByID(ctx, ".ctx-b", 100)
		if err != nil {
			t.Fatalf("GetSchemaByID .ctx-b: %v", err)
		}
		if gotB.Schema != `{"type":"int"}` {
			t.Errorf(".ctx-b: expected int schema, got %q", gotB.Schema)
		}
	})

	// --- Additional cross-context isolation tests ---

	t.Run("GetSchemaBySubjectVersion_CrossContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{
			Subject:     "cross-subj",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"string"}`,
			Fingerprint: "fp-cross-sv",
		}
		if err := store.CreateSchema(ctx, ".ctx-a", rec); err != nil {
			t.Fatalf("CreateSchema in .ctx-a: %v", err)
		}

		// Should succeed in .ctx-a
		got, err := store.GetSchemaBySubjectVersion(ctx, ".ctx-a", "cross-subj", 1)
		if err != nil {
			t.Fatalf("GetSchemaBySubjectVersion .ctx-a: %v", err)
		}
		if got.Schema != `{"type":"string"}` {
			t.Errorf(".ctx-a: unexpected schema: %q", got.Schema)
		}

		// Should fail in .ctx-b (subject not found)
		_, err = store.GetSchemaBySubjectVersion(ctx, ".ctx-b", "cross-subj", 1)
		if !errors.Is(err, storage.ErrSubjectNotFound) {
			t.Errorf("GetSchemaBySubjectVersion .ctx-b: expected ErrSubjectNotFound, got %v", err)
		}
	})

	t.Run("GetLatestSchema_CrossContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Create two versions in .ctx-a
		r1 := &storage.SchemaRecord{Subject: "latest-subj", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-latest-a-1"}
		r2 := &storage.SchemaRecord{Subject: "latest-subj", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-latest-a-2"}
		store.CreateSchema(ctx, ".ctx-a", r1)
		store.CreateSchema(ctx, ".ctx-a", r2)

		// Create one version in .ctx-b
		r3 := &storage.SchemaRecord{Subject: "latest-subj", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"long"}`, Fingerprint: "fp-latest-b-1"}
		store.CreateSchema(ctx, ".ctx-b", r3)

		// Latest in .ctx-a should be version 2
		latestA, err := store.GetLatestSchema(ctx, ".ctx-a", "latest-subj")
		if err != nil {
			t.Fatalf("GetLatestSchema .ctx-a: %v", err)
		}
		if latestA.Version != 2 {
			t.Errorf(".ctx-a: expected latest version 2, got %d", latestA.Version)
		}

		// Latest in .ctx-b should be version 1
		latestB, err := store.GetLatestSchema(ctx, ".ctx-b", "latest-subj")
		if err != nil {
			t.Fatalf("GetLatestSchema .ctx-b: %v", err)
		}
		if latestB.Version != 1 {
			t.Errorf(".ctx-b: expected latest version 1, got %d", latestB.Version)
		}
	})

	t.Run("ListSchemas_ScopedToContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Create 2 schemas in .ctx-a and 1 in .ctx-b
		for i := 0; i < 2; i++ {
			rec := &storage.SchemaRecord{
				Subject:     fmt.Sprintf("ls-subj-%d", i),
				SchemaType:  storage.SchemaTypeAvro,
				Schema:      `{"type":"string"}`,
				Fingerprint: fmt.Sprintf("fp-lsctx-a-%d", i),
			}
			store.CreateSchema(ctx, ".ctx-a", rec)
		}
		recB := &storage.SchemaRecord{
			Subject:     "ls-subj-b",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"int"}`,
			Fingerprint: "fp-lsctx-b-0",
		}
		store.CreateSchema(ctx, ".ctx-b", recB)

		schemasA, err := store.ListSchemas(ctx, ".ctx-a", &storage.ListSchemasParams{})
		if err != nil {
			t.Fatalf("ListSchemas .ctx-a: %v", err)
		}
		if len(schemasA) != 2 {
			t.Errorf(".ctx-a: expected 2 schemas, got %d", len(schemasA))
		}

		schemasB, err := store.ListSchemas(ctx, ".ctx-b", &storage.ListSchemasParams{})
		if err != nil {
			t.Fatalf("ListSchemas .ctx-b: %v", err)
		}
		if len(schemasB) != 1 {
			t.Errorf(".ctx-b: expected 1 schema, got %d", len(schemasB))
		}
	})

	t.Run("NextID_IndependentPerContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Set different next IDs for different contexts
		if err := store.SetNextID(ctx, ".ctx-a", 100); err != nil {
			t.Fatalf("SetNextID .ctx-a: %v", err)
		}
		if err := store.SetNextID(ctx, ".ctx-b", 500); err != nil {
			t.Fatalf("SetNextID .ctx-b: %v", err)
		}

		idA, err := store.NextID(ctx, ".ctx-a")
		if err != nil {
			t.Fatalf("NextID .ctx-a: %v", err)
		}
		if idA != 100 {
			t.Errorf(".ctx-a: expected 100, got %d", idA)
		}

		idB, err := store.NextID(ctx, ".ctx-b")
		if err != nil {
			t.Fatalf("NextID .ctx-b: %v", err)
		}
		if idB != 500 {
			t.Errorf(".ctx-b: expected 500, got %d", idB)
		}

		// Advancing .ctx-a should not affect .ctx-b
		idA2, err := store.NextID(ctx, ".ctx-a")
		if err != nil {
			t.Fatalf("NextID .ctx-a (second): %v", err)
		}
		if idA2 != 101 {
			t.Errorf(".ctx-a (second): expected 101, got %d", idA2)
		}

		idB2, err := store.NextID(ctx, ".ctx-b")
		if err != nil {
			t.Fatalf("NextID .ctx-b (second): %v", err)
		}
		if idB2 != 501 {
			t.Errorf(".ctx-b (second): expected 501, got %d", idB2)
		}
	})

	t.Run("GetSchemaByFingerprint_CrossContext", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{
			Subject:     "fp-subj",
			SchemaType:  storage.SchemaTypeAvro,
			Schema:      `{"type":"string"}`,
			Fingerprint: "fp-cross-fp-1",
		}
		if err := store.CreateSchema(ctx, ".ctx-a", rec); err != nil {
			t.Fatalf("CreateSchema in .ctx-a: %v", err)
		}

		// Should find by fingerprint in .ctx-a
		got, err := store.GetSchemaByFingerprint(ctx, ".ctx-a", "fp-subj", "fp-cross-fp-1", false)
		if err != nil {
			t.Fatalf("GetSchemaByFingerprint .ctx-a: %v", err)
		}
		if got.Schema != `{"type":"string"}` {
			t.Errorf(".ctx-a: unexpected schema: %q", got.Schema)
		}

		// Should NOT find by fingerprint in .ctx-b
		_, err = store.GetSchemaByFingerprint(ctx, ".ctx-b", "fp-subj", "fp-cross-fp-1", false)
		if err == nil {
			t.Error("GetSchemaByFingerprint .ctx-b: expected error, got nil")
		}
	})
}
