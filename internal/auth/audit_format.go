// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"fmt"
	"strings"
	"time"
)

// FormatJSON serializes an AuditEvent as a JSON line (with trailing newline).
func FormatJSON(event *AuditEvent) ([]byte, error) {
	data, err := event.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// FormatCEF serializes an AuditEvent as a CEF (Common Event Format) line.
// Format: CEF:0|AxonOps|SchemaRegistry|1.0|{EVENT_TYPE}|{description}|{severity}|{extensions}
func FormatCEF(event *AuditEvent) []byte {
	severity := cefSeverity(event)
	description := cefDescription(event)

	// Build extension key=value pairs
	var ext strings.Builder
	writeExtField(&ext, "rt", event.Timestamp.UTC().Format(time.RFC3339))
	writeExtField(&ext, "outcome", event.Outcome)
	if event.ActorID != "" {
		writeExtField(&ext, "suser", event.ActorID)
	}
	if event.ActorType != "" {
		writeExtField(&ext, "cs1", event.ActorType)
		writeExtField(&ext, "cs1Label", "actorType")
	}
	if event.AuthMethod != "" {
		writeExtField(&ext, "cs2", event.AuthMethod)
		writeExtField(&ext, "cs2Label", "authMethod")
	}
	if event.Role != "" {
		writeExtField(&ext, "cs3", event.Role)
		writeExtField(&ext, "cs3Label", "role")
	}
	if event.TargetType != "" {
		writeExtField(&ext, "cs4", event.TargetType)
		writeExtField(&ext, "cs4Label", "targetType")
	}
	if event.TargetID != "" {
		writeExtField(&ext, "cs5", event.TargetID)
		writeExtField(&ext, "cs5Label", "targetID")
	}
	if event.SourceIP != "" {
		writeExtField(&ext, "src", event.SourceIP)
	}
	if event.UserAgent != "" {
		writeExtField(&ext, "requestClientApplication", event.UserAgent)
	}
	writeExtField(&ext, "requestMethod", event.Method)
	writeExtField(&ext, "request", event.Path)
	if event.StatusCode != 0 {
		writeExtField(&ext, "cn1", fmt.Sprintf("%d", event.StatusCode))
		writeExtField(&ext, "cn1Label", "statusCode")
	}
	if event.SchemaID != 0 {
		writeExtField(&ext, "cn2", fmt.Sprintf("%d", event.SchemaID))
		writeExtField(&ext, "cn2Label", "schemaID")
	}
	if event.Duration > 0 {
		writeExtField(&ext, "cn3", fmt.Sprintf("%d", event.Duration.Milliseconds()))
		writeExtField(&ext, "cn3Label", "durationMs")
	}
	if event.Context != "" {
		writeExtField(&ext, "cs6", event.Context)
		writeExtField(&ext, "cs6Label", "context")
	}
	if event.Reason != "" {
		writeExtField(&ext, "reason", event.Reason)
	}
	if event.Error != "" {
		writeExtField(&ext, "msg", event.Error)
	}
	if event.BeforeHash != "" {
		writeExtField(&ext, "oldFileHash", event.BeforeHash)
	}
	if event.AfterHash != "" {
		writeExtField(&ext, "fileHash", event.AfterHash)
	}
	if event.RequestID != "" {
		writeExtField(&ext, "externalId", event.RequestID)
	}

	line := fmt.Sprintf("CEF:0|AxonOps|SchemaRegistry|1.0|%s|%s|%d|%s\n",
		cefEscapeHeader(string(event.EventType)),
		cefEscapeHeader(description),
		severity,
		ext.String(),
	)
	return []byte(line)
}

// cefSeverity maps audit events to CEF severity levels (0-10).
func cefSeverity(event *AuditEvent) int {
	if event.Outcome == "failure" {
		switch event.EventType {
		case AuditEventAuthFailure, AuditEventAuthForbidden:
			return 8
		default:
			return 5
		}
	}
	switch event.EventType {
	case AuditEventAuthFailure, AuditEventAuthForbidden:
		return 8
	case AuditEventSchemaRegister,
		AuditEventSchemaDeleteSoft, AuditEventSchemaDeletePermanent,
		AuditEventSubjectDeleteSoft, AuditEventSubjectDeletePermanent,
		AuditEventConfigUpdate, AuditEventConfigDelete,
		AuditEventModeUpdate, AuditEventModeDelete,
		AuditEventSchemaImport, AuditEventCompatibilityCheck,
		AuditEventUserCreate, AuditEventUserUpdate, AuditEventUserDelete,
		AuditEventPasswordChange,
		AuditEventAPIKeyCreate, AuditEventAPIKeyUpdate, AuditEventAPIKeyDelete,
		AuditEventAPIKeyRevoke, AuditEventAPIKeyRotate,
		AuditEventKEKCreate, AuditEventKEKUpdate,
		AuditEventKEKDeleteSoft, AuditEventKEKDeletePermanent,
		AuditEventKEKUndelete, AuditEventKEKTest,
		AuditEventDEKCreate,
		AuditEventDEKDeleteSoft, AuditEventDEKDeletePermanent,
		AuditEventDEKUndelete,
		AuditEventExporterCreate, AuditEventExporterUpdate, AuditEventExporterDelete,
		AuditEventExporterPause, AuditEventExporterResume, AuditEventExporterReset,
		AuditEventExporterConfigUpdate,
		AuditEventServerStartup, AuditEventServerShutdown:
		return 5
	case AuditEventMCPToolCall, AuditEventMCPToolError, AuditEventMCPAdminAction,
		AuditEventMCPConfirmIssued, AuditEventMCPConfirmRejected, AuditEventMCPConfirmed:
		return 5
	case AuditEventSchemaGet, AuditEventSchemaLookup,
		AuditEventConfigGet, AuditEventModeGet, AuditEventSubjectList:
		return 3
	default:
		return 3
	}
}

// cefDescription returns a human-readable description for the event type.
func cefDescription(event *AuditEvent) string {
	switch event.EventType {
	case AuditEventSchemaRegister:
		return "Schema registered"
	case AuditEventSchemaDeleteSoft:
		return "Schema soft-deleted"
	case AuditEventSchemaDeletePermanent:
		return "Schema permanently deleted"
	case AuditEventSchemaGet:
		return "Schema retrieved"
	case AuditEventSchemaLookup:
		return "Schema lookup"
	case AuditEventSchemaImport:
		return "Schema imported"
	case AuditEventConfigGet:
		return "Config retrieved"
	case AuditEventConfigUpdate:
		return "Config updated"
	case AuditEventConfigDelete:
		return "Config deleted"
	case AuditEventModeGet:
		return "Mode retrieved"
	case AuditEventModeUpdate:
		return "Mode updated"
	case AuditEventModeDelete:
		return "Mode deleted"
	case AuditEventAuthSuccess:
		return "Authentication succeeded"
	case AuditEventAuthFailure:
		return "Authentication failed"
	case AuditEventAuthForbidden:
		return "Access forbidden"
	case AuditEventSubjectDeleteSoft:
		return "Subject soft-deleted"
	case AuditEventSubjectDeletePermanent:
		return "Subject permanently deleted"
	case AuditEventSubjectList:
		return "Subjects listed"
	case AuditEventUserCreate:
		return "User created"
	case AuditEventUserUpdate:
		return "User updated"
	case AuditEventUserDelete:
		return "User deleted"
	case AuditEventPasswordChange:
		return "Password changed"
	case AuditEventAPIKeyCreate:
		return "API key created"
	case AuditEventAPIKeyUpdate:
		return "API key updated"
	case AuditEventAPIKeyDelete:
		return "API key deleted"
	case AuditEventAPIKeyRevoke:
		return "API key revoked"
	case AuditEventAPIKeyRotate:
		return "API key rotated"
	case AuditEventKEKCreate:
		return "KEK created"
	case AuditEventKEKUpdate:
		return "KEK updated"
	case AuditEventKEKDeleteSoft:
		return "KEK soft-deleted"
	case AuditEventKEKDeletePermanent:
		return "KEK permanently deleted"
	case AuditEventKEKUndelete:
		return "KEK undeleted"
	case AuditEventKEKTest:
		return "KEK tested"
	case AuditEventDEKCreate:
		return "DEK created"
	case AuditEventDEKDeleteSoft:
		return "DEK soft-deleted"
	case AuditEventDEKDeletePermanent:
		return "DEK permanently deleted"
	case AuditEventDEKUndelete:
		return "DEK undeleted"
	case AuditEventExporterCreate:
		return "Exporter created"
	case AuditEventExporterUpdate:
		return "Exporter updated"
	case AuditEventExporterDelete:
		return "Exporter deleted"
	case AuditEventExporterPause:
		return "Exporter paused"
	case AuditEventExporterResume:
		return "Exporter resumed"
	case AuditEventExporterReset:
		return "Exporter reset"
	case AuditEventExporterConfigUpdate:
		return "Exporter config updated"
	case AuditEventCompatibilityCheck:
		return "Compatibility check"
	case AuditEventServerStartup:
		return "Server started"
	case AuditEventServerShutdown:
		return "Server stopped"
	case AuditEventMCPToolCall:
		return "MCP tool call"
	case AuditEventMCPToolError:
		return "MCP tool error"
	case AuditEventMCPAdminAction:
		return "MCP admin action"
	case AuditEventMCPConfirmIssued:
		return "MCP confirmation issued"
	case AuditEventMCPConfirmRejected:
		return "MCP confirmation rejected"
	case AuditEventMCPConfirmed:
		return "MCP action confirmed"
	default:
		return string(event.EventType)
	}
}

// cefEscapeHeader escapes characters in CEF header fields.
// Header fields use pipe (|) as delimiter — pipes and backslashes must be escaped.
func cefEscapeHeader(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `|`, `\|`)
	return s
}

// cefEscapeExtValue escapes characters in CEF extension values.
// Extension values use equals (=) as key-value separator — equals and backslashes must be escaped.
func cefEscapeExtValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `=`, `\=`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

// writeExtField writes a key=value pair to the extension string.
func writeExtField(b *strings.Builder, key, value string) {
	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(cefEscapeExtValue(value))
}
