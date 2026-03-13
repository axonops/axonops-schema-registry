# Audit Logging

## Contents

- [Overview](#overview)
- [Configuration](#configuration)
  - [Stdout Output](#stdout-output)
  - [File Output (with Rotation)](#file-output-with-rotation)
  - [Syslog Output (RFC 5424)](#syslog-output-rfc-5424)
  - [Webhook Output](#webhook-output)
  - [Environment Variable Overrides](#environment-variable-overrides)
  - [Prometheus Metrics](#prometheus-metrics)
- [Audit Event Schema](#audit-event-schema)
  - [Timing Fields](#timing-fields)
  - [Event Classification](#event-classification)
  - [Actor (Who)](#actor-who)
  - [Target (What Was Affected)](#target-what-was-affected)
  - [Change Integrity](#change-integrity)
  - [Context and Correlation](#context-and-correlation)
  - [Transport](#transport)
  - [Detail](#detail)
- [Event Types](#event-types)
  - [Schema Events](#schema-events)
  - [Subject Events](#subject-events)
  - [Configuration Events](#configuration-events)
  - [Mode Events](#mode-events)
  - [Authentication Events](#authentication-events)
  - [Admin Events](#admin-events)
  - [Encryption Events (KEK/DEK)](#encryption-events-kekdek)
  - [Exporter Events](#exporter-events)
  - [MCP Events](#mcp-events)
- [Outcome and Reason Codes](#outcome-and-reason-codes)
- [Actor Types and Authentication Methods](#actor-types-and-authentication-methods)
- [Target Types](#target-types)
- [Change Integrity Hashes](#change-integrity-hashes)
- [Example Payloads](#example-payloads)
- [Log Forwarding and Analysis](#log-forwarding-and-analysis)
- [Related Documentation](#related-documentation)

## Overview

The audit logger records security-relevant events to a structured JSON log. Each event captures **who** performed an action, **what** was affected, **how** the actor authenticated, and **whether** the action succeeded or failed. Events are emitted for both the REST API and the MCP (Model Context Protocol) server.

Audit events follow industry-standard practices:

- **Actor identification**: Every event records the actor's identity, type (user, API key, MCP client, or anonymous), RBAC role, and authentication method.
- **Target identification**: Every event records what resource was affected (subject, schema, config, KEK, user, etc.) and its identifier.
- **Outcome classification**: Every event has a structured `outcome` (`success`, `failure`, or `partial_failure`) and, for failures, a machine-parseable `reason` code.
- **Change integrity**: Write operations on schemas, configuration, and modes include `before_hash` and `after_hash` fields — SHA-256 fingerprints of the object before and after the change — enabling integrity verification without logging sensitive content.
- **Cross-protocol consistency**: The same event schema is used for both REST and MCP operations, with `method` distinguishing the protocol (`GET`/`POST`/`PUT`/`DELETE` for REST, `MCP` for MCP tool calls).

## Configuration

The audit logger supports multiple simultaneous outputs: **stdout**, **file** (with rotation), **syslog** (RFC 5424 over TCP/UDP/TCP+TLS), and **webhook** (HTTP with batching and retry). Each output can be independently enabled and configured with its own format (`json` or `cef`).

```yaml
security:
  audit:
    enabled: true
    events:
      - schema_register
      - schema_delete
      - config_update
      - auth_failure
      - auth_forbidden
      - subject_delete
    include_body: false
    outputs:
      stdout:
        enabled: true
        format_type: json         # json or cef
      file:
        enabled: true
        path: /var/log/axonops-schema-registry/audit.log
        format_type: json
        max_size_mb: 100          # Max size before rotation
        max_backups: 5            # Number of rotated files to keep
        max_age_days: 30          # Days to retain rotated files
        compress: true            # Gzip rotated files
      syslog:
        enabled: false
        network: tcp              # tcp, udp, or tcp+tls
        address: "localhost:514"
        app_name: schema-registry
        facility: local0
        format_type: json
        tls_ca: ""                # CA cert path (for tcp+tls)
        tls_cert: ""              # Client cert path (for mTLS)
        tls_key: ""               # Client key path (for mTLS)
      webhook:
        enabled: false
        url: "https://splunk-hec:8088/services/collector/event"
        format_type: json
        batch_size: 100           # Events per batch
        flush_interval: "5s"      # Max time before flushing
        timeout: "10s"            # HTTP request timeout
        max_retries: 3            # Retries on 5xx errors
        buffer_size: 10000        # Channel buffer (overflow drops)
        headers:                  # Custom HTTP headers
          Authorization: "Splunk test-token"
```

### Top-Level Fields

| Field | Description | Default |
|-------|-------------|---------|
| `enabled` | Enable audit logging. | `false` |
| `events` | List of event types to record. If empty, all security-relevant events are logged by default (see [Event Types](#event-types)). | `[]` |
| `include_body` | Include the request body in audit entries (truncated to 1,000 characters). SHOULD only be enabled in development or debugging scenarios, as request bodies MAY contain sensitive data. | `false` |
| `buffer_size` | Async event buffer size. `Log()` enqueues events for a background goroutine; when the buffer is full, new events are dropped and counted via the `schema_registry_audit_buffer_dropped_total` metric. | `10000` |
| `log_file` | **Legacy**. Path to audit log file. Use `outputs.file` instead. Kept for backward compatibility — when no explicit `outputs` are configured, this field creates a plain file output. | `""` |

> **Async delivery:** `Log()` enqueues events onto a buffered channel. A single background goroutine reads, serializes (JSON/CEF), and fans out to all outputs. When the buffer is full, events are dropped with a warning log and the `schema_registry_audit_buffer_dropped_total` metric is incremented. On shutdown, the channel is drained with a 5-second deadline before outputs are closed.

### Stdout Output

| Field | Description | Default |
|-------|-------------|---------|
| `outputs.stdout.enabled` | Enable stdout audit output. | `false` |
| `outputs.stdout.format_type` | Serialization format: `json` or `cef`. | `json` |

### File Output (with Rotation)

Uses [lumberjack](https://github.com/natefinch/lumberjack) for automatic log rotation.

| Field | Description | Default |
|-------|-------------|---------|
| `outputs.file.enabled` | Enable file audit output. | `false` |
| `outputs.file.path` | Absolute path to the audit log file. **REQUIRED** when enabled. | `""` |
| `outputs.file.format_type` | Serialization format: `json` or `cef`. | `json` |
| `outputs.file.max_size_mb` | Maximum file size in MB before rotation. | `100` |
| `outputs.file.max_backups` | Number of rotated log files to retain. | `5` |
| `outputs.file.max_age_days` | Days to retain rotated log files. | `30` |
| `outputs.file.compress` | Gzip-compress rotated log files. | `true` |

### Syslog Output (RFC 5424)

Sends audit events to a syslog server over TCP, UDP, or TCP+TLS using RFC 5424 format.

| Field | Description | Default |
|-------|-------------|---------|
| `outputs.syslog.enabled` | Enable syslog audit output. | `false` |
| `outputs.syslog.network` | Transport protocol: `tcp`, `udp`, or `tcp+tls`. | `tcp` |
| `outputs.syslog.address` | Syslog server address (host:port). **REQUIRED** when enabled. | `""` |
| `outputs.syslog.app_name` | Application name in syslog messages. | `schema-registry` |
| `outputs.syslog.facility` | Syslog facility: `local0`–`local7`, `auth`, `daemon`, etc. | `local0` |
| `outputs.syslog.format_type` | Serialization format: `json` or `cef`. | `json` |
| `outputs.syslog.tls_ca` | Path to CA certificate file (for `tcp+tls`). | `""` |
| `outputs.syslog.tls_cert` | Path to client certificate file (for mTLS). | `""` |
| `outputs.syslog.tls_key` | Path to client private key file (for mTLS). | `""` |

### Webhook Output

Delivers batched audit events to an HTTP endpoint with exponential backoff retry on 5xx errors.

| Field | Description | Default |
|-------|-------------|---------|
| `outputs.webhook.enabled` | Enable webhook audit output. | `false` |
| `outputs.webhook.url` | HTTP endpoint URL. **REQUIRED** when enabled. | `""` |
| `outputs.webhook.format_type` | Serialization format: `json` or `cef`. | `json` |
| `outputs.webhook.batch_size` | Number of events per batch before flushing. | `100` |
| `outputs.webhook.flush_interval` | Maximum time between flushes (Go duration string). | `5s` |
| `outputs.webhook.timeout` | HTTP request timeout (Go duration string). | `10s` |
| `outputs.webhook.max_retries` | Maximum retry attempts on 5xx errors. | `3` |
| `outputs.webhook.buffer_size` | Internal channel buffer size. Events are dropped when full. | `10000` |
| `outputs.webhook.headers` | Custom HTTP headers (map of key-value pairs). | `{}` |

### Environment Variable Overrides

All audit configuration fields support environment variable overrides with the `SCHEMA_REGISTRY_AUDIT_` prefix:

```bash
SCHEMA_REGISTRY_AUDIT_ENABLED=true
SCHEMA_REGISTRY_AUDIT_INCLUDE_BODY=false
SCHEMA_REGISTRY_AUDIT_BUFFER_SIZE=10000
SCHEMA_REGISTRY_AUDIT_STDOUT_ENABLED=true
SCHEMA_REGISTRY_AUDIT_STDOUT_FORMAT=json
SCHEMA_REGISTRY_AUDIT_FILE_ENABLED=true
SCHEMA_REGISTRY_AUDIT_FILE_PATH=/var/log/audit.log
SCHEMA_REGISTRY_AUDIT_FILE_FORMAT=json
SCHEMA_REGISTRY_AUDIT_FILE_MAX_SIZE_MB=100
SCHEMA_REGISTRY_AUDIT_FILE_MAX_BACKUPS=5
SCHEMA_REGISTRY_AUDIT_FILE_MAX_AGE_DAYS=30
SCHEMA_REGISTRY_AUDIT_FILE_COMPRESS=true
SCHEMA_REGISTRY_AUDIT_SYSLOG_ENABLED=true
SCHEMA_REGISTRY_AUDIT_SYSLOG_NETWORK=tcp+tls
SCHEMA_REGISTRY_AUDIT_SYSLOG_ADDRESS=syslog.internal:6514
SCHEMA_REGISTRY_AUDIT_SYSLOG_APP_NAME=schema-registry
SCHEMA_REGISTRY_AUDIT_SYSLOG_FACILITY=local0
SCHEMA_REGISTRY_AUDIT_SYSLOG_FORMAT=json
SCHEMA_REGISTRY_AUDIT_SYSLOG_TLS_CA=/etc/ssl/ca.pem
SCHEMA_REGISTRY_AUDIT_SYSLOG_TLS_CERT=/etc/ssl/client.pem
SCHEMA_REGISTRY_AUDIT_SYSLOG_TLS_KEY=/etc/ssl/client-key.pem
SCHEMA_REGISTRY_AUDIT_WEBHOOK_ENABLED=true
SCHEMA_REGISTRY_AUDIT_WEBHOOK_URL=https://splunk:8088/
SCHEMA_REGISTRY_AUDIT_WEBHOOK_FORMAT=json
SCHEMA_REGISTRY_AUDIT_WEBHOOK_BATCH_SIZE=100
SCHEMA_REGISTRY_AUDIT_WEBHOOK_FLUSH_INTERVAL=5s
SCHEMA_REGISTRY_AUDIT_WEBHOOK_TIMEOUT=10s
SCHEMA_REGISTRY_AUDIT_WEBHOOK_MAX_RETRIES=3
SCHEMA_REGISTRY_AUDIT_WEBHOOK_BUFFER_SIZE=10000
```

### Prometheus Metrics

The following Prometheus metrics are available for monitoring audit output health:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `schema_registry_audit_events_total` | Counter | `output`, `status` | Total audit events written per output and status. |
| `schema_registry_audit_output_errors_total` | Counter | `output` | Total write errors per output. |
| `schema_registry_audit_buffer_dropped_total` | Counter | — | Events dropped due to async audit buffer overflow. |
| `schema_registry_audit_webhook_dropped_total` | Counter | — | Events dropped due to webhook buffer overflow. |
| `schema_registry_audit_webhook_batch_size` | Histogram | — | Distribution of webhook batch sizes. |
| `schema_registry_audit_webhook_flush_duration_seconds` | Histogram | — | Time to flush webhook batches. |

## Audit Event Schema

Each audit entry is a single-line JSON object. All fields use underscore-separated naming (flat structure, no nesting).

> **Core vs contextual fields:** Core fields (`timestamp`, `event_type`, `outcome`, `duration_ms`, `actor_id`, `actor_type`, `target_type`, `target_id`, `source_ip`, `user_agent`, `method`, `path`, `status_code`) are **always present** in the JSON output, even when their value is empty or zero. This follows industry-standard audit practice (AWS CloudTrail, CEF) where core fields MUST be present for reliable SIEM indexing and alerting. Contextual fields (`role`, `auth_method`, `schema_id`, `version`, `schema_type`, `before_hash`, `after_hash`, `context`, `request_id`, `reason`, `error`, `request_body`, `metadata`) are **omitted** when their value is zero/empty, as their presence depends on the operation type.

### Timing Fields

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | ISO 8601 | When the event occurred. |
| `duration_ms` | integer | Request processing time in milliseconds. |

### Event Classification

| Field | Type | Description |
|-------|------|-------------|
| `event_type` | string | Stable event type identifier (see [Event Types](#event-types)). |
| `outcome` | string | `success`, `failure`, or `partial_failure`. |

### Actor (Who)

| Field | Type | Description |
|-------|------|-------------|
| `actor_id` | string | Identity of the actor: username, API key name, or MCP principal. Empty for anonymous/unauthenticated requests. |
| `actor_type` | string | Type of actor: `user`, `api_key`, `mcp_client`, or `anonymous`. See [Actor Types](#actor-types-and-authentication-methods). |
| `role` | string | RBAC role at the time of the action: `admin`, `developer`, `readonly`, or empty if unauthenticated. |
| `auth_method` | string | Authentication mechanism used: `basic`, `api_key`, `jwt`, `oidc`, `ldap`, `mtls`, `bearer_token`, or empty. See [Authentication Methods](#actor-types-and-authentication-methods). |

### Target (What Was Affected)

| Field | Type | Description |
|-------|------|-------------|
| `target_type` | string | Type of resource affected. See [Target Types](#target-types). |
| `target_id` | string | Identifier of the affected resource (subject name, KEK name, exporter name, user ID, etc.). |
| `schema_id` | integer | Global schema ID, when applicable. |
| `version` | integer | Schema version number, when applicable. |
| `schema_type` | string | Schema format: `AVRO`, `PROTOBUF`, or `JSON`, when applicable. |

### Change Integrity

| Field | Type | Description |
|-------|------|-------------|
| `before_hash` | string | SHA-256 fingerprint of the object **before** the change, prefixed with `sha256:`. Present on update/delete operations where the previous state is available. |
| `after_hash` | string | SHA-256 fingerprint of the object **after** the change, prefixed with `sha256:`. Present on create/update operations. |

> **Important:** Hashes allow you to verify that a specific version of an object was present at the time of the audit event by re-hashing the stored object and comparing digests. They do NOT provide anonymity — do not treat hashes as safe for low-entropy or guessable values.

For schemas, the hash is the schema's canonical fingerprint (already computed and stored in the database). For config and mode values, the hash is computed at audit time from the string value.

### Context and Correlation

| Field | Type | Description |
|-------|------|-------------|
| `context` | string | Registry context namespace (`registryCtx`). Default is `"."`. Only present when a non-default context is used. |
| `request_id` | string | Unique request identifier for correlation (from chi middleware for REST, empty for MCP). |

### Transport

| Field | Type | Description |
|-------|------|-------------|
| `source_ip` | string | Client IP address (supports `X-Forwarded-For` for proxied requests). |
| `user_agent` | string | Client `User-Agent` header value. |
| `method` | string | HTTP method (`GET`, `POST`, `PUT`, `DELETE`) for REST, or `MCP` for MCP tool calls. |
| `path` | string | HTTP request path for REST, or MCP tool name for MCP events. |
| `status_code` | integer | HTTP response status code. Only present for REST events (omitted for MCP). |

### Detail

| Field | Type | Description |
|-------|------|-------------|
| `reason` | string | Structured failure reason code (see [Reason Codes](#outcome-and-reason-codes)). Only present on failure events. |
| `error` | string | Raw error message. Only present when an error occurred. |
| `request_body` | string | Truncated request body (max 1,000 characters). Only present when `include_body: true`. |
| `metadata` | object | Event-specific key-value pairs (e.g., MCP schema body excerpts). |

## Event Types

When the `events` list is empty in configuration, all events marked with **[default]** below are logged.

### Schema Events

| Event Type | Trigger | Default |
|------------|---------|---------|
| `schema_register` | `POST /subjects/{subject}/versions` | **[default]** |
| `schema_delete` | `DELETE /subjects/{subject}/versions/{version}` | **[default]** |
| `schema_get` | `GET /subjects/{subject}/versions/*` or `GET /schemas/ids/*` | |
| `schema_lookup` | `POST /subjects/{subject}` (check if schema exists) | **[default]** |
| `schema_import` | `POST /import/schemas` | **[default]** |

### Subject Events

| Event Type | Trigger | Default |
|------------|---------|---------|
| `subject_delete` | `DELETE /subjects/{subject}` | **[default]** |
| `subject_list` | `GET /subjects` | |

### Configuration Events

| Event Type | Trigger | Default |
|------------|---------|---------|
| `config_get` | `GET /config` or `GET /config/{subject}` | |
| `config_update` | `PUT /config` or `PUT /config/{subject}` | **[default]** |
| `config_delete` | `DELETE /config` or `DELETE /config/{subject}` | **[default]** |

### Mode Events

| Event Type | Trigger | Default |
|------------|---------|---------|
| `mode_get` | `GET /mode` or `GET /mode/{subject}` | |
| `mode_update` | `PUT /mode` or `PUT /mode/{subject}` | **[default]** |
| `mode_delete` | `DELETE /mode` or `DELETE /mode/{subject}` | **[default]** |

### Authentication Events

| Event Type | Trigger | Default |
|------------|---------|---------|
| `auth_success` | Successful authentication | |
| `auth_failure` | HTTP 401 (authentication failed) | **[default]** |
| `auth_forbidden` | HTTP 403 (authorization failed) | **[default]** |

### Admin Events

| Event Type | Trigger | Default |
|------------|---------|---------|
| `user_create` | `POST /admin/users` | **[default]** |
| `user_update` | `PUT /admin/users/{id}` | **[default]** |
| `user_delete` | `DELETE /admin/users/{id}` | **[default]** |
| `password_change` | `POST /me/password` | **[default]** |
| `apikey_create` | `POST /admin/apikeys` | **[default]** |
| `apikey_update` | `PUT /admin/apikeys/{id}` | **[default]** |
| `apikey_delete` | `DELETE /admin/apikeys/{id}` | **[default]** |
| `apikey_revoke` | `POST /admin/apikeys/{id}/revoke` | **[default]** |
| `apikey_rotate` | `POST /admin/apikeys/{id}/rotate` | **[default]** |

### Encryption Events (KEK/DEK)

| Event Type | Trigger | Default |
|------------|---------|---------|
| `kek_create` | `POST /dek-registry/v1/keks` or undelete | **[default]** |
| `kek_update` | `PUT /dek-registry/v1/keks/{name}` | **[default]** |
| `kek_delete` | `DELETE /dek-registry/v1/keks/{name}` | **[default]** |
| `kek_test` | `POST /dek-registry/v1/keks/{name}/test` | **[default]** |
| `dek_create` | `POST /dek-registry/v1/keks/{name}/deks` or undelete | **[default]** |
| `dek_delete` | `DELETE /dek-registry/v1/keks/{name}/deks/{subject}` | **[default]** |

### Exporter Events

| Event Type | Trigger | Default |
|------------|---------|---------|
| `exporter_create` | `POST /exporters` | **[default]** |
| `exporter_update` | `PUT /exporters/{name}` | **[default]** |
| `exporter_delete` | `DELETE /exporters/{name}` | **[default]** |
| `exporter_pause` | `PUT /exporters/{name}/pause` | **[default]** |
| `exporter_resume` | `PUT /exporters/{name}/resume` | **[default]** |
| `exporter_reset` | `PUT /exporters/{name}/reset` | **[default]** |

### MCP Events

| Event Type | Trigger | Default |
|------------|---------|---------|
| `mcp_tool_call` | MCP tool invoked successfully | **[default]** |
| `mcp_tool_error` | MCP tool invoked with error result | **[default]** |
| `mcp_admin_action` | MCP admin tool invoked | **[default]** |
| `mcp_confirm_issued` | Two-phase confirmation token issued | **[default]** |
| `mcp_confirm_rejected` | Confirmation token validation failed | **[default]** |
| `mcp_confirmed` | Destructive operation confirmed and executed | **[default]** |

## Outcome and Reason Codes

Every audit event has an `outcome` field:

| Outcome | Description |
|---------|-------------|
| `success` | The operation completed successfully. |
| `failure` | The operation failed entirely. |
| `partial_failure` | A bulk operation partially succeeded (e.g., `/import/schemas` imported some schemas but not all). |

> **Note:** The `/import/schemas` endpoint returns HTTP 422 when **all** schemas in a batch fail (0 imported, errors > 0), and HTTP 200 with `outcome: partial_failure` when some schemas succeed and others fail. This ensures audit events accurately reflect the actual result of bulk operations.

For failure events, the `reason` field provides a machine-parseable classification:

| Reason | Description | Typical Status Code |
|--------|-------------|---------------------|
| `no_valid_credentials` | No authentication credentials provided. | 401 |
| `invalid_credentials` | Credentials provided but invalid. | 401 |
| `permission_denied` | Authenticated but insufficient privileges. | 403 |
| `not_found` | Requested resource does not exist. | 404 |
| `already_exists` | Resource already exists (duplicate). | 409 |
| `incompatible` | Schema compatibility check failed. | 409 |
| `validation_error` | Invalid request payload or parameters. | 400 |
| `invalid_schema` | Schema validation or parsing failed. | 422 |
| `rate_limited` | Request rejected by rate limiter. | 429 |
| `internal_error` | Unexpected server error. | 500 |

For MCP events, the same `reason` codes apply. Since MCP events do not have HTTP status codes, the `reason` field is the primary way to classify MCP failures.

## Actor Types and Authentication Methods

### Actor Types (`actor_type`)

| Value | Description |
|-------|-------------|
| `user` | Authenticated via Basic Auth (username/password) against DB, config, htpasswd, or LDAP. |
| `api_key` | Authenticated via API key (header, query param, or Basic Auth format). |
| `mcp_client` | MCP tool call with bearer token authentication. |
| `anonymous` | No authentication provided, or authentication is disabled. |

### Authentication Methods (`auth_method`)

| Value | Description |
|-------|-------------|
| `basic` | HTTP Basic Authentication (username + password). |
| `api_key` | API key via header (`X-API-Key`), query parameter, or Basic Auth format. |
| `jwt` | JSON Web Token (Bearer token validated against signing key). |
| `oidc` | OpenID Connect (Bearer token validated against OIDC provider). |
| `ldap` | LDAP bind authentication (username + password via Basic Auth). |
| `mtls` | Mutual TLS (client certificate CN used as identity). |
| `bearer_token` | MCP static bearer token authentication. |

> **Note:** When authentication is disabled (`security.auth.enabled: false`), `actor_type` is `anonymous` and `auth_method` is empty.

## Target Types

| Value | Description | `target_id` contains |
|-------|-------------|---------------------|
| `subject` | Schema subject (topic). | Subject name |
| `schema` | Schema by global ID. | Schema ID (as string) |
| `config` | Compatibility configuration (global or per-subject). | Subject name, or `_global` for global config |
| `mode` | Registry mode (global or per-subject). | Subject name, or `_global` for global mode |
| `kek` | Key Encryption Key. | KEK name |
| `dek` | Data Encryption Key. | Subject name |
| `exporter` | Schema exporter (Schema Linking). | Exporter name |
| `user` | Admin user account. | Username or user ID |
| `apikey` | Admin API key. | API key name or ID |

## Change Integrity Hashes

Write operations on schemas, configuration, and mode include `before_hash` and `after_hash` fields to enable change verification without logging full content.

| Resource | Hash Source | Notes |
|----------|------------|-------|
| Schema | SHA-256 canonical fingerprint (stored in DB). | `before_hash` is the previous version's fingerprint. `after_hash` is the new version's fingerprint. For first versions (v1), `before_hash` is absent. |
| Config | SHA-256 of the compatibility level string. | e.g., `sha256:` + SHA-256(`"BACKWARD"`). |
| Mode | SHA-256 of the mode string. | e.g., `sha256:` + SHA-256(`"READWRITE"`). |
| KEK/DEK | Not yet implemented. | See issue #344. |
| Exporter | Not yet implemented. | See issue #345. |

Hashes are prefixed with `sha256:` for forward compatibility with other hash algorithms.

To verify an audit entry against stored data:
1. Retrieve the schema/config/mode at the version referenced in the audit event.
2. Compute its SHA-256 hash (for schemas, use the canonical fingerprint; for config/mode, hash the string value).
3. Compare with the `before_hash` or `after_hash` in the audit entry.

## Example Payloads

### Schema Registration (version 2+, Basic Auth)

```json
{
  "timestamp": "2026-03-11T14:30:00Z",
  "duration_ms": 42,
  "event_type": "schema_register",
  "outcome": "success",
  "actor_id": "jane",
  "actor_type": "user",
  "role": "developer",
  "auth_method": "basic",
  "target_type": "subject",
  "target_id": "payments-value",
  "schema_id": 42,
  "version": 3,
  "schema_type": "AVRO",
  "before_hash": "sha256:8b7df143d91c716ecfa5fc1730022f6b421b05cedee8fd52b1fc65a96030ad52",
  "after_hash": "sha256:2d91a1a0e1b2963e26c761be76c6e37b0bb2dc6f99a1e6a5baab519eda112cd5",
  "source_ip": "172.18.0.1",
  "user_agent": "curl/8.1",
  "method": "POST",
  "path": "/subjects/payments-value/versions",
  "status_code": 200,
  "request_id": "localhost/abc-123-def"
}
```

### Authentication Failure (no credentials)

```json
{
  "timestamp": "2026-03-11T14:31:00Z",
  "duration_ms": 1,
  "event_type": "auth_failure",
  "outcome": "failure",
  "actor_type": "anonymous",
  "reason": "no_valid_credentials",
  "source_ip": "172.18.0.1",
  "user_agent": "python-requests/2.31",
  "method": "POST",
  "path": "/admin/apikeys",
  "status_code": 401
}
```

### Authorization Failure (wrong role)

```json
{
  "timestamp": "2026-03-11T14:32:00Z",
  "duration_ms": 3,
  "event_type": "auth_forbidden",
  "outcome": "failure",
  "actor_id": "viewer",
  "actor_type": "user",
  "role": "readonly",
  "auth_method": "basic",
  "reason": "permission_denied",
  "source_ip": "172.18.0.1",
  "method": "POST",
  "path": "/admin/apikeys",
  "status_code": 403
}
```

### API Key Creating Another API Key

```json
{
  "timestamp": "2026-03-11T14:33:00Z",
  "duration_ms": 15,
  "event_type": "apikey_create",
  "outcome": "success",
  "actor_id": "ci-deployer",
  "actor_type": "api_key",
  "role": "admin",
  "auth_method": "api_key",
  "target_type": "apikey",
  "target_id": "new-service-key",
  "source_ip": "10.0.0.50",
  "method": "POST",
  "path": "/admin/apikeys",
  "status_code": 201
}
```

### MCP Tool Call (authenticated)

```json
{
  "timestamp": "2026-03-11T14:34:00Z",
  "duration_ms": 8,
  "event_type": "mcp_tool_call",
  "outcome": "success",
  "actor_id": "mcp-authenticated",
  "actor_type": "mcp_client",
  "auth_method": "bearer_token",
  "target_type": "subject",
  "target_id": "orders-value",
  "method": "MCP",
  "path": "get_latest_schema"
}
```

### MCP Tool Error (subject not found)

```json
{
  "timestamp": "2026-03-11T14:35:00Z",
  "duration_ms": 2,
  "event_type": "mcp_tool_error",
  "outcome": "failure",
  "actor_id": "mcp-authenticated",
  "actor_type": "mcp_client",
  "auth_method": "bearer_token",
  "target_type": "subject",
  "target_id": "nonexistent",
  "reason": "not_found",
  "error": "Subject 'nonexistent' not found",
  "method": "MCP",
  "path": "get_latest_schema"
}
```

### Config Update (global)

```json
{
  "timestamp": "2026-03-11T14:36:00Z",
  "duration_ms": 5,
  "event_type": "config_update",
  "outcome": "success",
  "actor_id": "admin",
  "actor_type": "user",
  "role": "admin",
  "auth_method": "basic",
  "target_type": "config",
  "target_id": "_global",
  "before_hash": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "after_hash": "sha256:7d1a54127b222502f5b79b5fb0803061152a44f92b37e23c6527baf665d4da9a",
  "source_ip": "172.18.0.1",
  "method": "PUT",
  "path": "/config",
  "status_code": 200
}
```

### LDAP Authentication

```json
{
  "timestamp": "2026-03-11T14:37:00Z",
  "duration_ms": 120,
  "event_type": "schema_register",
  "outcome": "success",
  "actor_id": "ldap-user",
  "actor_type": "user",
  "role": "developer",
  "auth_method": "ldap",
  "target_type": "subject",
  "target_id": "inventory-value",
  "schema_type": "PROTOBUF",
  "after_hash": "sha256:abc123...",
  "source_ip": "10.0.1.100",
  "method": "POST",
  "path": "/subjects/inventory-value/versions",
  "status_code": 200
}
```

## Log Forwarding and Analysis

Audit events can be delivered directly to external systems using the built-in outputs:

- **Splunk**: Use the webhook output with Splunk HEC (`url: https://splunk:8088/services/collector/event`) and add an `Authorization: Splunk <token>` header.
- **Elasticsearch / OpenSearch**: Use the webhook output pointing to the bulk API endpoint.
- **Syslog (SIEM)**: Use the syslog output with `tcp+tls` for encrypted delivery to your SIEM.
- **File + Fluentd/Logstash**: Use the file output with rotation, then ship with a sidecar agent.
- **Grafana Loki**: Use label extraction on `event_type` and `outcome` from the file output.

Audit log entries are single-line JSON (or CEF), making them compatible with standard log aggregation tools. For file-based forwarding:

Useful queries:

- All failures: `jq 'select(.outcome == "failure" or .outcome == "partial_failure")'`
- All admin actions: `jq 'select(.event_type | startswith("user_") or startswith("apikey_"))'`
- All actions by a specific user: `jq 'select(.actor_id == "jane")'`
- All permission denials: `jq 'select(.reason == "permission_denied")'`
- Schema changes with integrity hashes: `jq 'select(.after_hash != null)'`

## Related Documentation

- [Security](security.md) — TLS, RBAC, rate limiting, security hardening
- [Authentication](authentication.md) — Authentication methods, user management, API key lifecycle
- [Configuration](configuration.md) — Full configuration reference
- [MCP Server](mcp.md) — MCP server security, permission scopes
