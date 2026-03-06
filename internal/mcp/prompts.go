package mcp

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/mcp/content"
)

func (s *Server) registerPrompts() {
	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "design-schema",
		Description: "Guide for designing a new schema in the chosen format",
		Arguments: []*gomcp.PromptArgument{
			{Name: "format", Description: "Schema format: AVRO, PROTOBUF, or JSON", Required: true},
			{Name: "domain", Description: "Domain or topic for the schema (e.g. user-events, payments)", Required: false},
		},
	}, s.handleDesignSchemaPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "evolve-schema",
		Description: "Guide for safely evolving an existing schema with backward compatibility",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name of the schema to evolve", Required: true},
			{Name: "context", Description: "Registry context for multi-tenant isolation (defaults to default context)", Required: false},
		},
	}, s.handleEvolveSchemaPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "check-compatibility",
		Description: "Troubleshoot schema compatibility issues and suggest fixes",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name to check compatibility for", Required: true},
			{Name: "context", Description: "Registry context for multi-tenant isolation (defaults to default context)", Required: false},
		},
	}, s.handleCheckCompatibilityPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "migrate-schemas",
		Description: "Guide for migrating schemas between formats (e.g. Avro to Protobuf)",
		Arguments: []*gomcp.PromptArgument{
			{Name: "source_format", Description: "Source schema format (AVRO, PROTOBUF, JSON)", Required: true},
			{Name: "target_format", Description: "Target schema format (AVRO, PROTOBUF, JSON)", Required: true},
		},
	}, s.handleMigrateSchemasPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "setup-encryption",
		Description: "Guide for setting up client-side field encryption with KEK/DEK",
		Arguments: []*gomcp.PromptArgument{
			{Name: "kms_type", Description: "KMS provider type (e.g. aws-kms, azure-kms, gcp-kms, hcvault)", Required: true},
		},
	}, s.handleSetupEncryptionPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "configure-exporter",
		Description: "Guide for setting up schema linking via an exporter",
		Arguments: []*gomcp.PromptArgument{
			{Name: "exporter_type", Description: "Exporter context type: AUTO, CUSTOM, or NONE", Required: false},
		},
	}, s.handleConfigureExporterPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "review-schema-quality",
		Description: "Analyze a schema for naming conventions, nullability, documentation, and best practices",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name of the schema to review", Required: true},
			{Name: "context", Description: "Registry context for multi-tenant isolation (defaults to default context)", Required: false},
		},
	}, s.handleReviewSchemaQualityPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "plan-breaking-change",
		Description: "Plan a safe breaking schema change with migration strategy",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name where the breaking change is planned", Required: true},
			{Name: "context", Description: "Registry context for multi-tenant isolation (defaults to default context)", Required: false},
		},
	}, s.handlePlanBreakingChangePrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "debug-registration-error",
		Description: "Debug schema registration failures by error code",
		Arguments: []*gomcp.PromptArgument{
			{Name: "error_code", Description: "Error code from failed registration (e.g. 42201, 409, 40401)", Required: true},
		},
	}, s.handleDebugRegistrationErrorPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "setup-data-contracts",
		Description: "Guide for adding metadata, tags, and data quality rules to schemas",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name to add data contracts to", Required: true},
			{Name: "context", Description: "Registry context for multi-tenant isolation (defaults to default context)", Required: false},
		},
	}, s.handleSetupDataContractsPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "audit-subject-history",
		Description: "Review the version history and evolution of a schema subject",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name to audit", Required: true},
			{Name: "context", Description: "Registry context for multi-tenant isolation (defaults to default context)", Required: false},
		},
	}, s.handleAuditSubjectHistoryPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "compare-formats",
		Description: "Help choose between Avro, Protobuf, and JSON Schema for a use case",
		Arguments: []*gomcp.PromptArgument{
			{Name: "use_case", Description: "Use case description (e.g. event streaming, REST API, RPC)", Required: true},
		},
	}, s.handleCompareFormatsPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "schema-getting-started",
		Description: "Quick-start guide introducing available tools and common schema registry operations",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleGettingStartedPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "troubleshooting",
		Description: "Diagnostic guide for common schema registry issues and errors",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleTroubleshootingPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "schema-impact-analysis",
		Description: "Guided workflow for assessing the impact of a proposed schema change across dependents",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name to analyze impact for", Required: true},
			{Name: "context", Description: "Registry context for multi-tenant isolation (defaults to default context)", Required: false},
		},
	}, s.handleImpactAnalysisPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "schema-naming-conventions",
		Description: "Guide to subject naming strategies (topic_name, record_name, topic_record_name)",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleNamingConventionsPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "context-management",
		Description: "Guide for managing multi-tenant contexts and the 4-tier config/mode inheritance chain",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleContextManagementPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "glossary-lookup",
		Description: "Look up a schema registry concept and get directed to the relevant glossary resource",
		Arguments: []*gomcp.PromptArgument{
			{Name: "topic", Description: "Keyword or concept to look up (e.g. compatibility, CSFLE, contexts, avro)", Required: true},
		},
	}, s.handleGlossaryLookupPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "import-from-confluent",
		Description: "Step-by-step guide for migrating schemas from Confluent Schema Registry with ID preservation",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleImportFromConfluentPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "setup-rbac",
		Description: "Guide for configuring authentication and role-based access control (RBAC)",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleSetupRBACPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "schema-references-guide",
		Description: "Guide for cross-subject schema references with per-format name semantics (Avro, Protobuf, JSON Schema)",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleSchemaReferencesGuidePrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "full-encryption-lifecycle",
		Description: "End-to-end CSFLE workflow: KEK creation, DEK management, key rotation, rewrapping, and cleanup",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleFullEncryptionLifecyclePrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "data-rules-deep-dive",
		Description: "Comprehensive guide to data contract rules: domain, migration, and encoding rules with examples",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleDataRulesDeepDivePrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "registry-health-audit",
		Description: "Multi-step procedure for auditing registry health, configuration consistency, and schema quality",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleRegistryHealthAuditPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "schema-evolution-cookbook",
		Description: "Practical recipes for common schema evolution scenarios: add fields, rename, change types, and break compatibility safely",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleSchemaEvolutionCookbookPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "new-kafka-topic",
		Description: "End-to-end workflow for setting up key and value schemas for a new Kafka topic",
		Arguments: []*gomcp.PromptArgument{
			{Name: "topic_name", Description: "Kafka topic name (e.g., orders, user-events)", Required: true},
			{Name: "format", Description: "Schema format: AVRO (default), PROTOBUF, or JSON", Required: false},
		},
	}, s.handleNewKafkaTopicPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "debug-deserialization",
		Description: "Troubleshooting guide for consumer deserialization failures including wire format, schema ID extraction, and common causes",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleDebugDeserializationPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "deprecate-subject",
		Description: "Workflow for safely deprecating and removing a schema subject with dependency checks, locking, and cleanup",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name to deprecate", Required: true},
			{Name: "context", Description: "Registry context for multi-tenant isolation (defaults to default context)", Required: false},
		},
	}, s.handleDeprecateSubjectPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "cicd-integration",
		Description: "Guide for integrating schema validation, compatibility checking, and registration into CI/CD pipelines",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleCICDIntegrationPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "team-onboarding",
		Description: "Workflow for onboarding a new team with context creation, schema registration, RBAC setup, and naming conventions",
		Arguments: []*gomcp.PromptArgument{
			{Name: "team_name", Description: "Team name for the new context namespace", Required: true},
		},
	}, s.handleTeamOnboardingPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "governance-setup",
		Description: "Guide for setting up schema governance: naming conventions, quality gates, data contracts, RBAC, and audit",
		Arguments:   []*gomcp.PromptArgument{},
	}, s.handleGovernanceSetupPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "cross-cutting-change",
		Description: "Workflow for making a field change across multiple schemas: find affected schemas, test, and execute safely",
		Arguments: []*gomcp.PromptArgument{
			{Name: "field_name", Description: "Field name to change across schemas", Required: true},
		},
	}, s.handleCrossCuttingChangePrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "schema-review-checklist",
		Description: "Pre-registration checklist: syntax, compatibility, quality, naming, uniqueness, dependencies, and impact",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name for the schema being reviewed", Required: true},
			{Name: "context", Description: "Registry context for multi-tenant isolation (defaults to default context)", Required: false},
		},
	}, s.handleSchemaReviewChecklistPrompt)
}

// --- Helpers ---

func promptMessage(role, text string) *gomcp.PromptMessage {
	return &gomcp.PromptMessage{
		Role:    gomcp.Role(role),
		Content: &gomcp.TextContent{Text: text},
	}
}

// promptFromFS reads a prompt from an embedded file and returns it as a GetPromptResult.
func promptFromFS(fsys fs.FS, path, description string) (*gomcp.GetPromptResult, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read prompt file %s: %w", path, err)
	}
	return &gomcp.GetPromptResult{
		Description: description,
		Messages:    []*gomcp.PromptMessage{promptMessage("user", string(data))},
	}, nil
}

// promptTemplateFromFS reads a prompt template from an embedded file and replaces
// all placeholders with their corresponding values.
func promptTemplateFromFS(fsys fs.FS, path string, replacements map[string]string) (string, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", fmt.Errorf("read prompt template %s: %w", path, err)
	}
	result := string(data)
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result, nil
}

// --- Prompt handlers ---

// handleDesignSchemaPrompt selects format-specific guidance from embedded files.
func (s *Server) handleDesignSchemaPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	format := strings.ToUpper(req.Params.Arguments["format"])
	domain := req.Params.Arguments["domain"]

	if format == "" {
		return nil, fmt.Errorf("required argument 'format' is missing")
	}

	var file string
	switch format {
	case "AVRO":
		file = "prompts/design-schema-avro.md"
	case "PROTOBUF":
		file = "prompts/design-schema-protobuf.md"
	case "JSON":
		file = "prompts/design-schema-json.md"
	default:
		guidance := fmt.Sprintf("Unknown format %q. Supported formats: AVRO, PROTOBUF, JSON. Use the get_schema_types tool to verify.", format)
		return &gomcp.GetPromptResult{
			Description: fmt.Sprintf("Schema design guide for %s format", format),
			Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
		}, nil
	}

	data, err := fs.ReadFile(content.PromptsFS, file)
	if err != nil {
		return nil, fmt.Errorf("read prompt file: %w", err)
	}
	guidance := string(data)

	if domain != "" {
		guidance = fmt.Sprintf("Design a %s schema for the %q domain.\n\n%s", format, domain, guidance)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Schema design guide for %s format", format),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleEvolveSchemaPrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/evolve-schema.md", map[string]string{"{subject}": subject})
	if err != nil {
		return nil, err
	}

	registryCtx := resolveContext(req.Params.Arguments["context"])
	latest, err := s.registry.GetLatestSchema(ctx, registryCtx, subject)
	if err == nil && latest != nil {
		guidance += fmt.Sprintf("\n\nCurrent latest version: %d, schema type: %s", latest.Version, latest.SchemaType)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Schema evolution guide for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleCheckCompatibilityPrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/check-compatibility.md", map[string]string{"{subject}": subject})
	if err != nil {
		return nil, err
	}

	registryCtx := resolveContext(req.Params.Arguments["context"])
	config, err := s.registry.GetConfigFull(ctx, registryCtx, subject)
	if err == nil && config != nil {
		guidance += fmt.Sprintf("\n\nCurrent compatibility level: %v", config)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Compatibility troubleshooting for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleMigrateSchemasPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	source := strings.ToUpper(req.Params.Arguments["source_format"])
	target := strings.ToUpper(req.Params.Arguments["target_format"])

	if source == "" || target == "" {
		return nil, fmt.Errorf("required arguments 'source_format' and 'target_format' are missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/migrate-schemas.md", map[string]string{
		"{source}": source,
		"{target}": target,
	})
	if err != nil {
		return nil, err
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Migration guide from %s to %s", source, target),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleSetupEncryptionPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	kmsType := req.Params.Arguments["kms_type"]
	if kmsType == "" {
		return nil, fmt.Errorf("required argument 'kms_type' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/setup-encryption.md", map[string]string{"{kms_type}": kmsType})
	if err != nil {
		return nil, err
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Encryption setup guide for %s", kmsType),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleConfigureExporterPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	exporterType := req.Params.Arguments["exporter_type"]
	if exporterType == "" {
		exporterType = "AUTO"
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/configure-exporter.md", map[string]string{"{exporter_type}": exporterType})
	if err != nil {
		return nil, err
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Exporter configuration guide (%s context)", exporterType),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleReviewSchemaQualityPrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/review-schema-quality.md", map[string]string{"{subject}": subject})
	if err != nil {
		return nil, err
	}

	registryCtx := resolveContext(req.Params.Arguments["context"])
	latest, err := s.registry.GetLatestSchema(ctx, registryCtx, subject)
	if err == nil && latest != nil {
		guidance += fmt.Sprintf("\n\nCurrent version: %d, type: %s", latest.Version, latest.SchemaType)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Schema quality review for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handlePlanBreakingChangePrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/plan-breaking-change.md", map[string]string{"{subject}": subject})
	if err != nil {
		return nil, err
	}

	registryCtx := resolveContext(req.Params.Arguments["context"])
	latest, err := s.registry.GetLatestSchema(ctx, registryCtx, subject)
	if err == nil && latest != nil {
		guidance += fmt.Sprintf("\n\nCurrent version: %d, type: %s", latest.Version, latest.SchemaType)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Breaking change plan for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleDebugRegistrationErrorPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	errorCode := req.Params.Arguments["error_code"]
	if errorCode == "" {
		return nil, fmt.Errorf("required argument 'error_code' is missing")
	}

	var guidance string
	switch errorCode {
	case "42201":
		guidance = `Error 42201: Invalid schema

The schema failed validation. Common causes:
- Malformed JSON (check brackets, quotes, commas)
- Invalid Avro schema (missing type, name, or fields)
- Invalid Protobuf syntax (missing syntax declaration, package, or field numbers)
- Invalid JSON Schema (unsupported keywords or types)

Debug steps:
1. Use validate_schema to get a detailed error message
2. For Avro: ensure "type", "name", and "fields" are present for records
3. For Protobuf: ensure 'syntax = "proto3";' is the first line
4. For JSON Schema: ensure "type" is a valid JSON Schema type
5. Check for escape character issues in the schema string
6. Check for malformed JSON (missing brackets, quotes, commas)`

	case "409":
		guidance = `Error 409: Incompatible schema

The new schema is not compatible with existing versions under the current compatibility level.

Debug steps:
1. Use get_config to check the compatibility level
2. Use check_compatibility to get detailed incompatibility reasons
3. Use explain_compatibility_failure to understand what changed and why it breaks
4. Use get_latest_schema to compare with the current schema
5. Common fixes:
   - Add default values to new fields
   - Make new fields optional (nullable)
   - Don't remove or rename existing fields
   - Don't change field types
6. If the change is intentional, consider set_config to NONE temporarily`

	case "40401":
		guidance = `Error 40401: Subject not found

The specified subject does not exist in the registry.

Debug steps:
1. Use list_subjects to see all available subjects
2. Use match_subjects to find similarly named subjects (catches typos)
3. Check for typos in the subject name
4. The subject might have been soft-deleted — use list_subjects with deleted: true
5. If deleted, re-register the schema to create a new version`

	case "40402":
		guidance = `Error 40402: Version not found

The specified version does not exist for the given subject.

Debug steps:
1. Use list_versions to see available versions for the subject
2. The version might have been soft-deleted
3. Check if you're using the correct version number (1-based)`

	case "40403":
		guidance = `Error 40403: Schema not found

No schema exists with the specified global ID.

Debug steps:
1. Use get_max_schema_id to check the highest assigned ID
2. The schema might have been deleted
3. Verify you're using the correct ID (global, not version number)`

	default:
		data, err := fs.ReadFile(content.PromptsFS, "prompts/debug-registration-error.md")
		if err != nil {
			return nil, fmt.Errorf("read prompt file: %w", err)
		}
		guidance = fmt.Sprintf("Error code: %s\n\n%s", errorCode, string(data))
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Debug guide for error code %s", errorCode),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleSetupDataContractsPrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/setup-data-contracts.md", map[string]string{"{subject}": subject})
	if err != nil {
		return nil, err
	}

	registryCtx := resolveContext(req.Params.Arguments["context"])
	latest, err := s.registry.GetLatestSchema(ctx, registryCtx, subject)
	if err == nil && latest != nil {
		guidance += fmt.Sprintf("\n\nCurrent version: %d, type: %s", latest.Version, latest.SchemaType)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Data contracts setup for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleAuditSubjectHistoryPrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/audit-subject-history.md", map[string]string{"{subject}": subject})
	if err != nil {
		return nil, err
	}

	registryCtx := resolveContext(req.Params.Arguments["context"])
	versions, err := s.registry.GetVersions(ctx, registryCtx, subject, false)
	if err == nil {
		guidance += fmt.Sprintf("\n\nRegistered versions: %v", versions)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Version history audit for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleGettingStartedPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/getting-started.md", "Quick-start guide for the Schema Registry MCP server")
}

func (s *Server) handleTroubleshootingPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/troubleshooting.md", "Troubleshooting guide for schema registry issues")
}

func (s *Server) handleImpactAnalysisPrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/impact-analysis.md", map[string]string{"{subject}": subject})
	if err != nil {
		return nil, err
	}

	registryCtx := resolveContext(req.Params.Arguments["context"])
	latest, err := s.registry.GetLatestSchema(ctx, registryCtx, subject)
	if err == nil && latest != nil {
		guidance += fmt.Sprintf("\n\nCurrent version: %d, type: %s", latest.Version, latest.SchemaType)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Impact analysis guide for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleNamingConventionsPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/naming-conventions.md", "Subject naming conventions guide")
}

func (s *Server) handleContextManagementPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/context-management.md", "Multi-tenant context management guide")
}

func (s *Server) handleCompareFormatsPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	useCase := req.Params.Arguments["use_case"]
	if useCase == "" {
		return nil, fmt.Errorf("required argument 'use_case' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/compare-formats.md", map[string]string{"{use_case}": useCase})
	if err != nil {
		return nil, err
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Format comparison for %q", useCase),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleGlossaryLookupPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	topic := strings.ToLower(req.Params.Arguments["topic"])
	if topic == "" {
		return nil, fmt.Errorf("required argument 'topic' is missing")
	}

	type glossaryEntry struct {
		uri      string
		keywords []string
	}

	entries := []glossaryEntry{
		{"schema://glossary/core-concepts", []string{"subject", "version", "schema id", "wire format", "dedup", "fingerprint", "mode", "naming", "strategy", "register", "serializ"}},
		{"schema://glossary/compatibility", []string{"compatibility", "backward", "forward", "full", "transitive", "compat", "promotion", "alias"}},
		{"schema://glossary/data-contracts", []string{"data contract", "metadata", "ruleset", "rule", "tag", "merge", "governance", "domain rule", "migration rule", "encoding rule", "concurrency"}},
		{"schema://glossary/encryption", []string{"encrypt", "csfle", "kek", "dek", "kms", "vault", "envelope", "key rotation", "rewrap", "aes"}},
		{"schema://glossary/contexts", []string{"context", "multi-tenant", "tenant", "namespace", "isolation", "inheritance", "global", "__global"}},
		{"schema://glossary/exporters", []string{"exporter", "schema link", "linking", "replicate", "disaster recovery"}},
		{"schema://glossary/schema-types", []string{"avro", "protobuf", "proto", "json schema", "logical type", "wire type", "canonicali", "draft"}},
		{"schema://glossary/design-patterns", []string{"pattern", "envelope", "lifecycle", "snapshot", "delta", "fat", "thin", "rename", "ci/cd", "dlq", "dead letter"}},
		{"schema://glossary/best-practices", []string{"best practice", "naming", "convention", "mistake", "antipattern", "guidance"}},
		{"schema://glossary/migration", []string{"migrat", "confluent", "import", "import mode", "id preserv"}},
		{"schema://glossary/mcp-configuration", []string{"mcp config", "tool policy", "permission", "preset", "read-only", "confirmation", "origin"}},
		{"schema://glossary/error-reference", []string{"error code", "error ref", "40401", "42201", "diagnostic"}},
		{"schema://glossary/auth-and-security", []string{"auth", "rbac", "role", "api key", "rate limit", "audit"}},
		{"schema://glossary/storage-backends", []string{"storage", "backend", "postgres", "mysql", "cassandra", "stateless"}},
		{"schema://glossary/normalization-and-fingerprinting", []string{"fingerprint", "normal", "canonical", "sha-256", "dedup"}},
		{"schema://glossary/tool-selection-guide", []string{"tool", "which tool", "how to", "decision tree", "find schema"}},
	}

	var matchedURI string
	for _, entry := range entries {
		for _, kw := range entry.keywords {
			if strings.Contains(topic, kw) {
				matchedURI = entry.uri
				break
			}
		}
		if matchedURI != "" {
			break
		}
	}

	if matchedURI == "" {
		matchedURI = "schema://glossary/core-concepts"
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/glossary-lookup.md", map[string]string{
		"{topic}":       topic,
		"{matched_uri}": matchedURI,
	})
	if err != nil {
		return nil, err
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Glossary lookup for %q", topic),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleImportFromConfluentPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/import-from-confluent.md", "Confluent migration workflow")
}

func (s *Server) handleSetupRBACPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/setup-rbac.md", "Authentication and RBAC configuration guide")
}

func (s *Server) handleSchemaReferencesGuidePrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/schema-references-guide.md", "Schema references guide with per-format semantics")
}

func (s *Server) handleFullEncryptionLifecyclePrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/full-encryption-lifecycle.md", "End-to-end CSFLE encryption lifecycle")
}

func (s *Server) handleDataRulesDeepDivePrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/data-rules-deep-dive.md", "Data contract rules deep dive")
}

func (s *Server) handleRegistryHealthAuditPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/registry-health-audit.md", "Registry health audit procedure")
}

func (s *Server) handleSchemaEvolutionCookbookPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/schema-evolution-cookbook.md", "Schema evolution cookbook with practical recipes")
}

func (s *Server) handleNewKafkaTopicPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	topicName := req.Params.Arguments["topic_name"]
	if topicName == "" {
		return nil, fmt.Errorf("required argument 'topic_name' is missing")
	}

	format := req.Params.Arguments["format"]
	if format == "" {
		format = "AVRO"
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/new-kafka-topic.md", map[string]string{
		"{topic_name}": topicName,
		"{format}":     strings.ToUpper(format),
	})
	if err != nil {
		return nil, err
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Kafka topic setup for %q (%s)", topicName, strings.ToUpper(format)),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleDebugDeserializationPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/debug-deserialization.md", "Consumer deserialization troubleshooting guide")
}

func (s *Server) handleDeprecateSubjectPrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/deprecate-subject.md", map[string]string{"{subject}": subject})
	if err != nil {
		return nil, err
	}

	registryCtx := resolveContext(req.Params.Arguments["context"])
	latest, err := s.registry.GetLatestSchema(ctx, registryCtx, subject)
	if err == nil && latest != nil {
		guidance += fmt.Sprintf("\n\nCurrent version: %d, type: %s", latest.Version, latest.SchemaType)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Deprecation workflow for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleCICDIntegrationPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/cicd-integration.md", "CI/CD pipeline integration guide")
}

func (s *Server) handleTeamOnboardingPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	teamName := req.Params.Arguments["team_name"]
	if teamName == "" {
		return nil, fmt.Errorf("required argument 'team_name' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/team-onboarding.md", map[string]string{"{team_name}": teamName})
	if err != nil {
		return nil, err
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Team onboarding workflow for %q", teamName),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleGovernanceSetupPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	return promptFromFS(content.PromptsFS, "prompts/governance-setup.md", "Schema governance setup guide")
}

func (s *Server) handleCrossCuttingChangePrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	fieldName := req.Params.Arguments["field_name"]
	if fieldName == "" {
		return nil, fmt.Errorf("required argument 'field_name' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/cross-cutting-change.md", map[string]string{"{field_name}": fieldName})
	if err != nil {
		return nil, err
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Cross-cutting change workflow for field %q", fieldName),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleSchemaReviewChecklistPrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance, err := promptTemplateFromFS(content.PromptsFS, "prompts/schema-review-checklist.md", map[string]string{"{subject}": subject})
	if err != nil {
		return nil, err
	}

	registryCtx := resolveContext(req.Params.Arguments["context"])
	latest, err := s.registry.GetLatestSchema(ctx, registryCtx, subject)
	if err == nil && latest != nil {
		guidance += fmt.Sprintf("\n\nCurrent latest version: %d, type: %s", latest.Version, latest.SchemaType)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Schema review checklist for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}
