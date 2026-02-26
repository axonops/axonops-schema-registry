package registry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// CreateExporter creates a new exporter.
func (r *Registry) CreateExporter(ctx context.Context, exporter *storage.ExporterRecord) error {
	if strings.TrimSpace(exporter.Name) == "" {
		return fmt.Errorf("exporter name is required")
	}

	// Default context type
	if exporter.ContextType == "" {
		exporter.ContextType = "AUTO"
	}
	exporter.ContextType = strings.ToUpper(exporter.ContextType)
	if exporter.ContextType != "CUSTOM" && exporter.ContextType != "NONE" && exporter.ContextType != "AUTO" {
		return fmt.Errorf("invalid context type: %s (must be AUTO, CUSTOM, or NONE)", exporter.ContextType)
	}

	if err := r.storage.CreateExporter(ctx, exporter); err != nil {
		return err
	}

	// Set initial status to PAUSED
	return r.storage.SetExporterStatus(ctx, exporter.Name, &storage.ExporterStatusRecord{
		Name:  exporter.Name,
		State: "PAUSED",
		Ts:    time.Now().UnixMilli(),
	})
}

// GetExporter retrieves an exporter by name.
func (r *Registry) GetExporter(ctx context.Context, name string) (*storage.ExporterRecord, error) {
	return r.storage.GetExporter(ctx, name)
}

// UpdateExporter updates an existing exporter.
func (r *Registry) UpdateExporter(ctx context.Context, exporter *storage.ExporterRecord) error {
	if strings.TrimSpace(exporter.Name) == "" {
		return fmt.Errorf("exporter name is required")
	}

	if exporter.ContextType != "" {
		exporter.ContextType = strings.ToUpper(exporter.ContextType)
		if exporter.ContextType != "CUSTOM" && exporter.ContextType != "NONE" && exporter.ContextType != "AUTO" {
			return fmt.Errorf("invalid context type: %s (must be AUTO, CUSTOM, or NONE)", exporter.ContextType)
		}
	}

	return r.storage.UpdateExporter(ctx, exporter)
}

// DeleteExporter deletes an exporter.
func (r *Registry) DeleteExporter(ctx context.Context, name string) error {
	return r.storage.DeleteExporter(ctx, name)
}

// ListExporters returns all exporter names.
func (r *Registry) ListExporters(ctx context.Context) ([]string, error) {
	return r.storage.ListExporters(ctx)
}

// PauseExporter pauses an exporter.
func (r *Registry) PauseExporter(ctx context.Context, name string) error {
	// Verify exporter exists
	if _, err := r.storage.GetExporter(ctx, name); err != nil {
		return err
	}

	status, _ := r.storage.GetExporterStatus(ctx, name)
	if status == nil {
		status = &storage.ExporterStatusRecord{Name: name}
	}
	status.State = "PAUSED"
	status.Ts = time.Now().UnixMilli()
	return r.storage.SetExporterStatus(ctx, name, status)
}

// ResumeExporter resumes an exporter.
func (r *Registry) ResumeExporter(ctx context.Context, name string) error {
	// Verify exporter exists
	if _, err := r.storage.GetExporter(ctx, name); err != nil {
		return err
	}

	status, _ := r.storage.GetExporterStatus(ctx, name)
	if status == nil {
		status = &storage.ExporterStatusRecord{Name: name}
	}
	status.State = "RUNNING"
	status.Ts = time.Now().UnixMilli()
	return r.storage.SetExporterStatus(ctx, name, status)
}

// ResetExporter resets an exporter's offset back to zero.
func (r *Registry) ResetExporter(ctx context.Context, name string) error {
	// Verify exporter exists
	if _, err := r.storage.GetExporter(ctx, name); err != nil {
		return err
	}

	status, _ := r.storage.GetExporterStatus(ctx, name)
	if status == nil {
		status = &storage.ExporterStatusRecord{Name: name}
	}
	status.Offset = 0
	status.Trace = ""
	status.Ts = time.Now().UnixMilli()
	return r.storage.SetExporterStatus(ctx, name, status)
}

// GetExporterStatus retrieves the status of an exporter.
func (r *Registry) GetExporterStatus(ctx context.Context, name string) (*storage.ExporterStatusRecord, error) {
	return r.storage.GetExporterStatus(ctx, name)
}

// GetExporterConfig retrieves the configuration of an exporter.
func (r *Registry) GetExporterConfig(ctx context.Context, name string) (map[string]string, error) {
	return r.storage.GetExporterConfig(ctx, name)
}

// UpdateExporterConfig updates the configuration of an exporter.
func (r *Registry) UpdateExporterConfig(ctx context.Context, name string, config map[string]string) error {
	return r.storage.UpdateExporterConfig(ctx, name, config)
}
