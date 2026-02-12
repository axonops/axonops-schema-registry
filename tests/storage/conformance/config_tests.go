package conformance

import (
	"context"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// RunConfigTests tests all config and mode operations.
func RunConfigTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	// --- Global Config ---

	t.Run("GetGlobalConfig_Default", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		config, err := store.GetGlobalConfig(ctx)
		if err == storage.ErrNotFound {
			// Memory backend: no seeded default, returns ErrNotFound
		} else if err != nil {
			t.Fatalf("GetGlobalConfig: unexpected error %v", err)
		} else {
			// DB backends seed a default row (e.g. "BACKWARD")
			if config.CompatibilityLevel == "" {
				t.Error("expected non-empty compatibility level from seeded default")
			}
		}
	})

	t.Run("SetGlobalConfig_AndGet", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.SetGlobalConfig(ctx, &storage.ConfigRecord{CompatibilityLevel: "FULL"})
		if err != nil {
			t.Fatalf("SetGlobalConfig: %v", err)
		}

		config, err := store.GetGlobalConfig(ctx)
		if err != nil {
			t.Fatalf("GetGlobalConfig: %v", err)
		}
		if config.CompatibilityLevel != "FULL" {
			t.Errorf("expected FULL, got %q", config.CompatibilityLevel)
		}
	})

	t.Run("DeleteGlobalConfig_ResetsToDefault", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		store.SetGlobalConfig(ctx, &storage.ConfigRecord{CompatibilityLevel: "NONE"})

		if err := store.DeleteGlobalConfig(ctx); err != nil {
			t.Fatalf("DeleteGlobalConfig: %v", err)
		}

		config, err := store.GetGlobalConfig(ctx)
		if err == storage.ErrNotFound {
			// Memory backend: delete removes the config entirely
		} else if err != nil {
			t.Fatalf("GetGlobalConfig after delete: unexpected error %v", err)
		} else {
			// DB backends reset to default (e.g. "BACKWARD") instead of deleting
			if config.CompatibilityLevel == "NONE" {
				t.Error("expected config to be reset after delete, but still NONE")
			}
		}
	})

	// --- Per-Subject Config ---

	t.Run("SetConfig_PerSubject", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.SetConfig(ctx, "my-subject", &storage.ConfigRecord{CompatibilityLevel: "FORWARD"})
		if err != nil {
			t.Fatalf("SetConfig: %v", err)
		}

		config, err := store.GetConfig(ctx, "my-subject")
		if err != nil {
			t.Fatalf("GetConfig: %v", err)
		}
		if config.CompatibilityLevel != "FORWARD" {
			t.Errorf("expected FORWARD, got %q", config.CompatibilityLevel)
		}
		if config.Subject != "my-subject" {
			t.Errorf("expected subject 'my-subject', got %q", config.Subject)
		}
	})

	t.Run("DeleteConfig_PerSubject", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		store.SetConfig(ctx, "my-subject", &storage.ConfigRecord{CompatibilityLevel: "NONE"})

		if err := store.DeleteConfig(ctx, "my-subject"); err != nil {
			t.Fatalf("DeleteConfig: %v", err)
		}

		_, err := store.GetConfig(ctx, "my-subject")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("GetConfig_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetConfig(ctx, "nonexistent")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("DeleteConfig_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.DeleteConfig(ctx, "nonexistent")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Config_SubjectIsolation", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		store.SetConfig(ctx, "sub-a", &storage.ConfigRecord{CompatibilityLevel: "NONE"})
		store.SetConfig(ctx, "sub-b", &storage.ConfigRecord{CompatibilityLevel: "FULL"})

		configA, _ := store.GetConfig(ctx, "sub-a")
		configB, _ := store.GetConfig(ctx, "sub-b")
		if configA.CompatibilityLevel != "NONE" {
			t.Errorf("sub-a: expected NONE, got %q", configA.CompatibilityLevel)
		}
		if configB.CompatibilityLevel != "FULL" {
			t.Errorf("sub-b: expected FULL, got %q", configB.CompatibilityLevel)
		}
	})

	// --- Global Mode ---

	t.Run("GetGlobalMode_Default", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		mode, err := store.GetGlobalMode(ctx)
		if err == storage.ErrNotFound {
			// Memory backend: no seeded default, returns ErrNotFound
		} else if err != nil {
			t.Fatalf("GetGlobalMode: unexpected error %v", err)
		} else {
			// DB backends seed a default row (e.g. "READWRITE")
			if mode.Mode == "" {
				t.Error("expected non-empty mode from seeded default")
			}
		}
	})

	t.Run("SetGlobalMode_AndGet", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.SetGlobalMode(ctx, &storage.ModeRecord{Mode: "READONLY"})
		if err != nil {
			t.Fatalf("SetGlobalMode: %v", err)
		}

		mode, err := store.GetGlobalMode(ctx)
		if err != nil {
			t.Fatalf("GetGlobalMode: %v", err)
		}
		if mode.Mode != "READONLY" {
			t.Errorf("expected READONLY, got %q", mode.Mode)
		}
	})

	// --- Per-Subject Mode ---

	t.Run("SetMode_PerSubject", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.SetMode(ctx, "my-subject", &storage.ModeRecord{Mode: "IMPORT"})
		if err != nil {
			t.Fatalf("SetMode: %v", err)
		}

		mode, err := store.GetMode(ctx, "my-subject")
		if err != nil {
			t.Fatalf("GetMode: %v", err)
		}
		if mode.Mode != "IMPORT" {
			t.Errorf("expected IMPORT, got %q", mode.Mode)
		}
		if mode.Subject != "my-subject" {
			t.Errorf("expected subject 'my-subject', got %q", mode.Subject)
		}
	})

	t.Run("DeleteMode_PerSubject", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		store.SetMode(ctx, "my-subject", &storage.ModeRecord{Mode: "IMPORT"})

		if err := store.DeleteMode(ctx, "my-subject"); err != nil {
			t.Fatalf("DeleteMode: %v", err)
		}

		_, err := store.GetMode(ctx, "my-subject")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("GetMode_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetMode(ctx, "nonexistent")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("DeleteMode_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.DeleteMode(ctx, "nonexistent")
		if err != storage.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	// --- Lifecycle ---

	t.Run("IsHealthy", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		if !store.IsHealthy(ctx) {
			t.Error("expected store to be healthy")
		}
	})

	t.Run("Close", func(t *testing.T) {
		store := newStore()
		if err := store.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
}
