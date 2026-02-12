package exporter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

func TestNewManager(t *testing.T) {
	store := memory.NewStore()
	mgr := NewManager(store)
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	exporters := mgr.ListExporters()
	if len(exporters) != 0 {
		t.Errorf("expected 0 exporters, got %d", len(exporters))
	}
}

func TestCreateExporter(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	exp := &Exporter{
		Name: "test-exporter",
		Type: ExporterTypeHTTP,
	}
	err := mgr.CreateExporter(exp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exp.Status != StatusPaused {
		t.Errorf("expected status paused, got %s", exp.Status)
	}
	if exp.Format != FormatJSON {
		t.Errorf("expected format json, got %s", exp.Format)
	}
	if exp.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if exp.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestCreateExporter_EmptyName(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	err := mgr.CreateExporter(&Exporter{})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateExporter_Duplicate(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	mgr.CreateExporter(&Exporter{Name: "dup"})
	err := mgr.CreateExporter(&Exporter{Name: "dup"})
	if err == nil {
		t.Error("expected error for duplicate exporter")
	}
}

func TestCreateExporter_PreservesExplicitFormat(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	exp := &Exporter{Name: "test", Format: FormatAvro}
	mgr.CreateExporter(exp)
	if exp.Format != FormatAvro {
		t.Errorf("expected format avro, got %s", exp.Format)
	}
}

func TestGetExporter(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	mgr.CreateExporter(&Exporter{Name: "test"})

	found, err := mgr.GetExporter("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Name != "test" {
		t.Errorf("expected name test, got %s", found.Name)
	}
}

func TestGetExporter_NotFound(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	_, err := mgr.GetExporter("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent exporter")
	}
}

func TestUpdateExporter(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	orig := &Exporter{Name: "test", Type: ExporterTypeHTTP}
	mgr.CreateExporter(orig)

	// Record the original timestamps
	createdAt := orig.CreatedAt

	// Wait briefly so UpdatedAt differs
	time.Sleep(time.Millisecond)

	updated := &Exporter{
		Name:     "test",
		Type:     ExporterTypeFile,
		Subjects: []string{"sub1"},
	}
	err := mgr.UpdateExporter(updated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CreatedAt should be preserved
	if updated.CreatedAt != createdAt {
		t.Error("CreatedAt should be preserved from original")
	}
	if updated.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	// Verify the update persisted
	found, _ := mgr.GetExporter("test")
	if found.Type != ExporterTypeFile {
		t.Errorf("expected type file, got %s", found.Type)
	}
}

func TestUpdateExporter_NotFound(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	err := mgr.UpdateExporter(&Exporter{Name: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent exporter")
	}
}

func TestUpdateExporter_PreservesCounters(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	orig := &Exporter{Name: "test"}
	mgr.CreateExporter(orig)
	// Simulate some counts
	orig.ExportCount = 42
	orig.ErrorCount = 3

	updated := &Exporter{Name: "test", Type: ExporterTypeFile}
	mgr.UpdateExporter(updated)

	if updated.ExportCount != 42 {
		t.Errorf("expected ExportCount 42, got %d", updated.ExportCount)
	}
	if updated.ErrorCount != 3 {
		t.Errorf("expected ErrorCount 3, got %d", updated.ErrorCount)
	}
}

func TestDeleteExporter(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	mgr.CreateExporter(&Exporter{Name: "test"})
	err := mgr.DeleteExporter("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = mgr.GetExporter("test")
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteExporter_NotFound(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	err := mgr.DeleteExporter("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent exporter")
	}
}

func TestDeleteExporter_ClosesHandler(t *testing.T) {
	mgr := NewManager(memory.NewStore())
	mgr.CreateExporter(&Exporter{Name: "test"})

	// Register a mock handler
	mock := &mockHandler{closed: false}
	mgr.handlers["test"] = mock

	mgr.DeleteExporter("test")
	if !mock.closed {
		t.Error("expected handler to be closed on delete")
	}
}

func TestListExporters(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	mgr.CreateExporter(&Exporter{Name: "a"})
	mgr.CreateExporter(&Exporter{Name: "b"})
	mgr.CreateExporter(&Exporter{Name: "c"})

	list := mgr.ListExporters()
	if len(list) != 3 {
		t.Errorf("expected 3 exporters, got %d", len(list))
	}
}

func TestListExporters_Empty(t *testing.T) {
	mgr := NewManager(memory.NewStore())
	list := mgr.ListExporters()
	if len(list) != 0 {
		t.Errorf("expected 0 exporters, got %d", len(list))
	}
}

func TestStartExporter(t *testing.T) {
	mgr := NewManager(memory.NewStore())
	mgr.CreateExporter(&Exporter{Name: "test"})

	err := mgr.StartExporter("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exp, _ := mgr.GetExporter("test")
	if exp.Status != StatusActive {
		t.Errorf("expected active status, got %s", exp.Status)
	}
}

func TestStartExporter_NotFound(t *testing.T) {
	mgr := NewManager(memory.NewStore())
	err := mgr.StartExporter("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent exporter")
	}
}

func TestStopExporter(t *testing.T) {
	mgr := NewManager(memory.NewStore())
	mgr.CreateExporter(&Exporter{Name: "test"})
	mgr.StartExporter("test")

	err := mgr.StopExporter("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exp, _ := mgr.GetExporter("test")
	if exp.Status != StatusPaused {
		t.Errorf("expected paused status, got %s", exp.Status)
	}
}

func TestStopExporter_NotFound(t *testing.T) {
	mgr := NewManager(memory.NewStore())
	err := mgr.StopExporter("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent exporter")
	}
}

func TestExport_ActiveExporterWithHandler(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	exp := &Exporter{Name: "test"}
	mgr.CreateExporter(exp)
	mgr.StartExporter("test")

	mock := &mockHandler{}
	mgr.handlers["test"] = mock

	record := &storage.SchemaRecord{
		ID:          1,
		Subject:     "test-subject",
		Version:     1,
		SchemaType:  "AVRO",
		Schema:      `{"type":"string"}`,
		Fingerprint: "abc123",
	}

	err := mgr.Export(context.Background(), record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.exported {
		t.Error("expected handler to be called")
	}

	exp, _ = mgr.GetExporter("test")
	if exp.ExportCount != 1 {
		t.Errorf("expected export count 1, got %d", exp.ExportCount)
	}
}

func TestExport_PausedExporterSkipped(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	mgr.CreateExporter(&Exporter{Name: "test"})
	// Don't start — it stays paused

	mock := &mockHandler{}
	mgr.handlers["test"] = mock

	record := &storage.SchemaRecord{Subject: "s", Schema: `{}`}
	mgr.Export(context.Background(), record)

	if mock.exported {
		t.Error("paused exporter should not export")
	}
}

func TestExport_NoHandler(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	mgr.CreateExporter(&Exporter{Name: "test"})
	mgr.StartExporter("test")
	// No handler registered

	record := &storage.SchemaRecord{Subject: "s", Schema: `{}`}
	err := mgr.Export(context.Background(), record)
	if err != nil {
		t.Fatalf("should not error even without handler: %v", err)
	}
}

func TestExport_HandlerError(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	mgr.CreateExporter(&Exporter{Name: "test"})
	mgr.StartExporter("test")

	mock := &mockHandler{err: fmt.Errorf("connection refused")}
	mgr.handlers["test"] = mock

	record := &storage.SchemaRecord{Subject: "s", Schema: `{}`}
	mgr.Export(context.Background(), record)

	exp, _ := mgr.GetExporter("test")
	if exp.ErrorCount != 1 {
		t.Errorf("expected error count 1, got %d", exp.ErrorCount)
	}
	if exp.LastError != "connection refused" {
		t.Errorf("expected last error 'connection refused', got %q", exp.LastError)
	}
	if exp.Status != StatusError {
		t.Errorf("expected error status, got %s", exp.Status)
	}
}

func TestExport_SubjectFilter(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	exp := &Exporter{Name: "test", Subjects: []string{"allowed-subject"}}
	mgr.CreateExporter(exp)
	mgr.StartExporter("test")

	mock := &mockHandler{}
	mgr.handlers["test"] = mock

	// Export with non-matching subject
	record := &storage.SchemaRecord{Subject: "other-subject", Schema: `{}`}
	mgr.Export(context.Background(), record)

	if mock.exported {
		t.Error("should not export for non-matching subject")
	}

	// Export with matching subject
	record = &storage.SchemaRecord{Subject: "allowed-subject", Schema: `{}`}
	mgr.Export(context.Background(), record)

	if !mock.exported {
		t.Error("should export for matching subject")
	}
}

func TestExport_SchemaTypeFilter(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	exp := &Exporter{Name: "test", SchemaTypes: []string{"AVRO"}}
	mgr.CreateExporter(exp)
	mgr.StartExporter("test")

	mock := &mockHandler{}
	mgr.handlers["test"] = mock

	// Non-matching type
	record := &storage.SchemaRecord{Subject: "s", Schema: `{}`, SchemaType: "JSON"}
	mgr.Export(context.Background(), record)
	if mock.exported {
		t.Error("should not export for non-matching schema type")
	}

	// Matching type
	record = &storage.SchemaRecord{Subject: "s", Schema: `{}`, SchemaType: "AVRO"}
	mgr.Export(context.Background(), record)
	if !mock.exported {
		t.Error("should export for matching schema type")
	}
}

func TestExport_NoFiltersMatchesAll(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	mgr.CreateExporter(&Exporter{Name: "test"})
	mgr.StartExporter("test")

	mock := &mockHandler{}
	mgr.handlers["test"] = mock

	record := &storage.SchemaRecord{Subject: "any", Schema: `{}`, SchemaType: "PROTOBUF"}
	mgr.Export(context.Background(), record)

	if !mock.exported {
		t.Error("exporter with no filters should match all events")
	}
}

func TestGetExporterStatus(t *testing.T) {
	mgr := NewManager(memory.NewStore())

	mgr.CreateExporter(&Exporter{Name: "a"})
	mgr.CreateExporter(&Exporter{Name: "b"})
	mgr.StartExporter("b")

	statuses := mgr.GetExporterStatus()
	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}
	if *statuses["a"] != StatusPaused {
		t.Errorf("expected a to be paused, got %s", *statuses["a"])
	}
	if *statuses["b"] != StatusActive {
		t.Errorf("expected b to be active, got %s", *statuses["b"])
	}
}

func TestGetExporterStatus_Empty(t *testing.T) {
	mgr := NewManager(memory.NewStore())
	statuses := mgr.GetExporterStatus()
	if len(statuses) != 0 {
		t.Errorf("expected 0 statuses, got %d", len(statuses))
	}
}

// --- HTTP Exporter Tests ---

func TestHTTPExporter_Type(t *testing.T) {
	e := NewHTTPExporter()
	if e.Type() != ExporterTypeHTTP {
		t.Errorf("expected http type, got %s", e.Type())
	}
}

func TestHTTPExporter_Initialize(t *testing.T) {
	e := NewHTTPExporter()
	err := e.Initialize(map[string]string{
		"url":                  "https://example.com/webhook",
		"header.Authorization": "Bearer token",
		"header.Content-Type":  "application/json",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.url != "https://example.com/webhook" {
		t.Errorf("expected url to be set")
	}
	if e.headers["Authorization"] != "Bearer token" {
		t.Errorf("expected Authorization header to be set")
	}
	if e.headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header to be set")
	}
}

func TestHTTPExporter_Initialize_MissingURL(t *testing.T) {
	e := NewHTTPExporter()
	err := e.Initialize(map[string]string{})
	if err == nil {
		t.Error("expected error for missing url")
	}
}

func TestHTTPExporter_Initialize_EmptyURL(t *testing.T) {
	e := NewHTTPExporter()
	err := e.Initialize(map[string]string{"url": ""})
	if err == nil {
		t.Error("expected error for empty url")
	}
}

func TestHTTPExporter_Export(t *testing.T) {
	e := NewHTTPExporter()
	e.Initialize(map[string]string{"url": "https://example.com"})

	event := &ExportEvent{
		Subject:  "test",
		Version:  1,
		SchemaID: 1,
		Schema:   `{"type":"string"}`,
	}

	result, err := e.Export(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Bytes <= 0 {
		t.Error("expected positive byte count")
	}
}

func TestHTTPExporter_Close(t *testing.T) {
	e := NewHTTPExporter()
	if err := e.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- File Exporter Tests ---

func TestFileExporter_Type(t *testing.T) {
	e := NewFileExporter()
	if e.Type() != ExporterTypeFile {
		t.Errorf("expected file type, got %s", e.Type())
	}
}

func TestFileExporter_Initialize(t *testing.T) {
	e := NewFileExporter()
	err := e.Initialize(map[string]string{
		"path":   "/tmp/export",
		"format": "raw",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.path != "/tmp/export" {
		t.Errorf("expected path /tmp/export, got %s", e.path)
	}
	if e.format != FormatRaw {
		t.Errorf("expected format raw, got %s", e.format)
	}
}

func TestFileExporter_Initialize_DefaultFormat(t *testing.T) {
	e := NewFileExporter()
	e.Initialize(map[string]string{"path": "/tmp/export"})
	if e.format != FormatJSON {
		t.Errorf("expected default format json, got %s", e.format)
	}
}

func TestFileExporter_Initialize_MissingPath(t *testing.T) {
	e := NewFileExporter()
	err := e.Initialize(map[string]string{})
	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestFileExporter_Export(t *testing.T) {
	e := NewFileExporter()
	e.Initialize(map[string]string{"path": "/tmp/export"})

	event := &ExportEvent{
		Subject:  "test",
		Version:  1,
		SchemaID: 1,
		Schema:   `{"type":"string"}`,
	}

	result, err := e.Export(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestFileExporter_Close(t *testing.T) {
	e := NewFileExporter()
	if err := e.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Mock Handler ---

type mockHandler struct {
	exported bool
	closed   bool
	err      error
}

func (m *mockHandler) Type() ExporterType { return "mock" }

func (m *mockHandler) Initialize(config map[string]string) error { return nil }

func (m *mockHandler) Export(ctx context.Context, event *ExportEvent) (*ExportResult, error) {
	m.exported = true
	if m.err != nil {
		return nil, m.err
	}
	return &ExportResult{
		Success:    true,
		ExportedAt: time.Now(),
		Bytes:      100,
	}, nil
}

func (m *mockHandler) Close() error {
	m.closed = true
	return nil
}
