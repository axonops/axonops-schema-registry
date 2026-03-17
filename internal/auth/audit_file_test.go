package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func TestNewFileOutput_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	fo, err := NewFileOutput(config.AuditFileConfig{
		Enabled: true,
		Path:    path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer fo.Close()

	// Verify defaults applied
	if fo.logger.MaxSize != 100 {
		t.Errorf("MaxSize = %d, want 100", fo.logger.MaxSize)
	}
	if fo.logger.MaxBackups != 5 {
		t.Errorf("MaxBackups = %d, want 5", fo.logger.MaxBackups)
	}
	if fo.logger.MaxAge != 30 {
		t.Errorf("MaxAge = %d, want 30", fo.logger.MaxAge)
	}
	if !fo.logger.Compress {
		t.Error("Compress should default to true")
	}
}

func TestNewFileOutput_CustomConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	compress := false
	fo, err := NewFileOutput(config.AuditFileConfig{
		Enabled:    true,
		Path:       path,
		MaxSizeMB:  50,
		MaxBackups: 10,
		MaxAgeDays: 7,
		Compress:   &compress,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer fo.Close()

	if fo.logger.MaxSize != 50 {
		t.Errorf("MaxSize = %d, want 50", fo.logger.MaxSize)
	}
	if fo.logger.MaxBackups != 10 {
		t.Errorf("MaxBackups = %d, want 10", fo.logger.MaxBackups)
	}
	if fo.logger.MaxAge != 7 {
		t.Errorf("MaxAge = %d, want 7", fo.logger.MaxAge)
	}
	if fo.logger.Compress {
		t.Error("Compress should be false")
	}
}

func TestNewFileOutput_EmptyPath(t *testing.T) {
	_, err := NewFileOutput(config.AuditFileConfig{Enabled: true})
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestFileOutput_Name(t *testing.T) {
	dir := t.TempDir()
	fo, err := NewFileOutput(config.AuditFileConfig{
		Path: filepath.Join(dir, "test.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fo.Close()

	if fo.Name() != "file" {
		t.Errorf("expected name file, got %s", fo.Name())
	}
}

func TestFileOutput_WriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	fo, err := NewFileOutput(config.AuditFileConfig{
		Path: path,
	})
	if err != nil {
		t.Fatal(err)
	}

	event := map[string]string{
		"event_type": "schema_register",
		"actor_id":   "admin",
	}
	data, _ := json.Marshal(event)
	data = append(data, '\n')

	if err := fo.Write(data); err != nil {
		t.Fatalf("write error: %v", err)
	}
	fo.Close()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if !strings.Contains(string(content), "schema_register") {
		t.Error("expected schema_register in file content")
	}
}

func TestFileOutput_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	fo, err := NewFileOutput(config.AuditFileConfig{
		Path: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fo.Close()

	const goroutines = 20
	const eventsPerGoroutine = 50

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range eventsPerGoroutine {
				event := &AuditEvent{
					Timestamp: time.Now(),
					EventType: AuditEventSchemaRegister,
					Outcome:   "success",
					ActorID:   "worker",
					Method:    "POST",
					Path:      "/subjects/test/versions",
					Metadata:  map[string]string{"goroutine": string(rune('A' + id)), "event": string(rune('0' + j%10))},
				}
				data, _ := json.Marshal(event)
				data = append(data, '\n')
				_ = fo.Write(data)
			}
		}(i)
	}
	wg.Wait()
	fo.Close()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	// Count lines — each should be valid JSON
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	expected := goroutines * eventsPerGoroutine
	if len(lines) != expected {
		t.Errorf("expected %d lines, got %d", expected, len(lines))
	}

	for i, line := range lines {
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("line %d: invalid JSON: %v", i, err)
		}
	}
}

func TestFileOutput_RotationOnSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	// Use tiny max size to trigger rotation
	fo, err := NewFileOutput(config.AuditFileConfig{
		Path:       path,
		MaxSizeMB:  1, // 1 MB — lumberjack minimum
		MaxBackups: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Write enough data to exceed 1MB
	event := &AuditEvent{
		Timestamp:   time.Now(),
		EventType:   AuditEventSchemaRegister,
		Outcome:     "success",
		Method:      "POST",
		Path:        "/subjects/test/versions",
		RequestBody: strings.Repeat("x", 1000),
	}
	data, _ := json.Marshal(event)
	data = append(data, '\n')

	// Write ~2MB of data
	for range 2000 {
		_ = fo.Write(data)
	}
	fo.Close()

	// Check that backup files were created
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Should have at least 2 files (current + at least one backup)
	if len(entries) < 2 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected at least 2 files after rotation, got %d: %v", len(entries), names)
	}
}
