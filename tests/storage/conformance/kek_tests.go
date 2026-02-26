package conformance

import (
	"context"
	"errors"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// RunKEKTests tests all KEK (Key Encryption Key) CRUD operations.
func RunKEKTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	t.Run("CreateKEK_Basic", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "test-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "arn:aws:kms:us-east-1:123456789:key/abc-def",
			KmsProps: map[string]string{"region": "us-east-1"},
			Doc:      "Test KEK for encryption",
			Shared:   true,
		}
		if err := store.CreateKEK(ctx, kek); err != nil {
			t.Fatalf("CreateKEK: %v", err)
		}

		got, err := store.GetKEK(ctx, "test-kek", false)
		if err != nil {
			t.Fatalf("GetKEK: %v", err)
		}
		if got.Name != "test-kek" {
			t.Errorf("expected name 'test-kek', got %q", got.Name)
		}
		if got.KmsType != "aws-kms" {
			t.Errorf("expected kmsType 'aws-kms', got %q", got.KmsType)
		}
		if got.KmsKeyID != "arn:aws:kms:us-east-1:123456789:key/abc-def" {
			t.Errorf("expected kmsKeyId, got %q", got.KmsKeyID)
		}
		if got.KmsProps["region"] != "us-east-1" {
			t.Errorf("expected kmsProps region 'us-east-1', got %q", got.KmsProps["region"])
		}
		if got.Doc != "Test KEK for encryption" {
			t.Errorf("expected doc, got %q", got.Doc)
		}
		if !got.Shared {
			t.Error("expected shared=true")
		}
		if got.Deleted {
			t.Error("expected deleted=false for new KEK")
		}
		if got.Ts == 0 {
			t.Error("expected non-zero timestamp")
		}
	})

	t.Run("CreateKEK_Duplicate", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "dup-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
		}
		if err := store.CreateKEK(ctx, kek); err != nil {
			t.Fatalf("CreateKEK: %v", err)
		}

		dup := &storage.KEKRecord{
			Name:     "dup-kek",
			KmsType:  "gcp-kms",
			KmsKeyID: "key-2",
		}
		err := store.CreateKEK(ctx, dup)
		if !errors.Is(err, storage.ErrKEKExists) {
			t.Errorf("expected ErrKEKExists, got %v", err)
		}
	})

	t.Run("GetKEK_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetKEK(ctx, "nonexistent-kek", false)
		if !errors.Is(err, storage.ErrKEKNotFound) {
			t.Errorf("expected ErrKEKNotFound, got %v", err)
		}
	})

	t.Run("GetKEK_ExcludesDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "del-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
		}
		if err := store.CreateKEK(ctx, kek); err != nil {
			t.Fatalf("CreateKEK: %v", err)
		}

		// Soft-delete
		if err := store.DeleteKEK(ctx, "del-kek", false); err != nil {
			t.Fatalf("DeleteKEK(soft): %v", err)
		}

		// Without includeDeleted, should not be found
		_, err := store.GetKEK(ctx, "del-kek", false)
		if !errors.Is(err, storage.ErrKEKNotFound) {
			t.Errorf("expected ErrKEKNotFound for soft-deleted KEK, got %v", err)
		}
	})

	t.Run("GetKEK_IncludesDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "del-kek-inc",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
		}
		if err := store.CreateKEK(ctx, kek); err != nil {
			t.Fatalf("CreateKEK: %v", err)
		}

		// Soft-delete
		if err := store.DeleteKEK(ctx, "del-kek-inc", false); err != nil {
			t.Fatalf("DeleteKEK(soft): %v", err)
		}

		// With includeDeleted, should be found
		got, err := store.GetKEK(ctx, "del-kek-inc", true)
		if err != nil {
			t.Fatalf("GetKEK(includeDeleted=true): %v", err)
		}
		if !got.Deleted {
			t.Error("expected deleted=true")
		}
	})

	t.Run("UpdateKEK", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "upd-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
			Doc:      "original doc",
			Shared:   false,
		}
		if err := store.CreateKEK(ctx, kek); err != nil {
			t.Fatalf("CreateKEK: %v", err)
		}

		updated := &storage.KEKRecord{
			Name:     "upd-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
			KmsProps: map[string]string{"region": "eu-west-1"},
			Doc:      "updated doc",
			Shared:   true,
		}
		if err := store.UpdateKEK(ctx, updated); err != nil {
			t.Fatalf("UpdateKEK: %v", err)
		}

		got, err := store.GetKEK(ctx, "upd-kek", false)
		if err != nil {
			t.Fatalf("GetKEK: %v", err)
		}
		if got.Doc != "updated doc" {
			t.Errorf("expected doc 'updated doc', got %q", got.Doc)
		}
		if !got.Shared {
			t.Error("expected shared=true after update")
		}
		if got.KmsProps["region"] != "eu-west-1" {
			t.Errorf("expected kmsProps region 'eu-west-1', got %q", got.KmsProps["region"])
		}
	})

	t.Run("UpdateKEK_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "nonexistent-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
		}
		err := store.UpdateKEK(ctx, kek)
		if !errors.Is(err, storage.ErrKEKNotFound) {
			t.Errorf("expected ErrKEKNotFound, got %v", err)
		}
	})

	t.Run("DeleteKEK_Soft", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "soft-del-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
		}
		if err := store.CreateKEK(ctx, kek); err != nil {
			t.Fatalf("CreateKEK: %v", err)
		}

		if err := store.DeleteKEK(ctx, "soft-del-kek", false); err != nil {
			t.Fatalf("DeleteKEK(soft): %v", err)
		}

		// Should not appear without includeDeleted
		_, err := store.GetKEK(ctx, "soft-del-kek", false)
		if !errors.Is(err, storage.ErrKEKNotFound) {
			t.Errorf("expected ErrKEKNotFound after soft delete, got %v", err)
		}

		// Should appear with includeDeleted
		got, err := store.GetKEK(ctx, "soft-del-kek", true)
		if err != nil {
			t.Fatalf("GetKEK(includeDeleted=true): %v", err)
		}
		if !got.Deleted {
			t.Error("expected deleted=true after soft delete")
		}
	})

	t.Run("DeleteKEK_Permanent", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "perm-del-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
		}
		if err := store.CreateKEK(ctx, kek); err != nil {
			t.Fatalf("CreateKEK: %v", err)
		}

		if err := store.DeleteKEK(ctx, "perm-del-kek", true); err != nil {
			t.Fatalf("DeleteKEK(permanent): %v", err)
		}

		// Should not be found even with includeDeleted
		_, err := store.GetKEK(ctx, "perm-del-kek", true)
		if !errors.Is(err, storage.ErrKEKNotFound) {
			t.Errorf("expected ErrKEKNotFound after permanent delete, got %v", err)
		}
	})

	t.Run("UndeleteKEK", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "undel-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
		}
		if err := store.CreateKEK(ctx, kek); err != nil {
			t.Fatalf("CreateKEK: %v", err)
		}

		// Soft-delete
		if err := store.DeleteKEK(ctx, "undel-kek", false); err != nil {
			t.Fatalf("DeleteKEK(soft): %v", err)
		}

		// Verify soft-deleted
		_, err := store.GetKEK(ctx, "undel-kek", false)
		if !errors.Is(err, storage.ErrKEKNotFound) {
			t.Fatalf("expected ErrKEKNotFound after soft delete, got %v", err)
		}

		// Undelete
		if err := store.UndeleteKEK(ctx, "undel-kek"); err != nil {
			t.Fatalf("UndeleteKEK: %v", err)
		}

		// Should now be visible again
		got, err := store.GetKEK(ctx, "undel-kek", false)
		if err != nil {
			t.Fatalf("GetKEK after undelete: %v", err)
		}
		if got.Deleted {
			t.Error("expected deleted=false after undelete")
		}
	})

	t.Run("UndeleteKEK_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		// Undeleting a KEK that does not exist
		err := store.UndeleteKEK(ctx, "nonexistent-kek")
		if !errors.Is(err, storage.ErrKEKNotFound) {
			t.Errorf("expected ErrKEKNotFound, got %v", err)
		}
	})

	t.Run("UndeleteKEK_NotDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		kek := &storage.KEKRecord{
			Name:     "active-kek",
			KmsType:  "aws-kms",
			KmsKeyID: "key-1",
		}
		if err := store.CreateKEK(ctx, kek); err != nil {
			t.Fatalf("CreateKEK: %v", err)
		}

		// Undeleting a KEK that is not soft-deleted should return error
		err := store.UndeleteKEK(ctx, "active-kek")
		if !errors.Is(err, storage.ErrKEKNotFound) {
			t.Errorf("expected ErrKEKNotFound for non-deleted KEK, got %v", err)
		}
	})

	t.Run("ListKEKs", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		names := []string{"kek-charlie", "kek-alice", "kek-bob"}
		for _, name := range names {
			kek := &storage.KEKRecord{
				Name:     name,
				KmsType:  "aws-kms",
				KmsKeyID: "key-" + name,
			}
			if err := store.CreateKEK(ctx, kek); err != nil {
				t.Fatalf("CreateKEK(%s): %v", name, err)
			}
		}

		keks, err := store.ListKEKs(ctx, false)
		if err != nil {
			t.Fatalf("ListKEKs: %v", err)
		}
		if len(keks) != 3 {
			t.Fatalf("expected 3 KEKs, got %d", len(keks))
		}
		// Verify sorted by name
		if keks[0].Name != "kek-alice" {
			t.Errorf("expected first KEK 'kek-alice', got %q", keks[0].Name)
		}
		if keks[1].Name != "kek-bob" {
			t.Errorf("expected second KEK 'kek-bob', got %q", keks[1].Name)
		}
		if keks[2].Name != "kek-charlie" {
			t.Errorf("expected third KEK 'kek-charlie', got %q", keks[2].Name)
		}
	})

	t.Run("ListKEKs_ExcludesDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		for _, name := range []string{"kek-a", "kek-b", "kek-c"} {
			kek := &storage.KEKRecord{
				Name:     name,
				KmsType:  "aws-kms",
				KmsKeyID: "key-" + name,
			}
			if err := store.CreateKEK(ctx, kek); err != nil {
				t.Fatalf("CreateKEK(%s): %v", name, err)
			}
		}

		// Soft-delete kek-b
		if err := store.DeleteKEK(ctx, "kek-b", false); err != nil {
			t.Fatalf("DeleteKEK: %v", err)
		}

		keks, err := store.ListKEKs(ctx, false)
		if err != nil {
			t.Fatalf("ListKEKs: %v", err)
		}
		if len(keks) != 2 {
			t.Fatalf("expected 2 KEKs (excluding deleted), got %d", len(keks))
		}
		for _, kek := range keks {
			if kek.Name == "kek-b" {
				t.Error("soft-deleted KEK 'kek-b' should not appear in list")
			}
		}
	})

	t.Run("ListKEKs_IncludesDeleted", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		for _, name := range []string{"kek-x", "kek-y"} {
			kek := &storage.KEKRecord{
				Name:     name,
				KmsType:  "aws-kms",
				KmsKeyID: "key-" + name,
			}
			if err := store.CreateKEK(ctx, kek); err != nil {
				t.Fatalf("CreateKEK(%s): %v", name, err)
			}
		}

		// Soft-delete kek-x
		if err := store.DeleteKEK(ctx, "kek-x", false); err != nil {
			t.Fatalf("DeleteKEK: %v", err)
		}

		keks, err := store.ListKEKs(ctx, true)
		if err != nil {
			t.Fatalf("ListKEKs(includeDeleted=true): %v", err)
		}
		if len(keks) != 2 {
			t.Fatalf("expected 2 KEKs (including deleted), got %d", len(keks))
		}
	})
}
