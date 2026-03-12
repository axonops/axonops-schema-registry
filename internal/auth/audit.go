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

// AuditOutput writes serialized audit events to a destination.
// Implementations MUST be goroutine-safe.
type AuditOutput interface {
	// Write writes pre-serialized audit event data to the output.
	// Implementations SHOULD be non-blocking and MUST handle errors
	// internally for best-effort delivery.
	Write(data []byte) error
	// Close flushes pending data and releases resources.
	Close() error
	// Name returns a human-readable output name for logging and metrics.
	Name() string
}

// formattedOutput pairs an AuditOutput with a serialization format.
type formattedOutput struct {
	output     AuditOutput
	formatType string // "json" or "cef"
}

// StdoutOutput writes audit events to os.Stdout.
type StdoutOutput struct{}

// Write writes data to stdout.
func (o *StdoutOutput) Write(data []byte) error {
	_, err := os.Stdout.Write(data)
	return err
}

// Close is a no-op for stdout.
func (o *StdoutOutput) Close() error { return nil }

// Name returns "stdout".
func (o *StdoutOutput) Name() string { return "stdout" }

// WriterOutput wraps an io.Writer as an AuditOutput.
type WriterOutput struct {
	w      io.Writer
	closer io.Closer // optional; closed on Close()
	name   string
}

// Write writes data to the underlying writer.
func (o *WriterOutput) Write(data []byte) error {
	_, err := o.w.Write(data)
	return err
}

// Close closes the underlying closer if set.
func (o *WriterOutput) Close() error {
	if o.closer != nil {
		return o.closer.Close()
	}
	return nil
}

// Name returns the output name.
func (o *WriterOutput) Name() string { return o.name }

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
	AuditEventUserCreate     AuditEventType = "user_create"
	AuditEventUserUpdate     AuditEventType = "user_update"
	AuditEventUserDelete     AuditEventType = "user_delete"
	AuditEventPasswordChange AuditEventType = "password_change"
	AuditEventAPIKeyCreate   AuditEventType = "apikey_create"
	AuditEventAPIKeyUpdate   AuditEventType = "apikey_update"
	AuditEventAPIKeyDelete   AuditEventType = "apikey_delete"
	AuditEventAPIKeyRevoke   AuditEventType = "apikey_revoke"
	AuditEventAPIKeyRotate   AuditEventType = "apikey_rotate"

	// Encryption events (KEK/DEK)
	AuditEventKEKCreate AuditEventType = "kek_create"
	AuditEventKEKUpdate AuditEventType = "kek_update"
	AuditEventKEKDelete AuditEventType = "kek_delete"
	AuditEventKEKTest   AuditEventType = "kek_test"
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

// AuditEvent represents an audit log entry following industry-standard audit
// logging practices. See docs/auditing.md for the full field reference.
type AuditEvent struct {
	// Timing
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration_ms"`

	// Event classification
	EventType AuditEventType `json:"event_type"`
	Outcome   string         `json:"outcome"` // "success" or "failure"

	// Actor (who performed the action)
	ActorID    string `json:"actor_id,omitempty"`    // username, key name, or MCP principal
	ActorType  string `json:"actor_type,omitempty"`  // user, api_key, mcp_client, anonymous
	Role       string `json:"role,omitempty"`        // admin, developer, readonly
	AuthMethod string `json:"auth_method,omitempty"` // basic, api_key, jwt, oidc, ldap, mtls, bearer_token

	// Target (what was affected)
	TargetType string `json:"target_type,omitempty"` // subject, schema, config, mode, kek, dek, exporter, user, apikey
	TargetID   string `json:"target_id,omitempty"`   // subject name, KEK name, exporter name, etc.
	SchemaID   int64  `json:"schema_id,omitempty"`
	Version    int    `json:"version,omitempty"`
	SchemaType string `json:"schema_type,omitempty"` // AVRO, PROTOBUF, JSON

	// Change integrity
	BeforeHash string `json:"before_hash,omitempty"` // sha256:xxxx of object before change
	AfterHash  string `json:"after_hash,omitempty"`  // sha256:xxxx of object after change

	// Context and correlation
	Context   string `json:"context,omitempty"` // registryCtx namespace
	RequestID string `json:"request_id,omitempty"`

	// Transport
	SourceIP   string `json:"source_ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	Method     string `json:"method"`                // HTTP method or "MCP"
	Path       string `json:"path"`                  // HTTP path or MCP tool name
	StatusCode int    `json:"status_code,omitempty"` // HTTP status (REST only)

	// Detail
	Reason      string            `json:"reason,omitempty"`       // structured failure reason
	Error       string            `json:"error,omitempty"`        // raw error message
	RequestBody string            `json:"request_body,omitempty"` // truncated request body
	Metadata    map[string]string `json:"metadata,omitempty"`     // event-specific extras
}

// AuditLogger handles audit logging.
type AuditLogger struct {
	config        config.AuditConfig
	outputs       []formattedOutput
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

	// Determine which outputs to configure.
	// Priority: new outputs config > legacy log_file > default (stdout).
	hasExplicitOutputs := cfg.Outputs.Stdout.Enabled || cfg.Outputs.File.Enabled ||
		cfg.Outputs.Syslog.Enabled || cfg.Outputs.Webhook.Enabled

	if hasExplicitOutputs {
		if cfg.Outputs.Stdout.Enabled {
			al.outputs = append(al.outputs, formattedOutput{
				output:     &StdoutOutput{},
				formatType: normalizeFormat(cfg.Outputs.Stdout.FormatType),
			})
		}
		if cfg.Outputs.File.Enabled {
			fileOut, err := NewFileOutput(cfg.Outputs.File)
			if err != nil {
				return nil, err
			}
			al.outputs = append(al.outputs, formattedOutput{
				output:     fileOut,
				formatType: normalizeFormat(cfg.Outputs.File.FormatType),
			})
		}
		// Syslog and webhook outputs are wired in later phases.
	} else if cfg.LogFile != "" {
		// Legacy config: single file output
		file, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		al.outputs = append(al.outputs, formattedOutput{
			output:     &WriterOutput{w: file, closer: file, name: "file"},
			formatType: "json",
		})
	} else {
		// Default: stdout
		al.outputs = append(al.outputs, formattedOutput{
			output:     &StdoutOutput{},
			formatType: "json",
		})
	}

	return al, nil
}

// normalizeFormat returns "json" for empty/unknown format strings.
func normalizeFormat(f string) string {
	switch f {
	case "cef":
		return "cef"
	default:
		return "json"
	}
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

	al.outputs = append(al.outputs, formattedOutput{
		output:     &WriterOutput{w: w, name: "writer"},
		formatType: "json",
	})
	return al
}

// Close closes all audit outputs.
func (al *AuditLogger) Close() error {
	var firstErr error
	for _, fo := range al.outputs {
		if err := fo.output.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
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
	m[AuditEventPasswordChange] = true
	m[AuditEventAPIKeyCreate] = true
	m[AuditEventAPIKeyUpdate] = true
	m[AuditEventAPIKeyDelete] = true
	m[AuditEventAPIKeyRevoke] = true
	m[AuditEventAPIKeyRotate] = true

	// Encryption events
	m[AuditEventKEKCreate] = true
	m[AuditEventKEKUpdate] = true
	m[AuditEventKEKDelete] = true
	m[AuditEventKEKTest] = true
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

// Log logs an audit event by serializing it and writing to all configured outputs.
func (al *AuditLogger) Log(event *AuditEvent) {
	if !al.config.Enabled {
		return
	}

	if !al.enabledEvents[event.EventType] {
		return
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		slog.Warn("failed to marshal audit event", slog.String("error", err.Error()))
		return
	}
	data = append(data, '\n')

	for _, fo := range al.outputs {
		if writeErr := fo.output.Write(data); writeErr != nil {
			slog.Warn("failed to write audit event",
				slog.String("output", fo.output.Name()),
				slog.String("error", writeErr.Error()),
			)
		}
	}
}

// AuditHints allows handlers to pass additional audit information (such as
// before/after hashes and schema metadata) back to the audit middleware.
// The middleware injects a mutable *AuditHints into the request context before
// calling the handler; the handler fills in the fields it knows about.
type AuditHints struct {
	BeforeHash string // sha256:xxxx of the object before a change
	AfterHash  string // sha256:xxxx of the object after a change
	SchemaType string // AVRO, PROTOBUF, JSON
	SchemaID   int64
	Version    int
	Context    string // registry context namespace

	// Target fields — populated by handlers when the URL path alone is not
	// sufficient to determine the target (e.g., bulk import where subjects
	// are in the request body, not the URL).
	TargetType string // subject, schema, config, mode, kek, dek, exporter, user, apikey
	TargetID   string // subject name, KEK name, exporter name, etc.

	// Actor fields — populated by the auth middleware so the audit middleware
	// can read them even though the auth middleware runs after the audit
	// middleware in the chi middleware chain. Using a shared mutable pointer
	// solves the Go context layering problem where r.WithContext() creates a
	// new *Request invisible to outer middleware.
	ActorID    string // username, key name, or MCP principal
	ActorType  string // user, api_key, mcp_client, anonymous
	Role       string // admin, developer, readonly
	AuthMethod string // basic, api_key, jwt, oidc, ldap, mtls, bearer_token
}

type auditHintsKey struct{}

// GetAuditHints retrieves the mutable AuditHints from the request context.
// Returns nil if no hints are present (e.g., audit middleware is disabled).
// Handlers call this to set before_hash, after_hash, and other audit metadata.
func GetAuditHints(ctx context.Context) *AuditHints {
	if hints, ok := ctx.Value(auditHintsKey{}).(*AuditHints); ok {
		return hints
	}
	return nil
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

		// Inject mutable audit hints into the request context.
		// Handlers fill in fields like BeforeHash and AfterHash.
		hints := &AuditHints{}
		r = r.WithContext(context.WithValue(r.Context(), auditHintsKey{}, hints))

		// Create response wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Serve request
		next.ServeHTTP(rw, r)

		// Determine event type from request
		eventType := al.determineEventType(r, rw.statusCode)
		if eventType == "" {
			return
		}

		// Read actor info from hints (populated by the auth middleware via the
		// shared mutable pointer), or fall back to the user in context (for
		// tests and code paths that set user context directly).
		var actorID, role, authMethod, actorType string
		if hints.ActorType != "" {
			actorID = hints.ActorID
			actorType = hints.ActorType
			role = hints.Role
			authMethod = hints.AuthMethod
		} else if user := GetUser(r.Context()); user != nil {
			actorID = user.Username
			role = user.Role
			authMethod = user.Method
			actorType = actorTypeFromAuthMethod(authMethod)
		} else {
			actorType = "anonymous"
		}

		outcome := outcomeFromStatusCode(rw.statusCode)
		reason := reasonFromStatusCode(rw.statusCode)
		targetType, targetID := extractTarget(r.URL.Path, eventType)

		// Prefer handler-supplied target info over URL-extracted info.
		// This covers cases like bulk import where the target is in the
		// request body, not the URL path.
		if hints.TargetID != "" {
			targetID = hints.TargetID
		}
		if hints.TargetType != "" {
			targetType = hints.TargetType
		}

		event := &AuditEvent{
			Timestamp:   start,
			Duration:    time.Since(start),
			EventType:   eventType,
			Outcome:     outcome,
			ActorID:     actorID,
			ActorType:   actorType,
			Role:        role,
			AuthMethod:  authMethod,
			TargetType:  targetType,
			TargetID:    targetID,
			BeforeHash:  hints.BeforeHash,
			AfterHash:   hints.AfterHash,
			SchemaType:  hints.SchemaType,
			SchemaID:    hints.SchemaID,
			Version:     hints.Version,
			Context:     hints.Context,
			SourceIP:    getClientIP(r),
			UserAgent:   r.UserAgent(),
			Method:      r.Method,
			Path:        r.URL.Path,
			StatusCode:  rw.statusCode,
			Reason:      reason,
			RequestBody: requestBody,
			RequestID:   middleware.GetReqID(r.Context()),
		}

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

	// Account self-service — password change
	if contains(path, "/me/password") && r.Method == "POST" {
		return AuditEventPasswordChange
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
			// KEK test endpoint
			if contains(path, "/test") && r.Method == "POST" {
				return AuditEventKEKTest
			}
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
// Supports /subjects/{subject}/..., /config/{subject}, and /mode/{subject}.
func extractSubject(path string) string {
	// Try /subjects/{subject}/... first
	if contains(path, "/subjects/") {
		start := 10 // len("/subjects/")
		if start < len(path) {
			end := start
			for end < len(path) && path[end] != '/' {
				end++
			}
			return path[start:end]
		}
	}

	// Try /config/{subject} (per-subject config, not global /config)
	if len(path) > len("/config/") && contains(path, "/config/") {
		start := 8 // len("/config/")
		end := start
		for end < len(path) && path[end] != '/' {
			end++
		}
		if end > start {
			return path[start:end]
		}
	}

	// Try /mode/{subject} (per-subject mode, not global /mode)
	if len(path) > len("/mode/") && contains(path, "/mode/") {
		start := 6 // len("/mode/")
		end := start
		for end < len(path) && path[end] != '/' {
			end++
		}
		if end > start {
			return path[start:end]
		}
	}

	return ""
}

// actorTypeFromAuthMethod derives the actor type from the authentication method.
func actorTypeFromAuthMethod(method string) string {
	switch method {
	case "api_key":
		return "api_key"
	case "basic", "jwt", "oidc", "ldap", "mtls":
		return "user"
	default:
		return "anonymous"
	}
}

// outcomeFromStatusCode returns "success" or "failure" based on the HTTP status code.
func outcomeFromStatusCode(statusCode int) string {
	if statusCode >= 200 && statusCode < 400 {
		return "success"
	}
	return "failure"
}

// reasonFromStatusCode returns a structured reason code for failure status codes.
// Returns empty string for success status codes.
func reasonFromStatusCode(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 400:
		return ""
	case statusCode == http.StatusUnauthorized:
		return "no_valid_credentials"
	case statusCode == http.StatusForbidden:
		return "permission_denied"
	case statusCode == http.StatusNotFound:
		return "not_found"
	case statusCode == http.StatusConflict:
		return "already_exists"
	case statusCode == http.StatusBadRequest:
		return "validation_error"
	case statusCode == 422:
		return "invalid_schema"
	case statusCode == http.StatusTooManyRequests:
		return "rate_limited"
	default:
		if statusCode >= 500 {
			return "internal_error"
		}
		return ""
	}
}

// classifyMCPError derives a structured reason code from an MCP error message.
func classifyMCPError(errMsg string) string {
	lower := toLower(errMsg)
	switch {
	case contains(lower, "not found"):
		return "not_found"
	case contains(lower, "permission") || contains(lower, "forbidden") || contains(lower, "unauthorized"):
		return "permission_denied"
	case contains(lower, "already exists") || contains(lower, "duplicate"):
		return "already_exists"
	case contains(lower, "invalid schema") || contains(lower, "parse"):
		return "invalid_schema"
	case contains(lower, "incompatible"):
		return "incompatible"
	case contains(lower, "invalid") || contains(lower, "required") || contains(lower, "missing"):
		return "validation_error"
	default:
		return "internal_error"
	}
}

// toLower converts ASCII uppercase to lowercase without importing strings.
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// extractTarget derives the target_type and target_id from the URL path and event type.
func extractTarget(path string, eventType AuditEventType) (targetType, targetID string) {
	switch {
	// Subject/schema operations
	case contains(path, "/subjects/"):
		subject := extractSubject(path)
		if subject != "" {
			return "subject", subject
		}
	// Schema by ID
	case contains(path, "/schemas/ids/"):
		start := 13 // len("/schemas/ids/")
		if start < len(path) {
			end := start
			for end < len(path) && path[end] != '/' {
				end++
			}
			return "schema", path[start:end]
		}
	// Config operations
	case contains(path, "/config"):
		subject := extractSubject(path)
		if subject != "" {
			return "config", subject
		}
		return "config", "_global"
	// Mode operations
	case contains(path, "/mode"):
		subject := extractSubject(path)
		if subject != "" {
			return "mode", subject
		}
		return "mode", "_global"
	// KEK/DEK operations
	case contains(path, "/dek-registry/v1/keks"):
		return extractKEKDEKTarget(path)
	// Exporter operations
	case contains(path, "/exporters"):
		return extractExporterTarget(path)
	// Admin user operations
	case contains(path, "/admin/users"):
		return extractAdminTarget(path, "/admin/users/", "user")
	// Admin API key operations
	case contains(path, "/admin/apikeys"):
		return extractAdminTarget(path, "/admin/apikeys/", "apikey")
	// Import
	case contains(path, "/import/"):
		return "schema", ""
	}

	return "", ""
}

// extractKEKDEKTarget extracts target type and ID from KEK/DEK paths.
func extractKEKDEKTarget(path string) (string, string) {
	// /dek-registry/v1/keks/{kekName}/deks/{subject}...
	prefix := "/dek-registry/v1/keks/"
	if !contains(path, prefix) || len(path) <= len(prefix) {
		return "kek", ""
	}
	rest := path[len(prefix):]
	// Extract KEK name
	end := 0
	for end < len(rest) && rest[end] != '/' {
		end++
	}
	kekName := rest[:end]

	// Check if this is a DEK operation
	if contains(rest, "/deks") {
		return "dek", kekName
	}
	return "kek", kekName
}

// extractExporterTarget extracts the exporter name from the path.
func extractExporterTarget(path string) (string, string) {
	prefix := "/exporters/"
	if !contains(path, prefix) || len(path) <= len(prefix) {
		return "exporter", ""
	}
	rest := path[len(prefix):]
	end := 0
	for end < len(rest) && rest[end] != '/' {
		end++
	}
	return "exporter", rest[:end]
}

// extractAdminTarget extracts the target ID from admin paths.
func extractAdminTarget(path, prefix, targetType string) (string, string) {
	if !contains(path, prefix) || len(path) <= len(prefix) {
		return targetType, ""
	}
	rest := path[len(prefix):]
	end := 0
	for end < len(rest) && rest[end] != '/' {
		end++
	}
	return targetType, rest[:end]
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

	// Read actor info from AuditHints (populated by auth middleware) or
	// fall back to context user (for callers outside the middleware chain).
	var actorID, role, authMethod, actorType string
	if hints := GetAuditHints(r.Context()); hints != nil && hints.ActorType != "" {
		actorID = hints.ActorID
		actorType = hints.ActorType
		role = hints.Role
		authMethod = hints.AuthMethod
	} else if user := GetUser(r.Context()); user != nil {
		actorID = user.Username
		role = user.Role
		authMethod = user.Method
		actorType = actorTypeFromAuthMethod(authMethod)
	} else {
		actorType = "anonymous"
	}

	var errStr string
	if err != nil {
		errStr = err.Error()
	}

	outcome := outcomeFromStatusCode(statusCode)
	reason := reasonFromStatusCode(statusCode)
	targetType, targetID := extractTarget(r.URL.Path, eventType)

	event := &AuditEvent{
		Timestamp:  time.Now(),
		EventType:  eventType,
		Outcome:    outcome,
		ActorID:    actorID,
		ActorType:  actorType,
		Role:       role,
		AuthMethod: authMethod,
		TargetType: targetType,
		TargetID:   targetID,
		SourceIP:   getClientIP(r),
		UserAgent:  r.UserAgent(),
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: statusCode,
		Reason:     reason,
		Error:      errStr,
	}

	al.Log(event)
}

// LogMCPEvent logs an MCP tool call audit event.
// The actorID and actorType identify the authenticated principal.
// The authMethod indicates how the MCP client authenticated ("bearer_token" or "").
func (al *AuditLogger) LogMCPEvent(eventType AuditEventType, actorID, actorType, authMethod, toolName, status string, duration time.Duration, err error, subject string, metadata map[string]string) {
	if !al.config.Enabled {
		return
	}

	var errStr, reason string
	outcome := "success"
	if err != nil {
		errStr = err.Error()
		outcome = "failure"
		reason = classifyMCPError(errStr)
	} else if status == "error" {
		outcome = "failure"
		reason = "internal_error"
	}

	var targetType, targetID string
	if subject != "" {
		targetType = "subject"
		targetID = subject
	}

	event := &AuditEvent{
		Timestamp:  time.Now(),
		Duration:   duration,
		EventType:  eventType,
		Outcome:    outcome,
		ActorID:    actorID,
		ActorType:  actorType,
		AuthMethod: authMethod,
		TargetType: targetType,
		TargetID:   targetID,
		Method:     "MCP",
		Path:       toolName,
		Reason:     reason,
		Error:      errStr,
		Metadata:   metadata,
	}

	al.Log(event)
}

// LogMCPConfirmationEvent logs an MCP confirmation flow audit event.
// The actorID and actorType identify the authenticated principal.
func (al *AuditLogger) LogMCPConfirmationEvent(eventType AuditEventType, actorID, actorType, authMethod, toolName string, metadata map[string]string) {
	if !al.config.Enabled {
		return
	}

	event := &AuditEvent{
		Timestamp:  time.Now(),
		EventType:  eventType,
		Outcome:    "success",
		ActorID:    actorID,
		ActorType:  actorType,
		AuthMethod: authMethod,
		Method:     "MCP",
		Path:       toolName,
		Metadata:   metadata,
	}

	al.Log(event)
}

// MarshalJSON implements custom JSON marshaling for AuditEvent.
// Converts the Duration field from time.Duration to integer milliseconds.
func (e *AuditEvent) MarshalJSON() ([]byte, error) {
	type Alias AuditEvent
	return json.Marshal(&struct {
		*Alias
		DurationMs int64 `json:"duration_ms"`
	}{
		Alias:      (*Alias)(e),
		DurationMs: e.Duration.Milliseconds(),
	})
}
