# Configuration Reference

This document provides a complete reference for all configuration options available in AxonOps Schema Registry.

## Contents

- [Configuration File](#configuration-file)
- [Environment Variable Substitution](#environment-variable-substitution)
- [Server](#server)
- [Storage](#storage)
  - [In-Memory](#in-memory)
  - [PostgreSQL](#postgresql)
  - [MySQL](#mysql)
  - [Cassandra](#cassandra)
  - [HashiCorp Vault (Auth Storage)](#hashicorp-vault-auth-storage)
- [Compatibility](#compatibility)
- [Logging](#logging)
- [Security](#security)
  - [TLS](#tls)
  - [Authentication](#authentication)
  - [Bootstrap Admin User](#bootstrap-admin-user)
  - [Basic Authentication](#basic-authentication)
  - [API Key Authentication](#api-key-authentication)
  - [JWT Authentication](#jwt-authentication)
  - [LDAP Authentication](#ldap-authentication)
  - [OpenID Connect (OIDC)](#openid-connect-oidc)
  - [Role-Based Access Control (RBAC)](#role-based-access-control-rbac)
  - [Rate Limiting](#rate-limiting)
  - [Audit Logging](#audit-logging)
  - [Per-Principal Metrics](#per-principal-metrics)
- [MCP Server](#mcp-server)
- [Environment Variables](#environment-variables)
- [Complete Configuration Example](#complete-configuration-example)

---

## Configuration File

The registry accepts a YAML configuration file via the `--config` command-line flag:

```bash
schema-registry --config /etc/schema-registry/config.yaml
```

A `--version` flag is also available to print version information and exit:

```bash
schema-registry --version
```

If no configuration file is specified, built-in defaults are used (in-memory storage, port 8081, BACKWARD compatibility).

## Environment Variable Substitution

All values in the configuration file support environment variable substitution using standard shell syntax. The registry expands `${VAR_NAME}` references before parsing the YAML.

```yaml
storage:
  postgresql:
    password: ${PG_PASSWORD}
```

You can provide a default value with `${VAR_NAME:-default}` syntax:

```yaml
server:
  port: ${REGISTRY_PORT:-8081}
```

In addition, a set of dedicated environment variables (documented in the [Environment Variables](#environment-variables) section) override the corresponding configuration file values after the file is loaded.

---

## Server

HTTP server settings.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `server.host` | string | `"0.0.0.0"` | Address the HTTP server binds to. |
| `server.port` | int | `8081` | Port the HTTP server listens on. Must be 1--65535. |
| `server.read_timeout` | int | `30` | Maximum duration (seconds) for reading the entire request, including the body. |
| `server.write_timeout` | int | `30` | Maximum duration (seconds) before timing out writes of the response. |
| `server.docs_enabled` | bool | `false` | When `true`, serves Swagger UI at `/docs` and the OpenAPI specification at `/openapi.yaml`. |
| `server.shutdown_timeout` | int | `30` | Maximum duration (seconds) to wait for in-flight requests during graceful shutdown. |
| `server.cluster_id` | string | `""` | Optional cluster identifier, exposed via MCP server info. |
| `server.max_request_body_size` | int64 | `0` | Maximum request body size in bytes. `0` uses the default of 10 MB. |

```yaml
server:
  host: "0.0.0.0"
  port: 8081
  read_timeout: 30
  write_timeout: 30
  shutdown_timeout: 30
  docs_enabled: false
```

---

## Storage

The `storage` section selects and configures the persistence backend for schemas, subjects, and configuration state.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `storage.type` | string | `"memory"` | Backend type. Valid values: `memory`, `postgresql`, `mysql`, `cassandra`. |
| `storage.auth_type` | string | `""` (same as `type`) | Separate backend for authentication data. Valid values: `vault`, `postgresql`, `mysql`, `cassandra`, `memory`. When empty, authentication data is stored in the same backend as schema data. |

For detailed guidance on choosing and operating each backend, see [Storage Backends](storage-backends.md).

### In-Memory

The in-memory backend requires no additional configuration. All data is lost when the process exits. Useful for development and testing.

```yaml
storage:
  type: memory
```

### PostgreSQL

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `storage.postgresql.host` | string | `"localhost"` | PostgreSQL server hostname. |
| `storage.postgresql.port` | int | `5432` | PostgreSQL server port. |
| `storage.postgresql.database` | string | `"schema_registry"` | Database name. |
| `storage.postgresql.user` | string | `""` | Connection username. |
| `storage.postgresql.password` | string | `""` | Connection password. |
| `storage.postgresql.ssl_mode` | string | `"disable"` | PostgreSQL SSL mode. Values: `disable`, `require`, `verify-ca`, `verify-full`. |
| `storage.postgresql.max_open_conns` | int | `25` | Maximum number of open connections in the pool. |
| `storage.postgresql.max_idle_conns` | int | `5` | Maximum number of idle connections retained in the pool. |
| `storage.postgresql.conn_max_lifetime` | int | `300` | Maximum lifetime of a connection in seconds. |

```yaml
storage:
  type: postgresql
  postgresql:
    host: localhost
    port: 5432
    database: schema_registry
    user: registry
    password: ${PG_PASSWORD}
    ssl_mode: disable
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300
```

### MySQL

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `storage.mysql.host` | string | `"localhost"` | MySQL server hostname. |
| `storage.mysql.port` | int | `3306` | MySQL server port. |
| `storage.mysql.database` | string | `"schema_registry"` | Database name. |
| `storage.mysql.user` | string | `""` | Connection username. |
| `storage.mysql.password` | string | `""` | Connection password. |
| `storage.mysql.tls` | string | `"false"` | TLS mode. Values: `true`, `false`, `skip-verify`, `preferred`. |
| `storage.mysql.max_open_conns` | int | `25` | Maximum number of open connections in the pool. |
| `storage.mysql.max_idle_conns` | int | `5` | Maximum number of idle connections retained in the pool. |
| `storage.mysql.conn_max_lifetime` | int | `300` | Maximum lifetime of a connection in seconds. |

```yaml
storage:
  type: mysql
  mysql:
    host: localhost
    port: 3306
    database: schema_registry
    user: registry
    password: ${MYSQL_PASSWORD}
    tls: "false"
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300
```

### Cassandra

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `storage.cassandra.hosts` | list of strings | `["localhost"]` | Cassandra contact points. |
| `storage.cassandra.port` | int | `9042` | Cassandra native transport port. |
| `storage.cassandra.keyspace` | string | `"schema_registry"` | Keyspace name. Created automatically with migrations. |
| `storage.cassandra.local_dc` | string | `""` | Local datacenter name. When set, enables datacenter-aware routing (`DCAwareRoundRobinPolicy`). REQUIRED for multi-datacenter deployments. |
| `storage.cassandra.consistency` | string | `"LOCAL_QUORUM"` | Default consistency level for all operations. Used when `read_consistency` or `write_consistency` is not set. |
| `storage.cassandra.read_consistency` | string | `""` (falls back to `consistency`) | Consistency level for read operations. Useful in multi-datacenter deployments where read latency matters (e.g., `LOCAL_ONE`). |
| `storage.cassandra.write_consistency` | string | `""` (falls back to `consistency`) | Consistency level for write operations. Set independently for durability requirements (e.g., `LOCAL_QUORUM`). |
| `storage.cassandra.serial_consistency` | string | `"LOCAL_SERIAL"` | Serial consistency level for Lightweight Transactions (LWT). Controls the Paxos consensus scope for `IF NOT EXISTS` and conditional update operations. Values: `SERIAL` (cross-datacenter) or `LOCAL_SERIAL` (local datacenter only). `LOCAL_SERIAL` is strongly RECOMMENDED for multi-datacenter deployments to avoid cross-DC Paxos latency. |
| `storage.cassandra.username` | string | `""` | Authentication username. |
| `storage.cassandra.password` | string | `""` | Authentication password. |
| `storage.cassandra.timeout` | duration | `"10s"` | Timeout for query operations. |
| `storage.cassandra.connect_timeout` | duration | `"10s"` | Timeout for initial connection establishment. |
| `storage.cassandra.max_retries` | int | `50` | Maximum retry attempts for CAS (compare-and-swap) operations during ID allocation and fingerprint deduplication. |
| `storage.cassandra.id_block_size` | int | `50` | Number of schema IDs reserved per LWT call. Higher values reduce LWT frequency but MAY leave gaps in the ID sequence on crash. |

Schema migrations run automatically on startup.

```yaml
storage:
  type: cassandra
  cassandra:
    hosts:
      - node1.cassandra.local
      - node2.cassandra.local
      - node3.cassandra.local
    port: 9042
    keyspace: schema_registry
    local_dc: dc1
    consistency: LOCAL_QUORUM
    read_consistency: LOCAL_ONE
    write_consistency: LOCAL_QUORUM
    serial_consistency: LOCAL_SERIAL
    username: registry
    password: ${CASSANDRA_PASSWORD}
    timeout: 10s
    connect_timeout: 10s
```

### HashiCorp Vault (Auth Storage)

Vault is available as a dedicated authentication storage backend. Set `storage.auth_type: vault` to store users and API keys in Vault while keeping schema data in a separate backend. See [Authentication](authentication.md) for details.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `storage.vault.address` | string | `""` | Vault server address (e.g., `http://localhost:8200`). Required when `auth_type` is `vault`. |
| `storage.vault.token` | string | `""` | Vault authentication token. Can also be set via `VAULT_TOKEN`. |
| `storage.vault.namespace` | string | `""` | Vault namespace (enterprise feature). Can also be set via `VAULT_NAMESPACE`. |
| `storage.vault.mount_path` | string | `"secret"` | KV secrets engine mount path. |
| `storage.vault.base_path` | string | `"schema-registry"` | Base path within the secrets engine for all registry data. |
| `storage.vault.tls_cert_file` | string | `""` | Path to client TLS certificate for Vault communication. |
| `storage.vault.tls_key_file` | string | `""` | Path to client TLS private key. |
| `storage.vault.tls_ca_file` | string | `""` | Path to CA certificate for verifying the Vault server. |
| `storage.vault.tls_skip_verify` | bool | `false` | Skip TLS certificate verification. Not recommended for production. |

```yaml
storage:
  type: postgresql
  auth_type: vault
  postgresql:
    host: db.internal
    # ... schema storage config
  vault:
    address: https://vault.internal:8200
    token: ${VAULT_TOKEN}
    namespace: production
    mount_path: secret
    base_path: schema-registry
    tls_ca_file: /etc/ssl/certs/vault-ca.pem
```

---

## Compatibility

Controls the default schema compatibility level applied to new subjects.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `compatibility.default_level` | string | `"BACKWARD"` | Default compatibility level for new subjects. |

Valid compatibility levels:

| Level | Description |
|-------|-------------|
| `NONE` | No compatibility checking. |
| `BACKWARD` | New schema can read data written by the previous schema version. |
| `BACKWARD_TRANSITIVE` | New schema can read data written by all previous schema versions. |
| `FORWARD` | Previous schema can read data written by the new schema. |
| `FORWARD_TRANSITIVE` | All previous schemas can read data written by the new schema. |
| `FULL` | Both backward and forward compatible with the previous schema version. |
| `FULL_TRANSITIVE` | Both backward and forward compatible with all previous schema versions. |

Compatibility can also be overridden per subject via the `/config/{subject}` API endpoint.

```yaml
compatibility:
  default_level: BACKWARD
```

---

## Logging

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `logging.level` | string | `"info"` | Minimum log level. Values: `debug`, `info`, `warn`, `error`. |
| `logging.format` | string | `"json"` | Log output format. Values: `json`, `text`. |

```yaml
logging:
  level: info
  format: json
```

---

## Security

The `security` section contains TLS, authentication, rate limiting, and audit configuration. See [Security](security.md) for deployment guidance.

### TLS

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.tls.enabled` | bool | `false` | Enable TLS on the HTTP server. |
| `security.tls.cert_file` | string | `""` | Path to the server TLS certificate. |
| `security.tls.key_file` | string | `""` | Path to the server TLS private key. |
| `security.tls.ca_file` | string | `""` | Path to a CA certificate for verifying client certificates (mTLS). |
| `security.tls.min_version` | string | `""` | Minimum TLS version. Values: `TLS1.2`, `TLS1.3`. |
| `security.tls.client_auth` | string | `"none"` | Client certificate policy. Values: `none`, `request`, `require`, `verify`. |
| `security.tls.auto_reload` | bool | `false` | When `true`, sending `SIGHUP` to the process reloads TLS certificates from disk without restarting. New connections use the updated certificates; existing connections are unaffected. |

```yaml
security:
  tls:
    enabled: true
    cert_file: /etc/ssl/certs/registry.pem
    key_file: /etc/ssl/private/registry-key.pem
    min_version: "TLS1.2"
    auto_reload: true
```

### Authentication

Top-level authentication settings. See [Authentication](authentication.md) for a full walkthrough.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.auth.enabled` | bool | `false` | Enable authentication middleware. When `false`, all requests are unauthenticated. |
| `security.auth.methods` | list of strings | `[]` | Authentication methods to try, in order. Values: `basic`, `api_key`, `jwt`, `oidc`, `mtls`. |

```yaml
security:
  auth:
    enabled: true
    methods:
      - api_key
      - basic
```

### Bootstrap Admin User

Creates an initial admin user on startup when the users table is empty. Designed for first-run provisioning. The password SHOULD always be set via the `SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD` environment variable rather than in the configuration file.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.auth.bootstrap.enabled` | bool | `false` | Attempt to create an admin user on startup if the users table is empty. |
| `security.auth.bootstrap.username` | string | `""` | Username for the bootstrap admin. |
| `security.auth.bootstrap.password` | string | `""` | Password for the bootstrap admin. Use `SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD` instead of placing this in the file. |
| `security.auth.bootstrap.email` | string | `""` | Email address for the bootstrap admin (optional). |

```yaml
security:
  auth:
    bootstrap:
      enabled: true
      username: admin
      password: ${SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD}
      email: admin@example.com
```

### Basic Authentication

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.auth.basic.realm` | string | `""` | HTTP Basic Auth realm displayed to clients. |
| `security.auth.basic.users` | map (string to string) | `{}` | Static username-to-bcrypt-hash map. For config-based auth only; prefer database-managed users. |
| `security.auth.basic.htpasswd_file` | string | `""` | Path to an Apache-style htpasswd file. Only bcrypt hashes (`$2y$`, `$2a$`, `$2b$`) are supported. Loaded once at startup. Users from this file receive the `rbac.default_role`. |

```yaml
security:
  auth:
    basic:
      realm: "Schema Registry"
```

### API Key Authentication

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.auth.api_key.header` | string | `"X-API-Key"` | HTTP header used to transmit the API key. |
| `security.auth.api_key.query_param` | string | `"api_key"` | Query parameter name accepted as an alternative to the header. |
| `security.auth.api_key.storage_type` | string | `"database"` | API key storage backend. `"database"` (default) stores keys in the database with full CRUD via admin API. `"memory"` loads keys from the `keys` list in config — useful for simple single-server deployments. |
| `security.auth.api_key.secret` | string | `""` | HMAC-SHA256 pepper for hashing API keys before storage. Provides defense-in-depth: even if the database is compromised, keys cannot be verified without this secret. SHOULD be at least 32 bytes of random data. If empty, falls back to plain SHA-256 hashing. |
| `security.auth.api_key.key_prefix` | string | `"sr_"` | Prefix prepended to generated API keys for identification (e.g., `sr_live_abc123`). |
| `security.auth.api_key.cache_refresh_seconds` | int | `60` | How often (seconds) the in-memory API key cache is refreshed from the database. Ensures cluster-wide consistency. Set to `0` to disable caching. |
| `security.auth.api_key.keys` | list | `[]` | Config-defined API keys (used when `storage_type` is `"memory"`). Each entry has `name`, `key_hash` (bcrypt), and `role`. |

```yaml
security:
  auth:
    api_key:
      header: "X-API-Key"
      query_param: "api_key"
      storage_type: database
      secret: ${API_KEY_SECRET}
      key_prefix: "sr_"
      cache_refresh_seconds: 60
      # keys:                   # Used when storage_type is "memory"
      #   - name: ci-pipeline
      #     key_hash: "$2a$10$..."
      #     role: developer
```

### JWT Authentication

For integrating with external identity providers that issue JWTs.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.auth.jwt.issuer` | string | `""` | Expected `iss` claim value. |
| `security.auth.jwt.audience` | string | `""` | Expected `aud` claim value. |
| `security.auth.jwt.jwks_url` | string | `""` | URL of the JSON Web Key Set endpoint for signature verification. |
| `security.auth.jwt.public_key_file` | string | `""` | Path to a PEM-encoded public key file (alternative to JWKS). |
| `security.auth.jwt.algorithm` | string | `""` | Signing algorithm. Values: `RS256`, `ES256`. |
| `security.auth.jwt.claims_mapping` | map (string to string) | `{}` | Maps JWT claims to internal fields (e.g., `username: sub`, `role: role`). |
| `security.auth.jwt.default_role` | string | `"readonly"` | Fallback role assigned when no JWT claim matches a role mapping. |
| `security.auth.jwt.jwks_cache_ttl` | int | `300` | Time in seconds to cache JWKS keys before re-fetching. |
| `security.auth.jwt.http_timeout` | int | `10` | HTTP client timeout in seconds for JWKS endpoint requests. |

```yaml
security:
  auth:
    jwt:
      issuer: "https://auth.example.com"
      audience: "schema-registry"
      jwks_url: "https://auth.example.com/.well-known/jwks.json"
      algorithm: RS256
      claims_mapping:
        username: sub
        role: role
      default_role: readonly
      jwks_cache_ttl: 300
      http_timeout: 10
```

### LDAP Authentication

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.auth.ldap.enabled` | bool | `false` | Enable LDAP authentication. |
| `security.auth.ldap.url` | string | `""` | LDAP server URL (`ldap://host:389` or `ldaps://host:636`). |
| `security.auth.ldap.bind_dn` | string | `""` | Distinguished name of the service account used for searches. |
| `security.auth.ldap.bind_password` | string | `""` | Password for the service account. |
| `security.auth.ldap.base_dn` | string | `""` | Base DN for all LDAP searches. |
| `security.auth.ldap.user_search_filter` | string | `""` | LDAP filter for finding users (e.g., `(sAMAccountName=%s)`). `%s` is replaced with the login username. |
| `security.auth.ldap.user_search_base` | string | `""` | Base DN for user searches (e.g., `OU=Users,DC=example,DC=com`). |
| `security.auth.ldap.group_search_filter` | string | `""` | LDAP filter for finding group memberships (e.g., `(member=%s)`). `%s` is replaced with the user's DN. When set along with `group_search_base`, groups are discovered via an explicit LDAP search in addition to the `memberOf` attribute. |
| `security.auth.ldap.group_search_base` | string | `""` | Base DN for group searches (e.g., `OU=Groups,DC=example,DC=com`). Used together with `group_search_filter` to discover groups via LDAP search. |
| `security.auth.ldap.username_attribute` | string | `""` | LDAP attribute containing the username (`sAMAccountName`, `uid`, `userPrincipalName`). |
| `security.auth.ldap.email_attribute` | string | `""` | LDAP attribute containing the email address (`mail`). |
| `security.auth.ldap.group_attribute` | string | `""` | LDAP attribute containing group membership (`memberOf`). |
| `security.auth.ldap.role_mapping` | map (string to string) | `{}` | Maps LDAP group names to registry roles. |
| `security.auth.ldap.default_role` | string | `""` | Role assigned when no LDAP group matches a mapping. |
| `security.auth.ldap.start_tls` | bool | `false` | Upgrade an unencrypted connection using STARTTLS. |
| `security.auth.ldap.insecure_skip_verify` | bool | `false` | Skip TLS certificate verification. Not recommended for production. |
| `security.auth.ldap.ca_cert_file` | string | `""` | Path to CA certificate for verifying the LDAP server. |
| `security.auth.ldap.client_cert_file` | string | `""` | Path to client certificate for mTLS authentication to the LDAP server. |
| `security.auth.ldap.client_key_file` | string | `""` | Path to client private key for mTLS authentication to the LDAP server. |
| `security.auth.ldap.allow_fallback` | bool | `true` | When `true`, users not found in LDAP fall back to other configured auth methods (basic/API key). Users that exist in LDAP but provide wrong passwords are always rejected (no fallback). Set to `false` for strict LDAP-only authentication. |
| `security.auth.ldap.connection_timeout` | int | `10` | Connection timeout in seconds. |
| `security.auth.ldap.request_timeout` | int | `30` | Request timeout in seconds. |

```yaml
security:
  auth:
    ldap:
      enabled: true
      url: ldaps://ldap.example.com:636
      bind_dn: "CN=svc-registry,OU=ServiceAccounts,DC=example,DC=com"
      bind_password: ${LDAP_BIND_PASSWORD}
      base_dn: "DC=example,DC=com"
      user_search_filter: "(sAMAccountName=%s)"
      user_search_base: "OU=Users,DC=example,DC=com"
      group_search_filter: "(member=%s)"
      group_search_base: "OU=Groups,DC=example,DC=com"
      username_attribute: sAMAccountName
      email_attribute: mail
      group_attribute: memberOf
      role_mapping:
        "CN=SchemaAdmins,OU=Groups,DC=example,DC=com": admin
        "CN=SchemaDevelopers,OU=Groups,DC=example,DC=com": readwrite
      default_role: readonly
      ca_cert_file: /etc/ssl/certs/ldap-ca.pem
      client_cert_file: /etc/ssl/certs/ldap-client.pem
      client_key_file: /etc/ssl/private/ldap-client-key.pem
      allow_fallback: true            # Set to false for strict LDAP-only auth
      connection_timeout: 10
      request_timeout: 30
```

### OpenID Connect (OIDC)

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.auth.oidc.enabled` | bool | `false` | Enable OIDC authentication. |
| `security.auth.oidc.issuer_url` | string | `""` | OIDC issuer URL (e.g., `https://auth.example.com`). Used for discovery. |
| `security.auth.oidc.client_id` | string | `""` | Client ID for token validation. |
| `security.auth.oidc.client_secret` | string | `""` | Client secret. |
| `security.auth.oidc.username_claim` | string | `""` | JWT claim used as the username (`sub`, `preferred_username`, `email`). |
| `security.auth.oidc.roles_claim` | string | `""` | JWT claim containing role information (`roles`, `groups`). |
| `security.auth.oidc.role_mapping` | map (string to string) | `{}` | Maps OIDC roles/groups to registry roles. |
| `security.auth.oidc.default_role` | string | `""` | Role assigned when no claim matches a mapping. |
| `security.auth.oidc.required_audience` | string | `""` | Required value in the `aud` claim. |
| `security.auth.oidc.allowed_algorithms` | list of strings | `[]` | Accepted signing algorithms (e.g., `RS256`, `ES256`). |
| `security.auth.oidc.skip_issuer_check` | bool | `false` | Skip issuer validation. For testing only. |
| `security.auth.oidc.skip_expiry_check` | bool | `false` | Skip token expiry validation. For testing only. |

```yaml
security:
  auth:
    oidc:
      enabled: true
      issuer_url: "https://auth.example.com"
      client_id: "schema-registry"
      client_secret: ${OIDC_CLIENT_SECRET}
      scopes:
        - openid
        - profile
        - email
      username_claim: preferred_username
      roles_claim: groups
      role_mapping:
        schema-admins: admin
        developers: readwrite
      default_role: readonly
```

### Role-Based Access Control (RBAC)

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.auth.rbac.enabled` | bool | `false` | Enable RBAC enforcement. |
| `security.auth.rbac.default_role` | string | `""` | Role assigned to authenticated users with no explicit role. |
| `security.auth.rbac.super_admins` | list of strings | `[]` | Usernames with unrestricted access, including user and API key management. |

```yaml
security:
  auth:
    rbac:
      enabled: true
      default_role: readonly
      super_admins:
        - admin
```

### Rate Limiting

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.rate_limiting.enabled` | bool | `false` | Enable rate limiting middleware. |
| `security.rate_limiting.requests_per_second` | int | `0` | Maximum sustained request rate. |
| `security.rate_limiting.burst_size` | int | `0` | Maximum burst of requests allowed above the sustained rate. |
| `security.rate_limiting.per_client` | bool | `false` | Apply rate limits per client IP address rather than globally. |
| `security.rate_limiting.per_endpoint` | bool | `false` | Apply rate limits per API endpoint rather than globally. |

```yaml
security:
  rate_limiting:
    enabled: true
    requests_per_second: 100
    burst_size: 200
    per_client: true
    per_endpoint: false
```

### Audit Logging

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.audit.enabled` | bool | `false` | Enable audit logging. |
| `security.audit.events` | list of strings | `[]` | Event types to log (empty = all enabled by default). |
| `security.audit.include_body` | bool | `false` | Include request bodies in audit log entries. MAY increase log volume significantly. |
| `security.audit.buffer_size` | int | `10000` | Async event buffer size. `Log()` enqueues events for a background goroutine; events are dropped when the buffer is full. |

#### Audit Outputs

Events can be delivered to multiple outputs simultaneously. Each output has its own `enabled` flag and `format_type` (`json` or `cef`).

**Stdout Output:**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.audit.outputs.stdout.enabled` | bool | `true` | Enable stdout audit output. |
| `security.audit.outputs.stdout.format_type` | string | `json` | Format: `json` or `cef`. |

**File Output (with rotation):**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.audit.outputs.file.enabled` | bool | `false` | Enable file audit output. |
| `security.audit.outputs.file.path` | string | `""` | Path to audit log file. REQUIRED when enabled. |
| `security.audit.outputs.file.format_type` | string | `json` | Format: `json` or `cef`. |
| `security.audit.outputs.file.max_size_mb` | int | `100` | Max file size before rotation (MB). |
| `security.audit.outputs.file.max_backups` | int | `5` | Max number of rotated backup files. |
| `security.audit.outputs.file.max_age_days` | int | `30` | Max days to retain rotated files. |
| `security.audit.outputs.file.compress` | bool | `true` | Compress rotated files with gzip. |

**Syslog Output (RFC 5424):**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.audit.outputs.syslog.enabled` | bool | `false` | Enable syslog audit output. |
| `security.audit.outputs.syslog.network` | string | `tcp` | Network protocol: `tcp`, `udp`, or `tcp+tls`. |
| `security.audit.outputs.syslog.address` | string | `""` | Syslog server address (`host:port`). REQUIRED when enabled. |
| `security.audit.outputs.syslog.app_name` | string | `schema-registry` | RFC 5424 APP-NAME field. |
| `security.audit.outputs.syslog.facility` | string | `local0` | Syslog facility. Values: `kern`, `user`, `mail`, `daemon`, `auth`, `syslog`, `lpr`, `news`, `uucp`, `cron`, `local0`–`local7`. |
| `security.audit.outputs.syslog.format_type` | string | `json` | Format: `json` or `cef`. |
| `security.audit.outputs.syslog.tls_cert` | string | `""` | Path to TLS client certificate (for mTLS). |
| `security.audit.outputs.syslog.tls_key` | string | `""` | Path to TLS client key (for mTLS). |
| `security.audit.outputs.syslog.tls_ca` | string | `""` | Path to CA certificate for server verification. |

**Webhook Output:**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.audit.outputs.webhook.enabled` | bool | `false` | Enable webhook audit output. |
| `security.audit.outputs.webhook.url` | string | `""` | Webhook endpoint URL. REQUIRED when enabled. |
| `security.audit.outputs.webhook.format_type` | string | `json` | Format: `json` or `cef`. |
| `security.audit.outputs.webhook.batch_size` | int | `100` | Max events per batch. |
| `security.audit.outputs.webhook.flush_interval` | string | `5s` | Max time between flushes (Go duration). |
| `security.audit.outputs.webhook.timeout` | string | `10s` | HTTP request timeout (Go duration). |
| `security.audit.outputs.webhook.max_retries` | int | `3` | Max retry attempts on 5xx errors. |
| `security.audit.outputs.webhook.buffer_size` | int | `10000` | Max events buffered in memory. Events are dropped when full. |
| `security.audit.outputs.webhook.headers` | map | `{}` | Custom HTTP headers (e.g., auth tokens). |

#### Audit Environment Variable Overrides

All audit settings support env var overrides with `SCHEMA_REGISTRY_AUDIT_` prefix:

```
SCHEMA_REGISTRY_AUDIT_ENABLED
SCHEMA_REGISTRY_AUDIT_INCLUDE_BODY
SCHEMA_REGISTRY_AUDIT_BUFFER_SIZE
SCHEMA_REGISTRY_AUDIT_STDOUT_ENABLED / _FORMAT
SCHEMA_REGISTRY_AUDIT_FILE_ENABLED / _PATH / _FORMAT / _MAX_SIZE_MB / _MAX_BACKUPS / _MAX_AGE_DAYS / _COMPRESS
SCHEMA_REGISTRY_AUDIT_SYSLOG_ENABLED / _NETWORK / _ADDRESS / _APP_NAME / _FACILITY / _FORMAT / _TLS_CERT / _TLS_KEY / _TLS_CA
SCHEMA_REGISTRY_AUDIT_WEBHOOK_ENABLED / _URL / _FORMAT / _BATCH_SIZE / _FLUSH_INTERVAL / _TIMEOUT / _MAX_RETRIES / _BUFFER_SIZE
```

#### Example: Multi-Output Configuration

```yaml
security:
  audit:
    enabled: true
    include_body: false
    outputs:
      stdout:
        enabled: true
        format_type: json
      file:
        enabled: true
        path: /var/log/schema-registry/audit.log
        format_type: json
        max_size_mb: 100
        max_backups: 5
        max_age_days: 30
        compress: true
      syslog:
        enabled: true
        network: tcp+tls
        address: "syslog.example.com:6514"
        app_name: schema-registry
        facility: local0
        format_type: json
        tls_ca: /etc/ssl/certs/syslog-ca.pem
      webhook:
        enabled: true
        url: "https://splunk-hec.example.com:8088/services/collector/event"
        format_type: json
        batch_size: 100
        flush_interval: "5s"
        timeout: "10s"
        max_retries: 3
        buffer_size: 10000
        headers:
          Authorization: "Splunk YOUR-HEC-TOKEN"
```

### Per-Principal Metrics

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `security.metrics.per_principal_metrics` | *bool | `true` | Enable per-principal (user identity) Prometheus metrics. Adds a `principal` label to request count, error, and endpoint metrics. MAY increase cardinality in deployments with many distinct users. |

```yaml
security:
  metrics:
    per_principal_metrics: true
```

---

## MCP Server

The MCP (Model Context Protocol) server enables AI assistants to interact with the schema registry. It runs as a separate HTTP endpoint alongside the REST API. For full documentation, see the [MCP Guide](mcp.md).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mcp.enabled` | bool | `false` | Enable the MCP server |
| `mcp.host` | string | `127.0.0.1` | Bind address |
| `mcp.port` | int | `9081` | Port (separate from REST API) |
| `mcp.auth_token` | string | `""` | Bearer token for authentication (empty = no auth) |
| `mcp.read_only` | bool | `false` | Restrict to read-only tools |
| `mcp.tool_policy` | string | `allow_all` | Tool access: `allow_all`, `deny_list`, `allow_list` |
| `mcp.allowed_tools` | []string | `[]` | Tools to expose (for `allow_list` policy) |
| `mcp.denied_tools` | []string | `[]` | Tools to hide (for `deny_list` policy) |
| `mcp.allowed_origins` | []string | `["http://localhost:*", ...]` | Origin header allowlist |
| `mcp.require_confirmations` | bool | `false` | Two-phase confirmations for destructive operations |
| `mcp.confirmation_ttl` | int | `300` | Confirmation token TTL in seconds |
| `mcp.log_schemas` | bool | `false` | Log full schema bodies in debug output |
| `mcp.permission_preset` | string | `""` | Named preset: `readonly`, `developer`, `operator`, `admin`, `full` |
| `mcp.permission_scopes` | []string | `[]` | Individual permission scopes (when no preset is set) |

```yaml
mcp:
  enabled: true
  host: 127.0.0.1
  port: 9081
  auth_token: "my-secret-token"
  read_only: false
  permission_preset: developer
  allowed_origins:
    - "http://localhost:*"
    - "https://localhost:*"
    - "vscode-webview://*"
  require_confirmations: false
  log_schemas: false
```

### MCP Environment Variables

| Field | Environment Variable |
|-------|---------------------|
| `enabled` | `SCHEMA_REGISTRY_MCP_ENABLED` |
| `host` | `SCHEMA_REGISTRY_MCP_HOST` |
| `port` | `SCHEMA_REGISTRY_MCP_PORT` |
| `auth_token` | `SCHEMA_REGISTRY_MCP_AUTH_TOKEN` |
| `read_only` | `SCHEMA_REGISTRY_MCP_READ_ONLY` |
| `allowed_origins` | `SCHEMA_REGISTRY_MCP_ALLOWED_ORIGINS` (comma-separated) |
| `require_confirmations` | `SCHEMA_REGISTRY_MCP_REQUIRE_CONFIRMATIONS` |
| `confirmation_ttl` | `SCHEMA_REGISTRY_MCP_CONFIRMATION_TTL` |
| `log_schemas` | `SCHEMA_REGISTRY_MCP_LOG_SCHEMAS` |
| `permission_preset` | `SCHEMA_REGISTRY_MCP_PERMISSION_PRESET` |
| `permission_scopes` | `SCHEMA_REGISTRY_MCP_PERMISSION_SCOPES` (comma-separated) |
| `tool_policy` | `SCHEMA_REGISTRY_MCP_TOOL_POLICY` |
| `allowed_tools` | `SCHEMA_REGISTRY_MCP_ALLOWED_TOOLS` (comma-separated) |
| `denied_tools` | `SCHEMA_REGISTRY_MCP_DENIED_TOOLS` (comma-separated) |

---

## Environment Variables

The following environment variables override the corresponding configuration file values. They are applied after the configuration file is loaded.

### Server

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_HOST` | `server.host` | string |
| `SCHEMA_REGISTRY_PORT` | `server.port` | int |
| `SCHEMA_REGISTRY_DOCS_ENABLED` | `server.docs_enabled` | bool (`true`/`1`) |
| `SCHEMA_REGISTRY_SHUTDOWN_TIMEOUT` | `server.shutdown_timeout` | int |

### Storage

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_STORAGE_TYPE` | `storage.type` | string |
| `SCHEMA_REGISTRY_AUTH_TYPE` | `storage.auth_type` | string |

### PostgreSQL

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_PG_HOST` | `storage.postgresql.host` | string |
| `SCHEMA_REGISTRY_PG_PORT` | `storage.postgresql.port` | int |
| `SCHEMA_REGISTRY_PG_DATABASE` | `storage.postgresql.database` | string |
| `SCHEMA_REGISTRY_PG_USER` | `storage.postgresql.user` | string |
| `SCHEMA_REGISTRY_PG_PASSWORD` | `storage.postgresql.password` | string |
| `SCHEMA_REGISTRY_PG_SSLMODE` | `storage.postgresql.ssl_mode` | string |

### MySQL

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_MYSQL_HOST` | `storage.mysql.host` | string |
| `SCHEMA_REGISTRY_MYSQL_PORT` | `storage.mysql.port` | int |
| `SCHEMA_REGISTRY_MYSQL_DATABASE` | `storage.mysql.database` | string |
| `SCHEMA_REGISTRY_MYSQL_USER` | `storage.mysql.user` | string |
| `SCHEMA_REGISTRY_MYSQL_PASSWORD` | `storage.mysql.password` | string |
| `SCHEMA_REGISTRY_MYSQL_TLS` | `storage.mysql.tls` | string |

### Cassandra

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_CASSANDRA_HOSTS` | `storage.cassandra.hosts` | comma-separated string |
| `SCHEMA_REGISTRY_CASSANDRA_PORT` | `storage.cassandra.port` | int |
| `SCHEMA_REGISTRY_CASSANDRA_KEYSPACE` | `storage.cassandra.keyspace` | string |
| `SCHEMA_REGISTRY_CASSANDRA_LOCAL_DC` | `storage.cassandra.local_dc` | string |
| `SCHEMA_REGISTRY_CASSANDRA_CONSISTENCY` | `storage.cassandra.consistency` | string |
| `SCHEMA_REGISTRY_CASSANDRA_READ_CONSISTENCY` | `storage.cassandra.read_consistency` | string |
| `SCHEMA_REGISTRY_CASSANDRA_WRITE_CONSISTENCY` | `storage.cassandra.write_consistency` | string |
| `SCHEMA_REGISTRY_CASSANDRA_SERIAL_CONSISTENCY` | `storage.cassandra.serial_consistency` | string |
| `SCHEMA_REGISTRY_CASSANDRA_USERNAME` | `storage.cassandra.username` | string |
| `SCHEMA_REGISTRY_CASSANDRA_PASSWORD` | `storage.cassandra.password` | string |
| `SCHEMA_REGISTRY_CASSANDRA_TIMEOUT` | `storage.cassandra.timeout` | duration string |
| `SCHEMA_REGISTRY_CASSANDRA_CONNECT_TIMEOUT` | `storage.cassandra.connect_timeout` | duration string |
| `SCHEMA_REGISTRY_CASSANDRA_MAX_RETRIES` | `storage.cassandra.max_retries` | int |
| `SCHEMA_REGISTRY_CASSANDRA_ID_BLOCK_SIZE` | `storage.cassandra.id_block_size` | int |

### Compatibility and Logging

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_COMPATIBILITY_LEVEL` | `compatibility.default_level` | string |
| `SCHEMA_REGISTRY_LOG_LEVEL` | `logging.level` | string |

### Bootstrap

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_BOOTSTRAP_ENABLED` | `security.auth.bootstrap.enabled` | bool (`true`/`1`) |
| `SCHEMA_REGISTRY_BOOTSTRAP_USERNAME` | `security.auth.bootstrap.username` | string |
| `SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD` | `security.auth.bootstrap.password` | string |
| `SCHEMA_REGISTRY_BOOTSTRAP_EMAIL` | `security.auth.bootstrap.email` | string |

### HashiCorp Vault

| Variable | Overrides | Type | Notes |
|----------|-----------|------|-------|
| `SCHEMA_REGISTRY_VAULT_ADDRESS` | `storage.vault.address` | string | |
| `SCHEMA_REGISTRY_VAULT_TOKEN` | `storage.vault.token` | string | |
| `VAULT_TOKEN` | `storage.vault.token` | string | Standard Vault variable. Used only if `SCHEMA_REGISTRY_VAULT_TOKEN` is not set. |
| `SCHEMA_REGISTRY_VAULT_NAMESPACE` | `storage.vault.namespace` | string | |
| `VAULT_NAMESPACE` | `storage.vault.namespace` | string | Standard Vault variable. Used only if `SCHEMA_REGISTRY_VAULT_NAMESPACE` is not set. |
| `SCHEMA_REGISTRY_VAULT_MOUNT_PATH` | `storage.vault.mount_path` | string | |
| `SCHEMA_REGISTRY_VAULT_BASE_PATH` | `storage.vault.base_path` | string | |
| `SCHEMA_REGISTRY_VAULT_TLS_CERT_FILE` | `storage.vault.tls_cert_file` | string | Client certificate for mTLS |
| `SCHEMA_REGISTRY_VAULT_TLS_KEY_FILE` | `storage.vault.tls_key_file` | string | Client key for mTLS |
| `SCHEMA_REGISTRY_VAULT_TLS_CA_FILE` | `storage.vault.tls_ca_file` | string | CA certificate for server verification |
| `SCHEMA_REGISTRY_VAULT_TLS_SKIP_VERIFY` | `storage.vault.tls_skip_verify` | bool (`true`/`1`) | Skip TLS verification |

### JWT

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_JWT_DEFAULT_ROLE` | `security.auth.jwt.default_role` | string |
| `SCHEMA_REGISTRY_JWT_JWKS_CACHE_TTL` | `security.auth.jwt.jwks_cache_ttl` | int |
| `SCHEMA_REGISTRY_JWT_HTTP_TIMEOUT` | `security.auth.jwt.http_timeout` | int |

### Authentication

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_AUTH_ENABLED` | `security.auth.enabled` | bool (`true`/`1`) |
| `SCHEMA_REGISTRY_AUTH_METHODS` | `security.auth.methods` | comma-separated string |

### LDAP

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_LDAP_ENABLED` | `security.auth.ldap.enabled` | bool (`true`/`1`) |
| `SCHEMA_REGISTRY_LDAP_URL` | `security.auth.ldap.url` | string |
| `SCHEMA_REGISTRY_LDAP_BIND_DN` | `security.auth.ldap.bind_dn` | string |
| `SCHEMA_REGISTRY_LDAP_BIND_PASSWORD` | `security.auth.ldap.bind_password` | string |
| `SCHEMA_REGISTRY_LDAP_BASE_DN` | `security.auth.ldap.base_dn` | string |
| `SCHEMA_REGISTRY_LDAP_USER_SEARCH_FILTER` | `security.auth.ldap.user_search_filter` | string |
| `SCHEMA_REGISTRY_LDAP_USER_SEARCH_BASE` | `security.auth.ldap.user_search_base` | string |
| `SCHEMA_REGISTRY_LDAP_GROUP_SEARCH_FILTER` | `security.auth.ldap.group_search_filter` | string |
| `SCHEMA_REGISTRY_LDAP_GROUP_SEARCH_BASE` | `security.auth.ldap.group_search_base` | string |
| `SCHEMA_REGISTRY_LDAP_USERNAME_ATTRIBUTE` | `security.auth.ldap.username_attribute` | string |
| `SCHEMA_REGISTRY_LDAP_EMAIL_ATTRIBUTE` | `security.auth.ldap.email_attribute` | string |
| `SCHEMA_REGISTRY_LDAP_GROUP_ATTRIBUTE` | `security.auth.ldap.group_attribute` | string |
| `SCHEMA_REGISTRY_LDAP_DEFAULT_ROLE` | `security.auth.ldap.default_role` | string |
| `SCHEMA_REGISTRY_LDAP_START_TLS` | `security.auth.ldap.start_tls` | bool (`true`/`1`) |
| `SCHEMA_REGISTRY_LDAP_INSECURE_SKIP_VERIFY` | `security.auth.ldap.insecure_skip_verify` | bool (`true`/`1`) |
| `SCHEMA_REGISTRY_LDAP_CA_CERT_FILE` | `security.auth.ldap.ca_cert_file` | string |
| `SCHEMA_REGISTRY_LDAP_CLIENT_CERT_FILE` | `security.auth.ldap.client_cert_file` | string |
| `SCHEMA_REGISTRY_LDAP_CLIENT_KEY_FILE` | `security.auth.ldap.client_key_file` | string |
| `SCHEMA_REGISTRY_LDAP_ALLOW_FALLBACK` | `security.auth.ldap.allow_fallback` | bool (`true`/`1`) |
| `SCHEMA_REGISTRY_LDAP_CONNECTION_TIMEOUT` | `security.auth.ldap.connection_timeout` | int |
| `SCHEMA_REGISTRY_LDAP_REQUEST_TIMEOUT` | `security.auth.ldap.request_timeout` | int |

### OIDC (OpenID Connect)

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_OIDC_ENABLED` | `security.auth.oidc.enabled` | bool (`true`/`1`) |
| `SCHEMA_REGISTRY_OIDC_ISSUER_URL` | `security.auth.oidc.issuer_url` | string |
| `SCHEMA_REGISTRY_OIDC_CLIENT_ID` | `security.auth.oidc.client_id` | string |
| `SCHEMA_REGISTRY_OIDC_CLIENT_SECRET` | `security.auth.oidc.client_secret` | string |
| `SCHEMA_REGISTRY_OIDC_USERNAME_CLAIM` | `security.auth.oidc.username_claim` | string |
| `SCHEMA_REGISTRY_OIDC_ROLES_CLAIM` | `security.auth.oidc.roles_claim` | string |
| `SCHEMA_REGISTRY_OIDC_DEFAULT_ROLE` | `security.auth.oidc.default_role` | string |
| `SCHEMA_REGISTRY_OIDC_REQUIRED_AUDIENCE` | `security.auth.oidc.required_audience` | string |
| `SCHEMA_REGISTRY_OIDC_SKIP_ISSUER_CHECK` | `security.auth.oidc.skip_issuer_check` | bool (`true`/`1`) |
| `SCHEMA_REGISTRY_OIDC_SKIP_EXPIRY_CHECK` | `security.auth.oidc.skip_expiry_check` | bool (`true`/`1`) |

> **Note:** `role_mapping` (map type) and `allowed_algorithms` (slice type) cannot be set via environment variables. These MUST be configured in the YAML config file.

---

## Complete Configuration Example

The following example shows every configuration section with annotated comments. Copy and adapt as needed.

```yaml
# =============================================================================
# AxonOps Schema Registry -- Complete Configuration Reference
# =============================================================================

# --- HTTP Server -----------------------------------------------------------
server:
  host: "0.0.0.0"                    # Bind address
  port: 8081                          # Listen port
  read_timeout: 30                    # Read timeout (seconds)
  write_timeout: 30                   # Write timeout (seconds)
  shutdown_timeout: 30                # Graceful shutdown wait (seconds)
  docs_enabled: false                 # Swagger UI at /docs, OpenAPI at /openapi.yaml

# --- Storage Backend -------------------------------------------------------
storage:
  type: postgresql                    # memory | postgresql | mysql | cassandra
  auth_type: ""                       # Separate auth store: vault | (same as type if empty)

  postgresql:
    host: localhost
    port: 5432
    database: schema_registry
    user: registry
    password: ${PG_PASSWORD}
    ssl_mode: disable                 # disable | require | verify-ca | verify-full
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300            # seconds

  mysql:
    host: localhost
    port: 3306
    database: schema_registry
    user: registry
    password: ${MYSQL_PASSWORD}
    tls: "false"                      # true | false | skip-verify | preferred
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300            # seconds

  cassandra:
    hosts:
      - localhost
    port: 9042
    keyspace: schema_registry
    local_dc: ""                        # Set for multi-DC (e.g., dc1)
    consistency: LOCAL_QUORUM           # Default for all operations
    read_consistency: ""                # Override for reads (e.g., LOCAL_ONE)
    write_consistency: ""               # Override for writes (e.g., LOCAL_QUORUM)
    serial_consistency: LOCAL_SERIAL    # For LWT: LOCAL_SERIAL or SERIAL
    username: ""
    password: ""
    timeout: 10s                        # Query timeout
    connect_timeout: 10s                # Connection timeout
    max_retries: 50                     # CAS operation retry limit
    id_block_size: 50                   # IDs per LWT allocation

  vault:
    address: ""                       # e.g., https://vault.internal:8200
    token: ${VAULT_TOKEN}
    namespace: ""                     # Vault Enterprise namespace
    mount_path: secret                # KV v2 mount path
    base_path: schema-registry        # Base path for registry data
    tls_cert_file: ""
    tls_key_file: ""
    tls_ca_file: ""
    tls_skip_verify: false

# --- Compatibility ---------------------------------------------------------
compatibility:
  default_level: BACKWARD             # NONE | BACKWARD | BACKWARD_TRANSITIVE
                                      # FORWARD | FORWARD_TRANSITIVE
                                      # FULL | FULL_TRANSITIVE

# --- Logging ---------------------------------------------------------------
logging:
  level: info                         # debug | info | warn | error
  format: json                        # json | text

# --- Security --------------------------------------------------------------
security:

  # TLS termination
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
    ca_file: ""                       # For mTLS client verification
    min_version: "TLS1.2"            # TLS1.2 | TLS1.3
    client_auth: none                 # none | request | require | verify
    auto_reload: false                # SIGHUP reloads certs without restart

  # Authentication
  auth:
    enabled: false
    methods:                          # Tried in order
      - api_key
      - basic

    # First-run admin provisioning
    bootstrap:
      enabled: false
      username: admin
      password: ${SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD}
      email: ""

    # HTTP Basic Auth
    basic:
      realm: "Schema Registry"
      users: {}                       # username: bcrypt-hash (legacy; prefer DB users)
      htpasswd_file: ""               # Apache htpasswd file (bcrypt only)

    # API Key Auth
    api_key:
      header: "X-API-Key"
      query_param: "api_key"
      storage_type: database          # "database" (default) or "memory" (config-defined)
      secret: ${API_KEY_SECRET}       # HMAC pepper (>=32 bytes recommended)
      key_prefix: "sr_"
      cache_refresh_seconds: 60       # 0 to disable caching
      # keys:                         # Used when storage_type is "memory"
      #   - name: ci-pipeline
      #     key_hash: "$2a$10$..."    # bcrypt hash of the API key
      #     role: developer

    # JWT Auth (external IdP)
    jwt:
      issuer: ""
      audience: ""
      jwks_url: ""
      public_key_file: ""
      algorithm: RS256                # RS256 | ES256
      claims_mapping:
        username: sub
        role: role
      default_role: readonly            # Fallback when no JWT claim matches
      jwks_cache_ttl: 300               # JWKS cache TTL (seconds)
      http_timeout: 10                  # JWKS HTTP client timeout (seconds)

    # LDAP Auth
    ldap:
      enabled: false
      url: ""                         # ldap://host:389 | ldaps://host:636
      bind_dn: ""
      bind_password: ""
      base_dn: ""
      user_search_filter: ""          # e.g., (sAMAccountName=%s)
      user_search_base: ""
      group_search_filter: ""         # e.g., (member=%s) — %s is user DN
      group_search_base: ""           # e.g., OU=Groups,DC=example,DC=com
      username_attribute: ""          # sAMAccountName | uid | userPrincipalName
      email_attribute: ""             # mail
      group_attribute: ""             # memberOf
      role_mapping: {}                # LDAP group DN -> registry role
      default_role: ""
      start_tls: false
      insecure_skip_verify: false
      ca_cert_file: ""
      client_cert_file: ""            # Client cert for mTLS to LDAP server
      client_key_file: ""             # Client key for mTLS to LDAP server
      allow_fallback: true            # false = strict LDAP-only; true = fallback for user-not-found only
      connection_timeout: 10          # seconds
      request_timeout: 30             # seconds

    # OIDC Auth
    oidc:
      enabled: false
      issuer_url: ""
      client_id: ""
      client_secret: ""
      username_claim: ""              # sub | preferred_username | email
      roles_claim: ""                 # roles | groups
      role_mapping: {}                # OIDC role -> registry role
      default_role: ""
      required_audience: ""
      allowed_algorithms: []          # e.g., [RS256, ES256]
      skip_issuer_check: false        # Testing only
      skip_expiry_check: false        # Testing only

    # RBAC
    rbac:
      enabled: false
      default_role: readonly
      super_admins: []                # Usernames with full access

  # Rate limiting
  rate_limiting:
    enabled: false
    requests_per_second: 100
    burst_size: 200
    per_client: true                  # Per client IP vs. global
    per_endpoint: false               # Per endpoint vs. global

  # Audit logging
  audit:
    enabled: false
    log_file: ""
    events:                           # schema_register | schema_delete | config_update | mcp_tool_call | ...
      - schema_register
      - schema_delete
      - config_update
    include_body: false

  # Per-principal Prometheus metrics
  metrics:
    per_principal_metrics: true        # Adds "principal" label to metrics

# --- MCP Server (AI Assistant Access) -------------------------------------
mcp:
  enabled: false                      # Enable the MCP server
  host: 127.0.0.1                    # Bind address (localhost for security)
  port: 9081                          # MCP port (separate from REST)
  auth_token: ""                      # Bearer token (empty = no auth)
  read_only: false                    # Restrict to read-only tools
  permission_preset: ""               # readonly | developer | operator | admin | full
  permission_scopes: []               # Individual scopes (when no preset is set)
  tool_policy: allow_all              # allow_all | deny_list | allow_list
  allowed_tools: []                   # For allow_list policy
  denied_tools: []                    # For deny_list policy
  allowed_origins:                    # Origin header allowlist
    - "http://localhost:*"
    - "https://localhost:*"
    - "vscode-webview://*"
  require_confirmations: false        # Two-phase confirmations for destructive ops
  confirmation_ttl: 300               # Confirmation token TTL (seconds)
  log_schemas: false                  # Log full schema bodies (debug only)
```

---

## API-Managed Configuration

Some enterprise features are configured via the REST API rather than YAML:

- **Encryption (DEK Registry)** -- KEKs and DEKs are managed via the `/dek-registry/v1/` API endpoints. KMS connection properties are set per-KEK using the `kmsProps` field. See [Encryption](encryption.md).
- **Exporters (Schema Linking)** -- Exporters are configured via the `/exporters` API endpoints. See [Exporters](exporters.md).
- **Data Contract Defaults** -- Default and override metadata/ruleSet policies are configured via the `PUT /config` and `PUT /config/{subject}` endpoints. See [Data Contracts](data-contracts.md).

---

## Related Documentation

- [Storage Backends](storage-backends.md) -- detailed setup for each database backend
- [Authentication](authentication.md) -- authentication methods and user management
- [Security](security.md) -- TLS, RBAC, and security hardening
- [Deployment](deployment.md) -- production deployment patterns and Docker usage
- [Encryption](encryption.md) -- DEK Registry, CSFLE, and KMS providers
- [Exporters](exporters.md) -- Schema Linking via exporter management API
- [Data Contracts](data-contracts.md) -- metadata, rule sets, and governance policies
