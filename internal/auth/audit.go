// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// AuditEventType represents the type of audit event.
type AuditEventType string

const (
	// Schema events
	AuditEventSchemaRegister AuditEventType = "schema_register"
	AuditEventSchemaDelete   AuditEventType = "schema_delete"
	AuditEventSchemaGet      AuditEventType = "schema_get"
	AuditEventSchemaLookup   AuditEventType = "schema_lookup"
	AuditEventSchemaImport   AuditEventType = "schema_import"

	// Config events
	AuditEventConfigGet    AuditEventType = "config_get"
	AuditEventConfigUpdate AuditEventType = "config_update"
	AuditEventConfigDelete AuditEventType = "config_delete"

	// Mode events
	AuditEventModeGet    AuditEventType = "mode_get"
	AuditEventModeUpdate AuditEventType = "mode_update"
	AuditEventModeDelete AuditEventType = "mode_delete"

	// Auth events
	AuditEventAuthSuccess   AuditEventType = "auth_success"
	AuditEventAuthFailure   AuditEventType = "auth_failure"
	AuditEventAuthForbidden AuditEventType = "auth_forbidden"

	// Subject events
	AuditEventSubjectDelete AuditEventType = "subject_delete"
	AuditEventSubjectList   AuditEventType = "subject_list"

	// Admin events
	AuditEventUserCreate   AuditEventType = "user_create"
	AuditEventUserUpdate   AuditEventType = "user_update"
	AuditEventUserDelete   AuditEventType = "user_delete"
	AuditEventAPIKeyCreate AuditEventType = "apikey_create"
	AuditEventAPIKeyUpdate AuditEventType = "apikey_update"
	AuditEventAPIKeyDelete AuditEventType = "apikey_delete"
	AuditEventAPIKeyRevoke AuditEventType = "apikey_revoke"
	AuditEventAPIKeyRotate AuditEventType = "apikey_rotate"

	// Encryption events (KEK/DEK)
	AuditEventKEKCreate AuditEventType = "kek_create"
	AuditEventKEKUpdate AuditEventType = "kek_update"
	AuditEventKEKDelete AuditEventType = "kek_delete"
	AuditEventDEKCreate AuditEventType = "dek_create"
	AuditEventDEKDelete AuditEventType = "dek_delete"

	// Exporter events
	AuditEventExporterCreate AuditEventType = "exporter_create"
	AuditEventExporterUpdate AuditEventType = "exporter_update"
	AuditEventExporterDelete AuditEventType = "exporter_delete"
	AuditEventExporterPause  AuditEventType = "exporter_pause"
	AuditEventExporterResume AuditEventType = "exporter_resume"
	AuditEventExporterReset  AuditEventType = "exporter_reset"

	// MCP events
	AuditEventMCPToolCall    AuditEventType = "mcp_tool_call"
	AuditEventMCPToolError   AuditEventType = "mcp_tool_error"
	AuditEventMCPAdminAction AuditEventType = "mcp_admin_action"

	// MCP confirmation events
	AuditEventMCPConfirmIssued   AuditEventType = "mcp_confirm_issued"
	AuditEventMCPConfirmRejected AuditEventType = "mcp_confirm_rejected"
	AuditEventMCPConfirmed       AuditEventType = "mcp_confirmed"
)

// AuditEvent represents an audit log entry.
type AuditEvent struct {
	Timestamp   time.Time         `json:"timestamp"`
	EventType   AuditEventType    `json:"event_type"`
	User        string            `json:"user,omitempty"`
	Role        string            `json:"role,omitempty"`
	ClientIP    string            `json:"client_ip"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	StatusCode  int               `json:"status_code"`
	Duration    time.Duration     `json:"duration_ms"`
	Subject     string            `json:"subject,omitempty"`
	Version     int               `json:"version,omitempty"`
	SchemaID    int64             `json:"schema_id,omitempty"`
	RequestBody string            `json:"request_body,omitempty"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	RequestID   string            `json:"request_id,omitempty"`
}

// AuditLogger handles audit logging.
type AuditLogger struct {
	config        config.AuditConfig
	logger        *slog.Logger
	file          *os.File
	mu            sync.Mutex
	enabledEvents map[AuditEventType]bool
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(cfg config.AuditConfig) (*AuditLogger, error) {
	al := &AuditLogger{
		config:        cfg,
		enabledEvents: make(map[AuditEventType]bool),
	}

	// Build set of enabled events
	if len(cfg.Events) == 0 {
		setDefaultEnabledEvents(al.enabledEvents)
	} else {
		for _, event := range cfg.Events {
			al.enabledEvents[AuditEventType(event)] = true
		}
	}

	// Open log file if specified
	if cfg.LogFile != "" {
		file, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		al.file = file
		al.logger = slog.New(slog.NewJSONHandler(file, nil))
	} else {
		al.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	return al, nil
}

// NewAuditLoggerWithWriter creates a new audit logger that writes to the provided writer.
// This is useful for testing, where a bytes.Buffer can be used to capture audit output.
func NewAuditLoggerWithWriter(cfg config.AuditConfig, w io.Writer) *AuditLogger {
	al := &AuditLogger{
		config:        cfg,
		enabledEvents: make(map[AuditEventType]bool),
	}

	// Build set of enabled events (same logic as NewAuditLogger)
	if len(cfg.Events) == 0 {
		setDefaultEnabledEvents(al.enabledEvents)
	} else {
		for _, event := range cfg.Events {
			al.enabledEvents[AuditEventType(event)] = true
		}
	}

	al.logger = slog.New(slog.NewJSONHandler(w, nil))
	return al
}

// Close closes the audit logger.
func (al *AuditLogger) Close() error {
	if al.file != nil {
		return al.file.Close()
	}
	return nil
}

// setDefaultEnabledEvents populates the enabled events map with all write operations,
// auth failures, and MCP events. Read-only events (schema_get, config_get, mode_get,
// subject_list) are excluded by default to reduce log volume.
func setDefaultEnabledEvents(m map[AuditEventType]bool) {
	// Schema write operations
	m[AuditEventSchemaRegister] = true
	m[AuditEventSchemaDelete] = true
	m[AuditEventSchemaImport] = true
	m[AuditEventSchemaLookup] = true

	// Config/mode write operations
	m[AuditEventConfigUpdate] = true
	m[AuditEventConfigDelete] = true
	m[AuditEventModeUpdate] = true
	m[AuditEventModeDelete] = true

	// Auth events
	m[AuditEventAuthFailure] = true
	m[AuditEventAuthForbidden] = true

	// Subject events
	m[AuditEventSubjectDelete] = true

	// Admin events
	m[AuditEventUserCreate] = true
	m[AuditEventUserUpdate] = true
	m[AuditEventUserDelete] = true
	m[AuditEventAPIKeyCreate] = true
	m[AuditEventAPIKeyUpdate] = true
	m[AuditEventAPIKeyDelete] = true
	m[AuditEventAPIKeyRevoke] = true
	m[AuditEventAPIKeyRotate] = true

	// Encryption events
	m[AuditEventKEKCreate] = true
	m[AuditEventKEKUpdate] = true
	m[AuditEventKEKDelete] = true
	m[AuditEventDEKCreate] = true
	m[AuditEventDEKDelete] = true

	// Exporter events
	m[AuditEventExporterCreate] = true
	m[AuditEventExporterUpdate] = true
	m[AuditEventExporterDelete] = true
	m[AuditEventExporterPause] = true
	m[AuditEventExporterResume] = true
	m[AuditEventExporterReset] = true

	// MCP events
	m[AuditEventMCPToolCall] = true
	m[AuditEventMCPToolError] = true
	m[AuditEventMCPAdminAction] = true
	m[AuditEventMCPConfirmIssued] = true
	m[AuditEventMCPConfirmRejected] = true
	m[AuditEventMCPConfirmed] = true
}

// Log logs an audit event.
func (al *AuditLogger) Log(event *AuditEvent) {
	if !al.config.Enabled {
		return
	}

	if !al.enabledEvents[event.EventType] {
		return
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	attrs := []slog.Attr{
		slog.Time("timestamp", event.Timestamp),
		slog.String("event_type", string(event.EventType)),
		slog.String("user", event.User),
		slog.String("role", event.Role),
		slog.String("client_ip", event.ClientIP),
		slog.String("method", event.Method),
		slog.String("path", event.Path),
		slog.Int("status_code", event.StatusCode),
		slog.Duration("duration", event.Duration),
		slog.String("subject", event.Subject),
		slog.Int("version", event.Version),
		slog.Int64("schema_id", event.SchemaID),
		slog.String("error", event.Error),
	}
	if event.RequestBody != "" {
		attrs = append(attrs, slog.String("request_body", event.RequestBody))
	}
	if event.RequestID != "" {
		attrs = append(attrs, slog.String("request_id", event.RequestID))
	}
	if len(event.Metadata) > 0 {
		metadataAttrs := make([]any, 0, len(event.Metadata)*2)
		for k, v := range event.Metadata {
			metadataAttrs = append(metadataAttrs, slog.String(k, v))
		}
		attrs = append(attrs, slog.Group("metadata", metadataAttrs...))
	}
	al.logger.LogAttrs(context.Background(), slog.LevelInfo, "audit", attrs...)
}

// Middleware returns HTTP middleware for audit logging.
func (al *AuditLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !al.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		// Capture request body if configured
		var requestBody string
		if al.config.IncludeBody && r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(body))
			requestBody = string(body)
			// Limit body size in logs
			if len(requestBody) > 1000 {
				requestBody = requestBody[:1000] + "..."
			}
		}

		// Create response wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Serve request
		next.ServeHTTP(rw, r)

		// Determine event type from request
		eventType := al.determineEventType(r, rw.statusCode)
		if eventType == "" {
			return
		}

		// Get user info from context
		user := GetUser(r.Context())
		var username, role string
		if user != nil {
			username = user.Username
			role = user.Role
		}

		event := &AuditEvent{
			Timestamp:   start,
			EventType:   eventType,
			User:        username,
			Role:        role,
			ClientIP:    getClientIP(r),
			Method:      r.Method,
			Path:        r.URL.Path,
			StatusCode:  rw.statusCode,
			Duration:    time.Since(start),
			RequestBody: requestBody,
			RequestID:   middleware.GetReqID(r.Context()),
		}

		// Extract subject from path if applicable
		event.Subject = extractSubject(r.URL.Path)

		al.Log(event)
	})
}

// determineEventType determines the audit event type from the request.
func (al *AuditLogger) determineEventType(r *http.Request, statusCode int) AuditEventType {
	path := r.URL.Path

	// Auth failure
	if statusCode == http.StatusUnauthorized {
		return AuditEventAuthFailure
	}
	if statusCode == http.StatusForbidden {
		return AuditEventAuthForbidden
	}

	// Import operations
	if contains(path, "/import/") && r.Method == "POST" {
		return AuditEventSchemaImport
	}

	// Schema operations — registration, deletion, retrieval via versioned paths
	if contains(path, "/subjects/") && contains(path, "/versions") {
		switch r.Method {
		case "POST":
			return AuditEventSchemaRegister
		case "DELETE":
			return AuditEventSchemaDelete
		case "GET":
			return AuditEventSchemaGet
		}
	}

	// Schema lookup via POST /subjects/{subject} (no /versions in path)
	if contains(path, "/subjects/") && !contains(path, "/versions") && r.Method == "POST" {
		return AuditEventSchemaLookup
	}

	// Schema lookup by ID
	if contains(path, "/schemas/ids/") {
		return AuditEventSchemaGet
	}

	// Subject delete
	if contains(path, "/subjects/") && !contains(path, "/versions") && r.Method == "DELETE" {
		return AuditEventSubjectDelete
	}

	// Subject list
	if path == "/subjects" && r.Method == "GET" {
		return AuditEventSubjectList
	}

	// Admin operations — user management
	if contains(path, "/admin/users") {
		switch r.Method {
		case "POST":
			return AuditEventUserCreate
		case "PUT":
			return AuditEventUserUpdate
		case "DELETE":
			return AuditEventUserDelete
		}
	}

	// Admin operations — API key management
	if contains(path, "/admin/apikeys") {
		if contains(path, "/revoke") && r.Method == "POST" {
			return AuditEventAPIKeyRevoke
		}
		if contains(path, "/rotate") && r.Method == "POST" {
			return AuditEventAPIKeyRotate
		}
		switch r.Method {
		case "POST":
			return AuditEventAPIKeyCreate
		case "PUT":
			return AuditEventAPIKeyUpdate
		case "DELETE":
			return AuditEventAPIKeyDelete
		}
	}

	// KEK operations
	if contains(path, "/dek-registry/v1/keks") {
		// DEK operations (path includes /deks/)
		if contains(path, "/deks/") {
			switch r.Method {
			case "POST":
				if contains(path, "/undelete") {
					return AuditEventDEKCreate // undelete restores a DEK
				}
				return AuditEventDEKCreate
			case "DELETE":
				return AuditEventDEKDelete
			}
		} else if contains(path, "/deks") && r.Method == "POST" {
			return AuditEventDEKCreate
		}

		// KEK operations (no /deks/ in path)
		if !contains(path, "/deks") {
			switch r.Method {
			case "POST":
				if contains(path, "/undelete") {
					return AuditEventKEKCreate // undelete restores a KEK
				}
				return AuditEventKEKCreate
			case "PUT":
				return AuditEventKEKUpdate
			case "DELETE":
				return AuditEventKEKDelete
			}
		}
	}

	// Exporter operations
	if contains(path, "/exporters") {
		if contains(path, "/pause") && r.Method == "PUT" {
			return AuditEventExporterPause
		}
		if contains(path, "/resume") && r.Method == "PUT" {
			return AuditEventExporterResume
		}
		if contains(path, "/reset") && r.Method == "PUT" {
			return AuditEventExporterReset
		}
		switch r.Method {
		case "POST":
			return AuditEventExporterCreate
		case "PUT":
			return AuditEventExporterUpdate
		case "DELETE":
			return AuditEventExporterDelete
		}
	}

	// Config operations
	if contains(path, "/config") {
		switch r.Method {
		case "GET":
			return AuditEventConfigGet
		case "PUT":
			return AuditEventConfigUpdate
		case "DELETE":
			return AuditEventConfigDelete
		}
	}

	// Mode operations
	if contains(path, "/mode") {
		switch r.Method {
		case "GET":
			return AuditEventModeGet
		case "PUT":
			return AuditEventModeUpdate
		case "DELETE":
			return AuditEventModeDelete
		}
	}

	return ""
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// extractSubject extracts the subject name from a URL path.
func extractSubject(path string) string {
	// Pattern: /subjects/{subject}/...
	if !contains(path, "/subjects/") {
		return ""
	}

	// Find start after /subjects/
	start := 10 // len("/subjects/")
	if start >= len(path) {
		return ""
	}

	// Find end (next slash or end of string)
	end := start
	for end < len(path) && path[end] != '/' {
		end++
	}

	return path[start:end]
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// LogEvent is a convenience function for logging events.
func (al *AuditLogger) LogEvent(eventType AuditEventType, r *http.Request, statusCode int, err error) {
	if !al.config.Enabled {
		return
	}

	user := GetUser(r.Context())
	var username, role string
	if user != nil {
		username = user.Username
		role = user.Role
	}

	var errStr string
	if err != nil {
		errStr = err.Error()
	}

	event := &AuditEvent{
		Timestamp:  time.Now(),
		EventType:  eventType,
		User:       username,
		Role:       role,
		ClientIP:   getClientIP(r),
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: statusCode,
		Subject:    extractSubject(r.URL.Path),
		Error:      errStr,
	}

	al.Log(event)
}

// LogMCPEvent logs an MCP tool call audit event.
// The user and role parameters identify the authenticated principal making the call.
// For unauthenticated calls, pass "unknown" as user and empty string as role.
func (al *AuditLogger) LogMCPEvent(eventType AuditEventType, user, role, toolName, status string, duration time.Duration, err error, subject string, metadata map[string]string) {
	if !al.config.Enabled {
		return
	}

	var errStr string
	if err != nil {
		errStr = err.Error()
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		User:      user,
		Role:      role,
		Method:    "MCP",
		Path:      toolName,
		Duration:  duration,
		Error:     errStr,
		Subject:   subject,
		Metadata:  metadata,
	}
	if status == "error" {
		event.StatusCode = 1
	}

	al.Log(event)
}

// LogMCPConfirmationEvent logs an MCP confirmation flow audit event.
// The user and role parameters identify the authenticated principal.
func (al *AuditLogger) LogMCPConfirmationEvent(eventType AuditEventType, user, role, toolName string, metadata map[string]string) {
	if !al.config.Enabled {
		return
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		User:      user,
		Role:      role,
		Method:    "MCP",
		Path:      toolName,
		Metadata:  metadata,
	}

	al.Log(event)
}

// MarshalJSON implements custom JSON marshaling for AuditEvent.
func (e *AuditEvent) MarshalJSON() ([]byte, error) {
	type Alias AuditEvent
	return json.Marshal(&struct {
		*Alias
		Duration int64 `json:"duration_ms"`
	}{
		Alias:    (*Alias)(e),
		Duration: e.Duration.Milliseconds(),
	})
}
