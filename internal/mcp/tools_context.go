package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerContextTools() {
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "list_contexts",
		Description: "List all tenant contexts in the schema registry. Each context is an isolated namespace for subjects and schemas.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_contexts", s.handleListContexts))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "import_schemas",
		Description: "Bulk import schemas with preserved IDs (for Confluent migration). Registry mode MUST be set to IMPORT first.",
	}, instrumentedHandler(s, "import_schemas", s.handleImportSchemas))
}

type listContextsInput struct{}

func (s *Server) handleListContexts(ctx context.Context, _ *gomcp.CallToolRequest, _ listContextsInput) (*gomcp.CallToolResult, any, error) {
	contexts, err := s.registry.ListContexts(ctx)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if contexts == nil {
		contexts = []string{}
	}
	return jsonResult(contexts)
}

type importSchemaItem struct {
	ID         int64               `json:"id"`
	Subject    string              `json:"subject"`
	Version    int                 `json:"version"`
	SchemaType string              `json:"schema_type,omitempty"`
	Schema     string              `json:"schema"`
	References []storage.Reference `json:"references,omitempty"`
}

type importSchemasInput struct {
	Schemas []importSchemaItem `json:"schemas"`
}

func (s *Server) handleImportSchemas(ctx context.Context, _ *gomcp.CallToolRequest, input importSchemasInput) (*gomcp.CallToolResult, any, error) {
	reqs := make([]registry.ImportSchemaRequest, len(input.Schemas))
	for i, item := range input.Schemas {
		reqs[i] = registry.ImportSchemaRequest{
			ID:         item.ID,
			Subject:    item.Subject,
			Version:    item.Version,
			SchemaType: storage.SchemaType(item.SchemaType),
			Schema:     item.Schema,
			References: item.References,
		}
	}
	result, err := s.registry.ImportSchemas(ctx, registrycontext.DefaultContext, reqs)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(result)
}
