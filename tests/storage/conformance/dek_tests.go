package conformance

import (
	"context"
	"errors"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// createTestKEK is a helper that creates a KEK for DEK tests.
func createTestKEK(t *testing.T, ctx context.Context, store storage.Storage, name string) {
	t.Helper()
	kek := &storage.KEKRecord{
		Name:     name,
		KmsType:  "aws-kms",
		KmsKeyID: "key-" + name,
	}
	if err := store.CreateKEK(ctx, kek); err != nil {
		t.Fatalf("CreateKEK(%s): %v", name, err)
	}
}

// RunDEKTests tests all DEK (Data Encryption Key) CRUD operations.
func RunDEKTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	t.Run("CreateDEK_Basic", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "test-kek")

		dek := &storage.DEKRecord{
			KEKName:              "test-kek",
			Subject:              "test-subject",
			Algorithm:            "AES256_GCM",
			EncryptedKeyMaterial: "encrypted-material-abc",
		}
		if err := store.CreateDEK(ctx, dek); err != nil {
			t.Fatalf("CreateDEK: %v", err)
		}

		if dek.Version != 1 {
			t.Errorf("expected auto-assigned version 1, got %d", dek.Version)
		}
		if dek.Ts == 0 {
			t.Error("expected non-zero timestamp")
		}

		got, err := store.GetDEK(ctx, "test-kek", "test-subject", 1, "AES256_GCM", false)
		if err != nil {
			t.Fatalf("GetDEK: %v", err)
		}
		if got.KEKName != "test-kek" {
			t.Errorf("expected kekName 'test-kek', got %q", got.KEKName)
		}
		if got.Subject != "test-subject" {
			t.Errorf("expected subject 'test-subject', got %q", got.Subject)
		}
		if got.Version != 1 {
			t.Errorf("expected version 1, got %d", got.Version)
		}
		if got.Algorithm != "AES256_GCM" {
			t.Errorf("expected algorithm 'AES256_GCM', got %q", got.Algorithm)
		}
		if got.EncryptedKeyMaterial != "encrypted-material-abc" {
			t.Errorf("expected encryptedKeyMaterial, got %q", got.EncryptedKeyMaterial)
		}
		if got.Deleted {
			t.Error("expected deleted=false for new DEK")
		}
	})

	t.Run("CreateDEK_AutoVersion", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "ver-kek")

		dek1 := &storage.DEKRecord{
			KEKName:              "ver-kek",
			Subject:              "test-subject",
			Algorithm:            "AES256_GCM",
			EncryptedKeyMaterial: "material-v1",
		}
		if err := store.CreateDEK(ctx, dek1); err != nil {
			t.Fatalf("CreateDEK v1: %v", err)
		}
		if dek1.Version != 1 {
			t.Errorf("expected version 1, got %d", dek1.Version)
		}

		dek2 := &storage.DEKRecord{
			KEKName:              "ver-kek",
			Subject:              "test-subject",
			Algorithm:            "AES256_GCM",
			EncryptedKeyMaterial: "material-v2",
		}
		if err := store.CreateDEK(ctx, dek2); err != nil {
			t.Fatalf("CreateDEK v2: %v", err)
		}
		if dek2.Version != 2 {
			t.Errorf("expected version 2, got %d", dek2.Version)
		}
	})

	t.Run("CreateDEK_DefaultAlgorithm", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "alg-kek")

		dek := &storage.DEKRecord{
			KEKName:              "alg-kek",
			Subject:              "test-subject",
			Algorithm:            "AES256_GCM",
			EncryptedKeyMaterial: "material",
		}
		if err := store.CreateDEK(ctx, dek); err != nil {
			t.Fatalf("CreateDEK: %v", err)
		}

		// Verify the algorithm is stored as provided
		got, err := store.GetDEK(ctx, "alg-kek", "test-subject", 1, "AES256_GCM", false)
		if err != nil {
			t.Fatalf("GetDEK: %v", err)
		}
		if got.Algorithm != "AES256_GCM" {
			t.Errorf("expected algorithm 'AES256_GCM', got %q", got.Algorithm)
		}
	})

	t.Run("GetDEK_ByVersion", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "ver-get-kek")

		for i := 0; i < 3; i++ {
			dek := &storage.DEKRecord{
				KEKName:              "ver-get-kek",
				Subject:              "test-subject",
				Algorithm:            "AES256_GCM",
				EncryptedKeyMaterial: "material",
			}
			if err := store.CreateDEK(ctx, dek); err != nil {
				t.Fatalf("CreateDEK: %v", err)
			}
		}

		got, err := store.GetDEK(ctx, "ver-get-kek", "test-subject", 2, "AES256_GCM", false)
		if err != nil {
			t.Fatalf("GetDEK(version=2): %v", err)
		}
		if got.Version != 2 {
			t.Errorf("expected version 2, got %d", got.Version)
		}
	})

	t.Run("GetDEK_Latest", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "lat-kek")

		for i := 0; i < 3; i++ {
			dek := &storage.DEKRecord{
				KEKName:              "lat-kek",
				Subject:              "test-subject",
				Algorithm:            "AES256_GCM",
				EncryptedKeyMaterial: "material",
			}
			if err := store.CreateDEK(ctx, dek); err != nil {
				t.Fatalf("CreateDEK: %v", err)
			}
		}

		// Version <= 0 should return the latest
		got, err := store.GetDEK(ctx, "lat-kek", "test-subject", -1, "AES256_GCM", false)
		if err != nil {
			t.Fatalf("GetDEK(version=-1): %v", err)
		}
		if got.Version != 3 {
			t.Errorf("expected latest version 3, got %d", got.Version)
		}

		// Version 0 should also return the latest
		got, err = store.GetDEK(ctx, "lat-kek", "test-subject", 0, "AES256_GCM", false)
		if err != nil {
			t.Fatalf("GetDEK(version=0): %v", err)
		}
		if got.Version != 3 {
			t.Errorf("expected latest version 3, got %d", got.Version)
		}
	})

	t.Run("GetDEK_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetDEK(ctx, "nonexistent-kek", "test-subject", 1, "AES256_GCM", false)
		if !errors.Is(err, storage.ErrDEKNotFound) {
			t.Errorf("expected ErrDEKNotFound, got %v", err)
		}
	})

	t.Run("ListDEKs", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "list-kek")

		subjects := []string{"subject-c", "subject-a", "subject-b"}
		for _, subj := range subjects {
			dek := &storage.DEKRecord{
				KEKName:              "list-kek",
				Subject:              subj,
				Algorithm:            "AES256_GCM",
				EncryptedKeyMaterial: "material",
			}
			if err := store.CreateDEK(ctx, dek); err != nil {
				t.Fatalf("CreateDEK(%s): %v", subj, err)
			}
		}

		got, err := store.ListDEKs(ctx, "list-kek", false)
		if err != nil {
			t.Fatalf("ListDEKs: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 subjects, got %d", len(got))
		}
		// Verify sorted
		if got[0] != "subject-a" {
			t.Errorf("expected first subject 'subject-a', got %q", got[0])
		}
		if got[1] != "subject-b" {
			t.Errorf("expected second subject 'subject-b', got %q", got[1])
		}
		if got[2] != "subject-c" {
			t.Errorf("expected third subject 'subject-c', got %q", got[2])
		}
	})

	t.Run("ListDEKVersions", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "ver-list-kek")

		for i := 0; i < 3; i++ {
			dek := &storage.DEKRecord{
				KEKName:              "ver-list-kek",
				Subject:              "test-subject",
				Algorithm:            "AES256_GCM",
				EncryptedKeyMaterial: "material",
			}
			if err := store.CreateDEK(ctx, dek); err != nil {
				t.Fatalf("CreateDEK: %v", err)
			}
		}

		versions, err := store.ListDEKVersions(ctx, "ver-list-kek", "test-subject", "AES256_GCM", false)
		if err != nil {
			t.Fatalf("ListDEKVersions: %v", err)
		}
		if len(versions) != 3 {
			t.Fatalf("expected 3 versions, got %d", len(versions))
		}
		// Verify sorted
		if versions[0] != 1 || versions[1] != 2 || versions[2] != 3 {
			t.Errorf("expected [1 2 3], got %v", versions)
		}
	})

	t.Run("DeleteDEK_Soft", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "soft-del-dek-kek")

		dek := &storage.DEKRecord{
			KEKName:              "soft-del-dek-kek",
			Subject:              "test-subject",
			Algorithm:            "AES256_GCM",
			EncryptedKeyMaterial: "material",
		}
		if err := store.CreateDEK(ctx, dek); err != nil {
			t.Fatalf("CreateDEK: %v", err)
		}

		if err := store.DeleteDEK(ctx, "soft-del-dek-kek", "test-subject", 1, "AES256_GCM", false); err != nil {
			t.Fatalf("DeleteDEK(soft): %v", err)
		}

		// Should not be found without includeDeleted
		_, err := store.GetDEK(ctx, "soft-del-dek-kek", "test-subject", 1, "AES256_GCM", false)
		if !errors.Is(err, storage.ErrDEKNotFound) {
			t.Errorf("expected ErrDEKNotFound after soft delete, got %v", err)
		}

		// Should be found with includeDeleted
		got, err := store.GetDEK(ctx, "soft-del-dek-kek", "test-subject", 1, "AES256_GCM", true)
		if err != nil {
			t.Fatalf("GetDEK(includeDeleted=true): %v", err)
		}
		if !got.Deleted {
			t.Error("expected deleted=true after soft delete")
		}
	})

	t.Run("DeleteDEK_Permanent", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "perm-del-dek-kek")

		dek := &storage.DEKRecord{
			KEKName:              "perm-del-dek-kek",
			Subject:              "test-subject",
			Algorithm:            "AES256_GCM",
			EncryptedKeyMaterial: "material",
		}
		if err := store.CreateDEK(ctx, dek); err != nil {
			t.Fatalf("CreateDEK: %v", err)
		}

		if err := store.DeleteDEK(ctx, "perm-del-dek-kek", "test-subject", 1, "AES256_GCM", true); err != nil {
			t.Fatalf("DeleteDEK(permanent): %v", err)
		}

		// Should not be found even with includeDeleted
		_, err := store.GetDEK(ctx, "perm-del-dek-kek", "test-subject", 1, "AES256_GCM", true)
		if !errors.Is(err, storage.ErrDEKNotFound) {
			t.Errorf("expected ErrDEKNotFound after permanent delete, got %v", err)
		}
	})

	t.Run("UndeleteDEK", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "undel-dek-kek")

		dek := &storage.DEKRecord{
			KEKName:              "undel-dek-kek",
			Subject:              "test-subject",
			Algorithm:            "AES256_GCM",
			EncryptedKeyMaterial: "material",
		}
		if err := store.CreateDEK(ctx, dek); err != nil {
			t.Fatalf("CreateDEK: %v", err)
		}

		// Soft-delete
		if err := store.DeleteDEK(ctx, "undel-dek-kek", "test-subject", 1, "AES256_GCM", false); err != nil {
			t.Fatalf("DeleteDEK(soft): %v", err)
		}

		// Verify deleted
		_, err := store.GetDEK(ctx, "undel-dek-kek", "test-subject", 1, "AES256_GCM", false)
		if !errors.Is(err, storage.ErrDEKNotFound) {
			t.Fatalf("expected ErrDEKNotFound after soft delete, got %v", err)
		}

		// Undelete
		if err := store.UndeleteDEK(ctx, "undel-dek-kek", "test-subject", 1, "AES256_GCM"); err != nil {
			t.Fatalf("UndeleteDEK: %v", err)
		}

		// Should now be visible again
		got, err := store.GetDEK(ctx, "undel-dek-kek", "test-subject", 1, "AES256_GCM", false)
		if err != nil {
			t.Fatalf("GetDEK after undelete: %v", err)
		}
		if got.Deleted {
			t.Error("expected deleted=false after undelete")
		}
	})

	t.Run("DeleteDEK_AllVersions", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		createTestKEK(t, ctx, store, "all-del-kek")

		// Create 3 versions
		for i := 0; i < 3; i++ {
			dek := &storage.DEKRecord{
				KEKName:              "all-del-kek",
				Subject:              "test-subject",
				Algorithm:            "AES256_GCM",
				EncryptedKeyMaterial: "material",
			}
			if err := store.CreateDEK(ctx, dek); err != nil {
				t.Fatalf("CreateDEK: %v", err)
			}
		}

		// Verify 3 versions exist
		versions, err := store.ListDEKVersions(ctx, "all-del-kek", "test-subject", "AES256_GCM", false)
		if err != nil {
			t.Fatalf("ListDEKVersions: %v", err)
		}
		if len(versions) != 3 {
			t.Fatalf("expected 3 versions before delete, got %d", len(versions))
		}

		// Delete all versions (version=-1 is soft delete all)
		if err := store.DeleteDEK(ctx, "all-del-kek", "test-subject", -1, "AES256_GCM", false); err != nil {
			t.Fatalf("DeleteDEK(version=-1): %v", err)
		}

		// All versions should be soft-deleted (not visible without includeDeleted)
		versions, err = store.ListDEKVersions(ctx, "all-del-kek", "test-subject", "AES256_GCM", false)
		if err != nil {
			t.Fatalf("ListDEKVersions after delete all: %v", err)
		}
		if len(versions) != 0 {
			t.Errorf("expected 0 versions after soft-deleting all, got %d", len(versions))
		}

		// All versions should still be visible with includeDeleted
		versions, err = store.ListDEKVersions(ctx, "all-del-kek", "test-subject", "AES256_GCM", true)
		if err != nil {
			t.Fatalf("ListDEKVersions(includeDeleted=true): %v", err)
		}
		if len(versions) != 3 {
			t.Errorf("expected 3 versions with includeDeleted=true, got %d", len(versions))
		}
	})
}
