# Configuration Reference

This document provides a complete reference for all configuration options available in AxonOps Schema Registry.

## Table of Contents

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

```yaml
server:
  host: "0.0.0.0"
  port: 8081
  read_timeout: 30
  write_timeout: 30
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
| `storage.cassandra.keyspace` | string | `"schema_registry"` | Keyspace name. Created automatically with migrations. |
| `storage.cassandra.consistency` | string | `"LOCAL_QUORUM"` | Default consistency level for all operations. Used when `read_consistency` or `write_consistency` is not set. |
| `storage.cassandra.read_consistency` | string | `""` (falls back to `consistency`) | Consistency level for read operations. Useful in multi-datacenter deployments where read latency matters (e.g., `LOCAL_ONE`). |
| `storage.cassandra.write_consistency` | string | `""` (falls back to `consistency`) | Consistency level for write operations. Set independently for durability requirements (e.g., `LOCAL_QUORUM`). |
| `storage.cassandra.username` | string | `""` | Authentication username. |
| `storage.cassandra.password` | string | `""` | Authentication password. |

Schema migrations run automatically on startup.

```yaml
storage:
  type: cassandra
  cassandra:
    hosts:
      - node1.cassandra.local
      - node2.cassandra.local
      - node3.cassandra.local
    keyspace: schema_registry
    consistency: LOCAL_QUORUM
    read_consistency: LOCAL_ONE
    write_consistency: LOCAL_QUORUM
    username: registry
    password: ${CASSANDRA_PASSWORD}
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
| `security.tls.auto_reload` | bool | `false` | Automatically reload certificates when they change on disk, without requiring a restart. |

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

Creates an initial admin user on startup when the users table is empty. Designed for first-run provisioning. The password should always be set via the `SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD` environment variable rather than in the configuration file.

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
| `security.auth.basic.htpasswd_file` | string | `""` | Path to an Apache-style htpasswd file. |

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
| `security.auth.api_key.storage_type` | string | `""` | Where API keys are stored. Values: `memory`, `database`. |
| `security.auth.api_key.secret` | string | `""` | HMAC-SHA256 pepper for hashing API keys before storage. Provides defense-in-depth: even if the database is compromised, keys cannot be verified without this secret. Use at least 32 bytes of random data. If empty, falls back to plain SHA-256 hashing. |
| `security.auth.api_key.key_prefix` | string | `"sr_"` | Prefix prepended to generated API keys for identification (e.g., `sr_live_abc123`). |
| `security.auth.api_key.cache_refresh_seconds` | int | `60` | How often (seconds) the in-memory API key cache is refreshed from the database. Ensures cluster-wide consistency. Set to `0` to disable caching. |

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
| `security.auth.ldap.group_search_filter` | string | `""` | LDAP filter for finding group memberships (e.g., `(member=%s)`). `%s` is replaced with the user DN. |
| `security.auth.ldap.group_search_base` | string | `""` | Base DN for group searches (e.g., `OU=Groups,DC=example,DC=com`). |
| `security.auth.ldap.username_attribute` | string | `""` | LDAP attribute containing the username (`sAMAccountName`, `uid`, `userPrincipalName`). |
| `security.auth.ldap.email_attribute` | string | `""` | LDAP attribute containing the email address (`mail`). |
| `security.auth.ldap.group_attribute` | string | `""` | LDAP attribute containing group membership (`memberOf`). |
| `security.auth.ldap.role_mapping` | map (string to string) | `{}` | Maps LDAP group names to registry roles. |
| `security.auth.ldap.default_role` | string | `""` | Role assigned when no LDAP group matches a mapping. |
| `security.auth.ldap.start_tls` | bool | `false` | Upgrade an unencrypted connection using STARTTLS. |
| `security.auth.ldap.insecure_skip_verify` | bool | `false` | Skip TLS certificate verification. Not recommended for production. |
| `security.auth.ldap.ca_cert_file` | string | `""` | Path to CA certificate for verifying the LDAP server. |
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
| `security.auth.oidc.redirect_url` | string | `""` | OAuth2 callback URL. |
| `security.auth.oidc.scopes` | list of strings | `[]` | OAuth2 scopes to request (e.g., `openid`, `profile`, `email`). |
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
| `security.audit.log_file` | string | `""` | Path to the audit log file. |
| `security.audit.events` | list of strings | `[]` | Event types to log. Values: `schema_register`, `schema_delete`, `config_change`. |
| `security.audit.include_body` | bool | `false` | Include request bodies in audit log entries. May increase log volume significantly. |

```yaml
security:
  audit:
    enabled: true
    log_file: /var/log/schema-registry/audit.log
    events:
      - schema_register
      - schema_delete
      - config_change
    include_body: false
```

---

## Environment Variables

The following environment variables override the corresponding configuration file values. They are applied after the configuration file is loaded.

### Server

| Variable | Overrides | Type |
|----------|-----------|------|
| `SCHEMA_REGISTRY_HOST` | `server.host` | string |
| `SCHEMA_REGISTRY_PORT` | `server.port` | int |
| `SCHEMA_REGISTRY_DOCS_ENABLED` | `server.docs_enabled` | bool (`true`/`1`) |

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
    keyspace: schema_registry
    consistency: LOCAL_QUORUM         # Default for all operations
    read_consistency: ""              # Override for reads (e.g., LOCAL_ONE)
    write_consistency: ""             # Override for writes (e.g., LOCAL_QUORUM)
    username: ""
    password: ""

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
    auto_reload: false                # Reload certs on file change

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
      htpasswd_file: ""

    # API Key Auth
    api_key:
      header: "X-API-Key"
      query_param: "api_key"
      storage_type: database          # memory | database
      secret: ${API_KEY_SECRET}       # HMAC pepper (>=32 bytes recommended)
      key_prefix: "sr_"
      cache_refresh_seconds: 60       # 0 to disable caching

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

    # LDAP Auth
    ldap:
      enabled: false
      url: ""                         # ldap://host:389 | ldaps://host:636
      bind_dn: ""
      bind_password: ""
      base_dn: ""
      user_search_filter: ""          # e.g., (sAMAccountName=%s)
      user_search_base: ""
      group_search_filter: ""         # e.g., (member=%s)
      group_search_base: ""
      username_attribute: ""          # sAMAccountName | uid | userPrincipalName
      email_attribute: ""             # mail
      group_attribute: ""             # memberOf
      role_mapping: {}                # LDAP group DN -> registry role
      default_role: ""
      start_tls: false
      insecure_skip_verify: false
      ca_cert_file: ""
      connection_timeout: 10          # seconds
      request_timeout: 30             # seconds

    # OIDC Auth
    oidc:
      enabled: false
      issuer_url: ""
      client_id: ""
      client_secret: ""
      redirect_url: ""
      scopes: []                      # e.g., [openid, profile, email]
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
    events:                           # schema_register | schema_delete | config_change
      - schema_register
      - schema_delete
      - config_change
    include_body: false
```

---

## Related Documentation

- [Storage Backends](storage-backends.md) -- detailed setup for each database backend
- [Authentication](authentication.md) -- authentication methods and user management
- [Security](security.md) -- TLS, RBAC, and security hardening
- [Deployment](deployment.md) -- production deployment patterns and Docker usage
