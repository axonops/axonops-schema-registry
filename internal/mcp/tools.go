package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/auth"
)

func (s *Server) registerTools() {
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "health_check",
		Description: "Check if the schema registry is healthy and responding",
		Annotations: &gomcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, instrumentedHandler(s, "health_check", s.handleHealthCheck))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_server_info",
		Description: "Get schema registry server information including version and supported schema types",
		Annotations: &gomcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, instrumentedHandler(s, "get_server_info", s.handleGetServerInfo))

	s.registerSchemaReadTools()
	s.registerSchemaWriteTools()
	s.registerConfigTools()
	s.registerContextTools()
	s.registerDEKTools()
	s.registerExporterTools()
	s.registerMetadataTools()
	s.registerAdminTools()
	s.registerValidationTools()
	s.registerComparisonTools()
	s.registerIntelligenceTools()
	s.registerMetricsTools()

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "list_subjects",
		Description: "List all registered subjects in the schema registry",
		Annotations: &gomcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, instrumentedHandler(s, "list_subjects", s.handleListSubjects))
}

type healthCheckInput struct{}

func (s *Server) handleHealthCheck(_ context.Context, _ *gomcp.CallToolRequest, _ healthCheckInput) (*gomcp.CallToolResult, any, error) {
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{
			&gomcp.TextContent{Text: "Schema registry is healthy"},
		},
	}, nil, nil
}

type serverInfoInput struct{}

func (s *Server) handleGetServerInfo(_ context.Context, _ *gomcp.CallToolRequest, _ serverInfoInput) (*gomcp.CallToolResult, any, error) {
	return jsonResult(map[string]any{
		"version":      s.version,
		"schema_types": []string{"AVRO", "PROTOBUF", "JSON"},
	})
}

type listSubjectsInput struct {
	Deleted bool   `json:"deleted,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
	Pattern string `json:"pattern,omitempty"`
	Context string `json:"context,omitempty"`
}

func (s *Server) handleListSubjects(ctx context.Context, _ *gomcp.CallToolRequest, input listSubjectsInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, resolveContext(input.Context), input.Deleted)
	if err != nil {
		return &gomcp.CallToolResult{
			Content: []gomcp.Content{
				&gomcp.TextContent{Text: fmt.Sprintf("error: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	if input.Prefix != "" {
		var filtered []string
		for _, subj := range subjects {
			if strings.HasPrefix(subj, input.Prefix) {
				filtered = append(filtered, subj)
			}
		}
		subjects = filtered
	}

	if input.Pattern != "" {
		re, err := regexp.Compile(input.Pattern)
		if err != nil {
			return &gomcp.CallToolResult{
				Content: []gomcp.Content{
					&gomcp.TextContent{Text: fmt.Sprintf("error: invalid regex pattern: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}
		var filtered []string
		for _, subj := range subjects {
			if re.MatchString(subj) {
				filtered = append(filtered, subj)
			}
		}
		subjects = filtered
	}

	if subjects == nil {
		subjects = []string{}
	}
	return jsonResult(subjects)
}

// extractSubjectFromArgs attempts to extract a "subject" field from raw JSON arguments.
func extractSubjectFromArgs(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var args map[string]json.RawMessage
	if err := json.Unmarshal(raw, &args); err != nil {
		return ""
	}
	subjectRaw, ok := args["subject"]
	if !ok {
		return ""
	}
	var subject string
	if err := json.Unmarshal(subjectRaw, &subject); err != nil {
		return ""
	}
	return subject
}

// extractSchemaFromArgs extracts the "schema" field from tool arguments.
func extractSchemaFromArgs(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var args map[string]json.RawMessage
	if err := json.Unmarshal(raw, &args); err != nil {
		return ""
	}
	schemaRaw, ok := args["schema"]
	if !ok {
		return ""
	}
	var schema string
	if err := json.Unmarshal(schemaRaw, &schema); err != nil {
		return ""
	}
	return schema
}

// isToolAllowed checks if a tool should be registered based on the tool policy
// and read-only mode. Returns false if the tool should be hidden from clients.
func (s *Server) isToolAllowed(name string, readOnly bool) bool {
	// Permission scopes take precedence when configured.
	if s.resolvedScopes != nil {
		if !isScopeAllowed(name, s.resolvedScopes) {
			if s.metrics != nil {
				scope := toolPermissionScope[name]
				s.metrics.RecordMCPPermissionDenied(name, scope)
			}
			return false
		}
		return true
	}

	// Read-only mode: skip non-read-only tools
	if s.config.ReadOnly && !readOnly {
		return false
	}

	policy := s.config.ToolPolicy
	if policy == "" {
		policy = "allow_all"
	}

	switch policy {
	case "deny_list":
		for _, denied := range s.config.DeniedTools {
			if denied == name {
				return false
			}
		}
		return true
	case "allow_list":
		for _, allowed := range s.config.AllowedTools {
			if allowed == name {
				return true
			}
		}
		return false
	default: // "allow_all"
		return true
	}
}

// addToolIfAllowed registers a tool only if it passes the tool policy and
// read-only mode checks. Denied tools are invisible to MCP clients.
// The handler should already be wrapped with instrumentedHandler.
func addToolIfAllowed[T any](s *Server, tool *gomcp.Tool, handler gomcp.ToolHandlerFor[T, any]) {
	readOnly := tool.Annotations != nil && tool.Annotations.ReadOnlyHint
	if !s.isToolAllowed(tool.Name, readOnly) {
		return
	}
	gomcp.AddTool(s.mcpServer, tool, handler)
}

// instrumentedHandler wraps an MCP tool handler with metrics, audit logging,
// and structured logging.
func instrumentedHandler[T any](s *Server, name string, handler gomcp.ToolHandlerFor[T, any]) gomcp.ToolHandlerFor[T, any] {
	return func(ctx context.Context, req *gomcp.CallToolRequest, input T) (*gomcp.CallToolResult, any, error) {
		start := time.Now()

		if s.metrics != nil {
			s.metrics.MCPToolCallsActive.Inc()
			defer s.metrics.MCPToolCallsActive.Dec()
		}

		result, output, err := handler(ctx, req, input)

		duration := time.Since(start)
		status := "success"
		if err != nil || (result != nil && result.IsError) {
			status = "error"
		}

		// Record Prometheus metrics.
		if s.metrics != nil {
			s.metrics.RecordMCPToolCall(name, status, duration)
			principal := s.mcpPrincipal()
			s.metrics.RecordPrincipalMCPCall(principal, name, status)
		}

		// Structured log output.
		s.logger.Info("mcp_tool_call",
			slog.String("tool", name),
			slog.String("status", status),
			slog.Duration("duration", duration),
		)

		// Extract schema body once for both logging and audit.
		var schemaBody string
		if s.config.LogSchemas && len(req.Params.Arguments) > 0 {
			schemaBody = extractSchemaFromArgs(req.Params.Arguments)
		}

		// Conditionally log schema body at Debug level.
		if schemaBody != "" {
			s.logger.Debug("mcp_tool_schema_body",
				slog.String("tool", name),
				slog.String("schema", schemaBody),
			)
		}

		// Audit trail.
		if s.auditLogger != nil {
			var auditErr error
			if err != nil {
				auditErr = err
			} else if result != nil && result.IsError {
				auditErr = fmt.Errorf("tool returned error")
			}
			eventType := auth.AuditEventMCPToolCall
			if auditErr != nil {
				eventType = auth.AuditEventMCPToolError
			}
			subject := extractSubjectFromArgs(req.Params.Arguments)
			var auditMeta map[string]string
			if schemaBody != "" {
				truncated := schemaBody
				if len(truncated) > 1000 {
					truncated = truncated[:1000] + "..."
				}
				auditMeta = map[string]string{"schema": truncated}
			}
			actorID, actorType, authMethod := s.mcpActor()
			s.auditLogger.LogMCPEvent(eventType, actorID, actorType, authMethod, name, status, duration, auditErr, subject, auditMeta)
		}

		return result, output, err
	}
}
