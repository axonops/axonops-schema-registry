package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func TestNewAuditLogger_DefaultEvents(t *testing.T) {
	al, err := NewAuditLogger(config.AuditConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	// Default events should include write operations and auth failures
	expectedEnabled := []AuditEventType{
		AuditEventSchemaRegister, AuditEventSchemaDelete,
		AuditEventConfigUpdate, AuditEventModeUpdate,
		AuditEventAuthFailure, AuditEventAuthForbidden,
		AuditEventSubjectDelete,
	}
	for _, evt := range expectedEnabled {
		if !al.enabledEvents[evt] {
			t.Errorf("expected event %s to be enabled by default", evt)
		}
	}

	// Read-only events should NOT be in defaults
	if al.enabledEvents[AuditEventSchemaGet] {
		t.Error("expected schema_get not in defaults")
	}
	if al.enabledEvents[AuditEventSubjectList] {
		t.Error("expected subject_list not in defaults")
	}
}

func TestNewAuditLogger_CustomEvents(t *testing.T) {
	al, err := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		Events:  []string{"schema_register", "config_get"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	if !al.enabledEvents[AuditEventSchemaRegister] {
		t.Error("expected schema_register enabled")
	}
	if !al.enabledEvents[AuditEventConfigGet] {
		t.Error("expected config_get enabled")
	}
	if al.enabledEvents[AuditEventSchemaDelete] {
		t.Error("expected schema_delete not enabled")
	}
}

func TestNewAuditLogger_WithLogFile(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, err := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	if al.file == nil {
		t.Error("expected file to be opened")
	}

	// Log an event and verify it's written
	al.Log(&AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventSchemaRegister,
		Method:    "POST",
		Path:      "/subjects/test/versions",
	})

	al.Close()

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected log file to have content")
	}
}

func TestNewAuditLogger_InvalidLogFile(t *testing.T) {
	_, err := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: "/nonexistent/dir/audit.log",
	})
	if err == nil {
		t.Error("expected error for invalid log file path")
	}
}

func TestAuditLogger_Log_Disabled(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, err := NewAuditLogger(config.AuditConfig{
		Enabled: false,
		LogFile: logFile,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	al.Log(&AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventSchemaRegister,
	})

	// File should not have any audit entries (may not even exist)
	data, _ := os.ReadFile(logFile)
	if len(data) > 0 {
		t.Error("expected no log output when disabled")
	}
}

func TestAuditLogger_Log_FilteredEvent(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, err := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
		Events:  []string{"schema_register"}, // Only this event enabled
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	// Log a filtered event (config_get is not enabled)
	al.Log(&AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventConfigGet,
	})

	al.Close()

	data, _ := os.ReadFile(logFile)
	if len(data) > 0 {
		t.Error("expected no log output for filtered event")
	}
}

func TestAuditLogger_Close_NilFile(t *testing.T) {
	al, err := NewAuditLogger(config.AuditConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Close with no file should not error
	if err := al.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDetermineEventType_AuthFailure(t *testing.T) {
	al, _ := NewAuditLogger(config.AuditConfig{Enabled: true})
	defer al.Close()

	r := httptest.NewRequest("GET", "/subjects", nil)

	if got := al.determineEventType(r, http.StatusUnauthorized); got != AuditEventAuthFailure {
		t.Errorf("expected auth_failure, got %s", got)
	}
}

func TestDetermineEventType_AuthForbidden(t *testing.T) {
	al, _ := NewAuditLogger(config.AuditConfig{Enabled: true})
	defer al.Close()

	r := httptest.NewRequest("GET", "/subjects", nil)

	if got := al.determineEventType(r, http.StatusForbidden); got != AuditEventAuthForbidden {
		t.Errorf("expected auth_forbidden, got %s", got)
	}
}

func TestDetermineEventType_SchemaOps(t *testing.T) {
	al, _ := NewAuditLogger(config.AuditConfig{Enabled: true})
	defer al.Close()

	tests := []struct {
		method   string
		path     string
		expected AuditEventType
	}{
		{"POST", "/subjects/test/versions", AuditEventSchemaRegister},
		{"DELETE", "/subjects/test/versions/1", AuditEventSchemaDelete},
		{"GET", "/subjects/test/versions/1", AuditEventSchemaGet},
		{"GET", "/schemas/ids/1", AuditEventSchemaGet},
		{"DELETE", "/subjects/test", AuditEventSubjectDelete},
		{"GET", "/subjects", AuditEventSubjectList},
		{"GET", "/config", AuditEventConfigGet},
		{"PUT", "/config", AuditEventConfigUpdate},
		{"DELETE", "/config/test", AuditEventConfigDelete},
		{"GET", "/mode", AuditEventModeGet},
		{"PUT", "/mode", AuditEventModeUpdate},
	}

	for _, tt := range tests {
		r := httptest.NewRequest(tt.method, tt.path, nil)
		got := al.determineEventType(r, http.StatusOK)
		if got != tt.expected {
			t.Errorf("%s %s: expected %s, got %s", tt.method, tt.path, tt.expected, got)
		}
	}
}

func TestDetermineEventType_Unknown(t *testing.T) {
	al, _ := NewAuditLogger(config.AuditConfig{Enabled: true})
	defer al.Close()

	r := httptest.NewRequest("GET", "/health", nil)
	got := al.determineEventType(r, http.StatusOK)
	if got != "" {
		t.Errorf("expected empty event type for unknown path, got %s", got)
	}
}

func TestExtractSubject(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/subjects/my-topic/versions", "my-topic"},
		{"/subjects/my-topic/versions/1", "my-topic"},
		{"/subjects/my-topic", "my-topic"},
		{"/subjects/", ""},
		{"/config", ""},
		{"/schemas/ids/1", ""},
		{"/subjects/topic-with-dashes/versions/latest", "topic-with-dashes"},
	}

	for _, tt := range tests {
		got := extractSubject(tt.path)
		if got != tt.expected {
			t.Errorf("extractSubject(%q): expected %q, got %q", tt.path, tt.expected, got)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello", "world", false},
		{"/subjects/test/versions", "/subjects/", true},
		{"/config", "/subjects/", false},
		{"", "a", false},
		{"a", "", true},
	}

	for _, tt := range tests {
		if got := contains(tt.s, tt.substr); got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestAuditEvent_MarshalJSON(t *testing.T) {
	event := &AuditEvent{
		Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		EventType:  AuditEventSchemaRegister,
		User:       "admin",
		Role:       "admin",
		ClientIP:   "127.0.0.1",
		Method:     "POST",
		Path:       "/subjects/test/versions",
		StatusCode: 200,
		Duration:   150 * time.Millisecond,
		Subject:    "test",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Custom MarshalJSON converts duration to milliseconds
	if durationMs, ok := result["duration_ms"].(float64); !ok || durationMs != 150 {
		t.Errorf("expected duration_ms=150, got %v", result["duration_ms"])
	}

	if result["event_type"] != "schema_register" {
		t.Errorf("expected event_type=schema_register, got %v", result["event_type"])
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rw.statusCode)
	}
}

func TestAuditLogger_Middleware_Disabled(t *testing.T) {
	al, _ := NewAuditLogger(config.AuditConfig{Enabled: false})
	defer al.Close()

	called := false
	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/subjects", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to be called")
	}
}

func TestAuditLogger_Middleware_Enabled(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	defer al.Close()

	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("POST", "/subjects/my-topic/versions", strings.NewReader(`{"schema": "{}"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	al.Close()

	data, _ := os.ReadFile(logFile)
	if len(data) == 0 {
		t.Error("expected audit log entry")
	}
}

func TestAuditLogger_Middleware_IncludeBody(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled:     true,
		LogFile:     logFile,
		IncludeBody: true,
	})
	defer al.Close()

	body := `{"schema": "{\"type\": \"string\"}"}`
	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Body should still be readable after audit middleware reads it
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("POST", "/subjects/test/versions", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
}

func TestAuditLogger_Middleware_NoEventType(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	defer al.Close()

	// /health doesn't map to any event type
	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	al.Close()

	data, _ := os.ReadFile(logFile)
	if len(data) > 0 {
		t.Error("expected no audit entry for unknown path")
	}
}

func TestAuditLogger_Middleware_WithUser(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	defer al.Close()

	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("POST", "/subjects/test/versions", nil)
	ctx := context.WithValue(r.Context(), UserContextKey, &User{Username: "admin", Role: "admin"})
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	al.Close()

	data, _ := os.ReadFile(logFile)
	content := string(data)
	if !strings.Contains(content, "admin") {
		t.Error("expected audit log to contain user info")
	}
}

func TestAuditLogger_LogEvent(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	defer al.Close()

	r := httptest.NewRequest("POST", "/subjects/test/versions", nil)
	al.LogEvent(AuditEventSchemaRegister, r, http.StatusOK, nil)

	al.Close()

	data, _ := os.ReadFile(logFile)
	if len(data) == 0 {
		t.Error("expected log entry from LogEvent")
	}
}

func TestAuditLogger_LogEvent_Disabled(t *testing.T) {
	al, _ := NewAuditLogger(config.AuditConfig{Enabled: false})
	defer al.Close()

	r := httptest.NewRequest("POST", "/subjects/test/versions", nil)
	// Should not panic
	al.LogEvent(AuditEventSchemaRegister, r, http.StatusOK, nil)
}

func TestAuditLogger_LogEvent_WithError(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	defer al.Close()

	r := httptest.NewRequest("POST", "/subjects/test/versions", nil)
	al.LogEvent(AuditEventSchemaRegister, r, http.StatusInternalServerError, http.ErrNotSupported)

	al.Close()

	data, _ := os.ReadFile(logFile)
	if len(data) == 0 {
		t.Error("expected log entry")
	}
}
