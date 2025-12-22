// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

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

	// Config events
	AuditEventConfigGet    AuditEventType = "config_get"
	AuditEventConfigUpdate AuditEventType = "config_update"
	AuditEventConfigDelete AuditEventType = "config_delete"

	// Mode events
	AuditEventModeGet    AuditEventType = "mode_get"
	AuditEventModeUpdate AuditEventType = "mode_update"

	// Auth events
	AuditEventAuthSuccess   AuditEventType = "auth_success"
	AuditEventAuthFailure   AuditEventType = "auth_failure"
	AuditEventAuthForbidden AuditEventType = "auth_forbidden"

	// Subject events
	AuditEventSubjectDelete AuditEventType = "subject_delete"
	AuditEventSubjectList   AuditEventType = "subject_list"
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
		// Enable all events by default
		al.enabledEvents[AuditEventSchemaRegister] = true
		al.enabledEvents[AuditEventSchemaDelete] = true
		al.enabledEvents[AuditEventConfigUpdate] = true
		al.enabledEvents[AuditEventModeUpdate] = true
		al.enabledEvents[AuditEventAuthFailure] = true
		al.enabledEvents[AuditEventAuthForbidden] = true
		al.enabledEvents[AuditEventSubjectDelete] = true
	} else {
		for _, event := range cfg.Events {
			al.enabledEvents[AuditEventType(event)] = true
		}
	}

	// Open log file if specified
	if cfg.LogFile != "" {
		file, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

// Close closes the audit logger.
func (al *AuditLogger) Close() error {
	if al.file != nil {
		return al.file.Close()
	}
	return nil
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

	al.logger.Info("audit",
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
	)
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

	// Schema operations
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

	// Schema lookup
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
