# Schema Compatibility Configuration

This document explains how compatibility checking works in the AxonOps Schema Registry, the seven compatibility modes, and the specific rules applied for each schema type (Avro, JSON Schema, Protobuf).

## Table of Contents

- [Overview](#overview)
- [Compatibility Modes](#compatibility-modes)
- [Transitive vs Non-Transitive](#transitive-vs-non-transitive)
- [Configuration](#configuration)
- [Avro Compatibility Rules](#avro-compatibility-rules)
- [JSON Schema Compatibility Rules](#json-schema-compatibility-rules)
- [Protobuf Compatibility Rules](#protobuf-compatibility-rules)
- [API Examples](#api-examples)
- [Troubleshooting](#troubleshooting)
- [Design Notes](#design-notes)

---

## Overview

Compatibility checking is enforced **only at registration time** — when a new schema version is registered via `POST /subjects/{subject}/versions`. It is never enforced retroactively on existing schemas.

When a new schema is submitted, the registry compares it against one or more existing schema versions (depending on the mode) and rejects the registration if the check fails with a `409 Conflict` response.

---

## Compatibility Modes

| Mode | Direction | Versions Checked | Description |
|------|-----------|-----------------|-------------|
| `NONE` | — | — | No compatibility checks. Any schema can be registered. |
| `BACKWARD` | New reads old | Latest only | New schema (reader) can read data produced by the latest schema (writer). |
| `BACKWARD_TRANSITIVE` | New reads old | All versions | New schema (reader) can read data produced by **all** previous schema versions. |
| `FORWARD` | Old reads new | Latest only | The latest schema (reader) can read data produced by the new schema (writer). |
| `FORWARD_TRANSITIVE` | Old reads new | All versions | **All** previous schema versions can read data produced by the new schema. |
| `FULL` | Both directions | Latest only | Both `BACKWARD` and `FORWARD` must pass against the latest version. |
| `FULL_TRANSITIVE` | Both directions | All versions | Both `BACKWARD_TRANSITIVE` and `FORWARD_TRANSITIVE` must pass. |

### Which Mode Should I Use?

- **`BACKWARD` (default)**: Safe for most use cases. Consumers using the new schema can read data written by the old schema. Use when you upgrade consumers before producers.
- **`FORWARD`**: Use when you upgrade producers before consumers. Old consumers can read data from the new schema.
- **`FULL`**: Use when consumers and producers may be upgraded in any order. Most restrictive — only allows changes that are safe in both directions.
- **`NONE`**: Use temporarily to register a breaking change, then restore the previous mode.
- **Transitive variants** (`_TRANSITIVE`): Use when you need guarantees across the entire history, not just against the latest version. Important for systems that may replay old data.

---

## Transitive vs Non-Transitive

Non-transitive modes only compare against the **latest** registered version. Transitive modes compare against **all** previous versions.

### Example: Why This Matters

Consider a subject with three Avro schema versions evolving under `BACKWARD`:

```
v1: { "type": "record", "name": "User", "fields": [
       {"name": "id", "type": "int"} ] }

v2: { "type": "record", "name": "User", "fields": [
       {"name": "id", "type": "int"},
       {"name": "name", "type": "string", "default": ""} ] }

v3: { "type": "record", "name": "User", "fields": [
       {"name": "id", "type": "int"},
       {"name": "name", "type": "string"},
       {"name": "email", "type": "string", "default": ""} ] }
```

- **v2** is backward compatible with **v1**: the new `name` field has a default, so a reader using v2 can read v1 data (the missing `name` gets the default value).
- **v3** is backward compatible with **v2**: the new `email` field has a default.

Under `BACKWARD` mode, v3 is only checked against v2 (latest) — **passes**.

Under `BACKWARD_TRANSITIVE`, v3 is checked against both v1 and v2. If v3 requires reading `name` (which has no default in v3's perspective when reading v1 data), it **could fail** depending on the exact schema. The key insight: transitive mode catches incompatibilities that accumulate over time.

### When Transitive Fails But Non-Transitive Passes

```
v1: { fields: [{name: "id", type: "int"}] }
v2: { fields: [{name: "id", type: "int"}, {name: "score", type: "float", default: 0.0}] }
v3: { fields: [{name: "id", type: "int"}, {name: "score", type: "double", default: 0.0}] }
```

- v2 → v1: BACKWARD compatible (score has default)
- v3 → v2: BACKWARD compatible (float→double is a valid promotion)
- v3 → v1: BACKWARD compatible (score has default, and type promotion from v1 perspective is N/A since v1 has no score)

But consider a case where v3 removes a field that v1 had:
```
v1: { fields: [{name: "id", type: "int"}, {name: "name", type: "string"}] }
v2: { fields: [{name: "id", type: "int"}, {name: "name", type: "string"}, {name: "age", type: "int", default: 0}] }
v3: { fields: [{name: "id", type: "int"}, {name: "age", type: "int", default: 0}] }
```

- `BACKWARD`: v3 vs v2 — v3 removed `name`, but since the reader (v3) doesn't expect `name`, it just ignores it. However, backward compatibility means v3 (reader) reads v2 (writer) data. The v2 writer includes `name`, which v3 ignores — this is fine. But v3 reader doesn't have `name` at all, so if v3 tries to read v1 data, the data has `name` but v3 ignores it. The real issue is if v1 data DOESN'T have `age`, v3 needs `age` to have a default — which it does.
- `BACKWARD_TRANSITIVE`: v3 vs v1 also checked — same analysis for all versions.

---

## Configuration

### Resolution Order

When a schema is registered, the effective compatibility level is resolved:

1. **Per-subject override** — set via `PUT /config/{subject}`
2. **Global API setting** — set via `PUT /config`
3. **Config file default** — `compatibility.default_level` in `config.yaml` (defaults to `BACKWARD`)

### Setting Global Compatibility

```bash
# Set global compatibility to FULL
curl -X PUT http://localhost:8081/config \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -d '{"compatibility": "FULL"}'

# Get current global compatibility
curl http://localhost:8081/config
# Response: {"compatibilityLevel": "BACKWARD"}
```

### Setting Per-Subject Compatibility

```bash
# Set per-subject override
curl -X PUT http://localhost:8081/config/orders-value \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -d '{"compatibility": "NONE"}'

# Delete per-subject override (falls back to global)
curl -X DELETE http://localhost:8081/config/orders-value
```

### Deleting Global Config

`DELETE /config` resets the global compatibility level back to `BACKWARD` (the default). This matches Confluent behavior.

---

## Avro Compatibility Rules

Avro compatibility is based on the reader/writer model defined in the [Avro specification](https://avro.apache.org/docs/current/specification/).

### Breaking vs Non-Breaking Changes

| Change | BACKWARD | FORWARD | FULL |
|--------|----------|---------|------|
| Add field **with** default | Compatible | Compatible | Compatible |
| Add field **without** default | **Incompatible** | Compatible | **Incompatible** |
| Remove field (reader has default) | Compatible | Compatible | Compatible |
| Remove field (reader has no default) | Compatible | **Incompatible** | **Incompatible** |
| Rename field | **Incompatible** | **Incompatible** | **Incompatible** |
| Change field type (incompatible) | **Incompatible** | **Incompatible** | **Incompatible** |

### Type Promotion Rules

Avro supports automatic type promotion (widening):

| From | To | Compatible? |
|------|----|------------|
| `int` | `long`, `float`, `double` | Yes |
| `long` | `float`, `double` | Yes |
| `float` | `double` | Yes |
| `string` | `bytes` | Yes (bidirectional) |
| `bytes` | `string` | Yes (bidirectional) |

**Note:** Promotion is one-directional (except string↔bytes). `long` → `int` is **not** compatible. This means type promotions are backward compatible but not forward compatible, so they fail under `FULL` mode.

### Enum Rules

| Change | BACKWARD | FORWARD |
|--------|----------|---------|
| Add symbol | Compatible | **Incompatible** (old reader doesn't know new symbol) |
| Remove symbol | **Incompatible** (new reader may expect symbol) | Compatible |
| Add symbol with `default` | Compatible | Compatible |

### Union Rules

| Change | BACKWARD |
|--------|----------|
| Add type to union | Compatible (reader accepts more types) |
| Remove type from union | **Incompatible** (reader may encounter unknown type) |

### Fixed Type Rules

- Name must match exactly
- Size must match exactly
- Any change to name or size is **incompatible**

### Examples

**Backward compatible: adding a field with default**
```json
// v1
{"type": "record", "name": "User", "fields": [
  {"name": "id", "type": "int"}
]}

// v2 (backward compatible with v1)
{"type": "record", "name": "User", "fields": [
  {"name": "id", "type": "int"},
  {"name": "email", "type": "string", "default": ""}
]}
```

**Backward incompatible: adding a field without default**
```json
// v2 (FAILS backward check against v1)
{"type": "record", "name": "User", "fields": [
  {"name": "id", "type": "int"},
  {"name": "email", "type": "string"}
]}
```
Error: `409 Conflict` — reader field `email` has no default and is missing from writer.

---

## JSON Schema Compatibility Rules

JSON Schema compatibility checks structural changes that affect data producers and consumers.

### Breaking vs Non-Breaking Changes

| Change | BACKWARD | FORWARD | FULL |
|--------|----------|---------|------|
| Add optional property | Compatible | Compatible | Compatible |
| Add required property | **Incompatible** | Compatible | **Incompatible** |
| Remove property | **Incompatible** | Compatible | **Incompatible** |
| Make optional → required | **Incompatible** | Compatible | **Incompatible** |
| Make required → optional | Compatible | **Incompatible** | **Incompatible** |

### Constraint Rules

| Change | BACKWARD | FORWARD |
|--------|----------|---------|
| Increase `minLength` | **Incompatible** (tighter) | Compatible |
| Decrease `minLength` | Compatible (looser) | **Incompatible** |
| Increase `maxLength` | Compatible (looser) | **Incompatible** |
| Decrease `maxLength` | **Incompatible** (tighter) | Compatible |
| Increase `minItems` | **Incompatible** (tighter) | Compatible |
| Decrease `maxItems` | **Incompatible** (tighter) | Compatible |
| Add `minimum`/`maximum` | **Incompatible** (tighter) | Compatible |
| Remove `minimum`/`maximum` | Compatible (looser) | **Incompatible** |

### Enum Rules

| Change | BACKWARD | FORWARD |
|--------|----------|---------|
| Add enum value | Compatible | **Incompatible** |
| Remove enum value | **Incompatible** | Compatible |
| Remove enum constraint entirely | Compatible | **Incompatible** |

### additionalProperties Rules

| Change | BACKWARD | FORWARD |
|--------|----------|---------|
| `true` → `false` | **Incompatible** | Compatible |
| `false` → `true` | Compatible | **Incompatible** |

### Type Rules

| Change | BACKWARD |
|--------|----------|
| Widen type (e.g., `"integer"` → `"number"`) | Compatible |
| Narrow type (e.g., `"number"` → `"integer"`) | **Incompatible** |
| Add type to array (e.g., `["string"]` → `["string", "null"]`) | Compatible |
| Remove type from array | **Incompatible** |

### Examples

**Backward compatible: adding an optional property**
```json
// v1
{"type": "object", "properties": {"id": {"type": "integer"}}, "required": ["id"]}

// v2 (backward compatible)
{"type": "object", "properties": {"id": {"type": "integer"}, "name": {"type": "string"}}, "required": ["id"]}
```

**Backward incompatible: adding a required property**
```json
// v2 (FAILS backward check — new required field)
{"type": "object", "properties": {"id": {"type": "integer"}, "name": {"type": "string"}}, "required": ["id", "name"]}
```

---

## Protobuf Compatibility Rules

Protobuf compatibility is wire-format oriented. Field **numbers** are the identity — not field names.

### Breaking vs Non-Breaking Changes

| Change | BACKWARD | FORWARD | FULL |
|--------|----------|---------|------|
| Add optional field | Compatible | Compatible | Compatible |
| Add required field | **Incompatible** | **Incompatible** | **Incompatible** |
| Remove field | Compatible | **Incompatible** | **Incompatible** |
| Change field type (incompatible) | **Incompatible** | **Incompatible** | **Incompatible** |
| Reuse field number | **Incompatible** | **Incompatible** | **Incompatible** |

### Cardinality Changes

| Change | BACKWARD | FORWARD |
|--------|----------|---------|
| `optional` → `repeated` | Compatible | **Incompatible** |
| `repeated` → `optional` | **Incompatible** | Compatible |
| non-required → `required` | **Incompatible** | **Incompatible** |
| `required` → `optional` | Compatible | Compatible |

### Compatible Type Groups

These types share the same wire format and can be interchanged:

| Group | Types |
|-------|-------|
| Signed 32-bit | `int32`, `sint32`, `sfixed32` |
| Signed 64-bit | `int64`, `sint64`, `sfixed64` |
| Unsigned 32-bit | `uint32`, `fixed32` |
| Unsigned 64-bit | `uint64`, `fixed64` |

**Note:** `float` ↔ `double` is **not** compatible. `int32` ↔ `int64` is **not** compatible (different wire sizes).

### Enum Rules

| Change | BACKWARD | FORWARD |
|--------|----------|---------|
| Add enum value | Compatible | **Incompatible** |
| Remove enum value | **Incompatible** | Compatible |
| Change value number | **Incompatible** | **Incompatible** |

### Service Rules

| Change | BACKWARD | FORWARD |
|--------|----------|---------|
| Add method | Compatible | **Incompatible** |
| Remove method | **Incompatible** | Compatible |
| Change method input/output type | **Incompatible** | **Incompatible** |
| Change streaming mode | **Incompatible** | **Incompatible** |

### Examples

**Backward compatible: adding an optional field**
```protobuf
// v1
syntax = "proto3";
message User {
  int32 id = 1;
}

// v2 (backward compatible)
syntax = "proto3";
message User {
  int32 id = 1;
  string name = 2;
}
```

**Backward incompatible: changing field type**
```protobuf
// v2 (FAILS — field 1 changed from int32 to string)
syntax = "proto3";
message User {
  string id = 1;
}
```

---

## API Examples

### Register a Schema

```bash
curl -X POST http://localhost:8081/subjects/orders-value/versions \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -d '{"schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}'
```

### Check Compatibility Before Registering

Test whether a schema would be compatible without actually registering it:

```bash
# Check against latest version
curl -X POST http://localhost:8081/compatibility/subjects/orders-value/versions/latest \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -d '{"schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"total\",\"type\":\"float\",\"default\":0.0}]}"}'

# Response (compatible):
# {"is_compatible": true}

# Response (incompatible):
# {"is_compatible": false, "messages": ["..."]}
```

### Check Against All Versions (Transitive)

```bash
curl -X POST http://localhost:8081/compatibility/subjects/orders-value/versions \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -d '{"schema": "..."}'
```

### Register a Breaking Change (Temporary NONE)

```bash
# 1. Save current level
curl http://localhost:8081/config/orders-value

# 2. Set to NONE
curl -X PUT http://localhost:8081/config/orders-value \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -d '{"compatibility": "NONE"}'

# 3. Register breaking schema
curl -X POST http://localhost:8081/subjects/orders-value/versions \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -d '{"schema": "..."}'

# 4. Restore compatibility level
curl -X PUT http://localhost:8081/config/orders-value \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -d '{"compatibility": "BACKWARD"}'
```

---

## Troubleshooting

### Common Errors

#### 409 Conflict — Schema is incompatible

```json
{"error_code": 409, "message": "Schema being registered is incompatible with an earlier schema; ..."}
```

**Causes:**
- Adding a field without a default (BACKWARD mode)
- Removing a field (FORWARD mode)
- Changing a field type to an incompatible type
- Reusing a Protobuf field number

**Solutions:**
1. Add a default value to new fields
2. Use the compatibility check endpoint first to test without registering
3. Temporarily set compatibility to `NONE` (see example above)

#### 42201 — Invalid schema

```json
{"error_code": 42201, "message": "Invalid AVRO schema"}
```

The schema itself is malformed or cannot be parsed. This is unrelated to compatibility — the schema must be valid before compatibility is checked.

#### 42203 — Invalid compatibility level

```json
{"error_code": 42203, "message": "Invalid compatibility level"}
```

Valid levels: `NONE`, `BACKWARD`, `BACKWARD_TRANSITIVE`, `FORWARD`, `FORWARD_TRANSITIVE`, `FULL`, `FULL_TRANSITIVE`.

### FAQ

**Q: Can I change the compatibility level after schemas are registered?**
A: Yes. Changes only affect future registrations. Existing schemas are not re-validated.

**Q: What happens when I delete the per-subject config?**
A: The subject falls back to the global config level.

**Q: What happens when I delete the global config?**
A: The global config resets to `BACKWARD` (the default).

**Q: Does compatibility check happen for the first schema version?**
A: No. The first schema version for a subject is always accepted (there's nothing to compare against).

**Q: Can I check compatibility without registering?**
A: Yes. Use `POST /compatibility/subjects/{subject}/versions/latest` to test without registering.

---

## Design Notes

### Two-Layer Default System

The system has two distinct layers:

- **Storage layer**: All backends (memory, PostgreSQL, MySQL, Cassandra) seed a default `BACKWARD` config and `READWRITE` mode at initialization. `DELETE /config` resets to `BACKWARD` rather than removing the row.
- **Registry layer**: Has a `defaultConfig` from the config file that serves as a fallback if storage returns `ErrNotFound`.

This means:
- Changing `compatibility.default_level` in the config file and restarting takes effect for all subjects without an explicit API override
- `DELETE /config` restores the `BACKWARD` default

### Risks of Changing Compatibility Levels

Because compatibility is only checked at registration time, changing levels can create inconsistent histories:

1. Subject has 3 versions under `BACKWARD`
2. Operator sets `NONE` and registers an incompatible v4
3. Operator restores `BACKWARD`
4. v5 is only checked against v4 — not v1-v3

This matches Confluent behavior. Use transitive modes to catch accumulated incompatibilities.

### Proposed Improvements (Not Yet Implemented)

**Option A: Audit Trail** — Log every compatibility level change with timestamp, old value, new value, and user. Provides accountability without preventing changes.

**Option B: Compatibility Lock** — Add a `locked` field that prevents weakening compatibility. Can only be unlocked with `?force=true`. Prevents accidental weakening by operators.
