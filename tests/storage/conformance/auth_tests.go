package conformance

import (
	"context"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// RunAuthTests tests all user and API key CRUD operations.
func RunAuthTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	// --- User Tests ---

	t.Run("CreateUser_AssignsID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{
			Username:     "alice",
			Email:        "alice@example.com",
			PasswordHash: "hash-alice",
			Role:         "admin",
			Enabled:      true,
		}
		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if user.ID == 0 {
			t.Error("expected non-zero ID")
		}
		if user.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set")
		}
	})

	t.Run("GetUserByID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "bob", PasswordHash: "hash", Role: "reader", Enabled: true}
		store.CreateUser(ctx, user)

		got, err := store.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetUserByID: %v", err)
		}
		if got.Username != "bob" {
			t.Errorf("expected username 'bob', got %q", got.Username)
		}
	})

	t.Run("GetUserByUsername", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "carol", PasswordHash: "hash", Role: "writer", Enabled: true}
		store.CreateUser(ctx, user)

		got, err := store.GetUserByUsername(ctx, "carol")
		if err != nil {
			t.Fatalf("GetUserByUsername: %v", err)
		}
		if got.ID != user.ID {
			t.Errorf("ID mismatch: %d vs %d", got.ID, user.ID)
		}
	})

	t.Run("UpdateUser", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "dave", PasswordHash: "hash", Role: "reader", Enabled: true}
		store.CreateUser(ctx, user)

		user.Role = "admin"
		user.Email = "dave@example.com"
		if err := store.UpdateUser(ctx, user); err != nil {
			t.Fatalf("UpdateUser: %v", err)
		}

		got, _ := store.GetUserByID(ctx, user.ID)
		if got.Role != "admin" {
			t.Errorf("expected role 'admin', got %q", got.Role)
		}
		if got.Email != "dave@example.com" {
			t.Errorf("expected email 'dave@example.com', got %q", got.Email)
		}
	})

	t.Run("UpdateUser_ChangeUsername", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "eve", PasswordHash: "hash", Role: "reader", Enabled: true}
		store.CreateUser(ctx, user)

		// Create a copy with the new username (avoids pointer aliasing with stored data)
		updated := &storage.UserRecord{
			ID:           user.ID,
			Username:     "eve-renamed",
			PasswordHash: user.PasswordHash,
			Role:         user.Role,
			Enabled:      user.Enabled,
		}
		if err := store.UpdateUser(ctx, updated); err != nil {
			t.Fatalf("UpdateUser: %v", err)
		}

		// Old username should not work
		_, err := store.GetUserByUsername(ctx, "eve")
		if err != storage.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound for old username, got %v", err)
		}

		// New username should work
		got, err := store.GetUserByUsername(ctx, "eve-renamed")
		if err != nil {
			t.Fatalf("GetUserByUsername(new): %v", err)
		}
		if got.ID != user.ID {
			t.Errorf("ID mismatch")
		}
	})

	t.Run("UpdateUser_DuplicateUsername", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		u1 := &storage.UserRecord{Username: "frank", PasswordHash: "hash", Role: "reader", Enabled: true}
		u2 := &storage.UserRecord{Username: "grace", PasswordHash: "hash", Role: "reader", Enabled: true}
		store.CreateUser(ctx, u1)
		store.CreateUser(ctx, u2)

		// Try to rename grace to frank (copy to avoid pointer aliasing)
		updated := &storage.UserRecord{
			ID:           u2.ID,
			Username:     "frank",
			PasswordHash: u2.PasswordHash,
			Role:         u2.Role,
			Enabled:      u2.Enabled,
		}
		err := store.UpdateUser(ctx, updated)
		if err != storage.ErrUserExists {
			t.Errorf("expected ErrUserExists, got %v", err)
		}
	})

	t.Run("DeleteUser", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "henry", PasswordHash: "hash", Role: "reader", Enabled: true}
		store.CreateUser(ctx, user)

		if err := store.DeleteUser(ctx, user.ID); err != nil {
			t.Fatalf("DeleteUser: %v", err)
		}

		_, err := store.GetUserByID(ctx, user.ID)
		if err != storage.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound after delete, got %v", err)
		}

		_, err = store.GetUserByUsername(ctx, "henry")
		if err != storage.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound by username after delete, got %v", err)
		}
	})

	t.Run("ListUsers", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		u1 := &storage.UserRecord{Username: "user-a", PasswordHash: "h", Role: "reader", Enabled: true}
		u2 := &storage.UserRecord{Username: "user-b", PasswordHash: "h", Role: "admin", Enabled: true}
		store.CreateUser(ctx, u1)
		store.CreateUser(ctx, u2)

		users, err := store.ListUsers(ctx)
		if err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		if len(users) != 2 {
			t.Errorf("expected 2 users, got %d", len(users))
		}
		// Should be sorted by ID
		if users[0].ID > users[1].ID {
			t.Error("users not sorted by ID")
		}
	})

	t.Run("ListUsers_Empty", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		users, err := store.ListUsers(ctx)
		if err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		if len(users) != 0 {
			t.Errorf("expected 0 users, got %d", len(users))
		}
	})

	// --- API Key Tests ---

	t.Run("CreateAPIKey_AssignsID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "keyowner", PasswordHash: "h", Role: "admin", Enabled: true}
		store.CreateUser(ctx, user)

		key := &storage.APIKeyRecord{
			UserID:    user.ID,
			KeyHash:   "hash-key-1",
			KeyPrefix: "ak_12345",
			Name:      "my-key",
			Role:      "admin",
			Enabled:   true,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		if err := store.CreateAPIKey(ctx, key); err != nil {
			t.Fatalf("CreateAPIKey: %v", err)
		}
		if key.ID == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("GetAPIKeyByID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "u-gid", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		key := &storage.APIKeyRecord{UserID: user.ID, KeyHash: "hash-gid", KeyPrefix: "ak_", Name: "k1", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, key)

		got, err := store.GetAPIKeyByID(ctx, key.ID)
		if err != nil {
			t.Fatalf("GetAPIKeyByID: %v", err)
		}
		if got.Name != "k1" {
			t.Errorf("expected name 'k1', got %q", got.Name)
		}
	})

	t.Run("GetAPIKeyByHash", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "u-gh", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		key := &storage.APIKeyRecord{UserID: user.ID, KeyHash: "hash-gh", KeyPrefix: "ak_", Name: "k2", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, key)

		got, err := store.GetAPIKeyByHash(ctx, "hash-gh")
		if err != nil {
			t.Fatalf("GetAPIKeyByHash: %v", err)
		}
		if got.ID != key.ID {
			t.Errorf("ID mismatch: %d vs %d", got.ID, key.ID)
		}
	})

	t.Run("GetAPIKeyByUserAndName", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "u-un", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		key := &storage.APIKeyRecord{UserID: user.ID, KeyHash: "hash-un", KeyPrefix: "ak_", Name: "prod-key", Role: "writer", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, key)

		got, err := store.GetAPIKeyByUserAndName(ctx, user.ID, "prod-key")
		if err != nil {
			t.Fatalf("GetAPIKeyByUserAndName: %v", err)
		}
		if got.ID != key.ID {
			t.Errorf("ID mismatch")
		}
	})

	t.Run("UpdateAPIKey", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "u-upd", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		key := &storage.APIKeyRecord{UserID: user.ID, KeyHash: "hash-upd", KeyPrefix: "ak_", Name: "k3", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, key)

		key.Enabled = false
		key.Role = "admin"
		if err := store.UpdateAPIKey(ctx, key); err != nil {
			t.Fatalf("UpdateAPIKey: %v", err)
		}

		got, _ := store.GetAPIKeyByID(ctx, key.ID)
		if got.Enabled {
			t.Error("expected key to be disabled")
		}
		if got.Role != "admin" {
			t.Errorf("expected role 'admin', got %q", got.Role)
		}
	})

	t.Run("UpdateAPIKey_ChangeHash", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "u-ch", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		key := &storage.APIKeyRecord{UserID: user.ID, KeyHash: "old-hash", KeyPrefix: "ak_", Name: "k4", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, key)

		// Create a copy with new hash (avoids pointer aliasing with stored data)
		updated := &storage.APIKeyRecord{
			ID:        key.ID,
			UserID:    key.UserID,
			KeyHash:   "new-hash",
			KeyPrefix: key.KeyPrefix,
			Name:      key.Name,
			Role:      key.Role,
			Enabled:   key.Enabled,
			ExpiresAt: key.ExpiresAt,
		}
		if err := store.UpdateAPIKey(ctx, updated); err != nil {
			t.Fatalf("UpdateAPIKey: %v", err)
		}

		// Old hash should not find the key
		_, err := store.GetAPIKeyByHash(ctx, "old-hash")
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("expected ErrAPIKeyNotFound for old hash, got %v", err)
		}

		// New hash should find it
		got, err := store.GetAPIKeyByHash(ctx, "new-hash")
		if err != nil {
			t.Fatalf("GetAPIKeyByHash(new): %v", err)
		}
		if got.ID != key.ID {
			t.Errorf("ID mismatch")
		}
	})

	t.Run("DeleteAPIKey", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "u-del", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		key := &storage.APIKeyRecord{UserID: user.ID, KeyHash: "hash-del", KeyPrefix: "ak_", Name: "k5", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, key)

		if err := store.DeleteAPIKey(ctx, key.ID); err != nil {
			t.Fatalf("DeleteAPIKey: %v", err)
		}

		_, err := store.GetAPIKeyByID(ctx, key.ID)
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("expected ErrAPIKeyNotFound after delete, got %v", err)
		}

		_, err = store.GetAPIKeyByHash(ctx, "hash-del")
		if err != storage.ErrAPIKeyNotFound {
			t.Errorf("expected ErrAPIKeyNotFound by hash after delete, got %v", err)
		}
	})

	t.Run("ListAPIKeys", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		u1 := &storage.UserRecord{Username: "u-la1", PasswordHash: "h", Role: "admin", Enabled: true}
		u2 := &storage.UserRecord{Username: "u-la2", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, u1); err != nil {
			t.Fatalf("CreateUser u1: %v", err)
		}
		if err := store.CreateUser(ctx, u2); err != nil {
			t.Fatalf("CreateUser u2: %v", err)
		}

		k1 := &storage.APIKeyRecord{UserID: u1.ID, KeyHash: "hash-la-1", KeyPrefix: "ak_", Name: "k1", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		k2 := &storage.APIKeyRecord{UserID: u2.ID, KeyHash: "hash-la-2", KeyPrefix: "ak_", Name: "k2", Role: "admin", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, k1)
		store.CreateAPIKey(ctx, k2)

		keys, err := store.ListAPIKeys(ctx)
		if err != nil {
			t.Fatalf("ListAPIKeys: %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("expected 2 keys, got %d", len(keys))
		}
	})

	t.Run("ListAPIKeysByUserID", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		u1 := &storage.UserRecord{Username: "u-lu1", PasswordHash: "h", Role: "admin", Enabled: true}
		u2 := &storage.UserRecord{Username: "u-lu2", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, u1); err != nil {
			t.Fatalf("CreateUser u1: %v", err)
		}
		if err := store.CreateUser(ctx, u2); err != nil {
			t.Fatalf("CreateUser u2: %v", err)
		}

		k1 := &storage.APIKeyRecord{UserID: u1.ID, KeyHash: "hash-lu-1", KeyPrefix: "ak_", Name: "k1", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		k2 := &storage.APIKeyRecord{UserID: u1.ID, KeyHash: "hash-lu-2", KeyPrefix: "ak_", Name: "k2", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		k3 := &storage.APIKeyRecord{UserID: u2.ID, KeyHash: "hash-lu-3", KeyPrefix: "ak_", Name: "k3", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, k1)
		store.CreateAPIKey(ctx, k2)
		store.CreateAPIKey(ctx, k3)

		keys, err := store.ListAPIKeysByUserID(ctx, u1.ID)
		if err != nil {
			t.Fatalf("ListAPIKeysByUserID: %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("expected 2 keys for user %d, got %d", u1.ID, len(keys))
		}
	})

	t.Run("UpdateAPIKeyLastUsed", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		user := &storage.UserRecord{Username: "u-alu", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		key := &storage.APIKeyRecord{UserID: user.ID, KeyHash: "hash-lu", KeyPrefix: "ak_", Name: "k-lu", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, key)

		if key.LastUsed != nil {
			t.Error("expected LastUsed to be nil initially")
		}

		if err := store.UpdateAPIKeyLastUsed(ctx, key.ID); err != nil {
			t.Fatalf("UpdateAPIKeyLastUsed: %v", err)
		}

		got, _ := store.GetAPIKeyByID(ctx, key.ID)
		if got.LastUsed == nil {
			t.Error("expected LastUsed to be set after update")
		}
	})

	t.Run("CreateAPIKey_DuplicateHash", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		u1 := &storage.UserRecord{Username: "u-dh1", PasswordHash: "h", Role: "admin", Enabled: true}
		u2 := &storage.UserRecord{Username: "u-dh2", PasswordHash: "h", Role: "admin", Enabled: true}
		if err := store.CreateUser(ctx, u1); err != nil {
			t.Fatalf("CreateUser u1: %v", err)
		}
		if err := store.CreateUser(ctx, u2); err != nil {
			t.Fatalf("CreateUser u2: %v", err)
		}

		k1 := &storage.APIKeyRecord{UserID: u1.ID, KeyHash: "dup-hash", KeyPrefix: "ak_", Name: "k1", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		k2 := &storage.APIKeyRecord{UserID: u2.ID, KeyHash: "dup-hash", KeyPrefix: "ak_", Name: "k2", Role: "reader", Enabled: true, ExpiresAt: time.Now().Add(time.Hour)}
		store.CreateAPIKey(ctx, k1)

		err := store.CreateAPIKey(ctx, k2)
		if err != storage.ErrAPIKeyExists {
			t.Errorf("expected ErrAPIKeyExists for duplicate hash, got %v", err)
		}
	})
}
