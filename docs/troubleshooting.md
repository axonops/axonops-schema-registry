# Troubleshooting

This guide covers common issues encountered when running AxonOps Schema Registry, along with diagnostic commands and a complete error code reference.

## Contents

- [Common Issues](#common-issues)
  - [Connection Refused](#connection-refused)
  - [Database Connection Errors](#database-connection-errors)
  - [Schema Registration Fails](#schema-registration-fails)
  - [Authentication Issues](#authentication-issues)
  - [Soft Delete Confusion](#soft-delete-confusion)
  - [Performance Issues](#performance-issues)
  - [Import and Migration Issues](#import-and-migration-issues)
- [Error Code Reference](#error-code-reference)
- [Diagnostic Commands](#diagnostic-commands)
- [Getting Help](#getting-help)

---

## Common Issues

### Connection Refused

**Symptoms:** `curl: (7) Failed to connect to localhost port 8081: Connection refused`

**Diagnostics:**

```bash
# Check if the process is running
pgrep -f schema-registry

# Check container status
docker ps -a --filter name=schema-registry

# Verify which port the server is listening on
ss -tlnp | grep schema-registry
```

**Resolution:**

1. **Server not running.** Start or restart the registry process or container.

2. **Wrong port.** The default port is `8081`. Confirm the `server.port` value in your [configuration file](configuration.md):

   ```yaml
   server:
     port: 8081
   ```

3. **Binding to wrong interface.** By default the server MAY bind to `127.0.0.1`, which rejects connections from other hosts. Set `server.host` to `0.0.0.0` to accept connections on all interfaces:

   ```yaml
   server:
     host: "0.0.0.0"
   ```

4. **Docker networking.** When running in Docker, ensure the port is mapped correctly:

   ```bash
   # Map port explicitly
   docker run -d -p 8081:8081 ghcr.io/axonops/axonops-schema-registry:latest

   # Or use host networking
   docker run -d --network host ghcr.io/axonops/axonops-schema-registry:latest
   ```

   If connecting from another container, use the container name or Docker network IP rather than `localhost`.

**Root Cause:** The registry is not reachable because it is either not running, listening on a different port or interface, or Docker networking is misconfigured.

---

### Database Connection Errors

**Symptoms:** Server fails to start with a storage connection error, or returns `50002` errors after startup.

**Diagnostics:**

```bash
# Check server logs for storage errors
docker logs schema-registry 2>&1 | grep -i "storage\|database\|connect"

# Test database connectivity directly
psql -h HOST -p PORT -U USER -d DATABASE       # PostgreSQL
mysql -h HOST -P PORT -u USER -p DATABASE       # MySQL
cqlsh HOST PORT                                  # Cassandra
```

**Resolution:**

#### PostgreSQL

- Verify `host`, `port`, `user`, `password`, and `database` in your configuration.
- Check that `pg_hba.conf` allows connections from the registry host.
- Ensure the `ssl_mode` in your configuration matches the server's TLS settings (`disable`, `require`, `verify-ca`, `verify-full`).
- See [Storage Backends -- PostgreSQL](storage-backends.md) for connection pool tuning options.

#### MySQL

- Verify `host`, `port`, `user`, `password`, and `database`.
- Confirm the user has sufficient privileges:

  ```sql
  GRANT ALL PRIVILEGES ON schemaregistry.* TO 'user'@'%';
  FLUSH PRIVILEGES;
  ```

- The `tls_mode` MUST match the server configuration. Common values: `preferred`, `skip-verify`, `true`.
- See [Storage Backends -- MySQL](storage-backends.md) for connection pool tuning options.

#### Cassandra

- Verify that the contact point `hosts` and `port` are correct.
- The keyspace is created automatically on first startup when `migrate: true` is set. If `migrate` is disabled, you MUST create the keyspace manually before starting the registry.
- Cassandra 5.0 or later is REQUIRED for SAI (Storage Attached Index) support.
- The `consistency` level MUST match your cluster topology. For example, `QUORUM` requires a majority of replicas to be available; `LOCAL_ONE` is sufficient for single-datacenter reads.
- See [Storage Backends -- Cassandra](storage-backends.md) for detailed configuration.

**Root Cause:** The registry cannot reach the configured database due to incorrect connection parameters, network restrictions, insufficient privileges, or TLS configuration mismatches.

---

### Schema Registration Fails

#### 42201 Invalid Schema

**Symptoms:** Registration returns error code `42201` with a message indicating the schema is invalid.

**Diagnostics:**

- Check the error message for specific parse failures (missing fields, syntax errors, unresolved references).
- Validate the schema independently using a type-specific tool (e.g., `avro-tools`, `protoc`, or a JSON Schema validator).

**Resolution:**

- **Malformed JSON.** The schema string MUST be valid JSON.
- **Missing required fields.** For example, an Avro record type MUST include a `fields` array.
- **Invalid type references.** A field references a named type that is not defined within the schema or its declared references.
- **Protobuf syntax errors.** The `.proto` content does not parse correctly.
- **Unresolved references.** The `references` array in the registration request points to subjects or versions that do not exist in the registry.

**Root Cause:** The schema content is malformed or violates type-specific rules. The registry validates schemas at registration time and rejects content that cannot be parsed.

#### 409 Incompatible Schema

**Symptoms:** Registration returns HTTP `409` with a message indicating the schema is incompatible.

**Diagnostics:**

1. Check the current compatibility level:

   ```bash
   curl -s http://localhost:8081/config/my-subject | jq .
   ```

2. Run a compatibility check with verbose output:

   ```bash
   curl -s -X POST \
     -H "Content-Type: application/vnd.schemaregistry.v1+json" \
     -d '{"schema": "{...}", "schemaType": "AVRO"}' \
     "http://localhost:8081/compatibility/subjects/my-subject/versions/latest?verbose=true" | jq .
   ```

**Resolution:**

- Fix the schema to be compatible with existing versions, OR
- Temporarily set compatibility to `NONE` if you MUST register the schema regardless:

  ```bash
  curl -X PUT \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d '{"compatibility": "NONE"}' \
    http://localhost:8081/config/my-subject
  ```

> **Warning:** Remember to restore the desired compatibility level afterward. See [Compatibility](compatibility.md) for details on each level.

**Root Cause:** The new schema violates the compatibility rules configured for the subject. The registry checks compatibility at registration time to prevent breaking consumers.

#### 42205 Operation Not Permitted

**Symptoms:** Registration or deletion returns error code `42205` indicating the operation is not permitted.

**Diagnostics:**

```bash
# Check global mode
curl -s http://localhost:8081/mode | jq .

# Check per-subject mode
curl -s http://localhost:8081/mode/my-subject | jq .
```

**Resolution:**

Set the mode to `READWRITE` to allow writes:

```bash
curl -X PUT \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"mode": "READWRITE"}' \
  http://localhost:8081/mode/my-subject
```

**Root Cause:** The subject or global mode is set to `READONLY` or `IMPORT`, which restricts write operations.

---

### Authentication Issues

#### 401 Unauthorized

**Symptoms:** Requests return HTTP `401` with error code `40101`.

**Diagnostics:**

- Confirm authentication is enabled in the configuration (`security.auth.enabled: true`).
- Verify the credentials being sent (check headers, API key format, token expiration).

**Resolution:**

- **No credentials provided.** When authentication is enabled, every request MUST include credentials. See [Authentication](authentication.md) for supported methods.
- **Invalid username or password.** Verify the credentials are correct for basic auth.
- **Expired API key.** Check the key's expiration date. Expired keys return error code `40103`.
- **Disabled API key or user.** Disabled keys return `40104`; disabled users return `40105`.
- **LDAP bind failure.** Verify `bind_dn` and `bind_password` in the LDAP configuration. Ensure the LDAP server is reachable.
- **OIDC token issues.** The token MAY be expired, the audience claim MAY not match the configured value, or the issuer URL MAY be unreachable.

**Root Cause:** Authentication is enabled but the request does not contain valid credentials. The specific error code indicates the failure reason.

#### 403 Forbidden

**Symptoms:** Requests return HTTP `403` with error code `40301`.

**Diagnostics:**

- Check role assignments: `GET /admin/users/{id}` (requires admin privileges).
- Review the RBAC configuration in your [configuration](configuration.md).

**Resolution:**

- Assign the appropriate role to the user. Admin endpoints under `/admin/` require the `admin` role.
- Users listed in `super_admins` bypass all permission checks.

**Root Cause:** The user authenticated successfully but their assigned role lacks the required permission for the operation. See the [permission matrix](security.md#permission-matrix) for role-to-permission mappings.

---

### Soft Delete Confusion

#### 40404 Subject Was Soft-Deleted

**Symptoms:** Operations on a subject return error code `40404` indicating the subject was soft-deleted.

**Diagnostics:**

```bash
# List subjects including soft-deleted ones
curl -s "http://localhost:8081/subjects?deleted=true" | jq .
```

**Resolution:**

To permanently delete the soft-deleted subject:

```bash
curl -X DELETE "http://localhost:8081/subjects/my-subject?permanent=true"
```

Or re-register schemas under the same subject name to restore it.

**Root Cause:** The subject exists but was previously soft-deleted. Soft-deleted subjects do not appear in normal listings.

#### 40405 Subject Not Soft-Deleted

**Symptoms:** Permanent delete returns error code `40405`.

**Resolution:**

The deletion process REQUIRES two steps:

```bash
# Step 1: Soft-delete
curl -X DELETE http://localhost:8081/subjects/my-subject

# Step 2: Permanent delete
curl -X DELETE "http://localhost:8081/subjects/my-subject?permanent=true"
```

**Root Cause:** A permanent delete was attempted on a subject that has not been soft-deleted first. The two-step process is a safety mechanism.

#### 42206 Reference Exists

**Symptoms:** Delete operation returns error code `42206`.

**Resolution:** Remove or update the referencing schemas first, then retry the deletion.

**Root Cause:** The schema or subject being deleted is referenced by another schema. The registry prevents deletion of referenced schemas to avoid breaking dependent schemas.

---

### Performance Issues

#### Slow Schema Registration

**Symptoms:** Schema registration requests take significantly longer than expected (typically >100ms).

**Diagnostics:**

```bash
# Check storage latency via Prometheus metrics
curl -s http://localhost:8081/metrics | grep schema_registry_storage_latency_seconds

# Enable debug logging to identify the bottleneck
SCHEMA_REGISTRY_LOG_LEVEL=debug ./schema-registry --config config.yaml
```

**Resolution:**

- **PostgreSQL**: Increase `max_open_conns` if connection pool exhaustion is causing queuing. Check for lock contention with `pg_stat_activity`.
- **MySQL**: Similar pool tuning via `max_open_conns` and `max_idle_conns`.
- **Cassandra**: Consider using `LOCAL_ONE` consistency for reads if strong consistency is not required. Verify SAI indexes are healthy with `nodetool`.

**Root Cause:** Storage backend latency, connection pool exhaustion, or lock contention. The bottleneck is almost always in the database layer, not schema parsing.

#### High Memory Usage

**Symptoms:** The registry process consumes more memory than expected.

**Diagnostics:**

```bash
# Check the total schema count
curl -s http://localhost:8081/metrics | grep schema_registry_schemas_total
```

**Resolution:** Increase container memory limits for large registries.

**Root Cause:** The number of registered schemas directly affects memory usage, as schemas are cached for fast lookups.

---

### Import and Migration Issues

#### Schema ID Conflicts

**Symptoms:** Import returns an error indicating a schema ID already exists with different content.

**Diagnostics:**

```bash
# Check current mode
curl -s http://localhost:8081/mode | jq .

# Inspect the conflicting schema ID
curl -s http://localhost:8081/schemas/ids/{id} | jq .
```

**Resolution:**

Ensure the global mode is set to `IMPORT` before importing:

```bash
curl -X PUT \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"mode": "IMPORT"}' \
  http://localhost:8081/mode
```

After import completes, restore the mode to `READWRITE`.

**Root Cause:** The import API (`POST /import/schemas`) preserves the original schema IDs. If a schema with the same ID already exists with different content, the import is rejected to prevent data corruption.

#### Missing References

**Symptoms:** Import fails with errors about unresolved schema references.

**Resolution:**

Schemas that reference other schemas (via Avro named types, Protobuf imports, or JSON Schema `$ref`) MUST be imported in dependency order. Import referenced schemas first (lowest ID), then the schemas that depend on them.

**Root Cause:** The registry resolves references at import time. If a referenced schema has not been imported yet, the reference cannot be resolved. The migration script handles this by sorting schemas by ID.

See [Migration](migration.md) for the full import/export workflow.

---

## Error Code Reference

| Code | Constant | Description | Common Cause |
|------|----------|-------------|--------------|
| 409 | Incompatible schema | Compatibility check failed | New schema breaks compatibility rules |
| 40101 | Unauthorized | Authentication required or failed | Missing or invalid credentials |
| 40103 | API key expired | API key past expiration date | Renew or rotate the API key |
| 40104 | API key disabled | API key administratively disabled | Re-enable via admin API |
| 40105 | User disabled | User account disabled | Re-enable via admin API |
| 40301 | Forbidden | Insufficient permissions | User role lacks required access |
| 40401 | Subject not found | Subject does not exist | Typo in subject name |
| 40402 | Version not found | Version does not exist | Requested version number out of range |
| 40403 | Schema not found | Schema ID does not exist | Invalid or non-existent schema ID |
| 40404 | Subject soft-deleted | Subject was deleted but not permanently | Use `?permanent=true` or re-register |
| 40405 | Subject not soft-deleted | Permanent delete requires soft-delete first | Soft-delete the subject first |
| 40406 | Schema version soft-deleted | Specific version was soft-deleted | Use `?permanent=true` to remove permanently |
| 40407 | Version not soft-deleted | Permanent version delete requires soft-delete first | Soft-delete the version first |
| 40408 | Subject compatibility not found | No per-subject compatibility configured | Set compatibility or rely on global default |
| 40409 | Subject mode not found | No per-subject mode configured | Set mode or rely on global default |
| 42201 | Invalid schema | Schema content is malformed | Fix schema syntax or structure |
| 42202 | Invalid schema type or version | Unrecognized schema type or invalid version | Use AVRO, PROTOBUF, or JSON; use valid version number |
| 42203 | Invalid compatibility level | Unrecognized compatibility mode | Use NONE, BACKWARD, FORWARD, FULL, or transitive variants |
| 42204 | Invalid mode | Unrecognized mode value | Use READWRITE, READONLY, or IMPORT |
| 42205 | Operation not permitted | Write rejected due to mode | Change mode to READWRITE or IMPORT |
| 42206 | Reference exists | Schema is referenced by others | Remove referencing schemas first |
| 50001 | Internal server error | Unexpected server error | Check server logs for stack trace |
| 50002 | Storage error | Database connectivity or query failure | Verify database is reachable and healthy |

---

## Diagnostic Commands

```bash
# Health check
curl -s http://localhost:8081/ | jq .

# List all subjects
curl -s http://localhost:8081/subjects | jq .

# List subjects including soft-deleted
curl -s "http://localhost:8081/subjects?deleted=true" | jq .

# Check global compatibility configuration
curl -s http://localhost:8081/config | jq .

# Check per-subject compatibility
curl -s http://localhost:8081/config/my-subject | jq .

# Check global mode
curl -s http://localhost:8081/mode | jq .

# Check per-subject mode
curl -s http://localhost:8081/mode/my-subject | jq .

# Get schema by global ID
curl -s http://localhost:8081/schemas/ids/1 | jq .

# List supported schema types
curl -s http://localhost:8081/schemas/types | jq .

# Prometheus metrics (filter for schema registry)
curl -s http://localhost:8081/metrics | grep schema_registry

# Enable debug logging at startup
SCHEMA_REGISTRY_LOG_LEVEL=debug ./schema-registry --config config.yaml

# Docker: view container logs
docker logs schema-registry

# Docker: enable debug logging
docker run -d -p 8081:8081 \
  -e SCHEMA_REGISTRY_LOG_LEVEL=debug \
  ghcr.io/axonops/axonops-schema-registry:latest
```

---

## Getting Help

- **GitHub Issues**: [github.com/axonops/axonops-schema-registry/issues](https://github.com/axonops/axonops-schema-registry/issues)
- Enable debug logging for detailed diagnostics before reporting issues.
- When filing a bug report, include:
  - The error code and full error message
  - Relevant server log output (with debug logging enabled)
  - The request that triggered the error (redact sensitive credentials)
  - Storage backend type and version
  - Registry version or container image tag

---

See also: [Configuration](configuration.md) | [API Reference](api-reference.md) | [Authentication](authentication.md) | [Compatibility](compatibility.md) | [Storage Backends](storage-backends.md) | [Migration](migration.md)
