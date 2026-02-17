# Authentication

This guide covers how to configure and use authentication in AxonOps Schema Registry. Authentication controls who can access the registry API. When combined with role-based access control (RBAC), it also determines what each user can do.

## Contents

- [Overview](#overview)
- [Enabling Authentication](#enabling-authentication)
- [Bootstrap Admin User](#bootstrap-admin-user)
  - [Configuration-Based Bootstrap](#configuration-based-bootstrap)
  - [CLI-Based Bootstrap](#cli-based-bootstrap)
- [Basic Authentication](#basic-authentication)
  - [Configuration](#configuration)
  - [Usage](#usage)
  - [Authentication Order for Basic Auth](#authentication-order-for-basic-auth)
- [API Key Authentication](#api-key-authentication)
  - [Three Equivalent Methods](#three-equivalent-methods)
  - [Configuration](#configuration-1)
  - [Key Security](#key-security)
  - [Creating an API Key](#creating-an-api-key)
- [LDAP / Active Directory](#ldap--active-directory)
  - [Configuration](#configuration-2)
  - [How It Works](#how-it-works)
  - [LDAPS and StartTLS](#ldaps-and-starttls)
  - [Configuration Reference](#configuration-reference)
- [OIDC (OpenID Connect)](#oidc-openid-connect)
  - [Configuration](#configuration-3)
  - [Usage](#usage-1)
  - [How It Works](#how-it-works-1)
  - [Configuration Reference](#configuration-reference-1)
- [JWT (JSON Web Token)](#jwt-json-web-token)
  - [Configuration with Static Key](#configuration-with-static-key)
  - [Configuration with JWKS URL](#configuration-with-jwks-url)
  - [Usage](#usage-2)
  - [How It Works](#how-it-works-2)
  - [Configuration Reference](#configuration-reference-2)
- [mTLS (Mutual TLS)](#mtls-mutual-tls)
  - [Configuration](#configuration-4)
  - [Client Auth Modes](#client-auth-modes)
  - [Usage](#usage-3)
- [Roles and Permissions](#roles-and-permissions)
  - [RBAC Configuration](#rbac-configuration)
- [User Management API](#user-management-api)
  - [Create a User](#create-a-user)
  - [List Users](#list-users)
  - [Get a User](#get-a-user)
  - [Update a User](#update-a-user)
  - [Delete a User](#delete-a-user)
  - [Change Your Own Password](#change-your-own-password)
- [API Key Management API](#api-key-management-api)
  - [Create an API Key](#create-an-api-key)
  - [List API Keys](#list-api-keys)
  - [Rotate an API Key](#rotate-an-api-key)
  - [Revoke an API Key](#revoke-an-api-key)
  - [Delete an API Key](#delete-an-api-key)
- [Admin CLI](#admin-cli)
  - [Authentication](#authentication)
  - [User Commands](#user-commands)
  - [API Key Commands](#api-key-commands)
  - [Role Commands](#role-commands)
  - [Output Formats](#output-formats)
  - [Database Bootstrap](#database-bootstrap)
- [Combining Authentication Methods](#combining-authentication-methods)
- [Related Documentation](#related-documentation)

## Overview

Authentication is optional but recommended for production deployments. When enabled, the registry supports multiple authentication methods simultaneously. Requests are evaluated against each configured method in the order they appear in the `methods` list until one succeeds:

1. **API Key** -- header, query parameter, or Confluent-compatible Basic Auth format
2. **Basic Auth** -- database users with bcrypt-hashed passwords, with LDAP fallback if configured
3. **OIDC Bearer Token** -- OpenID Connect token validation
4. **JWT Bearer Token** -- static key or JWKS-based token validation
5. **mTLS Client Certificate** -- client certificate Common Name used as identity

When authentication is disabled (the default), all endpoints are accessible without credentials. The health check (`/`) and metrics (`/metrics`) endpoints are always unauthenticated regardless of configuration.

## Enabling Authentication

Add the `security.auth` section to your configuration file:

```yaml
security:
  auth:
    enabled: true
    methods:
      - api_key
      - basic
    rbac:
      enabled: true
      default_role: readonly
```

The `methods` list defines which authentication methods are active and the order in which they are tried. Valid values are `api_key`, `basic`, `jwt`, `oidc`, and `mtls`.

When a request fails all configured methods, the registry returns `401 Unauthorized` with appropriate `WWW-Authenticate` headers for each enabled method.

## Bootstrap Admin User

When deploying for the first time with an empty user database, you need an initial admin account. The bootstrap mechanism solves this chicken-and-egg problem.

### Configuration-Based Bootstrap

Add the bootstrap section to your configuration:

```yaml
security:
  auth:
    enabled: true
    methods:
      - basic
    bootstrap:
      enabled: true
      username: admin
      password: ${ADMIN_PASSWORD}
      email: admin@example.com
```

Set the password via environment variable:

```bash
export ADMIN_PASSWORD='a-strong-random-password'
```

The bootstrap process is idempotent. If one or more users already exist in the database, the bootstrap step is skipped entirely. The created user is assigned the `super_admin` role.

Bootstrap credentials can also be set entirely through environment variables:

| Variable | Description |
|----------|-------------|
| `SCHEMA_REGISTRY_BOOTSTRAP_ENABLED` | Set to `true` to enable bootstrap |
| `SCHEMA_REGISTRY_BOOTSTRAP_USERNAME` | Admin username |
| `SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD` | Admin password |
| `SCHEMA_REGISTRY_BOOTSTRAP_EMAIL` | Admin email (optional) |

### CLI-Based Bootstrap

The `schema-registry-admin` CLI tool can bootstrap an admin user by connecting directly to the database, bypassing the API entirely:

```bash
schema-registry-admin init \
  --storage-type postgresql \
  --pg-host localhost --pg-port 5432 \
  --pg-database schema_registry \
  --pg-user postgres --pg-password dbpass \
  --admin-username admin \
  --admin-password 'a-strong-random-password' \
  --admin-email admin@example.com
```

This approach is useful when you want to create the admin user before starting the registry for the first time.

## Basic Authentication

Basic Auth validates credentials against the database using bcrypt-hashed passwords. If LDAP is configured, it is tried as a fallback when database authentication fails.

### Configuration

```yaml
security:
  auth:
    enabled: true
    methods:
      - basic
    basic:
      realm: "Schema Registry"
```

The `realm` value appears in the `WWW-Authenticate: Basic realm="..."` response header.

### Usage

```bash
curl -u admin:password http://localhost:8081/subjects
```

```bash
curl -H "Authorization: Basic $(echo -n admin:password | base64)" \
  http://localhost:8081/subjects
```

### Authentication Order for Basic Auth

When a request arrives with Basic Auth credentials, the registry evaluates them in this order:

1. **API key lookup** -- the username is checked as a potential API key value (for Confluent client compatibility)
2. **LDAP** -- if an LDAP provider is configured, credentials are validated against the directory
3. **Database** -- credentials are validated against the local user database (bcrypt comparison)
4. **Config-based users** -- legacy fallback to users defined directly in the YAML config

## API Key Authentication

API keys provide programmatic access without exposing user passwords. They are scoped to a role, have optional expiration, and are stored as SHA-256 hashes (or HMAC-SHA256 when a server secret is configured).

### Three Equivalent Methods

**1. Confluent-compatible Basic Auth format**

This is the recommended method for Kafka producers, consumers, and tools that already support Confluent Schema Registry authentication. The API key is sent as the username; the password field is ignored.

```bash
curl -u "sr_live_abc123def456:x" http://localhost:8081/subjects
```

**2. X-API-Key header**

```bash
curl -H "X-API-Key: sr_live_abc123def456" http://localhost:8081/subjects
```

**3. Query parameter**

```bash
curl "http://localhost:8081/subjects?api_key=sr_live_abc123def456"
```

### Configuration

```yaml
security:
  auth:
    enabled: true
    methods:
      - api_key
    api_key:
      header: X-API-Key
      query_param: api_key
      key_prefix: "sr_live_"
      secret: "${API_KEY_SECRET}"
      cache_refresh_seconds: 60
```

| Field | Description | Default |
|-------|-------------|---------|
| `header` | HTTP header name for API key lookup | `X-API-Key` |
| `query_param` | Query parameter name for API key lookup | `api_key` |
| `key_prefix` | Prefix prepended to generated keys for identification | `""` |
| `secret` | HMAC-SHA256 secret for key hashing (defense-in-depth) | `""` (plain SHA-256) |
| `cache_refresh_seconds` | How often the key cache is refreshed from the database | `60` |

### Key Security

- **Hashed at rest** -- keys are stored as SHA-256 hashes (or HMAC-SHA256 with a server secret)
- **Shown once** -- the raw key is returned only at creation time and is never retrievable afterward
- **Optional HMAC pepper** -- when `secret` is configured, even a database compromise does not allow key verification without the secret
- **Validated on use** -- enabled status and expiration are checked on every request
- **Cached for performance** -- validated keys are cached in memory and refreshed from the database at the configured interval, ensuring cluster-wide consistency

### Creating an API Key

```bash
curl -u admin:password -X POST http://localhost:8081/admin/apikeys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ci-pipeline",
    "role": "developer",
    "expires_in": 2592000
  }'
```

The `expires_in` field is the key lifetime in seconds (2592000 = 30 days). The response includes the raw key:

```json
{
  "id": 1,
  "key": "sr_live_a1b2c3d4e5f6...",
  "key_prefix": "a1b2c3d4",
  "name": "ci-pipeline",
  "role": "developer",
  "user_id": 1,
  "username": "admin",
  "enabled": true,
  "created_at": "2026-02-16T10:00:00Z",
  "expires_at": "2026-03-18T10:00:00Z"
}
```

Save the `key` value immediately. It cannot be retrieved later.

## LDAP / Active Directory

LDAP authentication integrates with existing directory services. Users authenticate with their directory credentials, and LDAP group memberships are mapped to registry roles.

### Configuration

```yaml
security:
  auth:
    enabled: true
    methods:
      - basic
    ldap:
      enabled: true
      url: "ldap://ldap.example.com:389"
      bind_dn: "cn=service-account,ou=Services,dc=example,dc=com"
      bind_password: "${LDAP_BIND_PASSWORD}"
      base_dn: "dc=example,dc=com"
      user_search_base: "ou=Users,dc=example,dc=com"
      user_search_filter: "(sAMAccountName=%s)"
      username_attribute: "sAMAccountName"
      email_attribute: "mail"
      group_attribute: "memberOf"
      role_mapping:
        "CN=SchemaAdmins,OU=Groups,DC=example,DC=com": "admin"
        "CN=Developers,OU=Groups,DC=example,DC=com": "developer"
        "SchemaAdmins": "admin"
        "Developers": "developer"
      default_role: "readonly"
      start_tls: true
      ca_cert_file: "/etc/ssl/certs/ldap-ca.pem"
      connection_timeout: 10
      request_timeout: 30
```

### How It Works

1. The registry connects to the LDAP server using the service account (`bind_dn`).
2. It searches for the user with the configured `user_search_filter`, substituting `%s` with the provided username.
3. If the user is found, the registry re-binds with the user's own DN and supplied password to verify credentials.
4. On successful authentication, the user's group memberships are extracted from the `group_attribute`.
5. Groups are matched against `role_mapping` to determine the registry role. Both full DNs and Common Names (CN) are supported for matching, with case-insensitive comparison on CNs.

### LDAPS and StartTLS

For encrypted connections:

- **LDAPS** -- use `ldaps://` in the URL (e.g., `ldaps://ldap.example.com:636`)
- **StartTLS** -- set `start_tls: true` with a plain `ldap://` URL

Both methods support custom CA certificates via `ca_cert_file`. For development and testing environments, `insecure_skip_verify: true` disables certificate validation (not recommended for production).

### Configuration Reference

| Field | Description | Default |
|-------|-------------|---------|
| `url` | LDAP server URL | (required) |
| `bind_dn` | Service account DN for searches | (required) |
| `bind_password` | Service account password | (required) |
| `base_dn` | Base DN for searches | (required) |
| `user_search_base` | DN to start user searches from | Same as `base_dn` |
| `user_search_filter` | LDAP filter for finding users (`%s` = username) | `(sAMAccountName=%s)` |
| `username_attribute` | Attribute containing the username | `sAMAccountName` |
| `email_attribute` | Attribute containing the email | `mail` |
| `group_attribute` | Attribute containing group memberships | `memberOf` |
| `role_mapping` | Map of LDAP group names/DNs to registry roles | `{}` |
| `default_role` | Role assigned when no group mapping matches | `readonly` |
| `start_tls` | Upgrade connection to TLS via StartTLS | `false` |
| `ca_cert_file` | Path to CA certificate for TLS verification | `""` |
| `insecure_skip_verify` | Skip TLS certificate verification | `false` |
| `connection_timeout` | Connection timeout in seconds | `10` |
| `request_timeout` | Search request timeout in seconds | `30` |

## OIDC (OpenID Connect)

OIDC authentication validates Bearer tokens against an OpenID Connect provider. This works with Keycloak, Okta, Auth0, Azure AD, and any standards-compliant OIDC provider.

### Configuration

```yaml
security:
  auth:
    enabled: true
    methods:
      - oidc
    oidc:
      enabled: true
      issuer_url: "https://auth.example.com/realms/myorg"
      client_id: "schema-registry"
      username_claim: "preferred_username"
      roles_claim: "realm_access.roles"
      role_mapping:
        "schema-admin": "admin"
        "schema-developer": "developer"
        "schema-reader": "readonly"
      default_role: "readonly"
      required_audience: "schema-registry"
      allowed_algorithms:
        - RS256
        - ES256
```

### Usage

Obtain a token from your OIDC provider and pass it as a Bearer token:

```bash
TOKEN=$(curl -s -X POST "https://auth.example.com/realms/myorg/protocol/openid-connect/token" \
  -d "grant_type=client_credentials" \
  -d "client_id=schema-registry" \
  -d "client_secret=your-client-secret" | jq -r '.access_token')

curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/subjects
```

### How It Works

1. The registry fetches the OIDC discovery document from `issuer_url` at startup.
2. For each request with a Bearer token, the token is verified against the provider's public keys.
3. The `client_id` is checked against the token's audience.
4. If `required_audience` is set, the `aud` claim must contain that value.
5. The username is extracted from the configured `username_claim` (with fallback to `sub`).
6. Roles are extracted from `roles_claim`, which supports dot notation for nested claims (e.g., `realm_access.roles` for Keycloak).
7. Extracted roles are mapped to registry roles via `role_mapping`.

### Configuration Reference

| Field | Description | Default |
|-------|-------------|---------|
| `issuer_url` | OIDC provider URL (must serve `.well-known/openid-configuration`) | (required) |
| `client_id` | Client ID for token audience validation | (required) |
| `username_claim` | Token claim containing the username | `sub` |
| `roles_claim` | Token claim containing roles (supports dot notation) | `""` |
| `role_mapping` | Map of OIDC roles to registry roles | `{}` |
| `default_role` | Role assigned when no role mapping matches | `readonly` |
| `required_audience` | Required value in the `aud` claim | `""` |
| `allowed_algorithms` | Restrict accepted signing algorithms | `[]` (all supported) |
| `skip_issuer_check` | Skip issuer validation (testing only) | `false` |
| `skip_expiry_check` | Skip token expiry validation (testing only) | `false` |

## JWT (JSON Web Token)

JWT authentication validates tokens using a static public key file or a JWKS (JSON Web Key Set) URL. Use this when you have a custom token issuer that does not implement full OIDC discovery.

### Configuration with Static Key

```yaml
security:
  auth:
    enabled: true
    methods:
      - jwt
    jwt:
      public_key_file: "/etc/schema-registry/jwt-public.pem"
      algorithm: "RS256"
      issuer: "https://auth.example.com"
      audience: "schema-registry"
      claims_mapping:
        role: "custom_role_claim"
        roles: "custom_roles_claim"
```

### Configuration with JWKS URL

```yaml
security:
  auth:
    enabled: true
    methods:
      - jwt
    jwt:
      jwks_url: "https://auth.example.com/.well-known/jwks.json"
      algorithm: "RS256"
      issuer: "https://auth.example.com"
      audience: "schema-registry"
```

### Usage

```bash
curl -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  http://localhost:8081/subjects
```

### How It Works

1. The token is parsed and the signing method is validated against the configured `algorithm`.
2. If a `public_key_file` is configured, it is used for signature verification. Supported formats: RSA (PEM), ECDSA (PEM), or HMAC (raw bytes).
3. If a `jwks_url` is configured, the key is looked up by the token's `kid` header. The JWKS key set is cached for 5 minutes and refreshed automatically.
4. Standard claims are validated: `iss` (issuer), `aud` (audience), `exp` (expiration).
5. The username is extracted from `sub`, `preferred_username`, or `email` (in that order).
6. The role is extracted from the `role` or `roles` claim, with support for custom claim names via `claims_mapping`.

### Configuration Reference

| Field | Description | Default |
|-------|-------------|---------|
| `public_key_file` | Path to PEM-encoded public key file | `""` |
| `jwks_url` | URL to JWKS endpoint | `""` |
| `algorithm` | Expected signing algorithm (RS256, RS384, RS512, ES256, ES384, ES512, HS256, HS384, HS512) | `""` |
| `issuer` | Expected token issuer (`iss` claim) | `""` |
| `audience` | Expected token audience (`aud` claim) | `""` |
| `claims_mapping` | Map of standard claim names to custom claim names | `{}` |

## mTLS (Mutual TLS)

Mutual TLS authentication uses client certificates to identify users. The Common Name (CN) from the client certificate's subject is used as the username.

### Configuration

First, enable TLS on the server:

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/schema-registry/server.crt"
    key_file: "/etc/schema-registry/server.key"
    ca_file: "/etc/schema-registry/ca.crt"
    client_auth: "verify"
    min_version: "TLS1.2"
```

Then enable mTLS as an authentication method:

```yaml
security:
  auth:
    enabled: true
    methods:
      - mtls
    rbac:
      default_role: developer
```

### Client Auth Modes

| Mode | Behavior |
|------|----------|
| `none` | No client certificate requested |
| `request` | Client certificate requested but not required |
| `require` | Client certificate required but not verified against CA |
| `verify` | Client certificate required and verified against the CA in `ca_file` |

For mTLS authentication, use `verify` to ensure clients present valid certificates signed by your CA.

### Usage

```bash
curl --cert client.crt --key client.key --cacert ca.crt \
  https://localhost:8081/subjects
```

The authenticated username is the CN from the client certificate. Users authenticated via mTLS are assigned the `default_role` from the RBAC configuration.

## Roles and Permissions

The registry uses a fixed set of roles with predefined permissions. Roles cannot be customized, but the `super_admins` list in the RBAC configuration grants unrestricted access to specific usernames regardless of their assigned role.

| Role | Schema Read | Schema Write | Schema Delete | Config Read | Config Write | Mode Read | Mode Write | User Mgmt |
|------|:-----------:|:------------:|:-------------:|:-----------:|:------------:|:---------:|:----------:|:---------:|
| `super_admin` | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| `admin` | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Read only |
| `developer` | Yes | Yes | No | Yes | No | Yes | No | No |
| `readonly` | Yes | No | No | Yes | No | Yes | No | No |

### RBAC Configuration

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

Users listed in `super_admins` have all permissions regardless of their assigned role. The `default_role` is applied when an authentication method does not provide a role (e.g., mTLS, config-based basic auth).

## User Management API

User management requires the `admin:write` permission (`super_admin` role). For complete request and response schemas, see the [API Reference](api-reference.md).

### Create a User

```bash
curl -u admin:password -X POST http://localhost:8081/admin/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "jane",
    "password": "secure-password",
    "email": "jane@example.com",
    "role": "developer",
    "enabled": true
  }'
```

### List Users

```bash
curl -u admin:password http://localhost:8081/admin/users
```

### Get a User

```bash
curl -u admin:password http://localhost:8081/admin/users/1
```

### Update a User

```bash
curl -u admin:password -X PUT http://localhost:8081/admin/users/1 \
  -H "Content-Type: application/json" \
  -d '{
    "role": "admin",
    "enabled": true
  }'
```

### Delete a User

```bash
curl -u admin:password -X DELETE http://localhost:8081/admin/users/1
```

### Change Your Own Password

Any authenticated user can change their own password via the self-service endpoint:

```bash
curl -u jane:old-password -X POST http://localhost:8081/me/password \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "old-password",
    "new_password": "new-secure-password"
  }'
```

## API Key Management API

API key management requires the `admin:write` permission. Keys are created for the currently authenticated user by default. Super admins can create keys for other users by specifying `for_user_id`.

### Create an API Key

```bash
curl -u admin:password -X POST http://localhost:8081/admin/apikeys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-pipeline",
    "role": "developer",
    "expires_in": 7776000
  }'
```

The `expires_in` value is in seconds (7776000 = 90 days).

### List API Keys

```bash
curl -u admin:password http://localhost:8081/admin/apikeys
```

Filter by user:

```bash
curl -u admin:password "http://localhost:8081/admin/apikeys?user_id=1"
```

### Rotate an API Key

Rotation atomically creates a new key with the same settings and revokes the old one:

```bash
curl -u admin:password -X POST http://localhost:8081/admin/apikeys/1/rotate \
  -H "Content-Type: application/json" \
  -d '{
    "expires_in": 7776000
  }'
```

The response includes both the new key (with the raw key value) and the ID of the revoked key.

### Revoke an API Key

Revocation disables a key without deleting it, preserving the audit trail:

```bash
curl -u admin:password -X POST http://localhost:8081/admin/apikeys/1/revoke
```

### Delete an API Key

```bash
curl -u admin:password -X DELETE http://localhost:8081/admin/apikeys/1
```

## Admin CLI

The `schema-registry-admin` tool provides command-line management of users, API keys, and roles. It communicates with the registry over HTTP, so the server must be running (except for the `init` command, which connects directly to the database).

### Authentication

The CLI supports Basic Auth and API key authentication:

```bash
# Basic Auth
schema-registry-admin -u admin -p password user list

# API Key
schema-registry-admin -k sr_live_abc123... user list

# Custom server URL
schema-registry-admin -s https://registry.example.com:8081 -u admin -p password user list
```

### User Commands

```bash
schema-registry-admin user list
schema-registry-admin user get <id>
schema-registry-admin user create --name jane --pass secret --role developer --email jane@example.com
schema-registry-admin user update <id> --role admin
schema-registry-admin user update <id> --disabled
schema-registry-admin user delete <id>
```

### API Key Commands

```bash
schema-registry-admin apikey list
schema-registry-admin apikey list --user-id 1
schema-registry-admin apikey get <id>
schema-registry-admin apikey create --name ci-key --role developer --expires-in 720h
schema-registry-admin apikey create --name ops-key --role admin --expires-in 8760h --for-user-id 2
schema-registry-admin apikey update <id> --role admin
schema-registry-admin apikey revoke <id>
schema-registry-admin apikey rotate <id> --expires-in 720h
schema-registry-admin apikey delete <id>
```

### Role Commands

```bash
schema-registry-admin role list
```

### Output Formats

The CLI supports table (default) and JSON output:

```bash
schema-registry-admin -o json user list
schema-registry-admin -o json apikey get 1
```

### Database Bootstrap

The `init` command creates the initial admin user by connecting directly to the database:

```bash
schema-registry-admin init \
  --storage-type postgresql \
  --pg-host localhost --pg-port 5432 \
  --pg-database schema_registry \
  --pg-user postgres --pg-password dbpass \
  --admin-username admin \
  --admin-password 'secure-password'
```

Supported storage types: `postgresql`, `mysql`, `cassandra`, `memory`.

## Combining Authentication Methods

Multiple authentication methods can be enabled simultaneously. The registry tries each method in the order listed in the `methods` array and accepts the first successful authentication.

A common production configuration combines API keys for programmatic access with Basic Auth for interactive use and LDAP for enterprise directory integration:

```yaml
security:
  auth:
    enabled: true
    methods:
      - api_key
      - basic
    basic:
      realm: "Schema Registry"
    api_key:
      header: X-API-Key
      query_param: api_key
      key_prefix: "sr_"
      cache_refresh_seconds: 60
    ldap:
      enabled: true
      url: "ldaps://ldap.example.com:636"
      bind_dn: "cn=svc-schema-registry,ou=Services,dc=example,dc=com"
      bind_password: "${LDAP_BIND_PASSWORD}"
      base_dn: "dc=example,dc=com"
      user_search_filter: "(sAMAccountName=%s)"
      role_mapping:
        "SchemaAdmins": "admin"
        "Developers": "developer"
      default_role: "readonly"
    rbac:
      enabled: true
      default_role: readonly
      super_admins:
        - admin
```

With this configuration:
- Kafka producers and consumers use API keys via the Confluent-compatible `-u "API_KEY:x"` format
- DevOps teams authenticate using their LDAP credentials with `curl -u username:password`
- CI/CD pipelines use API keys via the `X-API-Key` header
- The bootstrap admin user retains full access via `super_admins`

## Related Documentation

- [Configuration](configuration.md) -- full configuration reference
- [Security](security.md) -- TLS, rate limiting, and audit logging
- [API Reference](api-reference.md) -- complete API endpoint documentation
