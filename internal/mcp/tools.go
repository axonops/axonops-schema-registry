package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
)

func (s *Server) registerTools() {
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "health_check",
		Description: "Check if the schema registry is healthy and responding",
		Annotations: &gomcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, s.handleHealthCheck)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_server_info",
		Description: "Get schema registry server information including version and supported schema types",
		Annotations: &gomcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, s.handleGetServerInfo)

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
	}, s.handleListSubjects)
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
