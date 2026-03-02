package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/auth"
	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
)

func (s *Server) registerTools() {
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "health_check",
		Description: "Check if the schema registry is healthy and responding",
		Annotations: &gomcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, instrumentedHandler(s, "health_check", s.handleHealthCheck))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
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

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
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
	info := map[string]any{
		"version":      s.version,
		"schema_types": []string{"AVRO", "PROTOBUF", "JSON"},
	}
	data, _ := json.Marshal(info)
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{
			&gomcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}

type listSubjectsInput struct {
	Deleted bool   `json:"deleted,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
}

func (s *Server) handleListSubjects(ctx context.Context, _ *gomcp.CallToolRequest, input listSubjectsInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, input.Deleted)
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

	if subjects == nil {
		subjects = []string{}
	}
	data, _ := json.Marshal(subjects)
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{
			&gomcp.TextContent{Text: string(data)},
		},
	}, nil, nil
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
			// TODO: Per-principal MCP metrics — requires per-session auth
			// identity extraction. Will be addressed during integrated
			// MCP testing via docker-compose and BDD.
		}

		// Structured log output.
		s.logger.Info("mcp_tool_call",
			slog.String("tool", name),
			slog.String("status", status),
			slog.Duration("duration", duration),
		)

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
			s.auditLogger.LogMCPEvent(eventType, name, status, duration, auditErr, nil)
		}

		return result, output, err
	}
}
