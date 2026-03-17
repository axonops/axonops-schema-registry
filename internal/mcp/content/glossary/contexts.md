# Contexts (Multi-Tenancy)

## Overview

A **context** is a logical namespace within the registry that provides multi-tenant isolation. Each context operates as an independent schema registry -- with its own schema IDs, subjects, version histories, compatibility configuration, and modes -- while sharing a single registry deployment.

## Key Concepts

| Concept | Details |
|---------|---------|
| **Default context** | "." (dot). All operations target this context unless specified otherwise. |
| **Global context** | "__GLOBAL". Used for cross-context configuration and mode settings. |
| **Qualified subjects** | Format: :.contextname:subject. Embeds context in the subject string. |
| **URL prefix routing** | /contexts/{context}/subjects/... scopes operations to a context. |

## What Contexts Isolate

| Resource | Isolation |
|----------|-----------|
| **Schema IDs** | Each context has its own auto-incrementing ID sequence. ID 1 in .team-a is independent of ID 1 in .team-b. |
| **Subjects** | Same subject name in different contexts = different subjects with separate version histories. |
| **Versions** | Version numbering is independent per context. |
| **Compatibility config** | Global and per-subject settings are scoped to the context. |
| **Modes** | READWRITE/READONLY/IMPORT modes are scoped to the context. |

## Accessing Contexts

Two equivalent methods:

**Qualified subject names:**
    :.team-a:orders-value
    POST /subjects/:.team-a:orders-value/versions

**URL prefix routing:**
    POST /contexts/.team-a/subjects/orders-value/versions

Both produce identical results.

## Context Naming Rules

- Alphanumeric characters, hyphens, underscores, and dots.
- Maximum 255 characters.
- Context names are prefixed with a dot in listings (e.g., ".team-a", ".staging").
- The default context "." is always present.

## The 4-Tier Config/Mode Inheritance Chain

Configuration and mode settings cascade through 4 levels (highest to lowest precedence):

1. **Per-subject** -- most specific, overrides everything below. Set via set_config/set_mode with a subject.
2. **Context global** -- per-context default. Set via set_config/set_mode with no subject within the context.
3. **Global (__GLOBAL)** -- cross-context default. Set via set_config/set_mode in the __GLOBAL context.
4. **Server default** -- hardcoded: BACKWARD compatibility, READWRITE mode.

To check the effective (resolved) config: **get_config** with a subject returns the resolved value after walking the chain.

## Common Use Cases

| Use Case | Description |
|----------|-------------|
| **Team isolation** | .team-a and .team-b get independent namespaces |
| **Environment separation** | .staging and .production schemas side by side |
| **Schema Linking** | Confluent Schema Linking uses contexts for cross-cluster replication |
| **Multi-tenant SaaS** | Each tenant gets a dedicated namespace |

## Context Support in MCP

### Tools (78+ accept `context`)

All schema, config, and mode tools accept an optional `context` parameter. Tools that do NOT accept context are system tools: `health_check`, `get_server_info`, `get_server_version`, `get_cluster_id`, `get_schema_types`, `get_registry_statistics`.

**Example:**
```json
{"subject": "orders-value", "schema": "{\"type\":\"string\"}", "context": ".staging"}
```

### Context-Scoped Resources (11 URI templates)

| URI Template | Description |
|-------------|-------------|
| `schema://contexts/{context}/subjects` | Subjects in a specific context |
| `schema://contexts/{context}/config` | Global config for a context |
| `schema://contexts/{context}/mode` | Global mode for a context |
| `schema://contexts/{context}/subjects/{subject}` | Subject details within a context |
| `schema://contexts/{context}/subjects/{subject}/versions` | Subject versions within a context |
| `schema://contexts/{context}/subjects/{subject}/versions/{version}` | Version detail within a context |
| `schema://contexts/{context}/subjects/{subject}/config` | Subject config within a context |
| `schema://contexts/{context}/subjects/{subject}/mode` | Subject mode within a context |
| `schema://contexts/{context}/schemas/{id}` | Schema by ID within a context |
| `schema://contexts/{context}/schemas/{id}/subjects` | Schema subjects within a context |
| `schema://contexts/{context}/schemas/{id}/versions` | Schema versions within a context |

Non-context-prefixed URIs (e.g., `schema://subjects`) default to the default context.

### Prompts (7 accept `context` argument)

`evolve-schema`, `check-compatibility`, `review-schema-quality`, `plan-breaking-change`, `setup-data-contracts`, `audit-subject-history`, `schema-impact-analysis`

When a `context` argument is provided, these prompts enrich their responses with data from the specified context.

## MCP Tools

- **list_contexts** -- list all contexts
- **get_config / set_config / delete_config** -- manage compatibility per context/subject
- **get_mode / set_mode / delete_mode** -- manage modes per context/subject
- **list_subjects** -- list subjects in a context
