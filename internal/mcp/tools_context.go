package mcp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerContextTools() {
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "list_contexts",
		Description: "List all tenant contexts in the schema registry. Each context is an isolated namespace for subjects and schemas.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_contexts", s.handleListContexts))

	addToolIfAllowed(s, &gomcp.Tool{
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
	Schemas      []importSchemaItem `json:"schemas"`
	DryRun       bool               `json:"dry_run,omitempty"`
	ConfirmToken string             `json:"confirm_token,omitempty"`
	Context      string             `json:"context,omitempty"`
}

func (s *Server) handleImportSchemas(ctx context.Context, _ *gomcp.CallToolRequest, input importSchemasInput) (*gomcp.CallToolResult, any, error) {
	if result := s.confirmationCheck("import_schemas", input.DryRun, input.ConfirmToken,
		map[string]any{"schema_count": len(input.Schemas), "schemas_hash": hashImportSchemas(input.Schemas)},
		map[string]any{"action": "import_schemas", "schema_count": len(input.Schemas)},
	); result != nil {
		return result, nil, nil
	}
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
	result, err := s.registry.ImportSchemas(ctx, resolveContext(input.Context), reqs)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(result)
}

// hashImportSchemas produces a deterministic hash of the import payload content
// so the confirmation token is scoped to specific schemas, not just their count.
func hashImportSchemas(schemas []importSchemaItem) string {
	parts := make([]string, len(schemas))
	for i, s := range schemas {
		parts[i] = fmt.Sprintf("%d:%s:%d", s.ID, s.Subject, s.Version)
	}
	sort.Strings(parts)
	h := sha256.Sum256([]byte(fmt.Sprintf("%v", parts)))
	return fmt.Sprintf("%x", h[:8]) // short hash for readability
}
