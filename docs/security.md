# Security

## Overview

Security in AxonOps Schema Registry spans authentication, authorization, transport encryption, rate limiting, and audit logging. All security features are optional and can be enabled independently. When no security configuration is present, the registry operates in open mode with all endpoints accessible without credentials.

For detailed coverage of authentication methods (Basic Auth, API keys, LDAP, OIDC, JWT, mTLS), user management, and the admin CLI, see the [Authentication](authentication.md) guide. This document focuses on transport security, access control policies, rate limiting, audit logging, and operational hardening.

## Transport Layer Security (TLS)

Enable HTTPS by providing a certificate and private key pair. The registry supports configurable minimum TLS versions, client certificate authentication (mTLS), and automatic certificate reloading.

### Configuration

```yaml
security:
  tls:
    enabled: true
    cert_file: /path/to/server.crt
    key_file: /path/to/server.key
    ca_file: /path/to/ca.crt
    min_version: "TLS1.2"
    client_auth: verify
    auto_reload: true
```

| Field | Description | Default |
|-------|-------------|---------|
| `enabled` | Enable HTTPS | `false` |
| `cert_file` | Path to PEM-encoded server certificate | (required when enabled) |
| `key_file` | Path to PEM-encoded private key | (required when enabled) |
| `ca_file` | Path to CA certificate for verifying client certificates | `""` |
| `min_version` | Minimum TLS version (`TLS1.0`, `TLS1.1`, `TLS1.2`, `TLS1.3`) | `TLS1.2` |
| `client_auth` | Client certificate policy (see below) | `none` |
| `auto_reload` | Reload certificates from disk without restart | `false` |

### Client Certificate Modes

| Mode | Behavior |
|------|----------|
| `none` | No client certificate requested |
| `request` | Client certificate requested but not required |
| `require` | Client certificate required but not verified against CA |
| `verify` | Client certificate required and verified against the CA in `ca_file` |

Use `verify` for mutual TLS (mTLS) authentication. This requires a valid `ca_file` containing the certificate authority that signed the client certificates.

### Certificate Reload

When `auto_reload: true`, the registry reloads the server certificate and key from disk on each new TLS handshake without requiring a process restart. This supports zero-downtime certificate rotation, which is useful with automated certificate management tools such as cert-manager or ACME clients.

### TLS Termination at a Load Balancer

If TLS is terminated at a reverse proxy or load balancer (e.g., NGINX, HAProxy, AWS ALB), disable TLS on the registry itself and configure the proxy to forward requests over plain HTTP. Ensure the proxy sets `X-Forwarded-For` and `X-Real-IP` headers so that rate limiting and audit logging record the correct client IP.

## Role-Based Access Control (RBAC)

The registry uses a fixed set of four built-in roles with hierarchical permissions. Roles cannot be customized, but the `super_admins` list grants unrestricted access to specific usernames regardless of their assigned role.

### Permission Matrix

| Role | Schema Read | Schema Write | Schema Delete | Config Read | Config Write | Mode Read | Mode Write | Import | User Mgmt |
|------|:-----------:|:------------:|:-------------:|:-----------:|:------------:|:---------:|:----------:|:------:|:---------:|
| `super_admin` | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Full |
| `admin` | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Read only |
| `developer` | Yes | Yes | No | Yes | No | Yes | No | No | No |
| `readonly` | Yes | No | No | Yes | No | Yes | No | No | No |

Permissions are mapped to API endpoints as follows:

| Permission | Applies to |
|------------|-----------|
| `schema:read` | `GET /subjects/*`, `GET /schemas/*`, `POST /compatibility/*` |
| `schema:write` | `POST /subjects/*/versions` |
| `schema:delete` | `DELETE /subjects/*` |
| `config:read` | `GET /config`, `GET /config/*` |
| `config:write` | `PUT /config`, `DELETE /config`, `PUT /config/*`, `DELETE /config/*` |
| `mode:read` | `GET /mode`, `GET /mode/*` |
| `mode:write` | `PUT /mode`, `PUT /mode/*` |
| `import:write` | `POST /import/*` |
| `admin:read` | `GET /admin/*` |
| `admin:write` | `POST/PUT/DELETE /admin/*` |

### Configuration

```yaml
security:
  auth:
    rbac:
      enabled: true
      default_role: readonly
      super_admins:
        - admin
        - ops-lead
```

| Field | Description | Default |
|-------|-------------|---------|
| `enabled` | Enable RBAC enforcement | `false` |
| `default_role` | Role assigned when an authentication method does not provide one | `""` |
| `super_admins` | Usernames with full access to all operations including user management | `[]` |

When RBAC is disabled, all authenticated users have unrestricted access. When enabled, users listed in `super_admins` bypass all permission checks. The `default_role` is applied to users authenticated via methods that do not inherently assign a role (e.g., mTLS, config-based basic auth).

## Credential Storage

### Passwords

User passwords are stored as bcrypt hashes with the default cost factor (10). Plaintext passwords are never written to disk or database. The `golang.org/x/crypto/bcrypt` package is used for hashing, and the cost factor is the `bcrypt.DefaultCost` constant (currently 10). Bcrypt is intentionally slow, which limits brute-force attack throughput.

### API Keys

API keys are stored as SHA-256 hashes by default. When the `api_key.secret` configuration value is set, the registry uses HMAC-SHA256 with that secret as a pepper, providing defense-in-depth: even if the database is compromised, the attacker cannot verify API keys without the secret.

```yaml
security:
  auth:
    api_key:
      secret: "${API_KEY_SECRET}"
```

Key security properties:

- **Hashed at rest** -- keys are never stored in plaintext
- **Shown once** -- the raw key is returned only at creation time and cannot be retrieved afterward
- **Pepper protection** -- when `secret` is configured, HMAC-SHA256 is used instead of plain SHA-256
- **Cluster consistency** -- if `secret` is configured, all registry instances must use the same value
- **Expiration enforced** -- key expiration is checked on every request
- **Key prefixes** -- configurable prefix (e.g., `sr_live_`) helps identify keys in logs and configuration

The `secret` value should be at least 32 bytes of cryptographically random data. Load it from an environment variable or secrets manager rather than hardcoding it in the configuration file.

### External Credential Storage with HashiCorp Vault

Users and API keys can be stored in HashiCorp Vault separately from schema data by setting `storage.auth_type: vault`. This allows you to use one backend (e.g., PostgreSQL) for schemas while keeping credentials in Vault's KV secrets engine.

```yaml
storage:
  type: postgresql
  auth_type: vault
  vault:
    address: "https://vault.example.com:8200"
    token: "${VAULT_TOKEN}"
    mount_path: "secret"
    base_path: "schema-registry"
```

See the [Configuration](configuration.md) guide for the full Vault configuration reference.

## Rate Limiting

The registry implements a token bucket algorithm to protect API endpoints from excessive request volume. Rate limiting is applied as HTTP middleware and operates independently of authentication.

### Configuration

```yaml
security:
  rate_limiting:
    enabled: true
    requests_per_second: 100
    burst_size: 200
    per_client: true
    per_endpoint: false
```

| Field | Description | Default |
|-------|-------------|---------|
| `enabled` | Enable rate limiting | `false` |
| `requests_per_second` | Sustained request rate (token refill rate) | `0` |
| `burst_size` | Maximum burst capacity (token bucket size) | `0` |
| `per_client` | Maintain separate rate limits per source IP | `false` |
| `per_endpoint` | Maintain separate rate limits per API path | `false` |

### Behavior

When a request exceeds the rate limit, the registry responds with HTTP `429 Too Many Requests` and includes the following headers:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Configured requests per second |
| `X-RateLimit-Remaining` | Tokens remaining in the bucket |
| `Retry-After` | Seconds until the client should retry (set to `1`) |

The `X-RateLimit-Limit` and `X-RateLimit-Remaining` headers are included on all responses, not only rate-limited ones.

### Scope

Rate limiting modes are mutually exclusive in order of precedence:

1. **Per-client** (`per_client: true`) -- each source IP gets its own token bucket. Client IP is determined from `X-Forwarded-For`, `X-Real-IP`, or `RemoteAddr` in that order.
2. **Per-endpoint** (`per_endpoint: true`) -- each combination of HTTP method and path gets its own bucket (e.g., `GET:/subjects` is separate from `POST:/subjects/test/versions`).
3. **Global** (both false) -- a single shared bucket applies to all requests.

### Exempt Endpoints

The health check (`GET /`) and Prometheus metrics (`GET /metrics`) endpoints are served before the rate limiting middleware is applied and are never rate-limited.

### Stale Client Cleanup

When using per-client rate limiting, the registry provides a `CleanupStaleClients` method that removes token buckets for clients that have not made a request within a configurable duration. This prevents unbounded memory growth in environments with many transient clients.

## Audit Logging

The audit logger records security-relevant events to a structured JSON log, either to a file or to standard output. Events are recorded after the request is processed, capturing the outcome (status code, duration).

### Configuration

```yaml
security:
  audit:
    enabled: true
    log_file: /var/log/axonops-schema-registry/audit.log
    events:
      - schema_register
      - schema_delete
      - config_change
      - auth_failure
      - auth_forbidden
      - subject_delete
    include_body: false
```

| Field | Description | Default |
|-------|-------------|---------|
| `enabled` | Enable audit logging | `false` |
| `log_file` | Path to the audit log file (stdout if empty) | `""` |
| `events` | List of event types to record (all security-relevant events if empty) | `[]` |
| `include_body` | Include the request body in audit entries (truncated to 1000 characters) | `false` |

### Event Types

| Event | Trigger |
|-------|---------|
| `schema_register` | `POST /subjects/{subject}/versions` |
| `schema_delete` | `DELETE /subjects/{subject}/versions/{version}` |
| `schema_get` | `GET /subjects/{subject}/versions/*` or `GET /schemas/ids/*` |
| `schema_lookup` | `POST /subjects/{subject}` |
| `config_get` | `GET /config` or `GET /config/{subject}` |
| `config_update` | `PUT /config` or `PUT /config/{subject}` |
| `config_delete` | `DELETE /config` or `DELETE /config/{subject}` |
| `mode_get` | `GET /mode` or `GET /mode/{subject}` |
| `mode_update` | `PUT /mode` or `PUT /mode/{subject}` |
| `auth_success` | Successful authentication |
| `auth_failure` | HTTP 401 response (authentication failed) |
| `auth_forbidden` | HTTP 403 response (authorization failed) |
| `subject_delete` | `DELETE /subjects/{subject}` |
| `subject_list` | `GET /subjects` |

When the `events` list is empty, the following events are logged by default: `schema_register`, `schema_delete`, `config_update`, `mode_update`, `auth_failure`, `auth_forbidden`, and `subject_delete`.

### Log Format

Each audit entry is a JSON object written to a single line:

```json
{
  "timestamp": "2026-02-16T10:30:00Z",
  "event_type": "schema_register",
  "user": "jane",
  "role": "developer",
  "client_ip": "192.168.1.50",
  "method": "POST",
  "path": "/subjects/payments-value/versions",
  "status_code": 200,
  "duration_ms": 42,
  "subject": "payments-value"
}
```

Fields include: `timestamp`, `event_type`, `user`, `role`, `client_ip`, `method`, `path`, `status_code`, `duration_ms`, `subject`, `version`, `schema_id`, `error`, and optionally `request_body` and `metadata`.

## Unauthenticated Endpoints

The following endpoints are always accessible without credentials, regardless of authentication configuration:

| Endpoint | Purpose |
|----------|---------|
| `GET /` | Health check |
| `GET /metrics` | Prometheus metrics |

When `docs_enabled: true` in the server configuration:

| Endpoint | Purpose |
|----------|---------|
| `GET /docs` | Swagger UI |
| `GET /openapi.yaml` | OpenAPI specification |

These endpoints are registered outside the authentication middleware chain and are also exempt from rate limiting.

## Security Hardening Checklist

1. **Enable TLS** with a minimum version of TLS 1.2 (`min_version: "TLS1.2"`), or terminate TLS at a load balancer and restrict the registry to private network traffic.
2. **Enable authentication** with at least one method (`basic`, `api_key`, `jwt`, `oidc`, or `mtls`).
3. **Enable RBAC** with a restrictive `default_role` (e.g., `readonly`) and explicitly assign higher-privilege roles only where needed.
4. **Use API keys with expiration** for programmatic access. Rotate keys regularly via the admin API or CLI.
5. **Configure the API key HMAC secret** (`api_key.secret`) on all registry instances and store it in an environment variable or secrets manager.
6. **Enable rate limiting** to prevent abuse and reduce the impact of credential brute-force attempts.
7. **Enable audit logging** and forward logs to a centralized system for monitoring and alerting.
8. **Use environment variables or Vault for secrets** -- never hardcode passwords, API key secrets, or Vault tokens in configuration files. The registry supports `${ENV_VAR}` substitution in YAML configuration.
9. **Run as a non-root user** -- the Docker image runs as UID/GID 1000 (`schemaregistry` user) by default.
10. **Configure CORS appropriately** if the registry serves browser-based clients.
11. **Restrict network access** -- bind the registry to an internal interface or use firewall rules to limit access to trusted networks.
12. **Set `client_auth: verify`** when using mTLS to ensure client certificates are validated against your CA.
13. **Review super_admins list regularly** -- users in this list bypass all RBAC checks.

## Related Documentation

- [Authentication](authentication.md) -- authentication methods, user management, API key lifecycle, admin CLI
- [Configuration](configuration.md) -- full configuration reference including all security options
- [Deployment](deployment.md) -- production deployment guidance
