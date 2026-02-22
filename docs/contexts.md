# Contexts (Multi-Tenancy)

Contexts provide namespace isolation within a single AxonOps Schema Registry instance. They allow multiple teams, environments, or applications to manage schemas independently without interference -- each context has its own subjects, schema IDs, compatibility configuration, and modes.

This feature is **Confluent-compatible**: context-qualified subject names and the `/contexts` API follow the same conventions used by Confluent Schema Registry's Schema Linking and multi-tenant deployments.

## Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [Default Context](#default-context)
- [Global Context (__GLOBAL)](#global-context-__global)
- [Subject Format](#subject-format)
- [URL Routing](#url-routing)
  - [Qualified Subjects (Root-Level)](#qualified-subjects-root-level)
  - [URL Prefix Routing](#url-prefix-routing)
- [Context Naming Rules](#context-naming-rules)
- [API Examples](#api-examples)
  - [List Contexts](#list-contexts)
  - [Register a Schema in a Context](#register-a-schema-in-a-context)
  - [List Subjects in a Context](#list-subjects-in-a-context)
  - [Get a Schema by ID in a Context](#get-a-schema-by-id-in-a-context)
  - [Per-Context Compatibility Configuration](#per-context-compatibility-configuration)
  - [Per-Context Mode](#per-context-mode)
  - [Delete a Subject in a Context](#delete-a-subject-in-a-context)
  - [Check Compatibility in a Context](#check-compatibility-in-a-context)
- [Isolation Guarantees](#isolation-guarantees)
- [Backward Compatibility](#backward-compatibility)
- [Related Documentation](#related-documentation)

---

## Overview

A **context** is a logical namespace inside the registry. Every subject, schema ID, compatibility level, and mode setting belongs to exactly one context. By default, all operations target the **default context** (`"."`), which means existing clients work without any changes.

Contexts are useful when you need to:

- **Isolate teams** -- give each team (e.g., `".team-a"`, `".team-b"`) an independent schema namespace so they can use the same subject names without conflicts.
- **Separate environments** -- run staging and production schemas side by side in a single registry instance (e.g., `".staging"`, `".production"`).
- **Support Schema Linking** -- Confluent Schema Linking uses contexts to replicate schemas from one cluster to another. AxonOps contexts are wire-compatible with this protocol.
- **Multi-tenant SaaS** -- provide each tenant a dedicated schema namespace within a shared registry deployment.

---

## Key Concepts

| Concept | Scope | Description |
|---------|-------|-------------|
| **Schema IDs** | Per-context | Each context maintains its own ID sequence. Schema ID `1` in `".team-a"` is independent of schema ID `1` in `".team-b"`. |
| **Subjects** | Per-context | Subject names are unique within a context. The subject `orders-value` in `".team-a"` is a different subject from `orders-value` in `".team-b"`. |
| **Compatibility config** | Per-context | Global and subject-level compatibility settings are scoped to the context. Setting `FULL` compatibility in `".team-a"` does not affect `".team-b"`. |
| **Mode** | Per-context | Read/write modes (`READWRITE`, `READONLY`, `IMPORT`) are scoped to the context. |
| **Versions** | Per-context | Version numbering is independent in each context. |

---

## Default Context

The default context is named `"."` (a single dot). When no context is specified in a request -- whether via a qualified subject name or a URL prefix -- all operations target this default context.

This ensures **full backward compatibility**: existing clients that do not use contexts continue to work exactly as before. Their schemas, subjects, and configuration all reside in the default context `"."`.

> The `GET /contexts` endpoint always includes `"."` in its response, even when no other contexts have been created.

---

## Global Context (__GLOBAL)

The `__GLOBAL` context (displayed as `.__GLOBAL`) is a special cross-context namespace reserved for configuration and mode operations that apply globally across all contexts. It is used by Confluent Schema Linking for cross-context coordination.

**Key properties:**

- Only **config** and **mode** operations are allowed under `.__GLOBAL`.
- Schemas and subjects **cannot** be registered under `.__GLOBAL`. Attempting to register a schema returns an error.
- The context is accessed using the same qualified-subject or URL-prefix patterns as any other context (e.g., `/contexts/.__GLOBAL/config`).
- `__GLOBAL` does NOT appear in the `GET /contexts` listing unless it has been explicitly configured.

> The `__GLOBAL` context exists for Confluent wire compatibility. Most deployments do not need to interact with it directly.

---

## Subject Format

Contexts use the Confluent-compatible **qualified subject name** format to embed context information directly in the subject string:

```
:.contextname:subject
```

The format consists of:

1. `:.` -- literal prefix that signals a context-qualified subject
2. `contextname` -- the context name (without the leading dot used in display form)
3. `:` -- separator
4. `subject` -- the actual subject name

**Examples:**

| Qualified Subject | Context (Display Form) | Subject |
|-------------------|----------------------|---------|
| `:.team-a:orders-value` | `.team-a` | `orders-value` |
| `:.production:users-key` | `.production` | `users-key` |
| `:.staging:payments-value` | `.staging` | `payments-value` |
| `orders-value` | `.` (default) | `orders-value` |

When a qualified subject is used in a request, the registry extracts the context and subject automatically. When the context is the default (`"."`), no prefix is added -- the subject is returned as a plain string.

This format round-trips correctly: parsing `:.production:my-topic` yields context `.production` and subject `my-topic`, and formatting them back produces `:.production:my-topic`.

---

## URL Routing

There are two ways to interact with context-scoped data. Both are fully supported and produce identical results.

### Qualified Subjects (Root-Level)

Embed the context in the subject name parameter using the `:.contextname:subject` format. The request goes to the standard root-level endpoints:

```
POST /subjects/:.team-a:orders-value/versions
GET  /subjects/:.team-a:orders-value/versions/latest
GET  /config/:.team-a:orders-value
```

This approach is useful when you want to target a specific context in a single request without changing your base URL. The **qualified subject takes precedence** over any URL prefix context.

### URL Prefix Routing

Use the `/contexts/{context}/` URL prefix to scope all operations to a specific context. The subject parameter is a plain (unqualified) name:

```
POST /contexts/.team-a/subjects/orders-value/versions
GET  /contexts/.team-a/subjects/orders-value/versions/latest
GET  /contexts/.team-a/config/orders-value
```

All standard registry endpoints are available under the `/contexts/{context}` prefix. The context name in the URL MUST include the leading dot (e.g., `.team-a`, not `team-a`). If the name is provided without the leading dot, the registry normalizes it by prepending one.

The full set of context-prefixed endpoints mirrors the root-level routes:

| Root-Level Endpoint | Context-Prefixed Equivalent |
|--------------------|-----------------------------|
| `GET /subjects` | `GET /contexts/{context}/subjects` |
| `POST /subjects/{subject}/versions` | `POST /contexts/{context}/subjects/{subject}/versions` |
| `GET /schemas/ids/{id}` | `GET /contexts/{context}/schemas/ids/{id}` |
| `GET /config` | `GET /contexts/{context}/config` |
| `PUT /mode` | `PUT /contexts/{context}/mode` |
| `POST /compatibility/subjects/{subject}/versions/{version}` | `POST /contexts/{context}/compatibility/subjects/{subject}/versions/{version}` |
| `POST /import/schemas` | `POST /contexts/{context}/import/schemas` |

> When both a URL prefix context and a qualified subject name are present, the **qualified subject takes precedence**. For example, `POST /contexts/.team-a/subjects/:.team-b:my-subject/versions` targets context `.team-b`, not `.team-a`.

---

## Context Naming Rules

Context names MUST conform to the following rules:

| Rule | Detail |
|------|--------|
| **Character set** | Alphanumeric characters (`a-z`, `A-Z`, `0-9`), hyphens (`-`), underscores (`_`), and dots (`.`). |
| **Pattern** | `^[a-zA-Z0-9._-]+$` |
| **Maximum length** | 255 characters. |
| **Case sensitivity** | Context names are **case-sensitive**. `.TeamA` and `.teama` are different contexts. |
| **Leading dot** | Context names in display form include a leading dot (e.g., `.team-a`). If a name is provided without the leading dot, the registry normalizes it automatically. |
| **Default context** | The name `"."` (a single dot) is reserved for the default context. |

**Valid names:** `.team-a`, `.production`, `.my_context`, `.ctx.v2`, `.ABC`, `.a1b2`

**Invalid names:** names containing spaces, slashes (`/`), at signs (`@`), exclamation marks (`!`), or other special characters; empty strings; strings longer than 255 characters.

---

## API Examples

All examples use `curl` against a registry running at `http://localhost:8081`.

### List Contexts

Returns all context names that have at least one subject or that have been configured. The default context `"."` is always included.

```bash
curl http://localhost:8081/contexts
```

```json
[".", ".team-a", ".team-b"]
```

### Register a Schema in a Context

**Using a qualified subject:**

```bash
curl -X POST http://localhost:8081/subjects/:.team-a:orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"amount\",\"type\":\"double\"}]}"
  }'
```

**Using URL prefix routing:**

```bash
curl -X POST http://localhost:8081/contexts/.team-a/subjects/orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"amount\",\"type\":\"double\"}]}"
  }'
```

Both produce the same response:

```json
{"id": 1}
```

### List Subjects in a Context

**Using qualified subject format** (returns qualified names at root level):

```bash
curl http://localhost:8081/subjects
```

```json
["orders-value", ":.team-a:orders-value", ":.team-b:users-value"]
```

**Using URL prefix routing** (returns plain names within the context):

```bash
curl http://localhost:8081/contexts/.team-a/subjects
```

```json
["orders-value"]
```

### Get a Schema by ID in a Context

Schema IDs are context-scoped. To retrieve schema ID `1` from the `.team-a` context:

```bash
curl http://localhost:8081/contexts/.team-a/schemas/ids/1
```

```json
{
  "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"amount\",\"type\":\"double\"}]}"
}
```

Schema ID `1` in the default context MAY contain a completely different schema.

### Per-Context Compatibility Configuration

Set the compatibility level for a subject within a context:

```bash
# Set compatibility for orders-value in .team-a
curl -X PUT http://localhost:8081/config/:.team-a:orders-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "FULL"}'
```

Or using URL prefix routing:

```bash
curl -X PUT http://localhost:8081/contexts/.team-a/config/orders-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "FULL"}'
```

Set the **global** compatibility level within a context (applies to all subjects in that context that do not have a subject-level override):

```bash
curl -X PUT http://localhost:8081/contexts/.team-a/config \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "BACKWARD_TRANSITIVE"}'
```

### Per-Context Mode

Set the mode for a context (e.g., to enable bulk import):

```bash
curl -X PUT http://localhost:8081/contexts/.team-a/mode \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"mode": "IMPORT"}'
```

### Delete a Subject in a Context

**Soft delete:**

```bash
curl -X DELETE http://localhost:8081/subjects/:.team-a:orders-value
```

**Permanent delete:**

```bash
curl -X DELETE "http://localhost:8081/subjects/:.team-a:orders-value?permanent=true"
```

### Check Compatibility in a Context

Check whether a new schema is compatible with the latest version of a subject in a specific context:

```bash
curl -X POST http://localhost:8081/contexts/.team-a/compatibility/subjects/orders-value/versions/latest \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"amount\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
  }'
```

```json
{"is_compatible": true}
```

---

## Isolation Guarantees

Contexts provide the following isolation properties:

- **Schema IDs are independent.** Each context maintains its own auto-incrementing ID sequence. Creating a schema in `.team-a` does not consume an ID in `.team-b`.
- **Subjects are independent.** The same subject name in different contexts refers to different subjects with separate version histories.
- **Deletes do not cross contexts.** Deleting a subject in one context has no effect on subjects with the same name in other contexts.
- **Compatibility configuration is independent.** Global and subject-level compatibility settings are scoped to their context. Changing the global config in `.team-a` does not alter the compatibility rules in `.team-b` or in the default context.
- **Mode is independent.** Setting a context to `IMPORT` mode does not affect the read/write mode of other contexts.
- **Schema content is not shared across contexts.** Even if two contexts register identical schema content, they receive independent schema IDs and maintain separate version histories. Schema deduplication (fingerprint-based) operates within a context.

---

## Backward Compatibility

The contexts feature is fully backward compatible with existing clients and deployments:

- Clients that do not use context-qualified subjects or URL prefix routing continue to operate in the default context (`"."`). No changes are REQUIRED.
- The `GET /contexts` endpoint returns `["."]` when only the default context is in use.
- Upgrading from a version without context support preserves all existing data in the default context.
- All existing API endpoints, error codes, and response formats remain unchanged.

---

## Related Documentation

- [API Reference](api-reference.md) -- complete endpoint documentation including context-scoped routes
- [Configuration](configuration.md) -- YAML configuration reference
- [Compatibility](compatibility.md) -- compatibility modes and per-subject configuration
- [Migration](migration.md) -- migrating from Confluent Schema Registry
- [Fundamentals](fundamentals.md) -- core schema registry concepts
