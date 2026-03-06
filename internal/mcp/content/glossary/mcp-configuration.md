# MCP Configuration

## Overview

The MCP (Model Context Protocol) server runs alongside the REST API on a separate port (default 9081). It provides AI assistants with structured access to schema registry operations via tools, resources, and prompts.

## Configuration Fields

All fields are under the `mcp:` section in the YAML config file.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable the MCP server |
| `host` | string | `127.0.0.1` | Listen address (localhost by default for security) |
| `port` | int | `9081` | Listen port |
| `auth_token` | string | `""` | Bearer token for MCP endpoint authentication |
| `read_only` | bool | `false` | Restrict to read-only tools (hides write tools from clients) |
| `tool_policy` | string | `allow_all` | Tool visibility: `allow_all`, `deny_list`, or `allow_list` |
| `allowed_tools` | []string | `[]` | Tools to allow (only used when `tool_policy: allow_list`) |
| `denied_tools` | []string | `[]` | Tools to deny (only used when `tool_policy: deny_list`) |
| `allowed_origins` | []string | localhost + vscode | Origin header allowlist for CORS/security |
| `require_confirmations` | bool | `false` | Enable two-phase confirmations for destructive operations |
| `confirmation_ttl` | int | `300` | Confirmation token TTL in seconds |
| `log_schemas` | bool | `false` | Log full schema bodies in debug output and audit trail |
| `permission_preset` | string | `""` | Named permission preset: `readonly`, `developer`, `operator`, `admin`, `full` |
| `permission_scopes` | []string | `[]` | Individual permission scopes (when preset is empty) |

## Environment Variable Overrides

Every MCP config field can be overridden via environment variables:

| Environment Variable | Config Field |
|---------------------|--------------|
| `SCHEMA_REGISTRY_MCP_ENABLED` | `enabled` |
| `SCHEMA_REGISTRY_MCP_HOST` | `host` |
| `SCHEMA_REGISTRY_MCP_PORT` | `port` |
| `SCHEMA_REGISTRY_MCP_AUTH_TOKEN` | `auth_token` |
| `SCHEMA_REGISTRY_MCP_READ_ONLY` | `read_only` |
| `SCHEMA_REGISTRY_MCP_ALLOWED_ORIGINS` | `allowed_origins` (comma-separated) |
| `SCHEMA_REGISTRY_MCP_REQUIRE_CONFIRMATIONS` | `require_confirmations` |
| `SCHEMA_REGISTRY_MCP_CONFIRMATION_TTL` | `confirmation_ttl` |
| `SCHEMA_REGISTRY_MCP_LOG_SCHEMAS` | `log_schemas` |
| `SCHEMA_REGISTRY_MCP_PERMISSION_PRESET` | `permission_preset` |
| `SCHEMA_REGISTRY_MCP_PERMISSION_SCOPES` | `permission_scopes` (comma-separated) |

## Read-Only Mode

When `read_only: true`, only tools annotated with `ReadOnlyHint: true` are registered. Write tools (register_schema, delete_subject, set_config, etc.) are completely hidden from MCP clients -- they do not appear in the tool listing.

This is ideal for production environments where AI assistants should only observe, not modify.

## Tool Policy

The tool policy controls which tools are visible to MCP clients:

- **`allow_all`** (default): All tools are registered.
- **`deny_list`**: All tools EXCEPT those in `denied_tools` are registered.
- **`allow_list`**: ONLY tools in `allowed_tools` are registered.

## Permission Scopes

Permission scopes provide fine-grained control over which categories of tools are available. There are 14 scopes mirroring the REST RBAC taxonomy:

`schema_read`, `schema_write`, `schema_delete`, `config_read`, `config_write`, `mode_read`, `mode_write`, `import`, `encryption_read`, `encryption_write`, `exporter_read`, `exporter_write`, `admin_read`, `admin_write`

**Resolution order:**
1. If `permission_preset` is set, expand to preset scopes
2. Else if `permission_scopes` is non-empty, use listed scopes
3. Else if `read_only: true`, equivalent to `readonly` preset
4. Else fall back to existing `tool_policy` / `allowed_tools` / `denied_tools`
5. Default (nothing configured): `full` (all tools, backward compatible)

## Named Presets

| Preset | Scopes |
|--------|--------|
| `readonly` | schema_read, config_read, mode_read, encryption_read, exporter_read |
| `developer` | readonly + schema_write, config_write |
| `operator` | developer + schema_delete, mode_write, encryption_write, exporter_write, import |
| `admin` | operator + admin_read, admin_write |
| `full` | All 14 scopes (default) |

## Two-Phase Confirmations

When `require_confirmations: true`, 8 destructive operations require a two-step flow:

1. **Dry run:** Call the tool normally. Instead of executing, it returns a `confirm_token` and a description of what would happen.
2. **Confirm:** Call the tool again with `confirm_token` to execute.

Tokens are single-use and expire after `confirmation_ttl` seconds (default 300).

**Affected tools:** delete_subject, delete_version, set_mode (IMPORT/READONLY), set_config (NONE), delete_config, import_schemas, delete_kek, delete_dek.

## Origin Validation

The `allowed_origins` list controls which HTTP Origin headers are accepted. Supports wildcards (`*` in hostname/port). Default allows localhost and VS Code webviews.

Set to `["*"]` to allow all origins (not recommended for production).

## Context Parameter

All schema/config/mode tools accept an optional `context` parameter for multi-tenant isolation. Context-scoped resources are available at `schema://contexts/{context}/...`. See the contexts glossary for details.

## Example Configurations

**Read-only AI assistant:**
```yaml
mcp:
  enabled: true
  read_only: true
  auth_token: "my-secret-token"
```

**Developer AI with permission scopes:**
```yaml
mcp:
  enabled: true
  permission_preset: developer
  auth_token: "my-secret-token"
```

**Restricted tool set:**
```yaml
mcp:
  enabled: true
  tool_policy: allow_list
  allowed_tools:
    - list_subjects
    - get_latest_schema
    - validate_schema
    - check_compatibility
```
