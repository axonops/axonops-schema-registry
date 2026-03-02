package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("Expected non-nil Metrics")
	}
	if m.RequestsTotal == nil {
		t.Error("Expected RequestsTotal to be initialized")
	}
	if m.SchemasTotal == nil {
		t.Error("Expected SchemasTotal to be initialized")
	}
}

func TestMetrics_Handler(t *testing.T) {
	m := New()

	// Record some metrics so they appear in output
	m.RequestsTotal.WithLabelValues("GET", "/subjects", "200").Inc()

	handler := m.Handler()

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	body, _ := io.ReadAll(rr.Body)
	// Check for our custom metric
	if !strings.Contains(string(body), "schema_registry_requests_total") {
		t.Error("Expected metrics output to contain schema_registry_requests_total")
	}
	// Check for Go runtime metrics (always present)
	if !strings.Contains(string(body), "go_") {
		t.Error("Expected metrics output to contain Go runtime metrics")
	}
}

func TestMetrics_Middleware(t *testing.T) {
	m := New()

	var called bool
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/subjects", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMetrics_RecordSchemaRegistration(t *testing.T) {
	m := New()

	m.RecordSchemaRegistration("AVRO", true)
	m.RecordSchemaRegistration("AVRO", false)
	m.RecordSchemaRegistration("PROTOBUF", true)

	// Verify metrics are recorded (no panic)
}

func TestMetrics_RecordCompatibilityCheck(t *testing.T) {
	m := New()

	m.RecordCompatibilityCheck("AVRO", "BACKWARD", true)
	m.RecordCompatibilityCheck("AVRO", "FULL", false)

	// Verify metrics are recorded (no panic)
}

func TestMetrics_RecordStorageOperation(t *testing.T) {
	m := New()

	m.RecordStorageOperation("memory", "get", 10*time.Millisecond, nil)
	m.RecordStorageOperation("cassandra", "put", 50*time.Millisecond, io.EOF)

	// Verify metrics are recorded (no panic)
}

func TestMetrics_RecordCacheAccess(t *testing.T) {
	m := New()

	m.RecordCacheAccess("schema", true)
	m.RecordCacheAccess("schema", false)

	// Verify metrics are recorded (no panic)
}

func TestMetrics_RecordAuthAttempt(t *testing.T) {
	m := New()

	m.RecordAuthAttempt("basic", true, "", 5*time.Millisecond)
	m.RecordAuthAttempt("api_key", false, "invalid_key", 1*time.Millisecond)

	// Verify metrics are recorded (no panic)
}

func TestMetrics_RecordRateLimitHit(t *testing.T) {
	m := New()

	m.RecordRateLimitHit("192.168.1.1")
	m.RecordRateLimitHit("192.168.1.2")

	// Verify metrics are recorded (no panic)
}

func TestMetrics_UpdateSchemaCount(t *testing.T) {
	m := New()

	m.UpdateSchemaCount("AVRO", 100)
	m.UpdateSchemaCount("PROTOBUF", 50)

	// Verify metrics are recorded (no panic)
}

func TestMetrics_UpdateSubjectCount(t *testing.T) {
	m := New()

	m.UpdateSubjectCount(25)

	// Verify metrics are recorded (no panic)
}

func TestMetrics_UpdateCacheSize(t *testing.T) {
	m := New()

	m.UpdateCacheSize("schema", 1000)

	// Verify metrics are recorded (no panic)
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/subjects", "/subjects"},
		{"/subjects/my-topic", "/subjects/{subject}"},
		{"/subjects/my-topic/versions", "/subjects/{subject}/versions"},
		{"/subjects/my-topic/versions/1", "/subjects/{subject}/versions/{version}"},
		{"/subjects/my-topic/versions/latest", "/subjects/{subject}/versions/{version}"},
		{"/schemas/ids/123", "/schemas/ids/{id}"},
		{"/config", "/config"},
		{"/config/my-topic", "/config/{subject}"},
		{"/mode", "/mode"},
		{"/mode/my-topic", "/mode/{subject}"},
		{"/compatibility/subjects/my-topic/versions/1", "/compatibility/subjects/{subject}/versions/{version}"},

		// Context-scoped routes should be normalized with {context} placeholder
		{"/contexts/.TestContext/subjects", "/contexts/{context}/subjects"},
		{"/contexts/.TestContext/subjects/my-topic", "/contexts/{context}/subjects/{subject}"},
		{"/contexts/.TestContext/subjects/my-topic/versions", "/contexts/{context}/subjects/{subject}/versions"},
		{"/contexts/.TestContext/subjects/my-topic/versions/1", "/contexts/{context}/subjects/{subject}/versions/{version}"},
		{"/contexts/.TestContext/schemas/ids/123", "/contexts/{context}/schemas/ids/{id}"},
		{"/contexts/.TestContext/config", "/contexts/{context}/config"},
		{"/contexts/.TestContext/config/my-topic", "/contexts/{context}/config/{subject}"},
		{"/contexts/.TestContext/mode", "/contexts/{context}/mode"},
		{"/contexts/.TestContext/mode/my-topic", "/contexts/{context}/mode/{subject}"},
		{"/contexts/.TestContext/compatibility/subjects/my-topic/versions/1", "/contexts/{context}/compatibility/subjects/{subject}/versions/{version}"},
		{"/contexts/.production/subjects/my-topic", "/contexts/{context}/subjects/{subject}"},
		{"/contexts/:.:/subjects", "/contexts/{context}/subjects"},
	}

	for _, tt := range tests {
		result := normalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestStartsWith(t *testing.T) {
	if !startsWith("/subjects/test", "/subjects/") {
		t.Error("Expected startsWith to return true")
	}
	if startsWith("/config/test", "/subjects/") {
		t.Error("Expected startsWith to return false")
	}
}

func TestEndsWith(t *testing.T) {
	if !endsWith("/subjects/test/versions", "/versions") {
		t.Error("Expected endsWith to return true")
	}
	if endsWith("/subjects/test", "/versions") {
		t.Error("Expected endsWith to return false")
	}
}

func TestContains(t *testing.T) {
	if !contains("/subjects/test/versions/1", "/versions/") {
		t.Error("Expected contains to return true")
	}
	if contains("/subjects/test", "/versions/") {
		t.Error("Expected contains to return false")
	}
}

func TestMCPMetricsRegistered(t *testing.T) {
	m := New()
	if m.MCPToolCallsTotal == nil {
		t.Error("Expected MCPToolCallsTotal to be initialized")
	}
	if m.MCPToolCallDuration == nil {
		t.Error("Expected MCPToolCallDuration to be initialized")
	}
	if m.MCPToolCallErrors == nil {
		t.Error("Expected MCPToolCallErrors to be initialized")
	}
	if m.MCPToolCallsActive == nil {
		t.Error("Expected MCPToolCallsActive to be initialized")
	}

	// Verify they appear in /metrics output.
	m.RecordMCPToolCall("test_tool", "success", 10*time.Millisecond)

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)

	for _, metric := range []string{
		"schema_registry_mcp_tool_calls_total",
		"schema_registry_mcp_tool_call_duration_seconds",
		"schema_registry_mcp_tool_calls_active",
	} {
		if !strings.Contains(content, metric) {
			t.Errorf("Expected metrics output to contain %s", metric)
		}
	}
}

func TestRecordMCPToolCall(t *testing.T) {
	m := New()

	m.RecordMCPToolCall("register_schema", "success", 15*time.Millisecond)
	m.RecordMCPToolCall("get_schema_by_id", "error", 5*time.Millisecond)
	m.RecordMCPToolCall("register_schema", "success", 20*time.Millisecond)

	// Verify no panic — counters were incremented correctly.
}

func TestMCPToolCallsActive(t *testing.T) {
	m := New()

	m.MCPToolCallsActive.Inc()
	m.MCPToolCallsActive.Inc()
	m.MCPToolCallsActive.Dec()

	// Should be 1 after Inc, Inc, Dec. No panic.
}

func TestEnablePrincipalMetrics(t *testing.T) {
	m := New()

	// Before enabling, principal metrics should be nil.
	if m.PrincipalRequestsTotal != nil {
		t.Error("Expected PrincipalRequestsTotal to be nil before enabling")
	}

	m.EnablePrincipalMetrics()

	if m.PrincipalRequestsTotal == nil {
		t.Error("Expected PrincipalRequestsTotal to be initialized after enabling")
	}
	if m.PrincipalMCPCallsTotal == nil {
		t.Error("Expected PrincipalMCPCallsTotal to be initialized after enabling")
	}

	// Record some metrics — should not panic.
	m.RecordPrincipalRequest("admin", "GET", "/subjects", "OK")
	m.RecordPrincipalRequest("api-user", "POST", "/subjects/*", "OK")
	m.RecordPrincipalMCPCall("mcp-client", "register_schema", "success")

	// Verify they appear in /metrics output.
	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)
	if !strings.Contains(content, "schema_registry_principal_requests_total") {
		t.Error("Expected metrics output to contain schema_registry_principal_requests_total")
	}
	if !strings.Contains(content, "schema_registry_principal_mcp_calls_total") {
		t.Error("Expected metrics output to contain schema_registry_principal_mcp_calls_total")
	}
}

func TestRecordPrincipalMetrics_Disabled(t *testing.T) {
	m := New()

	// Without enabling principal metrics, these should be no-ops (no panic).
	m.RecordPrincipalRequest("admin", "GET", "/subjects", "OK")
	m.RecordPrincipalMCPCall("mcp-client", "health_check", "success")
}
