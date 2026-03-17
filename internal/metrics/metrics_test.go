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

	// Verify gauge value is 1 after Inc, Inc, Dec.
	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)
	if !strings.Contains(content, "schema_registry_mcp_tool_calls_active 1") {
		t.Errorf("Expected mcp_tool_calls_active gauge to be 1, got output:\n%s",
			extractMetricLines(content, "schema_registry_mcp_tool_calls_active"))
	}
}

// extractMetricLines returns all lines matching a metric name for diagnostics.
func extractMetricLines(body, metric string) string {
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, metric) {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
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

	// Verify they appear in /metrics output with correct labels.
	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)

	// Verify principal HTTP request metrics with label values
	if !strings.Contains(content, `schema_registry_principal_requests_total{`) {
		t.Error("Expected metrics output to contain schema_registry_principal_requests_total")
	}
	if !strings.Contains(content, `principal="admin"`) {
		t.Error("Expected principal_requests_total to have principal=\"admin\" label")
	}
	if !strings.Contains(content, `principal="api-user"`) {
		t.Error("Expected principal_requests_total to have principal=\"api-user\" label")
	}

	// Verify principal MCP call metrics with label values
	if !strings.Contains(content, `schema_registry_principal_mcp_calls_total{`) {
		t.Error("Expected metrics output to contain schema_registry_principal_mcp_calls_total")
	}
	if !strings.Contains(content, `principal="mcp-client"`) {
		t.Errorf("Expected principal_mcp_calls_total to have principal=\"mcp-client\" label, got:\n%s",
			extractMetricLines(content, "schema_registry_principal_mcp_calls_total"))
	}
	if !strings.Contains(content, `tool="register_schema"`) {
		t.Errorf("Expected principal_mcp_calls_total to have tool=\"register_schema\" label, got:\n%s",
			extractMetricLines(content, "schema_registry_principal_mcp_calls_total"))
	}
}

func TestRecordPrincipalMetrics_Disabled(t *testing.T) {
	m := New()

	// Without enabling principal metrics, these should be no-ops (no panic).
	m.RecordPrincipalRequest("admin", "GET", "/subjects", "OK")
	m.RecordPrincipalMCPCall("mcp-client", "health_check", "success")
}

func TestRecordMCPConfirmation(t *testing.T) {
	m := New()

	m.RecordMCPConfirmation("token_issued")
	m.RecordMCPConfirmation("confirmed")
	m.RecordMCPConfirmation("token_rejected")

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)
	if !strings.Contains(content, "schema_registry_mcp_confirmations_total") {
		t.Error("Expected metrics output to contain schema_registry_mcp_confirmations_total")
	}
}

func TestRecordMCPPolicyDenial(t *testing.T) {
	m := New()

	m.RecordMCPPolicyDenial("origin_rejected")
	m.RecordMCPPolicyDenial("confirmation_required")

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)
	if !strings.Contains(content, "schema_registry_mcp_policy_denials_total") {
		t.Error("Expected metrics output to contain schema_registry_mcp_policy_denials_total")
	}
}

func TestConfluentMetricsRegistered(t *testing.T) {
	m := New()

	if m.ConfluentRegisteredCount == nil {
		t.Error("Expected ConfluentRegisteredCount to be initialized")
	}
	if m.ConfluentDeletedCount == nil {
		t.Error("Expected ConfluentDeletedCount to be initialized")
	}
	if m.ConfluentAPISuccessCount == nil {
		t.Error("Expected ConfluentAPISuccessCount to be initialized")
	}
	if m.ConfluentAPIFailureCount == nil {
		t.Error("Expected ConfluentAPIFailureCount to be initialized")
	}
	if m.ConfluentSchemasCreated == nil {
		t.Error("Expected ConfluentSchemasCreated to be initialized")
	}
	if m.ConfluentSchemasDeleted == nil {
		t.Error("Expected ConfluentSchemasDeleted to be initialized")
	}
	if m.ConfluentMasterSlaveRole == nil {
		t.Error("Expected ConfluentMasterSlaveRole to be initialized")
	}
	if m.ConfluentNodeCount == nil {
		t.Error("Expected ConfluentNodeCount to be initialized")
	}

	// Verify they appear in /metrics output
	m.RecordSchemaRegistration("AVRO", true)
	m.RecordSchemaDeletion("AVRO")

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)

	for _, metric := range []string{
		"kafka_schema_registry_registered_count",
		"kafka_schema_registry_deleted_count",
		"kafka_schema_registry_api_success_count",
		"kafka_schema_registry_api_failure_count",
		"kafka_schema_registry_schemas_created",
		"kafka_schema_registry_schemas_deleted",
		"kafka_schema_registry_master_slave_role",
		"kafka_schema_registry_node_count",
	} {
		if !strings.Contains(content, metric) {
			t.Errorf("Expected metrics output to contain %s", metric)
		}
	}
}

func TestConfluentMasterSlaveRole_Default(t *testing.T) {
	m := New()

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)

	// Should default to 1.0 (leader) for standalone
	if !strings.Contains(content, "kafka_schema_registry_master_slave_role 1") {
		t.Error("Expected master_slave_role to be 1.0 by default")
	}
	// Should default to 1 node for standalone
	if !strings.Contains(content, "kafka_schema_registry_node_count 1") {
		t.Error("Expected node_count to be 1 by default")
	}
}

func TestConfluentSchemaRegistrationCounters(t *testing.T) {
	m := New()

	// Record 3 AVRO and 2 JSON registrations
	m.RecordSchemaRegistration("AVRO", true)
	m.RecordSchemaRegistration("AVRO", true)
	m.RecordSchemaRegistration("AVRO", true)
	m.RecordSchemaRegistration("JSON", true)
	m.RecordSchemaRegistration("JSON", true)
	// Failed registration should NOT increment Confluent counters
	m.RecordSchemaRegistration("AVRO", false)

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)

	// registered_count should be 5 (3+2, not counting the failure)
	if !strings.Contains(content, "kafka_schema_registry_registered_count 5") {
		t.Errorf("Expected registered_count=5, got output:\n%s", extractMetric(content, "kafka_schema_registry_registered_count"))
	}

	// schemas_created with schema_type labels
	if !strings.Contains(content, `kafka_schema_registry_schemas_created{schema_type="avro"} 3`) {
		t.Errorf("Expected schemas_created avro=3, got output:\n%s", extractMetric(content, "kafka_schema_registry_schemas_created"))
	}
	if !strings.Contains(content, `kafka_schema_registry_schemas_created{schema_type="json"} 2`) {
		t.Errorf("Expected schemas_created json=2, got output:\n%s", extractMetric(content, "kafka_schema_registry_schemas_created"))
	}
}

func TestConfluentSchemaDeletionCounters(t *testing.T) {
	m := New()

	m.RecordSchemaDeletion("AVRO")
	m.RecordSchemaDeletion("AVRO")
	m.RecordSchemaDeletion("PROTOBUF")

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)

	if !strings.Contains(content, "kafka_schema_registry_deleted_count 3") {
		t.Errorf("Expected deleted_count=3, got output:\n%s", extractMetric(content, "kafka_schema_registry_deleted_count"))
	}
	if !strings.Contains(content, `kafka_schema_registry_schemas_deleted{schema_type="avro"} 2`) {
		t.Errorf("Expected schemas_deleted avro=2, got output:\n%s", extractMetric(content, "kafka_schema_registry_schemas_deleted"))
	}
	if !strings.Contains(content, `kafka_schema_registry_schemas_deleted{schema_type="protobuf"} 1`) {
		t.Errorf("Expected schemas_deleted protobuf=1, got output:\n%s", extractMetric(content, "kafka_schema_registry_schemas_deleted"))
	}
}

func TestConfluentAPICallCounters(t *testing.T) {
	m := New()

	// Simulate middleware recording: 2xx/3xx = success, 4xx/5xx = failure
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/created":
			w.WriteHeader(http.StatusCreated)
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))

	for _, path := range []string{"/ok", "/ok", "/created", "/notfound", "/error"} {
		req := httptest.NewRequest("GET", path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	metricsHandler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	metricsHandler.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)
	content := string(body)

	// 3 success (2x 200 + 1x 201), 2 failures (404 + 500)
	if !strings.Contains(content, "kafka_schema_registry_api_success_count 3") {
		t.Errorf("Expected api_success_count=3, got:\n%s", extractMetric(content, "kafka_schema_registry_api_success_count"))
	}
	if !strings.Contains(content, "kafka_schema_registry_api_failure_count 2") {
		t.Errorf("Expected api_failure_count=2, got:\n%s", extractMetric(content, "kafka_schema_registry_api_failure_count"))
	}
}

func TestConfluentSchemaTypeConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AVRO", "avro"},
		{"JSON", "json"},
		{"PROTOBUF", "protobuf"},
		{"", ""},
		{"avro", "avro"},
	}

	for _, tt := range tests {
		result := confluentSchemaType(tt.input)
		if result != tt.expected {
			t.Errorf("confluentSchemaType(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// extractMetric extracts all lines from metrics output containing the given metric name.
func extractMetric(content, name string) string {
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, name) {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
