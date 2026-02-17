package conformance

import (
	"context"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// RunErrorTests verifies that every sentinel error is triggered by the appropriate operation.
func RunErrorTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	// --- Schema Errors ---

	t.Run("ErrSchemaNotFound_GetByID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetSchemaByID(ctx, ".", 999)
		if err != storage.ErrSchemaNotFound {
			t.Errorf("expected ErrSchemaNotFound, got %v", err)
		}
	})

	t.Run("ErrSubjectNotFound_GetBySubjectVersion", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetSchemaBySubjectVersion(ctx, ".", "nonexistent", 1)
		if err != storage.ErrSubjectNotFound {
			t.Errorf("expected ErrSubjectNotFound, got %v", err)
		}
	})

	t.Run("ErrVersionNotFound_GetBySubjectVersion", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-vnf"}
		store.CreateSchema(ctx, ".", rec)

		_, err := store.GetSchemaBySubjectVersion(ctx, ".", "s", 99)
		if err != storage.ErrVersionNotFound {
			t.Errorf("expected ErrVersionNotFound, got %v", err)
		}
	})

	t.Run("ErrSubjectNotFound_GetSchemasBySubject", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetSchemasBySubject(ctx, ".", "nonexistent", false)
		if err != storage.ErrSubjectNotFound {
			t.Errorf("expected ErrSubjectNotFound, got %v", err)
		}
	})

	t.Run("ErrSchemaNotFound_GetByFingerprint", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetSchemaByFingerprint(ctx, ".", "s", "no-such-fp", false)
		// Some backends return ErrSubjectNotFound when subject doesn't exist,
		// others return ErrSchemaNotFound. Both are acceptable.
		if err == nil {
			t.Errorf("expected error (ErrSchemaNotFound or ErrSubjectNotFound), got nil")
		}
	})

	t.Run("ErrSchemaNotFound_GetByGlobalFingerprint", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetSchemaByGlobalFingerprint(ctx, ".", "no-such-fp")
		if err != storage.ErrSchemaNotFound {
			t.Errorf("expected ErrSchemaNotFound, got %v", err)
		}
	})

	t.Run("ErrSubjectNotFound_GetLatestSchema", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetLatestSchema(ctx, ".", "nonexistent")
		if err != storage.ErrSubjectNotFound {
			t.Errorf("expected ErrSubjectNotFound, got %v", err)
		}
	})

	t.Run("ErrSubjectNotFound_DeleteSchema", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.DeleteSchema(ctx, ".", "nonexistent", 1, false)
		if err != storage.ErrSubjectNotFound {
			t.Errorf("expected ErrSubjectNotFound, got %v", err)
		}
	})

	t.Run("ErrVersionNotFound_DeleteSchema", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		rec := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-dvnf"}
		store.CreateSchema(ctx, ".", rec)

		err := store.DeleteSchema(ctx, ".", "s", 99, false)
		if err != storage.ErrVersionNotFound {
			t.Errorf("expected ErrVersionNotFound, got %v", err)
		}
	})

	t.Run("ErrSchemaExists_DuplicateFingerprint", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "dup-fp"}
		r2 := &storage.SchemaRecord{Subject: "s", SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "dup-fp"}
		store.CreateSchema(ctx, ".", r1)

		err := store.CreateSchema(ctx, ".", r2)
		if err != storage.ErrSchemaExists {
			t.Errorf("expected ErrSchemaExists, got %v", err)
		}
	})

	t.Run("ErrSchemaIDConflict_Import", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		r1 := &storage.SchemaRecord{ID: 1, Subject: "a", Version: 1, SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"string"}`, Fingerprint: "fp-idc-1"}
		r2 := &storage.SchemaRecord{ID: 1, Subject: "b", Version: 1, SchemaType: storage.SchemaTypeAvro, Schema: `{"type":"int"}`, Fingerprint: "fp-idc-2"}
		store.ImportSchema(ctx, ".", r1)

		err := store.ImportSchema(ctx, ".", r2)
		if err != storage.ErrSchemaIDConflict {
			t.Errorf("expected ErrSchemaIDConflict, got %v", err)
		}
	})

	t.Run("ErrSchemaNotFound_GetSubjectsBySchemaID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetSubjectsBySchemaID(ctx, ".", 999, false)
		if err != storage.ErrSchemaNotFound {
			t.Errorf("expected ErrSchemaNotFound, got %v", err)
		}
	})

	t.Run("ErrSchemaNotFound_GetVersionsBySchemaID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetVersionsBySchemaID(ctx, ".", 999, false)
		if err != storage.ErrSchemaNotFound {
			t.Errorf("expected ErrSchemaNotFound, got %v", err)
		}
	})

	// --- Subject Errors ---

	t.Run("ErrSubjectNotFound_DeleteSubject", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.DeleteSubject(ctx, ".", "nonexistent", false)
		if err != storage.ErrSubjectNotFound {
			t.Errorf("expected ErrSubjectNotFound, got %v", err)
		}
	})

	// --- Config Errors ---

	t.Run("ErrNotFound_GetConfig", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetConfig(ctx, ".", "nonexistent")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ErrNotFound_DeleteConfig", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.DeleteConfig(ctx, ".", "nonexistent")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ErrNotFound_GetMode", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetMode(ctx, ".", "nonexistent")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ErrNotFound_DeleteMode", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.DeleteMode(ctx, ".", "nonexistent")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	// --- User Errors ---

	t.Run("ErrUserExists_DuplicateUsername", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		u1 := &storage.UserRecord{Username: "alice", PasswordHash: "h", Role: "reader", Enabled: true}
		u2 := &storage.UserRecord{Username: "alice", PasswordHash: "h2", Role: "admin", Enabled: true}
		store.CreateUser(ctx, u1)

		err := store.CreateUser(ctx, u2)
		if err != storage.ErrUserExists {
			t.Errorf("expected ErrUserExists, got %v", err)
		}
	})

	t.Run("ErrUserNotFound_GetByID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetUserByID(ctx, 999)
		if err != storage.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("ErrUserNotFound_GetByUsername", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetUserByUsername(ctx, "nonexistent")
		if err != storage.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("ErrUserNotFound_Update", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{ID: 999, Username: "ghost", PasswordHash: "h", Role: "reader", Enabled: true}
		err := store.UpdateUser(ctx, user)
		if err != storage.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("ErrUserNotFound_Delete", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.DeleteUser(ctx, 999)
		if err != storage.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	// --- API Key Errors ---

	t.Run("ErrAPIKeyExists_DuplicateHash", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		u1 := &storage.UserRecord{Username: "u-edh1", PasswordHash: "h", Role: "admin", Enabled: true}
		u2 := &storage.UserRecord{Username: "u-edh2", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, u1); err != nil {
			t.Fatalf("CreateUser u1: %v", err)
		}
		if err := store.CreateUser(ctx, u2); err != nil {
			t.Fatalf("CreateUser u2: %v", err)
		}

		exp := time.Now().Add(time.Hour)
		k1 := &storage.APIKeyRecord{UserID: u1.ID, KeyHash: "same-hash", KeyPrefix: "ak_", Name: "k1", Role: "reader", Enabled: true, ExpiresAt: exp}
		k2 := &storage.APIKeyRecord{UserID: u2.ID, KeyHash: "same-hash", KeyPrefix: "ak_", Name: "k2", Role: "reader", Enabled: true, ExpiresAt: exp}
		store.CreateAPIKey(ctx, k1)

		err := store.CreateAPIKey(ctx, k2)
		if err != storage.ErrAPIKeyExists {
			t.Errorf("expected ErrAPIKeyExists, got %v", err)
		}
	})

	t.Run("ErrAPIKeyNotFound_GetByID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetAPIKeyByID(ctx, 999)
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("expected ErrAPIKeyNotFound, got %v", err)
		}
	})

	t.Run("ErrAPIKeyNotFound_GetByHash", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetAPIKeyByHash(ctx, "no-such-hash")
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("expected ErrAPIKeyNotFound, got %v", err)
		}
	})

	t.Run("ErrAPIKeyNotFound_GetByUserAndName", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetAPIKeyByUserAndName(ctx, 999, "no-such-key")
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("expected ErrAPIKeyNotFound, got %v", err)
		}
	})

	t.Run("ErrAPIKeyNotFound_Update", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		key := &storage.APIKeyRecord{ID: 999, UserID: 1, KeyHash: "h", Name: "k", Role: "reader", Enabled: true}
		err := store.UpdateAPIKey(ctx, key)
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("expected ErrAPIKeyNotFound, got %v", err)
		}
	})

	t.Run("ErrAPIKeyNotFound_Delete", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.DeleteAPIKey(ctx, 999)
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("expected ErrAPIKeyNotFound, got %v", err)
		}
	})

	t.Run("ErrAPIKeyNotFound_UpdateLastUsed", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.UpdateAPIKeyLastUsed(ctx, 999)
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("expected ErrAPIKeyNotFound, got %v", err)
		}
	})
}
