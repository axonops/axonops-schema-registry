package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerExporterTools() {
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "list_exporters",
		Description: "List all exporter names. Exporters replicate schemas to a destination schema registry (Schema Linking).",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleListExporters)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "create_exporter",
		Description: "Create a new schema exporter for cross-cluster schema replication. Context types: AUTO, CUSTOM, NONE.",
	}, s.handleCreateExporter)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_exporter",
		Description: "Get an exporter's configuration by name.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleGetExporter)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "update_exporter",
		Description: "Update an existing exporter's settings (context type, subjects, rename format, config).",
	}, s.handleUpdateExporter)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "delete_exporter",
		Description: "Delete an exporter by name.",
	}, s.handleDeleteExporter)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "pause_exporter",
		Description: "Pause a running exporter. The exporter retains its current offset and can be resumed later.",
	}, s.handlePauseExporter)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "resume_exporter",
		Description: "Resume a paused exporter. The exporter continues from its last offset.",
	}, s.handleResumeExporter)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "reset_exporter",
		Description: "Reset an exporter's offset back to zero, causing it to re-export all schemas.",
	}, s.handleResetExporter)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_exporter_status",
		Description: "Get the current status of an exporter (state, offset, error trace).",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleGetExporterStatus)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_exporter_config",
		Description: "Get the destination configuration of an exporter.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleGetExporterConfig)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "update_exporter_config",
		Description: "Update the destination configuration of an exporter.",
	}, s.handleUpdateExporterConfig)
}

// --- Exporter handlers ---

type listExportersInput struct{}

func (s *Server) handleListExporters(ctx context.Context, _ *gomcp.CallToolRequest, _ listExportersInput) (*gomcp.CallToolResult, any, error) {
	names, err := s.registry.ListExporters(ctx)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if names == nil {
		names = []string{}
	}
	return jsonResult(names)
}

type createExporterInput struct {
	Name                string            `json:"name"`
	ContextType         string            `json:"context_type,omitempty"`
	Context             string            `json:"context,omitempty"`
	Subjects            []string          `json:"subjects,omitempty"`
	SubjectRenameFormat string            `json:"subject_rename_format,omitempty"`
	Config              map[string]string `json:"config,omitempty"`
}

func (s *Server) handleCreateExporter(ctx context.Context, _ *gomcp.CallToolRequest, input createExporterInput) (*gomcp.CallToolResult, any, error) {
	exporter := &storage.ExporterRecord{
		Name:                input.Name,
		ContextType:         input.ContextType,
		Context:             input.Context,
		Subjects:            input.Subjects,
		SubjectRenameFormat: input.SubjectRenameFormat,
		Config:              input.Config,
	}
	if err := s.registry.CreateExporter(ctx, exporter); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"name": input.Name})
}

type getExporterInput struct {
	Name string `json:"name"`
}

func (s *Server) handleGetExporter(ctx context.Context, _ *gomcp.CallToolRequest, input getExporterInput) (*gomcp.CallToolResult, any, error) {
	exporter, err := s.registry.GetExporter(ctx, input.Name)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(exporter)
}

type updateExporterInput struct {
	Name                string            `json:"name"`
	ContextType         string            `json:"context_type,omitempty"`
	Context             string            `json:"context,omitempty"`
	Subjects            []string          `json:"subjects,omitempty"`
	SubjectRenameFormat string            `json:"subject_rename_format,omitempty"`
	Config              map[string]string `json:"config,omitempty"`
}

func (s *Server) handleUpdateExporter(ctx context.Context, _ *gomcp.CallToolRequest, input updateExporterInput) (*gomcp.CallToolResult, any, error) {
	exporter := &storage.ExporterRecord{
		Name:                input.Name,
		ContextType:         input.ContextType,
		Context:             input.Context,
		Subjects:            input.Subjects,
		SubjectRenameFormat: input.SubjectRenameFormat,
		Config:              input.Config,
	}
	if err := s.registry.UpdateExporter(ctx, exporter); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"name": input.Name})
}

type deleteExporterInput struct {
	Name string `json:"name"`
}

func (s *Server) handleDeleteExporter(ctx context.Context, _ *gomcp.CallToolRequest, input deleteExporterInput) (*gomcp.CallToolResult, any, error) {
	if err := s.registry.DeleteExporter(ctx, input.Name); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"deleted": true})
}

type pauseExporterInput struct {
	Name string `json:"name"`
}

func (s *Server) handlePauseExporter(ctx context.Context, _ *gomcp.CallToolRequest, input pauseExporterInput) (*gomcp.CallToolResult, any, error) {
	if err := s.registry.PauseExporter(ctx, input.Name); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"name": input.Name, "state": "PAUSED"})
}

type resumeExporterInput struct {
	Name string `json:"name"`
}

func (s *Server) handleResumeExporter(ctx context.Context, _ *gomcp.CallToolRequest, input resumeExporterInput) (*gomcp.CallToolResult, any, error) {
	if err := s.registry.ResumeExporter(ctx, input.Name); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"name": input.Name, "state": "RUNNING"})
}

type resetExporterInput struct {
	Name string `json:"name"`
}

func (s *Server) handleResetExporter(ctx context.Context, _ *gomcp.CallToolRequest, input resetExporterInput) (*gomcp.CallToolResult, any, error) {
	if err := s.registry.ResetExporter(ctx, input.Name); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"name": input.Name, "state": "reset"})
}

type getExporterStatusInput struct {
	Name string `json:"name"`
}

func (s *Server) handleGetExporterStatus(ctx context.Context, _ *gomcp.CallToolRequest, input getExporterStatusInput) (*gomcp.CallToolResult, any, error) {
	status, err := s.registry.GetExporterStatus(ctx, input.Name)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(status)
}

type getExporterConfigInput struct {
	Name string `json:"name"`
}

func (s *Server) handleGetExporterConfig(ctx context.Context, _ *gomcp.CallToolRequest, input getExporterConfigInput) (*gomcp.CallToolResult, any, error) {
	config, err := s.registry.GetExporterConfig(ctx, input.Name)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if config == nil {
		config = map[string]string{}
	}
	return jsonResult(config)
}

type updateExporterConfigInput struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

func (s *Server) handleUpdateExporterConfig(ctx context.Context, _ *gomcp.CallToolRequest, input updateExporterConfigInput) (*gomcp.CallToolResult, any, error) {
	if err := s.registry.UpdateExporterConfig(ctx, input.Name, input.Config); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"name": input.Name})
}
