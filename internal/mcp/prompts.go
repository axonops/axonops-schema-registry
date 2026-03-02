package mcp

import (
	"context"
	"fmt"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
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
		},
	}, s.handleEvolveSchemaPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "check-compatibility",
		Description: "Troubleshoot schema compatibility issues and suggest fixes",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name to check compatibility for", Required: true},
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
		},
	}, s.handleReviewSchemaQualityPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "plan-breaking-change",
		Description: "Plan a safe breaking schema change with migration strategy",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name where the breaking change is planned", Required: true},
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
		},
	}, s.handleSetupDataContractsPrompt)

	s.mcpServer.AddPrompt(&gomcp.Prompt{
		Name:        "audit-subject-history",
		Description: "Review the version history and evolution of a schema subject",
		Arguments: []*gomcp.PromptArgument{
			{Name: "subject", Description: "Subject name to audit", Required: true},
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
			{Name: "subject", Description: "Subject name to analyse impact for", Required: true},
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
}

// --- Prompt handlers ---

func promptMessage(role, text string) *gomcp.PromptMessage {
	return &gomcp.PromptMessage{
		Role:    gomcp.Role(role),
		Content: &gomcp.TextContent{Text: text},
	}
}

func (s *Server) handleDesignSchemaPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	format := strings.ToUpper(req.Params.Arguments["format"])
	domain := req.Params.Arguments["domain"]

	if format == "" {
		return nil, fmt.Errorf("required argument 'format' is missing")
	}

	var guidance string
	switch format {
	case "AVRO":
		guidance = `Design an Avro schema following these best practices:
- Use a descriptive record name in PascalCase with a namespace (e.g. com.company.events)
- Use snake_case for field names
- Always include a namespace to avoid naming conflicts
- Use union types ["null", "type"] with default null for optional fields
- Use logical types for dates (timestamp-millis), decimals (bytes + decimal), and UUIDs (string + uuid)
- Consider schema evolution: add new fields with defaults, avoid removing or renaming fields
- Use enums for fixed sets of values

Available tools: register_schema, check_compatibility, get_latest_schema, lookup_schema`

	case "PROTOBUF":
		guidance = `Design a Protobuf schema following these best practices:
- Use syntax = "proto3" (required)
- Use a package declaration matching your domain (e.g. package company.events.v1)
- Use PascalCase for message and enum names, snake_case for field names
- Use explicit field numbers and never reuse deleted field numbers
- Use oneof for variant/union types
- Use repeated for arrays, map<K,V> for key-value pairs
- Use well-known types (google.protobuf.Timestamp, Duration, etc.) when appropriate
- Use enums with UNSPECIFIED = 0 as the first value
- Consider backward compatibility: only add new fields, never change field numbers

Available tools: register_schema (with schema_type: PROTOBUF), check_compatibility`

	case "JSON":
		guidance = `Design a JSON Schema following these best practices:
- Use "type": "object" as the root type
- Define a "required" array listing mandatory fields
- Use "additionalProperties": false to prevent unexpected fields
- Use format validators: "email", "uri", "date-time", "uuid"
- Use pattern for custom string validation (regex)
- Use minimum/maximum for number ranges, minLength/maxLength for strings
- Use enum for fixed value sets
- Use $ref for reusable type definitions
- Consider using oneOf/anyOf for variant types

Available tools: register_schema (with schema_type: JSON), check_compatibility`

	default:
		guidance = fmt.Sprintf("Unknown format %q. Supported formats: AVRO, PROTOBUF, JSON. Use the get_schema_types tool to verify.", format)
	}

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

	guidance := fmt.Sprintf(`Evolve the schema for subject %q safely.

Steps:
1. Use get_latest_schema to inspect the current schema for %q
2. Use get_config to check the compatibility level
3. Plan your changes following the compatibility rules:
   - BACKWARD: new schema can read old data (add optional fields with defaults)
   - FORWARD: old schema can read new data (only remove optional fields)
   - FULL: both backward and forward compatible
4. Use check_compatibility to validate your changes before registering
5. Use register_schema to register the evolved schema

Common safe changes:
- Add a new optional field with a default value
- Add a new field with a union type ["null", "type"] and default null
- Widen a type (e.g. int → long in Avro)

Breaking changes to avoid:
- Removing a required field
- Changing a field type incompatibly
- Renaming a field (treated as remove + add)`, subject, subject)

	// Try to include current schema context
	latest, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subject)
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

	guidance := fmt.Sprintf(`Troubleshoot compatibility issues for subject %q.

Steps:
1. Use get_config to check the current compatibility level for %q
2. Use list_versions to see all registered versions
3. Use get_latest_schema to inspect the current schema
4. Use check_compatibility to test your new schema against existing versions
5. If incompatible, review the error details and adjust your schema

Common compatibility fixes:
- BACKWARD violations: Add a default value to new required fields, or make them optional
- FORWARD violations: Don't remove fields that consumers might depend on
- FULL violations: Only add optional fields with defaults

If you need to make a breaking change:
- Consider using set_config to temporarily change the compatibility level
- Or create a new subject (e.g. subject-v2) for the breaking change
- Use set_mode READONLY to protect finalized subjects`, subject, subject)

	config, err := s.registry.GetConfigFull(ctx, registrycontext.DefaultContext, subject)
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

	guidance := fmt.Sprintf(`Migrate schemas from %s to %s format.

Steps:
1. Use list_subjects to find schemas to migrate
2. Use get_latest_schema to inspect each schema
3. Convert the schema to %s format following these guidelines:
4. Use register_schema with schema_type: %q to register the converted schema
5. Use check_compatibility to validate if needed

Migration considerations from %s to %s:`, source, target, target, target, source, target)

	switch {
	case source == "AVRO" && target == "PROTOBUF":
		guidance += `
- Avro records → Protobuf messages
- Avro unions ["null", "type"] → Protobuf optional fields
- Avro enums → Protobuf enums (add UNSPECIFIED = 0)
- Avro maps → Protobuf map<string, V>
- Avro arrays → Protobuf repeated fields
- Avro logical types → Protobuf well-known types
- Avro namespace → Protobuf package`

	case source == "AVRO" && target == "JSON":
		guidance += `
- Avro records → JSON objects with properties
- Avro string/int/long/float/double → JSON string/integer/number
- Avro unions → JSON oneOf
- Avro enums → JSON enum arrays
- Avro arrays → JSON arrays with items
- Avro maps → JSON objects with additionalProperties`

	case source == "PROTOBUF" && target == "AVRO":
		guidance += `
- Protobuf messages → Avro records
- Protobuf optional → Avro union ["null", "type"]
- Protobuf enums → Avro enums (remove UNSPECIFIED value)
- Protobuf map → Avro map type
- Protobuf repeated → Avro array type
- Protobuf package → Avro namespace`

	default:
		guidance += fmt.Sprintf(`
- Map types from %s to their %s equivalents
- Preserve field names and semantics
- Handle nullable/optional fields appropriately
- Test the converted schema with check_compatibility`, source, target)
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

	guidance := fmt.Sprintf(`Set up client-side field encryption (CSFLE) with %s.

Steps:
1. Create a KEK (Key Encryption Key) using the create_kek tool:
   - name: descriptive name (e.g. "production-kek")
   - kms_type: %q
   - kms_key_id: your KMS key identifier
   - kms_props: provider-specific properties

2. Create a DEK (Data Encryption Key) using the create_dek tool:
   - kek_name: name of the KEK created above
   - subject: schema subject to encrypt
   - algorithm: AES256_GCM (recommended) or AES256_SIV

3. The DEK is automatically wrapped (encrypted) by the KEK via your KMS

Available tools: create_kek, get_kek, list_keks, create_dek, get_dek, list_deks

KMS provider %q considerations:`, kmsType, kmsType, kmsType)

	switch strings.ToLower(kmsType) {
	case "hcvault":
		guidance += `
- kms_key_id: transit key name in Vault
- kms_props: {"vault.url": "https://vault:8200", "vault.token": "..."}`
	case "aws-kms":
		guidance += `
- kms_key_id: AWS KMS key ARN
- kms_props: {"aws.region": "us-east-1"}`
	case "gcp-kms":
		guidance += `
- kms_key_id: GCP KMS key resource name
- kms_props: {"gcp.project": "my-project"}`
	case "azure-kms":
		guidance += `
- kms_key_id: Azure Key Vault key identifier
- kms_props: {"azure.tenant.id": "...", "azure.client.id": "..."}`
	default:
		guidance += fmt.Sprintf(`
- Refer to your %s provider documentation for kms_key_id and kms_props`, kmsType)
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

	guidance := fmt.Sprintf(`Set up schema linking with a %s context exporter.

Steps:
1. Create an exporter using the create_exporter tool:
   - name: descriptive name (e.g. "prod-to-dr")
   - context_type: %q
   - subjects: list of subjects to export (empty = all)
   - config: destination registry connection details

2. Monitor the exporter using get_exporter_status
3. Control the exporter: pause_exporter, resume_exporter, reset_exporter

Context types:
- AUTO: exports all subjects automatically
- CUSTOM: exports only specified subjects with optional rename format
- NONE: no context prefix on exported subjects

Config properties:
- schema.registry.url: destination registry URL
- basic.auth.credentials.source: auth method
- basic.auth.user.info: username:password

Available tools: create_exporter, get_exporter, list_exporters, get_exporter_status, pause_exporter, resume_exporter`, exporterType, exporterType)

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

	guidance := fmt.Sprintf(`Review the schema quality for subject %q.

Use get_latest_schema to fetch the current schema, then evaluate:

1. **Naming conventions**:
   - Record/message names: PascalCase
   - Field names: snake_case
   - Enum values: UPPER_SNAKE_CASE
   - Namespace/package: reverse domain notation

2. **Nullability**:
   - Optional fields should be nullable (Avro: union with null, Protobuf: optional)
   - Required fields should NOT be nullable
   - Default values should be meaningful

3. **Type usage**:
   - Use logical/semantic types (timestamps, UUIDs, decimals) instead of raw primitives
   - Use enums for fixed value sets instead of plain strings
   - Use appropriate numeric precision (int vs long, float vs double)

4. **Evolution readiness**:
   - All fields should have sensible defaults for backward compatibility
   - Avoid required fields that might become optional later
   - Consider using a version field or schema fingerprint

5. **Documentation**:
   - Fields should have descriptive names that are self-documenting
   - Complex fields should have doc comments (Avro: "doc" field, Protobuf: // comments)

Available tools: get_latest_schema, list_versions, get_config`, subject)

	latest, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subject)
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

	guidance := fmt.Sprintf(`Plan a safe breaking change for subject %q.

Steps:
1. Use get_latest_schema to understand the current schema
2. Use get_config to check the compatibility level
3. Use list_versions to see the version history

Strategy options:

**Option A: New subject (recommended for major changes)**
- Create a new subject (e.g. %s-v2) with the new schema
- Migrate producers to the new subject
- Keep the old subject in READONLY mode for consumers
- Tools: register_schema, set_mode READONLY

**Option B: Compatibility bypass (for minor breaking changes)**
- Set compatibility to NONE temporarily: set_config with compatibility_level: NONE
- Register the breaking schema
- Restore compatibility: set_config with original level
- WARNING: existing consumers may fail to deserialize

**Option C: Multi-step evolution**
- Add new fields alongside old fields (backward compatible)
- Migrate all consumers to use new fields
- Remove old fields in a later version
- Requires NONE compatibility for the final removal step

Always test with check_compatibility before registering.`, subject, subject)

	latest, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subject)
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
1. Validate the schema syntax independently
2. For Avro: ensure "type", "name", and "fields" are present for records
3. For Protobuf: ensure 'syntax = "proto3";' is the first line
4. For JSON Schema: ensure "type" is a valid JSON Schema type
5. Check for escape character issues in the schema string`

	case "409":
		guidance = `Error 409: Incompatible schema

The new schema is not compatible with existing versions under the current compatibility level.

Debug steps:
1. Use get_config to check the compatibility level
2. Use check_compatibility to get detailed incompatibility reasons
3. Use get_latest_schema to compare with the current schema
4. Common fixes:
   - Add default values to new fields
   - Make new fields optional (nullable)
   - Don't remove or rename existing fields
   - Don't change field types
5. If the change is intentional, consider set_config to NONE temporarily`

	case "40401":
		guidance = `Error 40401: Subject not found

The specified subject does not exist in the registry.

Debug steps:
1. Use list_subjects to see all available subjects
2. Check for typos in the subject name
3. The subject might have been soft-deleted — use list_subjects with deleted: true
4. If deleted, re-register the schema to create a new version`

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
		guidance = fmt.Sprintf(`Error code: %s

General debug steps:
1. Check the error message for specific details
2. Use health_check to verify the registry is running
3. Use get_server_info to check server version and capabilities
4. Review the schema content for syntax errors
5. Check compatibility settings with get_config

Common error codes:
- 42201: Invalid schema
- 42203: Invalid compatibility level
- 409: Schema incompatible
- 40401: Subject not found
- 40402: Version not found
- 40403: Schema not found
- 50001: Internal server error`, errorCode)
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

	guidance := fmt.Sprintf(`Set up data contracts for subject %q.

Data contracts add metadata, tags, and data quality rules to schemas.

Steps:
1. Use get_latest_schema to inspect the current schema for %q
2. Use set_config_full to add metadata and rules:

   Metadata properties:
   - owner: team or person responsible
   - description: what this schema represents
   - tags: classification tags (e.g. pii, financial, internal)

   Data quality rules (ruleSet):
   - DOMAIN rules: field-level validation (e.g. email format, range checks)
   - MIGRATION rules: transform data between versions
   - All rules have: name, kind, type, mode, expr, tags

3. Use get_config_full to verify the configuration
4. Use get_subject_metadata to inspect applied metadata

Available tools: set_config_full, get_config_full, get_subject_config_full, get_subject_metadata

Example metadata structure:
{
  "properties": {"owner": "data-team", "description": "User events"},
  "ruleSet": {
    "domainRules": [
      {"name": "email_check", "kind": "CONDITION", "type": "DOMAIN", "mode": "WRITE", "expr": "email matches '^.+@.+$'"}
    ]
  }
}`, subject, subject)

	latest, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subject)
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

	guidance := fmt.Sprintf(`Audit the version history of subject %q.

Steps:
1. Use list_versions to get all version numbers for %q
2. Use get_schema_version for each version to see the full schema
3. Compare consecutive versions to identify changes:
   - Added fields
   - Removed fields
   - Type changes
   - Default value changes
4. Use get_config to check the compatibility policy
5. Use get_referenced_by to find schemas that reference this subject

This helps you understand:
- How the schema has evolved over time
- Whether evolution has followed best practices
- If any versions introduced breaking changes
- Which other schemas depend on this one

Available tools: list_versions, get_schema_version, get_latest_schema, get_config, get_referenced_by`, subject, subject)

	versions, err := s.registry.GetVersions(ctx, registrycontext.DefaultContext, subject, false)
	if err == nil {
		guidance += fmt.Sprintf("\n\nRegistered versions: %v", versions)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Version history audit for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleGettingStartedPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	guidance := `Welcome to the Schema Registry MCP server. Here's a quick-start guide.

## Core operations

- **list_subjects** — see all registered subjects
- **get_latest_schema** — fetch the current schema for a subject
- **register_schema** — register a new schema version
- **check_compatibility** — test a schema before registering

## Discovery

- **search_schemas** — search schema content by keyword or regex
- **match_subjects** — find subjects by name pattern (regex, glob, or fuzzy)
- **get_registry_statistics** — overview of subjects, versions, types, KEKs, and exporters

## Schema intelligence

- **score_schema_quality** — analyse naming, docs, type safety, and evolution readiness
- **diff_schemas** — compare two schema versions structurally
- **find_similar_schemas** — find schemas with overlapping field sets
- **suggest_schema_evolution** — generate a compatible schema change
- **explain_compatibility_failure** — human-readable explanations for compat errors

## Configuration

- **get_config / set_config** — manage compatibility levels (BACKWARD, FORWARD, FULL, NONE)
- **get_mode / set_mode** — manage modes (READWRITE, READONLY, IMPORT)

## Encryption (CSFLE)

- **create_kek / create_dek** — set up client-side field encryption
- **list_keks / list_deks** — inspect encryption keys

## Resources (read-only data)

Resources are available via URI patterns like ` + "`schema://subjects`" + `, ` + "`schema://subjects/{name}`" + `, etc.

## Getting help

Use the other prompts for detailed guidance: design-schema, evolve-schema, check-compatibility, troubleshooting, setup-encryption, and more.`

	return &gomcp.GetPromptResult{
		Description: "Quick-start guide for the Schema Registry MCP server",
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleTroubleshootingPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	guidance := `Diagnostic guide for common schema registry issues.

## Step 1: Check health
Use **health_check** to verify the registry is running and storage is connected.

## Step 2: Identify the error

| Error Code | Meaning | Likely cause |
|------------|---------|--------------|
| 42201 | Invalid schema | Malformed JSON, missing required Avro/Protobuf/JSON Schema fields |
| 42203 | Invalid compatibility level | Typo in compatibility level string |
| 409 | Incompatible schema | Schema violates the configured compatibility level |
| 40401 | Subject not found | Typo in subject name, or subject was soft-deleted |
| 40402 | Version not found | Version number does not exist for this subject |
| 40403 | Schema not found | Global schema ID does not exist |
| 50001 | Internal error | Storage backend issue, check server logs |

## Step 3: Debug by category

**Registration failures:**
1. Use **validate_schema** to check syntax without registering
2. Use **get_config** to check the compatibility level
3. Use **check_compatibility** to test against existing versions
4. Use **explain_compatibility_failure** for detailed fix suggestions

**Subject/version not found:**
1. Use **list_subjects** to see all subjects (add include_deleted for soft-deleted)
2. Use **match_subjects** with fuzzy mode to find similar names
3. Use **list_versions** to check available versions

**Performance issues:**
1. Use **get_registry_statistics** to check registry size
2. Use **count_versions** to check version count per subject
3. Large registries (>10k subjects) may need pagination on search operations

**Encryption issues:**
1. Use **list_keks** to verify KEK exists
2. Use **test_kek** to verify KMS connectivity
3. Use **list_deks** to check DEK status

Available tools: health_check, get_server_info, validate_schema, check_compatibility, explain_compatibility_failure, list_subjects, match_subjects`

	return &gomcp.GetPromptResult{
		Description: "Troubleshooting guide for schema registry issues",
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleImpactAnalysisPrompt(ctx context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	subject := req.Params.Arguments["subject"]
	if subject == "" {
		return nil, fmt.Errorf("required argument 'subject' is missing")
	}

	guidance := fmt.Sprintf(`Assess the impact of a proposed schema change on subject %q.

## Step 1: Understand the current state
1. Use **get_latest_schema** to fetch the current schema for %q
2. Use **get_config** to check the compatibility level
3. Use **list_versions** to see the version history

## Step 2: Find dependents
1. Use **get_referenced_by** to find schemas that reference this subject
2. Use **get_dependency_graph** to see the full transitive dependency tree
3. Use **find_similar_schemas** to identify structurally related schemas

## Step 3: Check field usage
1. Use **check_field_consistency** for fields you plan to change
2. Use **find_schemas_by_field** to find other schemas using the same field names

## Step 4: Validate the change
1. Use **check_compatibility** to test your proposed schema
2. Use **check_compatibility_multi** if the schema is used across multiple subjects
3. Use **explain_compatibility_failure** if compatibility fails
4. Use **diff_schemas** to see a structural comparison

## Step 5: Plan the rollout
1. Use **suggest_schema_evolution** to generate a compatible schema
2. Use **plan_migration_path** if the change requires multiple steps
3. Consider using **set_mode** READONLY on the subject during migration`, subject, subject)

	latest, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subject)
	if err == nil && latest != nil {
		guidance += fmt.Sprintf("\n\nCurrent version: %d, type: %s", latest.Version, latest.SchemaType)
	}

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Impact analysis guide for %q", subject),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleNamingConventionsPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	guidance := `Guide to subject naming strategies in the schema registry.

## Naming strategies

### topic_name (default)
Pattern: ` + "`{topic}-key`" + ` or ` + "`{topic}-value`" + `
Examples: ` + "`orders-key`" + `, ` + "`orders-value`" + `, ` + "`user-events-value`" + `
Use when: one schema per Kafka topic, simple key/value distinction.

### record_name
Pattern: ` + "`{fully.qualified.RecordName}`" + `
Examples: ` + "`com.example.Order`" + `, ` + "`com.example.UserEvent`" + `
Use when: multiple event types share a topic, schemas identified by record name.

### topic_record_name
Pattern: ` + "`{topic}-{fully.qualified.RecordName}`" + `
Examples: ` + "`orders-com.example.OrderCreated`" + `, ` + "`orders-com.example.OrderShipped`" + `
Use when: multiple event types per topic and you want topic context in the subject name.

## Validation
Use **validate_subject_name** to check if a name conforms to a strategy:
- validate_subject_name(subject: "orders-value", strategy: "topic_name")
- validate_subject_name(subject: "com.example.Order", strategy: "record_name")

## Best practices
- Pick one strategy per environment and use it consistently
- Use **detect_schema_patterns** to check current naming convention coverage
- Use **match_subjects** to find subjects that deviate from the dominant pattern
- Avoid mixing strategies in the same registry context
- Use lowercase with hyphens for topic names: ` + "`user-events`" + ` not ` + "`UserEvents`" + `
- Use reverse-domain namespace for record names: ` + "`com.company.domain.Type`" + ``

	return &gomcp.GetPromptResult{
		Description: "Subject naming conventions guide",
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleContextManagementPrompt(_ context.Context, _ *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	guidance := `Guide for managing multi-tenant contexts in the schema registry.

## What are contexts?
Contexts are tenant namespaces that isolate schemas, subjects, and configuration. The default context is "." (dot). Contexts enable multi-tenancy — different teams, environments, or applications can have independent schema registries within the same server.

## Listing and navigating contexts
- **list_contexts** — list all available contexts
- **list_subjects** — lists subjects in the default context
- Subjects can be qualified with context: ` + "`:.staging:my-subject`" + `

## The 4-tier config/mode inheritance chain
Configuration and mode settings cascade through 4 levels:

1. **Server default** — hardcoded BACKWARD compatibility, READWRITE mode
2. **Global (__GLOBAL)** — set via set_config/set_mode with no subject
3. **Context global** — per-context default (overrides __GLOBAL)
4. **Per-subject** — most specific (overrides everything above)

To check effective config: **get_config** with a subject name returns the resolved value.
To check effective mode: **get_mode** with a subject name returns the resolved value.

## Managing configuration per context
- **set_config** — set compatibility level (per-subject or global)
- **delete_config** — remove per-subject config (falls back to context global)
- **set_mode** — set mode (READWRITE, READONLY, READONLY_OVERRIDE, IMPORT)
- **delete_mode** — remove per-subject mode (falls back to context global)

## Import and migration
- Use **set_mode** with mode IMPORT to enable ID-preserving schema import
- Use **import_schemas** to bulk import schemas with preserved IDs
- Reset mode after import: **set_mode** with mode READWRITE

## Resources
- ` + "`schema://contexts`" + ` — list all contexts
- ` + "`schema://contexts/{context}/subjects`" + ` — subjects in a specific context

Available tools: list_contexts, get_config, set_config, delete_config, get_mode, set_mode, delete_mode, import_schemas`

	return &gomcp.GetPromptResult{
		Description: "Multi-tenant context management guide",
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}

func (s *Server) handleCompareFormatsPrompt(_ context.Context, req *gomcp.GetPromptRequest) (*gomcp.GetPromptResult, error) {
	useCase := req.Params.Arguments["use_case"]
	if useCase == "" {
		return nil, fmt.Errorf("required argument 'use_case' is missing")
	}

	guidance := fmt.Sprintf(`Compare Avro, Protobuf, and JSON Schema for the use case: %q

## Format Comparison

| Feature | Avro | Protobuf | JSON Schema |
|---------|------|----------|-------------|
| Serialization | Binary (compact) | Binary (compact) | Text (JSON) |
| Schema evolution | Excellent | Good | Limited |
| Type system | Rich (unions, logical types) | Strong (oneof, well-known types) | Flexible (oneOf, anyOf) |
| Code generation | Moderate | Excellent | Minimal |
| Human readability | Schema: JSON, Data: binary | Schema: .proto, Data: binary | Both: JSON |
| Kafka integration | Native | Supported | Supported |
| gRPC support | Limited | Native | Not applicable |
| Validation | Schema-level | Schema-level | Rich constraints |

## Recommendations by use case

**Event streaming (Kafka):** Avro
- Best schema evolution support with BACKWARD/FORWARD compatibility
- Compact binary serialization reduces Kafka storage/bandwidth
- Native Kafka ecosystem integration

**RPC/Microservices:** Protobuf
- Native gRPC support with code generation
- Strong typing across languages
- Efficient binary serialization

**REST APIs:** JSON Schema
- Human-readable request/response validation
- Rich constraint validation (patterns, ranges, formats)
- Direct JSON compatibility

**Mixed/CQRS systems:** Use multiple formats
- Avro for events (Kafka topics)
- Protobuf for commands (gRPC)
- JSON Schema for queries (REST responses)

Available tools: register_schema, get_schema_types`, useCase)

	return &gomcp.GetPromptResult{
		Description: fmt.Sprintf("Format comparison for %q", useCase),
		Messages:    []*gomcp.PromptMessage{promptMessage("user", guidance)},
	}, nil
}
