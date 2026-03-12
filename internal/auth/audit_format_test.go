package auth

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFormatJSON_ValidOutput(t *testing.T) {
	event := &AuditEvent{
		Timestamp: time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC),
		Duration:  42 * time.Millisecond,
		EventType: AuditEventSchemaRegister,
		Outcome:   "success",
		ActorID:   "admin",
		Method:    "POST",
		Path:      "/subjects/test/versions",
	}

	data, err := FormatJSON(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should end with newline
	if data[len(data)-1] != '\n' {
		t.Error("expected trailing newline")
	}

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data[:len(data)-1], &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["event_type"] != "schema_register" {
		t.Errorf("event_type = %v, want schema_register", result["event_type"])
	}
}

func TestFormatCEF_Header(t *testing.T) {
	event := &AuditEvent{
		Timestamp: time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC),
		EventType: AuditEventSchemaRegister,
		Outcome:   "success",
		Method:    "POST",
		Path:      "/subjects/test/versions",
	}

	line := string(FormatCEF(event))

	// Verify CEF header prefix
	if !strings.HasPrefix(line, "CEF:0|AxonOps|SchemaRegistry|1.0|") {
		t.Errorf("unexpected CEF header prefix: %s", line[:50])
	}

	// Should contain event type
	if !strings.Contains(line, "schema_register") {
		t.Error("expected event type in CEF header")
	}

	// Should contain description
	if !strings.Contains(line, "Schema registered") {
		t.Error("expected description in CEF header")
	}

	// Should end with newline
	if line[len(line)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestFormatCEF_Severity(t *testing.T) {
	tests := []struct {
		eventType AuditEventType
		outcome   string
		expected  int
	}{
		{AuditEventAuthFailure, "failure", 8},
		{AuditEventAuthForbidden, "failure", 8},
		{AuditEventSchemaRegister, "success", 5},
		{AuditEventSchemaDelete, "success", 5},
		{AuditEventConfigUpdate, "success", 5},
		{AuditEventMCPToolCall, "success", 5},
		{AuditEventSchemaGet, "success", 3},
		{AuditEventSubjectList, "success", 3},
		{AuditEventConfigGet, "success", 3},
		// Failures for non-auth events
		{AuditEventSchemaRegister, "failure", 5},
	}

	for _, tt := range tests {
		event := &AuditEvent{EventType: tt.eventType, Outcome: tt.outcome}
		got := cefSeverity(event)
		if got != tt.expected {
			t.Errorf("cefSeverity(%s, %s) = %d, want %d", tt.eventType, tt.outcome, got, tt.expected)
		}
	}
}

func TestFormatCEF_Extensions(t *testing.T) {
	event := &AuditEvent{
		Timestamp:  time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC),
		Duration:   42 * time.Millisecond,
		EventType:  AuditEventSchemaRegister,
		Outcome:    "success",
		ActorID:    "admin",
		ActorType:  "user",
		AuthMethod: "basic",
		Role:       "admin",
		TargetType: "subject",
		TargetID:   "payments-value",
		SourceIP:   "10.0.0.1",
		UserAgent:  "curl/8.1",
		Method:     "POST",
		Path:       "/subjects/payments-value/versions",
		StatusCode: 200,
		SchemaID:   42,
		Context:    "production",
		BeforeHash: "sha256:aaa",
		AfterHash:  "sha256:bbb",
		RequestID:  "req-123",
		Reason:     "test-reason",
		Error:      "some error",
	}

	line := string(FormatCEF(event))

	// Check extension fields
	checks := []string{
		"suser=admin",
		"cs1=user",
		"cs1Label=actorType",
		"cs2=basic",
		"cs2Label=authMethod",
		"cs3=admin",
		"cs3Label=role",
		"cs4=subject",
		"cs4Label=targetType",
		"cs5=payments-value",
		"cs5Label=targetID",
		"src=10.0.0.1",
		"requestClientApplication=curl/8.1",
		"requestMethod=POST",
		"request=/subjects/payments-value/versions",
		"cn1=200",
		"cn1Label=statusCode",
		"cn2=42",
		"cn2Label=schemaID",
		"cn3=42",
		"cn3Label=durationMs",
		"cs6=production",
		"cs6Label=context",
		"oldFileHash=sha256:aaa",
		"fileHash=sha256:bbb",
		"externalId=req-123",
		"reason=test-reason",
		"msg=some error",
		"outcome=success",
	}
	for _, check := range checks {
		if !strings.Contains(line, check) {
			t.Errorf("missing extension %q in:\n%s", check, line)
		}
	}
}

func TestFormatCEF_HeaderEscaping(t *testing.T) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "test|event",
		Outcome:   "success",
		Method:    "GET",
		Path:      "/test",
	}

	line := string(FormatCEF(event))
	// Pipe in event type should be escaped
	if !strings.Contains(line, `test\|event`) {
		t.Errorf("expected escaped pipe in header: %s", line)
	}
}

func TestFormatCEF_ExtensionEscaping(t *testing.T) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: AuditEventSchemaRegister,
		Outcome:   "success",
		ActorID:   "user=admin",
		Method:    "POST",
		Path:      "/test",
	}

	line := string(FormatCEF(event))
	// Equals in actor_id should be escaped in extension value
	if !strings.Contains(line, `suser=user\=admin`) {
		t.Errorf("expected escaped equals in extension: %s", line)
	}
}

func TestCEFDescription_AllEventTypes(t *testing.T) {
	eventTypes := []AuditEventType{
		AuditEventSchemaRegister, AuditEventSchemaDelete, AuditEventSchemaGet,
		AuditEventSchemaLookup, AuditEventSchemaImport,
		AuditEventConfigGet, AuditEventConfigUpdate, AuditEventConfigDelete,
		AuditEventModeGet, AuditEventModeUpdate, AuditEventModeDelete,
		AuditEventAuthSuccess, AuditEventAuthFailure, AuditEventAuthForbidden,
		AuditEventSubjectDelete, AuditEventSubjectList,
		AuditEventUserCreate, AuditEventUserUpdate, AuditEventUserDelete,
		AuditEventPasswordChange,
		AuditEventAPIKeyCreate, AuditEventAPIKeyUpdate, AuditEventAPIKeyDelete,
		AuditEventAPIKeyRevoke, AuditEventAPIKeyRotate,
		AuditEventKEKCreate, AuditEventKEKUpdate, AuditEventKEKDelete, AuditEventKEKTest,
		AuditEventDEKCreate, AuditEventDEKDelete,
		AuditEventExporterCreate, AuditEventExporterUpdate, AuditEventExporterDelete,
		AuditEventExporterPause, AuditEventExporterResume, AuditEventExporterReset,
		AuditEventMCPToolCall, AuditEventMCPToolError, AuditEventMCPAdminAction,
		AuditEventMCPConfirmIssued, AuditEventMCPConfirmRejected, AuditEventMCPConfirmed,
	}

	for _, et := range eventTypes {
		event := &AuditEvent{EventType: et}
		desc := cefDescription(event)
		if desc == "" {
			t.Errorf("cefDescription(%s) returned empty string", et)
		}
		// Description should NOT be the raw event type (should be human-readable)
		if desc == string(et) {
			t.Errorf("cefDescription(%s) returned raw event type, expected human-readable", et)
		}
	}
}

func TestCEFEscapeHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"pipe|here", `pipe\|here`},
		{`back\slash`, `back\\slash`},
		{`both\|chars`, `both\\\|chars`},
		{"", ""},
	}
	for _, tt := range tests {
		got := cefEscapeHeader(tt.input)
		if got != tt.expected {
			t.Errorf("cefEscapeHeader(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCEFEscapeExtValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"key=value", `key\=value`},
		{`back\slash`, `back\\slash`},
		{"line\nbreak", `line\nbreak`},
		{"return\rchar", `return\rchar`},
		{"", ""},
	}
	for _, tt := range tests {
		got := cefEscapeExtValue(tt.input)
		if got != tt.expected {
			t.Errorf("cefEscapeExtValue(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
