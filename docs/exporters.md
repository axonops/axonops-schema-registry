# Exporters (Schema Linking)

Exporters let you replicate schemas across registries by defining what to export, how to rename subjects, and where to send them. They provide a managed, lifecycle-driven approach to schema replication -- you create an exporter once, and the registry handles ongoing synchronization.

This feature is **Confluent-compatible**: the exporter API is wire-compatible with Confluent Schema Registry's Schema Linking (an Enterprise feature), available at no additional cost in AxonOps Schema Registry.

## Contents

- [Overview](#overview)
- [Exporter Data Model](#exporter-data-model)
- [Creating an Exporter](#creating-an-exporter)
- [Managing Exporters](#managing-exporters)
  - [List Exporters](#list-exporters)
  - [Get Exporter Details](#get-exporter-details)
  - [Update an Exporter](#update-an-exporter)
  - [Delete an Exporter](#delete-an-exporter)
- [Exporter Lifecycle](#exporter-lifecycle)
  - [Pause an Exporter](#pause-an-exporter)
  - [Resume an Exporter](#resume-an-exporter)
  - [Reset an Exporter](#reset-an-exporter)
- [Monitoring Exporter Status](#monitoring-exporter-status)
- [Exporter Configuration](#exporter-configuration)
  - [Get Exporter Config](#get-exporter-config)
  - [Update Exporter Config](#update-exporter-config)
- [Context-Scoped Exporters](#context-scoped-exporters)
- [API Reference Summary](#api-reference-summary)
- [Related Documentation](#related-documentation)

---

## Overview

Schema Linking solves a common operational problem: keeping schemas synchronized across multiple registry instances. Whether you are replicating schemas from a production registry to a disaster-recovery site, mirroring schemas between regions, or feeding a downstream analytics registry, exporters provide the configuration and lifecycle management for that replication.

An **exporter** is a named configuration object that defines:

1. **Which subjects** to export (via a filter list)
2. **How to rename** subjects at the destination (via a template)
3. **Where to send** schemas (via key-value configuration)
4. **Which context** the exporter operates within

Exporters follow a state machine model -- they can be paused, resumed, and reset -- giving operators full control over the replication pipeline.

---

## Exporter Data Model

Each exporter consists of the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier for the exporter. MUST be unique across the registry. |
| `contextType` | string | No | How the exporter resolves its context. One of `AUTO` (default), `CUSTOM`, or `NONE`. |
| `context` | string | No | Custom context path. Only used when `contextType` is `CUSTOM`. |
| `subjects` | list | No | Filter list of subject names to export. If empty, all subjects are exported. |
| `subjectRenameFormat` | string | No | Template for renaming subjects at the destination. Use `${subject}` as a placeholder for the original subject name. |
| `config` | map | No | Key-value pairs defining destination-specific configuration (e.g., connection URLs, credentials). |

**Context types:**

| Context Type | Behavior |
|-------------|----------|
| `AUTO` | The exporter automatically determines the context based on the registry's default configuration. This is the default. |
| `CUSTOM` | The exporter uses the context path specified in the `context` field. |
| `NONE` | The exporter operates without a context qualifier. |

---

## Creating an Exporter

Create an exporter by sending a `POST` request to `/exporters` with the exporter definition in the request body. The `name` field is REQUIRED.

```bash
curl -X POST http://localhost:8081/exporters \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "name": "dc-west-replica",
    "contextType": "AUTO",
    "subjects": ["orders-value", "payments-value"],
    "subjectRenameFormat": "${subject}",
    "config": {
      "schema.registry.url": "http://registry-west:8081"
    }
  }'
```

Response:

```json
{"name": "dc-west-replica"}
```

If an exporter with the same name already exists, the registry returns an error:

```json
{"error_code": 40950, "message": "Exporter already exists: dc-west-replica"}
```

> The `name` field MUST be non-empty. Omitting it or passing an empty string results in a `422 Unprocessable Entity` error.

---

## Managing Exporters

### List Exporters

Retrieve the names of all registered exporters:

```bash
curl http://localhost:8081/exporters
```

Response:

```json
["dc-west-replica", "analytics-mirror"]
```

If no exporters exist, the response is an empty array:

```json
[]
```

### Get Exporter Details

Retrieve the full configuration of a specific exporter by name:

```bash
curl http://localhost:8081/exporters/dc-west-replica
```

Response:

```json
{
  "name": "dc-west-replica",
  "contextType": "AUTO",
  "subjects": ["orders-value", "payments-value"],
  "subjectRenameFormat": "${subject}",
  "config": {
    "schema.registry.url": "http://registry-west:8081"
  }
}
```

If the exporter does not exist, the registry returns:

```json
{"error_code": 40450, "message": "Exporter not found: dc-west-replica"}
```

### Update an Exporter

Update an existing exporter's configuration. The exporter is identified by the `{name}` path parameter. All updatable fields MAY be included in the request body -- only provided fields are updated.

```bash
curl -X PUT http://localhost:8081/exporters/dc-west-replica \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "subjects": ["orders-value", "payments-value", "users-value"],
    "config": {
      "schema.registry.url": "http://registry-west:8081",
      "auth.token": "bearer-token-xyz"
    }
  }'
```

Response:

```json
{"name": "dc-west-replica"}
```

### Delete an Exporter

Permanently remove an exporter:

```bash
curl -X DELETE http://localhost:8081/exporters/dc-west-replica
```

Response:

```json
{"name": "dc-west-replica"}
```

> Deleting an exporter is irreversible. The exporter and all its associated state (status, offset) are removed.

---

## Exporter Lifecycle

Exporters follow a state machine with four states: `STARTING`, `RUNNING`, `PAUSED`, and `ERROR`. Lifecycle operations let you control replication without deleting and recreating the exporter.

```
STARTING --> RUNNING --> PAUSED
                |           |
                v           |
              ERROR <-------+
                |
                v
            (reset) --> STARTING
```

### Pause an Exporter

Pause a running exporter. The exporter retains its current offset so it can resume from where it left off.

```bash
curl -X PUT http://localhost:8081/exporters/dc-west-replica/pause
```

Response:

```json
{"name": "dc-west-replica"}
```

### Resume an Exporter

Resume a paused exporter. Replication continues from the last exported offset.

```bash
curl -X PUT http://localhost:8081/exporters/dc-west-replica/resume
```

Response:

```json
{"name": "dc-west-replica"}
```

### Reset an Exporter

Reset an exporter's offset to zero. This causes the exporter to re-export all schemas from the beginning on its next run. This is useful when the destination registry has been wiped or when you want to force a full resynchronization.

```bash
curl -X PUT http://localhost:8081/exporters/dc-west-replica/reset
```

Response:

```json
{"name": "dc-west-replica"}
```

> Resetting an exporter does NOT delete it or change its configuration. It only resets the replication offset.

---

## Monitoring Exporter Status

Retrieve the current status of an exporter, including its state, last exported offset, and any error information:

```bash
curl http://localhost:8081/exporters/dc-west-replica/status
```

Response (healthy):

```json
{
  "name": "dc-west-replica",
  "state": "RUNNING",
  "offset": 42,
  "ts": 1708444800
}
```

Response (error state):

```json
{
  "name": "dc-west-replica",
  "state": "ERROR",
  "offset": 37,
  "ts": 1708444800,
  "trace": "connection refused: http://registry-west:8081"
}
```

**Status fields:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | The exporter name. |
| `state` | string | Current state: `STARTING`, `RUNNING`, `PAUSED`, or `ERROR`. |
| `offset` | integer | The last successfully exported schema offset. |
| `ts` | integer | Unix timestamp of the last state change. |
| `trace` | string | Error trace information. Only present when `state` is `ERROR`. |

---

## Exporter Configuration

Exporter configuration is a set of key-value string pairs that define destination-specific settings. You can manage configuration independently from the exporter definition itself.

### Get Exporter Config

Retrieve the current configuration of an exporter:

```bash
curl http://localhost:8081/exporters/dc-west-replica/config
```

Response:

```json
{
  "schema.registry.url": "http://registry-west:8081",
  "auth.token": "bearer-token-xyz"
}
```

If the exporter has no configuration, the response is an empty object:

```json
{}
```

### Update Exporter Config

Replace the configuration of an exporter. The new configuration fully replaces the previous one.

```bash
curl -X PUT http://localhost:8081/exporters/dc-west-replica/config \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "config": {
      "schema.registry.url": "http://registry-west-new:8081",
      "auth.token": "updated-bearer-token"
    }
  }'
```

Response:

```json
{"name": "dc-west-replica"}
```

---

## Context-Scoped Exporters

All exporter endpoints are also available under a context prefix. This allows you to manage exporters that are scoped to a specific context namespace.

The URL pattern is:

```
/contexts/{context}/exporters/...
```

For example, to create an exporter within the `.production` context:

```bash
curl -X POST http://localhost:8081/contexts/.production/exporters \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "name": "prod-dr-replica",
    "subjects": ["orders-value"],
    "config": {
      "schema.registry.url": "http://dr-registry:8081"
    }
  }'
```

To list exporters in a specific context:

```bash
curl http://localhost:8081/contexts/.production/exporters
```

To get the status of a context-scoped exporter:

```bash
curl http://localhost:8081/contexts/.production/exporters/prod-dr-replica/status
```

All other operations (update, delete, pause, resume, reset, get/update config) follow the same pattern -- prepend `/contexts/{context}` to the exporter path.

> For more information about contexts and namespace isolation, see [Contexts](contexts.md).

---

## API Reference Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/exporters` | Create a new exporter |
| `GET` | `/exporters` | List all exporter names |
| `GET` | `/exporters/{name}` | Get exporter details |
| `PUT` | `/exporters/{name}` | Update an exporter |
| `DELETE` | `/exporters/{name}` | Delete an exporter |
| `PUT` | `/exporters/{name}/pause` | Pause an exporter |
| `PUT` | `/exporters/{name}/resume` | Resume an exporter |
| `PUT` | `/exporters/{name}/reset` | Reset exporter offset |
| `GET` | `/exporters/{name}/status` | Get exporter status |
| `GET` | `/exporters/{name}/config` | Get exporter configuration |
| `PUT` | `/exporters/{name}/config` | Update exporter configuration |

All endpoints above are also available under `/contexts/{context}/exporters/...` for context-scoped operations.

**Error codes:**

| Code | Meaning |
|------|---------|
| `40450` | Exporter not found |
| `40950` | Exporter already exists |
| `50001` | Internal server error |

---

## Related Documentation

- [Fundamentals](fundamentals.md) -- core schema registry concepts including schema IDs, subjects, and deduplication
- [Contexts](contexts.md) -- multi-tenancy and namespace isolation
- [Configuration](configuration.md) -- YAML configuration reference
- [API Reference](api-reference.md) -- complete endpoint documentation
- [Data Contracts](data-contracts.md) -- metadata, rules, and governance policies
