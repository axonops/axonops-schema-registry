package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerConfigTools() {
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_config",
		Description: "Get the compatibility configuration for a subject or the global default. Omit subject for global config.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_config", s.handleGetConfig))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "set_config",
		Description: "Set the compatibility level for a subject or globally. Valid levels: NONE, BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, FULL_TRANSITIVE",
	}, instrumentedHandler(s, "set_config", s.handleSetConfig))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "delete_config",
		Description: "Delete the compatibility configuration for a subject (reverts to global default) or delete the global config",
	}, instrumentedHandler(s, "delete_config", s.handleDeleteConfig))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_mode",
		Description: "Get the registry mode for a subject or the global default. Modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_mode", s.handleGetMode))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "set_mode",
		Description: "Set the registry mode for a subject or globally. Valid modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT",
	}, instrumentedHandler(s, "set_mode", s.handleSetMode))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "delete_mode",
		Description: "Delete the mode for a subject (reverts to global default) or delete the global mode",
	}, instrumentedHandler(s, "delete_mode", s.handleDeleteMode))
}

// --- Handler input types and implementations ---

type getConfigInput struct {
	Subject string `json:"subject,omitempty"`
	Context string `json:"context,omitempty"`
}

func (s *Server) handleGetConfig(ctx context.Context, _ *gomcp.CallToolRequest, input getConfigInput) (*gomcp.CallToolResult, any, error) {
	record, err := s.registry.GetConfigFull(ctx, resolveContext(input.Context), input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(record)
}

type setConfigInput struct {
	Subject            string `json:"subject,omitempty"`
	CompatibilityLevel string `json:"compatibility_level"`
	Normalize          *bool  `json:"normalize,omitempty"`
	Context            string `json:"context,omitempty"`
}

func (s *Server) handleSetConfig(ctx context.Context, _ *gomcp.CallToolRequest, input setConfigInput) (*gomcp.CallToolResult, any, error) {
	err := s.registry.SetConfig(ctx, resolveContext(input.Context), input.Subject, input.CompatibilityLevel, input.Normalize)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"compatibilityLevel": input.CompatibilityLevel})
}

type deleteConfigInput struct {
	Subject      string `json:"subject,omitempty"`
	DryRun       bool   `json:"dry_run,omitempty"`
	ConfirmToken string `json:"confirm_token,omitempty"`
	Context      string `json:"context,omitempty"`
}

func (s *Server) handleDeleteConfig(ctx context.Context, _ *gomcp.CallToolRequest, input deleteConfigInput) (*gomcp.CallToolResult, any, error) {
	if result := s.confirmationCheck("delete_config", input.DryRun, input.ConfirmToken,
		map[string]any{"subject": input.Subject},
		map[string]any{"action": "delete_config", "subject": input.Subject},
	); result != nil {
		return result, nil, nil
	}
	prev, err := s.registry.DeleteConfig(ctx, resolveContext(input.Context), input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"compatibilityLevel": prev})
}

type getModeInput struct {
	Subject string `json:"subject,omitempty"`
	Context string `json:"context,omitempty"`
}

func (s *Server) handleGetMode(ctx context.Context, _ *gomcp.CallToolRequest, input getModeInput) (*gomcp.CallToolResult, any, error) {
	mode, err := s.registry.GetMode(ctx, resolveContext(input.Context), input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"mode": mode})
}

type setModeInput struct {
	Subject      string `json:"subject,omitempty"`
	Mode         string `json:"mode"`
	Force        bool   `json:"force,omitempty"`
	DryRun       bool   `json:"dry_run,omitempty"`
	ConfirmToken string `json:"confirm_token,omitempty"`
	Context      string `json:"context,omitempty"`
}

func (s *Server) handleSetMode(ctx context.Context, _ *gomcp.CallToolRequest, input setModeInput) (*gomcp.CallToolResult, any, error) {
	if result := s.confirmationCheck("set_mode", input.DryRun, input.ConfirmToken,
		map[string]any{"subject": input.Subject, "mode": input.Mode},
		map[string]any{"action": "set_mode", "subject": input.Subject, "mode": input.Mode},
	); result != nil {
		return result, nil, nil
	}
	err := s.registry.SetMode(ctx, resolveContext(input.Context), input.Subject, input.Mode, input.Force)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"mode": input.Mode})
}

type deleteModeInput struct {
	Subject string `json:"subject,omitempty"`
	Context string `json:"context,omitempty"`
}

func (s *Server) handleDeleteMode(ctx context.Context, _ *gomcp.CallToolRequest, input deleteModeInput) (*gomcp.CallToolResult, any, error) {
	prev, err := s.registry.DeleteMode(ctx, resolveContext(input.Context), input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"mode": prev})
}
