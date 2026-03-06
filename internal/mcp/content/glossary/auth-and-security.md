# Auth and Security

## Authentication Methods

The schema registry supports 6 authentication methods, configured under `security.auth.methods`:

| Method | Description |
|--------|-------------|
| **basic** | Username/password via HTTP Basic auth. Users defined in config or htpasswd file. |
| **api_key** | API key via `X-API-Key` header or query parameter. Keys are HMAC-SHA256 hashed in storage. |
| **jwt** | JSON Web Token via `Authorization: Bearer` header. Validated against JWKS URL or public key. |
| **oidc** | OpenID Connect. Delegates to an external identity provider (Keycloak, Auth0, etc.). |
| **ldap** | LDAP/Active Directory. Binds to LDAP server for authentication, maps groups to roles. |
| **mtls** | Mutual TLS. Client certificates validated against a CA. |

Multiple methods can be enabled simultaneously. The first successful authentication wins.

## RBAC Roles

The registry uses a deny-by-default RBAC model with 4 built-in roles:

### super_admin
Full access to everything. Assigned via `security.auth.rbac.super_admins` list.
**Permissions:** schema:read, schema:write, schema:delete, config:read, config:write, mode:read, mode:write, import:write, admin:read, admin:write, encryption:read, encryption:write, exporter:read, exporter:write

### admin
Can manage schemas, configuration, encryption, and exporters. Cannot manage users/API keys (only admin:read).
**Permissions:** schema:read, schema:write, schema:delete, config:read, config:write, mode:read, mode:write, import:write, admin:read, encryption:read, encryption:write, exporter:read, exporter:write

### developer
Can register and read schemas, manage subject config.
**Permissions:** schema:read, schema:write, config:read, mode:read, encryption:read, exporter:read

### readonly
Can only read schemas, config, mode, encryption, and exporter data.
**Permissions:** schema:read, config:read, mode:read, encryption:read, exporter:read

## Deny-By-Default Model

If RBAC is enabled and no endpoint permission entry matches a request, access is **denied**. This prevents newly added routes from being accidentally unprotected.

## API Key Lifecycle

1. **Creation:** `create_apikey` generates a random key, HMAC-SHA256 hashes it, stores the hash. The raw key is returned ONCE.
2. **Usage:** Client sends key via `X-API-Key` header. Registry hashes it and compares to stored hash.
3. **Rotation:** `rotate_apikey` creates a new key and revokes the old one atomically.
4. **Revocation:** `revoke_apikey` or `delete_apikey` disables or removes the key.
5. **Expiry:** Keys have an `expires_at` timestamp. Expired keys return error 40103.
6. **Disable:** `update_apikey` with `enabled: false` disables without deleting.

## Rate Limiting

Token bucket rate limiter configured under `security.rate_limiting`:

| Field | Description |
|-------|-------------|
| `enabled` | Enable rate limiting |
| `requests_per_second` | Token refill rate |
| `burst_size` | Maximum burst size |
| `per_client` | Rate limit per client IP |
| `per_endpoint` | Rate limit per endpoint path |

When rate limited, the server returns HTTP 429 with `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` headers.

## Audit Logging

Event-based audit logging configured under `security.audit`:

| Field | Description |
|-------|-------------|
| `enabled` | Enable audit logging |
| `log_file` | Path to audit log file |
| `events` | Event types to log (e.g., `schema_register`, `schema_delete`, `config_update`) |
| `include_body` | Include request body in audit log |

**MCP audit events:** `mcp_tool_call` (successful tool invocation), `mcp_tool_error` (failed tool invocation). Includes tool name, status, duration, subject, and optional schema body.

## MCP Permission Scopes

The MCP server has its own permission scope system (independent from REST RBAC) with 14 scopes mirroring the REST taxonomy:

`schema_read`, `schema_write`, `schema_delete`, `config_read`, `config_write`, `mode_read`, `mode_write`, `import`, `encryption_read`, `encryption_write`, `exporter_read`, `exporter_write`, `admin_read`, `admin_write`

See `schema://glossary/mcp-configuration` for preset definitions and configuration details.

## MCP Tools

- **list_users** / **get_user** / **get_user_by_username** -- user management
- **create_user** / **update_user** / **delete_user** -- user lifecycle
- **list_apikeys** / **create_apikey** / **update_apikey** / **delete_apikey** -- API key management
- **rotate_apikey** / **revoke_apikey** -- API key rotation and revocation
- **list_roles** -- list available roles and their permissions
- **change_password** -- change user password
