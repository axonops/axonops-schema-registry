package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// errorResult returns an MCP error result.
func errorResult(err error) *gomcp.CallToolResult {
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{&gomcp.TextContent{Text: fmt.Sprintf("error: %v", err)}},
		IsError: true,
	}
}

// jsonResult returns an MCP result with JSON-serialized content.
func jsonResult(v any) (*gomcp.CallToolResult, any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{&gomcp.TextContent{Text: string(data)}},
	}, nil, nil
}

// textResult returns an MCP result with plain text content.
func textResult(text string) (*gomcp.CallToolResult, any, error) {
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{&gomcp.TextContent{Text: text}},
	}, nil, nil
}

func (s *Server) registerSchemaReadTools() {
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_schema_by_id",
		Description: "Get a schema by its global ID, returning the full schema record including subject, version, type, and schema content",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_schema_by_id", s.handleGetSchemaByID))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_raw_schema_by_id",
		Description: "Get the raw schema string by its global ID, without any metadata",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_raw_schema_by_id", s.handleGetRawSchemaByID))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_schema_version",
		Description: "Get a schema by subject name and version number",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_schema_version", s.handleGetSchemaVersion))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_raw_schema_version",
		Description: "Get the raw schema string by subject name and version number, without any metadata",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_raw_schema_version", s.handleGetRawSchemaVersion))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_latest_schema",
		Description: "Get the latest (most recent non-deleted) schema version for a subject",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_latest_schema", s.handleGetLatestSchema))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "list_versions",
		Description: "List all version numbers registered for a subject",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_versions", s.handleListVersions))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_subjects_for_schema",
		Description: "Get all subjects that use a specific schema ID",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_subjects_for_schema", s.handleGetSubjectsForSchema))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_versions_for_schema",
		Description: "Get all subject-version pairs that use a specific schema ID",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_versions_for_schema", s.handleGetVersionsForSchema))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_referenced_by",
		Description: "Get schemas that reference a specific subject-version pair",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_referenced_by", s.handleGetReferencedBy))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "lookup_schema",
		Description: "Check if a schema is already registered under a subject. Returns the existing schema record if found.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "lookup_schema", s.handleLookupSchema))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_schema_types",
		Description: "Get the list of supported schema types (e.g. AVRO, PROTOBUF, JSON)",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_schema_types", s.handleGetSchemaTypes))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "list_schemas",
		Description: "List schemas with optional filtering by subject prefix, deleted status, and pagination",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_schemas", s.handleListSchemas))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_max_schema_id",
		Description: "Get the highest schema ID currently assigned in the registry",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_max_schema_id", s.handleGetMaxSchemaID))
}

// --- Handler input types and implementations ---

type getSchemaByIDInput struct {
	ID int64 `json:"id"`
}

func (s *Server) handleGetSchemaByID(ctx context.Context, _ *gomcp.CallToolRequest, input getSchemaByIDInput) (*gomcp.CallToolResult, any, error) {
	record, err := s.registry.GetSchemaByID(ctx, registrycontext.DefaultContext, input.ID)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(record)
}

type getRawSchemaByIDInput struct {
	ID int64 `json:"id"`
}

func (s *Server) handleGetRawSchemaByID(ctx context.Context, _ *gomcp.CallToolRequest, input getRawSchemaByIDInput) (*gomcp.CallToolResult, any, error) {
	raw, err := s.registry.GetRawSchemaByID(ctx, registrycontext.DefaultContext, input.ID)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return textResult(raw)
}

type getSchemaVersionInput struct {
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

func (s *Server) handleGetSchemaVersion(ctx context.Context, _ *gomcp.CallToolRequest, input getSchemaVersionInput) (*gomcp.CallToolResult, any, error) {
	record, err := s.registry.GetSchemaBySubjectVersion(ctx, registrycontext.DefaultContext, input.Subject, input.Version)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(record)
}

type getRawSchemaVersionInput struct {
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

func (s *Server) handleGetRawSchemaVersion(ctx context.Context, _ *gomcp.CallToolRequest, input getRawSchemaVersionInput) (*gomcp.CallToolResult, any, error) {
	raw, err := s.registry.GetRawSchemaBySubjectVersion(ctx, registrycontext.DefaultContext, input.Subject, input.Version)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return textResult(raw)
}

type getLatestSchemaInput struct {
	Subject string `json:"subject"`
}

func (s *Server) handleGetLatestSchema(ctx context.Context, _ *gomcp.CallToolRequest, input getLatestSchemaInput) (*gomcp.CallToolResult, any, error) {
	record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(record)
}

type listVersionsInput struct {
	Subject string `json:"subject"`
	Deleted bool   `json:"deleted,omitempty"`
}

func (s *Server) handleListVersions(ctx context.Context, _ *gomcp.CallToolRequest, input listVersionsInput) (*gomcp.CallToolResult, any, error) {
	versions, err := s.registry.GetVersions(ctx, registrycontext.DefaultContext, input.Subject, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(versions)
}

type getSubjectsForSchemaInput struct {
	ID      int64 `json:"id"`
	Deleted bool  `json:"deleted,omitempty"`
}

func (s *Server) handleGetSubjectsForSchema(ctx context.Context, _ *gomcp.CallToolRequest, input getSubjectsForSchemaInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.GetSubjectsBySchemaID(ctx, registrycontext.DefaultContext, input.ID, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if subjects == nil {
		subjects = []string{}
	}
	return jsonResult(subjects)
}

type getVersionsForSchemaInput struct {
	ID      int64 `json:"id"`
	Deleted bool  `json:"deleted,omitempty"`
}

func (s *Server) handleGetVersionsForSchema(ctx context.Context, _ *gomcp.CallToolRequest, input getVersionsForSchemaInput) (*gomcp.CallToolResult, any, error) {
	versions, err := s.registry.GetVersionsBySchemaID(ctx, registrycontext.DefaultContext, input.ID, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if versions == nil {
		versions = []storage.SubjectVersion{}
	}
	return jsonResult(versions)
}

type getReferencedByInput struct {
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

func (s *Server) handleGetReferencedBy(ctx context.Context, _ *gomcp.CallToolRequest, input getReferencedByInput) (*gomcp.CallToolResult, any, error) {
	refs, err := s.registry.GetReferencedBy(ctx, registrycontext.DefaultContext, input.Subject, input.Version)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if refs == nil {
		refs = []storage.SubjectVersion{}
	}
	return jsonResult(refs)
}

type lookupSchemaInput struct {
	Subject    string `json:"subject"`
	Schema     string `json:"schema"`
	SchemaType string `json:"schema_type,omitempty"`
	Deleted    bool   `json:"deleted,omitempty"`
}

func (s *Server) handleLookupSchema(ctx context.Context, _ *gomcp.CallToolRequest, input lookupSchemaInput) (*gomcp.CallToolResult, any, error) {
	schemaType := storage.SchemaType(input.SchemaType)
	record, err := s.registry.LookupSchema(ctx, registrycontext.DefaultContext, input.Subject, input.Schema, schemaType, nil, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(record)
}

type getSchemaTypesInput struct{}

func (s *Server) handleGetSchemaTypes(_ context.Context, _ *gomcp.CallToolRequest, _ getSchemaTypesInput) (*gomcp.CallToolResult, any, error) {
	types := s.registry.GetSchemaTypes()
	return jsonResult(types)
}

type listSchemasInput struct {
	SubjectPrefix string `json:"subject_prefix,omitempty"`
	Deleted       bool   `json:"deleted,omitempty"`
	LatestOnly    bool   `json:"latest_only,omitempty"`
	Offset        int    `json:"offset,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

func (s *Server) handleListSchemas(ctx context.Context, _ *gomcp.CallToolRequest, input listSchemasInput) (*gomcp.CallToolResult, any, error) {
	params := &storage.ListSchemasParams{
		SubjectPrefix: input.SubjectPrefix,
		Deleted:       input.Deleted,
		LatestOnly:    input.LatestOnly,
		Offset:        input.Offset,
		Limit:         input.Limit,
	}
	schemas, err := s.registry.ListSchemas(ctx, registrycontext.DefaultContext, params)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if schemas == nil {
		schemas = []*storage.SchemaRecord{}
	}
	return jsonResult(schemas)
}

type getMaxSchemaIDInput struct{}

func (s *Server) handleGetMaxSchemaID(ctx context.Context, _ *gomcp.CallToolRequest, _ getMaxSchemaIDInput) (*gomcp.CallToolResult, any, error) {
	maxID, err := s.registry.GetMaxSchemaID(ctx, registrycontext.DefaultContext)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]int64{"max_id": maxID})
}
