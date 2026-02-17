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

**Symptom**: `curl: (7) Failed to connect to localhost port 8081: Connection refused`

**Causes and fixes**:

**Server not running.** Verify the process or container is running:

```bash
# Check if the process is running
pgrep -f schema-registry

# Check container status
docker ps -a --filter name=schema-registry
```

**Wrong port.** The default port is 8081. If you changed it, confirm the `server.port` value in your [configuration file](configuration.md):

```yaml
server:
  port: 8081
```

**Binding to wrong interface.** By default the server may bind to `127.0.0.1`, which rejects connections from other hosts. Set `server.host` to `0.0.0.0` to accept connections on all interfaces:

```yaml
server:
  host: "0.0.0.0"
```

**Docker networking.** When running in Docker, ensure the port is mapped correctly:

```bash
# Map port explicitly
docker run -d -p 8081:8081 ghcr.io/axonops/axonops-schema-registry:latest

# Or use host networking
docker run -d --network host ghcr.io/axonops/axonops-schema-registry:latest
```

If connecting from another container, use the container name or Docker network IP rather than `localhost`.

---

### Database Connection Errors

**Symptom**: Server fails to start with a storage connection error, or returns 50002 errors after startup.

#### PostgreSQL

- Verify host, port, username, password, and database name in your configuration.
- Check that `pg_hba.conf` allows connections from the registry host.
- Ensure the SSL mode in your configuration matches the server's TLS settings (`disable`, `require`, `verify-ca`, `verify-full`).
- Test connectivity directly:

```bash
psql -h HOST -p PORT -U USER -d DATABASE
```

See [Storage Backends -- PostgreSQL](storage-backends.md) for connection pool tuning options.

#### MySQL

- Verify host, port, username, password, and database name.
- Confirm the user has sufficient privileges:

```sql
GRANT ALL PRIVILEGES ON schemaregistry.* TO 'user'@'%';
FLUSH PRIVILEGES;
```

- The TLS mode must match the server configuration. Common values: `preferred`, `skip-verify`, `true`.
- Test connectivity directly:

```bash
mysql -h HOST -P PORT -u USER -p DATABASE
```

See [Storage Backends -- MySQL](storage-backends.md) for connection pool tuning options.

#### Cassandra

- Verify that the contact point hosts and port are correct.
- The keyspace is created automatically on first startup when `migrate: true` is set. If `migrate` is disabled, create the keyspace manually before starting the registry.
- Cassandra 5.0 or later is required for SAI (Storage Attached Index) support.
- The consistency level must match your cluster topology. For example, `QUORUM` requires a majority of replicas to be available; `LOCAL_ONE` is sufficient for single-datacenter reads.
- Test connectivity directly:

```bash
cqlsh HOST PORT
```

See [Storage Backends -- Cassandra](storage-backends.md) for detailed configuration.

---

### Schema Registration Fails

#### 42201 Invalid Schema

The schema content is malformed or violates type-specific rules. Common causes:

- **Malformed JSON.** The schema string is not valid JSON.
- **Missing required fields.** For example, an Avro record type must include a `fields` array.
- **Invalid type references.** A field references a named type that is not defined within the schema or its declared references.
- **Protobuf syntax errors.** The `.proto` content does not parse correctly.
- **Unresolved references.** The `references` array in the registration request points to subjects or versions that do not exist.

Verify the schema parses correctly in isolation before registering it.

#### 409 Incompatible Schema

The new schema violates the compatibility rules configured for the subject. To diagnose:

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

3. If you need to register the schema regardless, temporarily set compatibility to `NONE`:

```bash
curl -X PUT \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "NONE"}' \
  http://localhost:8081/config/my-subject
```

Remember to restore the desired compatibility level afterward. See [Compatibility](compatibility.md) for details on each level.

#### 42205 Operation Not Permitted

The subject or global mode is set to `READONLY` or `READWRITE` is not enabled for the operation you are attempting.

Check the current mode:

```bash
# Global mode
curl -s http://localhost:8081/mode | jq .

# Per-subject mode
curl -s http://localhost:8081/mode/my-subject | jq .
```

Set the mode to `READWRITE` to allow writes:

```bash
curl -X PUT \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"mode": "READWRITE"}' \
  http://localhost:8081/mode/my-subject
```

---

### Authentication Issues

#### 401 Unauthorized

Authentication is enabled but the request was rejected. Common causes:

- **No credentials provided.** When authentication is enabled, every request must include credentials. See [Authentication](authentication.md) for supported methods.
- **Invalid username or password.** Verify the credentials are correct for basic auth.
- **Expired API key.** Check the key's expiration date. Expired keys return error code 40103.
- **Disabled API key or user.** Disabled keys return 40104; disabled users return 40105.
- **LDAP bind failure.** Verify `bind_dn` and `bind_password` in the LDAP configuration. Ensure the LDAP server is reachable.
- **OIDC token issues.** The token may be expired, the audience claim may not match the configured value, or the issuer URL may be unreachable.

#### 403 Forbidden

The user authenticated successfully but lacks the required role for the operation.

- Check role assignments: `GET /admin/users/{id}` (requires admin privileges).
- Verify the RBAC role mapping in your [configuration](configuration.md). Admin endpoints under `/admin/` require the `admin` role.

---

### Soft Delete Confusion

#### 40404 Subject Was Soft-Deleted

The subject exists but was previously soft-deleted. It will not appear in normal listings.

To view soft-deleted subjects:

```bash
curl -s "http://localhost:8081/subjects?deleted=true" | jq .
```

To permanently delete a soft-deleted subject:

```bash
curl -X DELETE "http://localhost:8081/subjects/my-subject?permanent=true"
```

#### 40405 Subject Not Soft-Deleted

You attempted a permanent delete on a subject that has not been soft-deleted first. The deletion process requires two steps:

```bash
# Step 1: Soft-delete
curl -X DELETE http://localhost:8081/subjects/my-subject

# Step 2: Permanent delete
curl -X DELETE "http://localhost:8081/subjects/my-subject?permanent=true"
```

#### 42206 Reference Exists

You attempted to delete a schema or subject that is referenced by another schema. Remove or update the referencing schemas first, then retry the deletion.

---

### Performance Issues

#### Slow Schema Registration

- Check storage latency via Prometheus metrics: `schema_registry_storage_latency_seconds`.
- **PostgreSQL**: Increase `max_open_conns` if connection pool exhaustion is causing queuing. Check for lock contention with `pg_stat_activity`.
- **MySQL**: Similar pool tuning via `max_open_conns` and `max_idle_conns`.
- **Cassandra**: Consider using `LOCAL_ONE` consistency for reads if strong consistency is not required. Verify SAI indexes are healthy with `nodetool`.
- Enable debug logging to identify the bottleneck (see [Diagnostic Commands](#diagnostic-commands)).

#### High Memory Usage

- The number of registered schemas directly affects memory usage, as schemas are cached for fast lookups.
- For large registries, increase container memory limits.
- Check the total schema count via the `/schemas` endpoint or Prometheus metrics.

---

### Import and Migration Issues

#### Schema ID Conflicts

When using the import API (`POST /import/schemas`), the registry preserves the original schema IDs. If a schema with the same ID already exists with different content, the import will fail.

Ensure the global mode is set to `IMPORT` before importing:

```bash
curl -X PUT \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"mode": "IMPORT"}' \
  http://localhost:8081/mode
```

After import completes, restore the mode to `READWRITE`.

#### Missing References

Schemas that reference other schemas (via Avro named types, Protobuf imports, or JSON Schema `$ref`) must be imported in dependency order. Import referenced schemas first (lowest ID), then the schemas that depend on them.

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
