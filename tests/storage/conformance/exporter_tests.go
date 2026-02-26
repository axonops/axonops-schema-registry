package conformance

import (
	"context"
	"errors"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// RunExporterTests tests all exporter CRUD operations.
func RunExporterTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	t.Run("CreateExporter_Basic", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exporter := &storage.ExporterRecord{
			Name:                "test-exporter",
			ContextType:         "CUSTOM",
			Context:             ".my-context",
			Subjects:            []string{"subject-a", "subject-b"},
			SubjectRenameFormat: "${subject}",
			Config:              map[string]string{"schema.registry.url": "http://dest:8081"},
		}
		if err := store.CreateExporter(ctx, exporter); err != nil {
			t.Fatalf("CreateExporter: %v", err)
		}

		got, err := store.GetExporter(ctx, "test-exporter")
		if err != nil {
			t.Fatalf("GetExporter: %v", err)
		}
		if got.Name != "test-exporter" {
			t.Errorf("expected name 'test-exporter', got %q", got.Name)
		}
		if got.ContextType != "CUSTOM" {
			t.Errorf("expected contextType 'CUSTOM', got %q", got.ContextType)
		}
		if got.Context != ".my-context" {
			t.Errorf("expected context '.my-context', got %q", got.Context)
		}
		if len(got.Subjects) != 2 {
			t.Fatalf("expected 2 subjects, got %d", len(got.Subjects))
		}
		if got.Subjects[0] != "subject-a" || got.Subjects[1] != "subject-b" {
			t.Errorf("expected subjects [subject-a, subject-b], got %v", got.Subjects)
		}
		if got.SubjectRenameFormat != "${subject}" {
			t.Errorf("expected subjectRenameFormat '${subject}', got %q", got.SubjectRenameFormat)
		}
		if got.Config["schema.registry.url"] != "http://dest:8081" {
			t.Errorf("expected config url, got %q", got.Config["schema.registry.url"])
		}
	})

	t.Run("CreateExporter_Duplicate", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exporter := &storage.ExporterRecord{
			Name: "dup-exporter",
		}
		if err := store.CreateExporter(ctx, exporter); err != nil {
			t.Fatalf("CreateExporter: %v", err)
		}

		dup := &storage.ExporterRecord{
			Name: "dup-exporter",
		}
		err := store.CreateExporter(ctx, dup)
		if !errors.Is(err, storage.ErrExporterExists) {
			t.Errorf("expected ErrExporterExists, got %v", err)
		}
	})

	t.Run("GetExporter_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		_, err := store.GetExporter(ctx, "nonexistent-exporter")
		if !errors.Is(err, storage.ErrExporterNotFound) {
			t.Errorf("expected ErrExporterNotFound, got %v", err)
		}
	})

	t.Run("UpdateExporter", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exporter := &storage.ExporterRecord{
			Name:        "upd-exporter",
			ContextType: "AUTO",
			Subjects:    []string{"subject-a"},
			Config:      map[string]string{"key": "value1"},
		}
		if err := store.CreateExporter(ctx, exporter); err != nil {
			t.Fatalf("CreateExporter: %v", err)
		}

		updated := &storage.ExporterRecord{
			Name:                "upd-exporter",
			ContextType:         "CUSTOM",
			Context:             ".updated-context",
			Subjects:            []string{"subject-a", "subject-b", "subject-c"},
			SubjectRenameFormat: "prefix-${subject}",
			Config:              map[string]string{"key": "value2", "extra": "data"},
		}
		if err := store.UpdateExporter(ctx, updated); err != nil {
			t.Fatalf("UpdateExporter: %v", err)
		}

		got, err := store.GetExporter(ctx, "upd-exporter")
		if err != nil {
			t.Fatalf("GetExporter: %v", err)
		}
		if got.ContextType != "CUSTOM" {
			t.Errorf("expected contextType 'CUSTOM', got %q", got.ContextType)
		}
		if got.Context != ".updated-context" {
			t.Errorf("expected context '.updated-context', got %q", got.Context)
		}
		if len(got.Subjects) != 3 {
			t.Errorf("expected 3 subjects, got %d", len(got.Subjects))
		}
		if got.SubjectRenameFormat != "prefix-${subject}" {
			t.Errorf("expected subjectRenameFormat 'prefix-${subject}', got %q", got.SubjectRenameFormat)
		}
		if got.Config["key"] != "value2" {
			t.Errorf("expected config key 'value2', got %q", got.Config["key"])
		}
		if got.Config["extra"] != "data" {
			t.Errorf("expected config extra 'data', got %q", got.Config["extra"])
		}
	})

	t.Run("UpdateExporter_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exporter := &storage.ExporterRecord{
			Name: "nonexistent-exporter",
		}
		err := store.UpdateExporter(ctx, exporter)
		if !errors.Is(err, storage.ErrExporterNotFound) {
			t.Errorf("expected ErrExporterNotFound, got %v", err)
		}
	})

	t.Run("DeleteExporter", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exporter := &storage.ExporterRecord{
			Name: "del-exporter",
		}
		if err := store.CreateExporter(ctx, exporter); err != nil {
			t.Fatalf("CreateExporter: %v", err)
		}

		if err := store.DeleteExporter(ctx, "del-exporter"); err != nil {
			t.Fatalf("DeleteExporter: %v", err)
		}

		// Should not be found after deletion
		_, err := store.GetExporter(ctx, "del-exporter")
		if !errors.Is(err, storage.ErrExporterNotFound) {
			t.Errorf("expected ErrExporterNotFound after delete, got %v", err)
		}
	})

	t.Run("DeleteExporter_NotFound", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		err := store.DeleteExporter(ctx, "nonexistent-exporter")
		if !errors.Is(err, storage.ErrExporterNotFound) {
			t.Errorf("expected ErrExporterNotFound, got %v", err)
		}
	})

	t.Run("ListExporters", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		names := []string{"exporter-charlie", "exporter-alice", "exporter-bob"}
		for _, name := range names {
			exporter := &storage.ExporterRecord{
				Name: name,
			}
			if err := store.CreateExporter(ctx, exporter); err != nil {
				t.Fatalf("CreateExporter(%s): %v", name, err)
			}
		}

		got, err := store.ListExporters(ctx)
		if err != nil {
			t.Fatalf("ListExporters: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 exporters, got %d", len(got))
		}
		// Verify sorted by name
		if got[0] != "exporter-alice" {
			t.Errorf("expected first exporter 'exporter-alice', got %q", got[0])
		}
		if got[1] != "exporter-bob" {
			t.Errorf("expected second exporter 'exporter-bob', got %q", got[1])
		}
		if got[2] != "exporter-charlie" {
			t.Errorf("expected third exporter 'exporter-charlie', got %q", got[2])
		}
	})

	t.Run("GetExporterStatus_Default", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exporter := &storage.ExporterRecord{
			Name: "status-exporter",
		}
		if err := store.CreateExporter(ctx, exporter); err != nil {
			t.Fatalf("CreateExporter: %v", err)
		}

		status, err := store.GetExporterStatus(ctx, "status-exporter")
		if err != nil {
			t.Fatalf("GetExporterStatus: %v", err)
		}
		if status.Name != "status-exporter" {
			t.Errorf("expected name 'status-exporter', got %q", status.Name)
		}
		if status.State != "PAUSED" {
			t.Errorf("expected default state 'PAUSED', got %q", status.State)
		}
	})

	t.Run("SetExporterStatus", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exporter := &storage.ExporterRecord{
			Name: "running-exporter",
		}
		if err := store.CreateExporter(ctx, exporter); err != nil {
			t.Fatalf("CreateExporter: %v", err)
		}

		status := &storage.ExporterStatusRecord{
			Name:   "running-exporter",
			State:  "RUNNING",
			Offset: 42,
			Ts:     1000,
		}
		if err := store.SetExporterStatus(ctx, "running-exporter", status); err != nil {
			t.Fatalf("SetExporterStatus: %v", err)
		}

		got, err := store.GetExporterStatus(ctx, "running-exporter")
		if err != nil {
			t.Fatalf("GetExporterStatus: %v", err)
		}
		if got.State != "RUNNING" {
			t.Errorf("expected state 'RUNNING', got %q", got.State)
		}
		if got.Offset != 42 {
			t.Errorf("expected offset 42, got %d", got.Offset)
		}
	})

	t.Run("GetExporterConfig", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exporter := &storage.ExporterRecord{
			Name:   "config-exporter",
			Config: map[string]string{"schema.registry.url": "http://dest:8081", "timeout": "30"},
		}
		if err := store.CreateExporter(ctx, exporter); err != nil {
			t.Fatalf("CreateExporter: %v", err)
		}

		config, err := store.GetExporterConfig(ctx, "config-exporter")
		if err != nil {
			t.Fatalf("GetExporterConfig: %v", err)
		}
		if config["schema.registry.url"] != "http://dest:8081" {
			t.Errorf("expected url 'http://dest:8081', got %q", config["schema.registry.url"])
		}
		if config["timeout"] != "30" {
			t.Errorf("expected timeout '30', got %q", config["timeout"])
		}
	})

	t.Run("UpdateExporterConfig", func(t *testing.T) {
		store := newStore()
		defer store.Close()
		ctx := context.Background()

		exporter := &storage.ExporterRecord{
			Name:   "upd-config-exporter",
			Config: map[string]string{"key1": "value1"},
		}
		if err := store.CreateExporter(ctx, exporter); err != nil {
			t.Fatalf("CreateExporter: %v", err)
		}

		newConfig := map[string]string{"key1": "updated", "key2": "new-value"}
		if err := store.UpdateExporterConfig(ctx, "upd-config-exporter", newConfig); err != nil {
			t.Fatalf("UpdateExporterConfig: %v", err)
		}

		config, err := store.GetExporterConfig(ctx, "upd-config-exporter")
		if err != nil {
			t.Fatalf("GetExporterConfig: %v", err)
		}
		if config["key1"] != "updated" {
			t.Errorf("expected key1 'updated', got %q", config["key1"])
		}
		if config["key2"] != "new-value" {
			t.Errorf("expected key2 'new-value', got %q", config["key2"])
		}
	})
}
