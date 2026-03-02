package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerSchemaWriteTools() {
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "register_schema",
		Description: "Register a new schema version for a subject. If the same schema already exists, returns the existing record.",
	}, instrumentedHandler(s, "register_schema", s.handleRegisterSchema))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "delete_subject",
		Description: "Delete a subject and all its schema versions. Soft-deletes by default; use permanent=true for hard delete.",
	}, instrumentedHandler(s, "delete_subject", s.handleDeleteSubject))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "delete_version",
		Description: "Delete a specific schema version. Soft-deletes by default; use permanent=true for hard delete (requires prior soft-delete).",
	}, instrumentedHandler(s, "delete_version", s.handleDeleteVersion))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "check_compatibility",
		Description: "Check if a schema is compatible with existing versions of a subject according to the configured compatibility level",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "check_compatibility", s.handleCheckCompatibility))
}

// --- Handler input types and implementations ---

type registerSchemaInput struct {
	Subject    string             `json:"subject"`
	Schema     string             `json:"schema"`
	SchemaType string             `json:"schema_type,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
	Normalize  bool               `json:"normalize,omitempty"`
	Metadata   *storage.Metadata  `json:"metadata,omitempty"`
	RuleSet    *storage.RuleSet   `json:"rule_set,omitempty"`
}

func (s *Server) handleRegisterSchema(ctx context.Context, _ *gomcp.CallToolRequest, input registerSchemaInput) (*gomcp.CallToolResult, any, error) {
	schemaType := storage.SchemaType(input.SchemaType)
	opts := registry.RegisterOpts{
		Normalize: input.Normalize,
		Metadata:  input.Metadata,
		RuleSet:   input.RuleSet,
	}
	record, err := s.registry.RegisterSchema(ctx, registrycontext.DefaultContext, input.Subject, input.Schema, schemaType, input.References, opts)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(record)
}

type deleteSubjectInput struct {
	Subject   string `json:"subject"`
	Permanent bool   `json:"permanent,omitempty"`
}

func (s *Server) handleDeleteSubject(ctx context.Context, _ *gomcp.CallToolRequest, input deleteSubjectInput) (*gomcp.CallToolResult, any, error) {
	versions, err := s.registry.DeleteSubject(ctx, registrycontext.DefaultContext, input.Subject, input.Permanent)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(versions)
}

type deleteVersionInput struct {
	Subject   string `json:"subject"`
	Version   int    `json:"version"`
	Permanent bool   `json:"permanent,omitempty"`
}

func (s *Server) handleDeleteVersion(ctx context.Context, _ *gomcp.CallToolRequest, input deleteVersionInput) (*gomcp.CallToolResult, any, error) {
	ver, err := s.registry.DeleteVersion(ctx, registrycontext.DefaultContext, input.Subject, input.Version, input.Permanent)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]int{"version": ver})
}

type checkCompatibilityInput struct {
	Subject    string             `json:"subject"`
	Schema     string             `json:"schema"`
	SchemaType string             `json:"schema_type,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
	Version    string             `json:"version,omitempty"`
}

func (s *Server) handleCheckCompatibility(ctx context.Context, _ *gomcp.CallToolRequest, input checkCompatibilityInput) (*gomcp.CallToolResult, any, error) {
	version := input.Version
	if version == "" {
		version = "latest"
	}
	schemaType := storage.SchemaType(input.SchemaType)
	result, err := s.registry.CheckCompatibility(ctx, registrycontext.DefaultContext, input.Subject, input.Schema, schemaType, input.References, version)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(result)
}
