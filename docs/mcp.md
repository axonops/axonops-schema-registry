# MCP Server — AI-Assisted Schema Management

The AxonOps Schema Registry includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that enables AI assistants — Claude, Cursor, Windsurf, VS Code Copilot, and others — to interact directly with the registry for schema management, compatibility checking, encryption key management, and operational tasks.

The MCP server runs alongside the [REST API](api-reference.md) as a separate HTTP endpoint, sharing the same `registry.Registry` service layer, and can be enabled or disabled via configuration. All schema analysis and intelligence capabilities exposed via MCP are also available as [REST endpoints](api-reference.md#axonops-extensions) for CI/CD pipelines and custom tooling.

> For the complete auto-generated reference with all tool parameter schemas, see the [MCP API Reference](mcp-reference.md). For the full REST API (including additional AxonOps endpoints), see the [API Reference](api-reference.md).

## Contents

- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Tools](#tools)
- [Resources](#resources)
- [Prompts](#prompts)
- [Security](#security)
- [Schema Intelligence](#schema-intelligence)
- [Connecting AI Clients](#connecting-ai-clients)
- [Troubleshooting](#troubleshooting)
- [API Reference](#api-reference)

## Quick Start

### 1. Enable the MCP Server

Add to your configuration YAML:

```yaml
mcp:
  enabled: true
  host: 127.0.0.1
  port: 9081
```

Or via environment variables:

```bash
export SCHEMA_REGISTRY_MCP_ENABLED=true
export SCHEMA_REGISTRY_MCP_PORT=9081
```

### 2. Start the Registry

```bash
./schema-registry --config config.yaml
```

The MCP server starts on `http://127.0.0.1:9081/mcp` alongside the REST API on port 8081.

### 3. Connect Your AI Client

#### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "schema-registry": {
      "url": "http://localhost:9081/mcp"
    }
  }
}
```

#### VS Code / Cursor (`.vscode/mcp.json`)

```json
{
  "servers": {
    "schema-registry": {
      "type": "http",
      "url": "http://localhost:9081/mcp"
    }
  }
}
```

## Configuration

The MCP server is configured under the `mcp` key in the configuration YAML.

```yaml
mcp:
  enabled: false                    # Enable/disable the MCP server
  host: 127.0.0.1                  # Bind address (localhost by default for security)
  port: 9081                       # Port (separate from REST API)
  auth_token: ""                   # Bearer token for authentication (empty = no auth)
  read_only: false                 # Restrict to read-only tools
  tool_policy: allow_all           # Tool access policy: allow_all, deny_list, allow_list
  allowed_tools: []                # Tools to expose (for allow_list policy)
  denied_tools: []                 # Tools to hide (for deny_list policy)
  allowed_origins:                 # Origin header allowlist (MCP spec security)
    - "http://localhost:*"
    - "https://localhost:*"
    - "vscode-webview://*"
  require_confirmations: false     # Two-phase confirmations for destructive operations
  confirmation_ttl_secs: 300       # Confirmation token TTL (5 minutes)
  log_schemas: false               # Log full schema bodies (disabled by default)
  permission_preset: ""            # Named preset: readonly, developer, operator, admin, full
  permission_scopes: []            # Individual scopes (when no preset is set)
```

### Environment Variable Overrides

Every config field has a corresponding environment variable:

| Field | Environment Variable |
|-------|---------------------|
| `enabled` | `SCHEMA_REGISTRY_MCP_ENABLED` |
| `host` | `SCHEMA_REGISTRY_MCP_HOST` |
| `port` | `SCHEMA_REGISTRY_MCP_PORT` |
| `auth_token` | `SCHEMA_REGISTRY_MCP_AUTH_TOKEN` |
| `read_only` | `SCHEMA_REGISTRY_MCP_READ_ONLY` |
| `allowed_origins` | `SCHEMA_REGISTRY_MCP_ALLOWED_ORIGINS` (comma-separated) |
| `require_confirmations` | `SCHEMA_REGISTRY_MCP_REQUIRE_CONFIRMATIONS` |
| `confirmation_ttl_secs` | `SCHEMA_REGISTRY_MCP_CONFIRMATION_TTL` |
| `log_schemas` | `SCHEMA_REGISTRY_MCP_LOG_SCHEMAS` |
| `permission_preset` | `SCHEMA_REGISTRY_MCP_PERMISSION_PRESET` |
| `permission_scopes` | `SCHEMA_REGISTRY_MCP_PERMISSION_SCOPES` (comma-separated) |

## Tools

The MCP server exposes a comprehensive set of tools organized by functional area. Each tool is instrumented with Prometheus metrics and structured logging.

### Core Schema Operations

#### Read Operations

| Tool | Description |
|------|-------------|
| `get_schema_by_id` | Get a schema by its global ID |
| `get_raw_schema_by_id` | Get the raw schema string by global ID |
| `get_schema_version` | Get a schema at a specific subject version |
| `get_raw_schema_version` | Get the raw schema string at a specific subject version |
| `get_latest_schema` | Get the latest schema for a subject |
| `list_versions` | List all version numbers for a subject |
| `list_subjects` | List subjects (with optional regex `pattern` filtering) |
| `get_subjects_for_schema` | Get all subjects using a specific schema ID |
| `get_versions_for_schema` | Get all subject-version pairs for a schema ID |
| `get_referenced_by` | Get schemas that reference a given schema version |
| `lookup_schema` | Check if a schema already exists under a subject |
| `get_schema_types` | List supported schema types (AVRO, PROTOBUF, JSON) |
| `list_schemas` | List all schemas with optional filters |

#### Write Operations

| Tool | Description |
|------|-------------|
| `register_schema` | Register a new schema version (supports `metadata` and `ruleSet` for data contracts) |
| `delete_subject` | Soft-delete or permanently delete a subject |
| `delete_version` | Delete a specific schema version |
| `check_compatibility` | Check compatibility of a schema against a subject |

#### Server

| Tool | Description |
|------|-------------|
| `health_check` | Check registry health and storage connectivity |
| `get_server_info` | Get server version, features, supported types, build info |
| `get_max_schema_id` | Get the highest schema ID in the registry |

### Configuration & Mode

| Tool | Description |
|------|-------------|
| `get_config` | Get compatibility level (global or per-subject) |
| `set_config` | Set compatibility level (supports data contract fields) |
| `delete_config` | Delete per-subject or global compatibility config |
| `get_mode` | Get registry mode (global or per-subject) |
| `set_mode` | Set registry mode (READWRITE, READONLY, IMPORT) |
| `delete_mode` | Delete per-subject or global mode override |

### Context & Import

| Tool | Description |
|------|-------------|
| `list_contexts` | List all registry contexts (tenant namespaces) |
| `import_schemas` | Bulk import schemas preserving original IDs |

### Encryption — KEK/DEK

| Tool | Description |
|------|-------------|
| `create_kek` | Create a Key Encryption Key |
| `get_kek` | Get KEK details by name |
| `update_kek` | Update KEK properties |
| `delete_kek` | Delete a KEK (soft or permanent) |
| `undelete_kek` | Restore a soft-deleted KEK |
| `list_keks` | List all KEKs |
| `test_kek` | Test KMS connectivity for a KEK |
| `create_dek` | Create a Data Encryption Key |
| `get_dek` | Get DEK by KEK name and subject |
| `list_deks` | List DEK subjects under a KEK |
| `list_dek_versions` | List DEK versions for a subject |
| `delete_dek` | Delete a DEK |
| `undelete_dek` | Restore a soft-deleted DEK |

### Exporters — Schema Linking

| Tool | Description |
|------|-------------|
| `create_exporter` | Create a schema exporter |
| `get_exporter` | Get exporter details |
| `update_exporter` | Update exporter configuration |
| `delete_exporter` | Delete an exporter |
| `list_exporters` | List all exporters |
| `get_exporter_config` | Get exporter configuration map |
| `update_exporter_config` | Update exporter configuration map |
| `get_exporter_status` | Get exporter run status |
| `pause_exporter` | Pause an exporter |
| `resume_exporter` | Resume a paused exporter |
| `reset_exporter` | Reset exporter state |

### Metadata & Data Contracts

| Tool | Description |
|------|-------------|
| `get_config_full` | Get full config including data contract fields |
| `set_config_full` | Set full config with metadata/ruleSet defaults and overrides |
| `get_subject_config_full` | Get per-subject full config |
| `resolve_alias` | Resolve a subject alias to its target |
| `get_schemas_by_subject` | Get all schema versions for a subject |
| `check_write_mode` | Check if the registry accepts writes |
| `format_schema` | Pretty-print a schema string |
| `get_global_config_direct` | Get global config bypassing subject inheritance |
| `get_subject_metadata` | Get subject metadata with key/value filtering |
| `get_cluster_id` | Get the registry cluster ID |
| `get_server_version` | Get the server version string |
| `rewrap_dek` | Re-encrypt a DEK with the latest KEK version |

### Admin — Users & API Keys

| Tool | Description |
|------|-------------|
| `list_users` | List all users |
| `create_user` | Create a new user |
| `get_user` | Get user by ID |
| `get_user_by_username` | Get user by username |
| `update_user` | Update user properties |
| `delete_user` | Delete a user |
| `list_apikeys` | List all API keys |
| `create_apikey` | Create a new API key |
| `get_apikey` | Get API key by ID |
| `update_apikey` | Update API key properties |
| `delete_apikey` | Delete an API key |
| `revoke_apikey` | Revoke an active API key |
| `rotate_apikey` | Rotate an API key (generate new secret) |
| `list_roles` | List available RBAC roles |
| `change_password` | Change a user's password |

### Validation & Export

| Tool | Description |
|------|-------------|
| `validate_schema` | Validate a schema without registering it |
| `normalize_schema` | Get canonical form and fingerprint |
| `validate_subject_name` | Validate subject naming conventions |
| `search_schemas` | Search schema content (substring or regex) |
| `get_schema_history` | Get version history for a subject |
| `get_dependency_graph` | Get schema dependency graph (references and dependents) |
| `export_schema` | Export a single schema version with metadata |
| `export_subject` | Export all versions of a subject |
| `get_registry_statistics` | Get registry-wide statistics |
| `count_versions` | Count versions for a subject |
| `count_subjects` | Count total subjects |

### Comparison & Search

| Tool | Description |
|------|-------------|
| `check_compatibility_multi` | Check compatibility against multiple subjects |
| `diff_schemas` | Diff two schema versions (added/removed/changed fields) |
| `compare_subjects` | Compare fields between two subjects |
| `suggest_compatible_change` | Get rule-based suggestions for compatible changes |
| `match_subjects` | Match subjects by regex, glob, or fuzzy pattern |
| `explain_compatibility_failure` | Explain why a schema is incompatible with fix suggestions |

### Schema Intelligence

| Tool | Description |
|------|-------------|
| `find_schemas_by_field` | Find schemas containing a field (exact, fuzzy, or regex) |
| `find_schemas_by_type` | Find schemas by field type pattern |
| `find_similar_schemas` | Find schemas similar to a given subject (Jaccard similarity) |
| `score_schema_quality` | Score schema quality across 6 categories |
| `check_field_consistency` | Check if a field has consistent types across all schemas |
| `get_schema_complexity` | Compute complexity metrics and grade |
| `detect_schema_patterns` | Detect common field patterns across the registry |
| `suggest_schema_evolution` | Get evolution suggestions based on compatibility level |
| `plan_migration_path` | Plan minimum-step migration from current to target schema |

### Context Support

All schema, subject, config, mode, validation, comparison, intelligence, and metadata tools accept an optional `context` parameter for multi-tenant isolation. When omitted, the default context (`.`) is used. Schemas registered in one context are invisible to queries in another context.

**Tools that accept `context`:** All tools in the schema read, schema write, config, validation, comparison, intelligence, and metadata categories.

**Tools that do NOT accept `context`:** `health_check`, `get_server_info`, `get_schema_types`, `list_contexts`, `validate_subject_name`, and all KEK/DEK, exporter, and admin tools.

Example usage:

```json
// Register a schema in the "staging" context
{"subject": "orders-value", "schema": "...", "context": ".staging"}

// Query only schemas in the "staging" context
{"subject": "orders-value", "context": ".staging"}

// List subjects in the default context (no context parameter needed)
{}
```

### Read-Only Mode

When `mcp.read_only: true`, only tools annotated with `ReadOnlyHint: true` are registered. All write/delete tools are hidden from discovery and blocked from execution.

## Resources

The MCP server exposes a rich set of resources — data endpoints that AI clients can read directly without calling tools.

### Static Resources

| URI | Name | Description |
|-----|------|-------------|
| `schema://server/info` | `server-info` | Server version, commit, build time, supported types |
| `schema://server/config` | `server-config` | Global compatibility level and mode |
| `schema://subjects` | `subjects` | List of all registered subjects |
| `schema://types` | `schema-types` | Supported schema types |
| `schema://contexts` | `contexts` | List of all registry contexts |
| `schema://mode` | `global-mode` | Global registry mode |
| `schema://keks` | `keks` | List of all KEKs |
| `schema://exporters` | `exporters` | List of all exporter names |
| `schema://status` | `server-status` | Server health and storage connectivity |
| `schema://glossary/core-concepts` | `glossary-core-concepts` | Schema registry fundamentals: subjects, versions, IDs, modes, naming strategies |
| `schema://glossary/compatibility` | `glossary-compatibility` | All 7 compatibility modes, per-format rules, transitive semantics |
| `schema://glossary/data-contracts` | `glossary-data-contracts` | Metadata, tags, rulesets, 3-layer merge, optimistic concurrency |
| `schema://glossary/encryption` | `glossary-encryption` | CSFLE: envelope encryption, KEK/DEK model, KMS providers, algorithms |
| `schema://glossary/contexts` | `glossary-contexts` | Multi-tenancy, default context, __GLOBAL, 4-tier inheritance |
| `schema://glossary/exporters` | `glossary-exporters` | Schema linking, exporter lifecycle, context types |
| `schema://glossary/schema-types` | `glossary-schema-types` | Avro, Protobuf, JSON Schema deep reference |
| `schema://glossary/design-patterns` | `glossary-design-patterns` | Event envelope, entity lifecycle, snapshot vs delta, shared types |
| `schema://glossary/best-practices` | `glossary-best-practices` | Per-format guidance, naming, evolution, anti-patterns |
| `schema://glossary/migration` | `glossary-migration` | Confluent migration, IMPORT mode, ID preservation |
| `schema://glossary/mcp-configuration` | `glossary-mcp-configuration` | MCP server config, permissions, security |
| `schema://glossary/error-reference` | `glossary-error-reference` | All error codes, response formats, diagnostic guidance |
| `schema://glossary/auth-and-security` | `glossary-auth-and-security` | RBAC roles, auth methods, rate limiting, audit logging |
| `schema://glossary/storage-backends` | `glossary-storage-backends` | PostgreSQL, MySQL, Cassandra characteristics and trade-offs |
| `schema://glossary/normalization-and-fingerprinting` | `glossary-normalization` | Canonical forms, deduplication, metadata identity |
| `schema://glossary/tool-selection-guide` | `glossary-tool-selection` | Decision tree for choosing the right tool |

### Templated Resources

| URI Template | Name | Description |
|-------------|------|-------------|
| `schema://subjects/{subject}` | `subject-detail` | Subject details: latest version, type, config |
| `schema://subjects/{subject}/versions` | `subject-versions` | All version numbers for a subject |
| `schema://subjects/{subject}/versions/{version}` | `subject-version-detail` | Schema at a specific version |
| `schema://subjects/{subject}/config` | `subject-config` | Per-subject compatibility config |
| `schema://subjects/{subject}/mode` | `subject-mode` | Per-subject registry mode |
| `schema://schemas/{id}` | `schema-by-id` | Schema by global ID |
| `schema://schemas/{id}/subjects` | `schema-subjects` | Subjects using a schema ID |
| `schema://schemas/{id}/versions` | `schema-versions` | Subject-version pairs for a schema ID |
| `schema://exporters/{name}` | `exporter-detail` | Exporter details by name |
| `schema://keks/{name}` | `kek-detail` | KEK details by name |
| `schema://keks/{name}/deks` | `kek-deks` | DEK subjects under a KEK |
| `schema://contexts/{context}/subjects` | `context-subjects` | Subjects in a specific context |
| `schema://contexts/{context}/config` | `context-config` | Global config/mode for a specific context |
| `schema://contexts/{context}/mode` | `context-mode` | Global mode for a specific context |
| `schema://contexts/{context}/subjects/{subject}` | `context-subject-detail` | Subject details within a context |
| `schema://contexts/{context}/subjects/{subject}/versions` | `context-subject-versions` | Subject versions within a context |
| `schema://contexts/{context}/subjects/{subject}/versions/{version}` | `context-subject-version-detail` | Schema at version within a context |
| `schema://contexts/{context}/subjects/{subject}/config` | `context-subject-config` | Subject config within a context |
| `schema://contexts/{context}/subjects/{subject}/mode` | `context-subject-mode` | Subject mode within a context |
| `schema://contexts/{context}/schemas/{id}` | `context-schema-by-id` | Schema by ID within a context |
| `schema://contexts/{context}/schemas/{id}/subjects` | `context-schema-subjects` | Schema subjects within a context |
| `schema://contexts/{context}/schemas/{id}/versions` | `context-schema-versions` | Schema versions within a context |

## Prompts

The MCP server provides a library of prompts — guided workflows that AI assistants can use to walk users through complex operations.

> The MCP server also returns **server instructions** during the `initialize` handshake, providing AI clients with capabilities overview, glossary resource URIs, and critical rules for schema registry operations.

### Schema Design & Evolution

| Prompt | Required Args | Optional Args | Description |
|--------|---------------|---------------|-------------|
| `design-schema` | `format` (AVRO, PROTOBUF, JSON) | `domain` | Guide for designing a new schema with format-specific best practices |
| `evolve-schema` | `subject` | `context` | Step-by-step workflow for safely evolving an existing schema |
| `plan-breaking-change` | `subject` | `context` | Plan a breaking schema change with 3 migration strategies |
| `compare-formats` | `use_case` | — | Compare Avro, Protobuf, and JSON Schema for a specific use case |

### Compatibility & Quality

| Prompt | Required Args | Optional Args | Description |
|--------|---------------|---------------|-------------|
| `check-compatibility` | `subject` | `context` | Troubleshoot compatibility issues with debugging workflow |
| `review-schema-quality` | `subject` | `context` | Analyze naming, nullability, documentation, and best practices |
| `schema-impact-analysis` | `subject` | `context` | 5-step workflow: dependents, field usage, validation, rollout plan |

### Operations & Configuration

| Prompt | Required Args | Optional Args | Description |
|--------|---------------|---------------|-------------|
| `migrate-schemas` | `source_format`, `target_format` | — | Guide for migrating between schema formats |
| `setup-encryption` | `kms_type` (aws-kms, azure-kms, gcp-kms, hcvault) | — | Set up CSFLE with KEK/DEK for a specific KMS provider |
| `configure-exporter` | `exporter_type` | — | Configure schema linking via exporters |
| `setup-data-contracts` | `subject` | `context` | Add metadata, tags, and data quality rules |
| `audit-subject-history` | `subject` | `context` | Review version history and evolution of a subject |

### Getting Started & Troubleshooting

| Prompt | Required Args | Optional Args | Description |
|--------|---------------|---------------|-------------|
| `schema-getting-started` | — | — | Quick-start guide for new users |
| `troubleshooting` | — | — | Diagnostic guide for common issues |
| `debug-registration-error` | `error_code` | — | Debug a specific error code (42201, 409, 40401, etc.) |
| `schema-naming-conventions` | — | — | Subject naming strategies and best practices |
| `context-management` | — | — | Multi-tenant contexts and 4-tier config inheritance |

### Domain Knowledge & Guided Workflows

| Prompt | Required Args | Optional Args | Description |
|--------|---------------|---------------|-------------|
| `glossary-lookup` | `topic` | — | Maps a keyword to the relevant glossary resource URI |
| `import-from-confluent` | — | — | Step-by-step Confluent migration with tool names |
| `setup-rbac` | — | — | Auth/RBAC configuration guide with 4 roles |
| `schema-references-guide` | — | — | Cross-subject references with per-format name semantics |
| `full-encryption-lifecycle` | — | — | End-to-end CSFLE: KEK creation, DEK management, rotation, rewrap |
| `data-rules-deep-dive` | — | — | Comprehensive data contract rules with examples |
| `registry-health-audit` | — | — | Multi-step registry health check procedure |
| `schema-evolution-cookbook` | — | — | Practical recipes for common evolution scenarios |

### Onboarding & Governance

| Prompt | Required Args | Optional Args | Description |
|--------|---------------|---------------|-------------|
| `new-kafka-topic` | `topic_name` | `format` | End-to-end Kafka topic schema setup |
| `debug-deserialization` | — | — | Consumer deserialization troubleshooting |
| `deprecate-subject` | `subject` | `context` | Subject deprecation workflow |
| `cicd-integration` | — | — | CI/CD pipeline integration guide |
| `team-onboarding` | `team_name` | — | Team onboarding workflow with contexts |
| `governance-setup` | — | — | Schema governance and quality gates |
| `cross-cutting-change` | `field_name` | — | Cross-cutting field change workflow |
| `schema-review-checklist` | `subject` | `context` | Pre-registration review checklist |

## Security

### Bearer Token Authentication

Set `mcp.auth_token` to require authentication:

```yaml
mcp:
  auth_token: "my-secret-token"
```

Clients MUST include `Authorization: Bearer my-secret-token` in their MCP HTTP requests. Requests without a valid token receive `401 Unauthorized`.

### Origin Validation

The MCP server validates the `Origin` header to prevent DNS rebinding attacks (per the MCP specification). Default allowed origins are localhost only:

```yaml
mcp:
  allowed_origins:
    - "http://localhost:*"
    - "https://localhost:*"
    - "vscode-webview://*"
```

Patterns support wildcards (`*`) for port matching. Non-browser clients (no `Origin` header) are always allowed.

### Read-Only Mode

Restrict the MCP server to read-only operations:

```yaml
mcp:
  read_only: true
```

This hides all write/delete tools from discovery and blocks their execution. Only read-only tools remain available.

### Permission Scopes

Permission scopes provide RBAC-style access control for MCP tools. Scopes mirror the REST RBAC taxonomy:

`schema_read`, `schema_write`, `schema_delete`, `config_read`, `config_write`, `mode_read`, `mode_write`, `import`, `encryption_read`, `encryption_write`, `exporter_read`, `exporter_write`, `admin_read`, `admin_write`

5 named presets combine scopes for common use cases:

| Preset | Included Scopes |
|--------|----------------|
| `readonly` | schema_read, config_read, mode_read, encryption_read, exporter_read |
| `developer` | readonly + schema_write, config_write |
| `operator` | developer + schema_delete, mode_write, encryption_write, exporter_write, import |
| `admin` | operator + admin_read, admin_write |
| `full` | All 14 scopes (default) |

System tools (`health_check`, `get_server_info`, `get_server_version`, `get_cluster_id`, `get_schema_types`, `list_contexts`, `count_subjects`, `get_registry_statistics`) are always available regardless of preset.

```yaml
# Use a preset
mcp:
  permission_preset: developer

# Or specify individual scopes
mcp:
  permission_scopes:
    - schema_read
    - encryption_write
```

Environment variables: `SCHEMA_REGISTRY_MCP_PERMISSION_PRESET`, `SCHEMA_REGISTRY_MCP_PERMISSION_SCOPES` (comma-separated).

Resolution order: `permission_preset` > `permission_scopes` > `read_only` > `tool_policy` > default (`full`).

### Tool Policies

Fine-grained control over which tools are available:

```yaml
# Block specific tools
mcp:
  tool_policy: deny_list
  denied_tools:
    - delete_subject
    - delete_version
    - import_schemas

# Allow only specific tools
mcp:
  tool_policy: allow_list
  allowed_tools:
    - list_subjects
    - get_schema_version
    - check_compatibility
```

### Two-Phase Confirmations

For destructive operations, enable two-phase confirmations to require explicit user approval:

```yaml
mcp:
  require_confirmations: true
  confirmation_ttl_secs: 300  # 5 minutes
```

When enabled, 8 destructive tools require a dry-run preview before execution:

| Tool | Condition |
|------|-----------|
| `delete_subject` | When `permanent: true` |
| `delete_version` | When `permanent: true` |
| `import_schemas` | Always |
| `set_mode` | When `mode: IMPORT` |
| `delete_config` | When deleting global config |
| `delete_kek` | When `permanent: true` |
| `delete_dek` | When `permanent: true` |
| `delete_exporter` | Always |

**Flow:**

1. Call tool with `dry_run: true` → Returns preview and a `confirm_token`
2. Call tool with `confirm_token: "<token>"` → Executes the operation

Tokens are single-use, scoped to the exact operation, and expire after the configured TTL.

### Credential Protection

- Schema bodies are NOT logged by default (`mcp.log_schemas: false`)
- API key secrets, DEK key material, and auth tokens are never included in log output
- The MCP server binds to `127.0.0.1` by default — not exposed to the network

## Schema Intelligence

The MCP server includes deterministic, rule-based intelligence tools designed to support AI-assisted schema management workflows. These tools require no external AI services — all analysis is computed locally using the registry's own schema data.

> All schema intelligence capabilities are also available as [REST API endpoints](api-reference.md#axonops-extensions) for use in CI/CD pipelines, custom tooling, and programmatic access.

### Field Search (`find_schemas_by_field`)

Find schemas containing a specific field across the entire registry:

- **Exact mode**: Matches field name with automatic naming convention normalization (`userId` matches `user_id`, `UserID`, etc.)
- **Fuzzy mode**: Uses Levenshtein distance with configurable threshold (default: 0.6)
- **Regex mode**: RE2 syntax for safe, linear-time pattern matching

### Type Search (`find_schemas_by_type`)

Find schemas by field type patterns. Matches Avro types (`long`, `string`, `["null","string"]`), Protobuf types (`int64`, `google.protobuf.Timestamp`), and JSON Schema types (`integer`, `string`).

### Schema Similarity (`find_similar_schemas`)

Computes Jaccard similarity coefficient between schema field sets. Returns similar schemas ranked by similarity score with shared field lists.

### Schema Quality Scoring (`score_schema_quality`)

Scores schemas across 6 categories on a 0-100 scale:

| Category | What It Measures |
|----------|-----------------|
| **Naming** | Consistent naming conventions (snake_case, camelCase) |
| **Documentation** | Presence of `doc` fields and descriptions |
| **Type safety** | Use of specific types vs generic strings |
| **Evolution readiness** | Optional fields with defaults for backward compatibility |
| **Structure** | Nesting depth, field count, complexity |
| **Compatibility readiness** | Patterns that support all compatibility modes |

Grades: A (90-100), B (75-89), C (60-74), D (below 60).

### Field Consistency (`check_field_consistency`)

Checks whether a field has consistent types across all schemas in the registry. Detects naming variant inconsistencies (e.g., `user_id` as `long` in one schema and `string` in another).

### Schema Complexity (`get_schema_complexity`)

Computes complexity metrics:

- **Field count** — Total number of fields
- **Max depth** — Maximum nesting depth
- **Grade** — A (simple), B (moderate), C (complex), D (very complex)

### Pattern Detection (`detect_schema_patterns`)

Identifies common field groups that appear across multiple subjects. Useful for discovering shared entities (e.g., `id`, `created_at`, `updated_at` appearing in many schemas) and refactoring candidates.

### Evolution Suggestions (`suggest_schema_evolution`)

Provides compatibility-level-aware guidance for schema changes:

- For BACKWARD: Add fields with defaults, don't remove fields
- For FORWARD: Remove fields, don't add required fields
- For FULL: Only add optional fields with defaults

### Migration Planning (`plan_migration_path`)

Given a current schema and a target schema, computes the minimum set of compatible migration steps to evolve from one to the other, respecting the subject's compatibility level.

### Design Principles

All intelligence tools follow these principles:

- **Deterministic**: Same input always produces the same output (no LLM in the loop)
- **Bounded**: Max 50 results per call, max 100 schemas scanned
- **Safe regex**: RE2 syntax guarantees linear-time execution, safe for LLM-generated patterns
- **Naming normalization**: Automatic camelCase/snake_case/PascalCase equivalence

## Connecting AI Clients

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%/Claude/claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "schema-registry": {
      "url": "http://localhost:9081/mcp"
    }
  }
}
```

With authentication:

```json
{
  "mcpServers": {
    "schema-registry": {
      "url": "http://localhost:9081/mcp",
      "headers": {
        "Authorization": "Bearer my-secret-token"
      }
    }
  }
}
```

### Claude Code

```bash
claude mcp add schema-registry http://localhost:9081/mcp
```

### VS Code / Cursor

Create `.vscode/mcp.json` in your project:

```json
{
  "servers": {
    "schema-registry": {
      "type": "http",
      "url": "http://localhost:9081/mcp"
    }
  }
}
```

### Programmatic Access (Go)

```go
import "github.com/modelcontextprotocol/go-sdk/client"

c := client.NewStreamableHTTPClient("http://localhost:9081/mcp")
session, _ := c.Connect(ctx)
result, _ := session.CallTool(ctx, "list_subjects", map[string]any{})
```

## Troubleshooting

### MCP server not starting

Verify `mcp.enabled: true` in your config and check logs for bind errors. The default port is 9081 — ensure it is not in use.

### 401 Unauthorized

If `mcp.auth_token` is set, all requests MUST include the `Authorization: Bearer <token>` header.

### 403 Forbidden (origin rejected)

The `Origin` header does not match `mcp.allowed_origins`. Add your client's origin to the allowlist. Non-browser clients that omit the `Origin` header are not affected.

### Tools not appearing

- **Read-only mode**: Write tools are hidden when `mcp.read_only: true`
- **Permission scopes**: Check `permission_preset` or `permission_scopes` — tools outside the allowed scopes are hidden
- **Tool policy**: Check `tool_policy`, `allowed_tools`, and `denied_tools` configuration
- **Auth**: Admin tools (user/API key management) only appear when auth is configured

### Confirmation required

When `mcp.require_confirmations: true`, destructive operations require a two-phase flow. Call with `dry_run: true` first, then use the returned `confirm_token`.

### Schema content not in logs

Set `mcp.log_schemas: true` to include schema bodies in structured log output. Schema bodies are logged at **Debug** level under the `mcp_tool_schema_body` message key, so you MUST also set `logging.level: debug` to see them. When `log_schemas` is enabled, a truncated schema body (max 1000 characters) is also included in the `metadata` field of audit log entries. This setting is disabled by default to avoid logging sensitive data.

## API Reference

For the complete auto-generated reference with all tool parameter schemas, see [docs/mcp-reference.md](mcp-reference.md).

Regenerate it with:

```bash
make docs-mcp
```
