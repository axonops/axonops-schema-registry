package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
)

func (s *Server) registerConfigTools() {
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_config",
		Description: "Get the compatibility configuration for a subject or the global default. Omit subject for global config.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleGetConfig)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "set_config",
		Description: "Set the compatibility level for a subject or globally. Valid levels: NONE, BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, FULL_TRANSITIVE",
	}, s.handleSetConfig)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "delete_config",
		Description: "Delete the compatibility configuration for a subject (reverts to global default) or delete the global config",
	}, s.handleDeleteConfig)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_mode",
		Description: "Get the registry mode for a subject or the global default. Modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleGetMode)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "set_mode",
		Description: "Set the registry mode for a subject or globally. Valid modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT",
	}, s.handleSetMode)

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "delete_mode",
		Description: "Delete the mode for a subject (reverts to global default) or delete the global mode",
	}, s.handleDeleteMode)
}

// --- Handler input types and implementations ---

type getConfigInput struct {
	Subject string `json:"subject,omitempty"`
}

func (s *Server) handleGetConfig(ctx context.Context, _ *gomcp.CallToolRequest, input getConfigInput) (*gomcp.CallToolResult, any, error) {
	record, err := s.registry.GetConfigFull(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(record)
}

type setConfigInput struct {
	Subject            string `json:"subject,omitempty"`
	CompatibilityLevel string `json:"compatibility_level"`
	Normalize          *bool  `json:"normalize,omitempty"`
}

func (s *Server) handleSetConfig(ctx context.Context, _ *gomcp.CallToolRequest, input setConfigInput) (*gomcp.CallToolResult, any, error) {
	err := s.registry.SetConfig(ctx, registrycontext.DefaultContext, input.Subject, input.CompatibilityLevel, input.Normalize)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"compatibilityLevel": input.CompatibilityLevel})
}

type deleteConfigInput struct {
	Subject string `json:"subject,omitempty"`
}

func (s *Server) handleDeleteConfig(ctx context.Context, _ *gomcp.CallToolRequest, input deleteConfigInput) (*gomcp.CallToolResult, any, error) {
	prev, err := s.registry.DeleteConfig(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"compatibilityLevel": prev})
}

type getModeInput struct {
	Subject string `json:"subject,omitempty"`
}

func (s *Server) handleGetMode(ctx context.Context, _ *gomcp.CallToolRequest, input getModeInput) (*gomcp.CallToolResult, any, error) {
	mode, err := s.registry.GetMode(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"mode": mode})
}

type setModeInput struct {
	Subject string `json:"subject,omitempty"`
	Mode    string `json:"mode"`
	Force   bool   `json:"force,omitempty"`
}

func (s *Server) handleSetMode(ctx context.Context, _ *gomcp.CallToolRequest, input setModeInput) (*gomcp.CallToolResult, any, error) {
	err := s.registry.SetMode(ctx, registrycontext.DefaultContext, input.Subject, input.Mode, input.Force)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"mode": input.Mode})
}

type deleteModeInput struct {
	Subject string `json:"subject,omitempty"`
}

func (s *Server) handleDeleteMode(ctx context.Context, _ *gomcp.CallToolRequest, input deleteModeInput) (*gomcp.CallToolResult, any, error) {
	prev, err := s.registry.DeleteMode(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"mode": prev})
}
