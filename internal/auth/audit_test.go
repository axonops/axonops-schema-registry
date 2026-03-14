package auth

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5/middleware"

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
		AuditEventSchemaRegister,
		AuditEventSchemaDeleteSoft, AuditEventSchemaDeletePermanent,
		AuditEventConfigUpdate, AuditEventModeUpdate,
		AuditEventAuthFailure, AuditEventAuthForbidden,
		AuditEventSubjectDeleteSoft, AuditEventSubjectDeletePermanent,
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
	if al.enabledEvents[AuditEventSchemaDeleteSoft] {
		t.Error("expected schema_delete_soft not enabled")
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

	if len(al.outputs) == 0 {
		t.Error("expected at least one output to be configured")
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
		// Schema operations
		{"POST", "/subjects/test/versions", AuditEventSchemaRegister},
		{"DELETE", "/subjects/test/versions/1", AuditEventSchemaDeleteSoft},
		{"DELETE", "/subjects/test/versions/1?permanent=true", AuditEventSchemaDeletePermanent},
		{"GET", "/subjects/test/versions/1", AuditEventSchemaGet},
		{"GET", "/schemas/ids/1", AuditEventSchemaGet},
		{"POST", "/subjects/test", AuditEventSchemaLookup},
		{"DELETE", "/subjects/test", AuditEventSubjectDeleteSoft},
		{"DELETE", "/subjects/test?permanent=true", AuditEventSubjectDeletePermanent},
		{"GET", "/subjects", AuditEventSubjectList},
		// Import
		{"POST", "/import/schemas", AuditEventSchemaImport},
		// Compatibility check
		{"POST", "/compatibility/subjects/test/versions/1", AuditEventCompatibilityCheck},
		{"POST", "/compatibility/subjects/test/versions", AuditEventCompatibilityCheck},
		// Config operations
		{"GET", "/config", AuditEventConfigGet},
		{"PUT", "/config", AuditEventConfigUpdate},
		{"DELETE", "/config/test", AuditEventConfigDelete},
		// Mode operations (including DELETE)
		{"GET", "/mode", AuditEventModeGet},
		{"PUT", "/mode", AuditEventModeUpdate},
		{"DELETE", "/mode/test", AuditEventModeDelete},
		{"DELETE", "/mode", AuditEventModeDelete},
		// Admin — users
		{"POST", "/admin/users", AuditEventUserCreate},
		{"PUT", "/admin/users/1", AuditEventUserUpdate},
		{"DELETE", "/admin/users/1", AuditEventUserDelete},
		// Admin — API keys
		{"POST", "/admin/apikeys", AuditEventAPIKeyCreate},
		{"PUT", "/admin/apikeys/1", AuditEventAPIKeyUpdate},
		{"DELETE", "/admin/apikeys/1", AuditEventAPIKeyDelete},
		{"POST", "/admin/apikeys/1/revoke", AuditEventAPIKeyRevoke},
		{"POST", "/admin/apikeys/1/rotate", AuditEventAPIKeyRotate},
		// Account self-service
		{"POST", "/me/password", AuditEventPasswordChange},
		// KEK operations
		{"POST", "/dek-registry/v1/keks", AuditEventKEKCreate},
		{"PUT", "/dek-registry/v1/keks/my-kek", AuditEventKEKUpdate},
		{"DELETE", "/dek-registry/v1/keks/my-kek", AuditEventKEKDeleteSoft},
		{"DELETE", "/dek-registry/v1/keks/my-kek?permanent=true", AuditEventKEKDeletePermanent},
		{"POST", "/dek-registry/v1/keks/my-kek/undelete", AuditEventKEKUndelete},
		{"POST", "/dek-registry/v1/keks/my-kek/test", AuditEventKEKTest},
		// DEK operations
		{"POST", "/dek-registry/v1/keks/my-kek/deks", AuditEventDEKCreate},
		{"POST", "/dek-registry/v1/keks/my-kek/deks/my-subject", AuditEventDEKCreate},
		{"DELETE", "/dek-registry/v1/keks/my-kek/deks/my-subject", AuditEventDEKDeleteSoft},
		{"DELETE", "/dek-registry/v1/keks/my-kek/deks/my-subject?permanent=true", AuditEventDEKDeletePermanent},
		{"POST", "/dek-registry/v1/keks/my-kek/deks/my-subject/undelete", AuditEventDEKUndelete},
		// DEK version operations
		{"DELETE", "/dek-registry/v1/keks/my-kek/deks/my-subject/versions/1", AuditEventDEKDeleteSoft},
		{"DELETE", "/dek-registry/v1/keks/my-kek/deks/my-subject/versions/1?permanent=true", AuditEventDEKDeletePermanent},
		{"POST", "/dek-registry/v1/keks/my-kek/deks/my-subject/versions/1/undelete", AuditEventDEKUndelete},
		// Exporter operations
		{"POST", "/exporters", AuditEventExporterCreate},
		{"PUT", "/exporters/my-export", AuditEventExporterUpdate},
		{"DELETE", "/exporters/my-export", AuditEventExporterDelete},
		{"PUT", "/exporters/my-export/pause", AuditEventExporterPause},
		{"PUT", "/exporters/my-export/resume", AuditEventExporterResume},
		{"PUT", "/exporters/my-export/reset", AuditEventExporterReset},
		{"PUT", "/exporters/my-export/config", AuditEventExporterConfigUpdate},
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
		{"/config/my-subject", "my-subject"},
		{"/config/subject-with-dashes", "subject-with-dashes"},
		{"/mode", ""},
		{"/mode/my-subject", "my-subject"},
		{"/mode/subject-with-dashes", "subject-with-dashes"},
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
		Outcome:    "success",
		ActorID:    "admin",
		ActorType:  "user",
		Role:       "admin",
		AuthMethod: "basic",
		TargetType: "subject",
		TargetID:   "test",
		SourceIP:   "127.0.0.1",
		Method:     "POST",
		Path:       "/subjects/test/versions",
		StatusCode: 200,
		Duration:   150 * time.Millisecond,
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

func TestLogMCPEvent(t *testing.T) {
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

	al.LogMCPEvent(AuditEventMCPToolCall, "mcp-authenticated", "mcp_client", "bearer_token", "register_schema", "success", 15*time.Millisecond, nil, "test", map[string]string{"subject": "test"})

	al.Close()

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "register_schema") {
		t.Error("expected log to contain tool name")
	}
	if !strings.Contains(content, "MCP") {
		t.Error("expected log to contain MCP method")
	}
}

func TestLogMCPEvent_Error(t *testing.T) {
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

	al.LogMCPEvent(AuditEventMCPToolError, "mcp-authenticated", "mcp_client", "bearer_token", "get_schema_by_id", "error", 5*time.Millisecond, http.ErrNotSupported, "", nil)

	al.Close()

	data, _ := os.ReadFile(logFile)
	content := string(data)
	if !strings.Contains(content, "get_schema_by_id") {
		t.Error("expected log to contain tool name")
	}
}

func TestLogMCPEvent_Disabled(t *testing.T) {
	al, err := NewAuditLogger(config.AuditConfig{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	// Should not panic.
	al.LogMCPEvent(AuditEventMCPToolCall, "mcp-anonymous", "anonymous", "", "health_check", "success", time.Millisecond, nil, "", nil)
}

func TestMCPEventTypeFiltering(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	// Only enable mcp_tool_error, not mcp_tool_call
	al, err := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
		Events:  []string{"mcp_tool_error"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	// This should be filtered out
	al.LogMCPEvent(AuditEventMCPToolCall, "mcp-anonymous", "anonymous", "", "health_check", "success", time.Millisecond, nil, "", nil)
	// This should be logged
	al.LogMCPEvent(AuditEventMCPToolError, "mcp-anonymous", "anonymous", "", "get_schema_by_id", "error", time.Millisecond, http.ErrNotSupported, "", nil)

	al.Close()

	data, _ := os.ReadFile(logFile)
	content := string(data)
	if strings.Contains(content, "health_check") {
		t.Error("expected mcp_tool_call to be filtered out")
	}
	if !strings.Contains(content, "get_schema_by_id") {
		t.Error("expected mcp_tool_error to be logged")
	}
}

func TestAuditLogger_Log_EmitsRequestBody(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	defer al.Close()

	al.Log(&AuditEvent{
		Timestamp:   time.Now(),
		EventType:   AuditEventSchemaRegister,
		Method:      "POST",
		Path:        "/subjects/test/versions",
		RequestBody: `{"schema":"{\"type\":\"string\"}"}`,
	})

	al.Close()
	data, _ := os.ReadFile(logFile)
	content := string(data)
	if !strings.Contains(content, "request_body") {
		t.Error("expected audit log to contain request_body field")
	}
	if !strings.Contains(content, `schema`) {
		t.Error("expected audit log to contain the request body content")
	}
}

func TestAuditLogger_Log_EmitsMetadata(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	defer al.Close()

	al.Log(&AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventSchemaRegister,
		Method:    "POST",
		Path:      "/subjects/test/versions",
		Metadata:  map[string]string{"tool": "register_schema", "context": "default"},
	})

	al.Close()
	data, _ := os.ReadFile(logFile)
	content := string(data)
	if !strings.Contains(content, "metadata") {
		t.Error("expected audit log to contain metadata group")
	}
	if !strings.Contains(content, "register_schema") {
		t.Error("expected audit log to contain metadata value")
	}
}

func TestAuditLogger_Log_EmitsRequestID(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	defer al.Close()

	al.Log(&AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventSchemaRegister,
		Method:    "POST",
		Path:      "/subjects/test/versions",
		RequestID: "req-abc-123",
	})

	al.Close()
	data, _ := os.ReadFile(logFile)
	content := string(data)
	if !strings.Contains(content, "request_id") {
		t.Error("expected audit log to contain request_id field")
	}
	if !strings.Contains(content, "req-abc-123") {
		t.Error("expected audit log to contain the request ID value")
	}
}

func TestAuditLogger_Middleware_IncludesRequestID(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "audit.log")

	al, _ := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		LogFile: logFile,
	})
	defer al.Close()

	// Chain chi's RequestID middleware → audit middleware → handler
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := middleware.RequestID(al.Middleware(inner))

	r := httptest.NewRequest("POST", "/subjects/test/versions", nil)
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)

	al.Close()
	data, _ := os.ReadFile(logFile)
	content := string(data)
	if !strings.Contains(content, "request_id") {
		t.Error("expected audit log to contain request_id from chi RequestID middleware")
	}
}

func TestNewAuditLogger_DefaultIncludesConfirmationEvents(t *testing.T) {
	al, err := NewAuditLogger(config.AuditConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	confirmEvents := []AuditEventType{
		AuditEventMCPConfirmIssued,
		AuditEventMCPConfirmRejected,
		AuditEventMCPConfirmed,
	}
	for _, evt := range confirmEvents {
		if !al.enabledEvents[evt] {
			t.Errorf("expected confirmation event %s to be enabled by default", evt)
		}
	}
}

func TestLogMCPConfirmationEvent(t *testing.T) {
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

	al.LogMCPConfirmationEvent(AuditEventMCPConfirmIssued, "mcp-authenticated", "mcp_client", "bearer_token", "delete_subject", nil)

	al.Close()

	data, _ := os.ReadFile(logFile)
	content := string(data)
	if !strings.Contains(content, "mcp_confirm_issued") {
		t.Error("expected log to contain mcp_confirm_issued event type")
	}
	if !strings.Contains(content, "delete_subject") {
		t.Error("expected log to contain tool name")
	}
}

func TestLogMCPEvent_WithSubject(t *testing.T) {
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

	al.LogMCPEvent(AuditEventMCPToolCall, "mcp-authenticated", "mcp_client", "bearer_token", "register_schema", "success", 10*time.Millisecond, nil, "payments-value", nil)

	al.Close()

	data, _ := os.ReadFile(logFile)
	content := string(data)
	if !strings.Contains(content, "payments-value") {
		t.Error("expected log to contain subject name")
	}
}

func TestNewAuditLoggerWithWriter(t *testing.T) {
	var buf bytes.Buffer

	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	al.Log(&AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventSchemaRegister,
		Method:    "POST",
		Path:      "/subjects/test/versions",
		TargetID:  "test",
	})

	content := buf.String()
	if !strings.Contains(content, "schema_register") {
		t.Error("expected buffer to contain schema_register event")
	}
	if !strings.Contains(content, "test") {
		t.Error("expected buffer to contain subject name")
	}
}

func TestNewAuditLogger_DefaultIncludesMCPEvents(t *testing.T) {
	al, err := NewAuditLogger(config.AuditConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	mcpEvents := []AuditEventType{
		AuditEventMCPToolCall,
		AuditEventMCPToolError,
		AuditEventMCPAdminAction,
	}
	for _, evt := range mcpEvents {
		if !al.enabledEvents[evt] {
			t.Errorf("expected MCP event %s to be enabled by default", evt)
		}
	}
}

// --- Tests for new helper functions ---

func TestActorTypeFromAuthMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected string
	}{
		{"basic", "user"},
		{"jwt", "user"},
		{"oidc", "user"},
		{"ldap", "user"},
		{"ldap_fallback", "user"},
		{"api_key", "api_key"},
		{"", "anonymous"},
		{"unknown", "anonymous"},
	}
	for _, tt := range tests {
		got := actorTypeFromAuthMethod(tt.method)
		if got != tt.expected {
			t.Errorf("actorTypeFromAuthMethod(%q) = %q, want %q", tt.method, got, tt.expected)
		}
	}
}

func TestTransportSecurityFromRequest(t *testing.T) {
	tests := []struct {
		name     string
		setupTLS func(r *http.Request)
		expected string
	}{
		{"no TLS", func(r *http.Request) {}, "none"},
		{"TLS without client cert", func(r *http.Request) {
			r.TLS = &tls.ConnectionState{}
		}, "tls"},
		{"mTLS with client cert", func(r *http.Request) {
			r.TLS = &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{{}},
			}
		}, "mtls"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			tt.setupTLS(req)
			got := transportSecurityFromRequest(req)
			if got != tt.expected {
				t.Errorf("transportSecurityFromRequest() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestOutcomeFromStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "success"},
		{201, "success"},
		{204, "success"},
		{301, "success"},
		{399, "success"},
		{400, "failure"},
		{401, "failure"},
		{403, "failure"},
		{404, "failure"},
		{409, "failure"},
		{422, "failure"},
		{429, "failure"},
		{500, "failure"},
		{502, "failure"},
	}
	for _, tt := range tests {
		got := outcomeFromStatusCode(tt.code)
		if got != tt.expected {
			t.Errorf("outcomeFromStatusCode(%d) = %q, want %q", tt.code, got, tt.expected)
		}
	}
}

func TestReasonFromStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, ""},
		{201, ""},
		{204, ""},
		{401, "no_valid_credentials"},
		{403, "permission_denied"},
		{404, "not_found"},
		{409, "already_exists"},
		{400, "validation_error"},
		{422, "invalid_schema"},
		{429, "rate_limited"},
		{500, "internal_error"},
		{502, "internal_error"},
		{418, ""}, // I'm a teapot — unknown 4xx
	}
	for _, tt := range tests {
		got := reasonFromStatusCode(tt.code)
		if got != tt.expected {
			t.Errorf("reasonFromStatusCode(%d) = %q, want %q", tt.code, got, tt.expected)
		}
	}
}

func TestClassifyMCPError(t *testing.T) {
	tests := []struct {
		errMsg   string
		expected string
	}{
		{"subject 'foo' not found", "not_found"},
		{"Not Found", "not_found"},
		{"permission denied", "permission_denied"},
		{"forbidden access", "permission_denied"},
		{"unauthorized request", "permission_denied"},
		{"subject already exists", "already_exists"},
		{"duplicate entry", "already_exists"},
		{"invalid schema: parse error", "invalid_schema"},
		{"parse failure", "invalid_schema"},
		{"schema is incompatible", "incompatible"},
		{"invalid argument: missing field", "validation_error"},
		{"required field missing", "validation_error"},
		{"missing parameter", "validation_error"},
		{"something went terribly wrong", "internal_error"},
		{"", "internal_error"},
	}
	for _, tt := range tests {
		got := classifyMCPError(tt.errMsg)
		if got != tt.expected {
			t.Errorf("classifyMCPError(%q) = %q, want %q", tt.errMsg, got, tt.expected)
		}
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"ABC", "abc"},
		{"already lower", "already lower"},
		{"MiXeD123", "mixed123"},
		{"", ""},
	}
	for _, tt := range tests {
		got := toLower(tt.input)
		if got != tt.expected {
			t.Errorf("toLower(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtractTarget(t *testing.T) {
	tests := []struct {
		path      string
		eventType AuditEventType
		wantType  string
		wantID    string
	}{
		// Subject operations
		{"/subjects/payments-value/versions", AuditEventSchemaRegister, "subject", "payments-value"},
		{"/subjects/my-topic/versions/1", AuditEventSchemaGet, "subject", "my-topic"},
		{"/subjects/my-topic", AuditEventSubjectDeleteSoft, "subject", "my-topic"},
		// Schema by ID
		{"/schemas/ids/42", AuditEventSchemaGet, "schema", "42"},
		{"/schemas/ids/100/schema", AuditEventSchemaGet, "schema", "100"},
		// Config operations
		{"/config", AuditEventConfigGet, "config", "_global"},
		{"/config/my-subject", AuditEventConfigUpdate, "config", "my-subject"},
		// Mode operations
		{"/mode", AuditEventModeGet, "mode", "_global"},
		{"/mode/my-subject", AuditEventModeUpdate, "mode", "my-subject"},
		// KEK operations
		{"/dek-registry/v1/keks", AuditEventKEKCreate, "kek", ""},
		{"/dek-registry/v1/keks/my-kek", AuditEventKEKUpdate, "kek", "my-kek"},
		// DEK operations
		{"/dek-registry/v1/keks/my-kek/deks/my-subject", AuditEventDEKCreate, "dek", "my-kek"},
		// Exporter operations
		{"/exporters", AuditEventExporterCreate, "exporter", ""},
		{"/exporters/my-export", AuditEventExporterUpdate, "exporter", "my-export"},
		{"/exporters/my-export/pause", AuditEventExporterPause, "exporter", "my-export"},
		// Admin users
		{"/admin/users", AuditEventUserCreate, "user", ""},
		{"/admin/users/42", AuditEventUserUpdate, "user", "42"},
		// Admin API keys
		{"/admin/apikeys", AuditEventAPIKeyCreate, "apikey", ""},
		{"/admin/apikeys/99", AuditEventAPIKeyDelete, "apikey", "99"},
		{"/admin/apikeys/1/revoke", AuditEventAPIKeyRevoke, "apikey", "1"},
		// Import
		{"/import/schemas", AuditEventSchemaImport, "schema", ""},
		// Unknown
		{"/health", "", "", ""},
	}
	for _, tt := range tests {
		gotType, gotID := extractTarget(tt.path, tt.eventType)
		if gotType != tt.wantType || gotID != tt.wantID {
			t.Errorf("extractTarget(%q, %q) = (%q, %q), want (%q, %q)",
				tt.path, tt.eventType, gotType, gotID, tt.wantType, tt.wantID)
		}
	}
}

func TestExtractKEKDEKTarget(t *testing.T) {
	tests := []struct {
		path     string
		wantType string
		wantID   string
	}{
		{"/dek-registry/v1/keks/my-kek", "kek", "my-kek"},
		{"/dek-registry/v1/keks/my-kek/deks/my-subject", "dek", "my-kek"},
		{"/dek-registry/v1/keks/", "kek", ""},
		{"/dek-registry/v1/keks", "kek", ""},
	}
	for _, tt := range tests {
		gotType, gotID := extractKEKDEKTarget(tt.path)
		if gotType != tt.wantType || gotID != tt.wantID {
			t.Errorf("extractKEKDEKTarget(%q) = (%q, %q), want (%q, %q)",
				tt.path, gotType, gotID, tt.wantType, tt.wantID)
		}
	}
}

func TestExtractExporterTarget(t *testing.T) {
	tests := []struct {
		path     string
		wantType string
		wantID   string
	}{
		{"/exporters/my-export", "exporter", "my-export"},
		{"/exporters/my-export/pause", "exporter", "my-export"},
		{"/exporters", "exporter", ""},
		{"/exporters/", "exporter", ""},
	}
	for _, tt := range tests {
		gotType, gotID := extractExporterTarget(tt.path)
		if gotType != tt.wantType || gotID != tt.wantID {
			t.Errorf("extractExporterTarget(%q) = (%q, %q), want (%q, %q)",
				tt.path, gotType, gotID, tt.wantType, tt.wantID)
		}
	}
}

func TestExtractAdminTarget(t *testing.T) {
	tests := []struct {
		path       string
		prefix     string
		targetType string
		wantType   string
		wantID     string
	}{
		{"/admin/users/42", "/admin/users/", "user", "user", "42"},
		{"/admin/users", "/admin/users/", "user", "user", ""},
		{"/admin/apikeys/99", "/admin/apikeys/", "apikey", "apikey", "99"},
		{"/admin/apikeys/1/revoke", "/admin/apikeys/", "apikey", "apikey", "1"},
	}
	for _, tt := range tests {
		gotType, gotID := extractAdminTarget(tt.path, tt.prefix, tt.targetType)
		if gotType != tt.wantType || gotID != tt.wantID {
			t.Errorf("extractAdminTarget(%q, %q, %q) = (%q, %q), want (%q, %q)",
				tt.path, tt.prefix, tt.targetType, gotType, gotID, tt.wantType, tt.wantID)
		}
	}
}

// --- Tests verifying new fields appear in JSON and slog output ---

func TestAuditEvent_MarshalJSON_AllNewFields(t *testing.T) {
	event := &AuditEvent{
		Timestamp:  time.Date(2026, 3, 11, 14, 30, 0, 0, time.UTC),
		Duration:   42 * time.Millisecond,
		EventType:  AuditEventSchemaRegister,
		Outcome:    "success",
		ActorID:    "jane",
		ActorType:  "user",
		Role:       "developer",
		AuthMethod: "basic",
		TargetType: "subject",
		TargetID:   "payments-value",
		SchemaID:   42,
		Version:    3,
		SchemaType: "AVRO",
		BeforeHash: "sha256:8b7f1234",
		AfterHash:  "sha256:2d910abc",
		Context:    "production",
		RequestID:  "req-abc-123",
		SourceIP:   "172.18.0.1",
		UserAgent:  "curl/8.1",
		Method:     "POST",
		Path:       "/subjects/payments-value/versions",
		StatusCode: 200,
		Reason:     "",
		Metadata:   map[string]string{"custom": "value"},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify all new fields are present
	checks := map[string]interface{}{
		"outcome":     "success",
		"actor_id":    "jane",
		"actor_type":  "user",
		"role":        "developer",
		"auth_method": "basic",
		"target_type": "subject",
		"target_id":   "payments-value",
		"schema_id":   float64(42),
		"version":     float64(3),
		"schema_type": "AVRO",
		"before_hash": "sha256:8b7f1234",
		"after_hash":  "sha256:2d910abc",
		"context":     "production",
		"request_id":  "req-abc-123",
		"source_ip":   "172.18.0.1",
		"user_agent":  "curl/8.1",
		"duration_ms": float64(42),
	}
	for field, expected := range checks {
		if result[field] != expected {
			t.Errorf("field %q: got %v, want %v", field, result[field], expected)
		}
	}

	// Verify metadata
	meta, ok := result["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("expected metadata to be a map")
	}
	if meta["custom"] != "value" {
		t.Errorf("metadata[custom]: got %v, want 'value'", meta["custom"])
	}
}

func TestAuditEvent_MarshalJSON_CoreFieldsAlwaysPresent(t *testing.T) {
	event := &AuditEvent{
		Timestamp: time.Date(2026, 3, 11, 14, 30, 0, 0, time.UTC),
		EventType: AuditEventSchemaGet,
		Outcome:   "success",
		Method:    "GET",
		Path:      "/schemas/ids/1",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Core fields MUST always be present, even when zero-valued.
	coreFields := []string{
		"timestamp", "event_type", "outcome", "method", "path",
		"actor_id", "actor_type", "target_type", "target_id",
		"source_ip", "user_agent", "status_code",
	}
	for _, field := range coreFields {
		if _, ok := result[field]; !ok {
			t.Errorf("core field %q must always be present, but it was omitted", field)
		}
	}

	// Contextual fields should be omitted when empty/zero.
	contextualFields := []string{
		"role", "auth_method", "schema_type",
		"before_hash", "after_hash", "context",
		"reason", "error", "request_body", "request_id",
	}
	for _, field := range contextualFields {
		if _, ok := result[field]; ok {
			t.Errorf("contextual field %q should be omitted when empty, but it was present", field)
		}
	}
}

func TestAuditLogger_Log_EmitsNewFields(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	al.Log(&AuditEvent{
		Timestamp:  time.Now(),
		EventType:  AuditEventSchemaRegister,
		Outcome:    "success",
		ActorID:    "testuser",
		ActorType:  "user",
		AuthMethod: "basic",
		TargetType: "subject",
		TargetID:   "my-topic",
		SourceIP:   "10.0.0.1",
		UserAgent:  "test-agent/1.0",
		Method:     "POST",
		Path:       "/subjects/my-topic/versions",
		Reason:     "",
	})

	content := buf.String()
	for _, field := range []string{"outcome", "actor_id", "actor_type", "auth_method", "target_type", "target_id", "source_ip", "user_agent"} {
		if !strings.Contains(content, field) {
			t.Errorf("expected slog output to contain %q", field)
		}
	}
	for _, value := range []string{"testuser", "user", "basic", "subject", "my-topic", "10.0.0.1", "test-agent/1.0"} {
		if !strings.Contains(content, value) {
			t.Errorf("expected slog output to contain value %q", value)
		}
	}
}

func TestAuditLogger_Middleware_PopulatesNewFields(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("POST", "/subjects/test/versions", nil)
	r.Header.Set("User-Agent", "test-client/2.0")
	r.RemoteAddr = "192.168.1.100:12345"
	ctx := context.WithValue(r.Context(), UserContextKey, &User{
		Username: "admin",
		Role:     "admin",
		Method:   "basic",
	})
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	content := buf.String()
	// Verify actor fields
	if !strings.Contains(content, "admin") {
		t.Error("expected actor_id=admin in output")
	}
	if !strings.Contains(content, "user") {
		t.Error("expected actor_type=user in output")
	}
	if !strings.Contains(content, "basic") {
		t.Error("expected auth_method=basic in output")
	}
	// Verify outcome
	if !strings.Contains(content, "success") {
		t.Error("expected outcome=success in output")
	}
	// Verify target
	if !strings.Contains(content, "subject") {
		t.Error("expected target_type=subject in output")
	}
	if !strings.Contains(content, "test") {
		t.Error("expected target_id=test in output")
	}
	// Verify transport
	if !strings.Contains(content, "test-client/2.0") {
		t.Error("expected user_agent in output")
	}
	if !strings.Contains(content, "192.168.1.100") {
		t.Error("expected source_ip in output")
	}
}

func TestAuditLogger_Middleware_FailureOutcome(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	// Use DELETE which maps to schema_delete (enabled by default), not GET which maps to schema_get (not enabled)
	r := httptest.NewRequest("DELETE", "/subjects/missing/versions/1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	content := buf.String()
	if !strings.Contains(content, "failure") {
		t.Error("expected outcome=failure for 404 status")
	}
	if !strings.Contains(content, "not_found") {
		t.Error("expected reason=not_found for 404 status")
	}
}

func TestAuditLogger_Middleware_AnonymousActor(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// No user in context
	r := httptest.NewRequest("POST", "/subjects/test/versions", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	content := buf.String()
	if !strings.Contains(content, "anonymous") {
		t.Error("expected actor_type=anonymous when no user in context")
	}
}

func TestAuditLogger_Middleware_APIKeyActor(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("POST", "/subjects/test/versions", nil)
	ctx := context.WithValue(r.Context(), UserContextKey, &User{
		Username: "ci-pipeline",
		Role:     "developer",
		Method:   "api_key",
	})
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	content := buf.String()
	if !strings.Contains(content, "api_key") {
		t.Error("expected actor_type=api_key and auth_method=api_key for API key user")
	}
	if !strings.Contains(content, "ci-pipeline") {
		t.Error("expected actor_id=ci-pipeline in output")
	}
	if !strings.Contains(content, "developer") {
		t.Error("expected role=developer in output")
	}
}

func TestLogMCPEvent_PopulatesOutcomeAndReason(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	// Success case
	al.LogMCPEvent(AuditEventMCPToolCall, "mcp-authenticated", "mcp_client", "bearer_token", "health_check", "success", time.Millisecond, nil, "", nil)
	content := buf.String()
	if !strings.Contains(content, `"outcome":"success"`) && !strings.Contains(content, "success") {
		t.Error("expected outcome=success for successful MCP call")
	}

	// Error case with classified reason
	buf.Reset()
	al.LogMCPEvent(AuditEventMCPToolError, "mcp-authenticated", "mcp_client", "bearer_token", "get_schema_by_id", "error", time.Millisecond,
		http.ErrNotSupported, "", nil)
	content = buf.String()
	if !strings.Contains(content, "failure") {
		t.Error("expected outcome=failure for MCP error")
	}
}

func TestLogMCPEvent_ActorFields(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	al.LogMCPEvent(AuditEventMCPToolCall, "mcp-authenticated", "mcp_client", "bearer_token", "register_schema", "success", time.Millisecond, nil, "my-topic", nil)
	content := buf.String()
	if !strings.Contains(content, "mcp-authenticated") {
		t.Error("expected actor_id=mcp-authenticated")
	}
	if !strings.Contains(content, "mcp_client") {
		t.Error("expected actor_type=mcp_client")
	}
	if !strings.Contains(content, "bearer_token") {
		t.Error("expected auth_method=bearer_token")
	}
}

func TestAuditHints_ContextRoundTrip(t *testing.T) {
	// Verify that AuditHints can be stored and retrieved from context.
	hints := &AuditHints{
		BeforeHash: "sha256:aaa",
		AfterHash:  "sha256:bbb",
		SchemaType: "AVRO",
		SchemaID:   42,
		Version:    3,
		Context:    "production",
	}

	ctx := context.WithValue(context.Background(), auditHintsKey{}, hints)
	got := GetAuditHints(ctx)
	if got == nil {
		t.Fatal("expected AuditHints from context, got nil")
	}
	if got.BeforeHash != "sha256:aaa" {
		t.Errorf("BeforeHash = %q, want sha256:aaa", got.BeforeHash)
	}
	if got.AfterHash != "sha256:bbb" {
		t.Errorf("AfterHash = %q, want sha256:bbb", got.AfterHash)
	}
	if got.SchemaType != "AVRO" {
		t.Errorf("SchemaType = %q, want AVRO", got.SchemaType)
	}
	if got.SchemaID != 42 {
		t.Errorf("SchemaID = %d, want 42", got.SchemaID)
	}
	if got.Version != 3 {
		t.Errorf("Version = %d, want 3", got.Version)
	}
	if got.Context != "production" {
		t.Errorf("Context = %q, want production", got.Context)
	}
}

func TestAuditHints_NilWhenAbsent(t *testing.T) {
	got := GetAuditHints(context.Background())
	if got != nil {
		t.Error("expected nil AuditHints from context without hints")
	}
}

func TestAuditLogger_Middleware_PropagatesHints(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler sets audit hints (simulating what RegisterSchema handler does)
		if hints := GetAuditHints(r.Context()); hints != nil {
			hints.BeforeHash = "sha256:before123"
			hints.AfterHash = "sha256:after456"
			hints.SchemaType = "PROTOBUF"
			hints.SchemaID = 99
			hints.Version = 5
			hints.Context = "staging"
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("POST", "/subjects/test/versions", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	content := buf.String()
	if !strings.Contains(content, "sha256:before123") {
		t.Error("expected before_hash in audit output")
	}
	if !strings.Contains(content, "sha256:after456") {
		t.Error("expected after_hash in audit output")
	}
	if !strings.Contains(content, "PROTOBUF") {
		t.Error("expected schema_type in audit output")
	}
	if !strings.Contains(content, "staging") {
		t.Error("expected context in audit output")
	}
}

func TestAuditLogger_Middleware_OutcomeOverride(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler overrides the outcome even though status is 200.
		if hints := GetAuditHints(r.Context()); hints != nil {
			hints.Outcome = "partial_failure"
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("POST", "/import/schemas", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	content := buf.String()
	if !strings.Contains(content, `"outcome":"partial_failure"`) {
		t.Errorf("expected outcome=partial_failure in audit output, got: %s", content)
	}
}

func TestAuditLogger_Middleware_OutcomeDefaultFromStatus(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	handler := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler does NOT set outcome — middleware should derive from 422.
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))

	r := httptest.NewRequest("POST", "/import/schemas", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	content := buf.String()
	if !strings.Contains(content, `"outcome":"failure"`) {
		t.Errorf("expected outcome=failure from 422 status, got: %s", content)
	}
}

func TestLogMCPConfirmationEvent_ActorFields(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	al.LogMCPConfirmationEvent(AuditEventMCPConfirmIssued, "mcp-authenticated", "mcp_client", "bearer_token", "delete_subject", nil)
	content := buf.String()
	if !strings.Contains(content, "mcp-authenticated") {
		t.Error("expected actor_id=mcp-authenticated")
	}
	if !strings.Contains(content, "mcp_client") {
		t.Error("expected actor_type=mcp_client")
	}
	if !strings.Contains(content, "bearer_token") {
		t.Error("expected auth_method=bearer_token")
	}
	if !strings.Contains(content, "success") {
		t.Error("expected outcome=success for confirmation events")
	}
}

// --- AuditOutput interface tests ---

func TestStdoutOutput_Name(t *testing.T) {
	o := &StdoutOutput{}
	if o.Name() != "stdout" {
		t.Errorf("expected name stdout, got %s", o.Name())
	}
	if err := o.Close(); err != nil {
		t.Errorf("unexpected close error: %v", err)
	}
}

func TestWriterOutput_WriteAndClose(t *testing.T) {
	var buf bytes.Buffer
	o := &WriterOutput{w: &buf, name: "test"}
	if o.Name() != "test" {
		t.Errorf("expected name test, got %s", o.Name())
	}

	if err := o.Write([]byte("hello")); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if buf.String() != "hello" {
		t.Errorf("expected hello, got %s", buf.String())
	}

	// Close with no closer should succeed
	if err := o.Close(); err != nil {
		t.Errorf("unexpected close error: %v", err)
	}
}

func TestWriterOutput_CloseWithCloser(t *testing.T) {
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "audit-test-*.log")
	if err != nil {
		t.Fatal(err)
	}
	o := &WriterOutput{w: f, closer: f, name: "file"}
	if err := o.Write([]byte("test\n")); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := o.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	// Second close should fail (file already closed)
	if err := o.Close(); err == nil {
		t.Error("expected error on second close")
	}
}

func TestAuditLogger_MultipleOutputs(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	al := &AuditLogger{
		config:        config.AuditConfig{Enabled: true},
		enabledEvents: make(map[AuditEventType]bool),
		outputs: []formattedOutput{
			{output: &WriterOutput{w: &buf1, name: "buf1"}, formatType: "json"},
			{output: &WriterOutput{w: &buf2, name: "buf2"}, formatType: "json"},
		},
	}
	setDefaultEnabledEvents(al.enabledEvents)
	defer al.Close()

	al.Log(&AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventSchemaRegister,
		Outcome:   "success",
		Method:    "POST",
		Path:      "/subjects/test/versions",
	})

	for i, buf := range []*bytes.Buffer{&buf1, &buf2} {
		content := buf.String()
		if !strings.Contains(content, "schema_register") {
			t.Errorf("output %d: expected schema_register in output", i)
		}
		// Verify the output is valid JSON
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &event); err != nil {
			t.Errorf("output %d: invalid JSON: %v", i, err)
		}
	}
}

type failingOutput struct {
	name string
}

func (o *failingOutput) Write([]byte) error { return os.ErrPermission }
func (o *failingOutput) Close() error       { return nil }
func (o *failingOutput) Name() string       { return o.name }

func TestAuditLogger_FailingOutputDoesNotBlockOthers(t *testing.T) {
	var buf bytes.Buffer
	al := &AuditLogger{
		config:        config.AuditConfig{Enabled: true},
		enabledEvents: make(map[AuditEventType]bool),
		outputs: []formattedOutput{
			{output: &failingOutput{name: "failing"}, formatType: "json"},
			{output: &WriterOutput{w: &buf, name: "good"}, formatType: "json"},
		},
	}
	setDefaultEnabledEvents(al.enabledEvents)
	defer al.Close()

	al.Log(&AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventSchemaRegister,
		Outcome:   "success",
		Method:    "POST",
		Path:      "/subjects/test/versions",
	})

	if !strings.Contains(buf.String(), "schema_register") {
		t.Error("expected second output to receive event despite first output failing")
	}
}

func TestAuditLogger_Log_ProducesValidJSON(t *testing.T) {
	var buf bytes.Buffer
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &buf)
	defer al.Close()

	al.Log(&AuditEvent{
		Timestamp:  time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC),
		Duration:   42 * time.Millisecond,
		EventType:  AuditEventSchemaRegister,
		Outcome:    "success",
		ActorID:    "admin",
		ActorType:  "user",
		AuthMethod: "basic",
		TargetType: "subject",
		TargetID:   "test-topic",
		Method:     "POST",
		Path:       "/subjects/test-topic/versions",
		StatusCode: 200,
		Metadata:   map[string]string{"key": "value"},
	})

	var event map[string]interface{}
	line := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, line)
	}

	// Verify key fields
	if event["event_type"] != "schema_register" {
		t.Errorf("event_type = %v, want schema_register", event["event_type"])
	}
	if event["outcome"] != "success" {
		t.Errorf("outcome = %v, want success", event["outcome"])
	}
	if event["actor_id"] != "admin" {
		t.Errorf("actor_id = %v, want admin", event["actor_id"])
	}
	if event["duration_ms"] != float64(42) {
		t.Errorf("duration_ms = %v, want 42", event["duration_ms"])
	}
	meta, ok := event["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("expected metadata to be a map")
	}
	if meta["key"] != "value" {
		t.Errorf("metadata[key] = %v, want value", meta["key"])
	}
}

// --- Async audit logger tests ---

// mockAuditMetrics records audit metric calls for assertions.
type mockAuditMetrics struct {
	bufferDrops int
	mu          sync.Mutex
}

func (m *mockAuditMetrics) RecordAuditEvent(_, _ string)                   {}
func (m *mockAuditMetrics) RecordAuditOutputError(_ string)                {}
func (m *mockAuditMetrics) RecordAuditWebhookDrop()                        {}
func (m *mockAuditMetrics) RecordAuditWebhookFlush(_ int, _ time.Duration) {}
func (m *mockAuditMetrics) RecordAuditBufferDrop() {
	m.mu.Lock()
	m.bufferDrops++
	m.mu.Unlock()
}

func (m *mockAuditMetrics) getBufferDrops() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bufferDrops
}

// slowOutput blocks on Write for a configurable duration.
type slowOutput struct {
	delay  time.Duration
	events [][]byte
	mu     sync.Mutex
}

func (o *slowOutput) Write(data []byte) error {
	time.Sleep(o.delay)
	o.mu.Lock()
	o.events = append(o.events, append([]byte(nil), data...))
	o.mu.Unlock()
	return nil
}
func (o *slowOutput) Close() error { return nil }
func (o *slowOutput) Name() string { return "slow" }
func (o *slowOutput) eventCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.events)
}

func TestAuditLogger_AsyncNonBlocking(t *testing.T) {
	// Create an async logger with a slow output to verify Log() returns immediately.
	slow := &slowOutput{delay: 200 * time.Millisecond}
	al := &AuditLogger{
		config:        config.AuditConfig{Enabled: true},
		outputs:       []formattedOutput{{output: slow, formatType: "json"}},
		enabledEvents: make(map[AuditEventType]bool),
		ch:            make(chan *AuditEvent, 100),
		stopCh:        make(chan struct{}),
	}
	setDefaultEnabledEvents(al.enabledEvents)
	al.wg.Add(1)
	go al.drainLoop()

	start := time.Now()
	for i := range 10 {
		al.Log(&AuditEvent{
			Timestamp: time.Now(),
			EventType: AuditEventSchemaRegister,
			Method:    "POST",
			Path:      "/subjects/test/versions",
			Version:   i,
		})
	}
	elapsed := time.Since(start)

	// Log() should return nearly instantly — well under the slow output's delay.
	if elapsed > 50*time.Millisecond {
		t.Errorf("Log() took %v, expected non-blocking (<50ms)", elapsed)
	}

	al.Close()

	// After close, all 10 events should have been drained.
	if got := slow.eventCount(); got != 10 {
		t.Errorf("slow output received %d events, want 10", got)
	}
}

func TestAuditLogger_AsyncDrain(t *testing.T) {
	// Verify all events are delivered after Close().
	var buf bytes.Buffer
	al, err := NewAuditLogger(config.AuditConfig{
		Enabled: true,
		Outputs: config.AuditOutputsConfig{
			File: config.AuditFileConfig{
				Enabled: true,
				Path:    filepath.Join(t.TempDir(), "audit.log"),
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Replace the output with a writer for easy counting.
	al.outputs = []formattedOutput{{
		output:     &WriterOutput{w: &buf, name: "test"},
		formatType: "json",
	}}

	for i := range 50 {
		al.Log(&AuditEvent{
			Timestamp: time.Now(),
			EventType: AuditEventSchemaRegister,
			Method:    "POST",
			Path:      "/subjects/test/versions",
			Version:   i,
		})
	}

	al.Close()

	// Count lines in buffer — each event is one JSON line.
	lines := 0
	for _, b := range buf.Bytes() {
		if b == '\n' {
			lines++
		}
	}
	if lines != 50 {
		t.Errorf("got %d lines, want 50", lines)
	}
}

func TestAuditLogger_AsyncDropOnFull(t *testing.T) {
	// Create async logger with tiny buffer, fill it, verify drops + metric.
	block := make(chan struct{})
	blocking := &slowOutput{delay: 0}
	// Override Write to block until we release.
	origWrite := blocking.Write
	_ = origWrite
	blockingOutput := &blockingAuditOutput{block: block}

	metrics := &mockAuditMetrics{}
	al := &AuditLogger{
		config:        config.AuditConfig{Enabled: true},
		outputs:       []formattedOutput{{output: blockingOutput, formatType: "json"}},
		metrics:       metrics,
		enabledEvents: make(map[AuditEventType]bool),
		ch:            make(chan *AuditEvent, 2),
		stopCh:        make(chan struct{}),
	}
	setDefaultEnabledEvents(al.enabledEvents)
	al.wg.Add(1)
	go al.drainLoop()

	// The drain goroutine will pick up the first event and block on Write.
	// The next 2 events fill the channel buffer. The 4th+ should be dropped.
	for range 10 {
		al.Log(&AuditEvent{
			Timestamp: time.Now(),
			EventType: AuditEventSchemaRegister,
			Method:    "POST",
			Path:      "/subjects/test/versions",
		})
	}

	// We expect at least some drops (buffer is 2, output is blocked).
	drops := metrics.getBufferDrops()
	if drops == 0 {
		t.Error("expected at least one buffer drop, got 0")
	}

	// Unblock and close.
	close(block)
	al.Close()
}

// blockingAuditOutput blocks on Write until its block channel is closed.
type blockingAuditOutput struct {
	block chan struct{}
}

func (o *blockingAuditOutput) Write(_ []byte) error {
	<-o.block
	return nil
}
func (o *blockingAuditOutput) Close() error { return nil }
func (o *blockingAuditOutput) Name() string { return "blocking" }

func TestAuditLogger_SyncModeViaWriter(t *testing.T) {
	// NewAuditLoggerWithWriter should use sync mode (ch == nil).
	al := NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, &bytes.Buffer{})
	if al.ch != nil {
		t.Error("expected ch to be nil for NewAuditLoggerWithWriter (sync mode)")
	}
	if al.stopCh != nil {
		t.Error("expected stopCh to be nil for NewAuditLoggerWithWriter (sync mode)")
	}
	al.Close()
}

func TestAuditLogger_BufferSizeConfig(t *testing.T) {
	// Verify custom BufferSize sets channel capacity.
	al, err := NewAuditLogger(config.AuditConfig{
		Enabled:    true,
		BufferSize: 42,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	if al.ch == nil {
		t.Fatal("expected ch to be non-nil for NewAuditLogger")
	}
	if cap(al.ch) != 42 {
		t.Errorf("channel capacity = %d, want 42", cap(al.ch))
	}
}

func TestAuditLogger_DefaultBufferSize(t *testing.T) {
	// Verify default buffer size is 10000 when BufferSize is 0.
	al, err := NewAuditLogger(config.AuditConfig{
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer al.Close()

	if al.ch == nil {
		t.Fatal("expected ch to be non-nil for NewAuditLogger")
	}
	if cap(al.ch) != 10000 {
		t.Errorf("channel capacity = %d, want 10000", cap(al.ch))
	}
}
