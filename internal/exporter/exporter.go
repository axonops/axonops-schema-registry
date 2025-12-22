// Package exporter provides schema export functionality.
package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// ExporterType represents the type of export destination.
type ExporterType string

const (
	ExporterTypeS3       ExporterType = "s3"
	ExporterTypeGCS      ExporterType = "gcs"
	ExporterTypeAzure    ExporterType = "azure"
	ExporterTypeHTTP     ExporterType = "http"
	ExporterTypeKafka    ExporterType = "kafka"
	ExporterTypeFile     ExporterType = "file"
)

// ExporterStatus represents the status of an exporter.
type ExporterStatus string

const (
	StatusActive   ExporterStatus = "active"
	StatusPaused   ExporterStatus = "paused"
	StatusError    ExporterStatus = "error"
	StatusStarting ExporterStatus = "starting"
)

// ExportFormat represents the format for exported schemas.
type ExportFormat string

const (
	FormatJSON     ExportFormat = "json"
	FormatAvro     ExportFormat = "avro"
	FormatProtobuf ExportFormat = "protobuf"
	FormatRaw      ExportFormat = "raw"
)

// Exporter represents a schema exporter configuration.
type Exporter struct {
	Name         string            `json:"name"`
	Type         ExporterType      `json:"type"`
	Status       ExporterStatus    `json:"status"`
	Config       map[string]string `json:"config"`
	Subjects     []string          `json:"subjects,omitempty"`     // Filter by subjects
	SubjectPattern string          `json:"subject_pattern,omitempty"` // Regex pattern
	SchemaTypes  []string          `json:"schema_types,omitempty"` // Filter by type
	Format       ExportFormat      `json:"format"`
	Transform    string            `json:"transform,omitempty"` // JSONata transform
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	LastExportAt time.Time         `json:"last_export_at,omitempty"`
	ExportCount  int64             `json:"export_count"`
	ErrorCount   int64             `json:"error_count"`
	LastError    string            `json:"last_error,omitempty"`
}

// ExportEvent represents a schema export event.
type ExportEvent struct {
	Subject     string            `json:"subject"`
	Version     int               `json:"version"`
	SchemaID    int64             `json:"schema_id"`
	SchemaType  string            `json:"schema_type"`
	Schema      string            `json:"schema"`
	Fingerprint string            `json:"fingerprint"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ExportResult represents the result of an export operation.
type ExportResult struct {
	Success   bool      `json:"success"`
	ExportedAt time.Time `json:"exported_at"`
	Error     string    `json:"error,omitempty"`
	Bytes     int64     `json:"bytes,omitempty"`
}

// ExportHandler is the interface for export destinations.
type ExportHandler interface {
	Type() ExporterType
	Initialize(config map[string]string) error
	Export(ctx context.Context, event *ExportEvent) (*ExportResult, error)
	Close() error
}

// Manager manages schema exporters.
type Manager struct {
	mu        sync.RWMutex
	exporters map[string]*Exporter
	handlers  map[string]ExportHandler
	storage   storage.Storage
	running   bool
}

// NewManager creates a new exporter manager.
func NewManager(store storage.Storage) *Manager {
	return &Manager{
		exporters: make(map[string]*Exporter),
		handlers:  make(map[string]ExportHandler),
		storage:   store,
	}
}

// CreateExporter creates a new exporter.
func (m *Manager) CreateExporter(exp *Exporter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if exp.Name == "" {
		return fmt.Errorf("exporter name is required")
	}

	if _, exists := m.exporters[exp.Name]; exists {
		return fmt.Errorf("exporter already exists: %s", exp.Name)
	}

	now := time.Now()
	exp.CreatedAt = now
	exp.UpdatedAt = now
	exp.Status = StatusPaused

	if exp.Format == "" {
		exp.Format = FormatJSON
	}

	m.exporters[exp.Name] = exp
	return nil
}

// GetExporter retrieves an exporter by name.
func (m *Manager) GetExporter(name string) (*Exporter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exp, exists := m.exporters[name]
	if !exists {
		return nil, fmt.Errorf("exporter not found: %s", name)
	}
	return exp, nil
}

// UpdateExporter updates an existing exporter.
func (m *Manager) UpdateExporter(exp *Exporter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.exporters[exp.Name]
	if !exists {
		return fmt.Errorf("exporter not found: %s", exp.Name)
	}

	exp.CreatedAt = existing.CreatedAt
	exp.UpdatedAt = time.Now()
	exp.ExportCount = existing.ExportCount
	exp.ErrorCount = existing.ErrorCount

	m.exporters[exp.Name] = exp
	return nil
}

// DeleteExporter deletes an exporter.
func (m *Manager) DeleteExporter(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.exporters[name]; !exists {
		return fmt.Errorf("exporter not found: %s", name)
	}

	// Close handler if exists
	if handler, ok := m.handlers[name]; ok {
		handler.Close()
		delete(m.handlers, name)
	}

	delete(m.exporters, name)
	return nil
}

// ListExporters returns all exporters.
func (m *Manager) ListExporters() []*Exporter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Exporter, 0, len(m.exporters))
	for _, exp := range m.exporters {
		result = append(result, exp)
	}
	return result
}

// StartExporter starts an exporter.
func (m *Manager) StartExporter(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, exists := m.exporters[name]
	if !exists {
		return fmt.Errorf("exporter not found: %s", name)
	}

	exp.Status = StatusActive
	exp.UpdatedAt = time.Now()
	return nil
}

// StopExporter stops an exporter.
func (m *Manager) StopExporter(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, exists := m.exporters[name]
	if !exists {
		return fmt.Errorf("exporter not found: %s", name)
	}

	exp.Status = StatusPaused
	exp.UpdatedAt = time.Now()
	return nil
}

// Export exports a schema to all active exporters.
func (m *Manager) Export(ctx context.Context, record *storage.SchemaRecord) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	event := &ExportEvent{
		Subject:     record.Subject,
		Version:     record.Version,
		SchemaID:    record.ID,
		SchemaType:  string(record.SchemaType),
		Schema:      record.Schema,
		Fingerprint: record.Fingerprint,
		Timestamp:   time.Now(),
	}

	for name, exp := range m.exporters {
		if exp.Status != StatusActive {
			continue
		}

		// Check filters
		if !m.matchesFilters(exp, event) {
			continue
		}

		// Export
		handler, ok := m.handlers[name]
		if !ok {
			continue
		}

		result, err := handler.Export(ctx, event)
		if err != nil {
			exp.ErrorCount++
			exp.LastError = err.Error()
			exp.Status = StatusError
		} else if result.Success {
			exp.ExportCount++
			exp.LastExportAt = result.ExportedAt
		}
	}

	return nil
}

// matchesFilters checks if an event matches exporter filters.
func (m *Manager) matchesFilters(exp *Exporter, event *ExportEvent) bool {
	// Check subjects filter
	if len(exp.Subjects) > 0 {
		found := false
		for _, s := range exp.Subjects {
			if s == event.Subject {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check schema types filter
	if len(exp.SchemaTypes) > 0 {
		found := false
		for _, t := range exp.SchemaTypes {
			if t == event.SchemaType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// GetExporterStatus returns the status of all exporters.
func (m *Manager) GetExporterStatus() map[string]*ExporterStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ExporterStatus)
	for name, exp := range m.exporters {
		status := exp.Status
		result[name] = &status
	}
	return result
}

// HTTPExporter exports schemas via HTTP webhook.
type HTTPExporter struct {
	url     string
	headers map[string]string
}

// NewHTTPExporter creates a new HTTP exporter.
func NewHTTPExporter() *HTTPExporter {
	return &HTTPExporter{
		headers: make(map[string]string),
	}
}

// Type returns the exporter type.
func (e *HTTPExporter) Type() ExporterType {
	return ExporterTypeHTTP
}

// Initialize initializes the HTTP exporter.
func (e *HTTPExporter) Initialize(config map[string]string) error {
	url, ok := config["url"]
	if !ok || url == "" {
		return fmt.Errorf("url is required for HTTP exporter")
	}
	e.url = url

	for k, v := range config {
		if len(k) > 7 && k[:7] == "header." {
			e.headers[k[7:]] = v
		}
	}

	return nil
}

// Export exports a schema via HTTP.
func (e *HTTPExporter) Export(ctx context.Context, event *ExportEvent) (*ExportResult, error) {
	// In a real implementation, this would make an HTTP POST request
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	// Placeholder - would use http.Client to POST to e.url
	return &ExportResult{
		Success:    true,
		ExportedAt: time.Now(),
		Bytes:      int64(len(data)),
	}, nil
}

// Close closes the HTTP exporter.
func (e *HTTPExporter) Close() error {
	return nil
}

// FileExporter exports schemas to a file.
type FileExporter struct {
	path   string
	format ExportFormat
}

// NewFileExporter creates a new file exporter.
func NewFileExporter() *FileExporter {
	return &FileExporter{}
}

// Type returns the exporter type.
func (e *FileExporter) Type() ExporterType {
	return ExporterTypeFile
}

// Initialize initializes the file exporter.
func (e *FileExporter) Initialize(config map[string]string) error {
	path, ok := config["path"]
	if !ok || path == "" {
		return fmt.Errorf("path is required for file exporter")
	}
	e.path = path

	if format, ok := config["format"]; ok {
		e.format = ExportFormat(format)
	} else {
		e.format = FormatJSON
	}

	return nil
}

// Export exports a schema to a file.
func (e *FileExporter) Export(ctx context.Context, event *ExportEvent) (*ExportResult, error) {
	// In a real implementation, this would write to the file system
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	// Placeholder - would write to e.path
	return &ExportResult{
		Success:    true,
		ExportedAt: time.Now(),
		Bytes:      int64(len(data)),
	}, nil
}

// Close closes the file exporter.
func (e *FileExporter) Close() error {
	return nil
}
