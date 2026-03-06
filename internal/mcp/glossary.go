package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/mcp/content"
)

func (s *Server) registerGlossaryResources() {
	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/core-concepts",
		Name:        "glossary-core-concepts",
		Description: "Schema registry fundamentals: what a schema registry is, subjects, versions, IDs, deduplication, modes, naming strategies, and the serialization flow",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryCoreConceptsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/compatibility",
		Name:        "glossary-compatibility",
		Description: "All 7 compatibility modes, Avro type promotions, Protobuf wire types, JSON Schema constraints, transitive semantics, and configuration resolution",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryCompatibilityResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/data-contracts",
		Name:        "glossary-data-contracts",
		Description: "Data contracts: metadata properties, tags, sensitive fields, rulesets (domain/migration/encoding), rule structure, 3-layer merge, and optimistic concurrency",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryDataContractsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/encryption",
		Name:        "glossary-encryption",
		Description: "Client-side field level encryption (CSFLE): envelope encryption, KEK/DEK model, KMS providers, algorithms, key rotation, and rewrapping",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryEncryptionResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/contexts",
		Name:        "glossary-contexts",
		Description: "Multi-tenancy via contexts: default context, __GLOBAL, qualified subjects, URL routing, isolation guarantees, and 4-tier config/mode inheritance",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryContextsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/exporters",
		Name:        "glossary-exporters",
		Description: "Schema linking via exporters: exporter model, lifecycle states (STARTING/RUNNING/PAUSED/ERROR), context types (AUTO/CUSTOM/NONE), and configuration",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryExportersResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/schema-types",
		Name:        "glossary-schema-types",
		Description: "Deep reference for Avro (types, logical types, aliases, canonicalization), Protobuf (proto3, well-known types, wire types), and JSON Schema (drafts, keywords, combinators)",
		MIMEType:    "text/markdown",
	}, s.handleGlossarySchemaTypesResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/design-patterns",
		Name:        "glossary-design-patterns",
		Description: "Common schema design patterns: event envelope, entity lifecycle, snapshot vs delta, fat vs thin events, shared types, three-phase rename, and CI/CD integration",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryDesignPatternsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/best-practices",
		Name:        "glossary-best-practices",
		Description: "Actionable best practices for Avro, Protobuf, and JSON Schema: field naming, nullability, evolution readiness, common mistakes, and per-format guidance",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryBestPracticesResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/migration",
		Name:        "glossary-migration",
		Description: "Confluent migration: step-by-step procedure, IMPORT mode, ID preservation, the import API, verification, and rollback",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryMigrationResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/mcp-configuration",
		Name:        "glossary-mcp-configuration",
		Description: "MCP server configuration: all config fields, env var overrides, read-only mode, tool policy, permission scopes, presets, two-phase confirmations, and origin validation",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryMCPConfigurationResource)
}

// --- Glossary resource handlers ---
// Each handler reads its content from an embedded markdown file via content.GlossaryFS.

func (s *Server) handleGlossaryCoreConceptsResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/core-concepts.md", req.Params.URI)
}

func (s *Server) handleGlossaryCompatibilityResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/compatibility.md", req.Params.URI)
}

func (s *Server) handleGlossaryDataContractsResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/data-contracts.md", req.Params.URI)
}

func (s *Server) handleGlossaryEncryptionResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/encryption.md", req.Params.URI)
}

func (s *Server) handleGlossaryContextsResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/contexts.md", req.Params.URI)
}

func (s *Server) handleGlossaryExportersResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/exporters.md", req.Params.URI)
}

func (s *Server) handleGlossarySchemaTypesResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/schema-types.md", req.Params.URI)
}

func (s *Server) handleGlossaryDesignPatternsResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/design-patterns.md", req.Params.URI)
}

func (s *Server) handleGlossaryBestPracticesResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/best-practices.md", req.Params.URI)
}

func (s *Server) handleGlossaryMigrationResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/migration.md", req.Params.URI)
}

func (s *Server) handleGlossaryMCPConfigurationResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdownFromFS(content.GlossaryFS, "glossary/mcp-configuration.md", req.Params.URI)
}
